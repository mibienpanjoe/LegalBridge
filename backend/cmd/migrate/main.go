package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mibienpanjoe/legalbridge/migrations"
	"github.com/mibienpanjoe/legalbridge/pkg/config"
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

	sqls, err := migrations.All()
	if err != nil {
		log.Fatalf("load migrations: %v", err)
	}
	for i, sql := range sqls {
		if _, err := conn.Exec(ctx, sql); err != nil {
			log.Fatalf("apply migration %d: %v", i+1, err)
		}
	}

	fmt.Println("migrations applied successfully")
}
