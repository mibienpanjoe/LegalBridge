package store

import "fmt"

// DatabaseUnavailableError is returned when the PostgreSQL connection fails.
type DatabaseUnavailableError struct {
	Cause error
}

func (e *DatabaseUnavailableError) Error() string {
	return fmt.Sprintf("database unavailable: %v", e.Cause)
}

func (e *DatabaseUnavailableError) Unwrap() error { return e.Cause }

// InvalidEmbeddingDimensionError is returned by WriteChunks when a chunk
// carries an embedding whose length is not exactly 1024 (bge-m3 output).
type InvalidEmbeddingDimensionError struct {
	Got  int
	Want int
}

func (e *InvalidEmbeddingDimensionError) Error() string {
	return fmt.Sprintf("invalid embedding dimension: got %d, want %d", e.Got, e.Want)
}
