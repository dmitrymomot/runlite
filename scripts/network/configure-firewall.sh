#!/usr/bin/env bash
set -euo pipefail

# Firewall Configuration Script for RunLite
# Configures firewall to allow HTTP, HTTPS, and SSH access

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Ports to configure
HTTP_PORT=80
HTTPS_PORT=443
SSH_PORT=22

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

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

detect_firewall() {
    if command -v ufw &> /dev/null && ufw status &> /dev/null; then
        echo "ufw"
    elif command -v firewall-cmd &> /dev/null && systemctl is-active --quiet firewalld; then
        echo "firewalld"
    elif command -v iptables &> /dev/null; then
        echo "iptables"
    else
        echo "none"
    fi
}

configure_ufw() {
    log_step "Configuring UFW firewall..."

    # Check if UFW is installed
    if ! command -v ufw &> /dev/null; then
        log_error "UFW is not installed"
        return 1
    fi

    # Allow SSH first to prevent lockout
    log_info "Allowing SSH (port ${SSH_PORT})..."
    ufw allow ${SSH_PORT}/tcp comment 'SSH access for git push' || log_warn "Failed to add SSH rule"

    # Allow HTTP
    log_info "Allowing HTTP (port ${HTTP_PORT})..."
    ufw allow ${HTTP_PORT}/tcp comment 'HTTP web traffic' || log_warn "Failed to add HTTP rule"

    # Allow HTTPS
    log_info "Allowing HTTPS (port ${HTTPS_PORT})..."
    ufw allow ${HTTPS_PORT}/tcp comment 'HTTPS web traffic' || log_warn "Failed to add HTTPS rule"

    # Enable UFW if not already enabled
    if ! ufw status | grep -q "Status: active"; then
        log_warn "UFW is not enabled. Enabling now..."
        echo "y" | ufw enable || log_error "Failed to enable UFW"
    else
        log_info "UFW is already enabled"
    fi

    # Reload UFW to apply changes
    ufw reload &> /dev/null || true

    log_info "UFW configuration completed"
}

configure_firewalld() {
    log_step "Configuring firewalld..."

    # Check if firewalld is installed
    if ! command -v firewall-cmd &> /dev/null; then
        log_error "firewalld is not installed"
        return 1
    fi

    # Start firewalld if not running
    if ! systemctl is-active --quiet firewalld; then
        log_warn "firewalld is not running. Starting now..."
        systemctl start firewalld || log_error "Failed to start firewalld"
        systemctl enable firewalld || log_warn "Failed to enable firewalld"
    fi

    # Allow SSH
    log_info "Allowing SSH (port ${SSH_PORT})..."
    firewall-cmd --permanent --add-service=ssh || log_warn "Failed to add SSH service"
    firewall-cmd --permanent --add-port=${SSH_PORT}/tcp || log_warn "Failed to add SSH port"

    # Allow HTTP
    log_info "Allowing HTTP (port ${HTTP_PORT})..."
    firewall-cmd --permanent --add-service=http || log_warn "Failed to add HTTP service"
    firewall-cmd --permanent --add-port=${HTTP_PORT}/tcp || log_warn "Failed to add HTTP port"

    # Allow HTTPS
    log_info "Allowing HTTPS (port ${HTTPS_PORT})..."
    firewall-cmd --permanent --add-service=https || log_warn "Failed to add HTTPS service"
    firewall-cmd --permanent --add-port=${HTTPS_PORT}/tcp || log_warn "Failed to add HTTPS port"

    # Reload firewall to apply changes
    log_info "Reloading firewall configuration..."
    firewall-cmd --reload || log_error "Failed to reload firewall"

    log_info "firewalld configuration completed"
}

