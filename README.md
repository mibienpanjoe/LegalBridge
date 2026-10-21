# LegalBridge

<p align="center">
  <img src="docs/assets/screenshot-query.png" width="100%" alt="LegalBridge workspace: the Burkina Faso Labour Code open on page 14 with Articles 43 and 44 highlighted, beside a French answer that cites them" />
</p>

**Ask your legal document a question. Every answer points to the article it comes from.**

LegalBridge is a legal document assistant for West African businesses. Upload a law, a code or a contract as a PDF and ask questions in French or English. LegalBridge finds the passages that answer the question and writes a short answer that cites them. Select a citation and the document opens on that page with the passage highlighted, so you can check every claim against the source text.

It gives legal **information**, not legal advice. The interface says so on every screen.

Built for the **IA digne de confiance (Trustworthy AI)** hackathon category, on open-source models only.

---

## What it does

```
You:  « Que se passe-t-il à la fin de la période d'essai ? »

LegalBridge
  1. finds the passages of your document that answer it (semantic + keyword search)
  2. answers only from those passages, citing them
  3. opens the PDF on page 14 with Articles 43–44 highlighted

Answer: Si le travailleur continue après la fin de l'essai sans renouvellement
        écrit, le contrat devient un contrat à durée indéterminée [1]. …

[1] Article 43–44, p. 14 — « Article 43 : La prolongation des services après
    expiration du contrat à l'essai, sans qu'il n'y ait renouvellement écrit,
    équivaut à un contrat de travail à durée indéterminée… »
```

## Features

- **Cited answers you can verify.** Every answer cites the verbatim passages it uses. Answers without citations are discarded. When no passage is relevant enough, LegalBridge says the document doesn't cover the question instead of guessing.
- **Answers point into the document.** Citations carry the article and page ("Article 43–44, p. 14"). The built-in PDF viewer jumps to that page and highlights the passage.
- **Article-aware search.** Documents are split at article boundaries, and repeated page headers are removed. Each question searches only the document you uploaded, combining semantic similarity (`bge-m3`) with keyword ranking (BM25), so exact terms like *CNSS*, *SARL* or *Article 37* are found.
- **Live progress.** While a question runs you see what's happening: searching, the articles found, then writing the answer.
- **French first, English too.** The interface is in French by default and follows your browser language. Answers come back in the language of your question.
- **Private by default.** Uploaded documents and their index are deleted automatically 24 hours after upload. Questions and answers are never stored.
- **Light and dark themes**, and a layout that works on phones (the document opens as a full-screen sheet).
- **Open-source models only.** No OpenAI, no Anthropic.

<p align="center">
  <img src="docs/assets/screenshot-landing.png" width="80%" alt="LegalBridge landing page in French: upload zone beside a worked example from Article 1 of the Burkina Faso Labour Code, highlighted" />
</p>

---

