const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export interface Citation {
  index: number;
  document_name: string;
  passage: string;
  page_start?: number;
  page_end?: number;
  section?: string;
}

/** "Article 12 · p. 4–5 · code.pdf" — the parts of a citation's location that are known. */
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
  return parts.join(" · ");
}

export interface IngestResponse {
  document_id: string;
  chunk_count: number;
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

async function parseError(res: Response): Promise<Error> {
  try {
    const body: ApiErrorBody = await res.json();
    return new Error(body.error?.message ?? `Request failed (${res.status})`);
  } catch {
    return new Error(`Request failed (${res.status})`);
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
