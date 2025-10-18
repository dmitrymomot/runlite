# Task: Implement Basic Panel with Caddy Integration

## Objective

Create a minimal panel HTTP server with Caddy reverse proxy integration to verify the CLI tool works end-to-end. The panel should:
- Respond with "welcome to runlite" on `/`
- Provide a `/health` endpoint for status checks
- Run behind Caddy reverse proxy on the configured panel domain
- Support automatic HTTPS via Caddy

## Context

- **Prerequisite:** Complete the "Implement Base CLI Tool" task first
- **Purpose:** Validate that setup → start → status workflow functions correctly
- **This is a minimal implementation** - Full panel UI will be implemented later

## Architecture

```
User Request (https://panel.example.com)
    ↓
Caddy (port 80/443)
    ↓ reverse proxy
Panel Server (localhost:8080)
    ↓
"welcome to runlite"
```

## Implementation Tasks

### Task 1: Create internal/panel package

**File:** `internal/panel/server.go`

```go
package panel

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "runlite/internal/config"
)

// Serve starts the panel HTTP server
func Serve(cfg *config.Config, db *sql.DB) error {
    mux := http.NewServeMux()

    // Root handler - welcome message
    mux.HandleFunc("/", handleRoot)

    // Health check endpoint for status command
    mux.HandleFunc("/health", handleHealth)

    // Create server
    addr := fmt.Sprintf(":%d", cfg.PanelPort)
    srv := &http.Server{
        Addr:         addr,
        Handler:      mux,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Graceful shutdown handling
    go func() {
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
        <-sigChan

        log.Println("Shutting down panel server...")
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        if err := srv.Shutdown(ctx); err != nil {
            log.Printf("Server shutdown error: %v", err)
        }
    }()

    log.Printf("Panel server listening on %s", addr)
    return srv.ListenAndServe()
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
    }

    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    fmt.Fprint(w, "welcome to runlite")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    fmt.Fprint(w, `{"status":"ok"}`)
}
```

### Task 2: Create internal/proxy package for Caddy management

**File:** `internal/proxy/caddy.go`

```go
package proxy

import (
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"

    "runlite/internal/config"
)

// CaddyConfig represents the Caddy JSON configuration
type CaddyConfig struct {
    Apps CaddyApps `json:"apps"`
}

type CaddyApps struct {
    HTTP HTTPApp `json:"http"`
}

type HTTPApp struct {
    Servers map[string]Server `json:"servers"`
}

type Server struct {
    Listen []string `json:"listen"`
    Routes []Route  `json:"routes"`
}

type Route struct {
    Match   []Match   `json:"match"`
    Handle  []Handler `json:"handle"`
    Terminal bool     `json:"terminal,omitempty"`
}

type Match struct {
    Host []string `json:"host"`
}

type Handler struct {
    Handler   string     `json:"handler"`
    Upstreams []Upstream `json:"upstreams,omitempty"`
}

type Upstream struct {
    Dial string `json:"dial"`
}

// Start initializes Caddy with the panel route
func Start(cfg *config.Config) error {
    // Generate Caddy configuration
    caddyCfg := generatePanelConfig(cfg)

    // Write config to file
    configPath := "/etc/runlite/caddy.json"
    if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
        return fmt.Errorf("failed to create config directory: %w", err)
    }

    data, err := json.MarshalIndent(caddyCfg, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }

    if err := os.WriteFile(configPath, data, 0644); err != nil {
        return fmt.Errorf("failed to write config: %w", err)
    }

    // Start Caddy (assumes Caddy is installed)
    cmd := exec.Command("caddy", "start", "--config", configPath)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to start Caddy: %w", err)
    }

    return nil
}

// Stop gracefully stops Caddy
func Stop() error {
    cmd := exec.Command("caddy", "stop")
    return cmd.Run()
}

// Reload reloads Caddy configuration
func Reload() error {
    cmd := exec.Command("caddy", "reload", "--config", "/etc/runlite/caddy.json")
    return cmd.Run()
}

// generatePanelConfig creates Caddy config for the panel
func generatePanelConfig(cfg *config.Config) CaddyConfig {
    panelUpstream := fmt.Sprintf("localhost:%d", cfg.PanelPort)

    return CaddyConfig{
        Apps: CaddyApps{
            HTTP: HTTPApp{
                Servers: map[string]Server{
                    "srv0": {
                        Listen: []string{":80", ":443"},
                        Routes: []Route{
                            {
                                Match: []Match{
                                    {
                                        Host: []string{cfg.PanelDomain},
                                    },
                                },
                                Handle: []Handler{
                                    {
                                        Handler: "reverse_proxy",
                                        Upstreams: []Upstream{
                                            {Dial: panelUpstream},
                                        },
                                    },
                                },
                                Terminal: true,
                            },
                        },
                    },
                },
            },
        },
    }
}
```

