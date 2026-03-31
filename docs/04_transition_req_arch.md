# LegalBridge — Transition: Requirements to Architecture
Version: v1.0, 2026-03-31

## Method

Every invariant from `03_design_contract_invariant.md` must be assigned to exactly one component owner. Not a technology. Not a library. A conceptual responsibility. If an invariant is violated in production, this document answers: which component failed, and which file does the engineer open?

---

## Component Definitions

**Ingester** — Owns the document onboarding pipeline. Accepts a PDF byte stream, extracts full text, splits into overlapping chunks, calls the embedding API (`BAAI/bge-m3` via HuggingFace Inference API in production, Ollama locally) for each chunk, and writes all records to Store. It is the only component that writes document and chunk records. It is a valid call site for the embedding API during write operations.

**QueryProcessor** — Owns the answer generation pipeline. Accepts a plain-language question string, generates its embedding via the embedding API, retrieves the top 3 semantically relevant chunks from Store, constructs the RAG prompt, calls the Groq API (`llama-3.3-70b-versatile`), validates that the response contains citations, and returns the cited answer. It is the only component allowed to call the Groq API.

**Store** — Owns all database operations. Provides typed methods for: writing a document record, writing chunk records with embedding vectors, and performing top-k cosine similarity search. It is the only component with a live database connection. It exposes no query that bypasses vector dimension validation.

**APIHandler** — Owns the HTTP surface of the backend. Routes `POST /api/ingest` to Ingester, `POST /api/query` to QueryProcessor, and `GET /api/health` to a status probe. Translates component errors into HTTP status codes and standardized error envelopes. Owns startup validation of environment variables (INV-08).

**UI** — Owns the user-facing interface. Renders document upload, question input, answer display, and citations. Communicates exclusively with APIHandler over HTTPS. Owns the legal disclaimer display (BR-01).

---

## Invariant Assignments

### Ingester (owns: INV-04, INV-05)

**INV-04 — Embedding Model Consistency**: Ingester is the component that generates and persists the embedding vectors. It MUST use the shared `EMBEDDING_MODEL` constant from `pkg/config`. Because Ingester creates the vectors that QueryProcessor will later compare against, the model choice is locked at ingestion time. Drift is only possible if Ingester uses a different constant — enforced by using the single shared value.

**INV-05 — Passage Fidelity**: Ingester extracts text from PDFs and writes chunk records. The `content` field written to the database MUST be the verbatim extracted string from the PDF, with no transformation between extraction and write. Any summarization pipeline introduced between these two steps would be a violation introduced in Ingester.

### QueryProcessor (owns: INV-01, INV-02, INV-03, INV-06, INV-07)

**INV-01 — Answer Grounding**: QueryProcessor constructs the prompt sent to Claude. It is the sole decision-maker about what context Claude receives. The prompt template MUST include: "Answer using only the passages provided. Do not use external knowledge."

**INV-02 — Citation Completeness**: QueryProcessor receives the Groq API response before it is forwarded to APIHandler. It MUST inspect the response for citation content and discard any response that lacks citations, substituting a fallback message.

**INV-03 — No Legal Counsel**: QueryProcessor's system prompt MUST include: "You are a legal document retrieval tool. You do not provide legal advice, legal opinions, or legal counsel." This instruction is part of every Groq API call, making QueryProcessor the enforcement point.

**INV-06 — No Fabrication on Empty Retrieval**: QueryProcessor performs the similarity search and receives the results before deciding to call Claude. It MUST implement an early-exit branch: if `len(results) == 0`, return `NoResultsError` immediately without proceeding to the LLM call.

**INV-07 — Response Language Consistency**: QueryProcessor constructs the prompt and MUST append a language instruction: "Respond in the same language as the question."

### APIHandler (owns: INV-08)

**INV-08 — API Key Secrecy**: APIHandler (specifically `pkg/config.Load()`, called at startup by the server main function) MUST validate that `HF_API_KEY` and `GROQ_API_KEY` are present and non-empty. On failure, it MUST call `log.Fatal` with the missing variable name. Keys MUST NOT be logged at INFO or DEBUG level during normal startup.

### Store (no invariant ownership — structural enforcer only)

Store does not own invariants, but it MUST enforce structural constraints that prevent violations at the data layer:
- Chunk writes MUST fail if `embedding` is nil or has dimension ≠ 1024
- All vector queries MUST use pgvector's `<=>` cosine distance operator with a parameterized top-k value

---

## Invariant Coverage Table

| Invariant | Owner | Enforcement Point |
|-----------|-------|-------------------|
| INV-01 Answer Grounding | QueryProcessor | Prompt template: "Answer using only the provided passages" |
| INV-02 Citation Completeness | QueryProcessor | Post-LLM response validation; discard + fallback if no citations |
| INV-03 No Legal Counsel | QueryProcessor | System prompt: explicit "no legal advice" instruction |
| INV-04 Embedding Model Consistency | Ingester | Shared `config.EmbeddingModel` constant — single definition, two call sites |
| INV-05 Passage Fidelity | Ingester | Verbatim text written to `chunks.content`; no transformation pipeline |
| INV-06 No Fabrication on Empty Retrieval | QueryProcessor | Early return before LLM call when `len(results) == 0` |
| INV-07 Response Language Consistency | QueryProcessor | Language instruction appended to every RAG prompt |
| INV-08 API Key Secrecy | APIHandler | `config.Load()` panics on missing keys at startup; keys never logged |

---

## Coupling & Cohesion Decisions

**Why Ingester and QueryProcessor are separate despite both calling the embedding API:**
Both use `bge-m3`, but for different purposes and at different points in the data lifecycle. Keeping them separate ensures INV-04 enforcement is explicit: there are exactly two call sites for the embedding API, both using `config.EmbeddingModel`. If they were merged, the boundary between write-path and read-path behavior would blur, making it harder to reason about which invariants apply to each operation.

**Why Store has no invariant ownership:**
Invariants express behavioral guarantees, not data access mechanics. Store is a typed persistence layer — a utility. If INV-01 (answer grounding) is violated, the failure is in QueryProcessor's prompt, not in Store's query execution. Ownership belongs to the component making the behavioral decision, not the component executing the mechanism.

**Why APIHandler owns INV-08 rather than a dedicated config component:**
INV-08 is about startup behavior — failing fast before the server accepts any requests. This is correctly placed at the entry point (`main.go` / APIHandler initialization), not in a deep utility. Any engineer investigating an "API key not configured" failure will start at the server startup logs, making APIHandler the right discovery point.

**Why UI is a separate deployment (Vercel) rather than served by the Go backend:**
Next.js on Vercel provides zero-config HTTPS, CDN distribution, and preview deployments. Serving the frontend from the Go binary would add complexity with no benefit at MVP scale. CORS configuration on the Go backend allows the Vercel domain as an allowed origin.
