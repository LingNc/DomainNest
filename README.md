# DomainNest

域名分配与 DDNS 管理系统。允许用户注册、获得子域名分配，支持无限级细分和 ddns-go 动态更新。

## 功能

- 邀请制注册：用户通过邀请码注册，邀请关系可溯源
- 用户管理：个人信息编辑、密码修改、头像上传、DDNS Token 重置
- 密码找回：通过注册邮箱发送重置链接
- 域名树管理：创建子域名、转让、删除，无限级细分
- DNS 记录管理：A/AAAA/CNAME/TXT/MX/SRV/CAA/NS/ALIAS，通过用户自定义提供商实时同步
- DNS 提供商管理：支持 27+ 提供商（阿里云、腾讯云、华为云、Cloudflare、GoDaddy、Namecheap、Vercel 等），用户添加 AK/SK 后认领域名并管理
- DDNS 双端点：支持 ddns-go 的 Callback 和 Webhook 两种模式
- 权限系统：域名读/写/管理员权限授予与回收
- 好友系统：搜索用户、发送/接受好友请求、好友列表管理
- 消息系统：用户私信 + 系统通知，可操作通知（如权限授予确认）
- RAM Token：API 编程访问令牌，支持域名/类型/IP 限制
- 多语言：中文 / English 切换
- 管理后台：用户管理、域名分配、操作日志、SMTP 设置
- 响应式 Web 界面（Vue 3 + Element Plus）

## 快速开始

### Docker 部署（推荐）

```bash
git clone <repo-url> && cd DomainNest
docker-compose up -d
# 访问 http://localhost:8080
# 默认管理员: admin / admin123
```

### 本地开发

**前置条件：** Go 1.25+、Node.js 20+、MySQL 8.0+

```bash
# 1. 创建数据库
mysql -u root -e "CREATE DATABASE domainnest CHARACTER SET utf8mb4;"

# 2. 修改配置
cp config.yaml.example config.yaml
# 编辑 config.yaml，填入数据库连接信息

# 3. 启动后端
go run ./cmd/server

# 4. 启动前端（另一个终端）
cd web && npm install && npm run dev
# 访问 http://localhost:3000
```

### 生产部署

```bash
make build              # 一条命令：编译前端 + 同步 + 构建 Go 二进制

# 或手动：
cd web && npm run build         # 前端编译，postbuild 自动同步到嵌入目录
go build -o domainnest ./cmd/server

./domainnest                    # 单端口运行，反代 :8080 即可
```

> Docker 构建会自动处理同步步骤，无需手动操作。

## 配置

编辑 `config.yaml`（从 `config.yaml.example` 复制）：

```yaml
server:
  port: 8080                    # 后端 API 端口（生产部署时前端也在此端口）
  frontend_port: 3000           # 仅开发模式 (npm run dev) 使用
  mode: debug                   # debug / release

database:
  host: localhost
  port: 3306
  user: root
  password: ""
  dbname: domainnest

jwt:
  secret: "change-me-to-a-random-secret"  # openssl rand -hex 32
  expire_hours: 24

admin:
  username: admin
  password: "admin123"          # 首次启动自动创建

smtp:                            # 可选，用于密码找回邮件
  host: smtp.example.com
  port: 587
  username: ""
  password: ""
  from: noreply@example.com
  from_name: DomainNest
```

程序从当前工作目录读取 `config.yaml`，也支持 `./config/` 子目录。

## ddns-go 对接

### Callback（逐域名更新）

| 字段 | 值 |
|------|-----|
| URL | `http://your-server:8080/api/v1/ddns/callback?token=你的Token` |
| Body | `{"domain":"#{domain}","ip":"#{ip}","record_type":"#{recordType}","ttl":#{ttl}}` |

### Webhook（聚合更新）

| 字段 | 值 |
|------|-----|
| URL | `http://your-server:8080/api/v1/ddns/webhook` |
| Body | `{"ipv4Addr":"#{ipv4Addr}","ipv4Domains":"#{ipv4Domains}","ipv6Addr":"#{ipv6Addr}","ipv6Domains":"#{ipv6Domains}"}` |
| Headers | `Authorization: Bearer 你的Token` |

Token 在登录后 → 个人信息页面查看或重置。

## 项目结构

```
cmd/server/              入口
internal/
  config/                配置管理
  model/                 数据库模型（GORM）
  handler/               HTTP 处理器（Gin）
  service/               业务逻辑
  middleware/             JWT 认证、管理员校验、在线追踪
  dns/                   DNS 提供商 SDK
  router/                路由注册
  static/                前端 embed (go:embed)
web/                     Vue 3 前端
  src/
    api/                 API 请求封装
    views/               页面组件
    stores/              Pinia 状态管理
    i18n/                中英文国际化
    router/              前端路由
    constants/           常量定义
```

## 技术栈

- **后端：** Go 1.25+ + Gin + GORM + MySQL 8.0+
- **前端：** Vue 3 + Element Plus + Pinia + Vue Router + vue-i18n + Axios
- **构建：** Vite（前端）、go:embed（前端嵌入二进制）
- **部署：** 单二进制（前端嵌入），支持 Docker

## 许可证

[MIT License](LICENSE) © 2026 LingNc