### Task 3: Update start command to initialize Caddy

**File:** `internal/cli/start.go` (update)

```go
package cli

import (
    "fmt"
    "log"

    "github.com/urfave/cli/v2"
    "runlite/internal/config"
    "runlite/internal/database"
    "runlite/internal/panel"
    "runlite/internal/proxy"  // ADD THIS
)

func StartCommand() *cli.Command {
    return &cli.Command{
        Name:  "start",
        Usage: "Start the runlite panel server",
        Action: func(c *cli.Context) error {
            return runStart()
        },
    }
}

func runStart() error {
    // Load config
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("failed to load config: %w\nRun 'runlite setup' first", err)
    }

    // Connect to database
    db, err := database.Connect(cfg.DatabasePath)
    if err != nil {
        return fmt.Errorf("database connection failed: %w", err)
    }
    defer db.Close()

    // Start Caddy proxy (if panel domain is configured)
    if cfg.PanelDomain != "" {
        log.Println("Starting Caddy reverse proxy...")
        if err := proxy.Start(cfg); err != nil {
            // Warning only - panel can still work on localhost
            log.Printf("Warning: Failed to start Caddy: %v", err)
            log.Printf("Panel will only be accessible on localhost:%d", cfg.PanelPort)
        } else {
            log.Printf("Caddy started - panel accessible at https://%s", cfg.PanelDomain)
        }
    }

    // Start panel server (blocking)
    log.Printf("Starting panel server on :%d", cfg.PanelPort)
    return panel.Serve(cfg, db)
}
```

### Task 4: Add config.Load() implementation

**File:** `internal/config/config.go` (update)

```go
// Load reads configuration from database or environment variables
func Load() (*Config, error) {
    cfg := Defaults()

    // Determine database path
    dbPath := cfg.DatabasePath

    // Try to load from database
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
    defer db.Close()

    // Check if config table exists
    var count int
    err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='config'").Scan(&count)
    if err != nil || count == 0 {
        // Database not initialized - return defaults
        return cfg, nil
    }

    // Load from database
    dbCfg, err := LoadFromDB(db)
    if err != nil {
        // If load fails, return defaults
        return cfg, nil
    }

    return dbCfg, nil
}
```

**File:** `internal/config/store.go` (complete implementation)

```go
package config

import (
    "database/sql"
    "fmt"
    "strconv"
    "strings"
)

// Save stores configuration to SQLite config table
func Save(db *sql.DB, cfg *Config) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Store each config value
    configs := map[string]string{
        "panel.domain":             cfg.PanelDomain,
        "panel.port":               strconv.Itoa(cfg.PanelPort),
        "database.path":            cfg.DatabasePath,
        "github.client_id":         cfg.GitHubClientID,
        "github.client_secret":     cfg.GitHubClientSecret,
        "github.allowed_usernames": strings.Join(cfg.AllowedUsernames, ","),
        "paths.apps_dir":           cfg.AppsDir,
        "paths.builds_dir":         cfg.BuildsDir,
        "backups.enabled":          strconv.FormatBool(cfg.BackupsEnabled),
        "backups.destination":      cfg.BackupDestination,
    }

    for key, value := range configs {
        if err := set(tx, key, value); err != nil {
            return err
        }
    }

    return tx.Commit()
}

// LoadFromDB reads configuration from SQLite
func LoadFromDB(db *sql.DB) (*Config, error) {
    cfg := Defaults()

    var err error

    if cfg.PanelDomain, err = get(db, "panel.domain"); err != nil {
        cfg.PanelDomain = ""
    }

    if portStr, err := get(db, "panel.port"); err == nil {
        if port, err := strconv.Atoi(portStr); err == nil {
            cfg.PanelPort = port
        }
    }

    if cfg.GitHubClientID, err = get(db, "github.client_id"); err != nil {
        cfg.GitHubClientID = ""
    }

    if cfg.GitHubClientSecret, err = get(db, "github.client_secret"); err != nil {
        cfg.GitHubClientSecret = ""
    }

    if usernames, err := get(db, "github.allowed_usernames"); err == nil {
        cfg.AllowedUsernames = parseUsernames(usernames)
    }

    if cfg.AppsDir, err = get(db, "paths.apps_dir"); err != nil {
        cfg.AppsDir = "/var/lib/runlite/apps"
    }

    if cfg.BuildsDir, err = get(db, "paths.builds_dir"); err != nil {
        cfg.BuildsDir = "/var/lib/runlite/builds"
    }

    if enabledStr, err := get(db, "backups.enabled"); err == nil {
        cfg.BackupsEnabled = enabledStr == "true"
    }

    if cfg.BackupDestination, err = get(db, "backups.destination"); err != nil {
        cfg.BackupDestination = ""
    }

    return cfg, nil
}

// set stores a single config value in a transaction
func set(tx *sql.Tx, key, value string) error {
    _, err := tx.Exec(`
        INSERT INTO config (key, value, updated_at)
        VALUES (?, ?, CURRENT_TIMESTAMP)
        ON CONFLICT(key) DO UPDATE SET
            value = excluded.value,
            updated_at = CURRENT_TIMESTAMP
    `, key, value)
    return err
}

// get retrieves a single config value
func get(db *sql.DB, key string) (string, error) {
    var value string
    err := db.QueryRow(`SELECT value FROM config WHERE key = ?`, key).Scan(&value)
    return value, err
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

## File Structure After Implementation

```
runlite/
├── cmd/runlite/main.go
├── internal/
│   ├── cli/
│   │   ├── setup.go
│   │   ├── start.go           # Updated
│   │   ├── status.go
│   │   └── config_show.go
│   ├── config/
│   │   ├── config.go          # Updated with Load()
│   │   └── store.go           # Complete implementation
│   ├── database/
│   │   └── database.go
│   ├── panel/                 # NEW
│   │   └── server.go
│   └── proxy/                 # NEW
│       └── caddy.go
└── go.mod
```

## Testing the Implementation

### Prerequisites

Install Caddy:
```bash
# macOS
brew install caddy

