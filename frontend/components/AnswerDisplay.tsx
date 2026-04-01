import type { Citation } from "@/lib/api";

export type AnswerStatus = "searching" | "success";

interface Props {
  status: AnswerStatus;
  answer?: string;
  citations?: Citation[];
  noResults?: boolean;
}

export default function AnswerDisplay({
  status,
  answer,
  citations,
  noResults,
}: Props) {
  // ─── Loading skeleton ──────────────────────────────────────────────────────
  if (status === "searching") {
    return (
      <section aria-live="polite" aria-busy="true" aria-label="Searching for answer">
        <div className="animate-pulse space-y-3">
          <div className="h-4 w-3/4 rounded bg-[#E5E7EB]" />
          <div className="h-4 w-full rounded bg-[#E5E7EB]" />
          <div className="h-4 w-5/6 rounded bg-[#E5E7EB]" />
        </div>
        <p className="mt-3 text-sm text-[#9CA3AF]">Searching relevant passages…</p>
      </section>
    );
  }

  // ─── No results ────────────────────────────────────────────────────────────
  if (noResults) {
    return (
      <section
        aria-live="polite"
        className="rounded-lg border border-[#D1D5DB] bg-white px-6 py-5 text-sm text-[#4B5563]"
      >
        <p className="font-medium text-[#111827]">No relevant passages found</p>
        <p className="mt-1">
          The uploaded document doesn't appear to contain information related to
          your question. Try rephrasing or asking about a different topic.
        </p>
      </section>
    );
  }

  // ─── Answer + citations ────────────────────────────────────────────────────
  if (!answer) return null;

  return (
    <section
      aria-live="polite"
      aria-label="Answer"
      className="animate-in fade-in-0 slide-in-from-bottom-1 duration-300"
    >
      {/* Answer text */}
      <div className="rounded-lg border border-[#D1D5DB] bg-white px-6 py-5">
        <p className="text-[15px] leading-[1.7] text-[#111827] max-w-[720px]">
          {answer}
        </p>
      </div>

      {/* Citations section */}
      {citations && citations.length > 0 && (
        <div className="mt-6">
          <h3 className="mb-3 text-xs font-semibold uppercase tracking-[0.06em] text-[#4B5563]">
            Sources
          </h3>
          <ul className="space-y-3" aria-label="Source passages">
            {citations.map((c) => (
              <li key={c.index}>
                <CitationBlock citation={c} />
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  );
}

function CitationBlock({ citation }: { citation: Citation }) {
  return (
    <div className="rounded-r-lg border-l-4 border-l-[#C9873A] bg-[#F7F8FA] py-4 pl-4 pr-5">
      {/* Badge */}
      <span className="inline-block rounded px-1.5 py-0.5 text-[11px] font-semibold text-white bg-[#C9873A] mb-2">
        [{citation.index}]
      </span>

      {/* Passage text in monospace */}
      <p className="font-mono text-[13px] leading-[1.6] text-[#374151]">
        {citation.passage}
      </p>

      {/* Document label */}
      <p className="mt-2 text-[11px] font-medium text-[#9CA3AF]">
        {citation.document_name}
      </p>
    </div>
  );
}
