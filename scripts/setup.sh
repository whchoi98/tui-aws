#!/usr/bin/env bash
#
# tui-aws setup & start script
# Checks prerequisites, installs missing packages, builds and runs tui-aws.
# Supports: macOS (arm64/amd64), Linux (arm64/amd64)
#
set -o pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Minimum versions
MIN_GO_VERSION="1.21"
MIN_AWS_CLI_VERSION="2"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)       echo -e "${RED}Unsupported architecture: $ARCH${NC}"; exit 1 ;;
esac

echo -e "${CYAN}╔══════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║         tui-aws Setup & Launcher         ║${NC}"
echo -e "${CYAN}║   OS: ${GREEN}${OS}/${ARCH}$(printf '%*s' $((23 - ${#OS} - ${#ARCH})) '')${CYAN}║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════╝${NC}"
echo

# ─────────────────────────────────────────────
# Helper functions
# ─────────────────────────────────────────────

check_ok()   { echo -e "  ${GREEN}✓${NC} $1"; }
check_fail() { echo -e "  ${RED}✗${NC} $1"; }
check_warn() { echo -e "  ${YELLOW}!${NC} $1"; }
step()       { echo -e "\n${BLUE}[$1]${NC} $2"; }

confirm() {
    local msg="$1"
    read -r -p "$(echo -e "${YELLOW}  ? ${msg} [Y/n]: ${NC}")" response
    case "$response" in
        [nN][oO]|[nN]) return 1 ;;
        *) return 0 ;;
    esac
}

