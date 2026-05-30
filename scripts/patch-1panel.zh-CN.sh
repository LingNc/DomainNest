#!/bin/bash
set -euo pipefail

# DomainNest 1Panel httpreq 补丁安装脚本
# 支持 v1 LTS（分裂/单体）和 v2（分裂架构）
# 自动检测 1Panel 版本和架构，选择对应补丁

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PATCH_FILE_V1="${SCRIPT_DIR}/1panel-v1-httpreq.patch"
PATCH_FILE_V2="${SCRIPT_DIR}/1panel-v2-httpreq.patch"

REPO_URL="https://github.com/1Panel-dev/1Panel.git"
WORK_DIR="/tmp/1panel-patch-$$"
CLEANUP_WORK_DIR=0
SERVICE_FILES_URL="https://github.com/1Panel-dev/installer/raw/v2/initscript"

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

cleanup() {
  if [[ $CLEANUP_WORK_DIR -eq 1 && -d "$WORK_DIR" ]]; then
    rm -rf "$WORK_DIR"
  fi
}
trap cleanup EXIT

# 检查依赖
ensure_command() {
  local cmd="$1"
  local pkg="$2"
  command -v "$cmd" >/dev/null 2>&1 && return 0
  log_error "需要安装 $cmd"
  if [[ ! -t 0 ]]; then
    log_error "非交互模式，无法安装。请手动运行: apt-get install -y $pkg"
    exit 1
  fi
  read -p "是否使用 apt 进行安装? [y/N] " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    log_info "正在安装 $pkg..."
    apt-get update && apt-get install -y "$pkg"
  else
    exit 1
  fi
}

# 可选依赖：尝试安装，失败则返回 1 而非退出
ensure_optional_command() {
  local cmd="$1"
  local pkg="$2"
  command -v "$cmd" >/dev/null 2>&1 && return 0
  log_warn "$cmd 未安装"
  if [[ ! -t 0 ]]; then
    log_warn "非交互模式，跳过安装 $cmd"
    return 1
  fi
  read -p "是否使用 apt 安装 $cmd? [y/N] " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    log_info "正在安装 $pkg..."
    if apt-get update && apt-get install -y "$pkg"; then
      return 0
    else
      log_warn "$pkg 安装失败"
      return 1
    fi
  fi
  return 1
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

# 可选依赖：npm（前端构建需要，后端补丁不受影响）
# 如果通过 sudo 运行，尝试将原用户的 node/npm 路径加入 PATH
if [[ -n "${SUDO_USER:-}" ]]; then
  SUDO_USER_HOME=$(eval echo "~${SUDO_USER}")
  for node_dir in \
    "${SUDO_USER_HOME}/.nvm/versions/node"/*/bin \
    "${SUDO_USER_HOME}/.local/share/nvm/versions/node"/*/bin \
    "${SUDO_USER_HOME}/.npm-global/bin" \
    "${SUDO_USER_HOME}/node_modules/.bin"; do
    if [[ -d "$node_dir" ]] && [[ ":$PATH:" != *":$node_dir:"* ]]; then
      export PATH="$node_dir:$PATH"
    fi
  done
fi
HAS_NPM=0
if ensure_optional_command npm npm; then
  HAS_NPM=1
fi

# ============================================================
# 检测 1Panel 安装状态
# ============================================================
log_step "正在检测 1Panel 安装状态..."

# 1. 检测分裂架构（1panel-core 存在）
CORE_BIN=""
for candidate in /usr/local/bin/1panel-core /usr/bin/1panel-core; do
    if [[ -f "$candidate" ]]; then
        CORE_BIN="$candidate"
        break
    fi
done

# fallback: 通过 PATH 查找
if [[ -z "$CORE_BIN" ]]; then
    CORE_BIN=$(command -v 1panel-core 2>/dev/null || echo "")
fi

# fallback: 解析符号链接（v1.10.34-lts 创建 /usr/bin/1panel -> 1panel-core）
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
    # 分裂架构检测到，解析版本号
    log_info "检测到 1Panel 分裂架构: $CORE_BIN"
    PANEL_VERSION=$(timeout 5 "$CORE_BIN" version 2>/dev/null || echo "")
    if [[ "$PANEL_VERSION" == *"v2"* ]]; then
        INSTALL_TYPE="split-v2"
        log_info "检测到 v2 版本"
    else
        INSTALL_TYPE="split-v1"
        log_info "检测到 v1 LTS 版本"
    fi
else
    # 2. 检测单体架构（存在非 symlink 的 1panel 二进制）
    MONOLITHIC_BIN=""
    for p in /usr/local/bin/1panel /usr/bin/1panel; do
        if [[ -f "$p" ]]; then
            target=$(readlink -f "$p" 2>/dev/null)
            # 如果不是指向 1panel-core 的 symlink，则为单体
            if [[ "$target" != *"1panel-core"* ]]; then
                MONOLITHIC_BIN="$p"
                break
            fi
        fi
    done

    if [[ -z "$MONOLITHIC_BIN" ]]; then
        if ! systemctl list-unit-files 2>/dev/null | grep -q '1panel'; then
            log_error "1Panel 未安装"
            exit 1
        fi
        log_error "未找到 1Panel 二进制文件"
        exit 1
    fi

    log_info "检测到 1Panel 单体架构: $MONOLITHIC_BIN"
    PANEL_VERSION=$(timeout 5 "$MONOLITHIC_BIN" version 2>/dev/null || echo "")
    if [[ "$PANEL_VERSION" == *"v2"* ]]; then
        INSTALL_TYPE="monolithic-v2"
        log_info "检测到 v2 版本"
    else
        INSTALL_TYPE="monolithic-v1"
        log_info "检测到 v1 版本"
    fi
fi

# v2 警告：v2 有官方 HTTP Request 支持且更新频繁
if [[ "$INSTALL_TYPE" == *"v2"* ]]; then
    log_warn "检测到 1Panel v2。"
    log_warn "DomainNest 已原生支持 Technitium DNS 提供商，无需为此打补丁。"
    log_warn "如果你确实需要使用 HttpReq 提供商，请继续。"
    read -r -p "是否继续安装补丁？[y/N] " response
    if [[ ! "$response" =~ ^[yY]$ ]]; then
        exit 0
    fi
fi

# ============================================================
# 确定使用的补丁和克隆目标
# ============================================================
case "$INSTALL_TYPE" in
  split-v1|monolithic-v1)
    PATCH_FILE="$PATCH_FILE_V1"
    TAG_PATTERN='v1.*-lts'
    log_info "使用 v1 LTS 补丁"
    ;;
  split-v2|monolithic-v2)
    PATCH_FILE="$PATCH_FILE_V2"
    TAG_PATTERN='v2.*'
    log_info "使用 v2 补丁"
    ;;
