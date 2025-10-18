# Runlite Architecture

## Overview

Runlite is a lightweight PaaS for deploying Go applications with SQLite databases. It uses a modular architecture with clear separation of concerns.

## Project Structure

```
runlite/
├── cmd/
│   └── runlite/
│       └── main.go                 # Application entry point
│
├── internal/
│   ├── auth/                       # GitHub OAuth & authentication
│   ├── database/                   # SQLite operations & schema
│   ├── app/                        # App domain logic & models
│   ├── deploy/                     # Deployment engine (build & systemd)
│   ├── github/                     # GitHub API integration
│   ├── proxy/                      # Caddy reverse proxy management
│   ├── backups/                    # Litestream backup integration
│   ├── panel/                      # Web UI (control panel)
│   └── config/                     # Configuration management
│
├── scripts/
│   ├── install.sh                  # Installation script
│   └── runlite.service.template    # systemd unit template
│
├── tests/                          # Docker test environment (gitignored)
├── docs/                           # Documentation
├── go.mod
└── go.sum
```

## Package Responsibilities

### `cmd/runlite/main.go`

**Purpose:** Application entry point with minimal logic

**Responsibilities:**

- Parse command-line flags and environment variables
- Load configuration
- Initialize database connection
- Start web server (panel)
- Initialize background services (proxy, backups)
- Handle graceful shutdown on SIGTERM/SIGINT

**Example flow:**

```go
func main() {
    // Load config
    cfg := config.Load()

    // Initialize database
    db := database.Connect(cfg.DatabasePath)

    // Start services
    go proxy.Start(cfg)
    go backups.Start(cfg)

    // Start web panel
    panel.Serve(cfg, db)
}
```

---

### `internal/auth/`

**Purpose:** Authentication and authorization

**Files:**

- `oauth.go` - GitHub OAuth flow implementation
- `middleware.go` - HTTP middleware for protected routes
- `token.go` - OAuth token storage and validation

**Key Functions:**

- `HandleGitHubLogin()` - Initiate OAuth flow
- `HandleGitHubCallback()` - Process OAuth callback
- `ValidateUser(username)` - Check if username is in allowed usernames list
- `RequireAuth()` - Middleware to protect routes
- `StoreToken()` - Securely store OAuth token
- `GetStoredToken()` - Retrieve token for GitHub API calls

**Data Flow:**

1. User visits `/login`
2. Redirect to GitHub OAuth
3. GitHub redirects to `/auth/callback`
4. Validate username is in `AllowedUsernames` config (all allowed users have full access)
5. Store OAuth token in database
6. Set session cookie
7. Redirect to dashboard

---

### `internal/database/`

**Purpose:** Database operations and schema management

**Files:**

- `schema.go` - Table definitions and schema version
- `migrations.go` - Database migrations
- `queries.go` - Common query helpers

**Schema:**

```sql
-- Runlite configuration
CREATE TABLE config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Deployed applications
CREATE TABLE apps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    github_repo TEXT NOT NULL,           -- owner/repo
    github_branch TEXT DEFAULT 'main',
    domain TEXT UNIQUE NOT NULL,
    port INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Environment variables per app
CREATE TABLE app_env_vars (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    app_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE,
    UNIQUE(app_id, key)
);

-- Deployment history
CREATE TABLE deployments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    app_id INTEGER NOT NULL,
    commit_sha TEXT NOT NULL,
    commit_message TEXT,
    status TEXT NOT NULL,                -- pending, building, success, failed
    build_log TEXT,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

-- GitHub webhook secrets
CREATE TABLE webhooks (
    app_id INTEGER PRIMARY KEY,
    webhook_id INTEGER NOT NULL,          -- GitHub webhook ID
    secret TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);
```

**Key Functions:**

- `Connect(path)` - Open database connection
- `Migrate()` - Run pending migrations
- `Transaction(fn)` - Execute function in transaction

---

### `internal/app/`

**Purpose:** Core business logic for applications

**Files:**

- `models.go` - Domain models (App, Deployment, etc.)
- `service.go` - Business logic and orchestration
- `repository.go` - Database operations (CRUD)

**Models:**

