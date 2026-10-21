package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"
	pgxvector "github.com/pgvector/pgvector-go/pgx"
)

const embeddingDimension = 1024

// PostgresStore is the production Store backed by PostgreSQL + pgvector.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore opens a connection pool and registers the pgvector type on
// every new connection. Returns DatabaseUnavailableError on failure.
func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, &DatabaseUnavailableError{Cause: err}
	}

	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvector.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, &DatabaseUnavailableError{Cause: err}
	}

	return &PostgresStore{pool: pool}, nil
}

// WriteDocument inserts a document record and returns its generated UUID.
func (s *PostgresStore) WriteDocument(ctx context.Context, filename string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO documents (filename) VALUES ($1) RETURNING id`,
		filename,
	).Scan(&id)
	if err != nil {
		return "", &DatabaseUnavailableError{Cause: err}
	}
	return id, nil
}

// WriteChunks persists all chunks for a document atomically. It validates
// every embedding dimension before opening a transaction — if any chunk has
// a dimension other than 1024, it returns immediately without writing.
func (s *PostgresStore) WriteChunks(ctx context.Context, documentID string, chunks []Chunk) error {
	for i, c := range chunks {
		if len(c.Embedding) != embeddingDimension {
			return &InvalidEmbeddingDimensionError{
				Got:  len(c.Embedding),
				Want: embeddingDimension,
			}
		}
		_ = i
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return &DatabaseUnavailableError{Cause: err}
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for _, c := range chunks {
		_, err := tx.Exec(ctx,
			`INSERT INTO chunks (document_id, content, embedding, chunk_index, page_start, page_end, section)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			documentID,
			c.Content,
			pgvector.NewVector(c.Embedding),
			c.ChunkIndex,
			c.PageStart,
			c.PageEnd,
			c.Section,
		)
		if err != nil {
			return &DatabaseUnavailableError{Cause: err}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return &DatabaseUnavailableError{Cause: fmt.Errorf("commit: %w", err)}
	}
	return nil
}

// SimilaritySearch returns up to topK chunks of documentID whose cosine
// similarity to vector is at least minSimilarity, ordered nearest-first.
// All queries are parameterized. DocumentFilename is populated via JOIN with
// the documents table.
//
// The document's chunks are materialized first so the distance is computed
// exactly over that document only. Filtering after an approximate HNSW scan
// across all documents could silently drop relevant passages.
func (s *PostgresStore) SimilaritySearch(ctx context.Context, documentID string, vector []float32, topK int, minSimilarity float64) ([]Chunk, error) {
	rows, err := s.pool.Query(ctx,
		`WITH doc_chunks AS MATERIALIZED (
		     SELECT c.id, c.document_id, c.content, c.embedding, c.chunk_index,
		            c.page_start, c.page_end, c.section, c.created_at,
		            1 - (c.embedding <=> $2) AS similarity
		     FROM chunks c
		     WHERE c.document_id = $1
		 )
		 SELECT dc.id, dc.document_id, d.filename, dc.content, dc.embedding, dc.chunk_index,
		        dc.page_start, dc.page_end, dc.section, dc.created_at, dc.similarity
		 FROM doc_chunks dc
		 JOIN documents d ON d.id = dc.document_id
		 WHERE dc.similarity >= $4
		 ORDER BY dc.similarity DESC
		 LIMIT $3`,
		documentID,
		pgvector.NewVector(vector),
		topK,
		minSimilarity,
	)
	if err != nil {
		return nil, &DatabaseUnavailableError{Cause: err}
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var c Chunk
		var vec pgvector.Vector
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.DocumentFilename, &c.Content, &vec, &c.ChunkIndex,
			&c.PageStart, &c.PageEnd, &c.Section, &c.CreatedAt, &c.Similarity); err != nil {
			return nil, &DatabaseUnavailableError{Cause: err}
		}
		c.Embedding = vec.Slice()
		chunks = append(chunks, c)
	}
	if err := rows.Err(); err != nil {
		return nil, &DatabaseUnavailableError{Cause: err}
	}
	return chunks, nil
}

// Ping verifies that the database connection is alive.
func (s *PostgresStore) Ping(ctx context.Context) error {
	if err := s.pool.Ping(ctx); err != nil {
		return &DatabaseUnavailableError{Cause: err}
	}
	return nil
}
