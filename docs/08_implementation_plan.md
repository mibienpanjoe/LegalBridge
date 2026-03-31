# LegalBridge — Implementation Plan
Version: v1.0, 2026-03-31

This document breaks the LegalBridge build into discrete, sequenced phases. Each phase is independently shippable and leaves the system in a working (if incomplete) state. The commit strategy section defines commit granularity and message conventions.

---

## Guiding Principles

- **Vertical slices over horizontal layers** — each phase delivers a testable capability, not just a layer of code.
- **Invariants first** — before adding any pipeline logic, the contracts that enforce safety (INV-01 through INV-08) must be in place.
- **Local dev at every phase** — the Ollama path must work from Phase 2 onward so any contributor can run the system without API keys.
- **No partial writes, no silent failures** — error handling is built alongside the happy path, never as a follow-up.

---

## Phase 0 — Repository Scaffold

**Goal:** An empty but correctly shaped repo that compiles and boots.

### Tasks

1. Initialize Go module: `go mod init github.com/your-org/legalbridge`
2. Create directory skeleton:
   ```
   backend/
     cmd/server/main.go
     cmd/migrate/main.go
     internal/api/
     internal/ingester/
     internal/query/
     internal/store/
     pkg/config/
     migrations/
   frontend/
     app/
     components/
     lib/
   docs/
   ```
3. Add `backend/go.mod` with initial dependencies:
   - `github.com/gin-gonic/gin`
   - `github.com/jackc/pgx/v5`
   - `github.com/pgvector/pgvector-go`
   - `github.com/pdfcpu/pdfcpu`
   - `github.com/google/uuid`
4. Bootstrap Next.js frontend: `npx create-next-app@latest frontend --typescript --tailwind --app`
5. Install frontend deps: `shadcn/ui`, Tailwind config
6. Add root `.gitignore` (Go binaries, `node_modules`, `.env` files)
7. Add `backend/.env.example` with all required variables (no real values)
8. Add root `docker-compose.yml` with `postgres` service (pgvector image)

### Exit Criteria
- `go build ./...` passes in `backend/`
- `npm run dev` starts in `frontend/`
- `docker-compose up -d postgres` brings up the database

### Commits
```
chore: initialize Go module and directory skeleton
chore: bootstrap Next.js frontend with TypeScript and Tailwind
chore: add docker-compose with pgvector postgres service
chore: add .env.example and .gitignore
```

---

## Phase 1 — Configuration and Database Schema

**Goal:** A validated config layer and a runnable migration that creates the full schema.

### Tasks

1. **`pkg/config/config.go`**
   - Define `EmbeddingModel = "BAAI/bge-m3"` constant (INV-04 single source of truth)
   - Define `Config` struct: `DatabaseURL`, `EmbeddingProvider`, `LLMProvider`, `OllamaBaseURL`, `OllamaModel`, `HFAPIKey`, `GroqAPIKey`
   - Implement `Load() (*Config, error)` — read from env vars
   - Implement `Validate(c *Config) error` — enforce required fields by provider; `log.Fatal` if production API keys are empty (INV-08)

2. **`migrations/001_init.sql`**
   - `CREATE EXTENSION IF NOT EXISTS "pgcrypto";`
   - `CREATE EXTENSION IF NOT EXISTS "vector";`
   - `CREATE TABLE documents (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), filename TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());`
   - `CREATE TABLE chunks (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE, content TEXT NOT NULL, embedding vector(1024) NOT NULL, chunk_index INTEGER NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());`
   - `CREATE INDEX ON chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);`

3. **`cmd/migrate/main.go`**
   - Load config, open pgx connection, apply `001_init.sql`

### Exit Criteria
- `go run ./cmd/migrate` creates tables and index without error
- `config.Load()` returns `EmbeddingUnavailableError`-equivalent if a required env var is missing in production mode

### Commits
```
feat(config): add Config struct, Load(), and Validate() with INV-08 enforcement
feat(config): define EmbeddingModel constant as single source of truth (INV-04)
feat(migrations): add 001_init.sql — documents and chunks tables with ivfflat index
feat(migrate): add migration runner cmd
```

---

## Phase 2 — Store Layer

**Goal:** A fully implemented `Store` interface backed by PostgreSQL + pgvector. No business logic — only reads and writes.

### Tasks

