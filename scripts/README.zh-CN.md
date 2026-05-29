# DomainNest 1Panel HttpReq 补丁

为 1Panel 添加 HttpReq DNS provider 支持。

## 适用版本

- **v1 LTS 分裂架构** (v1.10.28+, `1panel-core` + `1panel-agent`)
- **v1 LTS 单体架构** (旧版升级，`1panel` 单二进制)
- **v2 分裂架构** (`1panel-core` + `1panel-agent`)

要求：Go 1.21+、git、patch

> **v2 用户注意**：DomainNest 已提供原生 Technitium DNS 支持。如不需要 HttpReq DNS provider，则无需此补丁。

## 快速安装

```bash
cd 1panel-patch
chmod +x patch-1panel.sh
sudo ./patch-1panel.sh
```

## 脚本功能

- 自动检测 1Panel 版本和架构类型（分裂/单体，v1/v2）
- 自动选择对应补丁文件并验证兼容性
- 编译带 HttpReq 支持的新 `1panel-agent` 二进制
- 自动处理单体架构到分裂架构的迁移（二进制替换、符号链接、服务文件切换）
- 备份原文件，支持幂等性运行（安全可多次执行）
- 回滚脚本可恢复原始版本（自动检测迁移状态并完整还原）

## 回滚

```bash
chmod +x unpatch-1panel.sh
sudo ./unpatch-1panel.sh
```

- **分裂架构回滚**：恢复 `1panel-agent` 二进制，重启服务
- **单体架构回滚**：停止并禁用分裂架构服务，从备份恢复单体二进制、`1panel.service` 和符号链接

## 包内容

- `1panel-v1-httpreq.patch` — v1 LTS 补丁
- `1panel-v2-httpreq.patch` — v2 补丁
- `patch-1panel.sh` — 安装脚本
- `unpatch-1panel.sh` — 回滚脚本