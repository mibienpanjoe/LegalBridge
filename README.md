# LegalBridge

<p align="center">
  <img src="docs/assets/screenshot-landing.png" width="48%" alt="LegalBridge landing page — upload zone with legal disclaimer" />
  &nbsp;&nbsp;
  <img src="docs/assets/screenshot-query.png" width="48%" alt="LegalBridge query view — document loaded with French suggested questions" />
</p>

**Multilingual legal document assistant for West African businesses.**

LegalBridge lets you upload a legal PDF and ask questions in plain English or French. It retrieves the most relevant passages from the document and synthesizes a precise, cited answer — so every claim is traceable back to the source text.

Built for the **IA digne de confiance (Trustworthy AI)** hackathon category. Powered entirely by open-source models — no paid API required.

---

## What it does

```
You: "What are the requirements for registering a foreign company in Ghana?"

LegalBridge searches your uploaded legal document
    ↓
Retrieves the 3 most relevant passages (semantic search)
    ↓
Synthesizes a precise answer with citations

Answer: "According to the Ghana Companies Act, a foreign company must register
         with the Registrar-General within 28 days [1]. Required documents include
         a certified copy of the company's charter or memorandum [2]."

[1] "Every foreign company shall, within twenty-eight days after establishing
     a place of business in Ghana, deliver to the Registrar..."
[2] "The documents required for registration under this section include..."
```

---

## Features

- **PDF ingestion** — upload any legal document; it is chunked, embedded, and indexed instantly
- **Semantic search** — finds relevant passages even when your wording differs from the document's
- **Cited answers** — every response includes the exact passages it was drawn from
- **English + French** — ask in either language; answers match your question's language
- **Zero hallucination policy** — the model is instructed to answer only from retrieved passages; if no relevant text is found, it says so
- **Fully open-source stack** — no OpenAI, no Anthropic, no per-token costs

---

## Tech stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.22+ · Gin framework |
| Frontend | Next.js 14 · TypeScript · shadcn/ui · Tailwind CSS |
| Vector database | PostgreSQL 16 + pgvector |
| Embeddings | `BAAI/bge-m3` — multilingual, open-source (Apache 2.0) |
| LLM | `llama-3.3-70b-versatile` via Groq (free tier) |
| Local dev | Ollama — runs both `bge-m3` and `llama3.2` with no API keys |
| Deployment | Railway/Render (backend) · Vercel (frontend) |

---

## Architecture

```
┌─────────────────────────────────────────┐
│  Next.js (Vercel)                       │
│  Upload · Ask · Read cited answers      │
└─────────────────┬───────────────────────┘
                  │ HTTPS
┌─────────────────▼───────────────────────┐
│  Go backend (Railway / Render)          │
│                                         │
│  POST /api/ingest                       │
│    PDF → extract → chunk → embed        │
│    → store in pgvector                  │
│                                         │
│  POST /api/query                        │
│    question → embed → similarity search │
│    → top 3 passages → LLM → answer      │
└──────────┬──────────────────────────────┘
           │
┌──────────▼──────────────────────────────┐
│  PostgreSQL + pgvector                  │
│  documents · chunks · embeddings        │
└─────────────────────────────────────────┘

External calls (production):
  api-inference.huggingface.co  ← bge-m3 embeddings
  api.groq.com                  ← llama-3.3-70b generation

Local development (no API keys):
  localhost:11434 (Ollama)      ← both embeddings + generation
```

---

