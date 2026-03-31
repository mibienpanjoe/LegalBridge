package store

import (
	"context"
	"errors"
	"testing"
)

// TestWriteChunks_DimensionValidation verifies that WriteChunks rejects any
// chunk whose embedding length differs from 1024, without touching the DB.
// The validation runs before the pool is accessed, so a nil pool is safe.
func TestWriteChunks_DimensionValidation(t *testing.T) {
	s := &PostgresStore{pool: nil}

	tests := []struct {
		name      string
		chunks    []Chunk
		wantErr   bool
		wantGot   int
	}{
		{
			name: "correct dimension 1024",
			// WriteChunks would proceed past validation and panic on nil pool,
			// so we only test wrong dimensions here.
		},
		{
			name: "wrong dimension 512",
			chunks: []Chunk{
				{Content: "a", Embedding: make([]float32, 512), ChunkIndex: 0},
			},
			wantErr: true,
			wantGot: 512,
		},
		{
			name: "wrong dimension 1536",
			chunks: []Chunk{
				{Content: "a", Embedding: make([]float32, 1536), ChunkIndex: 0},
			},
			wantErr: true,
			wantGot: 1536,
		},
		{
			name: "zero-length embedding",
			chunks: []Chunk{
				{Content: "a", Embedding: []float32{}, ChunkIndex: 0},
			},
			wantErr: true,
			wantGot: 0,
		},
		{
			name: "second chunk has wrong dimension",
			chunks: []Chunk{
				{Content: "a", Embedding: make([]float32, 1024), ChunkIndex: 0},
				{Content: "b", Embedding: make([]float32, 256), ChunkIndex: 1},
			},
			wantErr: true,
			wantGot: 256,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				return // skip cases that would reach the nil pool
			}
			err := s.WriteChunks(context.Background(), "doc-id", tt.chunks)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var dimErr *InvalidEmbeddingDimensionError
			if !errors.As(err, &dimErr) {
				t.Fatalf("expected InvalidEmbeddingDimensionError, got %T: %v", err, err)
			}
			if dimErr.Got != tt.wantGot {
				t.Errorf("Got=%d, want %d", dimErr.Got, tt.wantGot)
			}
			if dimErr.Want != embeddingDimension {
				t.Errorf("Want=%d, expected %d", dimErr.Want, embeddingDimension)
			}
		})
	}
}
