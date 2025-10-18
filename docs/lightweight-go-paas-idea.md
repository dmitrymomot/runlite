# Lightweight Go + SQLite PaaS

## Problem Statement

Current deployment solutions (Dokku, CapRover, etc.) rely on Docker, which introduces significant overhead for Go applications with SQLite databases:

- **Resource overhead**: Dokku on Ubuntu requires 2GB+ server, empty landing page uses 50% of resources
- **IO performance**: Docker volumes add latency for SQLite file operations
- **Cost impact**: $5 VPS insufficient, need $20+ VPS just for basic deployment
- **Complexity**: Multiple layers (Docker, nginx/reverse proxy, orchestration) for simple apps

### Real-world example

Single Go landing page with one input (waiting list) + SQLite:

- Dokku + Ubuntu + Let's Encrypt plugin + nginx + Docker volume mounting
- Uses ~1GB RAM just for infrastructure
- Overpriced for simple use case

## Market Context

**Trend: SQLite Renaissance**

- Developers moving back from Postgres/MySQL to SQLite
- Modern tools: Turso, Litestream, LiteFS gaining traction
- Cloud database costs pushing back to local-first approach

**Target audience**:

- Go developers building with SQLite
- Solo developers and small teams
- Cost-conscious indie hackers
- Anyone running multi-tenant apps with SQLite per tenant

## Solution

Standalone server management binary with zero Docker overhead:

### Core Features

1. **Single Binary Deployment**
    - One executable for entire PaaS
    - Easy server setup
    - No containers, no VM overhead

2. **Built-in Components**
    - Caddy reverse proxy with automatic HTTPS
    - Litestream for SQLite backups
    - Process supervision and auto-restart

3. **GitHub Integration**
    - Auto-deploy via GitHub webhooks
    - Add repo URL → minimal config → production app

4. **Multi-tenant Support**
    - Internal API for custom domain management
    - Perfect for SaaS apps with customer domains

5. **Environment Management**
    - Simple env variable configuration per app
    - Secrets management

6. **Advanced Deployment** (future)
    - Preview deploys
    - Blue/green deployment
    - Check new version before rollout

## Technical Approach

### MVP Architecture (v1 - 4-6 weeks)

**Scope IN:**

- Deploy single Go binary via GitHub webhook
- Caddy reverse proxy with auto-certs
- Litestream backup configuration
- Environment variables management
- Basic process supervision (restart on crash)

**Scope OUT (defer to v2+):**

- ❌ Blue/green deploy → simple restart with new binary
- ❌ Custom domains API → hardcode in config file
- ❌ Preview deploys → too complex for MVP
- ❌ Build queue → synchronous deploys sufficient

### Key Architecture Decisions

**Build Process:**

- **v1**: Accept pre-built binaries (faster to ship)
- **v2**: Optional server-side builds

**Process Management:**

- **Recommendation**: systemd units
- Battle-tested, zero custom code
- Built-in restart, logging, resource limits

**Configuration:**

- **Format**: TOML per app
- Simple, readable, version-controllable

**Interface:**

- **v1**: CLI + config files
- **v2**: Optional REST API

### Tech Stack

- Go 1.25+
- Caddy v2 (embedded or managed process)
- Litestream (embedded or managed process)
- systemd for process supervision

## Competitive Analysis

**Existing Solutions:**

- **Dokku**: Docker-based, 2GB+ overhead
- **CapRover**: Docker-based, complex setup
- **Coolify**: Docker-based, resource-intensive
- **Kamal 2.0**: Docker-based deployment

**Our Advantage:**

- Zero Docker overhead
- SQLite-first design
- Go community native
- Direct file I/O performance
- Lower resource requirements = lower costs

## Business Model

### Open Core Strategy

**Free (Open Source):**

- Core deployment functionality
- GitHub integration
- Caddy + Litestream integration
- Basic process management

**Paid Options:**

1. **Premium Support**
    - Priority bug fixes
    - Setup assistance
    - Custom feature development

2. **Commercial License** (optional)
    - For companies wanting support SLA
    - White-label options

3. **Hosted Version** (future)
    - Managed control plane
    - Multi-server orchestration

### Pricing Research Needed

- What would developers pay vs. Dokku?
- GitHub Sponsors vs. commercial license?
- Support retainer model?

## Validation Steps

### Before Building

1. **Community Feedback**
    - Post on /r/golang: "Would you pay for Dokku without Docker?"
    - Tweet the concept, gauge response
    - HackerNews "Show HN" draft

2. **User Interviews**
    - Talk to 5+ Go devs running SQLite apps
    - Understand current pain points
    - Pricing sensitivity research

3. **Dogfooding**
    - Migrate helprun.io to use this system
    - Document pain points and wins
    - Real-world performance testing

### Success Metrics

- Deploy helprun.io successfully
- Resource usage < 512MB for simple apps
- Deploy time < 30 seconds
- 10+ GitHub stars in first week
- 3+ external users in first month

## Roadmap

### Phase 1: MVP (4-6 weeks)

- [ ] Core binary deployment
- [ ] GitHub webhook handler
- [ ] Caddy integration
- [ ] Litestream setup
- [ ] systemd unit generation
- [ ] Basic CLI
- [ ] Deploy helprun.io on it

### Phase 2: Polish (2-3 months)

- [ ] Custom domains API
- [ ] Web UI for management
- [ ] Build process on server
- [ ] Rollback functionality
- [ ] Health checks
- [ ] Metrics/monitoring

### Phase 3: Advanced (6+ months)

- [ ] Blue/green deploys
- [ ] Preview environments
- [ ] Multi-server support
- [ ] Distributed Litestream
- [ ] Marketplace/plugins

## Open Questions

1. Build on server or accept pre-built binaries only?
2. How to handle database migrations during deploys?
3. Zero-downtime restarts possible without containers?
4. DNS management - integrate with providers or manual?
5. Secrets management - encrypted config files or separate vault?
6. Multi-server orchestration in future?

## Project Name Ideas

Need something memorable (not generic like "go-deploy"):

- **Ideas TBD**
- Should convey: lightweight, Go, simple, fast
- Check domain availability

## Next Steps

1. ✅ Document idea
2. Create new repository
3. Validate with Go community
4. Design MVP architecture
5. Start building

---

**Status**: Idea stage - pending validation
**Owner**: Personal side project
**Target**: Go community + self-use
