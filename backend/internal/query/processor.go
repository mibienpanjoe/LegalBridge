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
}

// NewQueryProcessor constructs a QueryProcessor with the given dependencies.
func NewQueryProcessor(s store.Store, ec EmbeddingClient, lc LLMClient) *QueryProcessor {
	return &QueryProcessor{store: s, embeddingClient: ec, llmClient: lc}
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
	if strings.TrimSpace(question) == "" {
		return nil, &EmptyQueryError{}
	}

	// Embed the question using the same model as ingestion (INV-04).
	queryVector, err := p.embeddingClient.Embed(ctx, question, config.EmbeddingModel)
	if err != nil {
		return nil, err
	}

	// Retrieve the closest passages of this document above the threshold.
	chunks, err := p.store.SimilaritySearch(ctx, documentID, queryVector, config.RetrievalTopK, config.MinSimilarity)
	if err != nil {
		return nil, err
	}

	// INV-06: do not call the LLM when retrieval is empty.
	if len(chunks) == 0 {
		return nil, &NoResultsError{}
	}

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
		citations = append(citations, Citation{
			Index:        n,
			DocumentName: chunks[n-1].DocumentFilename,
			Passage:      chunks[n-1].Content,
			PageStart:    chunks[n-1].PageStart,
			PageEnd:      chunks[n-1].PageEnd,
			Section:      chunks[n-1].Section,
		})
	}

	// Return citations ordered by index for deterministic output.
	sort.Slice(citations, func(i, j int) bool {
		return citations[i].Index < citations[j].Index
	})
	return citations
}