configure_iptables() {
    log_step "Configuring iptables..."

    # Check if iptables is installed
    if ! command -v iptables &> /dev/null; then
        log_error "iptables is not installed"
        return 1
    fi

    log_warn "Direct iptables configuration detected"
    log_warn "This script will add rules, but they may not persist after reboot"
    log_warn "Consider installing ufw or firewalld for easier management"

    # Allow established connections
    iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT &> /dev/null || true

    # Allow loopback
    iptables -A INPUT -i lo -j ACCEPT &> /dev/null || true

    # Allow SSH
    log_info "Allowing SSH (port ${SSH_PORT})..."
    if ! iptables -C INPUT -p tcp --dport ${SSH_PORT} -j ACCEPT &> /dev/null; then
        iptables -A INPUT -p tcp --dport ${SSH_PORT} -j ACCEPT
    fi

    # Allow HTTP
    log_info "Allowing HTTP (port ${HTTP_PORT})..."
    if ! iptables -C INPUT -p tcp --dport ${HTTP_PORT} -j ACCEPT &> /dev/null; then
        iptables -A INPUT -p tcp --dport ${HTTP_PORT} -j ACCEPT
    fi

    # Allow HTTPS
    log_info "Allowing HTTPS (port ${HTTPS_PORT})..."
    if ! iptables -C INPUT -p tcp --dport ${HTTPS_PORT} -j ACCEPT &> /dev/null; then
        iptables -A INPUT -p tcp --dport ${HTTPS_PORT} -j ACCEPT
    fi

    # Try to save rules
    if command -v iptables-save &> /dev/null; then
        log_info "Saving iptables rules..."
        if [[ -f /etc/debian_version ]]; then
            # Debian/Ubuntu
            iptables-save > /etc/iptables/rules.v4 2>/dev/null || \
            iptables-save > /etc/iptables.rules 2>/dev/null || \
            log_warn "Could not save iptables rules automatically"
        elif [[ -f /etc/redhat-release ]]; then
            # RHEL/CentOS
            service iptables save 2>/dev/null || \
            log_warn "Could not save iptables rules automatically"
        fi
    fi

    log_info "iptables configuration completed"
    log_warn "Please ensure iptables rules persist after reboot"
}

show_status() {
    local firewall_type=$1

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "Firewall Status"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    case $firewall_type in
        ufw)
            ufw status verbose || true
            ;;
        firewalld)
            firewall-cmd --list-all || true
            ;;
        iptables)
            iptables -L -n -v | grep -E "(${SSH_PORT}|${HTTP_PORT}|${HTTPS_PORT})" || \
            echo "Could not display filtered rules"
            ;;
        none)
            log_warn "No firewall detected"
            ;;
    esac

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

print_summary() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "Firewall configuration completed!"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "  Configured ports:"
    echo "    • SSH:   ${SSH_PORT}/tcp  (for git push deployments)"
    echo "    • HTTP:  ${HTTP_PORT}/tcp  (web traffic)"
    echo "    • HTTPS: ${HTTPS_PORT}/tcp (secure web traffic)"
    echo ""
    echo "  Your server is now ready to:"
    echo "    ✓ Accept web traffic on HTTP and HTTPS"
    echo "    ✓ Receive git push deployments via SSH"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

main() {
    log_info "Starting firewall configuration for RunLite..."
    echo ""

    check_root

    # Detect firewall type
    log_step "Detecting firewall type..."
    FIREWALL_TYPE=$(detect_firewall)
    log_info "Detected firewall: ${FIREWALL_TYPE}"
    echo ""

    # Configure based on detected firewall
    case $FIREWALL_TYPE in
        ufw)
            configure_ufw
            ;;
        firewalld)
            configure_firewalld
            ;;
        iptables)
            configure_iptables
            ;;
        none)
            log_error "No supported firewall found"
            log_error "Please install ufw, firewalld, or configure iptables manually"
            exit 1
            ;;
        *)
            log_error "Unknown firewall type: $FIREWALL_TYPE"
            exit 1
            ;;
    esac

    show_status "$FIREWALL_TYPE"
    print_summary
}

main "$@"