## Tech stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25 · Gin |
| Frontend | Next.js 16 · React 19 · TypeScript · Tailwind CSS 4 · shadcn/ui (Base UI) |
| AI UI components | [Vercel AI Elements](https://ai-sdk.dev/elements) (conversation, chain-of-thought, suggestion) |
| PDF viewer | react-pdf 11 (pdf.js) |
| Typography | Geist (interface) · Source Serif 4 (answers and quoted law) |
| Vector database | PostgreSQL 16 + pgvector |
| Embeddings | `BAAI/bge-m3`: multilingual, 1024 dimensions, Apache 2.0 |
| LLM | `llama-3.3-70b-versatile` via Groq |
| Local dev | Ollama runs both `bge-m3` and `llama3.2` with no API keys |
| Deployment | Render (backend + database) · Vercel (frontend) |

---

## Architecture

```
┌──────────────────────────────────────────────┐
│  Next.js (Vercel)                            │
│  PDF viewer + assistant · FR/EN · themes     │
└──────────────────────┬───────────────────────┘
                       │ HTTPS (JSON + Server-Sent Events)
┌──────────────────────▼───────────────────────┐
│  Go backend (Render): one binary             │
│                                              │
│  POST /api/ingest                            │
│    PDF → per-page lines (headers removed)    │
│    → article-aware chunks + page/article     │
│    → bge-m3 embeddings → pgvector            │
│                                              │
│  POST /api/query[/stream]                    │
│    document exists? → embed question         │
│    → top 20 of THIS document above 0.45      │
│      cosine (none → "not covered", no LLM)   │
│    → rerank: vector rank + BM25 rank (RRF)   │
│    → top 3 → LLM → citations validated       │
│                                              │
│  hourly sweep: delete uploads older than 24h │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│  PostgreSQL + pgvector                       │
│  documents (expires_at) · chunks (page,      │
│  article, embedding)                         │
└──────────────────────────────────────────────┘

External calls (production):
  router.huggingface.co   ← bge-m3 embeddings
  api.groq.com            ← llama-3.3-70b generation

Local development (no API keys):
  localhost:11434 (Ollama) ← embeddings + generation
```

The guarantees behind this pipeline (grounding, citation completeness, no LLM call on empty retrieval, embedding-model consistency) are specified in [`docs/03_design_contract_invariant.md`](docs/03_design_contract_invariant.md).

---

## Getting started

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Node.js 20.9+](https://nodejs.org/) (required by Next.js 16)
- [Docker + Docker Compose](https://docs.docker.com/get-docker/)
- One of:
  - **Ollama** (local, no accounts): see [ollama.com](https://ollama.com)
  - A **Hugging Face** account with Inference Providers credits, plus a **Groq** account (free tier), for deployed use

### Option A: local development with Ollama (no API keys)

```bash
git clone https://github.com/mibienpanjoe/LegalBridge.git
cd LegalBridge

# Models
ollama pull bge-m3
ollama pull llama3.2

# Database
docker-compose up -d postgres

# Backend configuration
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

Then:

```bash
cd backend
go run ./cmd/migrate      # applies every migration in backend/migrations
go run ./cmd/server       # listens on :8080

cd ../frontend
npm install
npm run dev               # http://localhost:3000
```

On a CPU-only machine the first embedding can take a while as Ollama loads the model; the client waits up to 120 s per call.

### Option B: cloud APIs (Hugging Face + Groq)

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/legalbridge
EMBEDDING_PROVIDER=huggingface
HF_API_KEY=hf_your_token_here
LLM_PROVIDER=groq
GROQ_API_KEY=gsk_your_token_here
```

Hugging Face bills embeddings against your account's Inference Providers credits. When they run out, uploads fail with `EMBEDDING_UNAVAILABLE` (the server log shows the provider's 402).

### Optional: a permanent demo document

`go run ./cmd/seed` ingests the PDF at `DEMO_PDF_PATH` once. Seeded documents never expire; user uploads are deleted after 24 hours.

### Full Docker setup

```bash
docker-compose up --build
```

---

## Environment variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `EMBEDDING_PROVIDER` | Yes | `ollama` or `huggingface` |
| `LLM_PROVIDER` | Yes | `ollama` or `groq` |
| `OLLAMA_BASE_URL` | If Ollama | Default `http://localhost:11434` |
| `OLLAMA_MODEL` | If Ollama (LLM) | Default `llama3.2` |
| `HF_API_KEY` | If Hugging Face | Hugging Face token |
| `GROQ_API_KEY` | If Groq | Groq API key |
| `CORS_ALLOWED_ORIGIN` | Production | The frontend's URL (localhost:3000 is always allowed) |
| `DEMO_PDF_PATH` | For `cmd/seed` | PDF to ingest as the permanent demo document |
| `NEXT_PUBLIC_API_URL` | Frontend | Backend URL, default `http://localhost:8080` |

Retrieval settings live in `backend/pkg/config/config.go`: `MinSimilarity` (0.45), `RetrievalCandidates` (20), `RetrievalTopK` (3), `UploadTTL` (24 h).

---

## API

### `POST /api/ingest`

```bash
curl -X POST http://localhost:8080/api/ingest -F "file=@code-du-travail.pdf"
```

```json
{
  "document_id": "3f4a7b2c-1e6d-4a89-b3c2-9d1e5f7a8b0c",
  "chunk_count": 98,
  "expires_at": "2026-10-22T04:30:00Z"
}
```

### `POST /api/query`

Questions are always about one document, identified by the `document_id` returned at upload.

```bash
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{"document_id": "3f4a7b2c-1e6d-4a89-b3c2-9d1e5f7a8b0c",
       "question": "Que se passe-t-il à la fin de la période d'\''essai ?"}'
```

```json
{
  "answer": "Si le travailleur continue après la fin de l'essai sans renouvellement écrit, le contrat devient un contrat à durée indéterminée [1]. …",
  "citations": [
    {
      "index": 1,
      "document_name": "code-du-travail.pdf",
      "passage": "Article 43 : La prolongation des services après expiration du contrat à l'essai…",
      "page_start": 14,
      "page_end": 14,
      "section": "Article 43–44"
    }
  ],
  "no_results": false
}
```

`no_results: true` means no passage was relevant enough; the LLM is not called. An unknown or expired document returns `404 DOCUMENT_NOT_FOUND`.

### `POST /api/query/stream`

Same request, answered as Server-Sent Events: `stage` events (`searching`, `sources` with the passages found, `generating`), then one `answer` event or an `error` event. Only progress is streamed, never partial answer text, so what you see has already passed citation checks.

### `GET /api/health`

```json
{ "status": "ok", "database": "ok", "timestamp": "2026-10-21T10:00:00Z" }
```

Full reference: [`docs/06_api_specification.md`](docs/06_api_specification.md)

---

## Project structure

```
LegalBridge/
├── backend/
│   ├── cmd/
│   │   ├── server/       # HTTP server + expiry sweeper
│   │   ├── migrate/      # applies backend/migrations/*.sql in order
│   │   └── seed/         # ingests the permanent demo document
│   ├── internal/
│   │   ├── api/          # Gin handlers, error → HTTP mapping, SSE
│   │   ├── ingester/     # PDF pages → article-aware chunks → embeddings
│   │   ├── query/        # retrieval, BM25 + RRF rerank, prompt, citations
│   │   └── store/        # PostgreSQL / pgvector
│   ├── pkg/config/       # env vars + shared constants (EmbeddingModel…)
│   └── migrations/       # 001_init … 004_document_expiry
│
├── frontend/
│   ├── app/              # layout (fonts, theme), page (landing / workspace)
│   ├── components/
│   │   ├── workspace/    # DocumentViewer, AssistantPanel, Composer, Landing
│   │   ├── ai-elements/  # Vercel AI Elements (source, installed via shadcn)
│   │   └── ui/           # shadcn/Base UI primitives
│   └── lib/              # API client (incl. SSE), i18n (FR/EN)
│
└── docs/                 # engineering specification
```

---

## Testing

```bash
cd backend
go test ./...                                   # unit tests

# against a migrated pgvector database
TEST_DATABASE_URL=postgres://… go test -tags integration ./internal/store/...

cd ../frontend
npx tsc --noEmit && npm run lint && npm run build
```

---

## Documentation

| Document | Covers |
|----------|--------|
| [`01_requirements_prd.md`](docs/01_requirements_prd.md) | Product vision, personas, MVP scope |
| [`02_requirements_srs.md`](docs/02_requirements_srs.md) | Functional requirements, privacy, error contracts |
| [`03_design_contract_invariant.md`](docs/03_design_contract_invariant.md) | System invariants and absolute prohibitions |
| [`04_transition_req_arch.md`](docs/04_transition_req_arch.md) | Invariant ownership by component |
| [`05_architecture.md`](docs/05_architecture.md) | Components, retrieval pipeline, data model, stack |
| [`06_api_specification.md`](docs/06_api_specification.md) | REST + SSE API, outbound calls |
| [`07_visual_identity.md`](docs/07_visual_identity.md) | Design tokens, typography, components, layout, tone |

---

## Contributing

Contributions are welcome. Please open an issue before a large pull request.

Keep these guarantees intact:
- Every answer includes citations; uncited answers are discarded (`INV-02`).
- Ingestion and queries use the same embedding model: `pkg/config.EmbeddingModel` is the single source of truth (`INV-04`).
- The LLM is never called without a passage above the similarity threshold (`INV-06`).
- Retrieval never crosses documents: every query is scoped to its `document_id`.

---

## License

MIT
