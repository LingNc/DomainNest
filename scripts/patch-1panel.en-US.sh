#!/bin/bash
set -euo pipefail

# DomainNest 1Panel httpreq Patch Installer
# Supports v1 LTS (split/monolithic) and v2 (split architecture)
# Auto-detects 1Panel version and architecture, selects appropriate patch

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PATCH_FILE_V1="${SCRIPT_DIR}/1panel-v1-httpreq.patch"
PATCH_FILE_V2="${SCRIPT_DIR}/1panel-v2-httpreq.patch"

REPO_URL="https://github.com/1Panel-dev/1Panel.git"
WORK_DIR="/tmp/1panel-patch-$$"
CLEANUP_WORK_DIR=0
SERVICE_FILES_URL="https://github.com/1Panel-dev/installer/raw/v2/initscript"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_step()  { echo -e "${GREEN}[STEP]${NC} $*"; }

# Check root privileges
if [[ $EUID -ne 0 ]]; then
    log_error "This script requires root privileges"
    log_error "Please run: sudo $0"
    exit 1
fi

cleanup() {
  if [[ $CLEANUP_WORK_DIR -eq 1 && -d "$WORK_DIR" ]]; then
    rm -rf "$WORK_DIR"
  fi
}
trap cleanup EXIT

# Check prerequisites
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
# Ensure Go from /usr/local/go/bin is in PATH (installed by this script previously)
if [[ -x /usr/local/go/bin/go ]] && [[ ":$PATH:" != *":/usr/local/go/bin:"* ]]; then
  export PATH=/usr/local/go/bin:$PATH
fi
ensure_command go golang-go
ensure_command patch patch
ensure_command curl curl
ensure_command timeout coreutils

# ============================================================
# Detect 1Panel installation
# ============================================================
log_step "Detecting 1Panel installation..."

# 1. Detect split architecture (1panel-core exists)
CORE_BIN=""
for candidate in /usr/local/bin/1panel-core /usr/bin/1panel-core; do
    if [[ -f "$candidate" ]]; then
        CORE_BIN="$candidate"
        break
    fi
done

# fallback: search PATH
if [[ -z "$CORE_BIN" ]]; then
    CORE_BIN=$(command -v 1panel-core 2>/dev/null || echo "")
fi

# fallback: resolve symlinks (v1.10.34-lts creates /usr/bin/1panel -> 1panel-core)
if [[ -z "$CORE_BIN" ]]; then
    for p in /usr/local/bin/1panel /usr/bin/1panel; do
        if [[ -f "$p" ]]; then
            target=$(readlink -f "$p" 2>/dev/null)
            if [[ "$target" == *"1panel-core"* ]]; then
                CORE_BIN="$target"
                break
            fi
        fi
    done
fi

INSTALL_TYPE=""  # "split-v1" / "split-v2" / "monolithic-v1" / "monolithic-v2"
PANEL_VERSION=""

if [[ -n "$CORE_BIN" ]]; then
    # Split architecture detected, parse version
    log_info "Detected 1Panel split architecture: $CORE_BIN"
    PANEL_VERSION=$(timeout 5 "$CORE_BIN" version 2>/dev/null || echo "")
    if [[ "$PANEL_VERSION" == *"v2"* ]]; then
        INSTALL_TYPE="split-v2"
        log_info "Detected v2"
    else
        INSTALL_TYPE="split-v1"
        log_info "Detected v1 LTS"
    fi
