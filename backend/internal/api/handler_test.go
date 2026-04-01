package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mibienpanjoe/legalbridge/internal/ingester"
	"github.com/mibienpanjoe/legalbridge/internal/query"
	"github.com/mibienpanjoe/legalbridge/internal/store"
)

// ─── fakes ───────────────────────────────────────────────────────────────────

type fakeIngestor struct {
	result *ingester.IngestResult
	err    error
}

func (f *fakeIngestor) Ingest(_ context.Context, _ []byte, _ string) (*ingester.IngestResult, error) {
	return f.result, f.err
}

type fakeQuerySvc struct {
	result *query.QueryResult
	err    error
}

func (f *fakeQuerySvc) Query(_ context.Context, _ string) (*query.QueryResult, error) {
	return f.result, f.err
}

type fakeStore struct{ pingErr error }

func (f *fakeStore) WriteDocument(_ context.Context, _ string) (string, error) { return "", nil }
func (f *fakeStore) WriteChunks(_ context.Context, _ string, _ []store.Chunk) error { return nil }
func (f *fakeStore) SimilaritySearch(_ context.Context, _ []float32, _ int) ([]store.Chunk, error) {
	return nil, nil
}
func (f *fakeStore) Ping(_ context.Context) error { return f.pingErr }

// ─── helpers ─────────────────────────────────────────────────────────────────

func newTestRouter(h *APIHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r)
	return r
}

// pdfMultipart builds a multipart/form-data body containing a minimal PDF.
func pdfMultipart(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", "test.pdf")
	if err != nil {
		t.Fatal(err)
	}
	// Minimal PDF: starts with %PDF- so http.DetectContentType returns application/pdf.
	fw.Write([]byte("%PDF-1.4\n%%EOF"))
	w.Close()
	return &buf, w.FormDataContentType()
}

// ─── tests ───────────────────────────────────────────────────────────────────

// TestHandleHealth_OK verifies that GET /api/health returns 200 with status=ok
// when the store is reachable.
func TestHandleHealth_OK(t *testing.T) {
	h := NewAPIHandler(&fakeIngestor{}, &fakeQuerySvc{}, &fakeStore{})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body healthResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("expected status=ok, got %q", body.Status)
	}
	if body.Database != "ok" {
		t.Errorf("expected database=ok, got %q", body.Database)
	}
}

// TestHandleIngest_OK verifies that POST /api/ingest with a valid PDF returns
// 200 with document_id and chunk_count.
func TestHandleIngest_OK(t *testing.T) {
	h := NewAPIHandler(
		&fakeIngestor{result: &ingester.IngestResult{DocumentID: "doc-123", ChunkCount: 5}},
		&fakeQuerySvc{},
		&fakeStore{},
	)
	r := newTestRouter(h)

	body, contentType := pdfMultipart(t)
	req := httptest.NewRequest(http.MethodPost, "/api/ingest", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ingestResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if resp.DocumentID != "doc-123" {
		t.Errorf("expected document_id=doc-123, got %q", resp.DocumentID)
	}
	if resp.ChunkCount != 5 {
		t.Errorf("expected chunk_count=5, got %d", resp.ChunkCount)
	}
}

// TestHandleQuery_OK verifies that POST /api/query with a valid question returns
// 200 with a cited answer.
func TestHandleQuery_OK(t *testing.T) {
	h := NewAPIHandler(
		&fakeIngestor{},
		&fakeQuerySvc{result: &query.QueryResult{
			Answer: "Registration is required within 28 days [1].",
			Citations: []query.Citation{
				{Index: 1, DocumentName: "act.pdf", Passage: "28-day rule passage"},
			},
		}},
		&fakeStore{},
	)
	r := newTestRouter(h)

	reqBody := strings.NewReader(`{"question":"How long to register?"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/query", reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp queryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if resp.NoResults {
		t.Error("expected no_results=false")
	}
	if len(resp.Citations) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(resp.Citations))
	}
	if resp.Citations[0].Index != 1 || resp.Citations[0].DocumentName != "act.pdf" {
		t.Errorf("unexpected citation: %+v", resp.Citations[0])
	}
}

// TestHandleQuery_NoResults verifies that NoResultsError is translated to
// 200 with no_results=true (per CLAUDE.md error translation table).
func TestHandleQuery_NoResults(t *testing.T) {
	h := NewAPIHandler(
		&fakeIngestor{},
		&fakeQuerySvc{err: &query.NoResultsError{}},
		&fakeStore{},
	)
	r := newTestRouter(h)

	reqBody := strings.NewReader(`{"question":"Unrelated question"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/query", reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp queryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !resp.NoResults {
		t.Error("expected no_results=true")
	}
}
