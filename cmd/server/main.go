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
