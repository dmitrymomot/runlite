#!/usr/bin/env bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/var/lib/runlite"
CONFIG_DIR="/etc/runlite"
SYSTEMD_DIR="/etc/systemd/system"
SERVICE_NAME="runlite.service"
DOMAIN_REGISTRY_SERVICE_NAME="runlite-domain-registry.service"
LITESTREAM_SERVICE_NAME="litestream.service"
GITHUB_REPO="dmitrymomot/runlite"
BINARY_NAME="runlite"
DOMAIN_REGISTRY_BINARY_NAME="runlite-domain-registry"

# Installation flags
INSTALL_LITESTREAM=false

# Version to install (set to "latest" on main branch, specific version on release tags)
# Can be overridden with RUNLITE_VERSION environment variable
VERSION="${RUNLITE_VERSION:-latest}"

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --with-litestream)
                INSTALL_LITESTREAM=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Usage: $0 [--with-litestream]"
                exit 1
                ;;
        esac
    done
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root or with sudo"
        exit 1
    fi
}

check_requirements() {
    log_info "Checking system requirements..."

    local missing_tools=()

    for tool in systemctl curl tar apt-get; do
        if ! command -v "$tool" &> /dev/null; then
            missing_tools+=("$tool")
        fi
    done

    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_error "Please install them and try again"
        exit 1
    fi

    log_info "All required tools are available"
}

detect_platform() {
    local os
    local arch

    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)

    # Normalize architecture names
    case "$arch" in
        x86_64)
            arch="amd64"
            ;;
        aarch64|arm64)
            arch="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac

    # Only support Linux for now
    if [[ "$os" != "linux" ]]; then
        log_error "Unsupported operating system: $os (only Linux is supported)"
        exit 1
    fi

    echo "${os}_${arch}"
}