esac

# 检查补丁文件是否存在
if [[ ! -f "$PATCH_FILE" ]]; then
  log_error "补丁文件未找到: $PATCH_FILE"
  exit 1
fi

# 查找最新标签
log_info "正在查找最新的 1Panel 标签 (${TAG_PATTERN})..."
LATEST_TAG=$(git ls-remote --tags "$REPO_URL" "refs/tags/${TAG_PATTERN}" 2>/dev/null | \
  awk -F/ '{print $NF}' | sort -V | tail -1)

if [[ -z "$LATEST_TAG" ]]; then
  log_error "未找到标签 pattern: ${TAG_PATTERN}"
  exit 1
fi
log_info "最新的标签: $LATEST_TAG"

# 更新为基于标签的目录名
WORK_DIR="/tmp/1panel-patch-${LATEST_TAG}"

# 清理旧的 PID-based 目录
rm -rf /tmp/1panel-patch-[0-9]* 2>/dev/null || true

# 克隆或复用源码
if [[ -d "$WORK_DIR" ]] && git -C "$WORK_DIR" describe --tags 2>/dev/null | grep -q "$LATEST_TAG"; then
    log_info "复用已有的源码目录: $WORK_DIR"
    CLEANUP_WORK_DIR=0
else
    rm -rf "$WORK_DIR"
    log_info "正在克隆 1Panel $LATEST_TAG..."
    git clone --depth 1 --branch "$LATEST_TAG" "$REPO_URL" "$WORK_DIR"
    CLEANUP_WORK_DIR=0
fi

# ============================================================
# 确定源码目录结构（分裂 vs 单体）
# ============================================================
SRC_SUBDIR=""  # "agent" 或 "backend"
if [[ -d "${WORK_DIR}/agent" ]]; then
  SRC_SUBDIR="agent"
elif [[ -d "${WORK_DIR}/backend" ]]; then
  SRC_SUBDIR="backend"
