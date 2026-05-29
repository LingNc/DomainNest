# DomainNest 1Panel HttpReq 补丁

## 适用版本

本补丁为 1Panel 添加 HttpReq DNS provider 支持，支持以下安装类型：

- **v1 LTS 分裂架构** (v1.10.28+, `1panel-core` + `1panel-agent`)
- **v1 LTS 单体架构** (旧版升级，`1panel` 单二进制)
- **v2 分裂架构** (`1panel-core` + `1panel-agent`)

要求：Go 1.21+、git、patch 命令

## 快速安装

```bash
# 下载补丁包并解压
cd 1panel-patch
chmod +x patch-1panel.sh
sudo ./patch-1panel.sh
```

脚本会自动：
1. 检测 1Panel 版本和架构（分裂/单体，v1/v2）
2. 选择对应补丁文件
3. 克隆对应版本的源码并验证兼容性
4. 应用补丁（支持幂等性检测）
5. 编译并备份原二进制
6. 重启服务

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

## 配置

在 1Panel 的「SSL 证书」→「DNS 账户」中添加：

- 类型：**HttpReq**
- 端点地址：`https://your-domainnest-server/httpreq`
- 用户名：DomainNest AccessKeyID
- 密码：DomainNest AccessKeySecret

## 回滚补丁

如需恢复原始版本：

```bash
chmod +x unpatch-1panel.sh
sudo ./unpatch-1panel.sh
```

脚本会自动查找备份文件并恢复。备份文件位于原二进制同目录，格式为 `*.backup.YYYYMMDDhhmmss`。