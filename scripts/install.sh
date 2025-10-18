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
GITHUB_REPO="dmitrymomot/runlite"
BINARY_NAME="runlite"

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

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root or with sudo"
        exit 1
    fi
}

check_requirements() {
    log_info "Checking system requirements..."

    local missing_tools=()

    for tool in systemctl curl tar; do
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

create_directories() {
    log_info "Creating required directories..."

    mkdir -p "$DATA_DIR"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "${DATA_DIR}/apps"

    chmod 755 "$DATA_DIR"
    chmod 755 "$CONFIG_DIR"

    log_info "Directories created"
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

show_status() {
    log_info "Checking service status..."

    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log_info "✓ runlite is running"

        # Show service status
        echo ""
        systemctl status "$SERVICE_NAME" --no-pager -l

        echo ""
        log_info "Installation complete!"
        echo ""
        echo "Next steps:"
        echo "  - View logs: journalctl -u runlite -f"
        echo "  - Test the server: curl http://localhost:8080"
        echo "  - Stop service: systemctl stop runlite"
        echo "  - Restart service: systemctl restart runlite"
        echo ""
    else
        log_error "runlite service failed to start"
        log_info "Check logs with: journalctl -u runlite -n 50"
        exit 1
    fi
}

main() {
    log_info "Starting runlite installation..."
    echo ""

    check_root
    check_requirements

    local platform
    platform=$(detect_platform)
    log_info "Detected platform: $platform"

    local version
    version=$(determine_version)

    download_binary "$version" "$platform"
    create_directories
    install_systemd_service

    echo ""
    show_status
}

main "$@"
