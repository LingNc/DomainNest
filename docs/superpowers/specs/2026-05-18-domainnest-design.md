# DomainNest - 域名分配与 DDNS 管理系统技术方案

## 1. 项目概述

### 1.1 目标

构建一套基于 Go 的 Web 管理系统，实现：
- 用户注册登录，获得子域名分配能力
- 无限级子域名细分，完整管理和转让权限
- 响应式 Web 界面，自助域名管理
- 安全 API 供 ddns-go 客户端动态更新 DNS

### 1.2 技术选型

| 组件 | 选型 | 版本 |
|------|------|------|
| 后端语言 | Go | 1.20+ |
| Web 框架 | Gin | latest |
| ORM | GORM | latest |
| 数据库 | MySQL | 8.0+ |
| 认证 | JWT (登录) + 静态 Token (DDNS) | - |
| 阿里云 SDK | alibabacloud-go/alidns-20150109/v5 | v5.4.1 |
| 配置管理 | Viper | latest |
| 前端 | Vue 3 + Element Plus + Vue Router + Pinia | - |
| 部署 | Docker + Go embed 嵌入前端 | - |

## 2. 系统架构

```
┌──────────────┐     ┌──────────────────────┐     ┌──────────────┐
│  Vue3 SPA    │────▶│   Go API Server      │────▶│  MySQL       │
│  (embed)     │     │   (Gin + GORM)       │     │              │
└──────────────┘     │                      │     └──────────────┘
                     │  /api/v1/auth/*       │
┌──────────────┐     │  /api/v1/users/*      │     ┌──────────────┐
│  ddns-go     │────▶│  /api/v1/domains/*    │────▶│  Alibaba DNS │
│  (Webhook)   │     │  /api/v1/records/*    │     │  API (V2.0)  │
└──────────────┘     │  /api/v1/ddns/*       │     └──────────────┘
                     │  /api/v1/admin/*      │
                     └──────────────────────┘
```

### 2.1 模块划分

```
DomainNest/
├── cmd/
│   └── server/          # 入口：初始化配置、DB、路由
├── internal/
│   ├── model/           # GORM 模型定义
│   ├── handler/         # HTTP handler
│   │   ├── auth.go      # 认证（登录/注册/Token）
│   │   ├── domain.go    # 域名节点管理
│   │   ├── record.go    # DNS 记录管理
│   │   ├── ddns.go      # DDNS 更新接口
│   │   └── admin.go     # 管理员接口
│   ├── service/         # 业务逻辑
│   │   ├── domain.go    # 域名树操作、转让
│   │   ├── record.go    # DNS 记录 CRUD + 同步
│   │   └── ddns.go      # DDNS 更新逻辑
│   ├── middleware/       # JWT 认证、Token 认证、操作日志
│   ├── aliyun/          # 阿里云 DNS SDK 封装
│   └── config/          # Viper 配置管理
├── web/                 # Vue3 前端源码
├── migrations/          # 数据库迁移脚本
├── config.yaml          # 配置文件模板
├── Dockerfile
└── docker-compose.yml
```

## 3. 数据库设计

### 3.1 ER 图

```
┌──────────────┐       ┌──────────────────┐       ┌──────────────┐
│    users     │       │   domain_nodes   │       │  dns_records │
├──────────────┤       ├──────────────────┤       ├──────────────┤
│ id (PK)      │◀──┐   │ id (PK)          │◀──┐   │ id (PK)      │
│ username     │   │   │ host             │   │   │ node_id (FK) │
│ password     │   │   │ full_domain (UQ) │   │   │ host         │
│ email        │   │   │ parent_id (FK)   │───┘   │ record_type  │
│ role         │   │   │ owner_id (FK)    │───────│ value        │
│ token (UQ)   │   └───│                  │       │ ttl          │
│ created_at   │       │ created_at       │       │ priority     │
│ updated_at   │       │ updated_at       │       │ line         │
└──────────────┘       └──────────────────┘       │ aliyun_record_id│
                                                    │ sync_status  │
┌──────────────┐                                    │ created_at   │
│operation_logs│                                    │ updated_at   │
├──────────────┤                                    └──────────────┘
│ id (PK)      │
│ user_id (FK) │
│ action       │
│ target_type  │
│ target_id    │
│ detail       │
│ ip_address   │
│ created_at   │
└──────────────┘
```

