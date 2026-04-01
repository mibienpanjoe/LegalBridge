package ingester

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/mibienpanjoe/legalbridge/internal/store"
)

// mockEmbeddingClient returns a 1024-dimensional zero vector and optionally fails
// after failAfter successful calls.
type mockEmbeddingClient struct {
	callCount int
	failAfter int // 0 means never fail
	err       error
}

func (m *mockEmbeddingClient) Embed(_ context.Context, _, _ string) ([]float32, error) {
	m.callCount++
	if m.failAfter > 0 && m.callCount > m.failAfter {
		return nil, m.err
	}
	return make([]float32, 1024), nil
}

// mockStore records WriteDocument and WriteChunks calls and surfaces injected errors.
type mockStore struct {
	docID         string
	writtenChunks []store.Chunk
	docWriteErr   error
	chunkWriteErr error
}

func (m *mockStore) WriteDocument(_ context.Context, _ string) (string, error) {
	return m.docID, m.docWriteErr
}

func (m *mockStore) WriteChunks(_ context.Context, _ string, chunks []store.Chunk) error {
	m.writtenChunks = append(m.writtenChunks, chunks...)
	return m.chunkWriteErr
}

func (m *mockStore) SimilaritySearch(_ context.Context, _ []float32, _ int) ([]store.Chunk, error) {
	return nil, nil
}

func (m *mockStore) Ping(_ context.Context) error { return nil }

// buildTestText returns a text of exactly wordCount unique words.
func buildTestText(wordCount int) string {
	words := make([]string, wordCount)
	for i := range words {
		words[i] = fmt.Sprintf("word%d", i)
	}
	return strings.Join(words, " ")
}

func TestIngestText_HappyPath(t *testing.T) {
	text := buildTestText(600) // > chunkSize → at least 2 chunks
	ec := &mockEmbeddingClient{}
	ms := &mockStore{docID: "doc-1"}
	ing := NewIngester(ms, ec)

	result, err := ing.ingestText(context.Background(), text, "test.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DocumentID != "doc-1" {
		t.Errorf("expected doc-1, got %s", result.DocumentID)
	}
	if result.ChunkCount != len(ms.writtenChunks) {
		t.Errorf("IngestResult.ChunkCount=%d but store received %d chunks",
			result.ChunkCount, len(ms.writtenChunks))
	}
	if result.ChunkCount < 2 {
		t.Errorf("expected ≥2 chunks from 600 words, got %d", result.ChunkCount)
	}
	if ec.callCount != result.ChunkCount {
		t.Errorf("Embed called %d times but ChunkCount=%d", ec.callCount, result.ChunkCount)
	}
}

func TestIngestText_AtomicOnEmbeddingFailure(t *testing.T) {
	// 600 words → 2 chunks; first Embed succeeds, second fails.
	// WriteChunks must never be called.
	text := buildTestText(600)
	ec := &mockEmbeddingClient{
		failAfter: 1,
		err:       &EmbeddingUnavailableError{Reason: "service down"},
	}
	ms := &mockStore{docID: "doc-2"}
	ing := NewIngester(ms, ec)

	_, err := ing.ingestText(context.Background(), text, "test.pdf")
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	var embErr *EmbeddingUnavailableError
	if !errors.As(err, &embErr) {
		t.Errorf("expected EmbeddingUnavailableError, got %T: %v", err, err)
	}

	// Atomic guarantee: nothing written to store
	if len(ms.writtenChunks) != 0 {
		t.Errorf("expected 0 chunks written on embedding failure, got %d", len(ms.writtenChunks))
	}
}

func TestIngestText_EmptyText(t *testing.T) {
	ing := NewIngester(&mockStore{}, &mockEmbeddingClient{})
	_, err := ing.ingestText(context.Background(), "", "test.pdf")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
	var extractErr *ExtractionFailedError
	if !errors.As(err, &extractErr) {
		t.Errorf("expected ExtractionFailedError, got %T", err)
	}
}
