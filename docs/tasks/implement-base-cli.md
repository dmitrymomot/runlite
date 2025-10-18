# Task: Implement Base CLI Tool

## Objective

Implement a CLI tool for runlite using `urfave/cli/v2` with the following commands:

- `runlite setup` - Interactive configuration wizard
- `runlite start` (or just `runlite`) - Start the panel server
- `runlite status` - Check if panel is running
- `runlite config show` - Display current configuration

## Context

- **Project:** runlite - lightweight PaaS for Go applications
- **Current state:** Basic HTTP server in `cmd/runlite/main.go`
- **Architecture:** See [docs/architecture.md](../architecture.md)
- **Related docs:** [docs/app-isolation.md](../app-isolation.md)

## Technical Decisions

1. **CLI Framework:** `urfave/cli/v2` (simpler than cobra, feature-complete)
2. **Config Storage:** SQLite `config` table (single source of truth)
3. **Config Location:** `/var/lib/runlite/runlite.db` (or `./runlite.db` for development)
4. **Default Command:** `start` (running `runlite` with no args starts the panel)

## Implementation Tasks

### Task 1: Add urfave/cli dependency

```bash
go get github.com/urfave/cli/v2
```

### Task 2: Create internal/config package

**File:** `internal/config/config.go`

```go
package config

import (
    "database/sql"
    "fmt"
    "os"
)

type Config struct {
    // Panel settings
    PanelDomain string
    PanelPort   int

    // Database
    DatabasePath string

    // GitHub OAuth
    GitHubClientID     string
    GitHubClientSecret string
    AllowedUsernames   []string  // Multiple users with full access

    // Paths
    AppsDir   string // /var/lib/runlite/apps
    BuildsDir string // /var/lib/runlite/builds

    // Optional features
    BackupsEnabled    bool
    BackupDestination string
}

// Load reads configuration from database or environment variables
func Load() (*Config, error) {
    // Implementation here
}

// Defaults returns a Config with default values
func Defaults() *Config {
    return &Config{
        PanelPort:    8080,
        DatabasePath: getEnv("DATABASE_PATH", "/var/lib/runlite/runlite.db"),
        AppsDir:      "/var/lib/runlite/apps",
        BuildsDir:    "/var/lib/runlite/builds",
    }
}

func getEnv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}
```

**File:** `internal/config/store.go`

```go
package config

import (
    "database/sql"
    "fmt"
)

// Save stores configuration to SQLite config table
func Save(db *sql.DB, cfg *Config) error {
    // Store each config value as key-value pairs in config table
    // For AllowedUsernames, store as comma-separated string
    // Example: Set(db, "github.allowed_usernames", "user1,user2,user3")
}

// LoadFromDB reads configuration from SQLite
func LoadFromDB(db *sql.DB) (*Config, error) {
    // Read config table and populate Config struct
    // For AllowedUsernames, parse comma-separated string into []string
    // Example: strings.Split(value, ",")
}

// Set stores a single config value
func Set(db *sql.DB, key, value string) error {
    _, err := db.Exec(`
        INSERT INTO config (key, value, updated_at)
        VALUES (?, ?, CURRENT_TIMESTAMP)
        ON CONFLICT(key) DO UPDATE SET
            value = excluded.value,
            updated_at = CURRENT_TIMESTAMP
    `, key, value)
    return err
}

// Get retrieves a single config value
func Get(db *sql.DB, key string) (string, error) {
    var value string
    err := db.QueryRow(`SELECT value FROM config WHERE key = ?`, key).Scan(&value)
    return value, err
}
```

### Task 3: Implement database initialization

**File:** `internal/database/database.go`