else
    # 2. Detect monolithic architecture (non-symlink 1panel binary exists)
    MONOLITHIC_BIN=""
    for p in /usr/local/bin/1panel /usr/bin/1panel; do
        if [[ -f "$p" ]]; then
            target=$(readlink -f "$p" 2>/dev/null)
            # If not a symlink pointing to 1panel-core, it's monolithic
            if [[ "$target" != *"1panel-core"* ]]; then
                MONOLITHIC_BIN="$p"
                break
            fi
        fi
    done

    if [[ -z "$MONOLITHIC_BIN" ]]; then
        if ! systemctl list-unit-files 2>/dev/null | grep -q '1panel'; then
            log_error "1Panel is not installed"
            exit 1
        fi
        log_error "Could not find 1Panel binary"
        exit 1
    fi

    log_info "Detected 1Panel monolithic architecture: $MONOLITHIC_BIN"
    PANEL_VERSION=$(timeout 5 "$MONOLITHIC_BIN" version 2>/dev/null || echo "")
    if [[ "$PANEL_VERSION" == *"v2"* ]]; then
        INSTALL_TYPE="monolithic-v2"
        log_info "Detected v2"
    else
        INSTALL_TYPE="monolithic-v1"
        log_info "Detected v1"
    fi
fi

# v2 warning: v2 has official HTTP Request support and updates frequently
if [[ "$INSTALL_TYPE" == *"v2"* ]]; then
    log_warn "Detected 1Panel v2."
    log_warn "DomainNest natively supports Technitium DNS provider — no patch needed for it."
    log_warn "Continue only if you specifically need the HttpReq provider."
    read -r -p "Continue installing patch? [y/N] " response
    if [[ ! "$response" =~ ^[yY]$ ]]; then
        exit 0
    fi
fi

# ============================================================
# Determine patch file and clone target
# ============================================================
case "$INSTALL_TYPE" in
  split-v1|monolithic-v1)
    PATCH_FILE="$PATCH_FILE_V1"
    TAG_PATTERN='v1.*-lts'
    log_info "Using v1 LTS patch"
    ;;
  split-v2|monolithic-v2)
    PATCH_FILE="$PATCH_FILE_V2"
    TAG_PATTERN='v2.*'
    log_info "Using v2 patch"
    ;;
esac

# Check if patch file exists
if [[ ! -f "$PATCH_FILE" ]]; then
  log_error "Patch file not found: $PATCH_FILE"
  exit 1
fi

# Find latest tag
log_info "Finding latest 1Panel tag (${TAG_PATTERN})..."
LATEST_TAG=$(git ls-remote --tags "$REPO_URL" "refs/tags/${TAG_PATTERN}" 2>/dev/null | \
  awk -F/ '{print $NF}' | sort -V | tail -1)

if [[ -z "$LATEST_TAG" ]]; then
  log_error "Could not find tag matching pattern: ${TAG_PATTERN}"
  exit 1
fi
log_info "Latest tag: $LATEST_TAG"

# Update WORK_DIR to use tag-based naming
WORK_DIR="/tmp/1panel-patch-${LATEST_TAG}"

# Clean up old PID-based directories
rm -rf /tmp/1panel-patch-[0-9]* 2>/dev/null || true

# Clone or reuse source
if [[ -d "$WORK_DIR" ]] && git -C "$WORK_DIR" describe --tags 2>/dev/null | grep -q "$LATEST_TAG"; then
    log_info "Reusing existing source directory: $WORK_DIR"
    CLEANUP_WORK_DIR=0
else
    rm -rf "$WORK_DIR"
    log_info "Cloning 1Panel $LATEST_TAG..."
    git clone --depth 1 --branch "$LATEST_TAG" "$REPO_URL" "$WORK_DIR"
    CLEANUP_WORK_DIR=0
fi

# ============================================================
# Determine source directory structure (split vs monolithic)
# ============================================================
SRC_SUBDIR=""  # "agent" or "backend"
if [[ -d "${WORK_DIR}/agent" ]]; then
  SRC_SUBDIR="agent"
elif [[ -d "${WORK_DIR}/backend" ]]; then
  SRC_SUBDIR="backend"
else
  log_error "Could not find agent/ or backend/ directory"
  exit 1
fi
SRC_DIR="${WORK_DIR}/${SRC_SUBDIR}"
log_info "Source directory: $SRC_DIR"

HAS_SPLIT_SOURCE=0
if [[ -d "${WORK_DIR}/agent" && -d "${WORK_DIR}/core" ]]; then
  HAS_SPLIT_SOURCE=1
fi

