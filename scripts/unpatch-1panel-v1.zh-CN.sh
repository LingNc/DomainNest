#!/bin/bash
set -euo pipefail

# DomainNest 1Panel v1 httpreq 补丁回滚脚本
# 恢复备份的 1panel-agent 二进制文件

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

# 检查 1Panel 是否已安装
log_step "正在检测 1Panel 安装状态..."
if ! systemctl list-unit-files 2>/dev/null | grep -q '1panel'; then
    if [[ ! -f /usr/local/bin/1panel ]] && [[ ! -f /usr/bin/1panel ]] && \
       [[ ! -f /usr/local/bin/1panel-core ]] && [[ ! -f /usr/bin/1panel-core ]]; then
        log_error "1Panel 未安装"
        exit 1
    fi
fi

# 检测安装路径
INSTALL_BIN=""
if [[ -f /usr/local/bin/1panel-agent ]]; then
  INSTALL_BIN="/usr/local/bin/1panel-agent"
elif [[ -f /usr/bin/1panel-agent ]]; then
  INSTALL_BIN="/usr/bin/1panel-agent"
else
  log_error "未找到 1panel-agent 二进制文件"
  exit 1
fi

# 查找备份文件
BACKUP_PATTERN="${INSTALL_BIN}.backup.*"
BACKUP_FILES=($(ls -1 $BACKUP_PATTERN 2>/dev/null | sort -V))

if [[ ${#BACKUP_FILES[@]} -eq 0 ]]; then
  log_error "未找到备份文件: $BACKUP_PATTERN"
  log_error "可能没有打过补丁，或备份已被删除"
  exit 1
fi

# 使用最新的备份
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
if systemctl is-active --quiet 1panel 2>/dev/null || systemctl is-active --quiet 1panel-agent 2>/dev/null; then
  log_info "正在重启 1Panel 服务..."
  systemctl restart 1panel 2>/dev/null || systemctl restart 1panel-agent 2>/dev/null || {
    log_warn "无法自动重启服务，请手动重启。"
  }
else
  log_warn "未检测到运行中的 1Panel 服务，请手动重启。"
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
