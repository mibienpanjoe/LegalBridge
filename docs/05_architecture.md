# LegalBridge — System Architecture
Version: v1.0, 2026-03-31

## Architectural Style

**Modular monolith (Go) + separate SPA frontend (Next.js), deployed as two independent units.**

The Go backend is a single binary with clearly bounded internal packages. It does not use microservices or message queues. The Next.js frontend is a separate deployment on Vercel and communicates with the backend via HTTP.

**Why not microservices:**
The MVP handles a single pre-loaded document and sequential demo queries. Distributing the ingestion and query pipelines across services would add network hops, shared-state coordination, and deployment complexity that provide no benefit at this scale. A modular monolith gives clean component boundaries without distributed system overhead.

**Why pgvector over a dedicated vector database (Pinecone, Weaviate):**
At MVP scale — one document, a few hundred chunks — pgvector running inside PostgreSQL is more than sufficient. It eliminates an entire infrastructure dependency. If scale demands it post-hackathon, the Store interface can be replaced behind the same contract.

---

## Component Architecture

### Ingester (`internal/ingester`)

**Responsibility:** Own the document onboarding pipeline from PDF bytes to stored embeddings.

**Owned invariants:** INV-04 (embedding model consistency), INV-05 (passage fidelity)

**Inputs:** `[]byte` (PDF file content), `string` (original filename)
**Outputs:** `IngestResult{DocumentID, ChunkCount}` or error

**Key behaviors:**
1. Extract full text from PDF using `pdfcpu` (Go library — no external process)
2. Split text into ~500-token chunks with 50-token overlap using a word-boundary tokenizer
3. For each chunk, call `EmbeddingClient.Embed(chunk.Content, config.EmbeddingModel)`
4. Batch-write all chunks with their embeddings to `Store.WriteChunks()`
5. On any embedding API failure: return error immediately without partial writes (EXC-04)

**Must NOT:**
- Modify chunk text between extraction and storage (INV-05)
- Use any embedding model other than `config.EmbeddingModel` (INV-04)
- Write a partial set of chunks if the embedding loop fails midway

---

### QueryProcessor (`internal/query`)

**Responsibility:** Own the answer generation pipeline from question string to cited response.

**Owned invariants:** INV-01, INV-02, INV-03, INV-06, INV-07

**Inputs:** `string` (user question)
**Outputs:** `QueryResult{Answer, Citations}` or error

**Key behaviors:**
1. Call `EmbeddingClient.Embed(question, config.EmbeddingModel)` → `queryVector`
2. Call `Store.SimilaritySearch(queryVector, topK=3)` → `[]Chunk`
3. **If `len(chunks) == 0`**: return `NoResultsError` immediately without calling Claude (INV-06)
4. Construct RAG prompt (see Prompt Template below)
5. Call Groq API: `POST /openai/v1/chat/completions` with the constructed prompt
6. Validate response: check that `Citations` field is non-empty (INV-02)
7. **If no citations in response**: return fallback message (EXC-01)
8. Return `QueryResult{Answer, Citations}`

**Prompt Template:**
```
System: You are a legal document retrieval assistant for West African businesses.
You provide information based exclusively on the legal documents provided to you.
You do NOT provide legal advice, legal opinions, or legal counsel.
You MUST cite the specific passage(s) from the provided context that support each claim.
Respond in the same language as the user's question.

Context passages:
[1] <passage_1_text>
[2] <passage_2_text>
[3] <passage_3_text>

User question: <question>

Instructions: Answer the question using only the context passages above.
For each claim, reference the passage number (e.g., "[1]").
Do not use any knowledge beyond what is provided in the passages.
```

**Must NOT:**
- Call Claude with an empty or zero-passage context (INV-06)
- Return an answer without citations (INV-02)
- Omit the "no legal advice" instruction from the system prompt (INV-03)
- Use a different embedding model than Ingester (INV-04)

---

### Store (`internal/store`)

**Responsibility:** Own all PostgreSQL read and write operations. Single source of persistence truth.

**No invariant ownership** — enforces structural data constraints only.

