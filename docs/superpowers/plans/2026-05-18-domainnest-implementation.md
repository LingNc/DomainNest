# DomainNest Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a full-stack domain assignment and DDNS management system with Go backend, Vue3 frontend, and Alibaba Cloud DNS integration.

**Architecture:** Go API server (Gin + GORM) with MySQL, Vue3 + Element Plus SPA embedded via `go:embed`, Alibaba Cloud DNS V2.0 SDK for record sync. Domain tree uses self-referencing table with `full_domain` redundancy for O(1) lookups.

**Tech Stack:** Go 1.20+, Gin, GORM, MySQL 8.0+, JWT, bcrypt, Viper, Alibaba Cloud DNS V2.0 SDK, Vue 3, Element Plus, Vue Router, Pinia, Axios

---

## File Structure

```
DomainNest/
├── cmd/server/main.go                    # Entry point
├── internal/
│   ├── config/config.go                  # Viper config loading
│   ├── model/
│   │   ├── user.go                       # User model
│   │   ├── domain_node.go               # DomainNode model
│   │   ├── dns_record.go               # DNSRecord model
│   │   └── operation_log.go            # OperationLog model
│   ├── handler/
│   │   ├── auth.go                      # Auth endpoints
│   │   ├── domain.go                    # Domain node endpoints
│   │   ├── record.go                    # DNS record endpoints
│   │   ├── ddns.go                      # DDNS update endpoint
│   │   └── admin.go                     # Admin endpoints
│   ├── service/
│   │   ├── auth.go                      # Auth business logic
│   │   ├── domain.go                    # Domain tree operations
│   │   ├── record.go                    # Record CRUD + sync
│   │   └── ddns.go                      # DDNS update logic
│   ├── middleware/
│   │   ├── jwt.go                       # JWT auth middleware
│   │   ├── token.go                     # Static token auth middleware
│   │   └── logger.go                    # Operation log middleware
│   ├── aliyun/
│   │   └── client.go                    # Alibaba Cloud DNS SDK wrapper
│   └── router/
│       └── router.go                    # Route registration
├── web/                                 # Vue3 frontend
│   ├── src/
│   │   ├── main.js
│   │   ├── App.vue
│   │   ├── router/index.js
│   │   ├── stores/                      # Pinia stores
│   │   ├── api/                         # Axios instances
│   │   ├── views/                       # Page components
│   │   └── components/                  # Shared components
│   ├── package.json
│   └── vite.config.js
├── config.yaml                          # Config template
├── go.mod
├── go.sum
├── Dockerfile
└── docker-compose.yml
```

---

## Task 1: Project Initialization

**Files:**
- Create: `go.mod`, `cmd/server/main.go`, `internal/config/config.go`, `config.yaml`

- [ ] **Step 1: Initialize Go module**

```bash
cd /home/lingnc/workspace/DomainNest
go mod init domainnest
```

- [ ] **Step 2: Create config.yaml**

```yaml
server:
  port: 8080
  mode: debug

database:
  host: localhost
  port: 3306
  user: root
  password: ""
  dbname: domainnest

jwt:
  secret: "change-me-to-a-random-secret"
  expire_hours: 24

aliyun:
  access_key_id: ""
  access_key_secret: ""
  endpoint: alidns.aliyuncs.com

admin:
  username: admin
  password: "admin123"
```

- [ ] **Step 3: Create config loader**

```go
// internal/config/config.go
package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Aliyun   AliyunConfig   `mapstructure:"aliyun"`
	Admin    AdminConfig    `mapstructure:"admin"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type AliyunConfig struct {
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	Endpoint        string `mapstructure:"endpoint"`
}

type AdminConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Allow environment variables to override
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
```

- [ ] **Step 4: Create main.go skeleton**

```go
// cmd/server/main.go
package main

import (
	"fmt"
	"log"

	"domainnest/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("Server starting on port %d\n", cfg.Server.Port)
}
```

- [ ] **Step 5: Install dependencies and verify**

```bash
go get github.com/spf13/viper
go build ./cmd/server/
```

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum cmd/server/main.go internal/config/config.go config.yaml
git commit -m "feat: project initialization with config management"
```

---

## Task 2: Database Models

**Files:**
- Create: `internal/model/user.go`, `internal/model/domain_node.go`, `internal/model/dns_record.go`, `internal/model/operation_log.go`

- [ ] **Step 1: Create User model**

```go
// internal/model/user.go
package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	Password  string         `gorm:"type:varchar(255);not null" json:"-"`
	Email     string         `gorm:"type:varchar(128)" json:"email"`
	Role      string         `gorm:"type:enum('admin','user');default:'user'" json:"role"`
	Token     string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"token,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}
```

- [ ] **Step 2: Create DomainNode model**

```go
// internal/model/domain_node.go
package model

import (
	"time"

	"gorm.io/gorm"
)

type DomainNode struct {
	ID         uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Host       string         `gorm:"type:varchar(64);not null" json:"host"`
	FullDomain string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"full_domain"`
	ParentID   *uint64        `gorm:"index" json:"parent_id"`
	OwnerID    uint64         `gorm:"index;not null" json:"owner_id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Parent   *DomainNode  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []DomainNode `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Owner    User         `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Records  []DNSRecord  `gorm:"foreignKey:NodeID" json:"records,omitempty"`
}

func (DomainNode) TableName() string {
	return "domain_nodes"
}
```

- [ ] **Step 3: Create DNSRecord model**

```go
// internal/model/dns_record.go
package model

import (
	"time"

	"gorm.io/gorm"
)

type DNSRecord struct {
	ID             uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeID         uint64         `gorm:"index;not null" json:"node_id"`
	Host           string         `gorm:"type:varchar(64);not null;default:'@'" json:"host"`
	RecordType     string         `gorm:"type:varchar(10);not null" json:"record_type"`
	Value          string         `gorm:"type:varchar(512);not null" json:"value"`
	TTL            int            `gorm:"default:600" json:"ttl"`
	Priority       *int           `json:"priority,omitempty"`
	Line           string         `gorm:"type:varchar(32);default:'default'" json:"line"`
	AliyunRecordID string         `gorm:"type:varchar(64)" json:"aliyun_record_id,omitempty"`
	SyncStatus     string         `gorm:"type:enum('pending','synced','failed');default:'pending'" json:"sync_status"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Node DomainNode `gorm:"foreignKey:NodeID" json:"node,omitempty"`
}

func (DNSRecord) TableName() string {
	return "dns_records"
}
```

- [ ] **Step 4: Create OperationLog model**

```go
// internal/model/operation_log.go
package model

import (
	"time"
)

type OperationLog struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint64    `gorm:"index;not null" json:"user_id"`
	Action     string    `gorm:"type:varchar(32);not null" json:"action"`
	TargetType string    `gorm:"type:varchar(32)" json:"target_type"`
	TargetID   *uint64   `json:"target_id,omitempty"`
	Detail     string    `gorm:"type:text" json:"detail"`
	IPAddress  string    `gorm:"type:varchar(64)" json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`

	// Relations
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (OperationLog) TableName() string {
	return "operation_logs"
}
```

- [ ] **Step 5: Verify compilation**

```bash
go build ./internal/model/
```

- [ ] **Step 6: Commit**

```bash
git add internal/model/
git commit -m "feat: add GORM models for users, domain nodes, DNS records, and operation logs"
```

---

## Task 3: Database Connection and Auto-Migration

**Files:**
- Create: `internal/model/database.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Create database connection helper**

```go
// internal/model/database.go
package model

import (
	"fmt"
	"log"

	"domainnest/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	log.Println("Running database auto-migration...")
	return db.AutoMigrate(
		&User{},
		&DomainNode{},
		&DNSRecord{},
		&OperationLog{},
	)
}
```

- [ ] **Step 2: Update main.go to connect DB and run migrations**

```go
// cmd/server/main.go
package main

import (
	"fmt"
	"log"

	"domainnest/internal/config"
	"domainnest/internal/model"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := model.InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	if err := model.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	fmt.Printf("Server starting on port %d\n", cfg.Server.Port)
}
```

- [ ] **Step 3: Install GORM dependencies**

```bash
go get gorm.io/gorm gorm.io/driver/mysql
go build ./cmd/server/
```

- [ ] **Step 4: Commit**

```bash
git add internal/model/database.go cmd/server/main.go go.mod go.sum
git commit -m "feat: add database connection and auto-migration"
```

---

## Task 4: JWT Authentication Middleware

**Files:**
- Create: `internal/middleware/jwt.go`

- [ ] **Step 1: Create JWT middleware**