### 3.2 表结构详情

#### users 表

```sql
CREATE TABLE users (
    id          BIGINT PRIMARY KEY AUTO_INCREMENT,
    username    VARCHAR(64) NOT NULL UNIQUE,
    password    VARCHAR(255) NOT NULL,  -- bcrypt 加密
    email       VARCHAR(128),
    role        ENUM('admin', 'user') DEFAULT 'user',
    token       VARCHAR(64) NOT NULL UNIQUE,  -- DDNS 静态密钥
    created_at  DATETIME,
    updated_at  DATETIME
);
```

#### domain_nodes 表

```sql
CREATE TABLE domain_nodes (
    id          BIGINT PRIMARY KEY AUTO_INCREMENT,
    host        VARCHAR(64) NOT NULL,       -- 本级主机名
    full_domain VARCHAR(255) NOT NULL UNIQUE, -- 完整域名（冗余）
    parent_id   BIGINT,                     -- 根节点为 NULL
    owner_id    BIGINT NOT NULL,
    created_at  DATETIME,
    updated_at  DATETIME,
    FOREIGN KEY (parent_id) REFERENCES domain_nodes(id),
    FOREIGN KEY (owner_id) REFERENCES users(id),
    INDEX idx_full_domain (full_domain),
    INDEX idx_owner (owner_id),
    INDEX idx_parent (parent_id)
);
```

#### dns_records 表

```sql
CREATE TABLE dns_records (
    id               BIGINT PRIMARY KEY AUTO_INCREMENT,
    node_id          BIGINT NOT NULL,
    host             VARCHAR(64) NOT NULL DEFAULT '@',  -- RR
    record_type      VARCHAR(10) NOT NULL,
    value            VARCHAR(512) NOT NULL,
    ttl              INT DEFAULT 600,
    priority         INT,
    line             VARCHAR(32) DEFAULT 'default',
    aliyun_record_id VARCHAR(64),
    sync_status      ENUM('pending','synced','failed') DEFAULT 'pending',
    created_at       DATETIME,
    updated_at       DATETIME,
    FOREIGN KEY (node_id) REFERENCES domain_nodes(id),
    INDEX idx_node (node_id),
    INDEX idx_sync (sync_status)
);
```

#### operation_logs 表

```sql
CREATE TABLE operation_logs (
    id          BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id     BIGINT NOT NULL,
    action      VARCHAR(32) NOT NULL,
    target_type VARCHAR(32),
    target_id   BIGINT,
    detail      TEXT,
    ip_address  VARCHAR(64),
    created_at  DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id),
    INDEX idx_user_time (user_id, created_at)
);
```

### 3.3 关键设计说明

- `domain_nodes.full_domain` 冗余存储完整域名，配合唯一索引实现 O(1) 查询
- `dns_records.aliyun_record_id` 存储阿里云 RecordId，用于后续更新/删除
- `dns_records.sync_status` 支持失败重试机制
- 转让节点时：事务内递归更新子树所有节点的 `owner_id`

## 4. API 设计

### 4.1 认证接口

#### POST /api/v1/auth/register

```json
// Request
{
  "username": "user1",
  "password": "pass123",
  "email": "user1@example.com"
}

// Response 200
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "username": "user1"
  }
}
```

#### POST /api/v1/auth/login

```json
// Request
{
  "username": "user1",
  "password": "pass123"
}

// Response 200
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": 1,
      "username": "user1",
      "role": "user"
    }
  }
}
```

#### GET /api/v1/auth/profile

Headers: `Authorization: Bearer <JWT>`

```json
// Response 200
{
  "code": 0,
  "data": {
    "id": 1,
    "username": "user1",
    "email": "user1@example.com",
    "role": "user",
    "ddns_token": "abc123..."
  }
}
```

#### PUT /api/v1/auth/token

重置 DDNS Token，返回新 Token。

#### PUT /api/v1/auth/password

```json
{
  "old_password": "pass123",
  "new_password": "newpass456"
}
```

### 4.2 域名节点接口

#### GET /api/v1/domains

列出当前用户名下所有域名节点。

```json
// Response 200
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "host": "a",
      "full_domain": "a.xxxx.com",
      "parent_id": null,
      "record_count": 3,
      "children": [
        {
          "id": 2,
          "host": "sub",
          "full_domain": "sub.a.xxxx.com",
          "record_count": 1
        }
      ]
    }
  ]
}
```

