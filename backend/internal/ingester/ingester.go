package ingester

import (
	"context"
	"fmt"
	"time"

	"github.com/mibienpanjoe/legalbridge/internal/store"
	"github.com/mibienpanjoe/legalbridge/pkg/config"
)

// IngestResult is returned after a successful ingest.
type IngestResult struct {
	DocumentID string
	ChunkCount int
	ExpiresAt  time.Time // zero when the document never expires
}

// Ingester orchestrates the full PDF → embedding → store pipeline.
type Ingester struct {
	store           store.Store
	embeddingClient EmbeddingClient
	ttl             time.Duration
}

// NewIngester constructs an Ingester with the given Store and EmbeddingClient.
// Documents it ingests expire after ttl; a zero ttl means they never expire.
func NewIngester(s store.Store, ec EmbeddingClient, ttl time.Duration) *Ingester {
	return &Ingester{store: s, embeddingClient: ec, ttl: ttl}
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

	var expiresAt time.Time
	if ing.ttl > 0 {
		expiresAt = time.Now().Add(ing.ttl).UTC().Truncate(time.Second)
	}

	documentID, err := ing.store.WriteDocument(ctx, filename, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("write document: %w", err)
	}

	if err := ing.store.WriteChunks(ctx, documentID, storeChunks); err != nil {
		return nil, fmt.Errorf("write chunks: %w", err)
	}

	return &IngestResult{
		DocumentID: documentID,
		ChunkCount: len(storeChunks),
		ExpiresAt:  expiresAt,
	}, nil
}