**Interface:**
```go
type Store interface {
    WriteDocument(ctx, filename string) (documentID string, err error)
    WriteChunks(ctx, documentID string, chunks []Chunk) error
    SimilaritySearch(ctx context.Context, vector []float32, topK int) ([]Chunk, error)
    Ping(ctx context.Context) error
}
```

**Structural enforcements:**
- `WriteChunks` MUST fail if any chunk has `len(embedding) != 1024`
- `SimilaritySearch` uses `<=>` (cosine distance) with a parameterized limit
- All queries are parameterized — no string interpolation

**Must NOT:**
- Expose any method that returns chunks without filtering by dimension
- Hold business logic or call external APIs

---

### APIHandler (`internal/api`)

**Responsibility:** Own the HTTP surface. Route requests, translate component errors to HTTP responses, validate startup configuration.

**Owned invariants:** INV-08 (API key validation at startup)

**Routes:**
| Method | Path | Handler |
|--------|------|---------|
| POST | `/api/ingest` | `handleIngest` → Ingester |
| POST | `/api/query` | `handleQuery` → QueryProcessor |
| GET | `/api/health` | `handleHealth` → Store.Ping |

**Error translation:**
| Component error | HTTP status | Response body |
|-----------------|-------------|---------------|
| `InvalidFileTypeError` | 400 | `INVALID_FILE_TYPE` |
| `ExtractionFailedError` | 422 | `EXTRACTION_FAILED` |
| `EmptyQueryError` | 400 | `EMPTY_QUERY` |
| `NoResultsError` | 200 | `no_results: true` |
| `LLMUnavailableError` | 503 | `LLM_UNAVAILABLE` |
| `EmbeddingUnavailableError` | 503 | `EMBEDDING_UNAVAILABLE` |
| `DatabaseUnavailableError` | 503 | `DATABASE_UNAVAILABLE` |

**Startup behavior:**
- `pkg/config.Load()` is called before the HTTP server starts
- If `HF_API_KEY` or `GROQ_API_KEY` is empty, `log.Fatal` is called immediately (INV-08, EXC-07)
- CORS middleware is configured to allow the Vercel frontend domain

---

### UI (`frontend/`)

**Responsibility:** Own the user-facing interface. Document upload, question input, answer display, citation rendering.

**Must display:**
- Legal disclaimer: "LegalBridge provides legal information, not legal advice. Consult a qualified lawyer for legal counsel."
- Source passages used in each answer (citations)
- Loading states during ingestion and query processing
- Specific error messages from API error codes

---

## Data Architecture

### Entity Model

```sql
-- Stores metadata for each ingested document
CREATE TABLE documents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename    TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Stores text chunks and their vector embeddings
CREATE TABLE chunks (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id  UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    content      TEXT NOT NULL,
    embedding    vector(1024) NOT NULL,
    chunk_index  INTEGER NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for cosine similarity search
CREATE INDEX ON chunks USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);
```

### Key Constraints
- `chunks.content` is immutable after write (INV-05 enforcement at the application layer)
- `chunks.embedding` dimension MUST be 1024 (`bge-m3` output dimension)
- `chunks.document_id` cascades on delete — removing a document removes all its chunks

---

## Flow Architecture

### Ingestion Flow
```
User → Upload PDF (via UI)
    ↓
POST /api/ingest (multipart/form-data, field: "file")
    ↓
APIHandler.handleIngest()
    ↓
Ingester.Ingest(fileBytes, filename)
    ├─► pdfcpu.ExtractText(fileBytes)          → rawText
    │       └─► empty result → ExtractionFailedError
    ├─► tokenize(rawText, chunkSize=500, overlap=50) → []chunkText
    ├─► for each chunkText:
    │       EmbeddingClient.Embed(chunkText, EmbeddingModel) → vector
    │           └─► API failure → EmbeddingUnavailableError (abort, no writes)
    └─► Store.WriteDocument(filename) → documentID
        Store.WriteChunks(documentID, chunks) → ok
    ↓
HTTP 200 { "document_id": "...", "chunk_count": N }
```

**Latency budget:** PDF parse (~2s for 100pp) + embedding API calls (~100ms × N chunks, batched) + DB writes = < 30s for 100-page document

---

