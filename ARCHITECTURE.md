# runlite Architecture

**Version:** 0.1.0 (MVP)
**Last Updated:** 2025-10-22

## Overview

runlite is a lightweight, self-hosted PaaS inspired by Dokku. It enables git push deployments with zero-downtime, managed by systemd and routed through an external Caddy server.

**Core Philosophy:** Keep it simple. Start as a tool for personal use, then grow for others.

## Current Implementation

### What Exists Today

```
cmd/
├── cli/                    # Main CLI application
│   └── main.go
└── domain-registry/        # Standalone HTTP service (port 3000)
    └── main.go

internal/
├── app/                    # Application metadata (app.json)
│   └── metadata.go
├── caddy/                  # Caddy Admin API client
│   ├── client.go
│   └── types.go
├── config/                 # Path utilities and validation
│   └── path.go
├── env/                    # Environment variable management
│   ├── storage.go          # File I/O operations
│   ├── parser.go           # Parse env format
│   ├── diff.go             # Compare env changes
│   └── types.go
└── registry/               # Domain registry (SQLite)
    ├── storage.go
    ├── api.go
    └── types.go

schema/                     # JSON Schema for runlite.yml validation
scripts/                    # Installation, config templates, systemd
```

### What's In Progress

- **Deployment orchestration** (`internal/deploy/`)
- **Git hooks** (post-receive)
- **Process management** (systemd integration)

### What's Planned

- Litestream backups
- Health checks
- Blue/green deployments
- Web UI

## Architecture Decisions

### 1. Configuration Storage

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

### 2. External Caddy

**Decision:** Use external Caddy, not embedded

**Implementation:**

- Caddy runs as separate systemd service
- runlite communicates via Admin API (localhost:2019)
- Routes identified by: `runlite-{appName}`

**Why:**

- ✅ Caddy manages its own HTTPS certificates
- ✅ Can restart runlite without affecting traffic
- ✅ Simpler than embedding

### 3. Git Push Deployment

**Decision:** Dokku-style git push, no GitHub webhooks (MVP)

**Flow:**

```
Developer → git push runlite main
         ↓
Git bare repo (/home/git/apps/{app}.git)
         ↓
post-receive hook
         ↓
runlite deploy {app} {commit}
```

**Why:**

- ✅ Instant feedback over SSH
- ✅ No public endpoint needed
- ✅ Works without internet
- ✅ Deploy uncommitted code (testing)

**Future:** Can add GitHub webhooks as optional feature

### 4. No Docker

**Decision:** systemd process management, no containers

**Why:**

- ✅ Less overhead
- ✅ Native binaries run faster
- ✅ Simpler to debug
- ✅ Direct filesystem access

**Implementation:**

- Each release → separate systemd unit
- Process isolation via systemd sandboxing
- Port allocation from range (8000-9000)

## System Architecture

### File Structure

