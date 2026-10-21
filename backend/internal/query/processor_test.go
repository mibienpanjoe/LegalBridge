package query

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mibienpanjoe/legalbridge/internal/store"
	"github.com/mibienpanjoe/legalbridge/pkg/config"
)

// ─── fakes ───────────────────────────────────────────────────────────────────

const testDocID = "3f4a7b2c-1e6d-4a89-b3c2-9d1e5f7a8b0c"

type fakeStore struct {
	chunks []store.Chunk
	err    error

	// recorded arguments of the last SimilaritySearch call
	gotDocID         string
	gotMinSimilarity float64

	contentsCalls int  // number of ChunkContents calls
	missing       bool // DocumentExists reports false
}

func (f *fakeStore) WriteDocument(_ context.Context, _ string, _ time.Time) (string, error) {
	return "", nil
}
func (f *fakeStore) DocumentExists(_ context.Context, _ string) (bool, error) { return !f.missing, nil }
func (f *fakeStore) DeleteExpiredDocuments(_ context.Context) (int64, error)  { return 0, nil }
func (f *fakeStore) WriteChunks(_ context.Context, _ string, _ []store.Chunk) error {
	return nil
}
func (f *fakeStore) SimilaritySearch(_ context.Context, documentID string, _ []float32, _ int, minSimilarity float64) ([]store.Chunk, error) {
	f.gotDocID = documentID
	f.gotMinSimilarity = minSimilarity
	return f.chunks, f.err
}
func (f *fakeStore) ChunkContents(_ context.Context, _ string) ([]store.Chunk, error) {
	f.contentsCalls++
	return f.chunks, nil
}
func (f *fakeStore) Ping(_ context.Context) error { return nil }

type fakeEmbedder struct{}

func (f *fakeEmbedder) Embed(_ context.Context, _, _ string) ([]float32, error) {
	return make([]float32, 1024), nil
}

type fakeLLM struct{ response string }

func (f *fakeLLM) Complete(_ context.Context, _, _ string) (string, error) {
	return f.response, nil
}

// ─── prompt invariant tests ──────────────────────────────────────────────────

// TestBuildRAGPrompt_InvariantStrings verifies that BuildRAGPrompt includes the
// three invariant instructions required by INV-01, INV-03, and INV-07.
func TestBuildRAGPrompt_InvariantStrings(t *testing.T) {
	chunks := []store.Chunk{
		{Content: "passage one", DocumentFilename: "doc.pdf"},
		{Content: "passage two", DocumentFilename: "doc.pdf"},
	}
	prompt := BuildRAGPrompt(chunks)

	// INV-01: grounding instruction must be present.
	if !strings.Contains(prompt, "Answer using only the provided passages. Do not use external knowledge.") {
		t.Error("INV-01 violated: grounding instruction missing from prompt")
	}

	// INV-03: no legal advice instruction must be present.
	if !strings.Contains(prompt, "do NOT provide legal advice") {
		t.Error("INV-03 violated: no-legal-advice instruction missing from prompt")
	}

	// INV-07: language matching instruction must be present.
	if !strings.Contains(prompt, "Respond in the same language as the user's question") {
		t.Error("INV-07 violated: language instruction missing from prompt")
	}

	// Passages must be embedded.
	if !strings.Contains(prompt, "[1] passage one") {
		t.Error("passage [1] missing from prompt")
	}
	if !strings.Contains(prompt, "[2] passage two") {
		t.Error("passage [2] missing from prompt")
	}
}

// ─── INV-06 test ─────────────────────────────────────────────────────────────

// TestQuery_INV06_EmptyRetrieval verifies that Query returns NoResultsError
// immediately when SimilaritySearch returns zero chunks, without calling the LLM.
func TestQuery_INV06_EmptyRetrieval(t *testing.T) {
	llm := &fakeLLM{response: "this should never be called"}
	callCount := 0
	sentinel := &fakeLLM{} // will panic if Complete is called
	_ = sentinel

	p := NewQueryProcessor(
		&fakeStore{chunks: nil},
		&fakeEmbedder{},
		&countingLLM{inner: llm, count: &callCount},
	)

	_, err := p.Query(context.Background(), testDocID, "What are the requirements?")

	var noResults *NoResultsError
	if !errors.As(err, &noResults) {
		t.Fatalf("expected NoResultsError, got %T: %v", err, err)
	}
	if callCount != 0 {
		t.Errorf("INV-06 violated: LLM was called %d time(s) with empty retrieval", callCount)
	}
}

// ─── INV-02 test ─────────────────────────────────────────────────────────────

// TestQuery_INV02_UncitedResponse verifies that when the LLM returns an answer
// with no [N] citation markers, Query returns the fallback message instead.
func TestQuery_INV02_UncitedResponse(t *testing.T) {
	chunks := []store.Chunk{
		{Content: "A foreign company must register within 28 days.", DocumentFilename: "act.pdf"},
	}
	p := NewQueryProcessor(
		&fakeStore{chunks: chunks},
		&fakeEmbedder{},
		&fakeLLM{response: "A foreign company must register within 28 days."}, // no [1] citation
	)

	result, err := p.Query(context.Background(), testDocID, "How long to register?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Answer != citationFallback {
		t.Errorf("INV-02 violated: expected fallback message, got %q", result.Answer)
	}
	if len(result.Citations) != 0 {
		t.Errorf("expected zero citations in fallback, got %d", len(result.Citations))
	}
}

