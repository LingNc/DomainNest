# DomainNest

域名分配与 DDNS 管理系统。允许用户注册、获得子域名分配，支持无限级细分和 ddns-go 动态更新。

## 功能

- 用户注册/登录，JWT 认证
- 域名树管理：创建子域名、转让、删除
- DNS 记录管理：A/AAAA/CNAME/TXT/MX，实时同步阿里云
- DDNS Webhook 接口，兼容 ddns-go
- 管理后台：用户管理、根域名创建、操作日志
- 响应式 Web 界面（Vue3 + Element Plus）

## 快速开始

### Docker 部署（推荐）

```bash
# 克隆项目
git clone <repo-url> && cd DomainNest

# 启动
docker-compose up -d

# 访问
# Web: http://localhost:8080
# 默认管理员: admin / admin123
```

### 本地开发

**前置条件：** Go 1.20+、Node.js 20+、MySQL 8.0+

```bash
# 1. 创建数据库
mysql -u root -e "CREATE DATABASE domainnest CHARACTER SET utf8mb4;"

# 2. 修改配置
cp config.yaml.example config.yaml
# 编辑 config.yaml，填入数据库密码和阿里云 AK/SK

# 3. 启动后端
go run ./cmd/server/

# 4. 启动前端（开发模式，另一个终端）
cd web && npm install && npm run dev
# 访问 http://localhost:3000
```

## 配置

编辑 `config.yaml`：

```yaml
server:
  port: 8080

database:
  host: localhost
  port: 3306
  user: root
  password: "your-password"
  dbname: domainnest

jwt:
  secret: "换成随机字符串"  # 用 openssl rand -hex 32 生成
  expire_hours: 24

aliyun:
  access_key_id: "你的阿里云 AK"
  access_key_secret: "你的阿里云 SK"
  endpoint: alidns.aliyuncs.com

admin:
  username: admin
  password: "admin123"  # 首次启动自动创建
```

环境变量可覆盖配置（前缀 `DB_`、`JWT_SECRET` 等），适合 Docker 部署。

## ddns-go 对接

在 ddns-go 的 Webhook 设置中配置：

| 字段 | 值 |
|------|-----|
| URL | `http://your-server:8080/api/v1/ddns/update?token=你的Token` |
| RequestBody | `{"domain":"#{domain}","ip":"#{ip}","record_type":"#{recordType}","ttl":#{ttl}}` |

Token 在登录后 → 设置页面查看或重置。

## API 概览

| 接口 | 方法 | 认证 | 说明 |
|------|------|------|------|
| `/api/v1/auth/register` | POST | 无 | 注册 |
| `/api/v1/auth/login` | POST | 无 | 登录 |
| `/api/v1/domains` | GET/POST | JWT | 域名列表/创建 |
| `/api/v1/domains/:id/records` | GET/POST | JWT | DNS 记录列表/创建 |
| `/api/v1/ddns/update` | POST | Token | DDNS 更新 |
| `/api/v1/admin/*` | 各种 | JWT+Admin | 管理接口 |

## 项目结构

```
cmd/server/          入口
internal/
  config/            配置管理
  model/             数据库模型
  handler/           HTTP 处理器
  service/           业务逻辑
  middleware/         认证、日志
  aliyun/            阿里云 DNS SDK
  router/            路由注册
  static/            前端 embed
web/                 Vue3 前端
```

## 许可证

[MIT License](LICENSE) © 2026 LingNc