#### POST /api/v1/domains

创建子域名节点。

```json
// Request
{
  "parent_id": 1,
  "host": "myself"
}

// Response 200
{
  "code": 0,
  "data": {
    "id": 3,
    "host": "myself",
    "full_domain": "myself.a.xxxx.com"
  }
}
```

#### POST /api/v1/domains/:id/transfer

转让节点（含所有子节点和 DNS 记录）。

```json
// Request
{
  "target_user_id": 2
}

// Response 200
{
  "code": 0,
  "message": "transfer successful"
}
```

#### DELETE /api/v1/domains/:id

删除节点（仅当无子节点且无 DNS 记录时允许）。

### 4.3 DNS 记录接口

#### GET /api/v1/domains/:id/records

```json
// Response 200
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "host": "@",
      "record_type": "A",
      "value": "1.2.3.4",
      "ttl": 600,
      "sync_status": "synced",
      "full_record": "a.xxxx.com"
    }
  ]
}
```

#### POST /api/v1/domains/:id/records

```json
// Request
{
  "host": "www",
  "record_type": "A",
  "value": "1.2.3.4",
  "ttl": 600
}

// Response 200
{
  "code": 0,
  "data": {
    "id": 2,
    "host": "www",
    "record_type": "A",
    "value": "1.2.3.4",
    "aliyun_record_id": "9999985",
    "sync_status": "synced"
  }
}
```

#### PUT /api/v1/records/:id

```json
{
  "value": "5.6.7.8",
  "ttl": 300
}
```

#### DELETE /api/v1/records/:id

同时删除阿里云端记录。

### 4.4 DDNS 更新接口

#### POST /api/v1/ddns/update

**认证方式（三选一）：**
1. URL 参数：`?token=YOUR_TOKEN`
2. Body 字段：`"token": "YOUR_TOKEN"`
3. Header：`Authorization: Bearer YOUR_TOKEN`

**ddns-go Webhook 配置：**
```
URL:         https://your-domain.com/api/v1/ddns/update?token=YOUR_TOKEN
RequestBody: {"domain":"#{domain}","ip":"#{ip}","record_type":"#{recordType}","ttl":#{ttl}}
```

**请求体：**
```json
{
  "domain": "home.a.xxxx.com",
  "ip": "1.2.3.4",
  "record_type": "A",
  "ttl": 600,
  "token": "YOUR_TOKEN"
}
```

**响应：**
```json
// 成功
{
  "code": 0,
  "message": "success",
  "data": {
    "domain": "home.a.xxxx.com",
    "ip": "1.2.3.4",
    "record_type": "A",
    "action": "updated"
  }
}

// 失败
{
  "code": 403,
  "message": "domain not found or access denied"
}
```

**处理逻辑：**
1. 提取 Token（URL > Body > Header），查找用户
2. 解析 `domain`，查找最长匹配的域名节点（用户拥有）
3. 计算 RR（host 前缀）
4. 查找该节点下同 RR + 类型的记录
5. 存在 → 更新阿里云 + 更新本地；不存在 → 创建阿里云 + 创建本地
6. 返回结果（幂等：相同 IP 更新返回 success）

### 4.5 管理员接口

#### POST /api/v1/admin/domains

创建根域名节点。

```json
{
  "host": "xxxx",
  "domain_suffix": "com"
}
```

#### POST /api/v1/admin/domains/:id/assign

分配根节点给用户。

```json
{
  "user_id": 1
}
```

#### GET /api/v1/admin/users

列出所有用户。

#### GET /api/v1/admin/logs

查看操作日志（支持分页、按用户/时间筛选）。

#### POST /api/v1/admin/records/:id/sync

重试失败的阿里云同步。

## 5. 阿里云 DNS 集成

### 5.1 SDK 版本

使用 V2.0 SDK：`github.com/alibabacloud-go/alidns-20150109/v5`

V1.0 SDK (`github.com/aliyun/alibaba-cloud-sdk-go`) 已于 2025-03-01 停止维护。

### 5.2 RAM 权限

最小权限策略：

```json
{
  "Effect": "Allow",
  "Action": [
    "alidns:AddDomainRecord",
    "alidns:UpdateDomainRecord",
    "alidns:DeleteDomainRecord",
    "alidns:DescribeDomainRecords",
    "alidns:DescribeSubDomainRecords",
    "alidns:DescribeDomainRecordInfo",
    "alidns:SetDomainRecordStatus"
  ],
  "Resource": "*"
}
```