## Getting started

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)
- [Docker + Docker Compose](https://docs.docker.com/get-docker/)
- One of:
  - **Ollama** (local, no accounts needed) — `brew install ollama` or see [ollama.com](https://ollama.com)
  - **HuggingFace** account (free) + **Groq** account (free) for deployed use

---

### Option A — Local development with Ollama (recommended, no API keys)

**1. Clone the repo**

```bash
git clone https://github.com/your-username/legalbridge.git
cd legalbridge
```

**2. Pull models**

```bash
ollama pull bge-m3
ollama pull llama3.2
```

**3. Start the database**

```bash
docker-compose up -d postgres
```

**4. Configure environment**

```bash
cp backend/.env.example backend/.env
```

Edit `backend/.env`:

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/legalbridge
EMBEDDING_PROVIDER=ollama
OLLAMA_BASE_URL=http://localhost:11434
LLM_PROVIDER=ollama
OLLAMA_MODEL=llama3.2
```

**5. Run database migrations**

```bash
cd backend
go run ./cmd/migrate
```

**6. Start the backend**

```bash
go run ./cmd/server
# Listening on :8080
```

**7. Start the frontend**

```bash
cd ../frontend
npm install
npm run dev
# Open http://localhost:3000
```

---

### Option B — With free cloud APIs (HuggingFace + Groq)

Sign up at [huggingface.co](https://huggingface.co) and [console.groq.com](https://console.groq.com) — both are free, no credit card required.

Edit `backend/.env`:

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/legalbridge
EMBEDDING_PROVIDER=huggingface
HF_API_KEY=hf_your_token_here
LLM_PROVIDER=groq
GROQ_API_KEY=gsk_your_token_here
```

Then follow steps 3, 5, 6, 7 from Option A.

---

### Full Docker setup (backend + database)

```bash
docker-compose up --build
```

The `docker-compose.yml` starts both the Go backend and a PostgreSQL + pgvector instance.

---

## Environment variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `EMBEDDING_PROVIDER` | Yes | `ollama` or `huggingface` |
| `OLLAMA_BASE_URL` | If Ollama | Default: `http://localhost:11434` |
| `OLLAMA_MODEL` | If Ollama (LLM) | Default: `llama3.2` |
| `HF_API_KEY` | If HuggingFace | HuggingFace Inference API token |
| `LLM_PROVIDER` | Yes | `ollama` or `groq` |
| `GROQ_API_KEY` | If Groq | Groq API key |

A complete template is in `backend/.env.example`.

---

## API reference

### `POST /api/ingest`

Upload a PDF document for indexing.

```bash
curl -X POST http://localhost:8080/api/ingest \
  -F "file=@ghana_companies_act.pdf"
```

```json
{
  "document_id": "3f4a7b2c-...",
  "filename": "ghana_companies_act.pdf",
  "chunk_count": 142
}
```

### `POST /api/query`

Ask a question against the indexed document.

```bash
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{"question": "What are the requirements to register a foreign company in Ghana?"}'
```

```json
{
  "query": "What are the requirements to register a foreign company in Ghana?",
  "answer": "According to the Ghana Companies Act... [1]",
  "citations": [
    {
      "index": 1,
      "document_name": "ghana_companies_act.pdf",
      "passage": "Every foreign company shall, within twenty-eight days..."
    }
  ],
  "no_results": false
}
```

### `GET /api/health`

```json
{ "status": "ok", "database": "ok", "timestamp": "2026-03-31T10:00:00Z" }
```

Full API reference: [`docs/06_api_specification.md`](docs/06_api_specification.md)

---

## Project structure

```
legalbridge/
├── backend/
│   ├── cmd/
│   │   ├── server/         # Entry point
│   │   └── migrate/        # DB migrations runner
│   ├── internal/
│   │   ├── api/            # HTTP handlers (Gin)
│   │   ├── ingester/       # PDF parsing, chunking, embedding
│   │   ├── query/          # Query embedding, search, LLM synthesis
│   │   └── store/          # PostgreSQL / pgvector operations
│   ├── pkg/config/         # Env vars, shared constants (EmbeddingModel)
│   ├── migrations/         # SQL migration files
│   ├── Dockerfile
│   └── docker-compose.yml
│
├── frontend/
│   ├── app/                # Next.js app router
│   ├── components/
│   │   ├── DocumentUpload.tsx
│   │   ├── QueryInput.tsx
│   │   └── AnswerDisplay.tsx
│   └── lib/api.ts          # API client
│
└── docs/                   # Full engineering documentation
    ├── 01_requirements_prd.md
    ├── 02_requirements_srs.md
    ├── 03_design_contract_invariant.md
    ├── 04_transition_req_arch.md
    ├── 05_architecture.md
    ├── 06_api_specification.md
    └── 07_visual_identity.md
```

---

## Documentation

The `docs/` folder contains a complete engineering specification suite:

| Document | Covers |
|----------|--------|
| [`01_requirements_prd.md`](docs/01_requirements_prd.md) | Product vision, personas, MVP scope |
| [`02_requirements_srs.md`](docs/02_requirements_srs.md) | Functional requirements, error contracts |
| [`03_design_contract_invariant.md`](docs/03_design_contract_invariant.md) | System invariants and absolute prohibitions |
| [`04_transition_req_arch.md`](docs/04_transition_req_arch.md) | Invariant ownership by component |
| [`05_architecture.md`](docs/05_architecture.md) | Component design, data model, ADRs |
| [`06_api_specification.md`](docs/06_api_specification.md) | Full REST API + outbound call specs |
| [`07_visual_identity.md`](docs/07_visual_identity.md) | Color system, typography, component styles |

---

## Contributing

Contributions are welcome. Please open an issue before submitting a large pull request.

**Key constraints to preserve:**
- Every answer must include citations — the `INV-02` guarantee must not be weakened
- The embedding model must be the same for ingestion and query — `pkg/config.EmbeddingModel` is the single source of truth
- The LLM must never be called with an empty retrieval result — see `INV-06` in [`docs/03_design_contract_invariant.md`](docs/03_design_contract_invariant.md)

---

## License

MIT — see [`LICENSE`](LICENSE)

---

