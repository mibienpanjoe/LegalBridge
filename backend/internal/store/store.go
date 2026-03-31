package store

import (
	"context"
	"time"
)

// Chunk is a text segment from an ingested document together with its
// pre-computed bge-m3 embedding vector.
type Chunk struct {
	ID         string
	DocumentID string
	Content    string
	Embedding  []float32
	ChunkIndex int
	CreatedAt  time.Time
}

// Store owns all PostgreSQL read and write operations.
// It holds no business logic and calls no external APIs.
type Store interface {
	// WriteDocument inserts a document record and returns its generated UUID.
	WriteDocument(ctx context.Context, filename string) (string, error)

	// WriteChunks persists all chunks for a document in a single transaction.
	// It returns an error (without writing anything) if any chunk has an
	// embedding whose length is not exactly 1024.
	WriteChunks(ctx context.Context, documentID string, chunks []Chunk) error

	// SimilaritySearch returns the topK chunks closest to vector by cosine
	// distance, ordered nearest-first.
	SimilaritySearch(ctx context.Context, vector []float32, topK int) ([]Chunk, error)

	// Ping checks that the database connection is healthy.
	Ping(ctx context.Context) error
}
