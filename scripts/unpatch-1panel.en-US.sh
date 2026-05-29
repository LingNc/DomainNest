#!/bin/bash
set -euo pipefail

# DomainNest 1Panel httpreq Patch Rollback Script
# Auto-detects 1Panel architecture, restores the appropriate backup

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

# ============================================================
# Detect 1Panel installation
# ============================================================
log_step "Detecting 1Panel installation..."

# Detect split architecture
CORE_BIN=""
for candidate in /usr/local/bin/1panel-core /usr/bin/1panel-core; do
    if [[ -f "$candidate" ]]; then
        CORE_BIN="$candidate"
        break
    fi
done

if [[ -z "$CORE_BIN" ]]; then
    CORE_BIN=$(command -v 1panel-core 2>/dev/null || echo "")
fi

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

INSTALL_TYPE=""
if [[ -n "$CORE_BIN" ]]; then
    PANEL_VERSION=$("$CORE_BIN" version 2>/dev/null || echo "")
    if [[ "$PANEL_VERSION" == *"v2"* ]]; then
        INSTALL_TYPE="split-v2"
    else
        INSTALL_TYPE="split-v1"
    fi
else
    # Monolithic architecture
    if [[ ! -f /usr/local/bin/1panel ]] && [[ ! -f /usr/bin/1panel ]]; then
        if ! systemctl list-unit-files 2>/dev/null | grep -q '1panel'; then
            log_error "1Panel is not installed"
            exit 1
        fi
    fi

    MONOLITHIC_BIN=""
    for p in /usr/local/bin/1panel /usr/bin/1panel; do
        if [[ -f "$p" ]]; then
            target=$(readlink -f "$p" 2>/dev/null)
            if [[ "$target" != *"1panel-core"* ]]; then
                MONOLITHIC_BIN="$p"
                break
            fi
        fi
    done

    if [[ -z "$MONOLITHIC_BIN" ]]; then
        log_error "Could not find 1Panel binary"
        exit 1
    fi

    PANEL_VERSION=$("$MONOLITHIC_BIN" version 2>/dev/null || echo "")
    if [[ "$PANEL_VERSION" == *"v2"* ]]; then
        INSTALL_TYPE="monolithic-v2"
    else
        INSTALL_TYPE="monolithic-v1"
    fi
fi

log_info "Detected installation type: $INSTALL_TYPE"

# Detect migrated monolithic (was monolithic, now has split binaries)
if [[ "$INSTALL_TYPE" == "split-v1" ]]; then
  for mono_path in /usr/local/bin/1panel /usr/bin/1panel; do
    if ls "${mono_path}.backup."* 2>/dev/null | head -1 >/dev/null; then
      INSTALL_TYPE="monolithic-v1"
      NEEDS_SPLIT_ROLLBACK=1
      log_info "Detected migrated monolithic installation, will rollback to monolithic"
      break
    fi
  done
fi

# ============================================================
# Find backup files
# ============================================================
case "$INSTALL_TYPE" in
  split-v1|split-v2)
    INSTALL_BIN=""
    if [[ -f /usr/local/bin/1panel-agent ]]; then
      INSTALL_BIN="/usr/local/bin/1panel-agent"
    elif [[ -f /usr/bin/1panel-agent ]]; then
      INSTALL_BIN="/usr/bin/1panel-agent"
    fi
    ;;
  monolithic-v1|monolithic-v2)
    INSTALL_BIN=""
    if [[ -f /usr/local/bin/1panel ]]; then
      INSTALL_BIN="/usr/local/bin/1panel"
    elif [[ -f /usr/bin/1panel ]]; then
      INSTALL_BIN="/usr/bin/1panel"
    fi
    ;;
esac

if [[ -z "$INSTALL_BIN" ]]; then
  log_error "Could not find binary for: $INSTALL_TYPE"
  exit 1
fi

BACKUP_PATTERN="${INSTALL_BIN}.backup.*"

BACKUP_FILES=($(ls -1 $BACKUP_PATTERN 2>/dev/null | sort -V || true))

