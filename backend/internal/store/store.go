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
	PageStart        int    // first source PDF page of the chunk (1-based; 0 if unknown)
	PageEnd          int    // last source PDF page of the chunk
	Section          string // article/section label, e.g. "Article 12"; empty if unknown
	CreatedAt        time.Time
	Similarity       float64 // cosine similarity to the query; populated by SimilaritySearch
}

// Store owns all PostgreSQL read and write operations.
// It holds no business logic and calls no external APIs.
type Store interface {
	// WriteDocument inserts a document record and returns its generated UUID.
	// A zero expiresAt means the document never expires.
	WriteDocument(ctx context.Context, filename string, expiresAt time.Time) (string, error)

	// DocumentExists reports whether the document exists and has not expired.
	DocumentExists(ctx context.Context, documentID string) (bool, error)

	// DeleteExpiredDocuments deletes every document whose expiry has passed,
	// together with its chunks, and returns how many were deleted.
	DeleteExpiredDocuments(ctx context.Context) (int64, error)

	// WriteChunks persists all chunks for a document in a single transaction.
	// It returns an error (without writing anything) if any chunk has an
	// embedding whose length is not exactly 1024.
	WriteChunks(ctx context.Context, documentID string, chunks []Chunk) error

	// SimilaritySearch returns up to topK chunks of the given document whose
	// cosine similarity to vector is at least minSimilarity, ordered
	// nearest-first. Chunks of other documents are never returned.
	SimilaritySearch(ctx context.Context, documentID string, vector []float32, topK int, minSimilarity float64) ([]Chunk, error)

	// ChunkContents returns the ID and content of every chunk of a document,
	// ordered by chunk index. Embeddings are not loaded.
	ChunkContents(ctx context.Context, documentID string) ([]Chunk, error)

	// Ping checks that the database connection is healthy.
	Ping(ctx context.Context) error
}
