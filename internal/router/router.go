package router

import (
	"domainnest/internal/config"
	"domainnest/internal/domain/notification"
	"domainnest/internal/handler"
	"domainnest/internal/middleware"
	"domainnest/internal/service"
	"domainnest/internal/static"
	"domainnest/internal/ws"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(cfg *config.Config, db *gorm.DB, authService *service.AuthService,
	domainService *service.DomainService, recordService *service.RecordService,
	ddnsService *service.DDNSService, emailService *service.EmailService,
	settingsService *service.SettingsService, permissionService *service.PermissionService,
	ramTokenService *service.RAMTokenService, friendService *service.FriendService,
	messageService *service.MessageService, providerService *service.ProviderService,
	syncService *service.SyncService, trashService *service.TrashService,
	filterPresetService *service.FilterPresetService, notificationService *notification.Service,
	inviteCodeService *service.InviteCodeService,
	hub *ws.Hub) *gin.Engine {

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Serve static files from embedded frontend build
	// StaticFS("/static") strips the /static prefix, so root at dist/static
	staticFileFS, _ := fs.Sub(static.StaticFiles, "dist/static")
	r.StaticFS("/static", http.FS(staticFileFS))

	// SPA fallback: serve index.html for all non-API routes
	staticRootFS, _ := fs.Sub(static.StaticFiles, "dist")
	r.GET("/favicon.svg", func(c *gin.Context) {
		c.FileFromFS("/favicon.svg", http.FS(staticRootFS))
	})
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.FileFromFS("/favicon.ico", http.FS(staticRootFS))
	})
	r.NoRoute(func(c *gin.Context) {
		if len(c.Request.URL.Path) > 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(404, gin.H{"code": 404, "message": "not found"})
			return
		}
		c.FileFromFS("/", http.FS(staticRootFS))
	})

	emailVerifySvc := service.NewEmailVerifyService(db, emailService)
	authHandler := handler.NewAuthHandler(authService, emailService, emailVerifySvc, settingsService, notificationService, db, &cfg.JWT)
	domainHandler := handler.NewDomainHandler(domainService, permissionService, notificationService, db)
	recordHandler := handler.NewRecordHandler(recordService, providerService, ddnsService, notificationService, db)
	ddnsHandler := handler.NewDDNSHandler(ddnsService, ramTokenService)
	adminHandler := handler.NewAdminHandler(db, domainService, notificationService, inviteCodeService)
	notificationHandler := handler.NewNotificationHandler(messageService)
	notificationSettingHandler := handler.NewNotificationSettingHandler(db)
	settingsHandler := handler.NewSettingsHandler(db, settingsService)
	permissionHandler := handler.NewPermissionHandler(permissionService, notificationService, db)
	ramTokenHandler := handler.NewRAMTokenHandler(ramTokenService, db)
	friendHandler := handler.NewFriendHandler(friendService, notificationService, db)
	messageHandler := handler.NewMessageHandler(messageService, friendService, db)
	providerHandler := handler.NewProviderHandler(providerService, notificationService, db)
	syncHandler := handler.NewSyncHandler(syncService, recordService, db)
	trashHandler := handler.NewTrashHandler(trashService, notificationService, db)
	filterPresetHandler := handler.NewFilterPresetHandler(filterPresetService, db)
	inviteCodeHandler := handler.NewInviteCodeHandler(inviteCodeService)

	v1 := r.Group("/api/v1")

	// WebSocket endpoint (JWT via query param)
	v1.GET("/ws", ws.HandleUpgrade(hub, cfg.JWT.Secret))

	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/reset-password", authHandler.ResetPassword)
		auth.GET("/check-username", authHandler.CheckUsername)
		auth.POST("/send-verify-email", authHandler.SendVerifyEmail)
		auth.POST("/verify-email", authHandler.VerifyEmail)
	}

	authProtected := v1.Group("/auth")
	authProtected.Use(middleware.JWTAuth(cfg.JWT.Secret), middleware.OnlineTracker(db))
	{
		authProtected.GET("/profile", authHandler.GetProfile)
		authProtected.PUT("/profile", authHandler.UpdateProfile)
		authProtected.PUT("/token", authHandler.ResetToken)
		authProtected.PUT("/password", authHandler.ChangePassword)
		authProtected.POST("/avatar", authHandler.UploadAvatar)
		authProtected.GET("/logs", authHandler.MyLogs)
		authProtected.GET("/permissions", permissionHandler.MyPermissions)
		authProtected.POST("/grant-invite", authHandler.GrantInviteQuota)
		authProtected.POST("/revoke-invite", authHandler.RevokeInviteQuota)
		authProtected.GET("/invite-logs", authHandler.GetInviteLogs)
		authProtected.POST("/verify-email-change", authHandler.VerifyEmail)
		authProtected.GET("/users/search", friendHandler.SearchAllUsers)
		authProtected.GET("/pending-returns", permissionHandler.GetPendingReturns)
		authProtected.DELETE("/account", authHandler.DeleteAccount)
		authProtected.POST("/invite-codes", inviteCodeHandler.Generate)
		authProtected.GET("/invite-codes", inviteCodeHandler.List)
		authProtected.DELETE("/invite-codes/:id", inviteCodeHandler.Delete)
		authProtected.POST("/invite-codes/batch-delete", inviteCodeHandler.BatchDelete)
	}

	domains := v1.Group("/domains")
	domains.Use(middleware.JWTAuth(cfg.JWT.Secret), middleware.OnlineTracker(db))
	{
		domains.GET("", domainHandler.List)
		domains.POST("", domainHandler.Create)
		domains.POST("/batch-delete", domainHandler.BatchDelete)
		domains.POST("/batch/permissions", permissionHandler.BatchGrantMultiDomain)
		domains.GET("/transferred-away", domainHandler.GetTransferredAway)
		domains.GET("/archived", domainHandler.GetArchivedDomains)
		domains.GET("/:id", domainHandler.Get)
		domains.POST("/:id/transfer", domainHandler.Transfer)
		domains.DELETE("/:id", domainHandler.Delete)
		domains.POST("/:id/nodes/convert", domainHandler.ConvertToNode)
		domains.POST("/:id/nodes/demote", domainHandler.DemoteNode)
		domains.GET("/:id/nodes/conversion-logs", domainHandler.GetConversionLogs)
		domains.POST("/:id/records/transfer", recordHandler.TransferByHost)
		domains.GET("/:id/records", recordHandler.List)
		domains.POST("/:id/records", recordHandler.Create)
		domains.POST("/:id/records/check-conflict", recordHandler.CheckConflict)
		domains.GET("/:id/records/export", recordHandler.Export)
		domains.POST("/:id/records/import", recordHandler.Import)
		domains.PUT("/:id/records/rename-tag", recordHandler.RenameTag)
		domains.PUT("/:id/records/delete-tag", recordHandler.DeleteTag)
		domains.POST("/:id/reactivate", domainHandler.ReactivateDomain)
		domains.POST("/:id/archive", domainHandler.ArchiveDomain)
		domains.POST("/:id/restore", domainHandler.RestoreDomain)
		domains.POST("/:id/return-to-claimer", domainHandler.ReturnSubdomain)
		domains.GET("/:id/archive-info", domainHandler.ArchiveInfo)
		domains.GET("/:id/permissions", permissionHandler.List)
		domains.POST("/:id/permissions", permissionHandler.Grant)
		domains.POST("/:id/permissions/batch", permissionHandler.BatchGrant)
		domains.DELETE("/:id/permissions/:userId", permissionHandler.Revoke)
		domains.POST("/:id/permissions/:userId/revoke-request", permissionHandler.RevokeRequest)
		domains.POST("/:id/permissions/:userId/accept-return", permissionHandler.AcceptReturn)
		domains.POST("/:id/permissions/:userId/reject-return", permissionHandler.RejectReturn)
		domains.GET("/:id/pending-records", permissionHandler.GetPendingRecords)
		domains.POST("/:id/pending-records/assign", permissionHandler.AssignPendingRecords)
		domains.POST("/:id/pending-records/delete", permissionHandler.DeletePendingRecords)
		domains.POST("/:id/sync", syncHandler.ManualSync)
		domains.GET("/:id/sync-logs", syncHandler.GetSyncLogs)
	}

	records := v1.Group("/records")
	records.Use(middleware.JWTAuth(cfg.JWT.Secret), middleware.OnlineTracker(db))
	{
		records.PUT("/:id/adopt", recordHandler.AdoptRecord)
		records.PUT("/:id", recordHandler.Update)
		records.DELETE("/:id", recordHandler.Delete)
		records.PUT("/:id/toggle", recordHandler.Toggle)
		records.POST("/:id/sync", recordHandler.SyncNow)
		records.POST("/batch-delete", recordHandler.BatchDelete)
		records.POST("/batch-toggle", recordHandler.BatchToggle)
		records.PUT("/batch-tag", recordHandler.BatchTag)
	}

	ddns := v1.Group("/ddns")
	ddns.Use(middleware.TokenAuth(db))
	{
		ddns.POST("/callback", ddnsHandler.Callback)
		ddns.POST("/update", ddnsHandler.Callback) // backwards compat
		ddns.POST("/webhook", ddnsHandler.Webhook)
	}

	ramTokens := v1.Group("/ram-tokens")
	ramTokens.Use(middleware.JWTAuth(cfg.JWT.Secret), middleware.OnlineTracker(db))
	{
		ramTokens.GET("", ramTokenHandler.List)
		ramTokens.POST("", ramTokenHandler.Create)
		ramTokens.GET("/:id", ramTokenHandler.Get)
		ramTokens.PUT("/:id", ramTokenHandler.Update)
		ramTokens.POST("/:id/reset", ramTokenHandler.ResetToken)
		ramTokens.DELETE("/:id", ramTokenHandler.Delete)
	}

	admin := v1.Group("/admin")
	admin.Use(middleware.JWTAuth(cfg.JWT.Secret), middleware.AdminRequired(), middleware.OnlineTracker(db))
	{
		admin.POST("/domains", adminHandler.CreateRootDomain)
		admin.POST("/domains/batch-delete", adminHandler.BatchDeleteDomains)
		admin.GET("/domains", adminHandler.ListDomains)
		admin.GET("/domains/tree", adminHandler.GetDomainTree)
		admin.GET("/domains/:id/detail", adminHandler.GetDomainDetail)
		admin.GET("/domains/:id/records", adminHandler.ListDomainRecords)
		admin.DELETE("/domains/:id/records/:rid", adminHandler.AdminDeleteRecord)
		admin.POST("/domains/:id/records/:rid/toggle", adminHandler.AdminToggleRecord)
		admin.POST("/domains/:id/change-owner", adminHandler.AssignDomain)
		admin.DELETE("/domains/:id/permissions/:userId", adminHandler.RevokePermission)
		admin.GET("/users", adminHandler.ListUsers)
		admin.PUT("/users/:id", adminHandler.UpdateUser)
		admin.POST("/users/:id/reset-password", adminHandler.AdminResetPassword)
		admin.DELETE("/users/:id", adminHandler.DisableUser)
		admin.POST("/users/:id/promote", adminHandler.PromoteToAdmin)
		admin.POST("/users/:id/demote", adminHandler.DemoteFromAdmin)
		admin.GET("/logs", adminHandler.ListLogs)
		admin.POST("/records/:id/sync", adminHandler.RetrySync)
		admin.POST("/notifications/broadcast", adminHandler.BroadcastNotification)
		admin.GET("/notifications", adminHandler.ListAllNotifications)
		admin.GET("/notifications/stats", adminHandler.GetNotificationStats)
		admin.DELETE("/notifications/:id", adminHandler.AdminDeleteNotification)
		admin.POST("/notifications/purge-expired", adminHandler.AdminPurgeExpiredNotifications)
		admin.GET("/settings/:category", settingsHandler.Get)
		admin.PUT("/settings/:category", settingsHandler.Set)
		admin.POST("/settings/smtp/test", settingsHandler.TestSMTP)
		admin.POST("/invite-codes", adminHandler.GenerateInviteCodes)
		admin.GET("/invite-codes", adminHandler.ListInviteCodes)
		admin.DELETE("/invite-codes/:id", adminHandler.DeleteInviteCode)
	}

	friends := v1.Group("/friends")
	friends.Use(middleware.JWTAuth(cfg.JWT.Secret), middleware.OnlineTracker(db))
	{
		friends.GET("", friendHandler.ListFriends)
		friends.POST("", friendHandler.SendRequest)
		friends.DELETE("/:id", friendHandler.RemoveFriend)
		friends.GET("/requests/pending", friendHandler.ListPendingRequests)
		friends.GET("/requests/sent", friendHandler.ListSentRequests)
		friends.POST("/requests/:id/accept", friendHandler.AcceptRequest)
		friends.POST("/requests/:id/reject", friendHandler.RejectRequest)
		friends.GET("/search", friendHandler.SearchUsers)
	}

	messages := v1.Group("/messages")
	messages.Use(middleware.JWTAuth(cfg.JWT.Secret), middleware.OnlineTracker(db))
	{
		messages.POST("", messageHandler.SendMessage)
		messages.GET("/conversations", messageHandler.GetConversations)
		messages.GET("/unread-count", messageHandler.UnreadCount)
		messages.GET("/notifications", messageHandler.GetNotifications)
		messages.GET("/notifications/unread-count", messageHandler.NotificationUnreadCount)
		messages.POST("/notifications/:id/read", messageHandler.MarkNotificationAsRead)
		messages.POST("/notifications/:id/action", messageHandler.HandleNotificationAction)
		messages.POST("/notifications/read-all", messageHandler.MarkAllNotificationsAsRead)
		messages.GET("/:id", messageHandler.GetMessages)
		messages.POST("/:id/read", messageHandler.MarkAsRead)
	}

	notifications := v1.Group("/notifications")
	notifications.Use(middleware.JWTAuth(cfg.JWT.Secret), middleware.OnlineTracker(db))
	{
		notifications.GET("", notificationHandler.List)
		notifications.GET("/unread-count", notificationHandler.UnreadCount)
		notifications.PUT("/:id/read", notificationHandler.MarkAsRead)
		notifications.PUT("/read-all", notificationHandler.MarkAllAsRead)
		notifications.DELETE("/:id", notificationHandler.Delete)
	}

	notifSettings := v1.Group("/notification-settings")
	notifSettings.Use(middleware.JWTAuth(cfg.JWT.Secret), middleware.OnlineTracker(db))
	{
		notifSettings.GET("", notificationSettingHandler.List)
		notifSettings.PUT("", notificationSettingHandler.Update)
	}

	providers := v1.Group("/providers")
	providers.Use(middleware.JWTAuth(cfg.JWT.Secret), middleware.OnlineTracker(db))
	{
		providers.GET("", providerHandler.List)
		providers.POST("", providerHandler.Create)
		providers.GET("/:id", providerHandler.Get)
		providers.PUT("/:id", providerHandler.Update)
		providers.DELETE("/:id", providerHandler.Delete)
		providers.GET("/:id/domains", providerHandler.ListDomains)
		providers.POST("/:id/claim", providerHandler.ClaimDomain)
		providers.POST("/:id/domains/:did/reclaim", domainHandler.ReclaimDomain)
	}

	trash := v1.Group("/trash")
	trash.Use(middleware.JWTAuth(cfg.JWT.Secret), middleware.OnlineTracker(db))
	{
		trash.GET("", trashHandler.List)
		trash.POST("/empty", trashHandler.Empty)
		trash.POST("/batch-trash", trashHandler.BatchTrash)
		trash.POST("/batch-restore", trashHandler.BatchRestore)
		trash.POST("/:id/trash", trashHandler.Trash)
		trash.POST("/:id/restore", trashHandler.Restore)
		trash.DELETE("/:id", trashHandler.Delete)
	}

	filterPresets := v1.Group("/filter-presets")
	filterPresets.Use(middleware.JWTAuth(cfg.JWT.Secret), middleware.OnlineTracker(db))
	{
		filterPresets.GET("", filterPresetHandler.List)
		filterPresets.POST("", filterPresetHandler.Save)
		filterPresets.DELETE("/:id", filterPresetHandler.Delete)
	}

	return r
}
