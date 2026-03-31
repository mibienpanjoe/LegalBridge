package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mibienpanjoe/legalbridge/pkg/config"
	"github.com/mibienpanjoe/legalbridge/migrations"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, migrations.Init); err != nil {
		log.Fatalf("apply migration: %v", err)
	}

	fmt.Println("migrations applied successfully")
}
