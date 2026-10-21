package ingester

import (
	"context"
	"fmt"

	"github.com/mibienpanjoe/legalbridge/internal/store"
	"github.com/mibienpanjoe/legalbridge/pkg/config"
)

// IngestResult is returned after a successful ingest.
type IngestResult struct {
	DocumentID string
	ChunkCount int
}

// Ingester orchestrates the full PDF → embedding → store pipeline.
type Ingester struct {
	store           store.Store
	embeddingClient EmbeddingClient
}

// NewIngester constructs an Ingester with the given Store and EmbeddingClient.
func NewIngester(s store.Store, ec EmbeddingClient) *Ingester {
	return &Ingester{store: s, embeddingClient: ec}
}

// Ingest extracts text from fileBytes, chunks it, generates embeddings, and
// persists everything to the store.
func (ing *Ingester) Ingest(ctx context.Context, fileBytes []byte, filename string) (*IngestResult, error) {
	pages, err := ExtractPages(fileBytes) // INV-05: line text is not transformed
	if err != nil {
		return nil, err
	}
	return ing.ingestPages(ctx, pages, filename)
}

// ingestPages is the testable core that accepts pre-extracted pages.
// It is atomic: if any embedding fails, nothing is written to the store.
//
// INV-04: config.EmbeddingModel is the only model name ever passed to Embed.
func (ing *Ingester) ingestPages(ctx context.Context, pages []Page, filename string) (*IngestResult, error) {
	segments := ChunkPages(pages)
	if len(segments) == 0 {
		return nil, &ExtractionFailedError{Reason: "chunking produced no segments"}
	}

	// Embed all chunks before writing anything (atomic guarantee).
	storeChunks := make([]store.Chunk, len(segments))
	for i, seg := range segments {
		vec, err := ing.embeddingClient.Embed(ctx, seg.Text, config.EmbeddingModel) // INV-04
		if err != nil {
			return nil, err // abort — nothing written yet
		}
		storeChunks[i] = store.Chunk{
			Content:    seg.Text,
			Embedding:  vec,
			ChunkIndex: i,
			PageStart:  seg.PageStart,
			PageEnd:    seg.PageEnd,
			Section:    seg.Section,
		}
	}

	documentID, err := ing.store.WriteDocument(ctx, filename)
	if err != nil {
		return nil, fmt.Errorf("write document: %w", err)
	}

	if err := ing.store.WriteChunks(ctx, documentID, storeChunks); err != nil {
		return nil, fmt.Errorf("write chunks: %w", err)
	}

	return &IngestResult{
		DocumentID: documentID,
		ChunkCount: len(storeChunks),
	}, nil
}
