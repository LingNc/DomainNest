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