version_gte() {
    # Returns 0 if $1 >= $2 (semantic version comparison, macOS compatible)
    local IFS=.
    local i v1=($1) v2=($2)
    for ((i=0; i<${#v2[@]}; i++)); do
        [[ -z ${v1[i]+x} ]] && v1[i]=0
        if ((v1[i] > v2[i])); then return 0; fi
        if ((v1[i] < v2[i])); then return 1; fi
    done
    return 0
}

# ─────────────────────────────────────────────
# 1. Check AWS CLI
# ─────────────────────────────────────────────

step "1/5" "Checking AWS CLI..."

install_aws_cli() {
    echo -e "  ${YELLOW}Installing AWS CLI v2...${NC}"
    local tmpdir
    tmpdir=$(mktemp -d)
    trap "rm -rf $tmpdir" RETURN

    if [[ "$OS" == "darwin" ]]; then
        curl -fsSL "https://awscli.amazonaws.com/AWSCLIV2.pkg" -o "$tmpdir/AWSCLIV2.pkg"
        echo -e "  ${YELLOW}Requires sudo password for installation...${NC}"
        sudo installer -pkg "$tmpdir/AWSCLIV2.pkg" -target /
        echo -e "  ${GREEN}AWS CLI v2 installation complete${NC}"
    else
        curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-${ARCH}.zip" -o "$tmpdir/awscli.zip"
        cd "$tmpdir"
        unzip -q awscli.zip
        echo -e "  ${YELLOW}Requires sudo for installation${NC}"
        sudo ./aws/install --update
        cd - > /dev/null
    fi
}

if command -v aws &>/dev/null; then
    AWS_VERSION=$(aws --version 2>&1 | sed -n 's/.*aws-cli\/\([0-9]*\).*/\1/p')
    AWS_VERSION=${AWS_VERSION:-0}
    if [[ "$AWS_VERSION" -ge "$MIN_AWS_CLI_VERSION" ]]; then
        check_ok "AWS CLI v2 ($(aws --version 2>&1 | head -1))"
    else
        check_warn "AWS CLI v1 detected — v2 required"
        if confirm "Install AWS CLI v2?"; then
            install_aws_cli
            check_ok "AWS CLI v2 installed"
        else
            echo -e "  ${RED}AWS CLI v2 is required. Exiting.${NC}"
            exit 1
        fi
    fi
else
    check_fail "AWS CLI not found"
    if confirm "Install AWS CLI v2?"; then
        install_aws_cli
        check_ok "AWS CLI v2 installed"
    else
        echo -e "  ${RED}AWS CLI is required. Exiting.${NC}"
        exit 1
    fi
fi

# ─────────────────────────────────────────────
# 2. Check Session Manager Plugin
# ─────────────────────────────────────────────

step "2/5" "Checking Session Manager Plugin..."

install_ssm_plugin() {
    echo -e "  ${YELLOW}Installing Session Manager Plugin...${NC}"
    local tmpdir
    tmpdir=$(mktemp -d)
    trap "rm -rf $tmpdir" RETURN

    if [[ "$OS" == "darwin" ]]; then
        curl -fsSL "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/mac_${ARCH}/sessionmanager-bundle.zip" \
            -o "$tmpdir/ssm.zip"
        cd "$tmpdir"
        unzip -q ssm.zip
        echo -e "  ${YELLOW}Requires sudo for installation${NC}"
        sudo ./sessionmanager-bundle/install -i /usr/local/sessionmanagerplugin -b /usr/local/bin/session-manager-plugin
        cd - > /dev/null
    else
        if command -v dpkg &>/dev/null; then
            # Debian/Ubuntu
            local deb_arch="$ARCH"
            [[ "$ARCH" == "amd64" ]] && deb_arch="64bit"
            [[ "$ARCH" == "arm64" ]] && deb_arch="arm64"
            curl -fsSL "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/ubuntu_${deb_arch}/session-manager-plugin.deb" \
                -o "$tmpdir/ssm.deb"
            echo -e "  ${YELLOW}Requires sudo for installation${NC}"
            sudo dpkg -i "$tmpdir/ssm.deb"
        elif command -v rpm &>/dev/null; then
            # RHEL/Amazon Linux/CentOS
            local rpm_arch="$ARCH"
            [[ "$ARCH" == "amd64" ]] && rpm_arch="64bit"
            [[ "$ARCH" == "arm64" ]] && rpm_arch="arm64"
            curl -fsSL "https://s3.amazonaws.com/session-manager-downloads/plugin/latest/linux_${rpm_arch}/session-manager-plugin.rpm" \
                -o "$tmpdir/ssm.rpm"
            echo -e "  ${YELLOW}Requires sudo for installation${NC}"
            sudo yum install -y "$tmpdir/ssm.rpm"
        else
            echo -e "  ${RED}Cannot determine package manager (dpkg/rpm). Install manually:${NC}"
            echo -e "  ${RED}https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html${NC}"
            exit 1
        fi
    fi
}

if command -v session-manager-plugin &>/dev/null; then
    check_ok "Session Manager Plugin installed"
else
    check_fail "Session Manager Plugin not found"
    if confirm "Install Session Manager Plugin?"; then
        install_ssm_plugin
        check_ok "Session Manager Plugin installed"
    else
        echo -e "  ${RED}Session Manager Plugin is required for SSM connections. Exiting.${NC}"
        exit 1
    fi
fi

# ─────────────────────────────────────────────
# 3. Check Go
# ─────────────────────────────────────────────

step "3/5" "Checking Go..."

GO_CMD=""
find_go() {
    # Check PATH first
    local go_in_path=""
    go_in_path="$(command -v go 2>/dev/null || true)"
    if [[ -n "$go_in_path" && -x "$go_in_path" ]]; then
        GO_CMD="$go_in_path"
        return 0
    fi

    # Check standard locations including Homebrew paths
    local candidates=(
        "/usr/local/go/bin/go"
        "/opt/homebrew/bin/go"
        "/opt/homebrew/opt/go/bin/go"
        "$HOME/go-install/go/bin/go"
        "$HOME/.local/go/bin/go"
        "/usr/lib/go/bin/go"
        "/snap/bin/go"
    )
    for p in "${candidates[@]}"; do
        if [[ -x "$p" ]]; then
            GO_CMD="$p"
            return 0
        fi
    done
    return 1
}

install_go() {
    if [[ "$OS" == "darwin" ]]; then
        # macOS: prefer Homebrew (avoids URL blocking by security software)
        if command -v brew &>/dev/null; then
            echo -e "  ${YELLOW}Installing Go via Homebrew...${NC}"
            brew install go
            GO_CMD="$(brew --prefix)/bin/go"
        else
            echo -e "  ${YELLOW}Installing Go via official installer...${NC}"
            local go_version="1.23.5"
            local url="https://dl.google.com/go/go${go_version}.darwin-${ARCH}.pkg"
            local tmpdir
            tmpdir=$(mktemp -d)
            trap "rm -rf $tmpdir" RETURN
            curl -fsSL "$url" -o "$tmpdir/go.pkg"
            echo -e "  ${YELLOW}Requires sudo password...${NC}"
            sudo installer -pkg "$tmpdir/go.pkg" -target /
            GO_CMD="/usr/local/go/bin/go"
        fi
    else
        # Linux: download tarball from Google's CDN (more reliable than go.dev)
        local go_version="1.23.5"
        echo -e "  ${YELLOW}Installing Go ${go_version}...${NC}"
        local tmpdir
        tmpdir=$(mktemp -d)
        trap "rm -rf $tmpdir" RETURN

        local url="https://dl.google.com/go/go${go_version}.linux-${ARCH}.tar.gz"
        curl -fsSL "$url" -o "$tmpdir/go.tar.gz"

        local install_dir="$HOME/.local"
        mkdir -p "$install_dir"
        rm -rf "$install_dir/go"
        tar -C "$install_dir" -xzf "$tmpdir/go.tar.gz"

        GO_CMD="$install_dir/go/bin/go"
    fi

    # Add to PATH for current session
    export PATH="$(dirname "$GO_CMD"):$HOME/go/bin:$PATH"
    export GOPATH="$HOME/go"

    # Suggest adding to shell profile
    echo -e "  ${YELLOW}Add to your shell profile (~/.bashrc or ~/.zshrc):${NC}"
    echo -e "  ${CYAN}export PATH=\"$(dirname "$GO_CMD"):\$HOME/go/bin:\$PATH\"${NC}"
}

if find_go; then
    GO_VERSION=$("$GO_CMD" version 2>&1 | head -1 | sed -n 's/.*go\([0-9]*\.[0-9]*\).*/\1/p' | tr -d '\n')
    GO_VERSION=${GO_VERSION:-0.0}
    if version_gte "$GO_VERSION" "$MIN_GO_VERSION"; then
        check_ok "Go $GO_VERSION ($GO_CMD)"
    else
        check_warn "Go $GO_VERSION found but >= $MIN_GO_VERSION required"
        if confirm "Install Go $MIN_GO_VERSION+?"; then
            install_go
            check_ok "Go installed ($GO_CMD)"
        else
            echo -e "  ${RED}Go >= $MIN_GO_VERSION is required to build. Exiting.${NC}"
            exit 1
        fi
    fi
else
    check_fail "Go not found"
    if confirm "Install Go? (installs to ~/.local/go/)"; then
        install_go
        check_ok "Go installed ($GO_CMD)"
    else
        echo -e "  ${RED}Go is required to build tui-aws. Exiting.${NC}"
        exit 1
    fi
fi

# Ensure Go paths are in current session
export PATH="$(dirname "$GO_CMD"):$HOME/go/bin:$PATH"

# ─────────────────────────────────────────────
# 4. Check AWS Credentials
# ─────────────────────────────────────────────

step "4/5" "Checking AWS credentials..."

has_credentials=false

# Check instance metadata (EC2 instance role)
if curl -s --connect-timeout 1 --max-time 2 http://169.254.169.254/latest/meta-data/iam/security-credentials/ &>/dev/null; then
    check_ok "EC2 Instance Role detected"
    has_credentials=true
fi

# Check environment variables
if [[ -n "${AWS_ACCESS_KEY_ID:-}" && -n "${AWS_SECRET_ACCESS_KEY:-}" ]]; then
    check_ok "AWS credentials in environment variables"
    has_credentials=true
fi

# Check credentials file
if [[ -f "$HOME/.aws/credentials" ]]; then
    profile_count=$(grep -c '^\[' "$HOME/.aws/credentials" 2>/dev/null || echo "0")
    check_ok "~/.aws/credentials ($profile_count profiles)"
    has_credentials=true
fi

# Check config file
if [[ -f "$HOME/.aws/config" ]]; then
    config_profiles=$(grep -c '^\[' "$HOME/.aws/config" 2>/dev/null || echo "0")
    check_ok "~/.aws/config ($config_profiles profiles)"
fi

if [[ "$has_credentials" == "false" ]]; then
    check_warn "No AWS credentials found"
    echo -e "  ${YELLOW}Configure credentials with one of:${NC}"
    echo -e "  ${CYAN}  aws configure${NC}"
    echo -e "  ${CYAN}  export AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=...${NC}"
    echo -e "  ${CYAN}  (or run on an EC2 instance with IAM Instance Profile)${NC}"
    echo
    if ! confirm "Continue without credentials? (tui-aws will show errors on load)"; then
        exit 1
    fi
fi

# ─────────────────────────────────────────────
# 5. Build tui-aws
# ─────────────────────────────────────────────

step "5/5" "Building tui-aws..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

# Download dependencies
echo -e "  ${YELLOW}Downloading dependencies...${NC}"
"$GO_CMD" mod download 2>&1 | tail -5 || true

# Build
BINARY="$PROJECT_DIR/tui-aws"
"$GO_CMD" build -ldflags "-X main.version=$(grep '^VERSION' Makefile 2>/dev/null | cut -d= -f2 | tr -d ' ' || echo 'dev')" \
    -o "$BINARY" ./main.go

if [[ -x "$BINARY" ]]; then
    check_ok "Built: $BINARY"
    VERSION=$("$BINARY" --version 2>&1)
    check_ok "Version: $VERSION"
else
    check_fail "Build failed"
    exit 1
fi

# ─────────────────────────────────────────────
# Optional: Install to PATH
# ─────────────────────────────────────────────

echo
if confirm "Install tui-aws to /usr/local/bin/ (requires sudo)?"; then
    sudo cp "$BINARY" /usr/local/bin/tui-aws
    check_ok "Installed to /usr/local/bin/tui-aws"
    echo -e "  ${GREEN}Run from anywhere: ${CYAN}tui-aws${NC}"
elif confirm "Install tui-aws to ~/bin/ (no sudo)?"; then
    mkdir -p "$HOME/bin"
    cp "$BINARY" "$HOME/bin/tui-aws"
    check_ok "Installed to ~/bin/tui-aws"
    if [[ ":$PATH:" != *":$HOME/bin:"* ]]; then
        echo -e "  ${YELLOW}Add to PATH: export PATH=\"\$HOME/bin:\$PATH\"${NC}"
    fi
fi

# ─────────────────────────────────────────────
# Summary
# ─────────────────────────────────────────────

echo
echo -e "${CYAN}╔══════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║            Setup Complete!                ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════╝${NC}"
echo
echo -e "  ${GREEN}Start tui-aws:${NC}"
echo -e "    ${CYAN}$BINARY${NC}"
echo
echo -e "  ${GREEN}Key bindings:${NC}"
echo -e "    ${CYAN}1-6${NC}   Switch tabs (EC2/VPC/Subnets/Routes/SG/Check)"
echo -e "    ${CYAN}p${NC}     Select AWS profile"
echo -e "    ${CYAN}r${NC}     Select region"
echo -e "    ${CYAN}Enter${NC} Action menu"
echo -e "    ${CYAN}/${NC}     Search"
echo -e "    ${CYAN}q${NC}     Quit"
echo
