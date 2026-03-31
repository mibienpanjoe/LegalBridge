# LegalBridge — System Contract & Invariants
Version: v1.0, 2026-03-31

## Actors & Allowed Actions

| Actor | Permitted Actions |
|-------|-------------------|
| User | Upload PDF documents; submit natural language questions; read synthesized answers and citations |
| Ingester | Read uploaded file bytes; call embedding API (`bge-m3`); write document records and chunk records (with embeddings) to Store |
| QueryProcessor | Read question input; call embedding API (`bge-m3`); read chunks from Store via similarity search; call Groq API; return answer and citations |
| Store | Read and write to `documents` and `chunks` tables in PostgreSQL; execute pgvector cosine similarity queries |
| Embedding Provider | Receive text input; return 1024-dimensional float vectors. HuggingFace Inference API (`BAAI/bge-m3`) in production; Ollama locally. |
| LLM Provider | Receive RAG prompt (question + retrieved passages); return synthesized text. Groq API (`llama-3.3-70b-versatile`) in production; Ollama locally. |

**What no actor may do:**
- No actor may modify the `content` field of a stored chunk after it is written
- No actor may call the Groq API without first retrieving at least one document passage above the similarity threshold
- No actor may return a synthesized answer to the user without attached citations
- No actor may embed a question using a model that differs from the model used during ingestion

---

## System Guarantees (Invariants)

### AI Behavior

**INV-01 — Answer Grounding**
The synthesized answer MUST be derived exclusively from the passages retrieved from the vector database and included in the RAG prompt. The LLM prompt MUST contain an explicit instruction prohibiting the model from supplementing with external legal knowledge. A response containing legal claims not traceable to the retrieved passages is a contract violation.

**INV-02 — Citation Completeness**
Every answer returned to the user MUST include at least one citation. A citation consists of: (1) the source document name and (2) the verbatim passage text from which the claim was drawn. An answer returned without citations is a contract violation, regardless of whether the answer content is accurate.

**INV-03 — No Legal Counsel**
The system MUST NOT present itself as providing legal advice, legal opinions, or legal counsel. Every response pathway — including the LLM system prompt, the UI disclaimer, and error messages — MUST treat the system as an information retrieval and synthesis tool. Any representation as a legal advisor is a contract violation.

### Embedding Integrity

**INV-04 — Embedding Model Consistency**
The embedding model used to encode document chunks during ingestion MUST be identical to the model used to encode queries during retrieval. A mismatch renders cosine similarity scores meaningless and produces incorrect retrieval results. This model identifier MUST be stored as a single shared constant (`EMBEDDING_MODEL = "bge-m3"`), never duplicated across call sites.

### Document Integrity

**INV-05 — Passage Fidelity**
The `content` field written to each chunk record MUST be the verbatim extracted text from the source PDF. No summarization, paraphrasing, reformulation, or transformation of the source text is permitted between extraction and storage. Citation display MUST use the stored `content` field directly, not a re-generated or reformatted version.

### Query Safety

**INV-06 — No Fabrication on Empty Retrieval**
If cosine similarity search returns no passages above the minimum similarity threshold, the system MUST return a "no relevant passages found" response and MUST NOT proceed to call the Groq API. Calling the LLM with empty or inadequate context and returning the hallucinated result as a cited answer is a contract violation.

**INV-07 — Response Language Consistency**
The LLM prompt MUST instruct the model to respond in the same language as the user's question. A French question MUST produce a French answer; an English question MUST produce an English answer. The language instruction MUST be included in every call to the Groq API.

### Infrastructure

**INV-08 — API Key Secrecy**
API credentials (`HF_API_KEY`, `GROQ_API_KEY`) MUST exist only in runtime environment variables. They MUST NOT appear in source code, committed configuration files, server logs, or HTTP responses. The application MUST fail at startup if required keys are missing.

---

## Absolute Prohibitions

| ID | The system MUST NEVER... |
|----|--------------------------|
| FRB-01 | Generate an answer containing legal claims not present in the retrieved document passages |
| FRB-02 | Return an answer to the user without at least one citation referencing a source passage |
| FRB-03 | Represent itself as providing legal advice, legal counsel, or legal opinion |
| FRB-04 | Use a different embedding model for query processing than the model used during document ingestion |
| FRB-05 | Modify, paraphrase, or summarize stored chunk text — citations must quote verbatim extracted text |
| FRB-06 | Call the Groq API when retrieved context is empty or below the similarity threshold |
| FRB-07 | Expose API keys in source code, configuration files, logs, or HTTP responses |
| FRB-08 | Create a partial document record when PDF text extraction fails |

---

## Exception Handlers

| ID | Trigger | Contracted Recovery |
|----|---------|---------------------|
| EXC-01 | LLM returns a response with no citations | Discard response; return: "Unable to generate a cited answer. Please rephrase your question." |
| EXC-02 | Similarity search returns zero results above threshold | Return `no_results: true` response immediately; do NOT call Groq API |
| EXC-03 | Groq API timeout (>10s) or non-2xx response | Return HTTP 503 fallback; do NOT retry synchronously |
| EXC-04 | Embedding API (HuggingFace/Ollama) failure during ingestion | Return HTTP 503; roll back — do not write document or chunk records |
| EXC-05 | Embedding API (HuggingFace/Ollama) failure during query | Return HTTP 503; do not proceed to similarity search |
| EXC-06 | PDF text extraction yields empty string | Return HTTP 422; do not create document record in the database |
| EXC-07 | Application starts with missing API key | Panic at startup with message: "Missing required env var: [KEY_NAME]" |
