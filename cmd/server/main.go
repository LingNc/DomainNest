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

	db, err := model.InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	if err := model.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	aliyunClient, err := aliyun.NewClient(&cfg.Aliyun)
	if err != nil {
		log.Printf("Warning: Failed to initialize Aliyun client: %v", err)
	}

	authService := service.NewAuthService(db)
	domainService := service.NewDomainService(db)
	recordService := service.NewRecordService(db)
	ddnsService := service.NewDDNSService(db, domainService, recordService, aliyunClient)

	if err := authService.EnsureAdmin(cfg.Admin.Username, cfg.Admin.Password); err != nil {
		log.Fatalf("Failed to ensure admin user: %v", err)
	}

	r := router.Setup(cfg, db, authService, domainService, recordService, ddnsService)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
