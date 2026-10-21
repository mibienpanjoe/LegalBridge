-- Where each chunk sits in its source PDF, so citations can point to
-- "Article 12, p. 4". Defaults keep chunks ingested before this migration valid.
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS page_start INTEGER NOT NULL DEFAULT 0;
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS page_end   INTEGER NOT NULL DEFAULT 0;
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS section    TEXT    NOT NULL DEFAULT '';