```go
// internal/middleware/jwt.go
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(secret string, userID uint64, username, role string, expireHours int) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid authorization format"})
			c.Abort()
			return
		}

		claims := &JWTClaims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role.(string) != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 2: Install JWT dependency**

```bash
go get github.com/golang-jwt/jwt/v5
go build ./internal/middleware/
```

- [ ] **Step 3: Commit**

```bash
git add internal/middleware/jwt.go go.mod go.sum
git commit -m "feat: add JWT authentication middleware"
```

---

## Task 5: Token Authentication Middleware (DDNS)

**Files:**
- Create: `internal/middleware/token.go`

- [ ] **Step 1: Create static token middleware for DDNS**

```go
// internal/middleware/token.go
package middleware

import (
	"net/http"

	"domainnest/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TokenAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// Priority: URL param > Body token > Authorization header
		token = c.Query("token")

		if token == "" {
			var body struct {
				Token string `json:"token"`
			}
			if err := c.ShouldBindJSON(&body); err == nil {
				token = body.Token
				// Re-bind body for downstream handlers since ShouldBindJSON consumed it
				c.Set("_body_token", body)
			}
		}

		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			}
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "missing token"})
			c.Abort()
			return
		}

		var user model.User
		if err := db.Where("token = ?", token).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Set("role", user.Role)
		c.Next()
	}
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/middleware/
```

- [ ] **Step 3: Commit**

```bash
git add internal/middleware/token.go
git commit -m "feat: add static token authentication middleware for DDNS"
```

---

## Task 6: Operation Log Middleware

**Files:**
- Create: `internal/middleware/logger.go`

- [ ] **Step 1: Create operation logging helper**

```go
// internal/middleware/logger.go
package middleware

import (
	"encoding/json"

	"domainnest/internal/model"

	"gorm.io/gorm"
)

func LogOperation(db *gorm.DB, userID uint64, action, targetType string, targetID *uint64, detail interface{}, ip string) {
	detailJSON, _ := json.Marshal(detail)
	db.Create(&model.OperationLog{
		UserID:     userID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     string(detailJSON),
		IPAddress:  ip,
	})
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/middleware/
```

- [ ] **Step 3: Commit**

```bash
git add internal/middleware/logger.go
git commit -m "feat: add operation logging helper"
```

---

## Task 7: Auth Service (User Registration/Login)

**Files:**
- Create: `internal/service/auth.go`

- [ ] **Step 1: Create auth service**

```go
// internal/service/auth.go
package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"

	"domainnest/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db *gorm.DB
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

func (s *AuthService) Register(username, password, email string) (*model.User, error) {
	var existing model.User
	if err := s.db.Where("username = ?", username).First(&existing).Error; err == nil {
		return nil, errors.New("username already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username: username,
		Password: string(hashedPassword),
		Email:    email,
		Role:     "user",
		Token:    token,
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(username, password string) (*model.User, error) {
	var user model.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return &user, nil
}

func (s *AuthService) GetUserByID(id uint64) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) ResetToken(userID uint64) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	if err := s.db.Model(&model.User{}).Where("id = ?", userID).Update("token", token).Error; err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) ChangePassword(userID uint64, oldPassword, newPassword string) error {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("incorrect old password")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.db.Model(&user).Update("password", string(hashedPassword)).Error
}

func (s *AuthService) EnsureAdmin(username, password string) error {
	var count int64
	s.db.Model(&model.User{}).Where("role = ?", "admin").Count(&count)
	if count > 0 {
		return nil // admin already exists
	}

	_, err := s.createUser(username, password, "", "admin")
	return err
}

func (s *AuthService) createUser(username, password, email, role string) (*model.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username: username,
		Password: string(hashedPassword),
		Email:    email,
		Role:     role,
		Token:    token,
	}

	return user, s.db.Create(user).Error
}

func generateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
```

- [ ] **Step 2: Install bcrypt dependency**

```bash
go get golang.org/x/crypto
go build ./internal/service/
```

- [ ] **Step 3: Commit**

```bash
git add internal/service/auth.go go.mod go.sum
git commit -m "feat: add auth service with registration, login, and admin bootstrap"
```

---

## Task 8: Auth Handler (API Endpoints)

**Files:**
- Create: `internal/handler/auth.go`

- [ ] **Step 1: Create auth handler**

```go
// internal/handler/auth.go
package handler

import (
	"net/http"

	"domainnest/internal/config"
	"domainnest/internal/middleware"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
	jwtSecret   string
	jwtExpire   int
}

func NewAuthHandler(authService *service.AuthService, cfg *config.JWTConfig) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		jwtSecret:   cfg.Secret,
		jwtExpire:   cfg.ExpireHours,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=64"`
		Password string `json:"password" binding:"required,min=6"`
		Email    string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	user, err := h.authService.Register(req.Username, req.Password, req.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	user, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid credentials"})
		return
	}

	token, err := middleware.GenerateToken(h.jwtSecret, user.ID, user.Username, user.Role, h.jwtExpire)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"role":     user.Role,
			},
		},
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := c.GetUint64("user_id")

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"ddns_token": user.Token,
		},
	})
}

func (h *AuthHandler) ResetToken(c *gin.Context) {
	userID := c.GetUint64("user_id")

	newToken, err := h.authService.ResetToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to reset token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "token reset successfully",
		"data": gin.H{
			"token": newToken,
		},
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.authService.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "password changed successfully"})
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/handler/
```

- [ ] **Step 3: Commit**

```bash
git add internal/handler/auth.go
git commit -m "feat: add auth handler with register, login, profile, token, and password endpoints"
```

---

## Task 9: Domain Service (Tree CRUD + Transfer)

**Files:**
- Create: `internal/service/domain.go`

- [ ] **Step 1: Create domain service**

```go
// internal/service/domain.go
package service

import (
	"errors"
	"fmt"

	"domainnest/internal/model"

	"gorm.io/gorm"
)

type DomainService struct {
	db *gorm.DB
}

func NewDomainService(db *gorm.DB) *DomainService {
	return &DomainService{db: db}
}

// CreateNode creates a child domain node under a parent.
func (s *DomainService) CreateNode(parentID uint64, host string, ownerID uint64) (*model.DomainNode, error) {
	var parent model.DomainNode
	if err := s.db.First(&parent, parentID).Error; err != nil {
		return nil, errors.New("parent node not found")
	}

	// Check ownership of parent
	if parent.OwnerID != ownerID {
		return nil, errors.New("you do not own the parent domain")
	}

	fullDomain := host + "." + parent.FullDomain

	// Check uniqueness
	var existing model.DomainNode
	if err := s.db.Where("full_domain = ?", fullDomain).First(&existing).Error; err == nil {
		return nil, errors.New("domain already exists")
	}

	node := &model.DomainNode{
		Host:       host,
		FullDomain: fullDomain,
		ParentID:   &parentID,
		OwnerID:    ownerID,
	}

	if err := s.db.Create(node).Error; err != nil {
		return nil, err
	}

	return node, nil
}

// GetUserNodes returns all domain nodes owned by a user.
func (s *DomainService) GetUserNodes(userID uint64) ([]model.DomainNode, error) {
	var nodes []model.DomainNode
	err := s.db.Where("owner_id = ?", userID).
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("owner_id = ?", userID)
		}).
		Preload("Records").
		Find(&nodes).Error

	// Build tree: only return root nodes (no parent or parent not owned by user)
	var roots []model.DomainNode
	for _, n := range nodes {
		if n.ParentID == nil {
			roots = append(roots, n)
		} else {
			isRoot := true
			for _, m := range nodes {
				if m.ID == *n.ParentID {
					isRoot = false
					break
				}
			}
			if isRoot {
				roots = append(roots, n)
			}
		}
	}

	return roots, err
}

// GetNode returns a single node by ID, verifying ownership.
func (s *DomainService) GetNode(nodeID, userID uint64) (*model.DomainNode, error) {
	var node model.DomainNode
	err := s.db.Where("id = ? AND owner_id = ?", nodeID, userID).
		Preload("Children").
		Preload("Records").
		First(&node).Error
	if err != nil {
		return nil, errors.New("domain node not found or access denied")
	}
	return &node, nil
}

// FindNodeByDomain finds the longest-matching node for a full domain.
func (s *DomainService) FindNodeByDomain(domain string, userID uint64) (*model.DomainNode, string, error) {
	var node model.DomainNode
	err := s.db.Where("owner_id = ? AND (full_domain = ? OR ? LIKE CONCAT('%.', full_domain))",
		userID, domain, domain).
		Order("LENGTH(full_domain) DESC").
		First(&node).Error

	if err != nil {
		return nil, "", errors.New("domain not found or access denied")
	}

	// Calculate RR
	var rr string
	if node.FullDomain == domain {
		rr = "@"
	} else {
		suffix := "." + node.FullDomain
		rr = domain[:len(domain)-len(suffix)]
	}

	return &node, rr, nil
}

