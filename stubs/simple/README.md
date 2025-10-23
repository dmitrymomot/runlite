# Simple Setup Template

Single-app deployment with Caddy reverse proxy and Litestream database replication.

## What Gets Installed

- **Caddy** (v2.7+) - Modern web server with automatic HTTPS
- **Litestream** (v0.3.13) - SQLite replication for backups

Both installed as static binaries to `/usr/local/bin/`.

## Prerequisites

- Linux with systemd
- Root/sudo access
- amd64 or arm64 architecture
- No specific distribution required (works on Debian, Ubuntu, RHEL, Alpine, etc.)

## Installation

```bash
sudo installer install --template=simple
```

Or if using built-in templates:

```bash
sudo runlite setup --template=simple
```

## Configuration

### 1. Configure Domain

Edit `/etc/caddy/Caddyfile` to add your domain:

```
myapp.example.com {
    reverse_proxy localhost:8000
}
```

Reload Caddy to apply changes:

```bash
sudo systemctl reload caddy
```

**For local testing without a domain:**

The default config includes a local listener on port 8080:

```
:8080 {
    reverse_proxy localhost:8000
}
```

### 2. Configure S3 Backups (Optional)

Litestream includes a local file backup by default at `/var/lib/runlite/backups/`.

For S3 backups, create `/etc/litestream.env`:

```bash
LITESTREAM_S3_BUCKET=my-backup-bucket
LITESTREAM_S3_REGION=us-east-1
LITESTREAM_ACCESS_KEY_ID=your-access-key
LITESTREAM_SECRET_ACCESS_KEY=your-secret-key
```

Restart Litestream:

```bash
sudo systemctl restart litestream
```

## Usage

### Create and Deploy Your First App

```bash
# Create app
runlite app create myapp

# Deploy from git
runlite deploy myapp --ref main
```

Your app will be available at:

- **With domain**: https://myapp.example.com (after DNS configured)
- **Local testing**: http://localhost:8080

## Files and Locations

### Binaries

- `/usr/local/bin/caddy` - Caddy web server
- `/usr/local/bin/litestream` - Litestream replication

### Configuration

- `/etc/caddy/Caddyfile` - Caddy configuration (static, edit manually)
- `/etc/litestream.yml` - Litestream configuration
- `/etc/litestream.env` - S3 credentials (optional, create manually)

### Data

- `/var/lib/runlite/` - runlite data directory
- `/var/lib/runlite/runlite.db` - Main SQLite database
- `/var/lib/runlite/backups/` - Local backup replica

### Systemd Services

- `caddy.service` - Caddy web server
- `litestream.service` - Litestream replication

## Service Management

```bash
# Check service status
sudo systemctl status caddy
sudo systemctl status litestream

# Restart services
sudo systemctl restart caddy
sudo systemctl restart litestream

# View logs
sudo journalctl -u caddy -f
sudo journalctl -u litestream -f

# Reload Caddy config (no downtime)
sudo systemctl reload caddy
```

## Differences from Multi-Tenant Template

This **simple** template is for single-app or manual setups:

- ✓ Static Caddyfile (edit manually)
- ✓ No domain registry service
- ✓ Simpler setup, fewer moving parts
- ✓ Good for personal projects

The **multi-tenant** template adds:

- Dynamic domain management
- On-demand TLS
- Domain registry service
- Automatic route updates

## Troubleshooting

### Caddy Not Starting

Check configuration syntax:

```bash
caddy validate --config /etc/caddy/Caddyfile
```

View detailed logs:

```bash
sudo journalctl -xeu caddy
```

### Litestream Not Replicating

Check S3 credentials if using S3:

```bash
sudo systemctl status litestream
sudo journalctl -u litestream -n 50
```

Verify local backup exists:

```bash
ls -lh /var/lib/runlite/backups/
```

### Cannot Access App

1. Check if your app is running:

    ```bash
    runlite app info myapp
    ```

2. Check if Caddy can reach your app:

    ```bash
    curl localhost:8000
    ```

3. Check Caddy logs:
    ```bash
    sudo journalctl -u caddy -n 100
    ```

## Upgrading

To upgrade Caddy or Litestream:

1. Download new binary manually
2. Stop the service
3. Replace the binary
4. Start the service

Example for Caddy:

```bash
sudo systemctl stop caddy
sudo curl -o /usr/local/bin/caddy https://caddyserver.com/api/download?os=linux&arch=amd64
sudo chmod +x /usr/local/bin/caddy
sudo systemctl start caddy
```

## Next Steps

- [Deploying Your First App](https://runlite.dev/docs/deploying)
- [runlite.yml Specification](https://runlite.dev/docs/spec)
- [Multi-Tenant Setup](https://runlite.dev/docs/multi-tenant)