# Verify lego version compatibility
if ! grep -q 'go-acme/lego/v4' "${SRC_DIR}/go.mod" 2>/dev/null; then
  log_error "This patch only supports 1Panel with lego v4"
  exit 1
fi

# Check Go version compatibility
GO_MOD_VERSION=$(grep '^go ' "${SRC_DIR}/go.mod" | awk '{print $2}')
if [[ -n "$GO_MOD_VERSION" ]]; then
  GO_MOD_MAJOR_MINOR=$(echo "$GO_MOD_VERSION" | cut -d. -f1,2)
  GO_INSTALLED_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
  GO_INSTALLED_MAJOR_MINOR=$(echo "$GO_INSTALLED_VERSION" | cut -d. -f1,2)
  if [[ "$(printf '%s\n' "$GO_MOD_MAJOR_MINOR" "$GO_INSTALLED_MAJOR_MINOR" | sort -V | head -1)" != "$GO_MOD_MAJOR_MINOR" ]]; then
    log_error "Go version too old: installed $GO_INSTALLED_VERSION, requires $GO_MOD_VERSION+"
    if [[ ! -t 0 ]]; then
      log_error "Non-interactive mode, please upgrade Go manually: https://go.dev/dl/"
      exit 1
    fi
    read -p "Auto-download and install Go $GO_MOD_VERSION? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      GOARCH=$(uname -m)
      case "$GOARCH" in
        x86_64)  GOARCH=amd64 ;;
        aarch64) GOARCH=arm64 ;;
        armv7l)  GOARCH=arm ;;
      esac
      GO_TARBALL="go${GO_MOD_VERSION}.linux-${GOARCH}.tar.gz"
      GO_DOWNLOAD_URL="https://go.dev/dl/${GO_TARBALL}"
      log_info "Downloading Go $GO_MOD_VERSION..."
      curl -fSL# "$GO_DOWNLOAD_URL" -o "/tmp/${GO_TARBALL}" || {
        log_error "Download failed, please upgrade Go manually: https://go.dev/dl/"
        exit 1
      }
      log_info "Installing Go $GO_MOD_VERSION..."
      rm -rf /usr/local/go
      tar -C /usr/local -xzf "/tmp/${GO_TARBALL}"
      rm -f "/tmp/${GO_TARBALL}"
      export PATH=/usr/local/go/bin:$PATH
      # Persist PATH for future sessions (use real user's home when run via sudo)
      REAL_HOME=$(eval echo "~${SUDO_USER:-$HOME}")
      GO_PATH_LINE='export PATH=/usr/local/go/bin:$PATH'
      SHELL_RC=""
      if [[ -f "$REAL_HOME/.bashrc" ]]; then
        SHELL_RC="$REAL_HOME/.bashrc"
      elif [[ -f "$REAL_HOME/.bash_profile" ]]; then
        SHELL_RC="$REAL_HOME/.bash_profile"
      elif [[ -f "$REAL_HOME/.profile" ]]; then
        SHELL_RC="$REAL_HOME/.profile"
      fi
      if [[ -n "$SHELL_RC" ]] && ! grep -q '/usr/local/go/bin' "$SHELL_RC" 2>/dev/null; then
        echo "$GO_PATH_LINE" >> "$SHELL_RC"
        log_info "Added Go PATH to $SHELL_RC"
      fi
      log_info "Go upgraded: $(go version)"
    else
      log_error "Please upgrade Go manually and retry"
      exit 1
    fi
  fi
fi

PATCH_ALREADY_APPLIED=false
if grep -q 'HttpReq' "${SRC_DIR}/utils/ssl/dns_provider.go" 2>/dev/null; then
  log_info "dns_provider.go already contains HttpReq — patch already applied, skipping patch step"
  read -p "Proceed directly to build? [Y/n] " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Nn]$ ]]; then
    exit 0
  fi
  PATCH_ALREADY_APPLIED=true
fi

if [[ "$PATCH_ALREADY_APPLIED" == "true" ]]; then
  log_info "Patch already applied, skipping"
