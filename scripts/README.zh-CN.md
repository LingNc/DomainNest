# DomainNest 1Panel HttpReq 补丁

## 背景

1Panel 存在两种架构模式：

- **分裂架构**：由 `1panel-core`（Web UI API 网关，监听 TCP 端口）和 `1panel-agent`（后端 worker，监听 Unix 套接字 `/etc/1panel/agent.sock`）组成。该架构从 v1.10.28+ 开始引入。
- **单体架构**：单一的 `1panel` 二进制处理所有功能（旧版 legacy 模式）。

v1.10.34-lts 源码使用分裂架构，因此补丁脚本可以将单体架构安装迁移到分裂架构。

## 适用版本

本补丁为 1Panel 添加 HttpReq DNS provider 支持，支持以下安装类型：

- **v1 LTS 分裂架构** (v1.10.28+, `1panel-core` + `1panel-agent`)
- **v1 LTS 单体架构** (旧版升级，`1panel` 单二进制)
- **v2 分裂架构** (`1panel-core` + `1panel-agent`)

要求：Go 1.21+、git、patch 命令

> **v2 用户注意**：DomainNest 已提供原生的 Technitium DNS 支持。如果您不需要 HttpReq DNS provider，则无需使用此补丁。

## 功能特点

- 自动检测并安装依赖（git、patch、curl、Go）
- Go 版本自动下载和安装（如果系统 Go 版本过低）
- 实时显示编译进度和耗时
- 单体架构到分裂架构迁移时自动下载服务文件
- 幂等性设计（安全可多次运行）
- 版本检测命令超时保护

## 快速安装

```bash
# 下载补丁包并解压
cd 1panel-patch
chmod +x patch-1panel.sh
sudo ./patch-1panel.sh
```

脚本自动执行以下步骤：

**分裂架构（v1 / v2）：**
1. 检测 1Panel 版本和架构（分裂/单体，v1/v2）
2. 选择对应补丁文件
3. 克隆对应版本的源码并验证兼容性
4. 应用补丁（支持幂等性检测）
5. 编译并备份原 `1panel-agent` 二进制
6. 安装新二进制并重启服务

**单体架构 v1（v1.10.34-lts）：**
1. 检测 1Panel 版本和架构
2. 选择对应补丁文件
3. 克隆 v1.10.34-lts 源码并验证兼容性
4. 应用补丁（支持幂等性检测）
5. 编译 `1panel-agent`（含 HttpReq patch）和 `1panel-core`（Web UI）
6. 备份原单体二进制为 `1panel.backup.TIMESTAMP`
7. 将两个新二进制安装到 `/usr/local/bin/`
8. 将 `/usr/local/bin/1panel` 替换为指向 `1panel-core` 的符号链接
9. 从 GitHub 下载 `1panel-core.service` 和 `1panel-agent.service`
10. 禁用旧 `1panel.service`，启用并启动新的两个服务

## 单体架构 v1：自动迁移至分裂架构

对于单体架构的 v1 安装，脚本执行以下迁移操作：

- 编译 `1panel-agent`（含 HttpReq 补丁）和 `1panel-core`
- 将原单体二进制备份为 `1panel.backup.TIMESTAMP`
- 将 `1panel.service` 备份为 `1panel.service.backup`
- 将两个新二进制安装到 `/usr/local/bin/`
- 将 `/usr/local/bin/1panel` 替换为指向 `1panel-core` 的符号链接
- 从 GitHub 下载官方的 `1panel-core.service` 和 `1panel-agent.service`
- 禁用旧的 `1panel.service`，启用并启动两个新服务
- 迁移完成后，系统运行分裂架构（与官方 v1.10.34-lts 版本相同）

## 手动步骤

### v1 LTS 分裂架构

```bash
git clone --depth 1 --branch v1.10.34-lts https://github.com/1Panel-dev/1Panel.git /tmp/1panel
cd /tmp/1panel/agent
patch -p2 < /path/to/1panel-v1-httpreq.patch
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o 1panel-agent cmd/server/main.go
sudo cp 1panel-agent /usr/local/bin/1panel-agent
sudo systemctl restart 1panel-agent
```

### v2 分裂架构

```bash
git clone --depth 1 --branch v2.1.13 https://github.com/1Panel-dev/1Panel.git /tmp/1panel
cd /tmp/1panel/agent
patch -p2 < /path/to/1panel-v2-httpreq.patch
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o 1panel-agent cmd/server/main.go
sudo cp 1panel-agent /usr/local/bin/1panel-agent
sudo systemctl restart 1panel-agent
```

> 对于单体架构 v1 安装，建议使用自动脚本进行迁移，因为该迁移涉及多个二进制文件、符号链接和服务文件的更改，手动执行较为复杂。

## 配置

在 1Panel 的「SSL 证书」→「DNS 账户」中添加：

- 类型：**HttpReq**
- 端点地址：`https://your-domainnest-server/httpreq`
- 用户名：DomainNest AccessKeyID
- 密码：DomainNest AccessKeySecret

## 回滚

如需恢复原始版本：

```bash
chmod +x unpatch-1panel.sh
sudo ./unpatch-1panel.sh
```

脚本会自动查找备份文件并恢复。

### 分裂架构回滚

- 从备份恢复 `1panel-agent` 二进制
- 重启服务

### 已迁移单体架构回滚

脚本通过备份文件检测迁移状态，然后：

1. 停止 `1panel-agent` 和 `1panel-core` 服务
2. 禁用并移除分裂架构服务及其服务文件
3. 从备份恢复单体二进制（正确处理符号链接）
4. 从备份恢复 `1panel.service`
5. 重新启用并启动 `1panel.service`

## 包内容

补丁包包含以下文件：

- `1panel-v1-httpreq.patch` — v1 LTS 补丁文件
- `1panel-v2-httpreq.patch` — v2 补丁文件
- `patch-1panel.sh` — 安装脚本（自动选择语言）
- `unpatch-1panel.sh` — 回滚脚本
- `README.md` — 本文件