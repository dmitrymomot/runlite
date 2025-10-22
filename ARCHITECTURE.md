# runlite Architecture

**Version:** 0.1.0 (MVP)
**Last Updated:** 2025-10-22

## Overview

runlite is a lightweight, self-hosted PaaS inspired by Dokku. It enables git push deployments with zero-downtime, managed by systemd and routed through an external Caddy server.

**Core Philosophy:** Keep it simple. Start as a tool for personal use, then grow for others.

**CLI Design:** Stateless - commands execute and exit. No background processes. Systemd manages all app lifecycles.

## Current Implementation

### What Exists Today

```
cmd/
├── cli/                    # Stateless CLI application
│   ├── main.go
│   └── commands/
│       ├── app/            # app:create, app:init, app:list, app:delete
│       ├── domain/         # domain:add, domain:remove, domain:list
│       ├── env/            # env:set, env:get, env:list, env:import, env:export
│       └── deploy/         # deploy:start, deploy:rollback, deploy:restart (stubs)
└── domain-registry/        # Standalone HTTP service (port 3000)
    └── main.go

internal/
├── app/                    # Application metadata lifecycle
│   ├── manager.go          # Manager pattern for deployment state
│   ├── metadata.go         # Deployment struct, state management
│   └── manager_test.go     # 90.8% coverage
├── caddy/                  # Caddy Admin API client
│   └── client.go           # Add/update/delete routes, zero-downtime updates
├── config/                 # Path utilities and validation
│   └── paths.go            # GetAppDir, ValidateAppName (security)
├── env/                    # Environment variable management
│   ├── storage.go          # Atomic file I/O
│   ├── parser.go           # KEY=value format parsing
│   ├── diff.go             # Compare changes
│   └── *_test.go           # Comprehensive tests
├── registry/               # Domain registry (SQLite)
│   ├── storage.go          # Domain → app mappings
│   └── api.go              # HTTP API handlers
└── spec/                   # runlite.yml parser and validator ✨ NEW
    ├── types.go            # AppSpec, BuildConfig, RunConfig, HealthConfig, etc.
    ├── parser.go           # ParseFile with YAML unmarshaling + defaults
    ├── validator.go        # Validation rules (app name, paths, durations)
    └── *_test.go           # 93.9% coverage (70+ test cases)

schema/                     # JSON Schema for runlite.yml validation
scripts/                    # Installation, config templates, systemd
```

### What's Next to Implement

- **Deployment orchestrator** (`internal/deploy/`) - Port allocation, systemd unit generation, build execution, health checks
- **Git hooks** (post-receive) - Trigger deployments on push

### Future Work

- Litestream backups
- GitHub webhooks (optional)
- Web UI

## Architecture Decisions

### 1. Stateless CLI + Systemd Process Management

**Decision:** CLI is stateless - run → execute → exit. Systemd manages all app processes.

**Why:**

- ✅ No background processes in runlite
- ✅ Systemd handles process crashes, restarts, logging
- ✅ Standard Linux tooling (`systemctl status`, `journalctl`)
- ✅ CLI can be restarted/updated without affecting running apps
- ✅ Native systemd sandboxing and resource limits

**Implementation:**

- Each release → separate systemd unit file
- Unit naming: `runlite-{appName}@{releaseID}.service`
- ReleaseID format: `{timestamp}-{commit-hash-7}` (e.g., `20251022143055-a3f5c2b`)
- CLI generates unit files, uses `systemctl` commands to manage lifecycle

**Example:**
```bash
# CLI generates unit file and exits
runlite deploy hook my-app abc123d

# Systemd manages the process
systemctl status runlite-my-app@20251022143055-a3f5c2b
journalctl -u runlite-my-app@20251022143055-a3f5c2b -f
```

### 2. Configuration Storage

**Decision:** File-based, not database-centric

**Implementation:**

- App metadata → JSON files at `/var/lib/runlite/apps/{appName}/app.json`
- Environment vars → Plain text at `/var/lib/runlite/apps/{appName}/env`
- Domain registry → SQLite (only for domains, no migrations)