if [[ ${#BACKUP_FILES[@]} -eq 0 ]]; then
  log_error "No backup files found: $BACKUP_PATTERN"
  log_error "The patch may not have been applied, or the backup was removed"
  exit 1
fi

LATEST_BACKUP="${BACKUP_FILES[-1]}"
log_info "Found backup: $LATEST_BACKUP"

# Confirm rollback
if [[ -t 0 ]]; then
  read -r -p "Restore this backup? [y/N] " response
  if [[ ! "$response" =~ ^[yY]$ ]]; then
    log_info "Cancelled"
    exit 0
  fi
else
  log_info "Non-interactive mode, restoring latest backup automatically..."
fi

# Detect split rollback requirement (check 1panel-core existence when restoring monolithic backup)
NEEDS_SPLIT_ROLLBACK=0
if [[ "$INSTALL_TYPE" == "monolithic-v1" ]]; then
  for p in /usr/local/bin/1panel-core /usr/bin/1panel-core; do
    if [[ -f "$p" ]]; then
      NEEDS_SPLIT_ROLLBACK=1
      break
    fi
  done
fi

# Stop service
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
    log_error "Cannot stop $SERVICE_NAME service"
    exit 1
  }
fi

# Restore backup
log_info "Restoring backup..."
cp "$LATEST_BACKUP" "$INSTALL_BIN"
chmod +x "$INSTALL_BIN"

# If split rollback is needed
if [[ "$NEEDS_SPLIT_ROLLBACK" -eq 1 ]]; then
  log_info "Split architecture detected, rolling back..."

  log_info "Stopping 1panel-agent and 1panel-core..."
  systemctl stop 1panel-agent 2>/dev/null || true
  systemctl stop 1panel-core 2>/dev/null || true

  log_info "Disabling 1panel-agent and 1panel-core..."
  systemctl disable 1panel-agent 2>/dev/null || true
  systemctl disable 1panel-core 2>/dev/null || true

  log_info "Removing split architecture binaries..."
  rm -f /usr/local/bin/1panel-agent /usr/bin/1panel-agent
  rm -f /usr/local/bin/1panel-core /usr/bin/1panel-core
  rm -f /usr/local/bin/1panel /usr/bin/1panel
  for f in /usr/bin/1panel-agent /usr/bin/1panel-core /usr/bin/1panel; do
    rm -f "$f" 2>/dev/null || true
  done

  log_info "Removing split architecture service files..."
  rm -f /etc/systemd/system/1panel-agent.service
  rm -f /etc/systemd/system/1panel-core.service
  systemctl daemon-reload

  # Restore original service file
  if [[ -f /etc/systemd/system/1panel.service.backup ]]; then
    cp /etc/systemd/system/1panel.service.backup /etc/systemd/system/1panel.service
    log_info "Restored 1panel.service"
  fi

  log_info "Enabling old monolithic 1panel.service..."
  systemctl enable 1panel 2>/dev/null || true

  log_info "Restoring old monolithic service..."
  if systemctl is-active --quiet 1panel 2>/dev/null; then
    systemctl restart 1panel || {
      log_warn "Cannot start 1panel, check: journalctl -u 1panel"
    }
  fi

  log_info "Split rollback complete"
fi

# Restart service
if [[ $SERVICE_WAS_ACTIVE -eq 1 ]]; then
  log_info "Starting $SERVICE_NAME service..."
  systemctl start "$SERVICE_NAME" || {
    log_warn "Could not auto-start service. Please start manually."
  }
fi

log_info "Rollback successful! Restored from: $LATEST_BACKUP"

# Ask whether to delete the backup
if [[ -t 0 ]]; then
  read -r -p "Delete this backup file? [y/N] " response
  if [[ "$response" =~ ^[yY]$ ]]; then
    rm -f "$LATEST_BACKUP"
    log_info "Backup file deleted"
  else
    log_info "Backup file kept: $LATEST_BACKUP"
  fi
fi