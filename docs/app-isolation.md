# App Isolation & Environment Management

## Overview

Runlite isolates deployed applications using systemd service units without containerization. Each app runs as an independent systemd service with its own environment variables, process space, and resource controls.

## Isolation Mechanisms

### 1. Process Isolation

Each app runs as a separate systemd service:

```
systemd
├── myapp1.service  (PID 1234)
├── myapp2.service  (PID 1235)
└── myapp3.service  (PID 1236)
```

**Guarantees:**
- Separate process namespaces
- Independent lifecycle (start/stop/restart)
- Isolated crash domains (one app crash doesn't affect others)
- Per-service resource limits (CPU, memory)

### 2. Environment Variable Isolation

Environment variables are loaded from per-app files and are **not visible** to other apps.

**Directory structure:**
```
/var/lib/runlite/apps/
├── myapp1/
│   ├── binary          # App executable
│   ├── env             # Environment variables (600 permissions)
│   └── data.db         # Optional app database
├── myapp2/
│   ├── binary
│   ├── env
│   └── data.db
└── myapp3/
    ├── binary
    └── env
```

Each `env` file contains app-specific variables:
```bash
# /var/lib/runlite/apps/myapp1/env
DATABASE_URL=sqlite:///var/lib/runlite/apps/myapp1/data.db
API_KEY=secret_key_for_app1
STRIPE_SECRET=sk_test_...
PORT=8001
```

### 3. Filesystem Isolation

**Working directory per app:**
```ini
[Service]
WorkingDirectory=/var/lib/runlite/apps/myapp1
```

**systemd security directives:**
```ini
[Service]
ProtectSystem=strict          # Read-only /usr, /boot, /efi
ProtectHome=true              # No access to /home
NoNewPrivileges=true          # Can't gain new privileges
PrivateTmp=true               # Private /tmp directory
ReadWritePaths=/var/lib/runlite/apps/myapp1  # Only this dir is writable
```

### 4. Network Isolation (port-based)

Each app listens on a unique localhost port:
- App 1: `localhost:8001`
- App 2: `localhost:8002`
- App 3: `localhost:8003`

Apps cannot directly communicate without going through the Caddy reverse proxy.

---

## Environment Variable Loading

### systemd EnvironmentFile Directive

Runlite uses systemd's `EnvironmentFile` directive to load environment variables from a per-app file.

**systemd unit template:**
```ini
[Unit]
Description={{.Name}} - Runlite App
After=network.target

[Service]
Type=simple
User=runlite
WorkingDirectory=/var/lib/runlite/apps/{{.Name}}
ExecStart=/var/lib/runlite/apps/{{.Name}}/binary
EnvironmentFile=/var/lib/runlite/apps/{{.Name}}/env
Environment="PORT={{.Port}}"
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier={{.Name}}

# Security hardening
ProtectSystem=strict
ProtectHome=true
NoNewPrivileges=true
PrivateTmp=true
ReadWritePaths=/var/lib/runlite/apps/{{.Name}}

[Install]
WantedBy=multi-user.target
```

**How it works:**
1. systemd reads `/var/lib/runlite/apps/myapp1/env` on service start
2. Each `KEY=value` line becomes an environment variable
3. Variables are available to the app process via `os.Getenv()`
4. Variables are **not** visible to other services or processes

### Environment File Format

Standard KEY=value format with support for quotes:

```bash
# Simple values
DATABASE_URL=sqlite:///var/lib/runlite/apps/myapp/data.db
PORT=8001

# Values with spaces (quoted)
APP_NAME="My Cool App"

# Multi-line values (not recommended, use secrets file instead)
PRIVATE_KEY="-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBg...
-----END PRIVATE KEY-----"

# Comments are supported
# DATABASE_URL=postgres://old-url  # old value
```

### File Permissions

Environment files contain secrets and must be protected:

```bash
# Owner: runlite user
# Permissions: 600 (read/write for owner only)
-rw------- 1 runlite runlite 245 Oct 18 10:30 /var/lib/runlite/apps/myapp1/env
```

This prevents:
- Other users from reading secrets
- Other apps from accessing env files
- Accidental exposure via directory listing

---

## Environment Variable Management

### Creating/Updating Variables

When a user updates environment variables through the panel:

```go
// internal/app/service.go
func (s *Service) UpdateEnvVars(appID int, envVars map[string]string) error {
    app, err := s.repo.FindApp(appID)
    if err != nil {
        return err
    }

    // 1. Update database
    if err := s.repo.UpdateEnvVars(appID, envVars); err != nil {
        return err
    }

    // 2. Write env file
    envPath := fmt.Sprintf("/var/lib/runlite/apps/%s/env", app.Name)
    if err := writeEnvFile(envPath, envVars); err != nil {
        return err
    }

    // 3. Restart service to load new env vars
    return s.deploy.RestartService(app.Name)
}

func writeEnvFile(path string, envVars map[string]string) error {
    var lines []string
    for k, v := range envVars {
        // Escape quotes in values
        v = strings.ReplaceAll(v, `"`, `\"`)
        lines = append(lines, fmt.Sprintf(`%s="%s"`, k, v))
    }

    content := strings.Join(lines, "\n")

    // Write with restrictive permissions
    return os.WriteFile(path, []byte(content), 0600)
}
```

### Reading Current Variables

To display current env vars in the panel UI:

```go
// internal/app/repository.go
func (r *Repository) GetEnvVars(appID int) (map[string]string, error) {
    rows, err := r.db.Query(`
        SELECT key, value
        FROM app_env_vars
        WHERE app_id = ?
    `, appID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    envVars := make(map[string]string)
    for rows.Next() {
        var key, value string
        if err := rows.Scan(&key, &value); err != nil {
            return nil, err
        }
        envVars[key] = value
    }

    return envVars, nil
}
```

### Deployment-Time Environment Loading

When deploying a new version of an app:

```go
// internal/deploy/builder.go
func (b *Builder) Build(app *app.App, commitSHA string) error {
    // ... build process ...

    // Ensure env file exists before starting service
    envPath := fmt.Sprintf("/var/lib/runlite/apps/%s/env", app.Name)
    if err := writeEnvFile(envPath, app.EnvVars); err != nil {
        return fmt.Errorf("failed to write env file: %w", err)
    }

    // Generate systemd unit (references env file)
    if err := GenerateSystemdUnit(app); err != nil {
        return err
    }

    // systemd will load env file on service start
    return RestartService(app.Name)
}
```

---

## Runtime Behavior

### Variable Precedence

If the same variable is defined in multiple places:

1. **Inline `Environment=` directive** (highest priority)
2. **`EnvironmentFile=`** (loaded first, can be overridden)

Example:
```ini
[Service]
EnvironmentFile=/var/lib/runlite/apps/myapp/env  # PORT=9999 in file
Environment="PORT=8001"                          # This wins
```

Result: `PORT=8001`

**Runlite usage:**
- User-defined vars → `EnvironmentFile`
- System-assigned values (PORT) → inline `Environment=` directive

### Reloading Variables

Environment variables are loaded **once** when the service starts. To apply changes:

```bash
# systemd does NOT reload env file on daemon-reload
sudo systemctl daemon-reload      # Only reloads unit file structure

# Must restart service to reload env vars
sudo systemctl restart myapp1     # Env file is re-read
```

Runlite handles this automatically when user updates env vars.

### Viewing Variables at Runtime

**From systemd (shows effective environment):**
```bash
sudo systemctl show myapp1 --property=Environment
```

**From Go app code:**
```go
dbURL := os.Getenv("DATABASE_URL")
port := os.Getenv("PORT")
```

---

## Security Considerations

### 1. File Permissions

```bash
# App directory
drwxr-x--- runlite runlite /var/lib/runlite/apps/myapp1

# Env file (most restrictive)
-rw------- runlite runlite /var/lib/runlite/apps/myapp1/env

# Binary
-rwxr-x--- runlite runlite /var/lib/runlite/apps/myapp1/binary
```

### 2. systemd Security Directives

Prevent privilege escalation and limit filesystem access:

```ini
[Service]
# Process restrictions
User=runlite
Group=runlite
NoNewPrivileges=true

# Filesystem restrictions
ProtectSystem=strict              # /usr, /boot read-only
ProtectHome=true                  # No /home access
ReadWritePaths=/var/lib/runlite/apps/myapp1  # Only app dir writable
PrivateTmp=true                   # Private /tmp

# Additional hardening (optional)
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictRealtime=true
```

### 3. Secret Storage Best Practices

For highly sensitive data:

**Option 1: Encrypted secrets (future enhancement)**
- Store encrypted values in database
- Decrypt at runtime when writing env file

**Option 2: External secret manager**
- Apps fetch secrets at startup using `SECRET_URL` env var
- Runlite only stores the secret reference

### 4. Audit Logging

Log environment variable changes:

```go
func (s *Service) UpdateEnvVars(appID int, envVars map[string]string) error {
    // ... update logic ...

    // Log change (without values)
    log.Printf("Updated env vars for app %d: %v", appID, keys(envVars))

    return nil
}
```

---

## Comparison to Container Isolation

| Feature | systemd | Containers |
|---------|---------|------------|
| Process isolation | ✅ | ✅ |
| Environment isolation | ✅ | ✅ |
| Filesystem isolation | ⚠️ Partial | ✅ Full |
| Network isolation | ❌ Port-based only | ✅ Network namespaces |
| Resource limits | ✅ cgroups | ✅ cgroups |
| Overhead | Minimal | 10-50MB per container |
| Complexity | Low | Medium |

**Runlite's choice:** systemd provides sufficient isolation for Go apps on a trusted single-server PaaS while maintaining simplicity.

---

## Examples

### Example 1: App with Database

```bash
# /var/lib/runlite/apps/api/env
DATABASE_URL=sqlite:///var/lib/runlite/apps/api/data.db
JWT_SECRET=random_secret_key_here
PORT=8001
LOG_LEVEL=info
```

### Example 2: App with External Services

```bash
# /var/lib/runlite/apps/worker/env
REDIS_URL=redis://localhost:6379
STRIPE_SECRET_KEY=sk_live_...
SENDGRID_API_KEY=SG.xxx
WEBHOOK_SECRET=whsec_...
```

### Example 3: Updating Variables via Panel

User flow:
1. Navigate to app settings in panel
2. Click "Environment Variables"
3. Add/edit: `NEW_VAR=value`
4. Click "Save"
5. Panel backend:
   - Updates `app_env_vars` table
   - Writes new `/var/lib/runlite/apps/myapp/env` file
   - Runs `systemctl restart myapp`
6. App restarts with new environment

---

**Version:** 1.0
**Last Updated:** 2025-10-18
**Related:** [architecture.md](./architecture.md)
