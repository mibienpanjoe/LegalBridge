# LegalBridge — Software Requirements Specification
Version: v1.0, 2026-03-31

## Normative Vocabulary
- **MUST / REQUIRED**: Absolute requirement. The system fails if not met.
- **SHOULD**: Recommended. Deviation is permitted with documented justification.
- **MAY**: Optional capability.

## Actors

| Actor | Description |
|-------|-------------|
| User | SME owner, startup founder, or legal consultant interacting via the web UI |
| Ingester | Internal component that parses PDFs, chunks text, generates embeddings, and writes to Store |
| QueryProcessor | Internal component that embeds queries, performs similarity search, and synthesizes cited answers |
| Store | Internal component that owns all PostgreSQL/pgvector read and write operations |
| Embedding Provider | External service providing vector embeddings (`BAAI/bge-m3`). HuggingFace Inference API in production; Ollama in local development. |
| LLM Provider | External LLM service for RAG answer synthesis. Groq API (`llama-3.3-70b-versatile`) in production; Ollama in local development. |

---

## Functional Requirements

### FR-010: Document Ingestion

- **FR-011**: The system MUST accept PDF file uploads via `POST /api/ingest` as `multipart/form-data`.
- **FR-012**: The system MUST extract full text content from the uploaded PDF.
- **FR-013**: The system MUST split extracted text into chunks of approximately 500 tokens with a minimum 50-token overlap between consecutive chunks.
- **FR-014**: The system MUST generate a vector embedding for each chunk using the `BAAI/bge-m3` model (1024-dimensional vectors).
- **FR-015**: The system MUST store each chunk's verbatim text, embedding vector, parent document reference, and chunk index in PostgreSQL.
- **FR-016**: The system MUST return HTTP 200 with a JSON body containing `document_id` and `chunk_count` upon successful ingestion.
- **FR-017**: The system MUST reject uploads where the MIME type is not `application/pdf` with HTTP 400 and error code `INVALID_FILE_TYPE`.

### FR-020: Natural Language Query

- **FR-021**: The system MUST accept plain-language questions via `POST /api/query` as a JSON body with a `question` field.
- **FR-022**: The system MUST generate a vector embedding for the incoming question using the same `BAAI/bge-m3` model used during ingestion (INV-04).
- **FR-023**: The system MUST retrieve the top 3 most semantically relevant chunks from the vector database using cosine similarity.
- **FR-024**: The system MUST construct a RAG prompt combining the user's question and the text of the 3 retrieved passages.
- **FR-025**: The system MUST send the RAG prompt to the Groq API (`llama-3.3-70b-versatile`) for answer synthesis.
- **FR-026**: The system MUST return the synthesized answer and the source citations in the response body.

### FR-030: Cited Answer Display

- **FR-031**: Every synthesized answer MUST include at least one citation. A citation consists of the source document name and the verbatim passage text from which the answer was derived.
- **FR-032**: The LLM prompt MUST explicitly instruct the model to base its answer exclusively on the provided passages and not use external knowledge.
- **FR-033**: The LLM prompt MUST instruct the model to respond in the same language as the question.

### FR-040: Pre-Loaded Demo Document

- **FR-041**: At least one real legal document (Ghana Companies Act or Nigeria tax code) MUST be pre-ingested and queryable at system startup without user action.
- **FR-042**: The pre-loaded document MUST support the hackathon demo question set (minimum 5 pre-verified questions with known cited answers).

### FR-050: Error Handling

- **FR-051**: If the Groq API is unavailable or times out (>10 seconds), the system MUST return HTTP 503 with error code `LLM_UNAVAILABLE` and a user-facing fallback message.
- **FR-052**: If PDF text extraction yields an empty string or fails, the system MUST return HTTP 422 with error code `EXTRACTION_FAILED` and MUST NOT create a document record.
- **FR-053**: If cosine similarity search returns no results above the minimum threshold, the system MUST return HTTP 200 with `no_results: true` and MUST NOT call the Groq API.
- **FR-054**: If the embedding API (HuggingFace/Ollama) is unavailable, the system MUST return HTTP 503 with error code `EMBEDDING_UNAVAILABLE` and halt the current operation without partial side effects.

---

## Business Rules

| ID | Rule |
|----|------|
| BR-01 | The system provides legal information, not legal advice or counsel. This MUST be communicated to the user via a disclaimer in the UI. |
| BR-02 | Synthesized answers MUST be grounded exclusively in the ingested document content. Answers supplemented with general legal knowledge not present in retrieved passages are a violation. |
| BR-03 | Citations MUST use verbatim extracted text. Paraphrased or summarized passage text in a citation is a violation. |
| BR-04 | The embedding model identifier (`bge-m3`) MUST be a single shared constant applied to both ingestion (Ingester) and query processing (QueryProcessor). Deviation between models corrupts similarity scores. |

---

## Non-Functional Constraints

### Performance
- End-to-end query latency (question submitted → cited answer displayed): < 5 seconds under normal conditions
- PDF ingestion for a document up to 100 pages: < 30 seconds
- pgvector cosine similarity search (top-3, single document): < 500ms

### Availability
- The system MUST be publicly accessible via HTTPS during the hackathon demo
- Backend: deployed to Railway or Render; Frontend: deployed to Vercel

### Security
- API keys (`HF_API_KEY`, `GROQ_API_KEY`) MUST be stored as environment variables and MUST NOT appear in source code or committed configuration files
- File upload MIME type MUST be validated before processing begins
- All HTTP transport MUST use HTTPS (enforced by deployment platform)

### Data Privacy
- For the MVP, no user-submitted questions, answers, or uploaded documents are persisted beyond the session
- No user accounts, authentication tokens, or PII are collected

### Scalability
- The MVP is scoped to a single pre-loaded document. Multi-document support is out of scope.
- The system MUST handle sequential queries from a single user during the demo

### Portability
- The Go backend MUST be containerized with Docker
- `docker-compose.yml` MUST define the backend service and a PostgreSQL + pgvector service for local development

---

## Error Cases

| ID | Trigger | Required Behavior |
|----|---------|-------------------|
| ERR-011 | Non-PDF file uploaded | HTTP 400: `INVALID_FILE_TYPE`, "Only PDF files are supported." |
| ERR-012 | PDF text extraction returns empty string | HTTP 422: `EXTRACTION_FAILED`, "Could not extract text from this PDF." |
| ERR-021 | `question` field is empty or missing | HTTP 400: `EMPTY_QUERY`, "Question must not be empty." |
| ERR-022 | Similarity search returns zero results above threshold | HTTP 200: `no_results: true`, "No relevant passages found for this question." |
| ERR-023 | Groq API timeout (>10s) or error | HTTP 503: `LLM_UNAVAILABLE`, "The answer service is temporarily unavailable." |
| ERR-024 | Embedding API (HuggingFace/Ollama) timeout or error | HTTP 503: `EMBEDDING_UNAVAILABLE`, "The embedding service is temporarily unavailable." |
| ERR-025 | PostgreSQL connection failure | HTTP 503: `DATABASE_UNAVAILABLE`, "The database is temporarily unavailable." |
