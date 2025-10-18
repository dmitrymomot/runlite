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

- **Go 1.25+** - Main platform + on-server builds
- **SQLite** - App configurations storage
- **Caddy v2** - Reverse proxy + automatic HTTPS
- **Litestream** - SQLite backup to S3/local
- **systemd** - Process supervision
- **templ + htmx** - Web UI (server-side rendering)
- **GitHub OAuth** - Authentication (single predefined username)

### Process Management

- Each deployed app = systemd service
- Automatic restart on crash
- Logs via journalctl
- Resource limits via systemd

### Configuration Storage

- Internal SQLite database for runlite configs
- Each app gets: repo URL, port, domain, env vars
- TOML export/import for backup

### Authentication

- GitHub OAuth during installation
- Single predefined username in config (`GITHUB_USERNAME=yourusername`)
- OAuth token stored securely for GitHub API access
- No password management needed

### Deployment Flow

1. User logs in via GitHub OAuth
2. Select repo from their GitHub repos list (via GitHub API)
3. Configure: domain, port, env vars
4. Runlite auto-creates GitHub webhook
5. On push to main branch:
    - Clone repo on server
    - Build Go binary on server (`go build`)
    - Generate/update systemd unit
    - Update Caddy reverse proxy config
    - Restart service
6. Build logs shown in web UI

## Roadmap

### Phase 1: MVP ✓ Foundation Complete

**Done:**

- [x] Installation script with systemd service
- [x] Basic HTTP server structure
- [x] GitHub Actions for releases

**In Progress - Authentication:**

- [ ] GitHub OAuth integration
- [ ] Store allowed username in config
- [ ] Secure token storage
- [ ] Login/logout flow

**In Progress - Web Control Panel:**

- [ ] SQLite database schema for apps
- [ ] Web UI framework (templ + htmx)
- [ ] Dashboard: list apps, status, build logs
- [ ] GitHub repos list (via API)
- [ ] App creation: select repo, configure domain/port/env
- [ ] Environment variables editor
- [ ] Logs viewer (build logs + journalctl)

**In Progress - Core Deployment:**

- [ ] Install Go compiler during runlite installation
- [ ] GitHub webhook handler
- [ ] Clone repository on webhook trigger
- [ ] Build Go binary on server (`go build`)
- [ ] Handle build errors and show in UI
- [ ] Generate systemd unit for app
- [ ] Start/stop/restart via web UI
- [ ] Deployment status and build logs tracking
- [ ] Auto-create GitHub webhook via API

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

- [ ] Multiple allowed GitHub usernames (team support)
- [ ] Custom domains management
- [ ] Deployment history with rollback
- [ ] Health checks and uptime monitoring
- [ ] Email/webhook notifications
- [ ] Pre-built binary support (optional alternative to builds)

### Phase 3: Advanced

- [ ] Blue/green deployments
- [ ] Preview environments
- [ ] Multi-server support
- [ ] API for automation

## Technical Decisions

### ✅ Decided

- **Authentication:** GitHub OAuth with single predefined username
- **Builds:** On-server compilation (install Go compiler during setup)
- **Repo Selection:** Browse GitHub repos via API (no manual URLs)
- **Interface:** Web UI only (no CLI tools)
- **Deployments:** Simple restart (no blue/green for MVP)
- **Webhooks:** Auto-created via GitHub API

### MVP Scope

**IN:**

- On-server Go builds
- GitHub OAuth authentication
- GitHub repos list integration
- Auto-webhook creation
- Build logs in UI
- Web UI only (no CLI)

**OUT (v2+):**

- Multiple users/teams
- Pre-built binaries as alternative
- Blue/green deployments
- Preview deploys
- Build queues

### Open Questions

1. Database migrations: Run automatically or manual trigger?
2. Zero-downtime: Worth implementing without containers?
3. Build caching: Use Go build cache to speed up rebuilds?
4. Secrets: Encrypted in SQLite or env vars only?
5. Dependencies: Handle private Go modules?

---

**Current Status:** Foundation complete, starting web UI development