# Linux
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy
```

### Test Workflow

```bash
# 1. Build runlite
go build -o runlite cmd/runlite/main.go

# 2. Run setup
./runlite setup
# Enter: panel.localhost (for local testing)

# 3. Start panel
./runlite start

# 4. Test in another terminal
./runlite status
# Should show: ✅ Panel is running

# 5. Test HTTP endpoint
curl http://localhost:8080
# Expected: welcome to runlite

curl http://localhost:8080/health
# Expected: {"status":"ok"}

# 6. Test via Caddy (if domain configured)
curl http://panel.localhost
# Expected: welcome to runlite
```

### Local Testing with /etc/hosts

For testing without real DNS:

```bash
# Add to /etc/hosts
sudo sh -c 'echo "127.0.0.1 panel.localhost" >> /etc/hosts'

# Then use panel.localhost as your domain in setup
```

## Expected Behavior

After running `runlite start`, you should see:

```
Starting Caddy reverse proxy...
Caddy started - panel accessible at https://panel.localhost
Panel server listening on :8080
```

Then:
- `http://localhost:8080/` → "welcome to runlite"
- `http://localhost:8080/health` → `{"status":"ok"}`
- `https://panel.localhost/` → "welcome to runlite" (via Caddy)
- `runlite status` → ✅ Panel is running

## Error Handling

If Caddy fails to start:
- Panel continues on localhost only
- Warning message logged
- Status command still works (checks localhost)

If database not configured:
- `runlite start` shows error: "Run 'runlite setup' first"

## Notes

1. **Caddy Installation:** Caddy must be installed separately
2. **Port 80/443:** Caddy needs permission to bind to these ports (run as root or use setcap)
3. **HTTPS:** Caddy automatically gets Let's Encrypt certificates for real domains
4. **Local Testing:** Use `.localhost` domains or edit `/etc/hosts`
5. **Graceful Shutdown:** Panel handles SIGTERM/SIGINT for clean shutdown
6. **Development Mode:** For development, you can skip Caddy and just use localhost:8080

## Granting Caddy Permissions (Linux)

To allow Caddy to bind to port 80/443 without root:

```bash
sudo setcap 'cap_net_bind_service=+ep' $(which caddy)
```

Or run runlite with sudo (not recommended for production):
```bash
sudo ./runlite start
```

## Next Steps

After verifying this works:
1. Implement GitHub OAuth authentication
2. Build actual panel UI with templ + htmx
3. Add app deployment functionality
4. Implement webhook handling

## Dependencies

Already added in previous task:
- `github.com/urfave/cli/v2`
- `github.com/mattn/go-sqlite3`

No additional dependencies needed.

---

**Status:** Ready to implement
**Estimated Time:** 1-2 hours
**Depends On:** Implement Base CLI Tool task
**Blocks:** Full panel UI implementation
