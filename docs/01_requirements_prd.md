# LegalBridge — Product Requirements Document
Version: v1.0, 2026-03-30

## 1. Problem Statement

West African SMEs and businesses operate across multiple national jurisdictions (Ghana, Nigeria, Côte d'Ivoire, Burkina Faso) with complex, fragmented legal frameworks. Legal documents are written in multiple languages, scattered across government gazettes, and difficult to interpret without professional legal training. Hiring a lawyer is expensive and inaccessible for most small businesses. Cross-border trade requires simultaneous understanding of multiple regulatory regimes, compounding the difficulty.

The result: businesses unknowingly violate compliance requirements, miss tax deadlines, fail to register properly, or make uninformed decisions that expose them to legal risk — not out of negligence, but out of lack of accessible, affordable legal information.

## 2. Personas

### Primary — West African SME Owner
- **Who:** A business owner in Ghana, Nigeria, or Côte d'Ivoire with 1–50 employees
- **Context:** Handles legal questions personally or delegates to staff; cannot afford a retained lawyer
- **Need:** Quick, plain-language answers to questions like "How do I register a foreign company in Ghana?" or "What are my employer tax obligations in Nigeria?"
- **Frustration:** Government websites are verbose, technical, and sometimes in a language they don't prefer; lawyers take days and charge by the hour

### Secondary — Cross-Border Trader / Startup Founder
- **Who:** Entrepreneur doing business across multiple West African countries
- **Context:** Needs to understand compliance in 2+ jurisdictions simultaneously
- **Need:** Answers that reference actual legal text, not generic summaries
- **Frustration:** Can't trust paraphrased overviews; needs to see the source passage

### Tertiary — Legal Consultant / Paralegal
- **Who:** Junior legal professional using the tool as a research aid
- **Context:** Has legal training but needs to quickly locate and cite specific statutory passages
- **Need:** Accurate passage retrieval with direct document references
- **Frustration:** Manual document search is slow; needs verified citations for client advice

## 3. Solution Overview

LegalBridge is a Retrieval-Augmented Generation (RAG) system that allows users to upload legal PDF documents and ask questions in plain language. The system extracts and indexes document content into a vector database, then retrieves the most semantically relevant passages when a question is posed. A Claude-powered language model synthesizes a precise, cited answer from those passages — never inventing information not present in the source documents.

Every answer includes citations to the exact passages used, ensuring the user can verify the AI's response against the original legal text.

## 4. MVP Scope

### Document Ingestion
- Upload a legal PDF document via the web interface
- System extracts text, splits into 500-token chunks with overlap, and generates vector embeddings
- Indexed document is immediately queryable

### Natural Language Q&A
- User types a question in plain language (English or French)
- System retrieves the top 3 most relevant passages from the indexed document
- Claude synthesizes a precise, cited answer from the retrieved passages

### Cited Answer Display
- Each answer includes the specific passages that support it
- Passages reference the source document by name
- Users can see exactly what text the AI drew from

### Pre-Loaded Demo Document
- Ghana Companies Act (or Nigeria tax code) pre-loaded for hackathon demo
- Demo questions pre-tested and verified to produce accurate cited answers

### Error Handling
- Graceful fallback if the LLM is unavailable
- Clear user-facing messages for upload failures or empty query results

## 5. Out of Scope (for MVP)

- Contract drafting or review
- Regulatory monitoring and automated alerts
- Machine translation between languages (English ↔ French ↔ Portuguese)
- User authentication, accounts, or subscriptions
- Managing multiple documents simultaneously
- Document version management or update tracking
- Kubernetes or enterprise-scale infrastructure
- Offline or mobile-native application
- Legal judgment or advice (the system provides information, not legal counsel)

## 6. Success Criteria

- Live demo responds to pre-written questions with cited answers in under 5 seconds, reliably (10/10 demo runs)
- Every answer includes at least one citation referencing the source document
- At least one real legal document (Ghana Companies Act or Nigeria tax code) is pre-loaded and queryable
- The system is deployed and publicly accessible (Vercel frontend + Railway/Render backend)
- Error handling prevents demo failures: API timeouts show a fallback message; upload failures show a clear error
- A non-technical user can ask a question and receive a cited answer without instructions