### 5.3 核心操作

```go
// 创建记录
req := &alidns.AddDomainRecordRequest{
    DomainName: tea.String("xxxx.com"),
    RR:         tea.String("www"),
    Type:       tea.String("A"),
    Value:      tea.String("1.2.3.4"),
    TTL:        tea.Int64(600),
}
resp, err := client.AddDomainRecordWithOptions(req, &util.RuntimeOptions{})
// 保存 resp.Body.RecordId 到本地

// 更新记录
req := &alidns.UpdateDomainRecordRequest{
    RecordId: tea.String("9999985"),
    RR:       tea.String("www"),
    Type:     tea.String("A"),
    Value:    tea.String("5.6.7.8"),
    TTL:      tea.Int64(600),
}
resp, err := client.UpdateDomainRecordWithOptions(req, &util.RuntimeOptions{})

// 删除记录
req := &alidns.DeleteDomainRecordRequest{
    RecordId: tea.String("9999985"),
}
resp, err := client.DeleteDomainRecordWithOptions(req, &util.RuntimeOptions{})
```

### 5.4 同步策略

每条 DNS 记录维护 `sync_status` 字段：
- `pending`：刚创建，等待同步
- `synced`：已同步到阿里云
- `failed`：同步失败，可手动重试

操作流程：
1. 写入本地 DB（status=pending）
2. 调用阿里云 API
3. 成功 → 更新 status=synced，保存 aliyun_record_id
4. 失败 → 更新 status=failed，记录错误信息

## 6. 域名查找算法

### 6.1 问题描述

ddns-go 发送 `domain=sub.a.xxxx.com`，需要找到：
- 哪个域名节点拥有这个域名
- 对应的 DNS 记录 host (RR) 是什么

### 6.2 算法

**最长前缀匹配：**

```
输入: domain = "sub.a.xxxx.com", user_id = 1

1. 查询: SELECT * FROM domain_nodes
   WHERE owner_id = 1
   AND (full_domain = 'sub.a.xxxx.com'
        OR 'sub.a.xxxx.com' LIKE CONCAT('%.', full_domain))
   ORDER BY LENGTH(full_domain) DESC
   LIMIT 1

2. 若找到节点 node（full_domain = "a.xxxx.com"）:
   - RR = strings.TrimSuffix("sub.a.xxxx.com", ".a.xxxx.com") = "sub"

3. 若找到节点 node（full_domain = "sub.a.xxxx.com"）:
   - RR = "@"

4. 若未找到:
   - 返回 403，域名不属于该用户
```

### 6.3 索引策略

`domain_nodes.full_domain` 上的唯一索引支持精确匹配和 LIKE 查询。对于少量域名节点（通常 < 1000），性能无问题。

## 7. 节点转让逻辑

### 7.1 转让范围

转让节点时，以下内容一并转移：
- 该节点本身
- 所有子孙节点（递归）
- 所有节点上的 DNS 记录

### 7.2 实现

```sql
-- 在事务中执行
-- 1. 获取子树所有节点 ID
WITH RECURSIVE subtree AS (
    SELECT id FROM domain_nodes WHERE id = ?
    UNION ALL
    SELECT dn.id FROM domain_nodes dn JOIN subtree s ON dn.parent_id = s.id
)
-- 2. 批量更新 owner_id
UPDATE domain_nodes SET owner_id = ? WHERE id IN (SELECT id FROM subtree);
```

MySQL 8.0+ 支持递归 CTE。如果使用 MySQL 5.7，可以用应用层递归实现。

## 8. 安全设计

### 8.1 认证

- JWT Token：登录后签发，有效期可配置（默认 24h）
- DDNS Token：32 字符随机 hex 字符串，存于 users.token，支持前端查看和重置
- 密码：bcrypt 加密存储

### 8.2 授权

- 所有域名/记录操作必须校验 `owner_id = current_user_id`
- 管理员接口校验 `role = 'admin'`
- 转让操作记录审计日志

### 8.3 防越权

每个 handler 中间件注入当前用户 ID，service 层查询时强制加 `owner_id` 条件。

## 9. 前端设计

### 9.1 技术栈