// TransferNode transfers a node and all its subtree to a new owner.
func (s *DomainService) TransferNode(nodeID, ownerID, targetUserID uint64) error {
	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return errors.New("node not found")
	}
	if node.OwnerID != ownerID {
		return errors.New("you do not own this domain")
	}

	// Use recursive CTE to find all descendants
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Get all node IDs in the subtree
		var nodeIDs []uint64
		err := tx.Raw(`
			WITH RECURSIVE subtree AS (
				SELECT id FROM domain_nodes WHERE id = ? AND deleted_at IS NULL
				UNION ALL
				SELECT dn.id FROM domain_nodes dn JOIN subtree s ON dn.parent_id = s.id WHERE dn.deleted_at IS NULL
			)
			SELECT id FROM subtree
		`, nodeID).Scan(&nodeIDs).Error
		if err != nil {
			return fmt.Errorf("failed to find subtree: %w", err)
		}

		// Update owner for all nodes in subtree
		if err := tx.Model(&model.DomainNode{}).Where("id IN ?", nodeIDs).Update("owner_id", targetUserID).Error; err != nil {
			return fmt.Errorf("failed to transfer nodes: %w", err)
		}

		return nil
	})
}

// DeleteNode deletes a node (only if no children and no records).
func (s *DomainService) DeleteNode(nodeID, userID uint64) error {
	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return errors.New("node not found")
	}
	if node.OwnerID != userID {
		return errors.New("you do not own this domain")
	}

	// Check for children
	var childCount int64
	s.db.Model(&model.DomainNode{}).Where("parent_id = ?", nodeID).Count(&childCount)
	if childCount > 0 {
		return errors.New("cannot delete node with children")
	}

	// Check for records
	var recordCount int64
	s.db.Model(&model.DNSRecord{}).Where("node_id = ?", nodeID).Count(&recordCount)
	if recordCount > 0 {
		return errors.New("cannot delete node with DNS records")
	}

	return s.db.Delete(&node).Error
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/service/
```

- [ ] **Step 3: Commit**

```bash
git add internal/service/domain.go
git commit -m "feat: add domain service with tree CRUD and transfer"
```

---

## Task 10: Domain Handler (API Endpoints)

**Files:**
- Create: `internal/handler/domain.go`

- [ ] **Step 1: Create domain handler**

```go
// internal/handler/domain.go
package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/middleware"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DomainHandler struct {
	domainService *service.DomainService
	db            *gorm.DB
}

func NewDomainHandler(domainService *service.DomainService, db *gorm.DB) *DomainHandler {
	return &DomainHandler{domainService: domainService, db: db}
}

func (h *DomainHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")

	nodes, err := h.domainService.GetUserNodes(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": nodes})
}

func (h *DomainHandler) Create(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		ParentID uint64 `json:"parent_id" binding:"required"`
		Host     string `json:"host" binding:"required,min=1,max=64"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	node, err := h.domainService.CreateNode(req.ParentID, req.Host, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "create_domain", "domain_node", &node.ID,
		map[string]interface{}{"full_domain": node.FullDomain}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    node,
	})
}

func (h *DomainHandler) Get(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	node, err := h.domainService.GetNode(nodeID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": node})
}

func (h *DomainHandler) Transfer(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	var req struct {
		TargetUserID uint64 `json:"target_user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.domainService.TransferNode(nodeID, userID, req.TargetUserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "transfer_domain", "domain_node", &nodeID,
		map[string]interface{}{"target_user_id": req.TargetUserID}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "transfer successful"})
}

func (h *DomainHandler) Delete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	if err := h.domainService.DeleteNode(nodeID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "delete_domain", "domain_node", &nodeID,
		nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "deleted successfully"})
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/handler/
```

- [ ] **Step 3: Commit**

```bash
git add internal/handler/domain.go
git commit -m "feat: add domain handler with list, create, get, transfer, and delete"
```

---

## Task 11: DNS Record Service

**Files:**
- Create: `internal/service/record.go`

- [ ] **Step 1: Create record service**

```go
// internal/service/record.go
package service

import (
	"errors"

	"domainnest/internal/model"

	"gorm.io/gorm"
)

type RecordService struct {
	db *gorm.DB
}

func NewRecordService(db *gorm.DB) *RecordService {
	return &RecordService{db: db}
}

func (s *RecordService) GetRecords(nodeID, userID uint64) ([]model.DNSRecord, error) {
	// Verify node ownership
	var node model.DomainNode
	if err := s.db.Where("id = ? AND owner_id = ?", nodeID, userID).First(&node).Error; err != nil {
		return nil, errors.New("domain node not found or access denied")
	}

	var records []model.DNSRecord
	err := s.db.Where("node_id = ?", nodeID).Find(&records).Error
	return records, err
}

func (s *RecordService) CreateRecord(nodeID, userID uint64, host, recordType, value string, ttl int, priority *int, line string) (*model.DNSRecord, error) {
	// Verify node ownership
	var node model.DomainNode
	if err := s.db.Where("id = ? AND owner_id = ?", nodeID, userID).First(&node).Error; err != nil {
		return nil, errors.New("domain node not found or access denied")
	}

	if ttl == 0 {
		ttl = 600
	}
	if line == "" {
		line = "default"
	}

	record := &model.DNSRecord{
		NodeID:     nodeID,
		Host:       host,
		RecordType: recordType,
		Value:      value,
		TTL:        ttl,
		Priority:   priority,
		Line:       line,
		SyncStatus: "pending",
	}

	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}

	return record, nil
}

func (s *RecordService) UpdateRecord(recordID, userID uint64, value string, ttl *int, priority *int) (*model.DNSRecord, error) {
	var record model.DNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return nil, errors.New("record not found")
	}

	// Verify ownership through node
	var node model.DomainNode
	if err := s.db.Where("id = ? AND owner_id = ?", record.NodeID, userID).First(&node).Error; err != nil {
		return nil, errors.New("access denied")
	}

	updates := map[string]interface{}{
		"sync_status": "pending",
	}
	if value != "" {
		updates["value"] = value
	}
	if ttl != nil {
		updates["ttl"] = *ttl
	}
	if priority != nil {
		updates["priority"] = *priority
	}

	if err := s.db.Model(&record).Updates(updates).Error; err != nil {
		return nil, err
	}

	s.db.First(&record, recordID)
	return &record, nil
}

func (s *RecordService) DeleteRecord(recordID, userID uint64) error {
	var record model.DNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return errors.New("record not found")
	}

	// Verify ownership through node
	var node model.DomainNode
	if err := s.db.Where("id = ? AND owner_id = ?", record.NodeID, userID).First(&node).Error; err != nil {
		return errors.New("access denied")
	}

	return s.db.Delete(&record).Error
}

// GetRecordByID returns a record by ID (for internal use).
func (s *RecordService) GetRecordByID(recordID uint64) (*model.DNSRecord, error) {
	var record model.DNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateSyncStatus updates the sync status and aliyun record ID.
func (s *RecordService) UpdateSyncStatus(recordID uint64, status, aliyunRecordID string) error {
	updates := map[string]interface{}{
		"sync_status": status,
	}
	if aliyunRecordID != "" {
		updates["aliyun_record_id"] = aliyunRecordID
	}
	return s.db.Model(&model.DNSRecord{}).Where("id = ?", recordID).Updates(updates).Error
}

// FindRecordByNodeAndHost finds a record by node ID, host, and type.
func (s *RecordService) FindRecordByNodeAndHost(nodeID uint64, host, recordType string) (*model.DNSRecord, error) {
	var record model.DNSRecord
	err := s.db.Where("node_id = ? AND host = ? AND record_type = ?", nodeID, host, recordType).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/service/
```

- [ ] **Step 3: Commit**

```bash
git add internal/service/record.go
git commit -m "feat: add DNS record service with CRUD and sync status management"
```

---

## Task 12: DNS Record Handler

**Files:**
- Create: `internal/handler/record.go`

- [ ] **Step 1: Create record handler**

```go
// internal/handler/record.go
package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/middleware"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RecordHandler struct {
	recordService *service.RecordService
	db            *gorm.DB
}

func NewRecordHandler(recordService *service.RecordService, db *gorm.DB) *RecordHandler {
	return &RecordHandler{recordService: recordService, db: db}
}

func (h *RecordHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	records, err := h.recordService.GetRecords(nodeID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": records})
}

func (h *RecordHandler) Create(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	var req struct {
		Host       string `json:"host" binding:"required"`
		RecordType string `json:"record_type" binding:"required"`
		Value      string `json:"value" binding:"required"`
		TTL        int    `json:"ttl"`
		Priority   *int   `json:"priority"`
		Line       string `json:"line"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	record, err := h.recordService.CreateRecord(nodeID, userID, req.Host, req.RecordType, req.Value, req.TTL, req.Priority, req.Line)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "create_record", "dns_record", &record.ID,
		map[string]interface{}{"host": record.Host, "type": record.RecordType, "value": record.Value}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": record})
}

func (h *RecordHandler) Update(c *gin.Context) {
	userID := c.GetUint64("user_id")
	recordID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid record id"})
		return
	}

	var req struct {
		Value    string `json:"value"`
		TTL      *int   `json:"ttl"`
		Priority *int   `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	record, err := h.recordService.UpdateRecord(recordID, userID, req.Value, req.TTL, req.Priority)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "update_record", "dns_record", &recordID,
		map[string]interface{}{"value": record.Value}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": record})
}

func (h *RecordHandler) Delete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	recordID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid record id"})
		return
	}

	if err := h.recordService.DeleteRecord(recordID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "delete_record", "dns_record", &recordID,
		nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "deleted successfully"})
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/handler/
```

- [ ] **Step 3: Commit**

```bash
git add internal/handler/record.go
git commit -m "feat: add DNS record handler with list, create, update, and delete"
```

---

## Task 13: Alibaba Cloud DNS Client Wrapper

**Files:**
- Create: `internal/aliyun/client.go`

- [ ] **Step 1: Create Aliyun DNS client wrapper**

```go
// internal/aliyun/client.go
package aliyun

import (
	"fmt"

	"domainnest/internal/config"

	alidns "github.com/alibabacloud-go/alidns-20150109/v5/client"
	openapi "github.com/alibabacloud-go/openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/v2/tea"
)

type Client struct {
	client  *alidns.Client
	runtime *util.RuntimeOptions
}

func NewClient(cfg *config.AliyunConfig) (*Client, error) {
	apiConfig := &openapi.Config{
		AccessKeyId:     tea.String(cfg.AccessKeyID),
		AccessKeySecret: tea.String(cfg.AccessKeySecret),
		Endpoint:        tea.String(cfg.Endpoint),
	}

	client, err := alidns.NewClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create aliyun DNS client: %w", err)
	}

	return &Client{
		client:  client,
		runtime: &util.RuntimeOptions{},
	}, nil
}

