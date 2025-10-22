# runlite

A lightweight, self-hosted PaaS for deploying applications via git push (Dokku-style).

**Status:** Early development (MVP phase) - Core infrastructure established, deployment orchestration in progress.

## Features

- 🚀 **Git push deployment** - Deploy with `git push runlite main`
- 🔒 **HTTPS** - Automatic certificates via external Caddy
- 🔧 **No Docker** - Native processes managed by systemd
- 📦 **Single binary** - One CLI for all operations
- ⚡ **Zero-downtime deploys** - Blue/green deployment ready
- 💾 **SQLite backups** - Automated via Litestream (planned)

## Quick Start

### Installation

```bash
# Install latest version (when released)
curl -1sLf 'https://raw.githubusercontent.com/dmitrymomot/runlite/main/scripts/install.sh' | sudo -E bash
```

### Requirements

- Linux (amd64 or arm64)
- systemd
- External Caddy server (for reverse proxy)

### Create Your First App

```bash
# Create app
runlite app:create my-app

# Add git remote
git remote add runlite git@server:~/apps/my-app.git

# Deploy
git push runlite main
```

## How It Works

runlite uses a **git push deployment** flow similar to Dokku:

```
git push runlite main
    ↓
Git hook triggers
    ↓
Read runlite.yml (from your repo)
    ↓
Run build script
    ↓
Start on random port
    ↓
Health check
    ↓
Update Caddy routes
    ↓
Route traffic (zero-downtime)
```

### App Configuration

Create `runlite.yml` in your app repository:

```yaml
app:
  name: my-app

build:
  script: |
    go build -o ./bin/server .
  artifacts:
    - ./bin/server

run:
  command: ./bin/server

health:
  path: /health
  timeout: 30s
```

## Project Structure

```
/var/lib/runlite/apps/{appName}/
├── app.json              # App metadata (name, status, timestamps)
├── env                   # Environment variables (plain text)
├── databases/            # SQLite databases (auto-backup planned)
└── releases/
    ├── my-app-abc123     # Current release
    └── my-app-xyz789     # Previous release (for rollback)

/home/git/apps/{appName}.git/
└── hooks/post-receive    # Deployment trigger
```

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for technical details.

**Key Patterns:**
- **Configuration-as-Code**: App metadata in JSON, environment in plain text
- **External Caddy**: Separate systemd service, controlled via Admin API
- **Atomic operations**: `write temp → rename` prevents corruption
- **Layered architecture**: CLI → Service Layer (planned) → Modules

## Development

### Building from Source

```bash
git clone https://github.com/dmitrymomot/runlite.git
cd runlite

# Required after every code change
task format check

# Build check (silent, for error checking only)
go build -o runlite cmd/cli/main.go > /dev/null 2>&1
```

### Running Tests

```bash
go test ./...
```

## Contributing

Contributions welcome! Please read [CLAUDE.md](./CLAUDE.md) for development guidelines.

**Key commands:**
```bash
task format check    # REQUIRED after every code change
task sqlc            # Generate SQL queries
task mocks           # Generate test mocks
task migration       # Create migration
```

## Roadmap

**✅ Implemented:** CLI structure, env management, app metadata, Caddy client, domain registry, config utilities

**🚧 In Progress:** Git hooks, deployment orchestration

**📋 Planned:** Litestream backups, health checks, systemd generation, Web UI

See [CLAUDE.md](./CLAUDE.md) for detailed implementation checklist.

## Release Process

```bash
git tag v1.0.0
git push origin v1.0.0
```

GitHub Actions will automatically:
- Build binaries for Linux (amd64 and arm64)
- Create GitHub Release with binaries and checksums
- Update installation script

## License

[License TBD]

## Author

Created by [@dmitrymomot](https://github.com/dmitrymomot)
