package router

import (
	"domainnest/internal/config"
	"domainnest/internal/handler"
	"domainnest/internal/middleware"
	"domainnest/internal/service"
	"domainnest/internal/static"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(cfg *config.Config, db *gorm.DB, authService *service.AuthService,
	domainService *service.DomainService, recordService *service.RecordService,
	ddnsService *service.DDNSService, emailService *service.EmailService,
	settingsService *service.SettingsService, permissionService *service.PermissionService,
	ramTokenService *service.RAMTokenService) *gin.Engine {

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Serve static files from embedded frontend build
	staticFS, _ := fs.Sub(static.StaticFiles, "dist")
	r.StaticFS("/static", http.FS(staticFS))

	// SPA fallback: serve index.html for all non-API routes
	r.NoRoute(func(c *gin.Context) {
		if len(c.Request.URL.Path) > 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(404, gin.H{"code": 404, "message": "not found"})
			return
		}
		c.FileFromFS("/", http.FS(staticFS))
	})

	authHandler := handler.NewAuthHandler(authService, emailService, db, &cfg.JWT)
	domainHandler := handler.NewDomainHandler(domainService, db)
	recordHandler := handler.NewRecordHandler(recordService, db)
	ddnsHandler := handler.NewDDNSHandler(ddnsService, ramTokenService)
	adminHandler := handler.NewAdminHandler(db)
	settingsHandler := handler.NewSettingsHandler(settingsService)
	permissionHandler := handler.NewPermissionHandler(permissionService, db)
	ramTokenHandler := handler.NewRAMTokenHandler(ramTokenService, db)

	v1 := r.Group("/api/v1")

	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/reset-password", authHandler.ResetPassword)
		auth.GET("/check-username", authHandler.CheckUsername)
	}

	authProtected := v1.Group("/auth")
	authProtected.Use(middleware.JWTAuth(cfg.JWT.Secret))
	{
		authProtected.GET("/profile", authHandler.GetProfile)
		authProtected.PUT("/profile", authHandler.UpdateProfile)
		authProtected.PUT("/token", authHandler.ResetToken)
		authProtected.PUT("/password", authHandler.ChangePassword)
		authProtected.GET("/permissions", permissionHandler.MyPermissions)
	}

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
		domains.GET("/:id/records/export", recordHandler.Export)
		domains.POST("/:id/records/import", recordHandler.Import)
		domains.GET("/:id/permissions", permissionHandler.List)
		domains.POST("/:id/permissions", permissionHandler.Grant)
		domains.DELETE("/:id/permissions/:userId", permissionHandler.Revoke)
	}

	records := v1.Group("/records")
	records.Use(middleware.JWTAuth(cfg.JWT.Secret))
	{
		records.PUT("/:id", recordHandler.Update)
		records.DELETE("/:id", recordHandler.Delete)
		records.PUT("/:id/toggle", recordHandler.Toggle)
		records.POST("/batch-delete", recordHandler.BatchDelete)
		records.POST("/batch-toggle", recordHandler.BatchToggle)
	}

	ddns := v1.Group("/ddns")
	ddns.Use(middleware.TokenAuth(db))
	{
		ddns.POST("/callback", ddnsHandler.Callback)
		ddns.POST("/update", ddnsHandler.Callback) // backwards compat
		ddns.POST("/webhook", ddnsHandler.Webhook)
	}

	ramTokens := v1.Group("/ram-tokens")
	ramTokens.Use(middleware.JWTAuth(cfg.JWT.Secret))
	{
		ramTokens.GET("", ramTokenHandler.List)
		ramTokens.POST("", ramTokenHandler.Create)
		ramTokens.GET("/:id", ramTokenHandler.Get)
		ramTokens.PUT("/:id", ramTokenHandler.Update)
		ramTokens.POST("/:id/reset", ramTokenHandler.ResetToken)
		ramTokens.DELETE("/:id", ramTokenHandler.Delete)
	}

	admin := v1.Group("/admin")
	admin.Use(middleware.JWTAuth(cfg.JWT.Secret), middleware.AdminRequired())
	{
		admin.POST("/domains", adminHandler.CreateRootDomain)
		admin.GET("/domains", adminHandler.ListDomains)
		admin.POST("/domains/:id/assign", adminHandler.AssignDomain)
		admin.GET("/users", adminHandler.ListUsers)
		admin.PUT("/users/:id", adminHandler.UpdateUser)
		admin.POST("/users/:id/reset-password", adminHandler.AdminResetPassword)
		admin.DELETE("/users/:id", adminHandler.DisableUser)
		admin.POST("/users/:id/promote", adminHandler.PromoteToAdmin)
		admin.POST("/users/:id/demote", adminHandler.DemoteFromAdmin)
		admin.GET("/logs", adminHandler.ListLogs)
		admin.POST("/records/:id/sync", adminHandler.RetrySync)
		admin.GET("/settings/:category", settingsHandler.Get)
		admin.PUT("/settings/:category", settingsHandler.Set)
		admin.POST("/settings/smtp/test", settingsHandler.TestSMTP)
	}

	return r
}