**Why:**

- ✅ Simple to debug (`cat app.json`)
- ✅ Easy to backup (`cp -r`)
- ✅ No schema migrations for app data
- ✅ Transparent state

**Trade-off:**

- ❌ No SQL queries across apps (acceptable for MVP)

### 3. External Caddy

**Decision:** Use external Caddy, not embedded

**Implementation:**

- Caddy runs as separate systemd service
- runlite communicates via Admin API (localhost:2019)
- Routes identified by: `runlite-{appName}`

**Why:**

- ✅ Caddy manages its own HTTPS certificates
- ✅ Can restart runlite without affecting traffic
- ✅ Simpler than embedding
- ✅ Caddy handles all TLS complexity

### 4. Git Push Deployment

**Decision:** Dokku-style git push, no GitHub webhooks (MVP)

**Flow:**

```
Developer → git push runlite main
         ↓
Git bare repo (/var/lib/runlite/apps/{app}/git/)
         ↓
post-receive hook
         ↓
runlite deploy hook {appName} {commit}
         ↓
CLI parses, builds, generates systemd unit, exits
         ↓
systemd starts app process
```

**Why:**

- ✅ Instant feedback over SSH
- ✅ No public endpoint needed
- ✅ Works without internet
- ✅ Can deploy uncommitted code (testing)

**Future:** Can add GitHub webhooks as optional feature

### 5. Blue/Green Deployment Pattern

**Decision:** Manager pattern tracks active/standby deployments

**Implementation:**

- `app.Manager` enforces single active + single standby per app
- Deployment states: `active`, `standby`, `failed`, `stopped`
- State transitions automatic when adding new deployment
- Rollback = Caddy route update + state swap (instant)

**Why:**

- ✅ Zero-downtime deployments
- ✅ Instant rollback (standby still running)
- ✅ Failed deployments never go active
- ✅ State machine prevents invalid transitions

## System Architecture

### File Structure

```
/var/lib/runlite/
└── apps/
    └── {appName}/
        ├── app.json              # Metadata with deployment history
        ├── env                   # Environment variables (KEY=value)
        ├── git/                  # Bare repository
        │   └── hooks/
        │       └── post-receive
        └── releases/
            ├── 20251022143055-a3f5c2b/   # ReleaseID = {timestamp}-{commit-hash}
            │   ├── server                # Built artifacts
            │   └── migrations/
            └── 20251022150320-d8f9e1a/
                └── ...

/etc/systemd/system/
├── caddy.service                                     # External Caddy
├── runlite-domain-registry.service                   # Domain registry API
└── runlite-{appName}@{releaseID}.service             # Per-release app units
```

### App Metadata Format

JSON file at `/var/lib/runlite/apps/{appName}/app.json`:

```json
{
  "name": "my-app",
  "created_at": "2025-10-22T10:00:00Z",
  "updated_at": "2025-10-22T14:30:00Z",
  "deployments": [
    {
      "release_id": "20251022143055-a3f5c2b",
      "commit": "a3f5c2b8e9d1f2a3b4c5d6e7f8a9b0c1d2e3f4a5",
      "port": 8080,
      "status": "active",
      "binary_path": "/var/lib/runlite/apps/my-app/releases/20251022143055-a3f5c2b/server",
      "deployed_at": "2025-10-22T14:30:00Z",
      "health_check_passed": true
    },
    {
      "release_id": "20251022120015-xyz789a",
      "commit": "xyz789abc012345",
      "port": 8081,
      "status": "standby",
      "binary_path": "/var/lib/runlite/apps/my-app/releases/20251022120015-xyz789a/server",
      "deployed_at": "2025-10-22T12:00:15Z",
      "health_check_passed": true
    },
    {
      "release_id": "20251021160045-def456b",
      "commit": "def456bcd789012",
      "port": 8082,
      "status": "failed",
      "binary_path": "/var/lib/runlite/apps/my-app/releases/20251021160045-def456b/server",
      "deployed_at": "2025-10-21T16:00:45Z",
      "health_check_passed": false,
      "failure_reason": "health check timeout after 30s"
    }
  ]
}
```

