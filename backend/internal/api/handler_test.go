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
	"time"

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
	stages []query.Stage // reported by QueryWithProgress before returning
}

func (f *fakeQuerySvc) Query(_ context.Context, _, _ string) (*query.QueryResult, error) {
	return f.result, f.err
}

func (f *fakeQuerySvc) QueryWithProgress(_ context.Context, _, _ string, progress query.ProgressFunc) (*query.QueryResult, error) {
	for _, s := range f.stages {
		progress(s)
	}
	return f.result, f.err
}

type fakeStore struct{ pingErr error }

func (f *fakeStore) WriteDocument(_ context.Context, _ string, _ time.Time) (string, error) {
	return "", nil
}
func (f *fakeStore) DocumentExists(_ context.Context, _ string) (bool, error)       { return true, nil }
func (f *fakeStore) DeleteExpiredDocuments(_ context.Context) (int64, error)        { return 0, nil }
func (f *fakeStore) WriteChunks(_ context.Context, _ string, _ []store.Chunk) error { return nil }
func (f *fakeStore) SimilaritySearch(_ context.Context, _ string, _ []float32, _ int, _ float64) ([]store.Chunk, error) {
	return nil, nil
}
func (f *fakeStore) ChunkContents(_ context.Context, _ string) ([]store.Chunk, error) {
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

	reqBody := strings.NewReader(`{"document_id":"3f4a7b2c-1e6d-4a89-b3c2-9d1e5f7a8b0c","question":"How long to register?"}`)
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

	reqBody := strings.NewReader(`{"document_id":"3f4a7b2c-1e6d-4a89-b3c2-9d1e5f7a8b0c","question":"Unrelated question"}`)
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

// TestHandleQuery_RequiresDocumentID verifies that a query without a valid
// document_id is rejected, so retrieval can never span other documents.
func TestHandleQuery_RequiresDocumentID(t *testing.T) {
	h := NewAPIHandler(&fakeIngestor{}, &fakeQuerySvc{}, &fakeStore{})
	r := newTestRouter(h)

	for _, body := range []string{
		`{"question":"How long to register?"}`,
		`{"document_id":"not-a-uuid","question":"How long to register?"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("body %s: expected 400, got %d", body, w.Code)
		}
	}
}

// TestHandleQuery_DocumentNotFound verifies that an expired or unknown
// document is reported as 404 DOCUMENT_NOT_FOUND.
func TestHandleQuery_DocumentNotFound(t *testing.T) {
	h := NewAPIHandler(&fakeIngestor{}, &fakeQuerySvc{err: &query.DocumentNotFoundError{}}, &fakeStore{})
	r := newTestRouter(h)

	body := `{"document_id":"3f4a7b2c-1e6d-4a89-b3c2-9d1e5f7a8b0c","question":"Encore là ?"}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "DOCUMENT_NOT_FOUND") {
		t.Errorf("expected DOCUMENT_NOT_FOUND code, got %s", w.Body.String())
	}
}

// sseEvents parses a text/event-stream body into (event, data) pairs.
func sseEvents(t *testing.T, body string) [][2]string {
	t.Helper()
	var events [][2]string
	for _, block := range strings.Split(strings.TrimSpace(body), "\n\n") {
		var ev, data string
		for _, line := range strings.Split(block, "\n") {
			if v, ok := strings.CutPrefix(line, "event: "); ok {
				ev = v
			} else if v, ok := strings.CutPrefix(line, "data: "); ok {
				data = v
			}
		}
		events = append(events, [2]string{ev, data})
	}
	return events
}

func postStream(t *testing.T, svc *fakeQuerySvc, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := newTestRouter(NewAPIHandler(&fakeIngestor{}, svc, &fakeStore{}))
	req := httptest.NewRequest(http.MethodPost, "/api/query/stream", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

const streamBody = `{"document_id":"3f4a7b2c-1e6d-4a89-b3c2-9d1e5f7a8b0c","question":"Durée de l'essai ?"}`

// TestHandleQueryStream_Stages verifies the event sequence of a successful
// streamed query: stages in order, then the validated answer.
func TestHandleQueryStream_Stages(t *testing.T) {
	svc := &fakeQuerySvc{
		stages: []query.Stage{
			{Name: query.StageSearching},
			{Name: query.StageSources, Passages: []query.Citation{{Index: 1, Passage: "Article 42 : …", Section: "Article 42"}}},
			{Name: query.StageGenerating},
		},
		result: &query.QueryResult{Answer: "Un mois [1].", Citations: []query.Citation{{Index: 1, Passage: "Article 42 : …"}}},
	}
	w := postStream(t, svc, streamBody)

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", ct)
	}
	events := sseEvents(t, w.Body.String())
	want := []string{"stage", "stage", "stage", "answer"}
	if len(events) != len(want) {
		t.Fatalf("expected %d events, got %d: %v", len(want), len(events), events)
	}
	for i, ev := range events {
		if ev[0] != want[i] {
			t.Errorf("event %d: got %q, want %q", i, ev[0], want[i])
		}
	}
	if !strings.Contains(events[1][1], `"stage":"sources"`) || !strings.Contains(events[1][1], `"section":"Article 42"`) {
		t.Errorf("sources stage missing passages: %s", events[1][1])
	}
	var answer queryResponse
	if err := json.Unmarshal([]byte(events[3][1]), &answer); err != nil || answer.Answer != "Un mois [1]." || len(answer.Citations) != 1 {
		t.Errorf("unexpected answer event %s (%v)", events[3][1], err)
	}
}

// TestHandleQueryStream_Errors verifies that pipeline errors become an error
// event carrying the API error code, and that no_results is an answer event.
func TestHandleQueryStream_Errors(t *testing.T) {
	w := postStream(t, &fakeQuerySvc{err: &query.LLMUnavailableError{Reason: "timeout"}}, streamBody)
	events := sseEvents(t, w.Body.String())
	if len(events) != 1 || events[0][0] != "error" || !strings.Contains(events[0][1], "LLM_UNAVAILABLE") {
		t.Errorf("expected one LLM_UNAVAILABLE error event, got %v", events)
	}

	w = postStream(t, &fakeQuerySvc{err: &query.NoResultsError{}}, streamBody)
	events = sseEvents(t, w.Body.String())
	if len(events) != 1 || events[0][0] != "answer" || !strings.Contains(events[0][1], `"no_results":true`) {
		t.Errorf("expected one no_results answer event, got %v", events)
	}

	w = postStream(t, &fakeQuerySvc{}, `{"question":"no document"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 before streaming for an invalid body, got %d", w.Code)
	}
}
