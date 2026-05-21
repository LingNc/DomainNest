package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"domainnest/internal/config"
	"domainnest/internal/model"
	"domainnest/internal/router"
	"domainnest/internal/service"
	"domainnest/internal/ws"
)

func main() {
	// Set up log file output (writes to both stderr and logs/server.log)
	os.MkdirAll("logs", 0o755)
	logFile, err := os.OpenFile("logs/server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Printf("Warning: cannot open log file: %v", err)
	} else {
		log.SetOutput(io.MultiWriter(os.Stderr, logFile))
		defer logFile.Close()
	}

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
	recordService := service.NewRecordService(db, permissionService, domainService)
	providerService := service.NewProviderService(db)
	ddnsService := service.NewDDNSService(db, domainService, recordService, providerService)
	settingsService := service.NewSettingsService(db)
	emailService := service.NewEmailServiceWithSettings(&cfg.SMTP, settingsService)
	ramTokenService := service.NewRAMTokenService(db)
	friendService := service.NewFriendService(db)
	messageService := service.NewMessageService(db)

	syncService := service.NewSyncService(db, ddnsService, &cfg.Sync)
	syncService.Start()
	defer syncService.Stop()

	wsHub := ws.InitHub()

	if err := authService.EnsureAdmin(cfg.Admin.Username, cfg.Admin.Password); err != nil {
		log.Fatalf("Failed to ensure admin user: %v", err)
	}

	r := router.Setup(cfg, db, authService, domainService, recordService, ddnsService, emailService, settingsService, permissionService, ramTokenService, friendService, messageService, providerService, syncService, wsHub)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}