else
  log_error "未找到 agent/ 或 backend/ 目录"
  exit 1
fi
SRC_DIR="${WORK_DIR}/${SRC_SUBDIR}"
log_info "源码目录: $SRC_DIR"

HAS_SPLIT_SOURCE=0
if [[ -d "${WORK_DIR}/agent" && -d "${WORK_DIR}/core" ]]; then
  HAS_SPLIT_SOURCE=1
fi

# 验证 lego 版本兼容性
if ! grep -q 'go-acme/lego/v4' "${SRC_DIR}/go.mod" 2>/dev/null; then
  log_error "此补丁仅支持使用 lego v4 的 1Panel"
  exit 1
fi

# 检查 Go 版本兼容性
GO_MOD_VERSION=$(grep '^go ' "${SRC_DIR}/go.mod" | awk '{print $2}')
if [[ -n "$GO_MOD_VERSION" ]]; then
  GO_MOD_MAJOR_MINOR=$(echo "$GO_MOD_VERSION" | cut -d. -f1,2)
  GO_INSTALLED_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
  GO_INSTALLED_MAJOR_MINOR=$(echo "$GO_INSTALLED_VERSION" | cut -d. -f1,2)
  if [[ "$(printf '%s\n' "$GO_MOD_MAJOR_MINOR" "$GO_INSTALLED_MAJOR_MINOR" | sort -V | head -1)" != "$GO_MOD_MAJOR_MINOR" ]]; then
    log_error "Go 版本过低: 已安装 $GO_INSTALLED_VERSION, 需要 $GO_MOD_VERSION+"
    if [[ ! -t 0 ]]; then
      log_error "非交互模式，请手动升级 Go: https://go.dev/dl/"
      exit 1
    fi
    read -p "是否自动下载安装 Go $GO_MOD_VERSION？[y/N] " -n 1 -r
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
      log_info "正在下载 Go $GO_MOD_VERSION..."
      curl -fSL# "$GO_DOWNLOAD_URL" -o "/tmp/${GO_TARBALL}" || {
        log_error "下载失败，请手动升级 Go: https://go.dev/dl/"
        exit 1
      }
      log_info "正在安装 Go $GO_MOD_VERSION..."
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
        log_info "已将 Go PATH 写入 $SHELL_RC"
      fi
      log_info "Go 已升级: $(go version)"
    else
      log_error "请手动升级 Go 后重试"
      exit 1
    fi
  fi
fi

PATCH_ALREADY_APPLIED=false
if grep -q 'HttpReq' "${SRC_DIR}/utils/ssl/dns_provider.go" 2>/dev/null; then
  log_info "后端补丁已应用，跳过"
  PATCH_ALREADY_APPLIED=true
fi

if [[ "$PATCH_ALREADY_APPLIED" == "true" ]]; then
  log_info "补丁已存在，跳过应用补丁步骤"
else
  # ============================================================
  # 应用补丁
  # ============================================================
  # 对于单体架构源码，需要调整补丁路径：a/agent -> a/backend
  if [[ "$SRC_SUBDIR" == "backend" ]]; then
    log_info "单体架构源码，转换补丁路径 (agent -> backend)..."
    PATCH_CONTENT=$(cat "$PATCH_FILE")
    # 只在 patch 文件头部行转换，保留后续内容
    PATCH_CONTENT=$(echo "$PATCH_CONTENT" | sed 's|^a/agent/|a/backend/|' | sed 's|^b/agent/|b/backend/|')
    TEMP_PATCH_FILE="${WORK_DIR}/adjusted.patch"
    echo "$PATCH_CONTENT" > "$TEMP_PATCH_FILE"
    APPLY_PATCH="$TEMP_PATCH_FILE"
  else
    APPLY_PATCH="$PATCH_FILE"
  fi

  log_info "正在应用补丁..."
  cd "$SRC_DIR"
  patch -p2 --dry-run < "$APPLY_PATCH" >/dev/null 2>&1 || {
    log_error "补丁试应用失败 — 补丁可能与此版本的 1Panel 不匹配"
    log_error "补丁文件: $APPLY_PATCH"
    log_error "源码目录: $SRC_DIR"
    exit 1
  }
  patch -p2 < "$APPLY_PATCH"
fi