### Query Flow
```
User → Submit question (via UI)
    ↓
POST /api/query { "question": "..." }
    ↓
APIHandler.handleQuery()
    ↓
QueryProcessor.Query(question)
    ├─► EmbeddingClient.Embed(question, EmbeddingModel) → queryVector   (~200ms)
    │       └─► failure → EmbeddingUnavailableError
    ├─► Store.SimilaritySearch(queryVector, topK=3) → []Chunk  (~100ms)
    │       └─► empty → NoResultsError (early exit, no LLM call) ← INV-06
    ├─► BuildRAGPrompt(question, chunks) → prompt
    ├─► Groq.Complete(prompt) → rawResponse                     (~3-4s)
    │       └─► timeout/error → LLMUnavailableError
    ├─► ValidateCitations(rawResponse)
    │       └─► no citations → EXC-01 fallback message
    └─► Return QueryResult{ Answer, Citations[] }
    ↓
HTTP 200 { "answer": "...", "citations": [...], "query": "..." }
```

**Latency budget:** Embedding (200ms) + similarity search (100ms) + Claude synthesis (3–4s) = ~4.5s total (target < 5s)

---

## Technology Mapping

| Technology | Component | Rationale |
|------------|-----------|-----------|
| Go 1.22+ | All backend components | Performance, concurrency, single binary deployment |
| Gin framework | APIHandler | Minimal HTTP router, well-documented middleware |
| pdfcpu | Ingester | Pure Go PDF text extraction, no system dependency |
| HuggingFace Inference API (HTTP) | Ingester, QueryProcessor | `BAAI/bge-m3` embeddings (production) |
| Ollama HTTP client | Ingester, QueryProcessor | `bge-m3` embeddings + `llama3.3` (local dev) |
| Groq SDK / HTTP | QueryProcessor | `llama-3.3-70b-versatile` for RAG synthesis (production) |
| pgx v5 | Store | PostgreSQL driver with pgvector support |
| pgvector-go | Store | Vector type bindings for pgx |
| PostgreSQL 16 + pgvector | Store | Vector similarity search, transactional writes |
| Docker | Infrastructure | Reproducible containerized deployment |
| Railway or Render | Deployment | Go backend + managed PostgreSQL |
| Next.js 14 + React 18 | UI | SSR-capable SPA, Vercel-native deployment |
| TypeScript | UI | Type safety for API contracts |
| shadcn/ui | UI | Accessible, unstyled component primitives |
| Tailwind CSS | UI | Utility-first styling |
| Vercel | Deployment | Zero-config frontend hosting with HTTPS |

---

## Deployment Architecture

```
┌────────────────────────────────────────────────────────┐
│  Vercel (Frontend)                                     │
│  Next.js app — static + SSR                            │
│  Domain: legalbridge.vercel.app                        │
└───────────────────────┬────────────────────────────────┘
                        │ HTTPS (CORS-allowed origin)
┌───────────────────────▼────────────────────────────────┐
│  Railway / Render (Backend)                            │
│  Docker container — Go binary                          │
│  PORT: 8080                                            │
│  ENV: HF_API_KEY, GROQ_API_KEY, DATABASE_URL           │
│                                                        │
│  ┌─────────────┐  ┌────────────────┐  ┌─────────────┐ │
│  │  APIHandler │  │    Ingester    │  │QueryProcessor│ │
│  └──────┬──────┘  └───────┬────────┘  └──────┬──────┘ │
│         │                 │                   │        │
│         └─────────────────▼───────────────────┘        │
│                      ┌─────────┐                       │
│                      │  Store  │                       │
│                      └────┬────┘                       │
└───────────────────────────┼────────────────────────────┘
                            │ TCP (DATABASE_URL)
┌───────────────────────────▼────────────────────────────┐
│  PostgreSQL 16 + pgvector                              │
│  Railway managed database (or Docker sidecar locally)  │
└────────────────────────────────────────────────────────┘

External calls from backend (production):
  → api-inference.huggingface.co (bge-m3 embeddings)
  → api.groq.com (llama-3.3-70b generation)

Local development (Ollama):
  → localhost:11434 (embeddings + generation)
```

---

## Project Structure

