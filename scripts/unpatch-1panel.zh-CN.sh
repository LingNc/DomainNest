#!/bin/bash
set -euo pipefail

# DomainNest 1Panel httpreq 补丁回滚脚本
# 自动检测 1Panel 架构，恢复对应的备份文件

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_step()  { echo -e "${GREEN}[STEP]${NC} $*"; }

# 检查 root 权限
if [[ $EUID -ne 0 ]]; then
    log_error "此脚本需要 root 权限运行"
    log_error "请使用: sudo $0"
    exit 1
fi

# ============================================================
# 检测 1Panel 安装状态
# ============================================================
log_step "正在检测 1Panel 安装状态..."

# 检测分裂架构
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
    # 单体架构
    if [[ ! -f /usr/local/bin/1panel ]] && [[ ! -f /usr/bin/1panel ]]; then
        if ! systemctl list-unit-files 2>/dev/null | grep -q '1panel'; then
            log_error "1Panel 未安装"
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
        log_error "未找到 1Panel 二进制文件"
        exit 1
    fi

    PANEL_VERSION=$("$MONOLITHIC_BIN" version 2>/dev/null || echo "")
    if [[ "$PANEL_VERSION" == *"v2"* ]]; then
        INSTALL_TYPE="monolithic-v2"
    else
        INSTALL_TYPE="monolithic-v1"
    fi
fi

log_info "检测到安装类型: $INSTALL_TYPE"

# ============================================================
# 查找备份文件
# ============================================================
case "$INSTALL_TYPE" in
  split-v1|split-v2)
    INSTALL_BIN=""
    if [[ -f /usr/local/bin/1panel-agent ]]; then
      INSTALL_BIN="/usr/local/bin/1panel-agent"
    elif [[ -f /usr/bin/1panel-agent ]]; then
      INSTALL_BIN="/usr/bin/1panel-agent"
    fi
    BACKUP_PATTERN="${INSTALL_BIN}.backup.*"
    ;;
  monolithic-v1|monolithic-v2)
    INSTALL_BIN=""
    if [[ -f /usr/local/bin/1panel ]]; then
      INSTALL_BIN="/usr/local/bin/1panel"
    elif [[ -f /usr/bin/1panel ]]; then
      INSTALL_BIN="/usr/bin/1panel"
    fi
    BACKUP_PATTERN="${INSTALL_BIN}.backup.*"
    ;;
esac

if [[ -z "$INSTALL_BIN" ]]; then
  log_error "未找到二进制文件: $INSTALL_TYPE"
  exit 1
fi

BACKUP_FILES=($(ls -1 $BACKUP_PATTERN 2>/dev/null | sort -V))

if [[ ${#BACKUP_FILES[@]} -eq 0 ]]; then
  log_error "未找到备份文件: $BACKUP_PATTERN"
  log_error "可能没有打过补丁，或备份已被删除"
  exit 1
fi

LATEST_BACKUP="${BACKUP_FILES[-1]}"
log_info "找到备份文件: $LATEST_BACKUP"

# 确认回滚
if [[ -t 0 ]]; then
  read -r -p "是否恢复此备份? [y/N] " response
  if [[ ! "$response" =~ ^[yY]$ ]]; then
    log_info "已取消"
    exit 0
  fi
else
  log_info "非交互模式，自动恢复最新备份..."
fi

# 恢复备份
log_info "正在恢复备份..."
cp "$LATEST_BACKUP" "$INSTALL_BIN"
chmod +x "$INSTALL_BIN"

# 重启服务
SERVICE_NAME=""
case "$INSTALL_TYPE" in
  split-v1|split-v2) SERVICE_NAME="1panel-agent" ;;
  monolithic-v1|monolithic-v2) SERVICE_NAME="1panel" ;;
esac

if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
  log_info "正在重启 $SERVICE_NAME 服务..."
  systemctl restart "$SERVICE_NAME" || {
    log_warn "无法自动重启服务，请手动重启。"
  }
else
  log_warn "$SERVICE_NAME 服务未运行，请手动重启。"
fi

log_info "回滚成功！已恢复: $LATEST_BACKUP"

# 询问是否删除备份
if [[ -t 0 ]]; then
  read -r -p "是否删除该备份文件? [y/N] " response
  if [[ "$response" =~ ^[yY]$ ]]; then
    rm -f "$LATEST_BACKUP"
    log_info "已删除备份文件"
  else
    log_info "备份文件保留: $LATEST_BACKUP"
  fi
fi