# ============================================================
# 应用前端补丁并构建前端
# ============================================================
FRONTEND_PATCH="${SCRIPT_DIR}/1panel-httpreq-frontend.patch"
if [[ -f "$FRONTEND_PATCH" && -d "${WORK_DIR}/frontend" ]]; then
  log_info "正在检查前端补丁..."
  (
    cd "${WORK_DIR}/frontend"
    # 先检查是否已应用（反向 dry-run 成功说明已应用）
    if patch -p1 -R --dry-run < "$FRONTEND_PATCH" >/dev/null 2>&1; then
      log_info "前端补丁已应用，跳过"
    elif patch -p1 --dry-run < "$FRONTEND_PATCH" >/dev/null 2>&1; then
      # 正向 dry-run 成功说明可以应用
      log_info "正在应用前端补丁..."
      patch -p1 < "$FRONTEND_PATCH"
    else
      log_warn "前端补丁不兼容，跳过"
    fi
  )

  if [[ $HAS_NPM -eq 1 ]]; then
    NPM_LOG=$(mktemp)
    npm_start=$SECONDS

    log_info "正在安装前端依赖..."
    (
      cd "${WORK_DIR}/frontend"
      npm install --prefer-offline
    ) >"$NPM_LOG" 2>&1 &
    npm_pid=$!

    sleep 0.5
    echo -e "${GREEN}[INFO]${NC} 正在下载前端依赖 (首次可能需要几分钟)..."
    last_printed=""
    while kill -0 "$npm_pid" 2>/dev/null; do
      elapsed=$(( SECONDS - npm_start ))
      last_line=$(tail -c 500 "$NPM_LOG" 2>/dev/null | grep -v '^$' | tail -1 || true)
      if [[ -n "$last_line" && "$last_line" != "$last_printed" ]]; then
        last_printed="$last_line"
        echo -ne "  ${last_line} (${elapsed}s)\033[K\r"
      fi
      sleep 2
    done
    echo -e "\033[K"

    rc=0
    wait "$npm_pid" || rc=$?
    npm_elapsed=$(( SECONDS - npm_start ))
    if [[ $rc -ne 0 ]]; then
      log_error "npm install 失败 (${npm_elapsed}s)，日志如下:"
      cat "$NPM_LOG"
      rm -f "$NPM_LOG"
      exit 1
    else
      log_info "前端依赖安装完成 (${npm_elapsed}s)"
      : > "$NPM_LOG"

      log_info "正在构建前端..."
      (
        cd "${WORK_DIR}/frontend"
        # 创建 xpack stub（商业版模块不存在于开源仓库）
        mkdir -p "${WORK_DIR}/frontend/src/xpack/api/modules"
        cat > "${WORK_DIR}/frontend/src/xpack/api/modules/appstore.ts" << 'STUB'
export const installAppToNodes = (): Promise<any> => {
    return Promise.reject(new Error('Multi-node install requires 1Panel Pro'));
};
STUB
        npm run build:pro
      ) >"$NPM_LOG" 2>&1 &
      npm_pid=$!

      sleep 0.5
      echo -e "${GREEN}[INFO]${NC} 正在编译前端..."
      last_printed=""
      while kill -0 "$npm_pid" 2>/dev/null; do
        elapsed=$(( SECONDS - npm_start ))
        last_line=$(tail -c 500 "$NPM_LOG" 2>/dev/null | grep -v '^$' | tail -1 || true)
        if [[ -n "$last_line" && "$last_line" != "$last_printed" ]]; then
          last_printed="$last_line"
          echo -ne "  ${last_line} (${elapsed}s)\033[K\r"
        fi
        sleep 2
      done
      echo -e "\033[K"

      rc=0
      wait "$npm_pid" || rc=$?
      npm_elapsed=$(( SECONDS - npm_start ))
      if [[ $rc -ne 0 ]]; then
        log_error "前端构建失败 (${npm_elapsed}s)，日志如下:"
        cat "$NPM_LOG"
        rm -f "$NPM_LOG"
        exit 1
      else
        log_info "前端构建完成 (${npm_elapsed}s)"
      fi
    fi
    rm -f "$NPM_LOG"

    # Copy built frontend to core web directory
    if [[ -d "${WORK_DIR}/frontend/dist" ]]; then
      rm -rf "${WORK_DIR}/core/cmd/server/web"
      cp -r "${WORK_DIR}/frontend/dist" "${WORK_DIR}/core/cmd/server/web"
      log_info "前端构建产物已复制"
    else
      log_warn "前端构建产物未找到，将使用原有前端"
    fi
  else
    log_warn "npm 未安装，跳过前端构建。HttpReq 将不出现在 DNS 提供商列表中"
  fi
