-- Retrieval is always scoped to one document, so look chunks up by document.
CREATE INDEX IF NOT EXISTS chunks_document_id_idx ON chunks (document_id);