else
  # ============================================================
  # Apply patch
  # ============================================================
  # For monolithic source, adjust patch path: a/agent -> a/backend
  if [[ "$SRC_SUBDIR" == "backend" ]]; then
    log_info "Monolithic source detected, adjusting patch paths (agent -> backend)..."
    PATCH_CONTENT=$(cat "$PATCH_FILE")
    PATCH_CONTENT=$(echo "$PATCH_CONTENT" | sed 's|^a/agent/|a/backend/|' | sed 's|^b/agent/|b/backend/|')
    TEMP_PATCH_FILE="${WORK_DIR}/adjusted.patch"
    echo "$PATCH_CONTENT" > "$TEMP_PATCH_FILE"
    APPLY_PATCH="$TEMP_PATCH_FILE"
  else
    APPLY_PATCH="$PATCH_FILE"
  fi

  log_info "Applying patch..."
  cd "$SRC_DIR"
  patch -p2 --dry-run < "$APPLY_PATCH" >/dev/null 2>&1 || {
    log_error "Patch dry-run failed — the patch may not match this 1Panel version"
    log_error "Patch file: $APPLY_PATCH"
    log_error "Source directory: $SRC_DIR"
    exit 1
  }
  patch -p2 < "$APPLY_PATCH"
fi

# ============================================================
# Build
# ============================================================
GOARCH=$(uname -m)
case "$GOARCH" in
  x86_64)  GOARCH=amd64 ;;
  aarch64) GOARCH=arm64 ;;
  armv7l)  GOARCH=arm ;;
esac

BUILD_LOG=$(mktemp)
trap 'rm -f "$BUILD_LOG"; cleanup' EXIT

build_binary() {
  local src_dir="$1"
  local build_target="$2"
  local output_name="$3"
  local label="$4"

  log_info "Building ${label}..."
  local build_start=$SECONDS
  (
    cd "$src_dir"
    CGO_ENABLED=0 GOOS=linux GOARCH=$GOARCH \
      go build -trimpath -ldflags '-s -w' -o "$output_name" "$build_target"
  ) >"$BUILD_LOG" 2>&1 &
  local build_pid=$!

  sleep 0.5
  echo -e "${GREEN}[INFO]${NC} Downloading dependencies and building ${label} (may take a few minutes on first run)..."
  local last_printed=""
  while kill -0 "$build_pid" 2>/dev/null; do
    local elapsed=$(( SECONDS - build_start ))
    local last_line
    last_line=$(tail -c 500 "$BUILD_LOG" 2>/dev/null | grep -v '^$' | tail -1 || true)
    if [[ -n "$last_line" && "$last_line" != "$last_printed" ]]; then
      last_printed="$last_line"
      echo -ne "  ${last_line} (${elapsed}s)\033[K\r"
    fi
    sleep 2
  done
  echo -e "\033[K"

  wait "$build_pid"
  local rc=$?
  elapsed=$(( SECONDS - build_start ))
  if [[ $rc -ne 0 ]]; then
    log_error "${label} build failed (${elapsed}s elapsed), log output:"
    cat "$BUILD_LOG"
    rm -f "$BUILD_LOG"
    exit 1
  fi
  log_info "${label} build completed (${elapsed}s elapsed)"
  : > "$BUILD_LOG"
}

if [[ "$INSTALL_TYPE" == "monolithic-v1" && "$HAS_SPLIT_SOURCE" -eq 1 ]]; then
  build_binary "$SRC_DIR" "cmd/server/main.go" "1panel-agent" "1panel-agent (with HttpReq patch)"
  AGENT_BUILD_OUTPUT="${SRC_DIR}/1panel-agent"
  if [[ ! -f "$AGENT_BUILD_OUTPUT" ]]; then
    log_error "Agent binary not found after build: $AGENT_BUILD_OUTPUT"
    exit 1
  fi

  CORE_SRC_DIR="${WORK_DIR}/core"
  build_binary "$CORE_SRC_DIR" "cmd/server/main.go" "1panel-core" "1panel-core (web UI)"
  CORE_BUILD_OUTPUT="${CORE_SRC_DIR}/1panel-core"
  if [[ ! -f "$CORE_BUILD_OUTPUT" ]]; then
    log_error "Core binary not found after build: $CORE_BUILD_OUTPUT"
    exit 1
  fi