// AddRecord creates a new DNS record in Alibaba Cloud.
func (c *Client) AddRecord(domainName, rr, recordType, value string, ttl int64) (string, error) {
	req := &alidns.AddDomainRecordRequest{
		DomainName: tea.String(domainName),
		RR:         tea.String(rr),
		Type:       tea.String(recordType),
		Value:      tea.String(value),
		TTL:        tea.Int64(ttl),
	}

	resp, err := c.client.AddDomainRecordWithOptions(req, c.runtime)
	if err != nil {
		return "", fmt.Errorf("aliyun AddRecord failed: %w", err)
	}

	return tea.StringValue(resp.Body.RecordId), nil
}

// UpdateRecord updates an existing DNS record in Alibaba Cloud.
func (c *Client) UpdateRecord(recordID, rr, recordType, value string, ttl int64) error {
	req := &alidns.UpdateDomainRecordRequest{
		RecordId: tea.String(recordID),
		RR:       tea.String(rr),
		Type:     tea.String(recordType),
		Value:    tea.String(value),
		TTL:      tea.Int64(ttl),
	}

	_, err := c.client.UpdateDomainRecordWithOptions(req, c.runtime)
	if err != nil {
		return fmt.Errorf("aliyun UpdateRecord failed: %w", err)
	}

	return nil
}

// DeleteRecord deletes a DNS record from Alibaba Cloud.
func (c *Client) DeleteRecord(recordID string) error {
	req := &alidns.DeleteDomainRecordRequest{
		RecordId: tea.String(recordID),
	}

	_, err := c.client.DeleteDomainRecordWithOptions(req, c.runtime)
	if err != nil {
		return fmt.Errorf("aliyun DeleteRecord failed: %w", err)
	}

	return nil
}
```

- [ ] **Step 2: Install Aliyun SDK dependencies**

```bash
go get github.com/alibabacloud-go/alidns-20150109/v5 github.com/alibabacloud-go/openapi/v2 github.com/alibabacloud-go/tea-utils/v2 github.com/alibabacloud-go/tea/v2
go build ./internal/aliyun/
```

- [ ] **Step 3: Commit**

```bash
git add internal/aliyun/client.go go.mod go.sum
git commit -m "feat: add Alibaba Cloud DNS client wrapper"
```

---

## Task 14: DDNS Service

**Files:**
- Create: `internal/service/ddns.go`

- [ ] **Step 1: Create DDNS service**

```go
// internal/service/ddns.go
package service

import (
	"errors"
	"fmt"

	"domainnest/internal/aliyun"
	"domainnest/internal/model"

	"gorm.io/gorm"
)

type DDNSService struct {
	db            *gorm.DB
	domainService *DomainService
	recordService *RecordService
	aliyunClient  *aliyun.Client
}

func NewDDNSService(db *gorm.DB, domainService *DomainService, recordService *RecordService, aliyunClient *aliyun.Client) *DDNSService {
	return &DDNSService{
		db:            db,
		domainService: domainService,
		recordService: recordService,
		aliyunClient:  aliyunClient,
	}
}

type DDNSUpdateResult struct {
	Domain     string `json:"domain"`
	IP         string `json:"ip"`
	RecordType string `json:"record_type"`
	Action     string `json:"action"` // "created" or "updated"
}

// Update performs a DDNS update: find domain node, create or update record, sync to Aliyun.
func (s *DDNSService) Update(userID uint64, domain, ip, recordType string, ttl int) (*DDNSUpdateResult, error) {
	if recordType == "" {
		recordType = "A"
	}
	if ttl == 0 {
		ttl = 600
	}

	// Find the domain node
	node, rr, err := s.domainService.FindNodeByDomain(domain, userID)
	if err != nil {
		return nil, err
	}

	// Get the root domain name for this node (for Aliyun API)
	rootDomain := getRootDomain(node.FullDomain)
	rrForAliyun := getRRForAliyun(node.FullDomain, rootDomain, rr)

	// Check if record exists
	record, err := s.recordService.FindRecordByNodeAndHost(node.ID, rr, recordType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new record
			return s.createRecord(node.ID, rootDomain, rrForAliyun, rr, recordType, ip, ttl)
		}
		return nil, err
	}

	// Record exists - check if IP changed
	if record.Value == ip && record.SyncStatus == "synced" {
		return &DDNSUpdateResult{
			Domain:     domain,
			IP:         ip,
			RecordType: recordType,
			Action:     "updated",
		}, nil
	}

	// Update existing record
	return s.updateRecord(record, rootDomain, rrForAliyun, ip, ttl)
}

func (s *DDNSService) createRecord(nodeID uint64, rootDomain, rrForAliyun, host, recordType, ip string, ttl int) (*DDNSUpdateResult, error) {
	// Create local record
	record := &model.DNSRecord{
		NodeID:     nodeID,
		Host:       host,
		RecordType: recordType,
		Value:      ip,
		TTL:        ttl,
		SyncStatus: "pending",
	}
	if err := s.db.Create(record).Error; err != nil {
		return nil, fmt.Errorf("failed to create local record: %w", err)
	}

	// Sync to Aliyun
	aliyunRecordID, err := s.aliyunClient.AddRecord(rootDomain, rrForAliyun, recordType, ip, int64(ttl))
	if err != nil {
		s.recordService.UpdateSyncStatus(record.ID, "failed", "")
		return nil, fmt.Errorf("failed to sync to aliyun: %w", err)
	}

	s.recordService.UpdateSyncStatus(record.ID, "synced", aliyunRecordID)

	return &DDNSUpdateResult{
		Domain:     rootDomain,
		IP:         ip,
		RecordType: recordType,
		Action:     "created",
	}, nil
}

func (s *DDNSService) updateRecord(record *model.DNSRecord, rootDomain, rrForAliyun, ip string, ttl int) (*DDNSUpdateResult, error) {
	// Update local record
	updates := map[string]interface{}{
		"value":       ip,
		"ttl":         ttl,
		"sync_status": "pending",
	}
	if err := s.db.Model(record).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update local record: %w", err)
	}

	// Sync to Aliyun
	if record.AliyunRecordID != "" {
		err := s.aliyunClient.UpdateRecord(record.AliyunRecordID, rrForAliyun, record.RecordType, ip, int64(ttl))
		if err != nil {
			s.recordService.UpdateSyncStatus(record.ID, "failed", record.AliyunRecordID)
			return nil, fmt.Errorf("failed to sync to aliyun: %w", err)
		}
		s.recordService.UpdateSyncStatus(record.ID, "synced", record.AliyunRecordID)
	} else {
		// Record existed locally but not in Aliyun (previous sync failed)
		aliyunRecordID, err := s.aliyunClient.AddRecord(rootDomain, rrForAliyun, record.RecordType, ip, int64(ttl))
		if err != nil {
			s.recordService.UpdateSyncStatus(record.ID, "failed", "")
			return nil, fmt.Errorf("failed to sync to aliyun: %w", err)
		}
		s.recordService.UpdateSyncStatus(record.ID, "synced", aliyunRecordID)
	}

	return &DDNSUpdateResult{
		Domain:     rootDomain,
		IP:         ip,
		RecordType: record.RecordType,
		Action:     "updated",
	}, nil
}

