-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "vector";

-- Stores metadata for each ingested document
CREATE TABLE IF NOT EXISTS documents (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    filename   TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Stores text chunks and their 1024-dimensional bge-m3 embeddings
CREATE TABLE IF NOT EXISTS chunks (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID        NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    content     TEXT        NOT NULL,
    embedding   vector(1024) NOT NULL,
    chunk_index INTEGER     NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Drop legacy IVFFlat index if it exists; IVFFlat requires ~3×lists rows to
-- produce valid centroids and returns 0 results on small datasets.
DROP INDEX IF EXISTS chunks_embedding_idx;

-- HNSW index for cosine similarity — works correctly at any dataset size,
-- including single-document uploads.
CREATE INDEX IF NOT EXISTS chunks_embedding_hnsw_idx
    ON chunks USING hnsw (embedding vector_cosine_ops);
