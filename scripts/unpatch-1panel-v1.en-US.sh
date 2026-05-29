#!/bin/bash
set -euo pipefail

# DomainNest 1Panel v1 httpreq Patch Rollback Script
# Restores the backed-up 1panel-agent binary

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

# Check if 1Panel is installed
log_step "Detecting 1Panel installation..."
if ! systemctl list-unit-files 2>/dev/null | grep -q '1panel'; then
    if [[ ! -f /usr/local/bin/1panel ]] && [[ ! -f /usr/bin/1panel ]] && \
       [[ ! -f /usr/local/bin/1panel-core ]] && [[ ! -f /usr/bin/1panel-core ]]; then
        log_error "1Panel is not installed on this machine"
        exit 1
    fi
fi

# Detect install path
INSTALL_BIN=""
if [[ -f /usr/local/bin/1panel-agent ]]; then
  INSTALL_BIN="/usr/local/bin/1panel-agent"
elif [[ -f /usr/bin/1panel-agent ]]; then
  INSTALL_BIN="/usr/bin/1panel-agent"
else
  log_error "Could not find 1panel-agent binary"
  exit 1
fi

# Find backup files
BACKUP_PATTERN="${INSTALL_BIN}.backup.*"
BACKUP_FILES=($(ls -1 $BACKUP_PATTERN 2>/dev/null | sort -V))

if [[ ${#BACKUP_FILES[@]} -eq 0 ]]; then
  log_error "No backup files found: $BACKUP_PATTERN"
  log_error "The patch may not have been applied, or the backup was removed"
  exit 1
fi

# Use the latest backup
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

# Restore backup
log_info "Restoring backup..."
cp "$LATEST_BACKUP" "$INSTALL_BIN"
chmod +x "$INSTALL_BIN"

# Restart service
if systemctl is-active --quiet 1panel 2>/dev/null || systemctl is-active --quiet 1panel-agent 2>/dev/null; then
  log_info "Restarting 1Panel service..."
  systemctl restart 1panel 2>/dev/null || systemctl restart 1panel-agent 2>/dev/null || {
    log_warn "Could not auto-restart service. Please restart manually."
  }
else
  log_warn "1Panel service not running. Please restart manually."
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