**Deployment Status Values:**

- `active` - Currently serving traffic (only one per app)
- `standby` - Last successful deployment, ready for instant rollback (only one per app)
- `failed` - Health check failed, never went active
- `stopped` - Old release, systemd unit stopped and disabled

**Manager Enforced Rules:**

- Only one `active` deployment at a time
- Only one `standby` deployment at a time
- Failed deployments cannot become active
- Maximum 10 deployments kept in history (auto-cleanup)

### Environment Variables Format

Plain text file at `/var/lib/runlite/apps/{appName}/env`:

```bash
DATABASE_URL=./databases/main.db
SECRET_KEY=xxxxx
API_TOKEN=yyyyy
```

**Operations:**

- `env/storage.go` - Read/write file atomically
- `env/parser.go` - Parse KEY=value format (supports multi-line values)
- `env/diff.go` - Compare changes between versions

## Deployment Flow

### Zero-Downtime Deployment (7 Phases)

#### Phase 1: Preparation
1. Git hook receives push, calls `runlite deploy hook {appName} {commitHash}`
2. Parse `runlite.yml` from git commit
3. Load environment variables from `/apps/{name}/env`
4. Generate ReleaseID: `{timestamp}-{commitHash[:7]}`
5. Create release directory: `/apps/{name}/releases/{releaseID}/`
6. Acquire deployment lock (prevent concurrent deployments)

#### Phase 2: Build
7. Checkout git commit to temp directory
8. Execute `spec.Build.Script` (user-defined build command)
9. Copy `spec.Build.Artifacts` to release directory
10. Validate artifacts exist

#### Phase 3: Systemd Unit Generation
11. Find FREE port in range 8000-9000 (bind test, not just random)
12. Generate systemd unit file:
    ```ini
    [Unit]
    Description=runlite - {appName} ({releaseID})
    After=network.target

    [Service]
    Type=simple
    User=runlite
    WorkingDirectory=/var/lib/runlite/apps/{appName}/releases/{releaseID}
    ExecStart={spec.Run.Command} {args with {{.Port}} replaced}
    EnvironmentFile=/var/lib/runlite/apps/{appName}/env
    Environment="PORT={allocatedPort}"
    Restart=on-failure
    RestartSec=5s

    [Install]
    WantedBy=multi-user.target
    ```
13. `systemctl daemon-reload`

#### Phase 4: Process Start
14. `systemctl start runlite-{appName}@{releaseID}`
15. Save deployment record as "standby" (`Manager.AddDeployment()`)

#### Phase 5: Health Check
16. Poll `http://localhost:{port}{spec.Health.Path}`
17. Timeout after `spec.Health.Timeout`, poll interval `spec.Health.Interval`
18. **On failure**: `systemctl stop`, mark "failed", exit (old deployment stays active)

#### Phase 6: Traffic Switch
19. Update Caddy route to new port (or add route if first deployment)
20. Manager automatically transitions: old active → standby, new standby → active

#### Phase 7: Drain & Cleanup
21. Wait `spec.Deploy.DrainPeriod` (default 30s - let old app finish requests)
22. Get old standby deployment from Manager
23. `systemctl stop runlite-{appName}@{oldReleaseID}`
24. `systemctl disable runlite-{appName}@{oldReleaseID}`
25. Mark old deployment as "stopped"
26. Release deployment lock

**Result:** Zero-downtime deployment. Old standby kept for instant rollback until next deployment.

### Rollback Flow

```
1. User runs: runlite deploy rollback {appName}
2. Load app.json metadata
3. Find standby deployment (last successful release)
4. Update Caddy route → switch to standby port
5. Manager swaps: active ↔ standby
6. Done (instant - standby process still running)
```

