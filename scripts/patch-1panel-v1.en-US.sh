#!/bin/bash
set -euo pipefail

# DomainNest 1Panel v1 httpreq Patch Installer
# Auto-downloads 1Panel v1 LTS, applies patch, builds and installs

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PATCH_FILE="${SCRIPT_DIR}/1panel-v1-httpreq.patch"

REPO_URL="https://github.com/1Panel-dev/1Panel.git"
WORK_DIR="/tmp/1panel-v1-patch-$$"
AGENT_DIR="${WORK_DIR}/agent"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

cleanup() {
  if [[ -d "$WORK_DIR" ]]; then
    rm -rf "$WORK_DIR"
  fi
}
trap cleanup EXIT

# Check prerequisites, offer to install if missing
ensure_command() {
  local cmd="$1"
  local pkg="$2"
  command -v "$cmd" >/dev/null 2>&1 && return 0
  log_error "$cmd is required"
  if [[ ! -t 0 ]]; then
    log_error "Non-interactive mode — cannot prompt. Please install manually: apt-get install -y $pkg"
    exit 1
  fi
  read -p "Install via apt? [y/N] " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    log_info "Installing $pkg..."
    apt-get update && apt-get install -y "$pkg"
  else
    exit 1
  fi
}

ensure_command git git
ensure_command go golang-go
ensure_command patch patch

# Version detection: ensure 1Panel is installed with correct architecture
log_step() { echo -e "${GREEN}[STEP]${NC} $*"; }

# Check if 1Panel is installed at all
log_step "Detecting 1Panel installation..."
if ! systemctl list-unit-files 2>/dev/null | grep -q '1panel'; then
    if [[ ! -f /usr/local/bin/1panel ]] && [[ ! -f /usr/bin/1panel ]] && \
       [[ ! -f /usr/local/bin/1panel-core ]] && [[ ! -f /usr/bin/1panel-core ]]; then
        log_error "1Panel is not installed on this machine"
        exit 1
    fi
fi

# Detect monolithic v1 (pre-split, NOT compatible)
if [[ -f /usr/local/bin/1panel ]] || [[ -f /usr/bin/1panel ]]; then
    if [[ ! -f /usr/local/bin/1panel-core ]] && [[ ! -f /usr/bin/1panel-core ]]; then
        log_error "Detected 1Panel monolithic v1 (pre-split architecture)."
        log_error "This patch requires the split architecture (v1.10.28+)."
        log_error "Please upgrade 1Panel to v1.10.28-lts or later."
        exit 1
    fi
fi

# Detect v2 (not supported, exit)
if [[ -f /usr/local/bin/1panel-core ]] || [[ -f /usr/bin/1panel-core ]]; then
    INSTALLED_VERSION=$(/usr/local/bin/1panel-core version 2>/dev/null || /usr/bin/1panel-core version 2>/dev/null || echo "")
    if [[ "$INSTALLED_VERSION" == *"v2"* ]]; then
        log_error "1Panel v2 detected. This patch only supports v1 LTS."
        log_error "v2 does not require this patch. Please wait for official support."
        exit 1
    fi
fi

# Find latest v1 LTS tag
log_info "Finding latest 1Panel v1 LTS tag..."
LATEST_TAG=$(git ls-remote --tags "$REPO_URL" 'refs/tags/v1.*-lts' 2>/dev/null | \
  awk -F/ '{print $NF}' | sort -V | tail -1)

if [[ -z "$LATEST_TAG" ]]; then
  log_error "Could not find v1 LTS tag"
  exit 1
fi
log_info "Latest v1 LTS tag: $LATEST_TAG"

# Check if patch file exists
if [[ ! -f "$PATCH_FILE" ]]; then
  log_error "Patch file not found: $PATCH_FILE"
  exit 1
fi

# Clone
log_info "Cloning 1Panel $LATEST_TAG..."
git clone --depth 1 --branch "$LATEST_TAG" "$REPO_URL" "$WORK_DIR"

# Verify version compatibility
if ! grep -q 'go-acme/lego/v4' "${AGENT_DIR}/go.mod" 2>/dev/null; then
  log_error "This patch only supports 1Panel v1 with lego v4"
  exit 1
fi

# Check if already patched (idempotency)
if grep -q 'HttpReq' "${AGENT_DIR}/utils/ssl/dns_provider.go"; then
  log_warn "dns_provider.go already contains HttpReq — patch may already be applied"
  read -p "Continue anyway? [y/N] " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 0
  fi
fi

# Apply patch
log_info "Applying patch..."
cd "$AGENT_DIR"
patch -p2 --dry-run < "$PATCH_FILE" >/dev/null 2>&1 || {
  log_error "Patch dry-run failed — the patch may not match this 1Panel version"
  exit 1
}
patch -p2 < "$PATCH_FILE"

# Build
log_info "Building 1panel-agent..."
cd "$AGENT_DIR"
GOARCH=$(uname -m)
case "$GOARCH" in
  x86_64)  GOARCH=amd64 ;;
  aarch64) GOARCH=arm64 ;;
  armv7l)  GOARCH=arm ;;
esac
CGO_ENABLED=0 GOOS=linux GOARCH=$GOARCH go build -trimpath -ldflags '-s -w' -o 1panel-agent cmd/server/main.go

# Detect install path
INSTALL_BIN=""
if [[ -f /usr/local/bin/1panel-agent ]]; then
  INSTALL_BIN="/usr/local/bin/1panel-agent"
elif [[ -f /usr/bin/1panel-agent ]]; then
  INSTALL_BIN="/usr/bin/1panel-agent"
else
  log_warn "Could not find existing 1panel-agent binary"
  log_info "Build output: ${AGENT_DIR}/1panel-agent"
  log_info "Please manually copy it to your 1Panel installation directory"
  exit 0
fi

# Backup and install
BACKUP_PATH="${INSTALL_BIN}.backup.$(date +%Y%m%d%H%M%S)"
log_info "Backing up existing binary to $BACKUP_PATH"
cp "$INSTALL_BIN" "$BACKUP_PATH"

log_info "Installing new binary..."
cp "${AGENT_DIR}/1panel-agent" "$INSTALL_BIN"
chmod +x "$INSTALL_BIN"

# Restart
if systemctl is-active --quiet 1panel 2>/dev/null || systemctl is-active --quiet 1panel-agent 2>/dev/null; then
  log_info "Restarting 1Panel service..."
  systemctl restart 1panel 2>/dev/null || systemctl restart 1panel-agent 2>/dev/null || {
    log_warn "Could not auto-restart service. Please restart manually."
  }
else
  log_warn "1Panel service not detected. Please restart manually."
fi

log_info "Patch installed successfully!"
log_info "Backup saved at: $BACKUP_PATH"
