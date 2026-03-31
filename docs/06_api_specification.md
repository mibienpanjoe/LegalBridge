# LegalBridge — API Specification
Version: v1.0, 2026-03-31

## Conventions

**Base URL:** `https://api.legalbridge.app/api` (Railway/Render deployment)
**Local development:** `http://localhost:8080/api`

**Content-Type:**
- `POST /api/ingest`: `multipart/form-data`
- `POST /api/query`: `application/json`
- All responses: `application/json`

**Authentication:** None for MVP. No `Authorization` header required.

**CORS:** The backend accepts requests from the configured Vercel frontend origin. During local development, `http://localhost:3000` is allowed.

**Error envelope** (all non-2xx responses):
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable description of what went wrong"
  }
}
```

**Success envelope** (all 2xx responses): no wrapper — each endpoint returns its own top-level shape.

---

## Endpoints

### POST /api/ingest

Ingest a PDF document. Extracts text, chunks it, generates embeddings, and stores the result in the vector database. The document becomes immediately queryable upon success.

**Auth required:** No

**Request:** `multipart/form-data`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `file` | File | Yes | PDF document to ingest. MIME type must be `application/pdf`. |

**Success — 200 OK:**
```json
{
  "document_id": "3f4a7b2c-1e6d-4a89-b3c2-9d1e5f7a8b0c",
  "filename": "ghana_companies_act.pdf",
  "chunk_count": 142
}
```

| Field | Type | Description |
|-------|------|-------------|
| `document_id` | UUID string | Unique identifier for the ingested document |
| `filename` | string | Original filename from the upload |
| `chunk_count` | integer | Number of text chunks created and stored |

**Errors:**

| Status | Code | Trigger |
|--------|------|---------|
| 400 | `INVALID_FILE_TYPE` | Uploaded file MIME type is not `application/pdf` |
| 400 | `MISSING_FILE` | No `file` field present in the request |
| 422 | `EXTRACTION_FAILED` | PDF text extraction returned an empty string or failed |
| 503 | `EMBEDDING_UNAVAILABLE` | Embedding API (HuggingFace/Ollama) timed out or returned an error |
| 503 | `DATABASE_UNAVAILABLE` | PostgreSQL connection failed during write |

---

### POST /api/query

Submit a natural language question. The system embeds the question, retrieves the top 3 most semantically relevant passages from the vector database, synthesizes a cited answer using Llama 3.3 via Groq, and returns the result.

**Auth required:** No

**Request body:**
```json
{
  "question": "What are the requirements for registering a foreign company in Ghana?"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `question` | string | Yes | Plain-language question in English or French. Must not be empty. |

**Success — 200 OK (answer found):**
```json
{
  "query": "What are the requirements for registering a foreign company in Ghana?",
  "answer": "According to the Ghana Companies Act, a foreign company must register with the Registrar-General within 28 days of establishing a place of business in Ghana [1]. The required documents include a certified copy of the company's charter, statutes, or memorandum and articles of association [2].",
  "citations": [
    {
      "index": 1,
      "document_name": "ghana_companies_act.pdf",
      "passage": "Every foreign company shall, within twenty-eight days after establishing a place of business in Ghana, deliver to the Registrar for registration a certified copy of the instrument constituting or defining the constitution of the company..."
    },
    {
      "index": 2,
      "document_name": "ghana_companies_act.pdf",
      "passage": "The documents required for registration under this section include a certified copy of the company's charter, statutes, or memorandum and articles of association..."
    }
  ],
  "no_results": false
}
```

**Success — 200 OK (no relevant passages found):**
```json
{
  "query": "What is the capital gains tax rate in Burkina Faso?",
  "answer": null,
  "citations": [],
  "no_results": true,
  "message": "No relevant passages were found in the document for this question. Try rephrasing or ask about a different topic."
}
```

**Response fields:**

| Field | Type | Description |
|-------|------|-------------|
| `query` | string | The original question submitted by the user |
| `answer` | string \| null | Synthesized answer from Llama 3.3. `null` if `no_results` is `true`. |
| `citations` | Citation[] | Array of source passages used. Empty array if `no_results` is `true`. |
| `no_results` | boolean | `true` if the similarity search returned no results above threshold |
| `message` | string | (present only when `no_results: true`) Human-readable explanation |

**Errors:**

| Status | Code | Trigger |
|--------|------|---------|
| 400 | `EMPTY_QUERY` | `question` field is empty string or missing |
| 503 | `LLM_UNAVAILABLE` | Groq API timed out (>10s) or returned a non-2xx response |
| 503 | `EMBEDDING_UNAVAILABLE` | Embedding API (HuggingFace/Ollama) timed out or returned an error |
| 503 | `DATABASE_UNAVAILABLE` | PostgreSQL connection failed during similarity search |

---

### GET /api/health

Returns the health status of the backend and its database connection. Used by the deployment platform for liveness/readiness probes and by the frontend to check backend availability.

**Auth required:** No

**Success — 200 OK (all systems healthy):**
```json
{
  "status": "ok",
  "database": "ok",
  "timestamp": "2026-03-31T10:00:00Z"
}
```

**Degraded — 503 Service Unavailable (database unreachable):**
```json
{
  "status": "degraded",
  "database": "unavailable",
  "timestamp": "2026-03-31T10:00:00Z"
}
```

---

## Outbound API Calls

The backend makes the following outbound calls to external APIs:

### Embedding API — BAAI/bge-m3

**Called by:** Ingester (during document ingestion), QueryProcessor (during query processing)

**Production — HuggingFace Inference API:**
```
POST https://api-inference.huggingface.co/pipeline/feature-extraction/BAAI/bge-m3
Authorization: Bearer {HF_API_KEY}
Content-Type: application/json

{
  "inputs": "<text to embed>",
  "options": { "wait_for_model": true }
}
```

**Expected response:** JSON array of 1024 floats: `[0.012, -0.045, ...]`

**Local development — Ollama:**
```
POST http://localhost:11434/api/embeddings
Content-Type: application/json

{
  "model": "bge-m3",
  "prompt": "<text to embed>"
}
```

**Expected response:** `{ "embedding": [0.012, -0.045, ...] }` (1024 floats)

**Timeout:** 10 seconds
**Retry policy:** No automatic retry in MVP. On failure, return `EMBEDDING_UNAVAILABLE` to caller.

---

### Groq API — llama-3.3-70b-versatile

**Called by:** QueryProcessor (during query processing)

Groq uses an OpenAI-compatible chat completions format.

**Production — Groq:**
```
POST https://api.groq.com/openai/v1/chat/completions
Authorization: Bearer {GROQ_API_KEY}
Content-Type: application/json

{
  "model": "llama-3.3-70b-versatile",
  "messages": [
    { "role": "system", "content": "<system prompt with no-legal-advice instruction>" },
    { "role": "user", "content": "<RAG prompt with question + retrieved passages>" }
  ],
  "max_tokens": 1024,
  "temperature": 0.1
}
```

**Local development — Ollama:**
```
POST http://localhost:11434/api/chat
Content-Type: application/json

{
  "model": "llama3.2",
  "messages": [
    { "role": "system", "content": "<system prompt>" },
    { "role": "user", "content": "<RAG prompt>" }
  ],
  "stream": false
}
```

**Expected response:** `choices[0].message.content` (Groq) or `message.content` (Ollama) — synthesized answer with inline citation markers (`[1]`, `[2]`, `[3]`).

**Timeout:** 10 seconds
**Retry policy:** No automatic retry in MVP. On timeout or error, return `LLM_UNAVAILABLE` to caller.

---

## Type Reference

### Citation Object

```json
{
  "index": 1,
  "document_name": "ghana_companies_act.pdf",
  "passage": "Verbatim extracted text from the source document chunk..."
}
```

| Field | Type | Description |
|-------|------|-------------|
| `index` | integer | 1-based index matching the `[N]` reference in the answer text |
| `document_name` | string | Original filename of the source document |
| `passage` | string | Verbatim extracted text from the chunk used in synthesis |

### Error Object

```json
{
  "error": {
    "code": "LLM_UNAVAILABLE",
    "message": "The answer service is temporarily unavailable. Please try again."
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `code` | string | Machine-readable error identifier (for frontend error handling) |
| `message` | string | Human-readable description suitable for display to the user |

---

## Endpoint Summary Table

| Method | Path | Description | Auth | Request Type |
|--------|------|-------------|------|--------------|
| POST | `/api/ingest` | Ingest a PDF document | None | `multipart/form-data` |
| POST | `/api/query` | Ask a question, get a cited answer | None | `application/json` |
| GET | `/api/health` | Backend + database health check | None | — |
