// cmd/seed ingests a demo PDF document into the database if no documents
// exist yet.  It is safe to run repeatedly — it is a no-op when the database
// already contains at least one document.
//
// Required env vars:
//
//	DATABASE_URL    — PostgreSQL connection string
//	DEMO_PDF_PATH   — path to the PDF file to ingest on first run
//
// Optional (same as the server):
//
//	EMBEDDING_PROVIDER, OLLAMA_BASE_URL, HF_API_KEY, …
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"github.com/mibienpanjoe/legalbridge/internal/ingester"
	"github.com/mibienpanjoe/legalbridge/internal/store"
	"github.com/mibienpanjoe/legalbridge/pkg/config"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("seed: DATABASE_URL is required")
	}

	ctx := context.Background()

	// Skip if a document was already ingested.
	exists, err := hasDocuments(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("seed: check documents: %v", err)
	}
	if exists {
		log.Println("seed: database already contains documents — skipping")
		return
	}

	pdfPath := os.Getenv("DEMO_PDF_PATH")
	if pdfPath == "" {
		log.Fatal("seed: DEMO_PDF_PATH is not set and no documents exist in the database")
	}

	pdfBytes, err := os.ReadFile(pdfPath)
	if err != nil {
		log.Fatalf("seed: read demo PDF %q: %v", pdfPath, err)
	}

	pg, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("seed: connect to database: %v", err)
	}

	ec := ingester.NewEmbeddingClient(cfg)
	ing := ingester.NewIngester(pg, ec)

	filename := filepath.Base(pdfPath)
	result, err := ing.Ingest(ctx, pdfBytes, filename)
	if err != nil {
		log.Fatalf("seed: ingest %q: %v", filename, err)
	}

	log.Printf("seed: ingested %q → %d chunks (document_id=%s)",
		filename, result.ChunkCount, result.DocumentID)
}

// hasDocuments returns true when the documents table contains at least one row.
func hasDocuments(ctx context.Context, databaseURL string) (bool, error) {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return false, fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(ctx)

	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM documents").Scan(&count)
	if err != nil {
		return false, fmt.Errorf("query: %w", err)
	}
	return count > 0, nil
}