```go
type App struct {
    ID           int
    Name         string
    GitHubRepo   string
    GitHubBranch string
    Domain       string
    Port         int
    EnvVars      map[string]string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type Deployment struct {
    ID            int
    AppID         int
    CommitSHA     string
    CommitMessage string
    Status        string
    BuildLog      string
    StartedAt     time.Time
    CompletedAt   *time.Time
}
```

**Service Functions:**

- `CreateApp(app)` - Create new app (validates, creates webhook, deploys)
- `DeleteApp(id)` - Delete app (stops service, removes systemd unit, deletes webhook)
- `RestartApp(id)` - Restart app service
- `GetAppLogs(id)` - Fetch logs from journalctl
- `UpdateEnvVars(id, vars)` - Update environment variables

**Repository Functions:**

- `SaveApp(app)` - Insert/update app in database
- `FindApp(id)` - Get app by ID
- `ListApps()` - Get all apps
- `DeleteApp(id)` - Remove app from database

---

### `internal/deploy/`

**Purpose:** Build and deployment engine

**Files:**

- `builder.go` - Git clone and Go build
- `systemd.go` - systemd unit generation and management
- `logs.go` - Build and runtime log capture

**Build Process:**

```go
func Build(app *app.App, commitSHA string) (*Deployment, error) {
    // 1. Create build directory
    buildDir := fmt.Sprintf("/var/lib/runlite/builds/%s/%s", app.Name, commitSHA)

    // 2. Clone repository
    git.Clone(app.GitHubRepo, app.GitHubBranch, buildDir)

    // 3. Checkout specific commit
    git.Checkout(commitSHA)

    // 4. Run go build
    output, err := exec.Command("go", "build", "-o", "app", ".").CombinedOutput()

    // 5. Move binary to app directory
    mv(buildDir+"/app", fmt.Sprintf("/var/lib/runlite/apps/%s/binary", app.Name))

    // 6. Generate systemd unit
    GenerateSystemdUnit(app)

    // 7. Reload systemd and restart service
    systemctl("daemon-reload")
    systemctl("restart", app.Name)

    return deployment, err
}
```

**systemd Unit Template:**

```ini
[Unit]
Description={{.Name}} - Runlite App
After=network.target

[Service]
Type=simple
User=runlite
WorkingDirectory=/var/lib/runlite/apps/{{.Name}}
ExecStart=/var/lib/runlite/apps/{{.Name}}/binary
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier={{.Name}}

# Environment variables
{{range $key, $value := .EnvVars}}
Environment="{{$key}}={{$value}}"
{{end}}

[Install]
WantedBy=multi-user.target
```

**Key Functions:**

- `Build(app, commitSHA)` - Full build process
- `GenerateSystemdUnit(app)` - Create systemd unit file
- `StartService(appName)` - Start app via systemctl
- `StopService(appName)` - Stop app
- `RestartService(appName)` - Restart app
- `GetBuildLogs(deployment)` - Retrieve build output
- `GetRuntimeLogs(appName)` - Get journalctl logs

---

### `internal/github/`

**Purpose:** GitHub API integration

**Files:**

- `client.go` - GitHub API client wrapper
- `repos.go` - Repository operations
- `webhooks.go` - Webhook management

**Key Functions:**

- `NewClient(token)` - Create authenticated GitHub client
- `ListUserRepos()` - Get user's repositories
- `CreateWebhook(repo, url, secret)` - Create webhook for repo
- `DeleteWebhook(repo, webhookID)` - Remove webhook
- `ValidateWebhookSignature(payload, signature, secret)` - Verify webhook authenticity
- `ParseWebhookPayload(body)` - Extract commit info from webhook

**Webhook Payload Handling:**

```go
type WebhookPayload struct {
    Ref        string // refs/heads/main
    After      string // commit SHA
    Repository struct {
        FullName string // owner/repo
    }
    HeadCommit struct {
        Message string
        ID      string
    }
}
```

---

### `internal/proxy/`

**Purpose:** Caddy reverse proxy management

**Files:**

- `caddy.go` - Caddy process lifecycle
- `config.go` - Dynamic route configuration

**Caddy Integration Options:**

**Option 1: Embedded Caddy (Recommended)**

```go
import "github.com/caddyserver/caddy/v2"

func Start(apps []*app.App) {
    cfg := generateCaddyConfig(apps)
    caddy.Load(cfg, true)
}

func AddRoute(app *app.App) {
    // Dynamically add route via Caddy Admin API
}
```