1. **`internal/store/store.go`** — define the `Store` interface and `Chunk`, `Document` types:
   ```go
   type Store interface {
       WriteDocument(ctx context.Context, filename string) (string, error)
       WriteChunks(ctx context.Context, documentID string, chunks []Chunk) error
       SimilaritySearch(ctx context.Context, vector []float32, topK int) ([]Chunk, error)
       Ping(ctx context.Context) error
   }
   ```

2. **`internal/store/postgres.go`** — implement `PostgresStore`:
   - `WriteDocument`: INSERT into `documents`, return UUID
   - `WriteChunks`: bulk INSERT into `chunks`; validate `len(embedding) == 1024` before any write — return error if invalid dimension
   - `SimilaritySearch`: parameterized `SELECT ... ORDER BY embedding <=> $1 LIMIT $2` using cosine distance
   - `Ping`: simple `SELECT 1` with timeout
   - All queries are parameterized (no string interpolation)

3. **`internal/store/errors.go`** — define `DatabaseUnavailableError`

4. Write unit tests for `WriteChunks` dimension validation (table-driven, can use a test DB or mock)

### Exit Criteria
- `go test ./internal/store/...` passes
- `SimilaritySearch` returns `[]Chunk` sorted by cosine similarity
- `WriteChunks` rejects embeddings with wrong dimension

### Commits
```
feat(store): define Store interface and domain types
feat(store): implement PostgresStore with WriteDocument and WriteChunks
feat(store): implement SimilaritySearch with cosine distance ordering
feat(store): add DatabaseUnavailableError and Ping
test(store): add WriteChunks dimension validation unit tests
```

---

## Phase 3 — Ingester Pipeline

**Goal:** Upload a PDF, get back a document ID and chunk count. The full ingest pipeline must work end-to-end with Ollama locally.

### Tasks

1. **`internal/ingester/pdf.go`** — PDF text extraction:
   - Use `pdfcpu` to extract raw text from `[]byte`
   - Return `ExtractionFailedError` if result is empty string (FR-052)
   - Do NOT transform the extracted text (INV-05)

2. **`internal/ingester/chunker.go`** — word-boundary tokenizer:
   - Split text into chunks of ~500 tokens with 50-token overlap
   - Return `[]string` of chunk texts

3. **`internal/ingester/embedding_client.go`** — embedding abstraction:
   - Define `EmbeddingClient` interface: `Embed(ctx, text, model string) ([]float32, error)`
   - Implement `HuggingFaceClient` (HTTP POST to HF Inference API)
   - Implement `OllamaClient` (HTTP POST to `localhost:11434/api/embeddings`)
   - Factory function `NewEmbeddingClient(cfg *config.Config) EmbeddingClient`
   - Return `EmbeddingUnavailableError` on API failure

4. **`internal/ingester/ingester.go`** — orchestrate the pipeline (INV-04, INV-05):
   - `Ingest(ctx, fileBytes []byte, filename string) (*IngestResult, error)`
   - Call `ExtractText` → if empty: return `ExtractionFailedError` immediately
   - Call `Chunk(text)` → `[]string`
   - For each chunk: call `EmbeddingClient.Embed(chunk, config.EmbeddingModel)` — use the constant, never a string literal
   - On any embedding failure: abort, do NOT write any chunks (atomic all-or-nothing)
   - Call `Store.WriteDocument` then `Store.WriteChunks`
   - Return `IngestResult{DocumentID, ChunkCount}`

5. Define typed errors: `InvalidFileTypeError`, `ExtractionFailedError`, `EmbeddingUnavailableError`

6. Tests: chunker unit tests (chunk count, overlap correctness), ingester integration test with mocked embedding client

### Exit Criteria
- `curl -X POST :8080/api/ingest -F file=@test.pdf` (once API is wired in Phase 5) returns `document_id` and `chunk_count`
- If embedding client fails midway, zero chunks are written
- `config.EmbeddingModel` is the only string passed to `Embed()` — no hardcoded model names

### Commits
```
feat(ingester): add PDF text extraction using pdfcpu (INV-05)
feat(ingester): add word-boundary chunker with 500-token/50-overlap split
feat(ingester): add EmbeddingClient interface and OllamaClient implementation
feat(ingester): add HuggingFaceClient implementation for production embeddings
feat(ingester): add Ingest() orchestrator with atomic write and INV-04 enforcement
feat(ingester): define InvalidFileTypeError and ExtractionFailedError
test(ingester): add chunker unit tests with overlap validation
test(ingester): add Ingest() integration test with mocked embedding client
```

