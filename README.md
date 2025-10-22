# runlite

A lightweight, self-hosted PaaS for deploying applications via git push (Dokku-style).

**Status:** Early development (MVP phase) - spec parser complete, deployment orchestrator next.

## Features

- 🚀 **Git push deployment** - Deploy with `git push runlite main`
- 🔒 **HTTPS** - Automatic certificates via external Caddy
- 🔧 **No Docker** - Native processes managed by systemd
- 📦 **Single binary** - One stateless CLI for all operations
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
runlite app create my-app

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
Generate systemd unit
    ↓
systemctl start
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
├── app.json              # App metadata (deployment history)
├── env                   # Environment variables (plain text)
├── git/                  # Bare git repository
│   └── hooks/
│       └── post-receive  # Deployment trigger
└── releases/
    ├── 20251022143055-a3f5c2b/   # ReleaseID: {timestamp}-{commit-hash}
    │   ├── server                # Built artifacts
    │   └── migrations/
    └── 20251022150320-d8f9e1a/   # Previous release (for rollback)
        └── ...
```

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for technical details.

**Key Patterns:**
- **Stateless CLI**: Commands execute and exit - systemd manages app processes
- **Configuration-as-Code**: App metadata in JSON, environment in plain text
- **External Caddy**: Separate systemd service, controlled via Admin API
- **Atomic operations**: `write temp → rename` prevents corruption
- **Blue/green deployments**: Active/standby pattern for zero-downtime

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

**✅ Implemented:**
- CLI structure (app, domain, env commands)
- App metadata with Manager pattern (active/standby/failed/stopped states)
- Caddy client (add/update/delete routes, zero-downtime updates)
- Domain registry (SQLite storage)
- Config utilities (path validation, security)
- Environment variable management (CRUD, diff, parser)
- **Spec parser (runlite.yml → Go structs, validation, defaults, 93.9% test coverage)**

**🚧 Next to Implement:**
- Deployment orchestrator (port allocation, systemd unit generation, build execution, health checks)
- Git hooks (post-receive trigger)

**📋 Future Work:**
- Litestream backups (automated SQLite backups)
- GitHub webhooks (optional alternative to git push)
- Web UI

See [CLAUDE.md](./CLAUDE.md) for detailed implementation plan.

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
