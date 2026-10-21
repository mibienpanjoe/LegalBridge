//go:build integration

// Integration tests for PostgresStore against a real pgvector database.
//
// Run with a migrated database:
//
//	TEST_DATABASE_URL=postgres://... go test -tags integration ./internal/store/...
package store

import (
	"context"
	"math"
	"os"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *PostgresStore {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping store integration tests")
	}
	s, err := NewPostgresStore(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(s.pool.Close)
	return s
}

// unitVector returns a 1024-dim vector whose first two components are
// (cos θ, sin θ), so the cosine similarity between two such vectors is
// cos(θ1 − θ2).
func unitVector(theta float64) []float32 {
	v := make([]float32, embeddingDimension)
	v[0] = float32(math.Cos(theta))
	v[1] = float32(math.Sin(theta))
	return v
}

func writeDoc(t *testing.T, s *PostgresStore, name string, thetas ...float64) string {
	t.Helper()
	ctx := context.Background()
	id, err := s.WriteDocument(ctx, name, time.Time{})
	if err != nil {
		t.Fatalf("write document: %v", err)
	}
	t.Cleanup(func() { s.pool.Exec(context.Background(), `DELETE FROM documents WHERE id = $1`, id) })

	chunks := make([]Chunk, len(thetas))
	for i, th := range thetas {
		chunks[i] = Chunk{Content: name, Embedding: unitVector(th), ChunkIndex: i,
			PageStart: i + 1, PageEnd: i + 2, Section: "Article " + name}
	}
	if err := s.WriteChunks(ctx, id, chunks); err != nil {
		t.Fatalf("write chunks: %v", err)
	}
	return id
}

// TestSimilaritySearch_ScopedToDocument verifies that chunks of another
// document are never returned, even when they are closer to the query.
func TestSimilaritySearch_ScopedToDocument(t *testing.T) {
	s := newTestStore(t)
	mine := writeDoc(t, s, "mine.pdf", 0.5, 0.6)
	writeDoc(t, s, "other.pdf", 0.0, 0.01) // nearer to the query than "mine"

	got, err := s.SimilaritySearch(context.Background(), mine, unitVector(0), 5, -1)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(got))
	}
	for _, c := range got {
		if c.DocumentID != mine || c.DocumentFilename != "mine.pdf" {
			t.Errorf("chunk from another document returned: %+v", c.DocumentFilename)
		}
	}
	if got[0].Similarity < got[1].Similarity {
		t.Error("results are not ordered nearest-first")
	}
	// Location metadata round-trips: chunk 0 (θ=0.5) is the nearest.
	if got[0].PageStart != 1 || got[0].PageEnd != 2 || got[0].Section != "Article mine.pdf" {
		t.Errorf("location not persisted: pages %d–%d, section %q", got[0].PageStart, got[0].PageEnd, got[0].Section)
	}
}

// TestSimilaritySearch_Threshold verifies that chunks below minSimilarity are
// dropped and that a document with no chunk above it yields zero results.
func TestSimilaritySearch_Threshold(t *testing.T) {
	s := newTestStore(t)
	// similarities to the query: cos(0.2)≈0.98, cos(1.0)≈0.54, cos(1.4)≈0.17
	doc := writeDoc(t, s, "doc.pdf", 0.2, 1.0, 1.4)

	got, err := s.SimilaritySearch(context.Background(), doc, unitVector(0), 5, 0.45)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 chunks above 0.45, got %d", len(got))
	}
	if math.Abs(got[0].Similarity-math.Cos(0.2)) > 1e-4 {
		t.Errorf("expected similarity %.4f, got %.4f", math.Cos(0.2), got[0].Similarity)
	}

	got, err = s.SimilaritySearch(context.Background(), doc, unitVector(0), 5, 0.99)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no chunk above 0.99, got %d", len(got))
	}
}

// TestChunkContents verifies that only the document's chunks are returned, in
// chunk order, without embeddings.
func TestChunkContents(t *testing.T) {
	s := newTestStore(t)
	doc := writeDoc(t, s, "doc.pdf", 0.1, 0.2, 0.3)
	writeDoc(t, s, "other.pdf", 0.1)

	got, err := s.ChunkContents(context.Background(), doc)
	if err != nil {
		t.Fatalf("chunk contents: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(got))
	}
	for _, c := range got {
		if c.ID == "" || c.Content != "doc.pdf" || c.Embedding != nil {
			t.Errorf("unexpected chunk %+v", c)
		}
	}
}

// TestDocumentExpiry verifies that an expired document is reported missing,
// is deleted by the sweep together with its chunks, and that documents
// without expiry or with a future expiry are kept.
func TestDocumentExpiry(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	expired, err := s.WriteDocument(ctx, "expired.pdf", time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatalf("write document: %v", err)
	}
	if err := s.WriteChunks(ctx, expired, []Chunk{{Content: "x", Embedding: unitVector(0)}}); err != nil {
		t.Fatalf("write chunks: %v", err)
	}
	future, _ := s.WriteDocument(ctx, "future.pdf", time.Now().Add(time.Hour))
	forever := writeDoc(t, s, "seed.pdf", 0)
	t.Cleanup(func() { s.pool.Exec(context.Background(), `DELETE FROM documents WHERE id = $1`, future) })

	for id, want := range map[string]bool{expired: false, future: true, forever: true} {
		got, err := s.DocumentExists(ctx, id)
		if err != nil || got != want {
			t.Errorf("DocumentExists(%s) = %v, %v; want %v", id, got, err, want)
		}
	}

	n, err := s.DeleteExpiredDocuments(ctx)
	if err != nil || n < 1 {
		t.Fatalf("DeleteExpiredDocuments = %d, %v; want ≥1", n, err)
	}
	var chunks int
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM chunks WHERE document_id = $1`, expired).Scan(&chunks)
	if chunks != 0 {
		t.Errorf("expected expired document's chunks to cascade-delete, %d left", chunks)
	}
	for _, id := range []string{future, forever} {
		if ok, _ := s.DocumentExists(ctx, id); !ok {
			t.Errorf("document %s should have been kept", id)
		}
	}
}
