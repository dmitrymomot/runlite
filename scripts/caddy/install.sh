#!/usr/bin/env bash
set -euo pipefail

# Caddy Installation Script for RunLite
# Installs Caddy web server with a simple welcome page

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Configuration
CADDY_VERSION="latest"
CADDY_USER="caddy"
CADDY_GROUP="caddy"
INSTALL_DIR="/usr/bin"
CONFIG_DIR="/etc/caddy"
DATA_DIR="/var/lib/caddy"
LOG_DIR="/var/log/caddy"
APPS_DIR="/apps"
WELCOME_DIR="${APPS_DIR}/welcome"

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

detect_platform() {
    local os arch

    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case $(uname -m) in
        x86_64)
            arch="amd64"
            ;;
        aarch64|arm64)
            arch="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac

    echo "${os}_${arch}"
}

check_dependencies() {
    local deps=("curl" "tar")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            log_error "Required dependency not found: $dep"
            exit 1
        fi
    done
}

download_caddy() {
    local platform=$1
    local temp_dir=$(mktemp -d)

    log_info "Detecting latest Caddy version..."
    local latest_url="https://api.github.com/repos/caddyserver/caddy/releases/latest"
    CADDY_VERSION=$(curl -s "$latest_url" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/')

    if [[ -z "$CADDY_VERSION" ]]; then
        log_error "Failed to detect latest Caddy version"
        exit 1
    fi

    log_info "Downloading Caddy v${CADDY_VERSION} for ${platform}..."
    local download_url="https://github.com/caddyserver/caddy/releases/download/v${CADDY_VERSION}/caddy_${CADDY_VERSION}_${platform}.tar.gz"

    curl -L -o "${temp_dir}/caddy.tar.gz" "$download_url" || {
        log_error "Failed to download Caddy"
        rm -rf "$temp_dir"
        exit 1
    }

    log_info "Extracting Caddy..."
    tar -xzf "${temp_dir}/caddy.tar.gz" -C "$temp_dir" caddy || {
        log_error "Failed to extract Caddy"
        rm -rf "$temp_dir"
        exit 1
    }

    log_info "Installing Caddy to ${INSTALL_DIR}..."
    mv "${temp_dir}/caddy" "${INSTALL_DIR}/caddy"
    chmod +x "${INSTALL_DIR}/caddy"

    rm -rf "$temp_dir"

    log_info "Caddy installed: $(${INSTALL_DIR}/caddy version)"
}

create_user() {
    if id "$CADDY_USER" &>/dev/null; then
        log_warn "User $CADDY_USER already exists, skipping creation"
    else
        log_info "Creating system user: $CADDY_USER"
        useradd --system --home-dir /var/lib/caddy --shell /usr/sbin/nologin "$CADDY_USER" || {
            log_error "Failed to create user"
            exit 1
        }
    fi
}

setup_directories() {
    log_info "Setting up directories..."

    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR"
    mkdir -p "$LOG_DIR"
    mkdir -p "$WELCOME_DIR"

    chown -R "${CADDY_USER}:${CADDY_GROUP}" "$CONFIG_DIR"
    chown -R "${CADDY_USER}:${CADDY_GROUP}" "$DATA_DIR"
    chown -R "${CADDY_USER}:${CADDY_GROUP}" "$LOG_DIR"
    chown -R "${CADDY_USER}:${CADDY_GROUP}" "$APPS_DIR"

    log_info "Directories created and permissions set"
}

grant_capabilities() {
    log_info "Granting CAP_NET_BIND_SERVICE capability..."

    if command -v setcap &> /dev/null; then
        setcap 'cap_net_bind_service=+ep' "${INSTALL_DIR}/caddy" || {
            log_warn "Failed to set capabilities, Caddy may not be able to bind to port 80/443"
        }
    else
        log_warn "setcap not found, install libcap2-bin (Debian/Ubuntu) or libcap (RHEL/CentOS)"
    fi
}

install_config() {
    log_info "Installing Caddy configuration..."

    if [[ -f "${SCRIPT_DIR}/Caddyfile.default" ]]; then
        cp "${SCRIPT_DIR}/Caddyfile.default" "${CONFIG_DIR}/Caddyfile"
        chown "${CADDY_USER}:${CADDY_GROUP}" "${CONFIG_DIR}/Caddyfile"
    else
        log_error "Caddyfile.default not found in ${SCRIPT_DIR}"
        exit 1
    fi
}

install_welcome_page() {
    log_info "Installing welcome page..."

    if [[ -f "${SCRIPT_DIR}/welcome.html" ]]; then
        cp "${SCRIPT_DIR}/welcome.html" "${WELCOME_DIR}/index.html"
        chown "${CADDY_USER}:${CADDY_GROUP}" "${WELCOME_DIR}/index.html"
    else
        log_error "welcome.html not found in ${SCRIPT_DIR}"
        exit 1
    fi
}

install_systemd_service() {
    log_info "Installing systemd service..."

    if [[ -f "${SCRIPT_DIR}/caddy.service" ]]; then
        cp "${SCRIPT_DIR}/caddy.service" /etc/systemd/system/caddy.service
        systemctl daemon-reload
        systemctl enable caddy.service
    else
        log_error "caddy.service not found in ${SCRIPT_DIR}"
        exit 1
    fi
}

start_caddy() {
    log_info "Starting Caddy..."
    systemctl start caddy.service

    sleep 2

    if systemctl is-active --quiet caddy.service; then
        log_info "Caddy started successfully"
    else
        log_error "Caddy failed to start"
        log_error "Check logs with: journalctl -u caddy -n 50"
        exit 1
    fi
}

validate_installation() {
    log_info "Validating installation..."

    local max_attempts=5
    local attempt=1

    while [[ $attempt -le $max_attempts ]]; do
        if curl -f -s http://localhost:80 > /dev/null; then
            log_info "Caddy is responding on port 80"
            return 0
        fi

        log_warn "Attempt $attempt/$max_attempts: Caddy not responding yet..."
        sleep 2
        ((attempt++))
    done

    log_error "Caddy is not responding after $max_attempts attempts"
    log_error "Check status with: systemctl status caddy"
    exit 1
}

print_summary() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "Caddy installation completed successfully!"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "  Configuration:  ${CONFIG_DIR}/Caddyfile"
    echo "  Data directory: ${DATA_DIR}"
    echo "  Log directory:  ${LOG_DIR}"
    echo "  Apps directory: ${APPS_DIR}"
    echo ""
    echo "  Service status: systemctl status caddy"
    echo "  View logs:      journalctl -u caddy -f"
    echo "  Reload config:  systemctl reload caddy"
    echo ""
    echo "  Welcome page:   http://localhost"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

main() {
    log_info "Starting Caddy installation for RunLite..."

    check_root
    check_dependencies

    local platform=$(detect_platform)
    log_info "Detected platform: $platform"

    download_caddy "$platform"
    create_user
    setup_directories
    grant_capabilities
    install_config
    install_welcome_page
    install_systemd_service
    start_caddy
    validate_installation
    print_summary
}

main "$@"