---

## Phase 4 — Query Processor

**Goal:** A question in, a cited answer out. This phase wires the full RAG pipeline and enforces all answer-quality invariants.

### Tasks

1. **`internal/query/prompt.go`** — RAG prompt builder:
   - `BuildRAGPrompt(question string, chunks []store.Chunk) string`
   - System instruction MUST include: "Answer using only the provided passages. Do not use external knowledge." (INV-01)
   - System instruction MUST include: "You do not provide legal advice." (INV-03)
   - System instruction MUST include: "Respond in the same language as the question." (INV-07)
   - Format context as numbered passages `[1]`, `[2]`, `[3]`

2. **`internal/query/llm_client.go`** — LLM abstraction:
   - Define `LLMClient` interface: `Complete(ctx, prompt string) (string, error)`
   - Implement `GroqClient` (HTTP POST to `api.groq.com/openai/v1/chat/completions`, model `llama-3.3-70b-versatile`)
   - Implement `OllamaLLMClient` (HTTP POST to Ollama, model from config)
   - 10-second timeout enforced at the HTTP client level (FR-051)
   - Return `LLMUnavailableError` on timeout or non-2xx

3. **`internal/query/processor.go`** — orchestrate the query pipeline:
   - `Query(ctx, question string) (*QueryResult, error)`
   - Return `EmptyQueryError` if `question` is empty (ERR-021)
   - Call `EmbeddingClient.Embed(question, config.EmbeddingModel)` → `queryVector` (INV-04)
   - Call `Store.SimilaritySearch(queryVector, 3)` → `[]Chunk`
   - **If `len(chunks) == 0`**: return `NoResultsError` immediately — do NOT call LLM (INV-06)
   - Call `BuildRAGPrompt(question, chunks)` → `prompt`
   - Call `LLMClient.Complete(prompt)` → `rawAnswer`
   - **Validate citations**: if response contains no `[N]` reference → return fallback `QueryResult` with `no_results: true` (INV-02)
   - Parse citations: extract passage references and map back to chunk content
   - Return `QueryResult{Answer, Citations}`

4. Define `QueryResult`, `Citation` types; define `EmptyQueryError`, `NoResultsError`, `LLMUnavailableError`

5. Tests:
   - Prompt builder: assert INV-01, INV-03, INV-07 strings are present in output
   - Processor: mock store returns empty → assert `NoResultsError` (INV-06)
   - Processor: mock LLM returns uncited text → assert fallback returned (INV-02)

### Exit Criteria
- Full RAG cycle works end-to-end with Ollama
- INV-01, INV-02, INV-03, INV-06, INV-07 are covered by tests
- LLM is never called with an empty chunk list

### Commits
```
feat(query): add BuildRAGPrompt with INV-01, INV-03, INV-07 instructions
feat(query): add LLMClient interface and OllamaLLMClient implementation
feat(query): add GroqClient with 10-second timeout (FR-051)
feat(query): add QueryProcessor.Query() with INV-06 early exit and INV-02 citation validation
feat(query): define EmptyQueryError, NoResultsError, LLMUnavailableError types
test(query): add prompt builder invariant assertions (INV-01, INV-03, INV-07)
test(query): add INV-06 unit test — no LLM call on empty retrieval
test(query): add INV-02 unit test — fallback on uncited LLM response
```

---

## Phase 5 — API Layer

**Goal:** Wire all internal components behind Gin HTTP routes. The backend is fully functional and testable via `curl`.

### Tasks

1. **`internal/api/middleware.go`**
   - CORS middleware: allow Vercel frontend origin + `localhost:3000`
   - Request logger middleware

2. **`internal/api/handler.go`** — define `APIHandler` struct (holds `Ingester`, `QueryProcessor`, `Store` deps):
   - `handleIngest`: validate MIME type → call `Ingester.Ingest()` → serialize response or translate error
   - `handleQuery`: parse JSON body → call `QueryProcessor.Query()` → serialize response or translate error
   - `handleHealth`: call `Store.Ping()` → return `{status, database, timestamp}`
   - Error translation table (see CLAUDE.md and architecture doc) — typed errors → HTTP codes

3. **`cmd/server/main.go`**
   - Call `config.Load()` and `config.Validate()` — fatal on invalid (INV-08)
   - Construct `Store`, `EmbeddingClient`, `LLMClient`, `Ingester`, `QueryProcessor`
   - Wire Gin router with all three routes
   - `router.Run(":8080")`

