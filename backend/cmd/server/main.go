package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/mibienpanjoe/legalbridge/internal/api"
	"github.com/mibienpanjoe/legalbridge/internal/ingester"
	"github.com/mibienpanjoe/legalbridge/internal/query"
	"github.com/mibienpanjoe/legalbridge/internal/store"
	"github.com/mibienpanjoe/legalbridge/pkg/config"
)

func main() {
	// INV-08: validate config at startup; fatal if required values are missing.
	cfg := config.Load()
	if err := config.Validate(cfg); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	ctx := context.Background()

	// Wire up the store.
	pg, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}

	// Wire up the embedding and LLM clients.
	embeddingClient := ingester.NewEmbeddingClient(cfg)
	llmClient := query.NewLLMClient(cfg)

	// Wire up the ingestion and query pipelines.
	ing := ingester.NewIngester(pg, embeddingClient)
	qp := query.NewQueryProcessor(pg, embeddingClient, llmClient)

	// Build the Gin router.
	r := gin.New()
	r.Use(api.LoggingMiddleware(), api.CORSMiddleware(cfg.CORSAllowedOrigin), gin.Recovery())

	handler := api.NewAPIHandler(ing, qp, pg)
	handler.RegisterRoutes(r)

	log.Println("LegalBridge backend listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
