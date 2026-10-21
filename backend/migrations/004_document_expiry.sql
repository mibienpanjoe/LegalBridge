-- Uploaded documents are deleted once expires_at has passed (chunks cascade).
-- NULL means the document never expires, e.g. the seeded demo document.
ALTER TABLE documents ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS documents_expires_at_idx ON documents (expires_at) WHERE expires_at IS NOT NULL;