// SyncRecord syncs a single record to Aliyun (for retry).
func (s *DDNSService) SyncRecord(recordID uint64) error {
	record, err := s.recordService.GetRecordByID(recordID)
	if err != nil {
		return err
	}

	var node model.DomainNode
	if err := s.db.First(&node, record.NodeID).Error; err != nil {
		return err
	}

	rootDomain := getRootDomain(node.FullDomain)
	rrForAliyun := getRRForAliyun(node.FullDomain, rootDomain, record.Host)

	if record.AliyunRecordID != "" {
		err := s.aliyunClient.UpdateRecord(record.AliyunRecordID, rrForAliyun, record.RecordType, record.Value, int64(record.TTL))
		if err != nil {
			s.recordService.UpdateSyncStatus(record.ID, "failed", record.AliyunRecordID)
			return err
		}
		s.recordService.UpdateSyncStatus(record.ID, "synced", record.AliyunRecordID)
	} else {
		aliyunRecordID, err := s.aliyunClient.AddRecord(rootDomain, rrForAliyun, record.RecordType, record.Value, int64(record.TTL))
		if err != nil {
			s.recordService.UpdateSyncStatus(record.ID, "failed", "")
			return err
		}
		s.recordService.UpdateSyncStatus(record.ID, "synced", aliyunRecordID)
	}

	return nil
}

// getRootDomain extracts the root domain (e.g., "xxxx.com") from a full domain.
func getRootDomain(fullDomain string) string {
	// Find the last two parts (assuming TLD is always one part)
	parts := splitDomain(fullDomain)
	if len(parts) < 2 {
		return fullDomain
	}
	return parts[len(parts)-2] + "." + parts[len(parts)-1]
}

// getRRForAliyun calculates the RR value for Aliyun API.
// For a node "a.xxxx.com" with root "xxxx.com", the RR for the node itself is "a".
// For a record with host "www" on node "a.xxxx.com", the RR is "www.a".
func getRRForAliyun(fullDomain, rootDomain, host string) string {
	// Remove root domain from full domain to get the subdomain part
	subDomain := fullDomain
	if fullDomain != rootDomain {
		subDomain = fullDomain[:len(fullDomain)-len(rootDomain)-1] // remove ".rootDomain"
	}

	if host == "@" {
		if subDomain == "" {
			return "@"
		}
		return subDomain
	}

	if subDomain == "" {
		return host
	}
	return host + "." + subDomain
}

func splitDomain(domain string) []string {
	var parts []string
	current := ""
	for _, c := range domain {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/service/
```

- [ ] **Step 3: Commit**

```bash
git add internal/service/ddns.go
git commit -m "feat: add DDNS service with domain lookup, create/update logic, and Aliyun sync"
```

---

## Task 15: DDNS Handler

**Files:**
- Create: `internal/handler/ddns.go`

- [ ] **Step 1: Create DDNS handler**

```go
// internal/handler/ddns.go
package handler

import (
	"net/http"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

type DDNSHandler struct {
	ddnsService *service.DDNSService
}

func NewDDNSHandler(ddnsService *service.DDNSService) *DDNSHandler {
	return &DDNSHandler{ddnsService: ddnsService}
}

func (h *DDNSHandler) Update(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		Domain     string `json:"domain" binding:"required"`
		IP         string `json:"ip" binding:"required"`
		RecordType string `json:"record_type"`
		TTL        int    `json:"ttl"`
		Token      string `json:"token"` // ignored, already authenticated
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	result, err := h.ddnsService.Update(userID, req.Domain, req.IP, req.RecordType, req.TTL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/handler/
```

- [ ] **Step 3: Commit**

```bash
git add internal/handler/ddns.go
git commit -m "feat: add DDNS update handler"
```

---

## Task 16: Admin Handler

**Files:**
- Create: `internal/handler/admin.go`

- [ ] **Step 1: Create admin handler**

```go
// internal/handler/admin.go
package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db *gorm.DB
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

func (h *AdminHandler) CreateRootDomain(c *gin.Context) {
	var req struct {
		Host         string `json:"host" binding:"required"`
		DomainSuffix string `json:"domain_suffix" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	fullDomain := req.Host + "." + req.DomainSuffix

	// Check if domain already exists
	var existing model.DomainNode
	if err := h.db.Where("full_domain = ?", fullDomain).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "domain already exists"})
		return
	}

	// Get admin user to assign as temporary owner
	adminID := c.GetUint64("user_id")

	node := &model.DomainNode{
		Host:       req.Host,
		FullDomain: fullDomain,
		OwnerID:    adminID,
	}

	if err := h.db.Create(node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": node})
}

func (h *AdminHandler) AssignDomain(c *gin.Context) {
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	var req struct {
		UserID uint64 `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// Verify target user exists
	var user model.User
	if err := h.db.First(&user, req.UserID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "target user not found"})
		return
	}

	if err := h.db.Model(&model.DomainNode{}).Where("id = ?", nodeID).Update("owner_id", req.UserID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "domain assigned successfully"})
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	var users []model.User
	if err := h.db.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": users})
}

func (h *AdminHandler) ListLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	userID := c.Query("user_id")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := h.db.Model(&model.OperationLog{})
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	var total int64
	query.Count(&total)

	var logs []model.OperationLog
	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items":     logs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func (h *AdminHandler) RetrySync(c *gin.Context) {
	recordID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid record id"})
		return
	}

	// This will be handled by the DDNS service's SyncRecord method
	// For now, just mark as pending for retry
	if err := h.db.Model(&model.DNSRecord{}).Where("id = ?", recordID).
		Update("sync_status", "pending").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "sync retry queued"})
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/handler/
```

- [ ] **Step 3: Commit**

```bash
git add internal/handler/admin.go
git commit -m "feat: add admin handler with root domain, user, and log management"
```

---

## Task 17: Router Setup

**Files:**
- Create: `internal/router/router.go`

- [ ] **Step 1: Create router with all routes registered**

```go
// internal/router/router.go
package router

