package ingester

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

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
	gotExpiresAt  time.Time
	writtenChunks []store.Chunk
	docWriteErr   error
	chunkWriteErr error
}

func (m *mockStore) WriteDocument(_ context.Context, _ string, expiresAt time.Time) (string, error) {
	m.gotExpiresAt = expiresAt
	return m.docID, m.docWriteErr
}
func (m *mockStore) DocumentExists(_ context.Context, _ string) (bool, error) { return true, nil }
func (m *mockStore) DeleteExpiredDocuments(_ context.Context) (int64, error)  { return 0, nil }

func (m *mockStore) WriteChunks(_ context.Context, _ string, chunks []store.Chunk) error {
	m.writtenChunks = append(m.writtenChunks, chunks...)
	return m.chunkWriteErr
}

func (m *mockStore) SimilaritySearch(_ context.Context, _ string, _ []float32, _ int, _ float64) ([]store.Chunk, error) {
	return nil, nil
}

func (m *mockStore) ChunkContents(_ context.Context, _ string) ([]store.Chunk, error) {
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

// textPages wraps text as a single one-line page; empty text yields no pages.
func textPages(text string) []Page {
	if text == "" {
		return nil
	}
	return []Page{{Number: 1, Lines: []string{text}}}
}

func TestIngestText_HappyPath(t *testing.T) {
	text := buildTestText(600) // > chunkSize → at least 2 chunks
	ec := &mockEmbeddingClient{}
	ms := &mockStore{docID: "doc-1"}
	ing := NewIngester(ms, ec, 0)

	result, err := ing.ingestPages(context.Background(), textPages(text), "test.pdf")
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
	if ms.writtenChunks[0].PageStart != 1 {
		t.Errorf("expected page 1 recorded on chunk, got %d", ms.writtenChunks[0].PageStart)
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
	ing := NewIngester(ms, ec, 0)

	_, err := ing.ingestPages(context.Background(), textPages(text), "test.pdf")
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
	ing := NewIngester(&mockStore{}, &mockEmbeddingClient{}, 0)
	_, err := ing.ingestPages(context.Background(), textPages(""), "test.pdf")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
	var extractErr *ExtractionFailedError
	if !errors.As(err, &extractErr) {
		t.Errorf("expected ExtractionFailedError, got %T", err)
	}
}

// TestOllamaTag verifies that the shared HuggingFace model id resolves to the
// Ollama library tag for the same model.
func TestOllamaTag(t *testing.T) {
	cases := map[string]string{
		"BAAI/bge-m3": "bge-m3",
		"bge-m3":      "bge-m3",
	}
	for in, want := range cases {
		if got := ollamaTag(in); got != want {
			t.Errorf("ollamaTag(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestIngest_ExpiryFromTTL verifies that uploads get an expiry of now+ttl and
// that a zero ttl (seeded documents) stores no expiry.
func TestIngest_ExpiryFromTTL(t *testing.T) {
	ms := &mockStore{docID: "doc"}
	before := time.Now()
	res, err := NewIngester(ms, &mockEmbeddingClient{}, 24*time.Hour).
		ingestPages(context.Background(), textPages("Article 1 : texte."), "a.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d := ms.gotExpiresAt.Sub(before); d < 24*time.Hour-time.Second || d > 24*time.Hour+time.Second {
		t.Errorf("expected expiry ≈ now+24h, got now+%v", d)
	}
	if !res.ExpiresAt.Equal(ms.gotExpiresAt) {
		t.Errorf("IngestResult.ExpiresAt %v differs from stored %v", res.ExpiresAt, ms.gotExpiresAt)
	}

	ms = &mockStore{docID: "seed"}
	if _, err := NewIngester(ms, &mockEmbeddingClient{}, 0).
		ingestPages(context.Background(), textPages("Article 1 : texte."), "seed.pdf"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ms.gotExpiresAt.IsZero() {
		t.Errorf("expected no expiry with zero ttl, got %v", ms.gotExpiresAt)
	}
}
