package query

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/mibienpanjoe/legalbridge/internal/store"
	"github.com/mibienpanjoe/legalbridge/pkg/config"
)

// EmbeddingClient generates a vector embedding for a piece of text.
// This interface is satisfied by *ingester.OllamaClient and
// *ingester.HuggingFaceClient via Go's implicit interface matching.
type EmbeddingClient interface {
	Embed(ctx context.Context, text, model string) ([]float32, error)
}

// Citation is a source passage cited in the answer.
type Citation struct {
	Index        int    // 1-based index matching the [N] marker in the answer
	DocumentName string // original filename of the source document
	Passage      string // verbatim chunk text (INV-05 guarantees this is unmodified)
	PageStart    int    // first source page of the passage (0 if unknown)
	PageEnd      int    // last source page of the passage
	Section      string // article/section label, e.g. "Article 12"; may be empty
}

// StageName identifies a milestone of the query pipeline.
type StageName string

const (
	StageSearching  StageName = "searching"  // embedding the question and searching the document
	StageSources    StageName = "sources"    // passages selected; Stage.Passages is set
	StageGenerating StageName = "generating" // waiting for the LLM to write the cited answer
)

// Stage is reported to a ProgressFunc as the pipeline advances.
type Stage struct {
	Name StageName
	// Passages are the retrieved passages, numbered as in the RAG prompt.
	// They are verbatim chunks, not citations: which ones the answer cites is
	// only known once the response has been validated (INV-02).
	Passages []Citation
}

// ProgressFunc receives pipeline stages while a query runs. It is called on
// the querying goroutine and must not block for long.
type ProgressFunc func(Stage)

// QueryResult is returned by a successful Query call.
type QueryResult struct {
	Answer    string
	Citations []Citation
}

const citationFallback = "I found relevant passages but could not produce a properly cited answer. " +
	"Please rephrase your question and try again."

var citationRe = regexp.MustCompile(`\[(\d+)\]`)

// QueryProcessor owns the answer-generation pipeline from question to cited response.
type QueryProcessor struct {
	store           store.Store
	embeddingClient EmbeddingClient
	llmClient       LLMClient
	lexical         *lexicalCache
}

// lexicalCacheSize is the number of documents whose BM25 index is kept in memory.
const lexicalCacheSize = 64

// NewQueryProcessor constructs a QueryProcessor with the given dependencies.
func NewQueryProcessor(s store.Store, ec EmbeddingClient, lc LLMClient) *QueryProcessor {
	return &QueryProcessor{
		store:           s,
		embeddingClient: ec,
		llmClient:       lc,
		lexical:         newLexicalCache(lexicalCacheSize),
	}
}

// Query runs the full RAG pipeline for the given question against a single
// document. Passages from other documents are never retrieved.
//
// Invariants enforced here:
//   - INV-04: config.EmbeddingModel is the only model name passed to Embed.
//   - INV-06: returns NoResultsError immediately when SimilaritySearch returns
//     no chunk above config.MinSimilarity; the LLM is never called with an
//     empty context.
//   - INV-02: validates that the LLM response contains at least one [N] citation;
//     returns a fallback message if not.
func (p *QueryProcessor) Query(ctx context.Context, documentID, question string) (*QueryResult, error) {
	return p.QueryWithProgress(ctx, documentID, question, nil)
}

// QueryWithProgress is Query, reporting each pipeline stage to progress (which
// may be nil). The answer itself is only returned once fully validated, so
// streaming progress never weakens INV-02 or INV-06.
func (p *QueryProcessor) QueryWithProgress(ctx context.Context, documentID, question string, progress ProgressFunc) (*QueryResult, error) {
	report := func(s Stage) {
		if progress != nil {
			progress(s)
		}
	}

	if strings.TrimSpace(question) == "" {
		return nil, &EmptyQueryError{}
	}

	exists, err := p.store.DocumentExists(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &DocumentNotFoundError{}
	}

	report(Stage{Name: StageSearching})

	// Embed the question using the same model as ingestion (INV-04).
	queryVector, err := p.embeddingClient.Embed(ctx, question, config.EmbeddingModel)
	if err != nil {
		return nil, err
	}

	// Retrieve the closest passages of this document above the threshold.
	candidates, err := p.store.SimilaritySearch(ctx, documentID, queryVector, config.RetrievalCandidates, config.MinSimilarity)
	if err != nil {
		return nil, err
	}

	// INV-06: do not call the LLM when retrieval is empty.
	if len(candidates) == 0 {
		return nil, &NoResultsError{}
	}

	// Hybrid rerank: promote candidates that share rare terms with the question.
	chunks := candidates
	if len(candidates) > config.RetrievalTopK {
		ix, err := p.lexicalIndex(ctx, documentID)
		if err != nil {
			return nil, err
		}
		chunks = fuseRankings(candidates, ix, question, config.RetrievalTopK)
	}

	passages := make([]Citation, len(chunks))
	for i, c := range chunks {
		passages[i] = citationFor(i+1, c)
	}
	report(Stage{Name: StageSources, Passages: passages})
	report(Stage{Name: StageGenerating})

	systemPrompt := BuildRAGPrompt(chunks)
	answer, err := p.llmClient.Complete(ctx, systemPrompt, question)
	if err != nil {
		return nil, err
	}

	// INV-02: validate that the response cites at least one passage.
	citations := parseCitations(answer, chunks)
	if len(citations) == 0 {
		return &QueryResult{Answer: citationFallback, Citations: nil}, nil
	}

	return &QueryResult{Answer: answer, Citations: citations}, nil
}

// citationFor builds the citation numbered index for chunk.
func citationFor(index int, c store.Chunk) Citation {
	return Citation{
		Index:        index,
		DocumentName: c.DocumentFilename,
		Passage:      c.Content,
		PageStart:    c.PageStart,
		PageEnd:      c.PageEnd,
		Section:      c.Section,
	}
}

// lexicalIndex returns the document's BM25 index, building and caching it on
// first use.
func (p *QueryProcessor) lexicalIndex(ctx context.Context, documentID string) (*lexicalIndex, error) {
	if ix, ok := p.lexical.get(documentID); ok {
		return ix, nil
	}
	contents, err := p.store.ChunkContents(ctx, documentID)
	if err != nil {
		return nil, err
	}
	ix := newLexicalIndex(contents)
	p.lexical.put(documentID, ix)
	return ix, nil
}

// parseCitations extracts [N] references from the LLM answer and maps each
// back to the corresponding chunk. References out of range are ignored.
func parseCitations(answer string, chunks []store.Chunk) []Citation {
	matches := citationRe.FindAllStringSubmatch(answer, -1)
	seen := map[int]bool{}
	var citations []Citation

	for _, m := range matches {
		n, _ := strconv.Atoi(m[1])
		if seen[n] || n < 1 || n > len(chunks) {
			continue
		}
		seen[n] = true
		citations = append(citations, citationFor(n, chunks[n-1]))
	}

	// Return citations ordered by index for deterministic output.
	sort.Slice(citations, func(i, j int) bool {
		return citations[i].Index < citations[j].Index
	})
	return citations
}