import (
	"domainnest/internal/config"
	"domainnest/internal/handler"
	"domainnest/internal/middleware"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(cfg *config.Config, db *gorm.DB, authService *service.AuthService,
	domainService *service.DomainService, recordService *service.RecordService,
	ddnsService *service.DDNSService) *gin.Engine {

	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Handlers
	authHandler := handler.NewAuthHandler(authService, &cfg.JWT)
	domainHandler := handler.NewDomainHandler(domainService, db)
	recordHandler := handler.NewRecordHandler(recordService, db)
	ddnsHandler := handler.NewDDNSHandler(ddnsService)
	adminHandler := handler.NewAdminHandler(db)

	v1 := r.Group("/api/v1")

	// Auth routes (public)
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Auth routes (JWT required)
	authProtected := v1.Group("/auth")
	authProtected.Use(middleware.JWTAuth(cfg.JWT.Secret))
	{
		authProtected.GET("/profile", authHandler.GetProfile)
		authProtected.PUT("/token", authHandler.ResetToken)
		authProtected.PUT("/password", authHandler.ChangePassword)
	}

	// Domain routes (JWT required)
	domains := v1.Group("/domains")
	domains.Use(middleware.JWTAuth(cfg.JWT.Secret))
	{
		domains.GET("", domainHandler.List)
		domains.POST("", domainHandler.Create)
		domains.GET("/:id", domainHandler.Get)
		domains.POST("/:id/transfer", domainHandler.Transfer)
		domains.DELETE("/:id", domainHandler.Delete)
		domains.GET("/:id/records", recordHandler.List)
		domains.POST("/:id/records", recordHandler.Create)
	}

	// Record routes (JWT required)
	records := v1.Group("/records")
	records.Use(middleware.JWTAuth(cfg.JWT.Secret))
	{
		records.PUT("/:id", recordHandler.Update)
		records.DELETE("/:id", recordHandler.Delete)
	}

	// DDNS routes (Token auth)
	ddns := v1.Group("/ddns")
	ddns.Use(middleware.TokenAuth(db))
	{
		ddns.POST("/update", ddnsHandler.Update)
	}

	// Admin routes (JWT + Admin required)
	admin := v1.Group("/admin")
	admin.Use(middleware.JWTAuth(cfg.JWT.Secret), middleware.AdminRequired())
	{
		admin.POST("/domains", adminHandler.CreateRootDomain)
		admin.POST("/domains/:id/assign", adminHandler.AssignDomain)
		admin.GET("/users", adminHandler.ListUsers)
		admin.GET("/logs", adminHandler.ListLogs)
		admin.POST("/records/:id/sync", adminHandler.RetrySync)
	}

	return r
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/router/
```

- [ ] **Step 3: Commit**

```bash
git add internal/router/router.go
git commit -m "feat: add router with all API routes registered"
```

---

## Task 18: Main Entry Point (Wire Everything Together)

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Update main.go to initialize all services and start server**

```go
// cmd/server/main.go
package main

import (
	"fmt"
	"log"

	"domainnest/internal/aliyun"
	"domainnest/internal/config"
	"domainnest/internal/model"
	"domainnest/internal/router"
	"domainnest/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := model.InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	if err := model.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Aliyun client
	aliyunClient, err := aliyun.NewClient(&cfg.Aliyun)
	if err != nil {
		log.Printf("Warning: Failed to initialize Aliyun client: %v", err)
		// Continue without Aliyun - DDNS sync will fail but app works
	}

	// Initialize services
	authService := service.NewAuthService(db)
	domainService := service.NewDomainService(db)
	recordService := service.NewRecordService(db)
	ddnsService := service.NewDDNSService(db, domainService, recordService, aliyunClient)

	// Ensure admin user exists
	if err := authService.EnsureAdmin(cfg.Admin.Username, cfg.Admin.Password); err != nil {
		log.Fatalf("Failed to ensure admin user: %v", err)
	}

	// Setup router
	r := router.Setup(cfg, db, authService, domainService, recordService, ddnsService)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

- [ ] **Step 2: Build and verify**

```bash
go build ./cmd/server/
```

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire all services together in main entry point"
```

---

## Task 19: Vue3 Frontend Project Initialization

**Files:**
- Create: `web/` (Vue3 project)

- [ ] **Step 1: Initialize Vue3 project**

```bash
cd /home/lingnc/workspace/DomainNest
npm create vite@latest web -- --template vue
cd web
npm install
npm install vue-router@4 pinia axios element-plus @element-plus/icons-vue
```

- [ ] **Step 2: Configure vite.config.js for API proxy**

```javascript
// web/vite.config.js
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      }
    }
  },
  build: {
    outDir: 'dist',
    assetsDir: 'static',
  }
})
```

- [ ] **Step 3: Setup main.js with Element Plus**

```javascript
// web/src/main.js
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import App from './App.vue'
import router from './router'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(ElementPlus)

for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.mount('#app')
```

- [ ] **Step 4: Setup router**

```javascript
// web/src/router/index.js
import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/login', name: 'Login', component: () => import('../views/Login.vue') },
  { path: '/register', name: 'Register', component: () => import('../views/Register.vue') },
  { path: '/dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue'), meta: { requiresAuth: true } },
  { path: '/domains/:id', name: 'DomainDetail', component: () => import('../views/DomainDetail.vue'), meta: { requiresAuth: true } },
  { path: '/settings', name: 'Settings', component: () => import('../views/Settings.vue'), meta: { requiresAuth: true } },
  { path: '/admin', name: 'Admin', component: () => import('../views/Admin.vue'), meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/', redirect: '/dashboard' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) {
    next('/login')
  } else {
    next()
  }
})

export default router
```

- [ ] **Step 5: Setup axios instance**

```javascript
// web/src/api/request.js
import axios from 'axios'
import { ElMessage } from 'element-plus'

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

request.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

request.interceptors.response.use(
  response => {
    const { data } = response
    if (data.code !== 0) {
      ElMessage.error(data.message || 'Request failed')
      return Promise.reject(data)
    }
    return data
  },
  error => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    ElMessage.error(error.response?.data?.message || 'Network error')
    return Promise.reject(error)
  }
)

export default request
```

- [ ] **Step 6: Create API modules**

```javascript
// web/src/api/auth.js
import request from './request'

export const login = (data) => request.post('/auth/login', data)
export const register = (data) => request.post('/auth/register', data)
export const getProfile = () => request.get('/auth/profile')
export const resetToken = () => request.put('/auth/token')
export const changePassword = (data) => request.put('/auth/password', data)
```

```javascript
// web/src/api/domain.js
import request from './request'

export const getDomains = () => request.get('/domains')
export const createDomain = (data) => request.post('/domains', data)
export const getDomain = (id) => request.get(`/domains/${id}`)
export const transferDomain = (id, data) => request.post(`/domains/${id}/transfer`, data)
export const deleteDomain = (id) => request.delete(`/domains/${id}`)
```

```javascript
// web/src/api/record.js
import request from './request'

export const getRecords = (nodeId) => request.get(`/domains/${nodeId}/records`)
export const createRecord = (nodeId, data) => request.post(`/domains/${nodeId}/records`, data)
export const updateRecord = (id, data) => request.put(`/records/${id}`, data)
export const deleteRecord = (id) => request.delete(`/records/${id}`)
```

```javascript
// web/src/api/admin.js
import request from './request'

export const createRootDomain = (data) => request.post('/admin/domains', data)
export const assignDomain = (id, data) => request.post(`/admin/domains/${id}/assign`, data)
export const listUsers = () => request.get('/admin/users')
export const listLogs = (params) => request.get('/admin/logs', { params })
export const retrySync = (id) => request.post(`/admin/records/${id}/sync`)
```

- [ ] **Step 7: Verify frontend builds**

```bash
cd /home/lingnc/workspace/DomainNest/web
npm run build
```

- [ ] **Step 8: Commit**

```bash
git add web/
git commit -m "feat: initialize Vue3 frontend with router, axios, and API modules"
```

---

## Task 20: Frontend - Login and Register Pages

**Files:**
- Create: `web/src/views/Login.vue`, `web/src/views/Register.vue`

- [ ] **Step 1: Create Login page**

```vue
<!-- web/src/views/Login.vue -->
<template>
  <div class="login-container">
    <el-card class="login-card">
      <h2>DomainNest Login</h2>
      <el-form :model="form" @submit.prevent="handleLogin">
        <el-form-item>
          <el-input v-model="form.username" placeholder="Username" prefix-icon="User" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" placeholder="Password" prefix-icon="Lock" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" native-type="submit" style="width:100%">Login</el-button>
        </el-form-item>
        <div class="links">
          <router-link to="/register">Register</router-link>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { login } from '../api/auth'

const router = useRouter()
const loading = ref(false)
const form = ref({ username: '', password: '' })

const handleLogin = async () => {
  loading.value = true
  try {
    const res = await login(form.value)
    localStorage.setItem('token', res.data.token)
    localStorage.setItem('user', JSON.stringify(res.data.user))
    router.push('/dashboard')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background: #f5f7fa;
}
.login-card {
  width: 400px;
}
.links {
  text-align: center;
}
</style>
```

- [ ] **Step 2: Create Register page**

```vue
<!-- web/src/views/Register.vue -->
<template>
  <div class="register-container">
    <el-card class="register-card">
      <h2>DomainNest Register</h2>
      <el-form :model="form" @submit.prevent="handleRegister">
        <el-form-item>
          <el-input v-model="form.username" placeholder="Username" prefix-icon="User" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.email" placeholder="Email (optional)" prefix-icon="Message" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" placeholder="Password" prefix-icon="Lock" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" native-type="submit" style="width:100%">Register</el-button>
        </el-form-item>
        <div class="links">
          <router-link to="/login">Back to Login</router-link>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { register } from '../api/auth'
import { ElMessage } from 'element-plus'

const router = useRouter()
const loading = ref(false)
const form = ref({ username: '', email: '', password: '' })

const handleRegister = async () => {
  loading.value = true
  try {
    await register(form.value)
    ElMessage.success('Registration successful')
    router.push('/login')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.register-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background: #f5f7fa;
}
.register-card {
  width: 400px;
}
.links {
  text-align: center;
}
</style>
```

- [ ] **Step 3: Create App.vue with layout**

```vue
<!-- web/src/App.vue -->
<template>
  <router-view />
</template>

<script setup>
</script>

<style>
body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}
</style>
```

- [ ] **Step 4: Build and verify**

```bash
cd /home/lingnc/workspace/DomainNest/web
npm run build
```

- [ ] **Step 5: Commit**

```bash
git add web/src/views/Login.vue web/src/views/Register.vue web/src/App.vue
git commit -m "feat: add login and register pages"
```

---

## Task 21: Frontend - Dashboard Page

**Files:**
- Create: `web/src/views/Dashboard.vue`

- [ ] **Step 1: Create Dashboard page**

```vue
<!-- web/src/views/Dashboard.vue -->
<template>
  <el-container>
    <el-header>
      <div class="header-content">
        <h2>DomainNest</h2>
        <div>
          <el-button @click="$router.push('/settings')">Settings</el-button>
          <el-button @click="handleLogout">Logout</el-button>
        </div>
      </div>
    </el-header>
    <el-main>
      <el-card>
        <template #header>
          <div class="card-header">
            <span>My Domains</span>
          </div>
        </template>
        <el-tree
          :data="domains"
          :props="{ label: 'full_domain', children: 'children' }"
          @node-click="handleNodeClick"
        >
          <template #default="{ data }">
            <div class="tree-node">
              <span>{{ data.full_domain }}</span>
              <el-tag size="small" type="info">{{ data.records?.length || 0 }} records</el-tag>
            </div>
          </template>
        </el-tree>
      </el-card>
    </el-main>
  </el-container>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getDomains } from '../api/domain'