4. Smoke tests: `go test ./internal/api/...` with `httptest` — one happy-path test per endpoint

### Exit Criteria
- `go run ./cmd/server` starts and `GET /api/health` returns 200
- `POST /api/ingest` with a real PDF returns `document_id` and `chunk_count`
- `POST /api/query` with a known question returns `answer` and `citations`
- All ERR-0xx cases from the SRS return the correct HTTP status and error code

### Commits
```
feat(api): add CORS and logging middleware
feat(api): add handleIngest with MIME validation and error translation
feat(api): add handleQuery with typed error translation
feat(api): add handleHealth endpoint
feat(api): wire Gin router and server entry point with INV-08 startup validation
test(api): add httptest smoke tests for all three endpoints
```

---

## Phase 6 — Frontend

**Goal:** A usable UI that covers the full user journey: upload → question → cited answer.

### Tasks

1. **`lib/api.ts`** — typed API client:
   - `ingestDocument(file: File): Promise<IngestResponse>`
   - `queryDocument(question: string): Promise<QueryResponse>`
   - Typed response interfaces matching the API spec
   - Error handling for non-2xx responses

2. **`components/Disclaimer.tsx`** — static legal disclaimer (BR-01):
   - "LegalBridge provides legal information, not legal advice. Consult a qualified lawyer for legal counsel."
   - Displayed persistently at the top of the page

3. **`components/DocumentUpload.tsx`**
   - Drag-and-drop + click-to-upload PDF input
   - Progress indicator during upload (`/api/ingest` call)
   - Display `document_id` and `chunk_count` on success
   - Show specific error message for `INVALID_FILE_TYPE`, `EXTRACTION_FAILED`

4. **`components/QueryInput.tsx`**
   - Text area for question input (English or French)
   - Submit button; disable while loading
   - Show `EMBEDDING_UNAVAILABLE`, `LLM_UNAVAILABLE` errors inline

5. **`components/AnswerDisplay.tsx`**
   - Render answer text
   - Render each citation as an expandable block: passage index + verbatim text
   - If `no_results: true`: display "No relevant passages found"

6. **`app/page.tsx`** — compose all components; manage upload/query state

7. Apply visual identity from `docs/07_visual_identity.md` — color tokens, typography, spacing

### Exit Criteria
- Full user journey works in the browser with Ollama running locally
- Legal disclaimer is visible on every page load (BR-01)
- Citations are displayed as verbatim passage blocks
- Loading and error states are handled for each async operation

### Commits
```
feat(frontend): add typed API client with ingestDocument and queryDocument
feat(frontend): add Disclaimer component (BR-01)
feat(frontend): add DocumentUpload component with drag-and-drop and error states
feat(frontend): add QueryInput component with loading and error states
feat(frontend): add AnswerDisplay component with citation blocks and no_results state
feat(frontend): wire all components in main page with upload/query state management
feat(frontend): apply visual identity tokens (colors, typography, spacing)
```

---

## Phase 7 — Integration, Demo Document, and Hardening

**Goal:** The system is demo-ready. A pre-loaded document is available at startup and the five demo questions all return cited answers.

### Tasks

1. **Pre-load script (`cmd/seed/main.go`)**:
   - Download or include Ghana Companies Act (or equivalent) as a PDF
   - Call `Ingester.Ingest()` on startup if no document exists in the DB
   - Log chunk count on success

2. **End-to-end tests** — verify the 5 hackathon demo questions:
   - Each question hits the real pipeline (Ollama locally)
   - Assert `len(citations) >= 1` and `no_results == false`
   - Record expected answers for manual verification

3. **Performance check**:
   - Time a full query cycle; confirm < 5s with Groq (FR-NF performance)
   - Time a 100-page PDF ingest; confirm < 30s

4. **Hardening**:
   - Add request size limit to `handleIngest` (reject files > 20MB)
   - Confirm all env vars absent from logs (INV-08)
   - Add `recover()` middleware in Gin to prevent panics from crashing the server

5. **Docker build test**: `docker-compose up --build` — full stack boots, health check passes

### Exit Criteria
- `GET /api/health` returns `{"status":"ok","database":"ok"}` in Docker
- All 5 demo questions return cited answers against the pre-loaded document
- No API keys or sensitive values appear in stdout or logs