**Option 2: Managed Process**

```go
func Start() {
    // Write Caddyfile
    writeCaddyfile(apps)

    // Start Caddy process
    cmd := exec.Command("caddy", "run", "--config", "/etc/runlite/Caddyfile")
    cmd.Start()
}
```

**Caddy Configuration:**

```json
{
    "apps": {
        "http": {
            "servers": {
                "srv0": {
                    "listen": [":80", ":443"],
                    "routes": [
                        {
                            "match": [{ "host": ["app.example.com"] }],
                            "handle": [
                                {
                                    "handler": "reverse_proxy",
                                    "upstreams": [{ "dial": "localhost:8001" }]
                                }
                            ]
                        }
                    ]
                }
            }
        }
    }
}
```

**Key Functions:**

- `Start()` - Initialize Caddy
- `AddApp(app)` - Add reverse proxy route
- `RemoveApp(app)` - Remove route
- `Reload()` - Reload configuration

---

### `internal/backups/`

**Purpose:** Litestream backup integration

**Files:**

- `litestream.go` - Litestream process management
- `config.go` - Backup configuration

**Litestream Config Generation:**

```yaml
dbs:
    - path: /var/lib/runlite/apps/myapp/data.db
      replicas:
          - type: s3
            bucket: my-backups
            path: myapp
            region: us-east-1
```

**Key Functions:**

- `Start()` - Start Litestream process
- `AddDatabase(appName, dbPath, destination)` - Configure backup for app database
- `RemoveDatabase(appName)` - Stop backing up database
- `Restore(appName, timestamp)` - Restore from backup

---

### `internal/panel/`

**Purpose:** Web UI and HTTP handlers

**Structure:**

```
panel/
├── handlers/
│   ├── dashboard.go     # Dashboard page
│   ├── apps.go          # App management
│   ├── deployments.go   # Deployment history
│   └── webhooks.go      # Webhook receiver
├── templates/
│   ├── layout.templ     # Base layout
│   ├── dashboard.templ  # Dashboard view
│   ├── apps.templ       # Apps list
│   └── app_form.templ   # Create/edit app
└── middleware/
    └── auth.go          # Authentication middleware
```

**Routes:**

```
GET  /                          → Dashboard (requires auth)
GET  /login                     → GitHub OAuth login
GET  /auth/callback             → OAuth callback handler
POST /logout                    → Logout

GET  /apps                      → List apps
GET  /apps/new                  → New app form
POST /apps                      → Create app
GET  /apps/:id                  → App details
PUT  /apps/:id                  → Update app
DELETE /apps/:id                → Delete app
POST /apps/:id/restart          → Restart app
GET  /apps/:id/logs             → View logs

GET  /deployments/:id           → Deployment details
GET  /deployments/:id/logs      → Build logs

POST /webhooks/:app_id          → GitHub webhook receiver
```

**templ Templates:**

```templ
// layout.templ
templ Layout(title string) {
    <!DOCTYPE html>
    <html>
    <head>
        <title>{title} - Runlite</title>
        <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    </head>
    <body>
        <nav><!-- Navigation --></nav>
        { children... }
    </body>
    </html>
}

// dashboard.templ
templ Dashboard(apps []App) {
    @Layout("Dashboard") {
        <h1>Your Apps</h1>
        for _, app := range apps {
            <div class="app-card">
                <h2>{app.Name}</h2>
                <p>{app.Domain}</p>
                <button hx-post={"/apps/" + app.ID + "/restart"}>Restart</button>
            </div>
        }
    }
}
```

---

### `internal/config/`

**Purpose:** Configuration management

**Files:**

- `config.go` - Configuration struct and loading
- `defaults.go` - Default values

**Configuration:**