```
/var/lib/runlite/
└── apps/
    └── {appName}/
        ├── app.json              # Metadata (name, created_at, updated_at, status)
        ├── env                   # Environment variables (KEY=value format)
        ├── databases/            # SQLite databases (auto-discovered)
        │   ├── main.db
        │   └── tenant_*.db
        └── releases/
            ├── {app}-{commit7}   # Current release binary
            └── {app}-{commit7}   # Previous release (for rollback)

/home/git/apps/
└── {appName}.git/                # Bare repository
    └── hooks/
        └── post-receive          # Deployment trigger

/etc/systemd/system/
├── runlite-domain-registry.service
└── runlite-{appName}-{commit7}.service   # Per-release units
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
      "release_id": "my-app-abc123d",
      "commit": "abc123def456789",
      "port": 8080,
      "status": "active",
      "binary_path": "/var/lib/runlite/apps/my-app/releases/my-app-abc123d",
      "deployed_at": "2025-10-22T14:30:00Z",
      "health_check_passed": true
    },
    {
      "release_id": "my-app-xyz789a",
      "commit": "xyz789abc012345",
      "port": 8081,
      "status": "standby",
      "binary_path": "/var/lib/runlite/apps/my-app/releases/my-app-xyz789a",
      "deployed_at": "2025-10-21T10:15:00Z",
      "health_check_passed": true
    },
    {
      "release_id": "my-app-def456b",
      "commit": "def456bcd789012",
      "port": 8082,
      "status": "failed",
      "binary_path": "/var/lib/runlite/apps/my-app/releases/my-app-def456b",
      "deployed_at": "2025-10-20T16:45:00Z",
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
- `stopped` - Old release, cleaned up

**Constraints:**

- Only one `active` deployment at a time
- Only one `standby` deployment at a time
- Failed deployments never become active
- Maximum 10 deployments kept in history

### Environment Variables Format

Plain text file at `/var/lib/runlite/apps/{appName}/env`:

```bash
PORT=8080
DATABASE_URL=./databases/main.db
SECRET_KEY=xxxxx
```

**Operations:**

- `env/storage.go` - Read/write file atomically
- `env/parser.go` - Parse KEY=value format
- `env/diff.go` - Compare changes between versions

## Deployment Flow (Planned)

### Blue/Green Deployment

```
1. Load app.json metadata
2. Clone repo to temp directory
3. Read runlite.yml from commit
4. Run build script (user-defined)
5. Create release directory
6. Copy artifacts to /var/lib/runlite/apps/{app}/releases/{app}-{commit}
7. Allocate free port (8000-9000 range)
8. Start systemd service on new port
9. Health check (poll /health endpoint)
10. Update Caddy route → switch traffic to new port
11. Update app.json with new deployment info
12. Wait drain period (30s)
13. Stop old previous release
```

**Result:** Zero-downtime deployment, previous release kept for instant rollback.

### Rollback

```
1. Load app.json metadata
2. Find standby deployment (rollback target)
3. Update Caddy route → switch to standby port
4. Swap active ↔ standby status in app.json
5. Move new active to front of deployments array
```

**Fast:** Standby deployment still running, just route switch + metadata update.

**Safe:** Only successful deployments (health_check_passed=true) can be standby.

## Module Architecture

### Layered Design

```
┌─────────────────────────────────────────┐
│         CLI Layer (Thin)                │
│  cmd/cli/commands/{category}/{action}   │
├─────────────────────────────────────────┤
│      Service Layer (Planned)            │
│  Business logic, orchestration          │
├─────────────────────────────────────────┤
│      Module Layer (Implemented)         │
│  ├── app/        App metadata           │
│  ├── caddy/      Caddy API client       │
│  ├── config/     Path utilities         │
│  ├── env/        Environment vars       │
│  └── registry/   Domain registry        │
└─────────────────────────────────────────┘
```

### Current Modules

#### app/ - Application Metadata

**Purpose:** Manage app.json lifecycle with deployment history tracking

**Pattern:** Manager-based access with controlled state transitions

**Key Types:**

```go
type Metadata struct {
    Name        string       `json:"name"`
    CreatedAt   time.Time    `json:"created_at"`
    UpdatedAt   time.Time    `json:"updated_at"`
    Deployments []Deployment `json:"deployments,omitempty"`
}

type Deployment struct {
    ReleaseID         string     `json:"release_id"`
    Commit            string     `json:"commit"`
    Port              int        `json:"port"`
    Status            string     `json:"status"` // active, standby, failed, stopped
    BinaryPath        string     `json:"binary_path"`
    DeployedAt        time.Time  `json:"deployed_at"`
    StoppedAt         *time.Time `json:"stopped_at,omitempty"`
    HealthCheckPassed bool       `json:"health_check_passed"`
    FailureReason     string     `json:"failure_reason,omitempty"`
}

type Manager struct {
    dataDir string
}
```

**Manager API:**

```go
// Basic operations
func NewManager(dataDir string) *Manager
func (m *Manager) Create(appName string) error
func (m *Manager) Load(appName string) (*Metadata, error)
func (m *Manager) Save(meta *Metadata) error
func (m *Manager) Exists(appName string) bool

// Deployment operations
func (m *Manager) AddDeployment(appName string, deployment Deployment) error
func (m *Manager) GetActiveDeployment(appName string) (*Deployment, error)
func (m *Manager) GetStandbyDeployment(appName string) (*Deployment, error)
func (m *Manager) Rollback(appName string) (*Deployment, error)
func (m *Manager) MarkDeploymentStopped(appName, releaseID string) error

// Query operations
func (m *Manager) ListDeployments(appName string) ([]Deployment, error)
func (m *Manager) GetDeploymentByReleaseID(appName, releaseID string) (*Deployment, error)
func (m *Manager) GetDeploymentsByStatus(appName, status string) ([]Deployment, error)
func (m *Manager) GetStats(appName string) (total, active, standby, failed, stopped int, err error)

// Maintenance
func (m *Manager) CleanupOldDeployments(appName string, olderThan time.Duration) error
```

**Why Manager Pattern:**

- ✅ Enforces business rules (only one active, only one standby)
- ✅ Automatic state transitions (active→standby→stopped)
- ✅ Atomic file operations (temp file → rename)
- ✅ Auto-cleanup (max 10 deployments in history)
- ✅ Prevents invalid state (failed deployments can't become active)

**Usage Example:**

```go
mgr := app.NewManager("/var/lib/runlite")

