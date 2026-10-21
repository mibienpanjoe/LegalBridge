package store

import (
	"context"
	"time"
)

// Chunk is a text segment from an ingested document together with its
// pre-computed bge-m3 embedding vector.
type Chunk struct {
	ID               string
	DocumentID       string
	DocumentFilename string // populated by SimilaritySearch via JOIN with documents
	Content          string
	Embedding        []float32
	ChunkIndex       int
	CreatedAt        time.Time
	Similarity       float64 // cosine similarity to the query; populated by SimilaritySearch
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

	// SimilaritySearch returns up to topK chunks of the given document whose
	// cosine similarity to vector is at least minSimilarity, ordered
	// nearest-first. Chunks of other documents are never returned.
	SimilaritySearch(ctx context.Context, documentID string, vector []float32, topK int, minSimilarity float64) ([]Chunk, error)

	// Ping checks that the database connection is healthy.
	Ping(ctx context.Context) error
}
