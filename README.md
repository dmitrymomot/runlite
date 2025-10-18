# runlite

A lightweight, self-hosted PaaS (Platform as a Service) for Go applications and static websites.

## Features

- 🚀 Single binary deployment
- 🔒 Built-in HTTPS with automatic certificates (via Caddy)
- 💾 SQLite-first with automated backups (via Litestream)
- 🐙 GitHub webhook integration for auto-deployment
- 🔧 Zero Docker overhead
- 📦 Simple systemd process management

## Installation

### Quick Install (Latest Version)

Install the latest version of runlite on a fresh Linux server:

```bash
curl -1sLf 'https://raw.githubusercontent.com/dmitrymomot/runlite/main/scripts/install.sh' | sudo -E bash
```

### Install Specific Version

To install a specific version, use the install script from that version's tag:

```bash
# Install v1.0.14
curl -1sLf 'https://raw.githubusercontent.com/dmitrymomot/runlite/v1.0.14/scripts/install.sh' | sudo -E bash
```

Or override the version using an environment variable:

```bash
# Install v1.0.14 using main branch script
curl -1sLf 'https://raw.githubusercontent.com/dmitrymomot/runlite/main/scripts/install.sh' | RUNLITE_VERSION=v1.0.14 sudo -E bash
```

### Requirements

- Linux (amd64 or arm64)
- systemd
- Root access (for service installation)

### Manual Installation

If you prefer to install manually:

1. Download the latest release from [GitHub Releases](https://github.com/dmitrymomot/runlite/releases)
2. Extract and install the binary:
   ```bash
   tar -xzf runlite_linux_amd64.tar.gz
   sudo install -m 755 runlite /usr/local/bin/runlite
   ```
3. Create required directories:
   ```bash
   sudo mkdir -p /var/lib/runlite/apps
   sudo mkdir -p /etc/runlite
   ```
4. Download and install the systemd service:
   ```bash
   sudo curl -L -o /etc/systemd/system/runlite.service \
     https://raw.githubusercontent.com/dmitrymomot/runlite/main/scripts/runlite.service.template
   ```
5. Enable and start the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable runlite
   sudo systemctl start runlite
   ```

## Usage

### Check Status

```bash
# Check if runlite is running
sudo systemctl status runlite

# View logs
sudo journalctl -u runlite -f
```

### Test the Server

```bash
curl http://localhost:8080
# Should return: welcome to runlite
```

### Stop/Restart

```bash
# Stop the service
sudo systemctl stop runlite

# Restart the service
sudo systemctl restart runlite
```

## Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/dmitrymomot/runlite.git
cd runlite

# Build the binary
go build -o runlite ./cmd/runlite

# Run locally
./runlite
```

### Running Tests

```bash
go test ./...
```

## Architecture

- **Embedded Caddy**: Reverse proxy with automatic HTTPS
- **Embedded Litestream**: SQLite database replication and backups
- **systemd**: Process supervision for deployed applications
- **GitHub Webhooks**: Automated deployment on push

## Roadmap

### Phase 1: MVP (Current)
- [x] Basic HTTP server
- [x] Installation script
- [ ] GitHub webhook handler
- [ ] Caddy integration
- [ ] Litestream setup
- [ ] Application deployment
- [ ] systemd unit generation

### Phase 2: Polish
- [ ] Custom domains API
- [ ] Web UI for management
- [ ] Server-side builds
- [ ] Rollback functionality
- [ ] Health checks
- [ ] Metrics/monitoring

### Phase 3: Advanced
- [ ] Blue/green deployments
- [ ] Preview environments
- [ ] Multi-server support
- [ ] Marketplace/plugins

## Release Process

Releases are automated via GitHub Actions. To create a new release:

1. **Create and push a version tag:**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **GitHub Actions automatically:**
   - Builds binaries for Linux (amd64 and arm64)
   - Updates the VERSION variable in `install.sh`
   - Creates a GitHub Release with:
     - Pre-built binaries
     - SHA256 checksums
     - Updated installation script
     - Installation instructions

3. **The release is immediately available:**
   ```bash
   # Users can install the specific version
   curl -1sLf 'https://raw.githubusercontent.com/dmitrymomot/runlite/v1.0.0/scripts/install.sh' | sudo -E bash
   ```

### Version Format

Follow [Semantic Versioning](https://semver.org/):
- `v1.0.0` - Major release
- `v1.1.0` - Minor release (new features)
- `v1.0.1` - Patch release (bug fixes)

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

### Local Development

```bash
# Build and test locally
go build -o runlite ./cmd/runlite
./runlite --version

# Run tests
go test ./...

# Test with custom version
go build -ldflags "-X main.version=v1.2.3" -o runlite ./cmd/runlite
./runlite --version
```

### Testing Installation After Release

After creating a release, you can test the installation in a clean Docker environment:

```bash
# Start a Debian container with systemd
docker run -d \
  --name runlite-test \
  --privileged \
  -v /sys/fs/cgroup:/sys/fs/cgroup:ro \
  debian:bookworm-slim \
  /bin/bash -c "apt-get update -qq && apt-get install -y -qq systemd curl sudo && exec /lib/systemd/systemd"

# Wait for systemd to start
sleep 3

# Test the installation
docker exec -it runlite-test bash -c "\
  curl -1sLf 'https://raw.githubusercontent.com/dmitrymomot/runlite/v0.0.1/scripts/install.sh' | sudo -E bash"

# Test the service
docker exec -it runlite-test curl http://localhost:8080

# Clean up
docker rm -f runlite-test
```

## License

[License TBD]

## Author

Created by [@dmitrymomot](https://github.com/dmitrymomot)
