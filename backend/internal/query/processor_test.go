package query

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mibienpanjoe/legalbridge/internal/store"
)

// ─── fakes ───────────────────────────────────────────────────────────────────

type fakeStore struct {
	chunks []store.Chunk
	err    error
}

func (f *fakeStore) WriteDocument(_ context.Context, _ string) (string, error) { return "", nil }
func (f *fakeStore) WriteChunks(_ context.Context, _ string, _ []store.Chunk) error {
	return nil
}
func (f *fakeStore) SimilaritySearch(_ context.Context, _ []float32, _ int) ([]store.Chunk, error) {
	return f.chunks, f.err
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

	_, err := p.Query(context.Background(), "What are the requirements?")

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

	result, err := p.Query(context.Background(), "How long to register?")
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

	result, err := p.Query(context.Background(), "What are the registration requirements?")
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

	_, err := p.Query(context.Background(), "   ")
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