fi

# ============================================================
# 编译
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

  log_info "正在编译 ${label}..."
  local build_start=$SECONDS
  (
    cd "$src_dir"
    CGO_ENABLED=0 GOOS=linux GOARCH=$GOARCH \
      go build -trimpath -ldflags '-s -w' -o "$output_name" "$build_target"
  ) >"$BUILD_LOG" 2>&1 &
  local build_pid=$!

  sleep 0.5
  echo -e "${GREEN}[INFO]${NC} 正在下载依赖并编译 ${label} (首次可能需要几分钟)..."
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

  local rc=0
  wait "$build_pid" || rc=$?
  elapsed=$(( SECONDS - build_start ))
  if [[ $rc -ne 0 ]]; then
    log_error "${label} 编译失败 (耗时 ${elapsed}s)，日志如下:"
    cat "$BUILD_LOG"
    rm -f "$BUILD_LOG"
    exit 1
  fi
  log_info "${label} 编译完成 (耗时 ${elapsed}s)"
  : > "$BUILD_LOG"
}

if [[ "$INSTALL_TYPE" == "monolithic-v1" && "$HAS_SPLIT_SOURCE" -eq 1 ]]; then
  build_binary "$SRC_DIR" "cmd/server/main.go" "1panel-agent" "1panel-agent (含 HttpReq 补丁)"
  AGENT_BUILD_OUTPUT="${SRC_DIR}/1panel-agent"
  if [[ ! -f "$AGENT_BUILD_OUTPUT" ]]; then
    log_error "编译产物未找到: $AGENT_BUILD_OUTPUT"
    exit 1
  fi

  CORE_SRC_DIR="${WORK_DIR}/core"
  build_binary "$CORE_SRC_DIR" "cmd/server/main.go" "1panel-core" "1panel-core (Web UI)"
  CORE_BUILD_OUTPUT="${CORE_SRC_DIR}/1panel-core"
  if [[ ! -f "$CORE_BUILD_OUTPUT" ]]; then
    log_error "编译产物未找到: $CORE_BUILD_OUTPUT"
    exit 1
  fi
else
  BUILD_TARGET="cmd/server/main.go"
  if [[ "$SRC_SUBDIR" == "backend" ]]; then
    BUILD_TARGET="cmd/1panel/main.go"
  fi
  log_info "构建目标: $BUILD_TARGET"
  build_binary "$SRC_DIR" "$BUILD_TARGET" "1panel-agent" "1panel-agent"

  # Also build 1panel-core if frontend was patched (core embeds frontend)
  if [[ -d "${WORK_DIR}/core/cmd/server/web" ]]; then
    build_binary "${WORK_DIR}/core" "cmd/server/main.go" "1panel-core" "1panel-core (含前端补丁)"
  fi
fi