- Vue 3 + Composition API
- Element Plus UI 组件库
- Vue Router 4
- Pinia 状态管理
- Axios HTTP 客户端

### 9.2 页面路由

| 路由 | 页面 | 说明 |
|------|------|------|
| `/login` | 登录页 | 账号密码登录 |
| `/register` | 注册页 | 注册新账号 |
| `/dashboard` | 仪表盘 | 我的域名节点树形列表 |
| `/domains/:id` | 节点详情 | DNS 记录管理（CRUD） |
| `/domains/:id/create-child` | 创建子域名 | 在节点下创建子域名 |
| `/domains/:id/transfer` | 转让域名 | 选择目标用户，确认转让 |
| `/settings` | 个人设置 | Token 管理、密码修改 |
| `/admin` | 管理后台 | 用户管理、根域名、操作日志 |

### 9.3 交互流程

```
登录/注册
    │
    ▼
仪表盘（域名节点树）
    │
    ├── 点击节点 ──▶ 节点详情
    │                   ├── DNS 记录列表（增删改查）
    │                   ├── 创建子域名
    │                   └── 转让域名
    │
    └── 设置页
            ├── DDNS Token（查看/重置）
            ├── ddns-go 配置示例
            └── 修改密码
```

### 9.4 ddns-go 配置指引（嵌入设置页）

在设置页展示 ddns-go 的配置示例，方便用户复制：

```
URL:         https://your-domain.com/api/v1/ddns/update?token={用户的Token}
RequestBody: {"domain":"#{domain}","ip":"#{ip}","record_type":"#{recordType}","ttl":#{ttl}}
```

## 10. 部署

### 10.1 配置文件 (config.yaml)

```yaml
server:
  port: 8080
  mode: release  # debug/release

database:
  host: localhost
  port: 3306
  user: domainnest
  password: ""
  dbname: domainnest

jwt:
  secret: ""
  expire_hours: 24

aliyun:
  access_key_id: ""
  access_key_secret: ""
  endpoint: alidns.aliyuncs.com

admin:
  username: admin
  password: ""  # 首次启动时自动创建管理员账号
```

**管理员初始化流程：**
1. 系统首次启动时，检查 `users` 表是否存在 `role='admin'` 的用户
2. 若不存在，读取 `config.yaml` 中的 `admin.username` 和 `admin.password`，自动创建管理员
3. 管理员通过 Web 界面或 API 创建根域名节点（如 `xxxx.com`），并分配给指定用户
4. 若配置文件中未设置 admin 密码，启动时通过 CLI 交互式输入

### 10.2 Dockerfile

```dockerfile
FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o domainnest ./cmd/server/

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/domainnest .
COPY --from=builder /app/config.yaml .
EXPOSE 8080
CMD ["./domainnest"]
```

### 10.3 docker-compose.yml

```yaml
version: '3'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      - db
    environment:
      - DB_HOST=db

  db:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: rootpass
      MYSQL_DATABASE: domainnest
      MYSQL_USER: domainnest
      MYSQL_PASSWORD: domainpass
    volumes:
      - mysql_data:/var/lib/mysql

volumes:
  mysql_data:
```

## 11. 开发任务分解

### Phase 1: 基础框架
1. 项目初始化（go mod、目录结构、配置管理）
2. 数据库连接和 GORM 模型定义
3. JWT 认证中间件
4. 操作日志中间件

### Phase 2: 核心后端
5. 用户注册/登录 API
6. 管理员 CLI 初始化（创建管理员 + 根域名）
7. 域名节点 CRUD API
8. 节点转让 API（含事务）
9. DNS 记录 CRUD API

### Phase 3: 阿里云集成
10. 阿里云 DNS SDK 封装
11. DNS 记录同步（创建/更新/删除）
12. 同步失败重试机制

### Phase 4: DDNS 接口
13. DDNS 更新 API（Token 认证 + 域名查找 + 自动创建/更新）

### Phase 5: 前端
14. Vue3 项目初始化 + 路由
15. 登录/注册页面
16. 仪表盘（域名节点树）
17. 节点详情（DNS 记录管理）
18. 创建子域名 + 转让页面
19. 个人设置（Token 管理）
20. 管理后台页面

### Phase 6: 部署
21. Go embed 嵌入前端静态文件
22. Dockerfile + docker-compose
23. 配置文件模板