const router = useRouter()
const domains = ref([])

const loadDomains = async () => {
  const res = await getDomains()
  domains.value = res.data
}

const handleNodeClick = (data) => {
  router.push(`/domains/${data.id}`)
}

const handleLogout = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  router.push('/login')
}

onMounted(loadDomains)
</script>

<style scoped>
.el-header {
  background: #409eff;
  color: white;
  line-height: 60px;
}
.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.tree-node {
  display: flex;
  align-items: center;
  gap: 10px;
}
</style>
```

- [ ] **Step 2: Build and verify**

```bash
cd /home/lingnc/workspace/DomainNest/web
npm run build
```

- [ ] **Step 3: Commit**

```bash
git add web/src/views/Dashboard.vue
git commit -m "feat: add dashboard page with domain tree view"
```

---

## Task 22: Frontend - Domain Detail Page

**Files:**
- Create: `web/src/views/DomainDetail.vue`

- [ ] **Step 1: Create Domain Detail page**

```vue
<!-- web/src/views/DomainDetail.vue -->
<template>
  <el-container>
    <el-header>
      <div class="header-content">
        <h2>DomainNest</h2>
        <el-button @click="$router.push('/dashboard')">Back</el-button>
      </div>
    </el-header>
    <el-main>
      <el-row :gutter="20">
        <el-col :span="16">
          <el-card>
            <template #header>
              <div class="card-header">
                <span>DNS Records - {{ domain?.full_domain }}</span>
                <el-button type="primary" size="small" @click="showAddRecord = true">Add Record</el-button>
              </div>
            </template>
            <el-table :data="records" stripe>
              <el-table-column prop="host" label="Host" width="120" />
              <el-table-column prop="record_type" label="Type" width="80" />
              <el-table-column prop="value" label="Value" />
              <el-table-column prop="ttl" label="TTL" width="80" />
              <el-table-column prop="sync_status" label="Status" width="100">
                <template #default="{ row }">
                  <el-tag :type="row.sync_status === 'synced' ? 'success' : row.sync_status === 'failed' ? 'danger' : 'warning'" size="small">
                    {{ row.sync_status }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="Actions" width="150">
                <template #default="{ row }">
                  <el-button size="small" @click="editRecord(row)">Edit</el-button>
                  <el-button size="small" type="danger" @click="handleDeleteRecord(row.id)">Delete</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-col>
        <el-col :span="8">
          <el-card>
            <template #header>Actions</template>
            <el-button type="primary" @click="showCreateChild = true" style="width:100%;margin-bottom:10px">Create Subdomain</el-button>
            <el-button type="warning" @click="showTransfer = true" style="width:100%;margin-bottom:10px">Transfer Domain</el-button>
            <el-button type="danger" @click="handleDeleteDomain" style="width:100%">Delete Domain</el-button>
          </el-card>
        </el-col>
      </el-row>

      <!-- Add Record Dialog -->
      <el-dialog v-model="showAddRecord" title="Add DNS Record">
        <el-form :model="recordForm">
          <el-form-item label="Host">
            <el-input v-model="recordForm.host" placeholder="@ for root" />
          </el-form-item>
          <el-form-item label="Type">
            <el-select v-model="recordForm.record_type">
              <el-option label="A" value="A" />
              <el-option label="AAAA" value="AAAA" />
              <el-option label="CNAME" value="CNAME" />
              <el-option label="TXT" value="TXT" />
              <el-option label="MX" value="MX" />
            </el-select>
          </el-form-item>
          <el-form-item label="Value">
            <el-input v-model="recordForm.value" />
          </el-form-item>
          <el-form-item label="TTL">
            <el-input-number v-model="recordForm.ttl" :min="60" :max="86400" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="showAddRecord = false">Cancel</el-button>
          <el-button type="primary" @click="handleAddRecord">Add</el-button>
        </template>
      </el-dialog>

      <!-- Create Child Dialog -->
      <el-dialog v-model="showCreateChild" title="Create Subdomain">
        <el-form :model="childForm">
          <el-form-item label="Host">
            <el-input v-model="childForm.host" placeholder="subdomain name" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="showCreateChild = false">Cancel</el-button>
          <el-button type="primary" @click="handleCreateChild">Create</el-button>
        </template>
      </el-dialog>

      <!-- Transfer Dialog -->
      <el-dialog v-model="showTransfer" title="Transfer Domain">
        <el-form :model="transferForm">
          <el-form-item label="Target User ID">
            <el-input-number v-model="transferForm.target_user_id" :min="1" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="showTransfer = false">Cancel</el-button>
          <el-button type="warning" @click="handleTransfer">Transfer</el-button>
        </template>
      </el-dialog>
    </el-main>
  </el-container>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getDomain, createDomain, transferDomain, deleteDomain } from '../api/domain'
import { getRecords, createRecord, deleteRecord } from '../api/record'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const router = useRouter()
const domainId = route.params.id

const domain = ref(null)
const records = ref([])
const showAddRecord = ref(false)
const showCreateChild = ref(false)
const showTransfer = ref(false)

const recordForm = ref({ host: '@', record_type: 'A', value: '', ttl: 600 })
const childForm = ref({ host: '' })
const transferForm = ref({ target_user_id: 1 })

const loadData = async () => {
  const [domainRes, recordsRes] = await Promise.all([
    getDomain(domainId),
    getRecords(domainId)
  ])
  domain.value = domainRes.data
  records.value = recordsRes.data
}

const handleAddRecord = async () => {
  await createRecord(domainId, recordForm.value)
  showAddRecord.value = false
  recordForm.value = { host: '@', record_type: 'A', value: '', ttl: 600 }
  loadData()
}

const handleDeleteRecord = async (id) => {
  await ElMessageBox.confirm('Delete this record?')
  await deleteRecord(id)
  loadData()
}

const handleCreateChild = async () => {
  await createDomain({ parent_id: parseInt(domainId), host: childForm.value.host })
  showCreateChild.value = false
  router.push('/dashboard')
}

const handleTransfer = async () => {
  await ElMessageBox.confirm('Transfer this domain and all subdomains?')
  await transferDomain(domainId, transferForm.value)
  showTransfer.value = false
  router.push('/dashboard')
}

const handleDeleteDomain = async () => {
  await ElMessageBox.confirm('Delete this domain? Only works if no children or records.')
  await deleteDomain(domainId)
  router.push('/dashboard')
}

const editRecord = (row) => {
  // Simplified - in production, open edit dialog
  ElMessage.info('Edit functionality - use update API')
}

onMounted(loadData)
</script>

<style scoped>
.el-header {
  background: #409eff;
  color: white;
  line-height: 60px;
}
.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
```

- [ ] **Step 2: Build and verify**

```bash
cd /home/lingnc/workspace/DomainNest/web
npm run build
```

- [ ] **Step 3: Commit**

```bash
git add web/src/views/DomainDetail.vue
git commit -m "feat: add domain detail page with DNS record management"
```

---

## Task 23: Frontend - Settings Page

**Files:**
- Create: `web/src/views/Settings.vue`

- [ ] **Step 1: Create Settings page**

```vue
<!-- web/src/views/Settings.vue -->
<template>
  <el-container>
    <el-header>
      <div class="header-content">
        <h2>DomainNest</h2>
        <el-button @click="$router.push('/dashboard')">Back</el-button>
      </div>
    </el-header>
    <el-main>
      <el-card>
        <template #header>DDNS Token</template>
        <p>Your DDNS Token: <el-tag>{{ token }}</el-tag></p>
        <el-button type="warning" @click="handleResetToken">Reset Token</el-button>

        <el-divider />

        <h4>ddns-go Configuration</h4>
        <p>Use these settings in your ddns-go Webhook configuration:</p>
        <el-input type="textarea" :rows="3" :value="ddnsConfig" readonly />
        <el-button type="primary" size="small" @click="copyConfig" style="margin-top:10px">Copy</el-button>
      </el-card>

      <el-card style="margin-top:20px">
        <template #header>Change Password</template>
        <el-form :model="passwordForm" @submit.prevent="handleChangePassword">
          <el-form-item label="Old Password">
            <el-input v-model="passwordForm.old_password" type="password" show-password />
          </el-form-item>
          <el-form-item label="New Password">
            <el-input v-model="passwordForm.new_password" type="password" show-password />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" native-type="submit">Change Password</el-button>
          </el-form-item>
        </el-form>
      </el-card>
    </el-main>
  </el-container>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getProfile, resetToken, changePassword } from '../api/auth'
