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
	ddnsService *service.DDNSService) *gin.Engine {

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

	authHandler := handler.NewAuthHandler(authService, &cfg.JWT)
	domainHandler := handler.NewDomainHandler(domainService, db)
	recordHandler := handler.NewRecordHandler(recordService, db)
	ddnsHandler := handler.NewDDNSHandler(ddnsService)
	adminHandler := handler.NewAdminHandler(db)

	v1 := r.Group("/api/v1")

	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	authProtected := v1.Group("/auth")
	authProtected.Use(middleware.JWTAuth(cfg.JWT.Secret))
	{
		authProtected.GET("/profile", authHandler.GetProfile)
		authProtected.PUT("/token", authHandler.ResetToken)
		authProtected.PUT("/password", authHandler.ChangePassword)
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
	}

	records := v1.Group("/records")
	records.Use(middleware.JWTAuth(cfg.JWT.Secret))
	{
		records.PUT("/:id", recordHandler.Update)
		records.DELETE("/:id", recordHandler.Delete)
	}

	ddns := v1.Group("/ddns")
	ddns.Use(middleware.TokenAuth(db))
	{
		ddns.POST("/update", ddnsHandler.Update)
	}

	admin := v1.Group("/admin")
	admin.Use(middleware.JWTAuth(cfg.JWT.Secret), middleware.AdminRequired())
	{
		admin.POST("/domains", adminHandler.CreateRootDomain)
		admin.GET("/domains", adminHandler.ListDomains)
		admin.POST("/domains/:id/assign", adminHandler.AssignDomain)
		admin.GET("/users", adminHandler.ListUsers)
		admin.GET("/logs", adminHandler.ListLogs)
		admin.POST("/records/:id/sync", adminHandler.RetrySync)
	}

	return r
}