```go
type Config struct {
    // Server
    Port int
    Host string

    // Database
    DatabasePath string

    // GitHub OAuth
    GitHubClientID     string
    GitHubClientSecret string
    AllowedUsernames   []string  // Multiple users with full access (no RBAC)

    // Paths
    AppsDir       string  // /var/lib/runlite/apps
    BuildsDir     string  // /var/lib/runlite/builds

    // Features
    BackupsEnabled bool
    BackupDestination string
}

func Load() *Config {
    return &Config{
        Port:             getEnvInt("PORT", 8080),
        DatabasePath:     getEnv("DATABASE_PATH", "/var/lib/runlite/runlite.db"),
        GitHubClientID:   getEnv("GITHUB_CLIENT_ID", ""),
        AllowedUsernames: parseUsernames(getEnv("GITHUB_USERNAMES", "")),
        AppsDir:          "/var/lib/runlite/apps",
        BuildsDir:        "/var/lib/runlite/builds",
    }
}

// parseUsernames splits comma-separated usernames
func parseUsernames(s string) []string {
    if s == "" {
        return []string{}
    }
    parts := strings.Split(s, ",")
    result := make([]string, 0, len(parts))
    for _, p := range parts {
        if username := strings.TrimSpace(p); username != "" {
            result = append(result, username)
        }
    }
    return result
}
```

**Environment Variables:**

```bash
PORT=8080
DATABASE_PATH=/var/lib/runlite/runlite.db
GITHUB_CLIENT_ID=xxx
GITHUB_CLIENT_SECRET=xxx
GITHUB_USERNAMES=alice,bob,charlie  # Comma-separated list
BACKUP_DESTINATION=s3://my-bucket/backups
```

---

## Data Flow

### App Creation Flow

```
User → Panel UI → Handler
                    ↓
            [Validate Input]
                    ↓
            app.Service.CreateApp()
                    ↓
        ┌───────────┴───────────┐
        ↓                       ↓
    Save to DB          GitHub API: Create Webhook
        ↓                       ↓
    Deploy Initial Build ← Trigger First Deployment
        ↓
    [Build Process]
        ↓
    Generate systemd Unit
        ↓
    Start Service
        ↓
    Update Caddy Config
        ↓
    Return Success
```

### Webhook Deployment Flow

```
GitHub Push → Webhook Endpoint
                    ↓
            [Validate Signature]
                    ↓
            [Parse Payload]
                    ↓
        Find App by Repo
                    ↓
        Create Deployment Record
                    ↓
        deploy.Build(app, commitSHA)
            ↓
        Clone Repo
            ↓
        go build
            ↓
        Update Binary
            ↓
        Restart systemd Service
            ↓
        Update Deployment Status
            ↓
        Return 200 OK to GitHub
```

---

## Component Interactions

```
┌─────────────┐
│   Browser   │
└──────┬──────┘
       │ HTTP
       ↓
┌─────────────────────────────────────┐
│         panel (Web UI)              │
│  ┌──────────┐     ┌──────────┐     │
│  │ Handlers │ ←→  │Templates │     │
│  └────┬─────┘     └──────────┘     │
│       │                             │
└───────┼─────────────────────────────┘
        │
        ↓
┌────────────────────────────────────────┐
│         app (Business Logic)            │
│  ┌─────────┐         ┌──────────┐     │
│  │ Service │   ←→    │Repository│     │
│  └────┬────┘         └─────┬────┘     │
└───────┼───────────────────┼────────────┘
        │                   │
   ┌────┼────┬──────┬───────┼──────┐
   ↓    ↓    ↓      ↓       ↓      ↓
┌──────┐ ┌──────┐ ┌─────┐ ┌────┐ ┌──────┐
│deploy│ │github│ │proxy│ │auth│ │database│
└──────┘ └──────┘ └─────┘ └────┘ └──────┘
   │
   ↓
┌──────────┐
│ systemd  │
└──────────┘
```

---

## Technical Decisions

### Why SQLite?

- Single file database (easy backups)
- No separate database server needed
- Perfect for single-server deployments
- Built-in Go support
- Litestream provides replication

### Why templ + htmx?

- Server-side rendering (simple, fast)
- No build step for frontend
- Minimal JavaScript
- Progressive enhancement
- Works without JS enabled

### Why systemd?

- Battle-tested process supervisor
- Built into every modern Linux
- Free logging via journalctl
- Resource limits & restart policies
- Zero custom code needed

### Why Build on Server?

- Simpler UX (no CI/CD required)
- More flexible (build any commit)
- Still lightweight (~500MB for Go)
- Build caching speeds up rebuilds
- Can add pre-built binary support later

---

**Version:** 1.0
**Last Updated:** 2025-10-18
**Status:** Implementation starting