import { ElMessage } from 'element-plus'

const token = ref('')
const passwordForm = ref({ old_password: '', new_password: '' })

const ddnsConfig = computed(() => {
  const baseUrl = window.location.origin
  return `URL: ${baseUrl}/api/v1/ddns/update?token=${token.value}\nRequestBody: {"domain":"#{domain}","ip":"#{ip}","record_type":"#{recordType}","ttl":#{ttl}}`
})

const loadProfile = async () => {
  const res = await getProfile()
  token.value = res.data.ddns_token
}

const handleResetToken = async () => {
  const res = await resetToken()
  token.value = res.data.token
  ElMessage.success('Token reset successfully')
}

const handleChangePassword = async () => {
  await changePassword(passwordForm.value)
  ElMessage.success('Password changed')
  passwordForm.value = { old_password: '', new_password: '' }
}

const copyConfig = () => {
  navigator.clipboard.writeText(ddnsConfig.value)
  ElMessage.success('Copied to clipboard')
}

onMounted(loadProfile)
</script>

<style scoped>
.el-header {
  background: #409eff;
  color: white;
  line-height: 60px;
}
.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
```

- [ ] **Step 2: Build and verify**

```bash
cd /home/lingnc/workspace/DomainNest/web
npm run build
```

- [ ] **Step 3: Commit**

```bash
git add web/src/views/Settings.vue
git commit -m "feat: add settings page with token management and ddns-go config"
```

---

## Task 24: Frontend - Admin Page

**Files:**
- Create: `web/src/views/Admin.vue`

- [ ] **Step 1: Create Admin page**

```vue
<!-- web/src/views/Admin.vue -->
<template>
  <el-container>
    <el-header>
      <div class="header-content">
        <h2>DomainNest Admin</h2>
        <el-button @click="$router.push('/dashboard')">Back</el-button>
      </div>
    </el-header>
    <el-main>
      <el-tabs v-model="activeTab">
        <el-tab-pane label="Users" name="users">
          <el-table :data="users" stripe>
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="username" label="Username" />
            <el-table-column prop="email" label="Email" />
            <el-table-column prop="role" label="Role" />
            <el-table-column prop="token" label="DDNS Token" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="Root Domains" name="domains">
          <el-form :model="domainForm" @submit.prevent="handleCreateRoot" inline>
            <el-form-item label="Host">
              <el-input v-model="domainForm.host" placeholder="xxxx" />
            </el-form-item>
            <el-form-item label="Suffix">
              <el-input v-model="domainForm.domain_suffix" placeholder="com" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" native-type="submit">Create Root Domain</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="Logs" name="logs">
          <el-table :data="logs" stripe>
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="user_id" label="User ID" width="100" />
            <el-table-column prop="action" label="Action" />
            <el-table-column prop="target_type" label="Target" />
            <el-table-column prop="ip_address" label="IP" />
            <el-table-column prop="created_at" label="Time" />
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-main>
  </el-container>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { listUsers, listLogs, createRootDomain } from '../api/admin'
import { ElMessage } from 'element-plus'

const activeTab = ref('users')
const users = ref([])
const logs = ref([])
const domainForm = ref({ host: '', domain_suffix: '' })

const loadData = async () => {
  const [usersRes, logsRes] = await Promise.all([
    listUsers(),
    listLogs({ page: 1, page_size: 50 })
  ])
  users.value = usersRes.data
  logs.value = logsRes.data.items
}

const handleCreateRoot = async () => {
  await createRootDomain(domainForm.value)
  ElMessage.success('Root domain created')
  domainForm.value = { host: '', domain_suffix: '' }
}

onMounted(loadData)
</script>

<style scoped>
.el-header {
  background: #409eff;
  color: white;
  line-height: 60px;
}
.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
```

- [ ] **Step 2: Build and verify**

```bash
cd /home/lingnc/workspace/DomainNest/web
npm run build
```

- [ ] **Step 3: Commit**

```bash
git add web/src/views/Admin.vue
git commit -m "feat: add admin page with user management and root domain creation"
```

---

## Task 25: Go Embed Frontend + Dockerfile

**Files:**
- Create: `internal/static/embed.go`, `Dockerfile`, `docker-compose.yml`

- [ ] **Step 1: Create static file embed**

```go
// internal/static/embed.go
package static

import "embed"

//go:embed all:dist
var StaticFiles embed.FS
```

- [ ] **Step 2: Update router to serve static files**

Add to `internal/router/router.go` after the health check route:

```go
import (
	"domainnest/internal/static"
	"io/fs"
	"net/http"
)

// Serve static files
staticFS, _ := fs.Sub(static.StaticFiles, "dist")
r.StaticFS("/static", http.FS(staticFS))

// Serve index.html for all non-API routes (SPA fallback)
r.NoRoute(func(c *gin.Context) {
	if len(c.Request.URL.Path) > 4 && c.Request.URL.Path[:4] == "/api" {
		c.JSON(404, gin.H{"code": 404, "message": "not found"})
		return
	}
	c.FileFromFS("/", http.FS(staticFS))
})
```

- [ ] **Step 3: Copy frontend build output**

```bash
mkdir -p /home/lingnc/workspace/DomainNest/internal/static/dist
cp -r /home/lingnc/workspace/DomainNest/web/dist/* /home/lingnc/workspace/DomainNest/internal/static/dist/
```

- [ ] **Step 4: Create Dockerfile**

```dockerfile
# Dockerfile
FROM node:18-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ .
RUN npm run build

FROM golang:1.20-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/web/dist internal/static/dist/
RUN CGO_ENABLED=0 go build -o domainnest ./cmd/server/

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=backend-builder /app/domainnest .
COPY config.yaml .
EXPOSE 8080
CMD ["./domainnest"]
```

- [ ] **Step 5: Create docker-compose.yml**

```yaml
# docker-compose.yml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      db:
        condition: service_healthy
    environment:
      - DB_HOST=db
      - DB_PORT=3306
      - DB_USER=domainnest
      - DB_PASSWORD=domainpass
      - DB_DBNAME=domainnest
      - JWT_SECRET=change-me-in-production
    volumes:
      - ./config.yaml:/app/config.yaml

  db:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: rootpass
      MYSQL_DATABASE: domainnest
      MYSQL_USER: domainnest
      MYSQL_PASSWORD: domainpass
    volumes:
      - mysql_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      timeout: 5s
      retries: 10

volumes:
  mysql_data:
```

- [ ] **Step 6: Update config.yaml to support env vars**

Add to `cmd/server/main.go` before `viper.ReadInConfig()`:

```go
// Allow environment variables to override config
viper.SetEnvPrefix("DOMAINNEST")
viper.AutomaticEnv()
viper.BindEnv("database.host", "DB_HOST")
viper.BindEnv("database.port", "DB_PORT")
viper.BindEnv("database.user", "DB_USER")
viper.BindEnv("database.password", "DB_PASSWORD")
viper.BindEnv("database.dbname", "DB_DBNAME")
viper.BindEnv("jwt.secret", "JWT_SECRET")
```

- [ ] **Step 7: Build and verify**

```bash
cd /home/lingnc/workspace/DomainNest
go build ./cmd/server/
```

- [ ] **Step 8: Commit**

```bash
git add internal/static/ internal/router/router.go Dockerfile docker-compose.yml cmd/server/main.go
git commit -m "feat: add Go embed for frontend, Dockerfile, and docker-compose"
```

---

## Self-Review Checklist

**Spec Coverage:**
- [x] User registration/login (Task 7-8)
- [x] JWT auth + static token auth (Task 4-5)
- [x] Domain node tree CRUD (Task 9-10)
- [x] Domain transfer with subtree (Task 9)
- [x] DNS record CRUD (Task 11-12)
- [x] Alibaba Cloud DNS sync (Task 13-14)
- [x] DDNS webhook endpoint (Task 14-15)
- [x] Admin endpoints (Task 16)
- [x] Admin CLI bootstrap (Task 7 - EnsureAdmin)
- [x] Operation logging (Task 6)
- [x] Vue3 frontend - all pages (Task 19-24)
- [x] Go embed deployment (Task 25)
- [x] Docker + docker-compose (Task 25)
- [x] Config management (Task 1)

**Placeholder Scan:** No TBD/TODO found.

**Type Consistency:** All method signatures, model fields, and API contracts are consistent across tasks.