determine_version() {
    # If VERSION is already set to a specific version (not "latest"), use it
    if [[ "$VERSION" != "latest" ]]; then
        log_info "Installing version: $VERSION"
        echo "$VERSION"
        return
    fi

    # Otherwise, fetch the latest release from GitHub API
    log_info "Fetching latest release version..."

    local latest_version
    latest_version=$(curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" \
        | grep '"tag_name":' \
        | sed -E 's/.*"([^"]+)".*/\1/')

    if [[ -z "$latest_version" ]]; then
        log_error "Failed to fetch latest release version"
        log_warn "Falling back to 'latest' tag"
        echo "latest"
    else
        log_info "Latest version: $latest_version"
        echo "$latest_version"
    fi
}

install_dependencies() {
    log_info "Installing system dependencies..."

    apt-get update -qq

    log_info "Installing git..."
    apt-get install -y git

    log_info "System dependencies installed"
}

install_caddy() {
    log_info "Installing Caddy..."

    # Install required packages for adding repositories
    apt-get install -y debian-keyring debian-archive-keyring apt-transport-https curl

    # Add Caddy repository
    log_info "Adding Caddy repository..."
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list

    # Update package list and install Caddy
    apt-get update -qq
    log_info "Installing caddy package..."
    apt-get install -y caddy

    # Create Caddy config directory
    mkdir -p /etc/caddy

    # Copy Caddyfile template
    log_info "Installing Caddyfile template..."
    local caddyfile_url="https://raw.githubusercontent.com/${GITHUB_REPO}/main/scripts/config/Caddyfile.template"

    if ! curl -L -f -o /etc/caddy/Caddyfile "$caddyfile_url"; then
        log_warn "Failed to download Caddyfile template from GitHub"
        log_info "Creating basic Caddyfile..."

        cat > /etc/caddy/Caddyfile <<'EOF'
{
	# Global options
	admin off

	# Enable on-demand TLS with domain registry verification
	on_demand_tls {
		ask http://127.0.0.1:3000/domains/{host}/verify
		interval 2m
		burst 5
	}
}

# Default HTTPS handler for all domains
:443 {
	# Enable on-demand TLS
	tls {
		on_demand
	}

	# Security headers
	header {
		# Enable HSTS
		Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
		# Prevent clickjacking
		X-Frame-Options "SAMEORIGIN"
		# Prevent MIME type sniffing
		X-Content-Type-Options "nosniff"
		# Enable XSS protection
		X-XSS-Protection "1; mode=block"
		# Referrer policy
		Referrer-Policy "strict-origin-when-cross-origin"
		# Remove server header
		-Server
	}

	# Reverse proxy to applications based on domain
	reverse_proxy * {
		to http://127.0.0.1:8080

		# Preserve original host header
		header_up Host {host}
		header_up X-Real-IP {remote_host}
		header_up X-Forwarded-For {remote_host}
		header_up X-Forwarded-Proto {scheme}
		header_up X-Forwarded-Host {host}
	}
}

# HTTP handler - serve example app for IP-based access
:80 {
	# Serve example app for IP-based access or fallback
	root * /var/lib/runlite/apps/example
	file_server
	try_files {path} /index.html

	# Also support ACME HTTP-01 challenge for domain verification
	# This allows Let's Encrypt to verify domains via HTTP
}
EOF
    fi

    # Create Caddy home directory for autosave.json (config persistence)
    log_info "Creating Caddy data directory..."
    mkdir -p /var/lib/caddy/.local/share/caddy
    chown -R caddy:caddy /var/lib/caddy

    # Create custom systemd service with --resume flag for config persistence
    log_info "Creating Caddy systemd service with config persistence..."
    cat > /etc/systemd/system/caddy.service <<'EOF'
[Unit]
Description=Caddy Web Server (API Mode with Config Persistence)
Documentation=https://caddyserver.com/docs/
After=network-online.target
Wants=network-online.target systemd-networkd-wait-online.service

[Service]
Type=notify
User=caddy
Group=caddy
ExecStart=/usr/bin/caddy run --environ --resume
ExecReload=/usr/bin/caddy reload --force
TimeoutStopSec=5s
LimitNOFILE=1048576
LimitNPROC=512
PrivateTmp=true
ProtectSystem=full
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
EOF

    # Reload systemd to recognize new service
    systemctl daemon-reload

    # Enable and start Caddy
    log_info "Enabling and starting Caddy service..."
    systemctl enable caddy
    systemctl start caddy

    # Wait for Caddy to start
    sleep 2

    # Load initial configuration from Caddyfile
    log_info "Loading initial configuration..."
    if caddy load --config /etc/caddy/Caddyfile --adapter caddyfile 2>/dev/null; then
        log_info "✓ Initial configuration loaded successfully"
    else
        log_warn "Failed to load configuration via API, restarting service..."
        systemctl restart caddy
    fi

    log_info "Caddy installed and configured with persistent storage"
}

install_litestream() {
    if [[ "$INSTALL_LITESTREAM" != "true" ]]; then
        log_info "Skipping Litestream installation (use --with-litestream to install)"
        return
    fi

    log_info "Installing Litestream..."

    # Add Litestream repository
    log_info "Adding Litestream repository..."
    wget -qO - https://repo.litestream.io/gpg.key | gpg --dearmor | tee /usr/share/keyrings/litestream-archive-keyring.gpg > /dev/null
    echo "deb [signed-by=/usr/share/keyrings/litestream-archive-keyring.gpg] https://repo.litestream.io/debian/$(lsb_release -cs) stable main" | tee /etc/apt/sources.list.d/litestream.list

    # Update package list and install Litestream
    apt-get update -qq
    log_info "Installing litestream package..."
    apt-get install -y litestream

    # Create Litestream config directory
    mkdir -p /etc/litestream

    # Copy litestream.yml template
    log_info "Installing litestream.yml template..."
    local litestream_config_url="https://raw.githubusercontent.com/${GITHUB_REPO}/main/scripts/config/litestream.yml.template"

    if ! curl -L -f -o /etc/litestream/litestream.yml "$litestream_config_url"; then
        log_warn "Failed to download litestream.yml template from GitHub"
        log_info "Creating basic litestream.yml..."

        cat > /etc/litestream/litestream.yml <<'EOF'
# Litestream Configuration
# Documentation: https://litestream.io/reference/config/

dbs:
  # Runlite main system database (app metadata, deployments)
  - path: /var/lib/runlite/runlite.db
    replicas:
      - type: s3
        bucket: ${LITESTREAM_S3_BUCKET}
        path: runlite/db
        region: ${LITESTREAM_S3_REGION:-us-east-1}
        access-key-id: ${LITESTREAM_ACCESS_KEY_ID}
        secret-access-key: ${LITESTREAM_SECRET_ACCESS_KEY}
        sync-interval: 10s
        retention: 168h  # 7 days
        retention-check-interval: 1h
        snapshot-interval: 24h

  # Domain registry database
  - path: /var/lib/runlite/domains.db
    replicas:
      - type: s3
        bucket: ${LITESTREAM_S3_BUCKET}
        path: runlite/domains
        region: ${LITESTREAM_S3_REGION:-us-east-1}
        access-key-id: ${LITESTREAM_ACCESS_KEY_ID}
        secret-access-key: ${LITESTREAM_SECRET_ACCESS_KEY}
        sync-interval: 10s
        retention: 168h  # 7 days
        retention-check-interval: 1h
        snapshot-interval: 24h
EOF
    fi

    # Copy systemd service
    log_info "Installing Litestream systemd service..."
    local litestream_service_url="https://raw.githubusercontent.com/${GITHUB_REPO}/main/scripts/systemd/litestream.service"

    if ! curl -L -f -o "${SYSTEMD_DIR}/${LITESTREAM_SERVICE_NAME}" "$litestream_service_url"; then
        log_warn "Failed to download Litestream service file from GitHub"
        log_info "Creating basic service file..."

        cat > "${SYSTEMD_DIR}/${LITESTREAM_SERVICE_NAME}" <<EOF
[Unit]
Description=Litestream Replication Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/litestream replicate
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
EOF
    fi

    # Reload systemd daemon
    systemctl daemon-reload

    # Enable and start Litestream
    log_info "Enabling and starting Litestream service..."
    systemctl enable "$LITESTREAM_SERVICE_NAME"
    systemctl start "$LITESTREAM_SERVICE_NAME"

    log_info "Litestream installed and configured"
}

download_binary() {
    local version=$1
    local platform=$2

    log_info "Downloading runlite binary..."

    # Construct download URL
    # Expected format: https://github.com/dmitrymomot/runlite/releases/download/v0.1.0/runlite_linux_amd64.tar.gz
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/runlite_${platform}.tar.gz"

    log_info "Download URL: $download_url"

    local tmp_dir
    tmp_dir=$(mktemp -d)

    if ! curl -L -f -o "${tmp_dir}/runlite.tar.gz" "$download_url"; then
        log_error "Failed to download binary from $download_url"
        log_warn "Note: Make sure releases are published with binaries"
        rm -rf "$tmp_dir"
        exit 1
    fi

    log_info "Extracting binary..."
    tar -xzf "${tmp_dir}/runlite.tar.gz" -C "$tmp_dir"

    if [[ ! -f "${tmp_dir}/${BINARY_NAME}" ]]; then
        log_error "Binary not found in archive"
        rm -rf "$tmp_dir"
        exit 1
    fi

    log_info "Installing binary to ${INSTALL_DIR}/${BINARY_NAME}..."
    install -m 755 "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"

    rm -rf "$tmp_dir"
    log_info "Binary installed successfully"
}

download_domain_registry_binary() {
    local version=$1
    local platform=$2

    log_info "Downloading runlite-domain-registry binary..."

    # Construct download URL
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/runlite-domain-registry_${platform}.tar.gz"

    log_info "Download URL: $download_url"

    local tmp_dir
    tmp_dir=$(mktemp -d)

    if ! curl -L -f -o "${tmp_dir}/runlite-domain-registry.tar.gz" "$download_url"; then
        log_error "Failed to download domain registry binary from $download_url"
        log_warn "Note: Make sure releases are published with binaries"
        rm -rf "$tmp_dir"
        exit 1
    fi

    log_info "Extracting binary..."
    tar -xzf "${tmp_dir}/runlite-domain-registry.tar.gz" -C "$tmp_dir"

    if [[ ! -f "${tmp_dir}/${DOMAIN_REGISTRY_BINARY_NAME}" ]]; then
        log_error "Domain registry binary not found in archive"
        rm -rf "$tmp_dir"
        exit 1
    fi

    log_info "Installing binary to ${INSTALL_DIR}/${DOMAIN_REGISTRY_BINARY_NAME}..."
    install -m 755 "${tmp_dir}/${DOMAIN_REGISTRY_BINARY_NAME}" "${INSTALL_DIR}/${DOMAIN_REGISTRY_BINARY_NAME}"

    rm -rf "$tmp_dir"
    log_info "Domain registry binary installed successfully"
}

create_directories() {
    log_info "Creating required directories..."

    mkdir -p "$DATA_DIR"
    mkdir -p "${DATA_DIR}/apps"
    mkdir -p "$CONFIG_DIR"
    mkdir -p /etc/caddy

    if [[ "$INSTALL_LITESTREAM" == "true" ]]; then
        mkdir -p /etc/litestream
    fi

    chmod 755 "$DATA_DIR"
    chmod 755 "${DATA_DIR}/apps"
    chmod 755 "$CONFIG_DIR"

    log_info "Directories created"
}

install_example_app() {
    log_info "Installing example app..."

    # Create example app directory
    mkdir -p "${DATA_DIR}/apps/example"

    # Try to download example app from GitHub
    local example_url="https://raw.githubusercontent.com/${GITHUB_REPO}/main/scripts/config/example/index.html"

    if ! curl -L -f -o "${DATA_DIR}/apps/example/index.html" "$example_url"; then
        log_warn "Failed to download example app from GitHub"
        log_info "Creating embedded example app..."

        # Fallback: create embedded example app
        cat > "${DATA_DIR}/apps/example/index.html" <<'EXAMPLE_EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to Runlite</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gradient-to-br from-blue-50 to-indigo-100 min-h-screen flex items-center justify-center p-4">
    <div class="max-w-2xl mx-auto bg-white rounded-lg shadow-xl p-8">
        <div class="text-center mb-8">
            <h1 class="text-4xl font-bold text-gray-900 mb-4">
                Welcome to <span class="text-indigo-600">Runlite</span> 🚀
            </h1>
            <div class="flex items-center justify-center space-x-2 mb-4">
                <div class="w-3 h-3 bg-green-500 rounded-full animate-pulse"></div>
                <span class="text-lg text-gray-700">System Online</span>
            </div>
            <p class="text-gray-600">Your Go PaaS is up and running!</p>
        </div>

        <div class="space-y-4 mb-8">
            <div class="border-l-4 border-indigo-600 pl-4">
                <h3 class="font-semibold text-gray-900 mb-2">Quick Start</h3>
                <div class="bg-gray-900 text-gray-100 p-3 rounded font-mono text-sm">
                    runlite domain add yourdomain.com<br>
                    runlite app create myapp<br>
                    runlite deploy
                </div>
            </div>
        </div>

        <div class="text-center">
            <a href="https://github.com/dmitrymomot/runlite" target="_blank"
               class="inline-flex items-center px-6 py-3 border border-transparent text-base font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700">
                View Documentation
            </a>
        </div>

        <div class="mt-8 pt-6 border-t text-center text-sm text-gray-600">
            Powered by Caddy, Litestream, and Go
        </div>
    </div>
</body>
</html>
EXAMPLE_EOF
    fi

    # Set proper permissions
    chmod 644 "${DATA_DIR}/apps/example/index.html"

    log_info "✓ Example app installed at ${DATA_DIR}/apps/example/"
}

install_systemd_service() {
    log_info "Installing systemd service..."

    # Download service template from GitHub
    local service_url="https://raw.githubusercontent.com/${GITHUB_REPO}/main/scripts/runlite.service.template"

    if ! curl -L -f -o "${SYSTEMD_DIR}/${SERVICE_NAME}" "$service_url"; then
        log_error "Failed to download systemd service template"
        log_warn "Trying to create a basic service file..."

        # Fallback: create basic service file
        cat > "${SYSTEMD_DIR}/${SERVICE_NAME}" <<EOF
[Unit]
Description=Runlite - Lightweight Go PaaS
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=${INSTALL_DIR}/${BINARY_NAME}
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
    fi

    log_info "Reloading systemd daemon..."
    systemctl daemon-reload

    log_info "Enabling runlite service..."
    systemctl enable "$SERVICE_NAME"

    log_info "Starting runlite service..."
    systemctl start "$SERVICE_NAME"

    sleep 2

    log_info "Systemd service installed and started"
}

install_domain_registry_service() {
    log_info "Installing domain registry systemd service..."

    # Download service file from GitHub
    local service_url="https://raw.githubusercontent.com/${GITHUB_REPO}/main/scripts/systemd/runlite-domain-registry.service"

    if ! curl -L -f -o "${SYSTEMD_DIR}/${DOMAIN_REGISTRY_SERVICE_NAME}" "$service_url"; then
        log_warn "Failed to download domain registry service file from GitHub"
        log_info "Creating basic service file..."

        cat > "${SYSTEMD_DIR}/${DOMAIN_REGISTRY_SERVICE_NAME}" <<EOF
[Unit]
Description=Runlite Domain Registry Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/${DOMAIN_REGISTRY_BINARY_NAME}
Restart=always
RestartSec=5
User=root
Environment=PORT=3000

[Install]
WantedBy=multi-user.target
EOF
    fi

    log_info "Reloading systemd daemon..."
    systemctl daemon-reload

    log_info "Enabling domain registry service..."
    systemctl enable "$DOMAIN_REGISTRY_SERVICE_NAME"

    log_info "Starting domain registry service..."
    systemctl start "$DOMAIN_REGISTRY_SERVICE_NAME"

    sleep 2

    log_info "Domain registry service installed and started"
}

show_status() {
    log_info "Checking service status..."
    echo ""

    local all_services_ok=true

    # Check runlite service
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log_info "✓ runlite is running"
    else
        log_error "✗ runlite service failed to start"
        all_services_ok=false
    fi

    # Check domain registry service
    if systemctl is-active --quiet "$DOMAIN_REGISTRY_SERVICE_NAME"; then
        log_info "✓ runlite-domain-registry is running"
    else
        log_error "✗ runlite-domain-registry service failed to start"
        all_services_ok=false
    fi

    # Check Caddy service
    if systemctl is-active --quiet caddy; then
        log_info "✓ caddy is running"
    else
        log_error "✗ caddy service failed to start"
        all_services_ok=false
    fi

    # Check Litestream service if installed
    if [[ "$INSTALL_LITESTREAM" == "true" ]]; then
        if systemctl is-active --quiet "$LITESTREAM_SERVICE_NAME"; then
            log_info "✓ litestream is running"
        else
            log_error "✗ litestream service failed to start"
            all_services_ok=false
        fi
    fi

    echo ""

    if [[ "$all_services_ok" == "true" ]]; then
        log_info "Installation complete!"
        echo ""
        echo "Quick Start:"
        echo "  - Example app: http://$(hostname -I | awk '{print $1}' || echo 'localhost')"
        echo "  - Test locally: curl http://localhost"
        echo ""
        echo "Next steps:"
        echo "  - View runlite logs: journalctl -u runlite -f"
        echo "  - View domain registry logs: journalctl -u runlite-domain-registry -f"
        echo "  - View Caddy logs: journalctl -u caddy -f"
        if [[ "$INSTALL_LITESTREAM" == "true" ]]; then
            echo "  - View Litestream logs: journalctl -u litestream -f"
            echo "  - Configure Litestream: edit /etc/litestream/litestream.yml"
        fi
        echo "  - Configure Caddy: edit /etc/caddy/Caddyfile"
        echo "  - Restart services: systemctl restart runlite runlite-domain-registry caddy"
        echo ""
    else
        log_error "Some services failed to start"
        echo ""
        echo "Check logs with:"
        echo "  - journalctl -u runlite -n 50"
        echo "  - journalctl -u runlite-domain-registry -n 50"
        echo "  - journalctl -u caddy -n 50"
        if [[ "$INSTALL_LITESTREAM" == "true" ]]; then
            echo "  - journalctl -u litestream -n 50"
        fi
        echo ""
        exit 1
    fi
}

main() {
    log_info "Starting runlite installation..."
    echo ""

    parse_args "$@"

    check_root
    check_requirements

    local platform
    platform=$(detect_platform)
    log_info "Detected platform: $platform"

    local version
    version=$(determine_version)

    install_dependencies
    install_caddy
    install_litestream

    download_binary "$version" "$platform"
    download_domain_registry_binary "$version" "$platform"

    create_directories
    install_example_app

    install_systemd_service
    install_domain_registry_service

    echo ""
    show_status
}

main "$@"
