#!/bin/bash
set -euo pipefail

# DomainNest 1Panel v1 httpreq 补丁安装脚本
# 自动下载 1Panel v1 LTS，应用补丁，编译并安装

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PATCH_FILE="${SCRIPT_DIR}/1panel-v1-httpreq.patch"

REPO_URL="https://github.com/1Panel-dev/1Panel.git"
WORK_DIR="/tmp/1panel-v1-patch-$$"
AGENT_DIR="${WORK_DIR}/agent"

# 颜色
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

# 检查依赖
command -v git >/dev/null 2>&1 || { log_error "需要安装 git"; exit 1; }
command -v go >/dev/null 2>&1 || { log_error "需要安装 Go"; exit 1; }
command -v patch >/dev/null 2>&1 || { log_error "需要安装 patch 命令"; exit 1; }

# 查找最新的 v1 LTS 标签
log_info "正在查找最新的 1Panel v1 LTS 标签..."
LATEST_TAG=$(git ls-remote --tags "$REPO_URL" 'refs/tags/v1.*-lts' 2>/dev/null | \
  awk -F/ '{print $NF}' | sort -V | tail -1)

if [[ -z "$LATEST_TAG" ]]; then
  log_error "未找到 v1 LTS 标签"
  exit 1
fi
log_info "最新的 v1 LTS 标签: $LATEST_TAG"

# 检查补丁文件是否存在
if [[ ! -f "$PATCH_FILE" ]]; then
  log_error "补丁文件未找到: $PATCH_FILE"
  exit 1
fi

# 克隆
log_info "正在克隆 1Panel $LATEST_TAG..."
git clone --depth 1 --branch "$LATEST_TAG" "$REPO_URL" "$WORK_DIR"

# 验证版本兼容性
if ! grep -q 'go-acme/lego/v4' "${AGENT_DIR}/go.mod" 2>/dev/null; then
  log_error "此补丁仅支持使用 lego v4 的 1Panel v1"
  exit 1
fi

# 检查是否已打过补丁（幂等性）
if grep -q 'HttpReq' "${AGENT_DIR}/utils/ssl/dns_provider.go"; then
  log_warn "dns_provider.go 已包含 HttpReq — 补丁可能已应用"
  read -p "是否继续？[y/N] " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 0
  fi
fi

# 应用补丁
log_info "正在应用补丁..."
cd "$AGENT_DIR"
patch -p2 --dry-run < "$PATCH_FILE" >/dev/null 2>&1 || {
  log_error "补丁试应用失败 — 补丁可能与此版本的 1Panel 不匹配"
  exit 1
}
patch -p2 < "$PATCH_FILE"

# 编译
log_info "正在编译 1panel-agent..."
cd "$AGENT_DIR"
GOARCH=$(uname -m)
case "$GOARCH" in
  x86_64)  GOARCH=amd64 ;;
  aarch64) GOARCH=arm64 ;;
  armv7l)  GOARCH=arm ;;
esac
CGO_ENABLED=0 GOOS=linux GOARCH=$GOARCH go build -trimpath -ldflags '-s -w' -o 1panel-agent cmd/server/main.go

# 检测安装路径
INSTALL_BIN=""
if [[ -f /usr/local/bin/1panel-agent ]]; then
  INSTALL_BIN="/usr/local/bin/1panel-agent"
elif [[ -f /usr/bin/1panel-agent ]]; then
  INSTALL_BIN="/usr/bin/1panel-agent"
else
  log_warn "未找到已安装的 1panel-agent 二进制文件"
  log_info "编译产物: ${AGENT_DIR}/1panel-agent"
  log_info "请手动将其复制到 1Panel 安装目录"
  exit 0
fi

# 备份并安装
BACKUP_PATH="${INSTALL_BIN}.backup.$(date +%Y%m%d%H%M%S)"
log_info "正在备份现有二进制文件到 $BACKUP_PATH"
cp "$INSTALL_BIN" "$BACKUP_PATH"

log_info "正在安装新二进制文件..."
cp "${AGENT_DIR}/1panel-agent" "$INSTALL_BIN"
chmod +x "$INSTALL_BIN"

# 重启服务
if systemctl is-active --quiet 1panel 2>/dev/null || systemctl is-active --quiet 1panel-agent 2>/dev/null; then
  log_info "正在重启 1Panel 服务..."
  systemctl restart 1panel 2>/dev/null || systemctl restart 1panel-agent 2>/dev/null || {
    log_warn "无法自动重启服务，请手动重启。"
  }
else
  log_warn "未检测到 1Panel 服务，请手动重启。"
fi

log_info "补丁安装成功！"
log_info "备份文件: $BACKUP_PATH"