// Add successful deployment
mgr.AddDeployment("my-app", app.Deployment{
    ReleaseID:         "my-app-abc123d",
    Commit:            "abc123d",
    Port:              8080,
    Status:            app.DeploymentStatusActive,
    BinaryPath:        "/var/lib/runlite/apps/my-app/releases/my-app-abc123d",
    DeployedAt:        time.Now(),
    HealthCheckPassed: true,
})
// Automatically: previous active → standby, old standby → stopped

// Rollback
newActive, err := mgr.Rollback("my-app")
// Returns: standby deployment (now active)

// Get current
active, err := mgr.GetActiveDeployment("my-app")
```

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

#### caddy/ - Caddy Admin API

**Purpose:** Communicate with external Caddy server

**API:**

```go
type Client struct {
    baseURL string // http://localhost:2019
}

func (c *Client) AddRoute(routeID, domain string, port int) error
func (c *Client) UpdateRoute(routeID string, port int) error
func (c *Client) DeleteRoute(routeID string) error
```

**Route ID format:** `runlite-{appName}`

#### registry/ - Domain Registry

**Purpose:** Track domain → app mappings

**Storage:** SQLite (internal/registry/storage.go)

**Schema:**

```sql
CREATE TABLE domains (
    domain TEXT PRIMARY KEY,
    app_name TEXT NOT NULL,
    created_at DATETIME NOT NULL
);
```

**API:**

```go
func (s *Storage) Add(domain, appName string) error
func (s *Storage) Get(domain string) (string, error)
func (s *Storage) Delete(domain string) error
func (s *Storage) List(appName string) ([]Domain, error)
```

**Note:** This is the only SQLite database in runlite (for domains only). No migrations - schema in code.

#### config/ - Path Utilities

**Purpose:** Validate and construct file paths

**API:**

```go
func AppDir(dataDir, appName string) string
func ValidateAppName(name string) error
```

**Pattern:** Prevent path traversal attacks

## Key Patterns

### 1. Atomic File Operations

All file writes follow:

```go
// Write to temp
tmpFile := path + ".tmp"
os.WriteFile(tmpFile, data, 0644)

// Atomic rename
os.Rename(tmpFile, path)
```

**Why:** Prevents corruption on crashes

### 2. Validation at Boundaries

```go
// Before any operation
if err := config.ValidateAppName(name); err != nil {
    return err
}
```

**Validation rules:**

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

## Deployment Flow Implementation Checklist

See [CLAUDE.md](./CLAUDE.md#deployment-flow---implementation-checklist) for detailed task breakdown.

**Summary:**

### Phase 1: Core Deployment Orchestrator

- [ ] Deploy() function - main orchestration
- [ ] runlite.yml parser/validator
- [ ] User build script executor
- [ ] Port allocation
- [ ] Process starter
- [ ] Health check poller
- [ ] Caddy route updater
- [ ] Metadata persistence
- [ ] Error handling & rollback

### Phase 2: Git Integration

- [ ] Post-receive hook template
- [ ] Hook installer
- [ ] Push-to-deploy trigger

### Phase 3: Process Management

- [ ] PID tracking
- [ ] Process stop/restart
- [ ] Systemd service generator
- [ ] Service lifecycle management

### Phase 4: Advanced Features

- [ ] Blue/green deployment
- [ ] Zero-downtime switching
- [ ] Deployment history/audit
- [ ] Rollback capability

## Testing Strategy

### Unit Tests

- Each module independently testable
- Mock external dependencies (Caddy API, filesystem)

### Integration Tests

- Full deployment flow
- Git hook triggers
- Caddy route updates

### Manual Testing

- Deploy real Go apps
- Test rollback scenarios
- Verify zero-downtime

## Security Considerations

### App Isolation

- Each app runs as separate systemd unit
- Process sandboxing via systemd directives
- Port range isolation (8000-9000)

### Input Validation

- App names sanitized (prevent path traversal)
- Domain validation (prevent injection)
- Environment variable limits

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

- Simple range scan (8000-9000)
- Fast for <1000 apps
- Consider port tracking file if scaling needed

### File Operations

- Atomic renames (no locks needed)
- Sequential consistency guaranteed

### Caddy API

- HTTP calls are synchronous
- Consider retry logic for production

## Future Enhancements

### v0.2

- TUI with Bubble Tea
- GitHub webhooks (optional)
- Litestream integration

### v0.3

- Web dashboard
- Multi-user support
- Resource limits (CPU/memory)

### v1.0

- Production-ready
- Full test coverage
- Documentation complete

## References

- [CLAUDE.md](./CLAUDE.md) - Development guidelines and implementation checklist
- [README.md](./README.md) - User-facing documentation
- [schema/](./schema/) - runlite.yml JSON Schema

## Questions?

See [CLAUDE.md](./CLAUDE.md) for development context and patterns.