```
legalbridge/
├── backend/
│   ├── main.go
│   ├── Dockerfile
│   ├── docker-compose.yml
│   ├── pkg/
│   │   └── config/
│   │       └── config.go         # EmbeddingModel constant, env var loading (INV-04, INV-08)
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handler.go        # Gin routes, error translation
│   │   │   └── middleware.go     # CORS, logging
│   │   ├── ingester/
│   │   │   ├── ingester.go       # Ingest() entry point
│   │   │   ├── pdf.go            # pdfcpu text extraction (INV-05)
│   │   │   └── chunker.go        # 500-token chunking with overlap
│   │   ├── query/
│   │   │   ├── processor.go      # Query() entry point
│   │   │   └── prompt.go         # RAG prompt template (INV-01, INV-03, INV-07)
│   │   └── store/
│   │       ├── store.go          # Store interface
│   │       └── postgres.go       # pgx + pgvector implementation
│   └── migrations/
│       └── 001_init.sql          # documents, chunks tables + ivfflat index
│
└── frontend/
    ├── app/
    │   ├── page.tsx              # Main page
    │   └── layout.tsx
    ├── components/
    │   ├── DocumentUpload.tsx    # PDF upload with drag-and-drop
    │   ├── QueryInput.tsx        # Question input + submit
    │   ├── AnswerDisplay.tsx     # Answer + citation blocks
    │   └── Disclaimer.tsx        # Legal disclaimer (BR-01)
    └── lib/
        └── api.ts                # API client (ingest + query)
```

---

## Invariant Traceability Matrix

| Invariant | Architecture Enforcement Mechanism |
|-----------|-------------------------------------|
| INV-01 Answer Grounding | `query/prompt.go` prompt template: "Answer using only the passages provided" |
| INV-02 Citation Completeness | `query/processor.go`: validates `Citations` in response before return |
| INV-03 No Legal Counsel | `query/prompt.go` system prompt: "You do not provide legal advice" |
| INV-04 Embedding Model Consistency | `pkg/config/config.go`: `EmbeddingModel = "bge-m3"` — single constant, imported by Ingester and QueryProcessor |
| INV-05 Passage Fidelity | `ingester/ingester.go`: raw extracted text written directly to `chunks.content`; no transform |
| INV-06 No Fabrication on Empty Retrieval | `query/processor.go`: `if len(chunks) == 0 { return NoResultsError }` before Claude call |
| INV-07 Response Language Consistency | `query/prompt.go`: "Respond in the same language as the question" |
| INV-08 API Key Secrecy | `pkg/config/config.go`: `log.Fatal` if key is empty; keys never written to logs |

---

## Architectural Constraints & ADRs

**ADR-01: Modular monolith over microservices**
A single Go binary with bounded internal packages. Microservices add distributed state management, network failure modes, and deployment orchestration that are unjustified at MVP scale. All components share one PostgreSQL instance; inter-component communication is function calls, not HTTP.

**ADR-02: pgvector over dedicated vector database**
PostgreSQL + pgvector eliminates a dedicated infrastructure dependency (Pinecone, Weaviate). At hundreds-of-chunks scale, pgvector's IVFFlat index performs adequately. The Store interface abstracts the implementation — migration to a dedicated vector DB is a Store swap, not an architecture change.

**ADR-03: BAAI/bge-m3 for embeddings, Llama 3.3 via Groq for synthesis**
`BAAI/bge-m3` is an open-source (Apache 2.0), multilingual embedding model that supports English and French — a hard requirement for West African legal documents. It produces 1024-dimensional vectors compatible with pgvector and outperforms comparably-sized English-only models on multilingual retrieval benchmarks. Groq's free tier (no credit card required) serves Meta's Llama 3.3 70B with LPU inference, meeting the <5s response SLA without any API cost. Both providers use open-source model weights; contributors can self-host both via Ollama with zero external API dependency.

**ADR-04: Separate frontend deployment on Vercel**
Next.js on Vercel provides HTTPS, CDN, and zero-config preview deployments. Serving the UI from the Go binary would require embedding static assets, adding build complexity. The separation is clean: the Go binary is a pure API server; the frontend is a pure SPA.

**ADR-05: No authentication in MVP**
User accounts, API tokens, and session management are explicitly out of scope (PRD §5). Adding auth would require additional infrastructure (session store, JWT signing) and delay the demo build. The pre-loaded document approach removes the need for user-specific document scoping.
