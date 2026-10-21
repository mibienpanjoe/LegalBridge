const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export interface Citation {
  index: number;
  document_name: string;
  passage: string;
  page_start?: number;
  page_end?: number;
  section?: string;
}

/** "Article 12, p. 4–5, code.pdf": the parts of a citation's location that are known. */
export function citationLocation(c: Citation): string {
  const parts: string[] = [];
  if (c.section) parts.push(c.section);
  if (c.page_start) {
    parts.push(
      c.page_end && c.page_end > c.page_start
        ? `p. ${c.page_start}–${c.page_end}`
        : `p. ${c.page_start}`
    );
  }
  parts.push(c.document_name);
  return parts.join(", ");
}

export interface IngestResponse {
  document_id: string;
  chunk_count: number;
  /** ISO timestamp after which the upload is deleted; absent if it never expires. */
  expires_at?: string;
}

export interface QueryResponse {
  answer: string;
  citations: Citation[];
  no_results: boolean;
}

interface ApiErrorBody {
  error: {
    code: string;
    message: string;
  };
}

/** An error returned by the API, carrying its machine-readable code. */
export class ApiError extends Error {
  constructor(
    message: string,
    readonly code?: string
  ) {
    super(message);
  }
}

async function parseError(res: Response): Promise<ApiError> {
  try {
    const body: ApiErrorBody = await res.json();
    return new ApiError(
      body.error?.message ?? `Request failed (${res.status})`,
      body.error?.code
    );
  } catch {
    return new ApiError(`Request failed (${res.status})`);
  }
}

export async function ingestDocument(file: File): Promise<IngestResponse> {
  const form = new FormData();
  form.append("file", file);

  const res = await fetch(`${API_BASE}/api/ingest`, {
    method: "POST",
    body: form,
  });

  if (!res.ok) throw await parseError(res);
  return res.json();
}

export async function queryDocument(
  documentId: string,
  question: string
): Promise<QueryResponse> {
  const res = await fetch(`${API_BASE}/api/query`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ document_id: documentId, question }),
  });

  if (!res.ok) throw await parseError(res);
  return res.json();
}

export type QueryStage =
  | { stage: "searching" }
  | { stage: "sources"; passages: Citation[] }
  | { stage: "generating" };

/**
 * Asks a question over `POST /api/query/stream`, reporting pipeline stages to
 * `onStage` as Server-Sent Events arrive. Resolves with the validated answer.
 */
export async function queryDocumentStream(
  documentId: string,
  question: string,
  onStage: (stage: QueryStage) => void,
  signal?: AbortSignal
): Promise<QueryResponse> {
  const res = await fetch(`${API_BASE}/api/query/stream`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ document_id: documentId, question }),
    signal,
  });
  if (!res.ok || !res.body) throw await parseError(res);

  const reader = res.body.pipeThrough(new TextDecoderStream()).getReader();
  let buffer = "";
  for (;;) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += value;

    let sep: number;
    while ((sep = buffer.indexOf("\n\n")) !== -1) {
      const block = buffer.slice(0, sep);
      buffer = buffer.slice(sep + 2);

      let event = "message";
      let data = "";
      for (const line of block.split("\n")) {
        if (line.startsWith("event: ")) event = line.slice(7);
        else if (line.startsWith("data: ")) data += line.slice(6);
      }
      if (!data) continue;

      const payload = JSON.parse(data);
      if (event === "stage") onStage(payload as QueryStage);
      else if (event === "answer") return payload as QueryResponse;
      else if (event === "error") {
        const err = (payload as ApiErrorBody).error;
        throw new ApiError(err?.message ?? "Query failed.", err?.code);
      }
    }
  }
  throw new ApiError("The answer stream ended unexpectedly.");
}