else
  BUILD_TARGET="cmd/server/main.go"
  if [[ "$SRC_SUBDIR" == "backend" ]]; then
    BUILD_TARGET="cmd/1panel/main.go"
  fi
  log_info "Build target: $BUILD_TARGET"
  build_binary "$SRC_DIR" "$BUILD_TARGET" "1panel-agent" "1panel-agent"
fi

# ============================================================
# Install
# ============================================================

if [[ "$INSTALL_TYPE" == "monolithic-v1" && "$HAS_SPLIT_SOURCE" -eq 1 ]]; then
  MONO_BIN=""
  for p in /usr/local/bin/1panel /usr/bin/1panel; do
    if [[ -f "$p" ]]; then
      target=$(readlink -f "$p" 2>/dev/null)
      if [[ "$target" != *"1panel-core"* ]]; then
        MONO_BIN="$p"
        break
      fi
    fi
  done

  if [[ -z "$MONO_BIN" ]]; then
    log_error "Monolithic 1panel binary not found"
    exit 1
  fi

  NEW_BIN_DIR="/usr/local/bin"

  log_info "Stopping 1panel service..."
  if systemctl is-active --quiet 1panel 2>/dev/null; then
    systemctl stop 1panel || {
      log_error "Cannot stop 1panel service"
      exit 1
    }
  fi

  # Backup original service file
  if [[ -f /etc/systemd/system/1panel.service ]]; then
    cp /etc/systemd/system/1panel.service /etc/systemd/system/1panel.service.backup
    log_info "Backed up 1panel.service to /etc/systemd/system/1panel.service.backup"
  fi

  BACKUP_PATH="${MONO_BIN}.backup.$(date +%Y%m%d%H%M%S)"
  log_info "Backing up monolithic binary to $BACKUP_PATH"
  cp "$MONO_BIN" "$BACKUP_PATH"

  log_info "Installing 1panel-agent to ${NEW_BIN_DIR}/1panel-agent"
  cp "$AGENT_BUILD_OUTPUT" "${NEW_BIN_DIR}/1panel-agent"
  chmod +x "${NEW_BIN_DIR}/1panel-agent"

  log_info "Installing 1panel-core to ${NEW_BIN_DIR}/1panel-core"
  cp "$CORE_BUILD_OUTPUT" "${NEW_BIN_DIR}/1panel-core"
  chmod +x "${NEW_BIN_DIR}/1panel-core"

  for bin_name in 1panel-core 1panel-agent; do
    if [[ "${NEW_BIN_DIR}" != "/usr/bin" ]]; then
      ln -sf "${NEW_BIN_DIR}/${bin_name}" "/usr/bin/${bin_name}"
    fi
  done

  if [[ "$MONO_BIN" == "/usr/bin/1panel" ]]; then
    ln -sf "${NEW_BIN_DIR}/1panel-core" "/usr/bin/1panel"
  fi

  # Replace monolithic binary with symlink
  if [[ -f "${NEW_BIN_DIR}/1panel" && ! -L "${NEW_BIN_DIR}/1panel" ]]; then
    rm -f "${NEW_BIN_DIR}/1panel"
    ln -sf "${NEW_BIN_DIR}/1panel-core" "${NEW_BIN_DIR}/1panel"
    log_info "Replaced ${NEW_BIN_DIR}/1panel with symlink to 1panel-core"
  fi

  log_info "Downloading systemd service files..."
  SVC_DOWNLOAD_OK=1
  curl -fSL# "${SERVICE_FILES_URL}/1panel-core.service" \
    -o /etc/systemd/system/1panel-core.service 2>/dev/null || SVC_DOWNLOAD_OK=0
  curl -fSL# "${SERVICE_FILES_URL}/1panel-agent.service" \
    -o /etc/systemd/system/1panel-agent.service 2>/dev/null || SVC_DOWNLOAD_OK=0

  if [[ "$SVC_DOWNLOAD_OK" -eq 0 ]]; then
    log_error "Failed to download service files from GitHub."
    log_error "URL:"
    log_error "  ${SERVICE_FILES_URL}/1panel-core.service"
    log_error "  ${SERVICE_FILES_URL}/1panel-agent.service"
    log_error ""
    log_error "Binaries installed but services not configured."
    log_error "Please manually download service files to /etc/systemd/system/"
    log_error "Then run: systemctl daemon-reload && systemctl enable --now 1panel-core 1panel-agent"
    CLEANUP_WORK_DIR=1
    rm -f "$BUILD_LOG"
    exit 1
  fi

  systemctl daemon-reload

  log_info "Disabling old 1panel.service..."
  systemctl disable 1panel 2>/dev/null || true

  log_info "Enabling 1panel-core.service and 1panel-agent.service..."
  systemctl enable 1panel-core 1panel-agent

  log_info "Starting 1panel-core and 1panel-agent..."
  systemctl start 1panel-core || {
    log_warn "Cannot start 1panel-core, check: journalctl -u 1panel-core"
  }
  systemctl start 1panel-agent || {
    log_warn "Cannot start 1panel-agent, check: journalctl -u 1panel-agent"
  }

  log_info "Migration complete: monolithic -> split architecture"
  log_info "Old monolithic binary backup: $BACKUP_PATH"