```go
package database

import (
    "database/sql"
    "fmt"

    _ "github.com/mattn/go-sqlite3"
)

// Connect opens SQLite database and ensures schema exists
func Connect(path string) (*sql.DB, error) {
    db, err := sql.Open("sqlite3", path)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // Enable foreign keys
    if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
        return nil, err
    }

    // Run migrations
    if err := Migrate(db); err != nil {
        return nil, fmt.Errorf("migration failed: %w", err)
    }

    return db, nil
}

// Migrate applies database schema
func Migrate(db *sql.DB) error {
    schema := `
    CREATE TABLE IF NOT EXISTS config (
        key TEXT PRIMARY KEY,
        value TEXT NOT NULL,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    -- Add other tables from architecture.md later
    `

    _, err := db.Exec(schema)
    return err
}
```

### Task 4: Implement 'runlite setup' command

**File:** `internal/cli/setup.go`

```go
package cli

import (
    "bufio"
    "fmt"
    "os"
    "strings"

    "github.com/urfave/cli/v2"
    "runlite/internal/config"
    "runlite/internal/database"
)

func SetupCommand() *cli.Command {
    return &cli.Command{
        Name:  "setup",
        Usage: "Configure runlite (interactive)",
        Action: func(c *cli.Context) error {
            return runSetup()
        },
    }
}

func runSetup() error {
    fmt.Println("🚀 Runlite Setup")
    fmt.Println()

    // Load or create config
    cfg := config.Defaults()

    // Interactive prompts
    cfg.PanelDomain = prompt("Panel domain (e.g., panel.example.com)", "")
    cfg.GitHubClientID = prompt("GitHub OAuth Client ID", "")
    cfg.GitHubClientSecret = promptSecret("GitHub OAuth Client Secret")

    // Prompt for allowed GitHub usernames (comma-separated)
    usernames := prompt("Allowed GitHub usernames (comma-separated)", "")
    cfg.AllowedUsernames = parseUsernames(usernames)

    // Optional: backup configuration
    if promptYesNo("Enable automatic backups?", false) {
        cfg.BackupsEnabled = true
        cfg.BackupDestination = prompt("Backup destination (e.g., s3://bucket/path)", "")
    }

    // Save to database
    db, err := database.Connect(cfg.DatabasePath)
    if err != nil {
        return fmt.Errorf("database connection failed: %w", err)
    }
    defer db.Close()

    if err := config.Save(db, cfg); err != nil {
        return fmt.Errorf("failed to save config: %w", err)
    }

    fmt.Println()
    fmt.Println("✅ Setup complete!")
    fmt.Printf("   Panel will be available at: https://%s\n", cfg.PanelDomain)
    fmt.Println()
    fmt.Println("Next steps:")
    fmt.Println("  1. Ensure DNS points to this server")
    fmt.Println("  2. Run: runlite start")

    return nil
}

func prompt(label, defaultValue string) string {
    reader := bufio.NewReader(os.Stdin)
    if defaultValue != "" {
        fmt.Printf("%s [%s]: ", label, defaultValue)
    } else {
        fmt.Printf("%s: ", label)
    }

    input, _ := reader.ReadString('\n')
    input = strings.TrimSpace(input)

    if input == "" {
        return defaultValue
    }
    return input
}

func promptSecret(label string) string {
    // TODO: Use terminal package to hide input
    return prompt(label, "")
}

func promptYesNo(label string, defaultValue bool) bool {
    defaultStr := "n"
    if defaultValue {
        defaultStr = "y"
    }

    result := prompt(fmt.Sprintf("%s (y/n)", label), defaultStr)
    return strings.ToLower(result) == "y"
}

func parseUsernames(input string) []string {
    // Split by comma and trim whitespace
    parts := strings.Split(input, ",")
    usernames := make([]string, 0, len(parts))
    for _, part := range parts {
        if username := strings.TrimSpace(part); username != "" {
            usernames = append(usernames, username)
        }
    }
    return usernames
}
```

### Task 5: Implement 'runlite start' command

**File:** `internal/cli/start.go`

```go
package cli

import (
    "fmt"
    "log"

    "github.com/urfave/cli/v2"
    "runlite/internal/config"
    "runlite/internal/database"
    "runlite/internal/panel"
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

    // Start panel server
    log.Printf("Starting runlite panel on :%d", cfg.PanelPort)
    if cfg.PanelDomain != "" {
        log.Printf("Panel URL: https://%s", cfg.PanelDomain)
    }

    // TODO: Initialize Caddy proxy
    // TODO: Initialize backup service if enabled

    return panel.Serve(cfg, db)
}
```

### Task 6: Implement 'runlite status' command

**File:** `internal/cli/status.go`

```go
package cli

import (
    "fmt"
    "net/http"
    "time"

    "github.com/urfave/cli/v2"
    "runlite/internal/config"
)

func StatusCommand() *cli.Command {
    return &cli.Command{
        Name:  "status",
        Usage: "Check if runlite panel is running",
        Action: func(c *cli.Context) error {
            return runStatus()
        },
    }
}

func runStatus() error {
    cfg, err := config.Load()
    if err != nil {
        fmt.Println("❌ Runlite not configured")
        fmt.Println("   Run: runlite setup")
        return nil
    }

    // Check if panel is responding
    url := fmt.Sprintf("http://localhost:%d/health", cfg.PanelPort)
    client := &http.Client{Timeout: 2 * time.Second}

    resp, err := client.Get(url)
    if err != nil {
        fmt.Println("❌ Panel is not running")
        fmt.Println("   Run: runlite start")
        return nil
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusOK {
        fmt.Println("✅ Panel is running")
        if cfg.PanelDomain != "" {
            fmt.Printf("   URL: https://%s\n", cfg.PanelDomain)
        }
        fmt.Printf("   Port: %d\n", cfg.PanelPort)
    } else {
        fmt.Printf("⚠️  Panel responded with status %d\n", resp.StatusCode)
    }

    return nil
}
```

### Task 7: Implement 'runlite config show' command

**File:** `internal/cli/config_show.go`

```go
package cli

import (
    "fmt"
    "strings"

    "github.com/urfave/cli/v2"
    "runlite/internal/config"
)

func ConfigCommand() *cli.Command {
    return &cli.Command{
        Name:  "config",
        Usage: "Configuration management",
        Subcommands: []*cli.Command{
            {
                Name:  "show",
                Usage: "Display current configuration",
                Action: func(c *cli.Context) error {
                    return showConfig()
                },
            },
        },
    }
}

func showConfig() error {
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    fmt.Println("Current Configuration:")
    fmt.Println()
    fmt.Printf("  Panel Domain:    %s\n", cfg.PanelDomain)
    fmt.Printf("  Panel Port:      %d\n", cfg.PanelPort)
    fmt.Printf("  Database:        %s\n", cfg.DatabasePath)
    fmt.Printf("  Apps Directory:  %s\n", cfg.AppsDir)
    fmt.Printf("  Builds Directory:%s\n", cfg.BuildsDir)
    fmt.Println()
    fmt.Printf("  GitHub Usernames:%s\n", strings.Join(cfg.AllowedUsernames, ", "))
    fmt.Printf("  GitHub Client ID:%s\n", cfg.GitHubClientID)
    fmt.Println()
    if cfg.BackupsEnabled {
        fmt.Printf("  Backups:         Enabled\n")
        fmt.Printf("  Backup Dest:     %s\n", cfg.BackupDestination)
    } else {
        fmt.Printf("  Backups:         Disabled\n")
    }

    return nil
}
```

### Task 8: Update main.go

**File:** `cmd/runlite/main.go`

```go
package main

import (
    "log"
    "os"

    "github.com/urfave/cli/v2"
    "runlite/internal/cli"
)

var version = "dev"

func main() {
    app := &cli.App{
        Name:    "runlite",
        Usage:   "Lightweight PaaS for Go applications",
        Version: version,
        Commands: []*cli.Command{
            cli.SetupCommand(),
            cli.StartCommand(),
            cli.StatusCommand(),
            cli.ConfigCommand(),
        },
        // Default action: start the panel
        Action: func(c *cli.Context) error {
            // If no command specified, run start
            return cli.RunStart()
        },
    }

    if err := app.Run(os.Args); err != nil {
        log.Fatal(err)
    }
}
```

## File Structure After Implementation

```
runlite/
├── cmd/runlite/main.go          # Updated CLI entry point
├── internal/
│   ├── cli/                     # NEW: CLI commands
│   │   ├── setup.go
│   │   ├── start.go
│   │   ├── status.go
│   │   └── config_show.go
│   ├── config/                  # NEW: Configuration management
│   │   ├── config.go
│   │   └── store.go
│   ├── database/                # NEW: Database operations
│   │   └── database.go
│   └── panel/                   # To be implemented later
│       └── server.go
└── go.mod
```

## Testing the Implementation

```bash
# Build
go build -o runlite cmd/runlite/main.go

# Run setup
./runlite setup

# Start panel
./runlite start

# Check status (in another terminal)
./runlite status

# Show config
./runlite config show
```

## Dependencies to Add

```bash
go get github.com/urfave/cli/v2
go get github.com/mattn/go-sqlite3
```

## Notes

1. **Error Handling:** All commands should provide helpful error messages
2. **Database Path:** Use environment variable `DATABASE_PATH` or default to `/var/lib/runlite/runlite.db`
3. **Development Mode:** For local development, use `./runlite.db`
4. **Panel Server:** The `internal/panel` package needs a `Serve(cfg, db)` function (implement later)
5. **Health Endpoint:** Panel server needs a `/health` endpoint for status checks
6. **Secret Input:** Consider using `golang.org/x/term` for hiding password input in setup
7. **Authentication:** The `internal/auth` package should validate GitHub username against `AllowedUsernames` slice. All allowed users have full access (no RBAC). Example validation:
   ```go
   func IsAllowed(username string, allowedUsernames []string) bool {
       for _, allowed := range allowedUsernames {
           if username == allowed {
               return true
           }
       }
       return false
   }
   ```

## Future Enhancements

- `runlite apps` - List deployed apps
- `runlite apps:create` - Deploy new app
- `runlite apps:delete` - Remove app
- `runlite logs <app>` - View app logs
- `runlite config set <key> <value>` - Update single config value

## References

- [Architecture Documentation](../architecture.md)
- [App Isolation Documentation](../app-isolation.md)
- [urfave/cli Documentation](https://cli.urfave.org/)

---

**Status:** Ready to implement
**Estimated Time:** 2-3 hours
**Priority:** High (blocking other features)
