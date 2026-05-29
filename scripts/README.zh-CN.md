# DomainNest 1Panel v1 httpreq 补丁

## 适用版本

- 1Panel v1.x LTS（已测试 v1.10.34-lts）
- 要求：Go 1.21+、git、patch 命令

## 快速安装

```bash
# 下载补丁包并解压
cd 1panel-v1-patch
chmod +x patch-1panel-v1.sh
sudo ./patch-1panel-v1.sh
```

脚本会自动：
1. 获取最新的 1Panel v1 LTS 标签
2. 克隆源码并检查兼容性
3. 应用补丁（支持重复检测，不会重复打补丁）
4. 编译 1panel-agent
5. 备份原二进制并替换
6. 重启服务

## 手动步骤

如需手动操作：

```bash
git clone --depth 1 --branch v1.10.34-lts https://github.com/1Panel-dev/1Panel.git /tmp/1panel
cd /tmp/1panel/agent
patch -p2 < /path/to/1panel-v1-httpreq.patch
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o 1panel-agent cmd/server/main.go
sudo cp 1panel-agent /usr/local/bin/1panel-agent
sudo systemctl restart 1panel
```

## 配置

在 1Panel 的「SSL 证书」→「DNS 账户」中添加：

- 类型：**HttpReq**
- 端点地址：`https://your-domainnest-server/httpreq`
- 用户名：DomainNest AccessKeyID
- 密码：DomainNest AccessKeySecret