else
  INSTALL_BIN=""
  case "$INSTALL_TYPE" in
    split-v1|split-v2)
      if [[ -f /usr/local/bin/1panel-agent ]]; then
        INSTALL_BIN="/usr/local/bin/1panel-agent"
      elif [[ -f /usr/bin/1panel-agent ]]; then
        INSTALL_BIN="/usr/bin/1panel-agent"
      fi
      ;;
    monolithic-v1|monolithic-v2)
      if [[ -f /usr/local/bin/1panel ]]; then
        INSTALL_BIN="/usr/local/bin/1panel"
      elif [[ -f /usr/bin/1panel ]]; then
        INSTALL_BIN="/usr/bin/1panel"
      fi
      ;;
  esac

  if [[ -z "$INSTALL_BIN" ]]; then
    log_warn "Could not find installed binary"
    log_info "Build output: ${SRC_DIR}/1panel-agent"
    log_info "Please manually copy it to your 1Panel installation directory"
    exit 0
  fi

  SERVICE_NAME=""
  case "$INSTALL_TYPE" in
    split-v1|split-v2) SERVICE_NAME="1panel-agent" ;;
    monolithic-v1|monolithic-v2) SERVICE_NAME="1panel" ;;
  esac

  SERVICE_WAS_ACTIVE=0
  if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    SERVICE_WAS_ACTIVE=1
    log_info "Stopping $SERVICE_NAME service..."
    systemctl stop "$SERVICE_NAME" || {
      log_error "Failed to stop $SERVICE_NAME service"
      exit 1
    }
  fi

  BACKUP_PATH="${INSTALL_BIN}.backup.$(date +%Y%m%d%H%M%S)"
  log_info "Backing up existing binary to $BACKUP_PATH"
  cp "$INSTALL_BIN" "$BACKUP_PATH"

  log_info "Installing new binary..."
  cp "${SRC_DIR}/1panel-agent" "$INSTALL_BIN"
  chmod +x "$INSTALL_BIN"

  if [[ $SERVICE_WAS_ACTIVE -eq 1 ]] || systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    log_info "Starting $SERVICE_NAME service..."
    systemctl start "$SERVICE_NAME" || {
      log_warn "Could not auto-start service. Please start manually."
    }
  else
    log_warn "$SERVICE_NAME service not running. Please start manually."
  fi
fi

CLEANUP_WORK_DIR=1
log_info "Patch installed successfully!"
log_info "Backup saved at: $BACKUP_PATH"