**Fast:** Standby deployment already running, just route switch + metadata update.
**Safe:** Only successful deployments (health_check_passed=true) can be standby.

## Module Architecture

### Layered Design

```
┌─────────────────────────────────────────┐
│      CLI Layer (Thin, Stateless)        │
│  cmd/cli/commands/{category}/{action}   │
├─────────────────────────────────────────┤
│   Orchestration Layer (To Implement)    │
│  internal/deploy/ - Deployment flow     │
├─────────────────────────────────────────┤
│      Module Layer (Implemented)         │
│  ├── app/        Metadata + Manager     │
│  ├── spec/       YAML parser            │
│  ├── caddy/      API client             │
│  ├── config/     Path validation        │
│  ├── env/        Environment vars       │
│  └── registry/   Domain registry        │
└─────────────────────────────────────────┘
```

### Core Modules

#### app/ - Application Metadata Lifecycle

**Purpose:** Manage app.json with deployment history and state transitions

**Pattern:** Manager-based access with enforced business rules

**Key Types:**

```go
type Metadata struct {
    Name        string       `json:"name"`
    CreatedAt   time.Time    `json:"created_at"`
    UpdatedAt   time.Time    `json:"updated_at"`
    Deployments []Deployment `json:"deployments,omitempty"`
}

type Deployment struct {
    ReleaseID         string     `json:"release_id"`      // {timestamp}-{commit-hash-7}
    Commit            string     `json:"commit"`          // Full commit hash
    Port              int        `json:"port"`
    Status            string     `json:"status"`          // active, standby, failed, stopped
    BinaryPath        string     `json:"binary_path"`
    DeployedAt        time.Time  `json:"deployed_at"`
    StoppedAt         *time.Time `json:"stopped_at,omitempty"`
    HealthCheckPassed bool       `json:"health_check_passed"`
    FailureReason     string     `json:"failure_reason,omitempty"`
}
```

**Manager API:**

```go
// Lifecycle
func NewManager(dataDir string) *Manager
func (m *Manager) Create(appName string) error
func (m *Manager) Load(appName string) (*Metadata, error)
func (m *Manager) Save(meta *Metadata) error
func (m *Manager) Exists(appName string) bool

// Deployment state management
func (m *Manager) AddDeployment(appName string, deployment Deployment) error
func (m *Manager) GetActiveDeployment(appName string) (*Deployment, error)
func (m *Manager) GetStandbyDeployment(appName string) (*Deployment, error)
func (m *Manager) Rollback(appName string) (*Deployment, error)
func (m *Manager) MarkDeploymentStopped(appName, releaseID string) error
```

**Why Manager Pattern:**

- ✅ Enforces single active + single standby rule
- ✅ Automatic state transitions (old active → standby → stopped)
- ✅ Atomic file operations (prevents corruption)
- ✅ Auto-cleanup (max 10 deployments)
- ✅ Prevents invalid states

#### spec/ - runlite.yml Parser ✨

**Purpose:** Parse and validate user deployment configuration

**API:**

```go
func ParseFile(path string) (*AppSpec, error)
```

**Features:**

- YAML unmarshaling with custom handlers (Artifact string/object formats, duration parsing)
- Default value application (health.path="/health", timeouts, etc.)
- Comprehensive validation (app name regex, path prefixes, duration bounds)
- 93.9% test coverage

**AppSpec Structure:**

```go
type AppSpec struct {
    App      AppConfig       // name (required)
    Build    *BuildConfig    // script, artifacts
    Run      *RunConfig      // command, args ({{.Port}} template support)
    Health   *HealthConfig   // path, timeout, interval
    Database *DatabaseConfig // path or file
    Static   *StaticConfig   // path
    Deploy   *DeployConfig   // drain_period
}
```

#### caddy/ - Caddy Admin API Client

**Purpose:** Communicate with external Caddy server for routing

**API:**

