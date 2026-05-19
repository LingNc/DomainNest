package main

import (
	"fmt"
	"log"

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

	db, err := model.InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	if err := model.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	authService := service.NewAuthService(db)
	permissionService := service.NewPermissionService(db)
	domainService := service.NewDomainService(db, permissionService)
	recordService := service.NewRecordService(db, permissionService)
	providerService := service.NewProviderService(db)
	ddnsService := service.NewDDNSService(db, domainService, recordService, providerService)
	settingsService := service.NewSettingsService(db)
	emailService := service.NewEmailServiceWithSettings(&cfg.SMTP, settingsService)
	ramTokenService := service.NewRAMTokenService(db)
	friendService := service.NewFriendService(db)
	messageService := service.NewMessageService(db)

	if err := authService.EnsureAdmin(cfg.Admin.Username, cfg.Admin.Password); err != nil {
		log.Fatalf("Failed to ensure admin user: %v", err)
	}

	r := router.Setup(cfg, db, authService, domainService, recordService, ddnsService, emailService, settingsService, permissionService, ramTokenService, friendService, messageService, providerService)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