// TestQuery_CitedResponse verifies the happy path: a cited LLM response is
// returned with citations correctly mapped to chunks.
func TestQuery_CitedResponse(t *testing.T) {
	chunks := []store.Chunk{
		{Content: "A foreign company must register within 28 days.", DocumentFilename: "act.pdf"},
		{Content: "Required documents include a certified copy of the charter.", DocumentFilename: "act.pdf"},
	}
	p := NewQueryProcessor(
		&fakeStore{chunks: chunks},
		&fakeEmbedder{},
		&fakeLLM{response: "According to the Act, registration is required within 28 days [1] and certified documents are needed [2]."},
	)

	result, err := p.Query(context.Background(), testDocID, "What are the registration requirements?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Citations) != 2 {
		t.Fatalf("expected 2 citations, got %d", len(result.Citations))
	}
	if result.Citations[0].Index != 1 || result.Citations[0].DocumentName != "act.pdf" {
		t.Errorf("unexpected citation[0]: %+v", result.Citations[0])
	}
	if result.Citations[1].Index != 2 {
		t.Errorf("unexpected citation[1] index: %d", result.Citations[1].Index)
	}
}

// TestQuery_EmptyQuestion verifies that an empty question returns EmptyQueryError.
func TestQuery_EmptyQuestion(t *testing.T) {
	p := NewQueryProcessor(&fakeStore{}, &fakeEmbedder{}, &fakeLLM{})

	_, err := p.Query(context.Background(), testDocID, "   ")
	var emptyErr *EmptyQueryError
	if !errors.As(err, &emptyErr) {
		t.Fatalf("expected EmptyQueryError, got %T: %v", err, err)
	}
}

// ─── helper ──────────────────────────────────────────────────────────────────

type countingLLM struct {
	inner LLMClient
	count *int
}

func (c *countingLLM) Complete(ctx context.Context, sys, user string) (string, error) {
	*c.count++
	return c.inner.Complete(ctx, sys, user)
}

// TestQuery_ScopedToDocumentWithThreshold verifies that retrieval is limited to
// the requested document and filtered by config.MinSimilarity (INV-06).
func TestQuery_ScopedToDocumentWithThreshold(t *testing.T) {
	fs := &fakeStore{chunks: []store.Chunk{{Content: "passage", DocumentFilename: "doc.pdf"}}}
	p := NewQueryProcessor(fs, &fakeEmbedder{}, &fakeLLM{response: "Answer [1]."})

	if _, err := p.Query(context.Background(), testDocID, "question?"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fs.gotDocID != testDocID {
		t.Errorf("expected search scoped to %q, got %q", testDocID, fs.gotDocID)
	}
	if fs.gotMinSimilarity != config.MinSimilarity {
		t.Errorf("expected minSimilarity %v, got %v", config.MinSimilarity, fs.gotMinSimilarity)
	}
}

// TestBuildRAGPrompt_PassageLocation verifies that passages are labelled with
// their article and pages so the model can name what it cites.
func TestBuildRAGPrompt_PassageLocation(t *testing.T) {
	prompt := BuildRAGPrompt([]store.Chunk{
		{Content: "first", Section: "Article 12", PageStart: 4, PageEnd: 4},
		{Content: "second", Section: "Article 13–15", PageStart: 4, PageEnd: 5},
		{Content: "third"},
	})
	for _, want := range []string{
		"[1] (Article 12, p. 4) first",
		"[2] (Article 13–15, p. 4–5) second",
		"[3] third",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

// TestQuery_DocumentNotFound verifies that a missing or expired document is
// reported before any embedding or LLM call.
func TestQuery_DocumentNotFound(t *testing.T) {
	calls := 0
	p := NewQueryProcessor(&fakeStore{missing: true}, &fakeEmbedder{}, &countingLLM{inner: &fakeLLM{}, count: &calls})

	_, err := p.Query(context.Background(), testDocID, "question ?")
	var notFound *DocumentNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("expected DocumentNotFoundError, got %T: %v", err, err)
	}
	if calls != 0 {
		t.Errorf("LLM called %d time(s) for a missing document", calls)
	}
}

// TestQueryWithProgress_Stages verifies the reported stage order and that the
// sources stage carries the passages, numbered as in the prompt.
func TestQueryWithProgress_Stages(t *testing.T) {
	fs := &fakeStore{chunks: []store.Chunk{{Content: "Article 42 : un mois.", Section: "Article 42", DocumentFilename: "code.pdf"}}}
	p := NewQueryProcessor(fs, &fakeEmbedder{}, &fakeLLM{response: "Un mois [1]."})

	var got []Stage
	if _, err := p.QueryWithProgress(context.Background(), testDocID, "Durée ?", func(s Stage) { got = append(got, s) }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := []StageName{}
	for _, s := range got {
		names = append(names, s.Name)
	}
	want := []StageName{StageSearching, StageSources, StageGenerating}
	if fmt.Sprint(names) != fmt.Sprint(want) {
		t.Fatalf("stages %v, want %v", names, want)
	}
	if len(got[1].Passages) != 1 || got[1].Passages[0].Index != 1 || got[1].Passages[0].Section != "Article 42" {
		t.Errorf("unexpected sources passages %+v", got[1].Passages)
	}
}