```go
type Client struct {
    baseURL string // http://localhost:2019
}

func (c *Client) AddAppRoute(appName string, domains []string, port int) error
func (c *Client) UpdateAppRoute(appName string, port int) error
func (c *Client) DeleteAppRoute(appName string) error
func (c *Client) AddDomainToApp(appName, domain string) error
func (c *Client) RemoveDomainFromApp(appName, domain string) error
func (c *Client) ConfigureOnDemandTLS() error
```

**Route ID format:** `runlite-{appName}`

**Zero-downtime:** `UpdateAppRoute` changes upstream port atomically

#### env/ - Environment Variables

**Purpose:** Manage app environment variables in plain text files

**API:**

```go
func Load(envPath string) (map[string]string, error)
func Save(envPath string, vars map[string]string) error
func Parse(content string) (map[string]string, error)
func Diff(old, new map[string]string) Changes
```

**Pattern:** Parse → Modify → Atomic save

#### registry/ - Domain Registry

**Purpose:** Track domain → app mappings (SQLite)

**Schema:**

```sql
CREATE TABLE domains (
    domain TEXT PRIMARY KEY,
    app_name TEXT NOT NULL,
    created_at DATETIME NOT NULL
);
```

**Note:** Only SQLite database in runlite (for domains only). No migrations - schema in code.

#### config/ - Path Utilities

**Purpose:** Validate and construct safe file paths

**API:**

```go
func GetDataDir() string
func GetAppDir(appName string) string
func GetAppEnvPath(appName string) string
func EnsureAppDir(appName string) error
func ValidateAppName(name string) error  // Prevents path traversal
```

## Key Patterns

### 1. Atomic File Operations

All file writes follow:

```go
tmpFile := path + ".tmp"
os.WriteFile(tmpFile, data, 0644)
os.Rename(tmpFile, path)  // Atomic
```

**Why:** Prevents corruption on crashes

### 2. Validation at Boundaries

```go
if err := config.ValidateAppName(name); err != nil {
    return err
}
```

**Rules:**
- App names: `^[a-zA-Z0-9_-]+$`
- No path traversal
- No absolute paths in user config

### 3. Error Wrapping

```go
if err := doThing(); err != nil {
    return fmt.Errorf("context: %w", err)
}
```

**Why:** Clear error traces

### 4. Context-First Functions

```go
func Deploy(ctx context.Context, appName string) error
```

**Why:** Cancellation, timeouts, tracing

## Security Considerations

### Process Isolation

- Each app release → separate systemd unit
- Systemd sandboxing directives
- User isolation (apps run as `runlite` user)
- Port range isolation (8000-9000)

### Input Validation

- App names sanitized (prevent path traversal)
- Domain validation (prevent injection)
- Artifact paths must start with `./`
- No arbitrary script execution (validated build scripts)

### File Permissions

- App directories: 0755
- Metadata files: 0644
- Environment files: 0600 (secrets)
- Binaries: 0755

### SSH Security

- Force command in authorized_keys
- No shell access for git user
- Per-app or global SSH keys

## Performance Considerations

### Port Allocation

- Bind test to verify port is free
- Range: 8000-9000 (200 apps max)
- Fast for typical usage

### File Operations

- Atomic renames (no locks needed)
- Sequential consistency guaranteed

### Caddy API

- HTTP calls are synchronous
- Fast (localhost communication)
- Could add retry logic for production

## Future Enhancements

### v0.2
- Litestream integration (automated SQLite backups)
- GitHub webhooks (optional)
- TUI with Bubble Tea

### v0.3
- Web dashboard
- Multi-user support
- Resource limits (CPU/memory via systemd)

### v1.0
- Production-ready
- Full test coverage
- Documentation complete

## References

- [CLAUDE.md](./CLAUDE.md) - Development guidelines and implementation plan
- [README.md](./README.md) - User-facing documentation
- [schema/](./schema/) - runlite.yml JSON Schema

## Questions?

See [CLAUDE.md](./CLAUDE.md) for development context and detailed implementation plan.