### Commits
```
feat(seed): add seed command to pre-ingest demo document on startup
feat(api): add request size limit middleware for ingest endpoint
fix(api): add panic recovery middleware to prevent server crashes
test(e2e): add demo question set with citation assertions
chore(docker): verify full-stack docker-compose build and health check
```

---

## Phase 8 — Deployment

**Goal:** The system is publicly accessible via HTTPS for the hackathon demo.

### Tasks

1. **Backend — Railway or Render**:
   - Push Docker image; configure `DATABASE_URL`, `HF_API_KEY`, `GROQ_API_KEY`, `EMBEDDING_PROVIDER=huggingface`, `LLM_PROVIDER=groq`
   - Confirm `GET /api/health` returns 200 on the public URL
   - Run `cmd/migrate` as a release command (or add to Docker entrypoint)

2. **Frontend — Vercel**:
   - Set `NEXT_PUBLIC_API_URL` to the Railway/Render backend URL
   - Confirm CORS origin in backend matches the Vercel domain
   - Deploy; smoke-test the full UI flow

3. **Update `README.md`** with live demo URL and production setup notes

4. **Tag release**: `git tag v1.0.0`

### Exit Criteria
- Live URL accessible over HTTPS
- Upload + query cycle works end-to-end in production
- Health endpoint confirms database connectivity

### Commits
```
chore(deploy): add Railway/Render deployment config and release command
feat(api): update CORS allowed origins to include production Vercel domain
chore(frontend): set NEXT_PUBLIC_API_URL for Vercel deployment
docs: add live demo URL and production setup to README
chore(release): tag v1.0.0
```

---

## Commit Strategy

### Message Format

```
<type>(<scope>): <imperative description>
```

**Types:**
| Type | When to use |
|------|-------------|
| `feat` | New behavior — adds a capability that didn't exist |
| `fix` | Corrects a broken behavior |
| `test` | Adds or modifies tests without changing production code |
| `chore` | Tooling, config, CI, Docker, deps — no production logic |
| `docs` | Documentation only |
| `refactor` | Internal restructuring with no behavior change |

**Scopes** map directly to the architecture:
`config`, `store`, `ingester`, `query`, `api`, `frontend`, `deploy`, `migrations`, `seed`

### Rules

1. **One logical unit per commit** — a commit implements one coherent thing (one interface, one handler, one component). Do not bundle unrelated changes.
2. **Tests commit immediately after the code they test** — never batch tests into a separate phase-end commit.
3. **No "WIP" commits on main** — rebase or squash before merging; every commit on `main` must pass `go test ./...` and `npm run build`.
4. **Invariant enforcement commits are annotated** — if a commit directly enforces a documented invariant, reference it: `feat(query): add INV-06 early exit before LLM call`.
5. **Phase boundaries get a tag** — after each phase is complete and all tests pass, tag the commit: `git tag phase-N`.

### Branch Strategy

```
main          ← always deployable; phase tags live here
feature/*     ← one branch per phase or per feature within a phase
```

Merge via PR (or direct push for solo dev); no force-push to `main`.

### Example commit sequence (Phase 3 excerpt)

```
feat(ingester): add PDF text extraction using pdfcpu (INV-05)
test(ingester): assert ExtractionFailedError on empty PDF output
feat(ingester): add word-boundary chunker with 500-token/50-overlap split
test(ingester): add chunker unit tests — chunk count and overlap correctness
feat(ingester): add EmbeddingClient interface with OllamaClient
feat(ingester): add HuggingFaceClient for production embedding provider
feat(ingester): add Ingest() orchestrator with atomic write and INV-04 enforcement
test(ingester): add Ingest() integration test — verify zero writes on embedding failure
```

---

## Phase Summary

| Phase | Deliverable | Key Invariants Enforced |
|-------|-------------|-------------------------|
| 0 | Repo scaffold, Docker, env template | — |
| 1 | Config validation, DB schema, migrations | INV-08 |
| 2 | Store layer (PostgreSQL + pgvector) | dimension constraint |
| 3 | Full ingestion pipeline (PDF → embeddings → DB) | INV-04, INV-05 |
| 4 | Full query pipeline (question → cited answer) | INV-01, INV-02, INV-03, INV-06, INV-07 |
| 5 | API layer (Gin routes, error translation) | INV-08 startup |
| 6 | Frontend (upload, query, citations UI) | BR-01 disclaimer |
| 7 | Demo readiness (seed doc, e2e tests, hardening) | All invariants verified |
| 8 | Production deployment (Railway + Vercel) | — |