# ============================================================
# 安装
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
    log_error "未找到单体 1panel 二进制文件"
    exit 1
  fi

  NEW_BIN_DIR="/usr/local/bin"

  log_info "正在停止 1panel 服务..."
  if systemctl is-active --quiet 1panel 2>/dev/null; then
    systemctl stop 1panel || {
      log_error "无法停止 1panel 服务"
      exit 1
    }
  fi

  # 备份原始服务文件
  if [[ -f /etc/systemd/system/1panel.service ]]; then
    cp /etc/systemd/system/1panel.service /etc/systemd/system/1panel.service.backup
    log_info "已备份 1panel.service 到 /etc/systemd/system/1panel.service.backup"
  fi

  BACKUP_PATH="${MONO_BIN}.backup.$(date +%Y%m%d%H%M%S)"
  log_info "正在备份单体二进制文件到 $BACKUP_PATH"
  cp "$MONO_BIN" "$BACKUP_PATH"

  log_info "正在安装 1panel-agent 到 ${NEW_BIN_DIR}/1panel-agent"
  cp "$AGENT_BUILD_OUTPUT" "${NEW_BIN_DIR}/1panel-agent"
  chmod +x "${NEW_BIN_DIR}/1panel-agent"

  log_info "正在安装 1panel-core 到 ${NEW_BIN_DIR}/1panel-core"
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

  # 替换单体二进制为 symlink
  if [[ -f "${NEW_BIN_DIR}/1panel" && ! -L "${NEW_BIN_DIR}/1panel" ]]; then
    rm -f "${NEW_BIN_DIR}/1panel"
    ln -sf "${NEW_BIN_DIR}/1panel-core" "${NEW_BIN_DIR}/1panel"
    log_info "已将 ${NEW_BIN_DIR}/1panel 替换为指向 1panel-core 的符号链接"
  fi

  log_info "正在下载 systemd 服务文件..."
  SVC_DOWNLOAD_OK=1
  curl -fSL# "${SERVICE_FILES_URL}/1panel-core.service" \
    -o /etc/systemd/system/1panel-core.service 2>/dev/null || SVC_DOWNLOAD_OK=0
  curl -fSL# "${SERVICE_FILES_URL}/1panel-agent.service" \
    -o /etc/systemd/system/1panel-agent.service 2>/dev/null || SVC_DOWNLOAD_OK=0

  if [[ "$SVC_DOWNLOAD_OK" -eq 0 ]]; then
    log_error "无法从 GitHub 下载服务文件。"
    log_error "URL:"
    log_error "  ${SERVICE_FILES_URL}/1panel-core.service"
    log_error "  ${SERVICE_FILES_URL}/1panel-agent.service"
    log_error ""
    log_error "二进制文件已安装但服务未配置。"
    log_error "请手动下载服务文件到 /etc/systemd/system/"
    log_error "然后运行: systemctl daemon-reload && systemctl enable --now 1panel-core 1panel-agent"
    CLEANUP_WORK_DIR=1
    rm -f "$BUILD_LOG"
    exit 1
  fi

  systemctl daemon-reload

  log_info "正在禁用旧 1panel.service..."
  systemctl disable 1panel 2>/dev/null || true

  log_info "正在启用 1panel-core.service 和 1panel-agent.service..."
  systemctl enable 1panel-core 1panel-agent

  log_info "正在启动 1panel-core 和 1panel-agent..."
  systemctl start 1panel-core || {
    log_warn "无法启动 1panel-core，请检查: journalctl -u 1panel-core"
  }
  systemctl start 1panel-agent || {
    log_warn "无法启动 1panel-agent，请检查: journalctl -u 1panel-agent"
  }

  log_info "迁移完成: 单体架构 -> 分裂架构"
  log_info "旧单体二进制备份: $BACKUP_PATH"

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
    log_warn "未找到已安装的二进制文件"
    log_info "编译产物: ${SRC_DIR}/1panel-agent"
    log_info "请手动将其复制到 1Panel 安装目录"
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
    log_info "正在停止 $SERVICE_NAME 服务..."
    systemctl stop "$SERVICE_NAME" || {
      log_error "无法停止 $SERVICE_NAME 服务"
      exit 1
    }
  fi

  BACKUP_PATH="${INSTALL_BIN}.backup.$(date +%Y%m%d%H%M%S)"
  log_info "正在备份现有二进制文件到 $BACKUP_PATH"
  cp "$INSTALL_BIN" "$BACKUP_PATH"

  log_info "正在安装新二进制文件..."
  cp "${SRC_DIR}/1panel-agent" "$INSTALL_BIN"
  chmod +x "$INSTALL_BIN"

  # Install 1panel-core if it was rebuilt (with frontend patch)
  if [[ -f "${WORK_DIR}/core/1panel-core" ]]; then
    CORE_BIN_PATH=""
    if [[ -f /usr/local/bin/1panel-core ]]; then
      CORE_BIN_PATH="/usr/local/bin/1panel-core"
    elif [[ -f /usr/bin/1panel-core ]]; then
      CORE_BIN_PATH="/usr/bin/1panel-core"
    fi
    if [[ -n "$CORE_BIN_PATH" ]]; then
      log_info "正在安装新 1panel-core..."
      cp "${WORK_DIR}/core/1panel-core" "$CORE_BIN_PATH"
      chmod +x "$CORE_BIN_PATH"
    fi
  fi

  if [[ $SERVICE_WAS_ACTIVE -eq 1 ]] || systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    log_info "正在启动 $SERVICE_NAME 服务..."
    systemctl start "$SERVICE_NAME" || {
      log_warn "无法自动启动服务，请手动启动。"
    }
  else
    log_warn "未检测到 $SERVICE_NAME 服务，请手动启动。"
  fi
fi

CLEANUP_WORK_DIR=1
log_info "补丁安装成功！"
log_info "备份文件: $BACKUP_PATH"