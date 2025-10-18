# Runlite - Lightweight Go PaaS

## Problem

Deploy Go applications with SQLite without Docker overhead. Existing solutions (Dokku, CapRover) require 2GB+ RAM just for infrastructure. We can do better.

## Solution

Web-based PaaS for deploying Go binaries:
- Zero Docker overhead
- Runs directly on Linux via systemd
- Built-in HTTPS (Caddy)
- SQLite backups (Litestream)
- GitHub webhook deployments
- Web UI for everything (zero SSH required)

## Architecture

### Tech Stack
- **Go 1.25+** - Main platform
- **SQLite** - App configurations storage
- **Caddy v2** - Reverse proxy + automatic HTTPS
- **Litestream** - SQLite backup to S3/local
- **systemd** - Process supervision
- **templ + htmx** - Web UI (server-side rendering)

### Process Management
- Each deployed app = systemd service
- Automatic restart on crash
- Logs via journalctl
- Resource limits via systemd

### Configuration Storage
- Internal SQLite database for runlite configs
- Each app gets: repo URL, port, domain, env vars
- TOML export/import for backup

### Deployment Flow
1. User adds app via web UI (repo URL + basic config)
2. Runlite generates GitHub webhook URL
3. On push/release, webhook triggers deployment
4. Download binary from GitHub release
5. Generate systemd unit
6. Update Caddy reverse proxy config
7. Start/restart service

## Roadmap

### Phase 1: MVP ✓ Foundation Complete

**Done:**
- [x] Installation script with systemd service
- [x] Basic HTTP server structure
- [x] GitHub Actions for releases

**In Progress - Web Control Panel:**
- [ ] SQLite database schema for apps
- [ ] Web UI framework (templ + htmx)
- [ ] Dashboard: list apps, status
- [ ] App creation form (repo URL, port, domain)
- [ ] Environment variables editor
- [ ] Logs viewer (journalctl integration)

**In Progress - Core Deployment:**
- [ ] GitHub webhook handler
- [ ] Download binary from releases
- [ ] Generate systemd unit for app
- [ ] Start/stop/restart via web UI
- [ ] Deployment status tracking

**In Progress - Reverse Proxy:**
- [ ] Caddy integration
- [ ] Auto-configure routes
- [ ] Automatic HTTPS

**In Progress - Backups:**
- [ ] Litestream setup
- [ ] Backup configuration via UI

**Validation:**
- [ ] Deploy helprun.io on runlite

### Phase 2: Polish

- [ ] Authentication (password or GitHub OAuth)
- [ ] Custom domains management
- [ ] Deployment history
- [ ] Health checks
- [ ] Email/webhook notifications

### Phase 3: Advanced

- [ ] Blue/green deployments
- [ ] Preview environments
- [ ] Multi-server support
- [ ] API for automation

## Technical Decisions

### MVP Scope
**IN:**
- Pre-built binaries only (from GitHub releases)
- Web UI only (no CLI)
- Simple restart deployments
- Manual domain configuration

**OUT (v2+):**
- Server-side builds
- CLI tools
- Blue/green deployments
- Preview deploys
- Build queues

### Open Questions
1. Web UI auth: Simple password, GitHub OAuth, or both?
2. Database migrations: How to handle during deploys?
3. Zero-downtime: Possible without containers?
4. DNS: Integrate providers or manual setup?
5. Secrets: Encrypted SQLite or env vars only?

---

**Current Status:** Foundation complete, starting web UI development
