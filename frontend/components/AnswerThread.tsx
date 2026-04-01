"use client";

import { useState } from "react";
import type { HistoryEntry } from "@/app/page";
import type { Citation } from "@/lib/api";

interface Props {
  history: HistoryEntry[];
  isSearching: boolean;
  searchingQuestion?: string;
}

export default function AnswerThread({
  history,
  isSearching,
  searchingQuestion,
}: Props) {
  return (
    <section aria-label="Conversation" className="space-y-6">
      {/* Past Q&A entries */}
      {history.map((entry, i) => (
        <AnswerEntry key={i} entry={entry} />
      ))}

      {/* Loading skeleton for in-flight query */}
      {isSearching && (
        <div aria-live="polite" aria-busy="true">
          {/* Fix 8: show the question being searched */}
          {searchingQuestion && (
            <QuestionBubble question={searchingQuestion} />
          )}
          <div className="mt-3 animate-pulse space-y-3 rounded-lg border border-[#D1D5DB] bg-white px-6 py-5">
            <div className="h-4 w-3/4 rounded bg-[#E5E7EB]" />
            <div className="h-4 w-full rounded bg-[#E5E7EB]" />
            <div className="h-4 w-5/6 rounded bg-[#E5E7EB]" />
          </div>
          <p className="mt-2 text-sm text-[#9CA3AF]">Searching relevant passages…</p>
        </div>
      )}
    </section>
  );
}

// ─── Single Q&A card ──────────────────────────────────────────────────────────

function AnswerEntry({ entry }: { entry: HistoryEntry }) {
  if (entry.noResults) {
    return (
      <div>
        <QuestionBubble question={entry.question} />
        <div className="mt-3 rounded-lg border border-[#D1D5DB] bg-white px-6 py-5 text-sm text-[#4B5563]">
          <p className="font-medium text-[#111827]">No relevant passages found</p>
          <p className="mt-1">
            The document doesn't appear to contain information related to your
            question. Try rephrasing or asking about a different topic.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="animate-in fade-in-0 slide-in-from-bottom-1 duration-300">
      {/* Fix 8: question label */}
      <QuestionBubble question={entry.question} />

      {/* Answer */}
      <div className="mt-3 rounded-lg border border-[#D1D5DB] bg-white px-6 py-5">
        <p className="text-[15px] leading-[1.7] text-[#111827] max-w-[720px]">
          {/* Fix 5: render [N] refs as clickable anchors */}
          <AnswerText text={entry.answer} citations={entry.citations} />
        </p>
      </div>

      {/* Fix 1: collapsible citations */}
      {entry.citations && entry.citations.length > 0 && (
        <div className="mt-4">
          <h3 className="mb-3 text-xs font-semibold uppercase tracking-[0.06em] text-[#4B5563]">
            Sources
          </h3>
          <ul className="space-y-3" aria-label="Source passages">
            {entry.citations.map((c) => (
              <li key={c.index}>
                <CitationBlock citation={c} />
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

// ─── Question bubble ──────────────────────────────────────────────────────────

function QuestionBubble({ question }: { question: string }) {
  return (
    <div className="flex items-start gap-2">
      <span className="mt-0.5 shrink-0 rounded-full bg-[#1B3A5C] px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide text-white">
        You
      </span>
      <p className="text-[15px] font-medium text-[#111827]">{question}</p>
    </div>
  );
}

// ─── Answer text with clickable [N] refs ──────────────────────────────────────

function AnswerText({
  text,
  citations,
}: {
  text: string;
  citations: Citation[];
}) {
  const parts = text.split(/(\[\d+\])/g);

  const scrollTo = (idx: number) => {
    document.getElementById(`citation-${idx}`)?.scrollIntoView({
      behavior: "smooth",
      block: "center",
    });
  };

  return (
    <>
      {parts.map((part, i) => {
        const match = part.match(/^\[(\d+)\]$/);
        if (match) {
          const idx = parseInt(match[1]);
          const exists = citations.some((c) => c.index === idx);
          if (exists) {
            return (
              <button
                key={i}
                onClick={() => scrollTo(idx)}
                className="inline-flex items-center rounded px-1 py-0.5 text-[12px] font-semibold text-white bg-[#C9873A] hover:bg-[#b5762e] transition-colors mx-0.5 align-middle"
                aria-label={`Jump to source ${idx}`}
              >
                {part}
              </button>
            );
          }
        }
        return <span key={i}>{part}</span>;
      })}
    </>
  );
}

// ─── Citation block (collapsible) ─────────────────────────────────────────────

const PREVIEW_CHARS = 220;

function CitationBlock({ citation }: { citation: Citation }) {
  const [expanded, setExpanded] = useState(false);
  const isLong = citation.passage.length > PREVIEW_CHARS;
  const displayed =
    expanded || !isLong
      ? citation.passage
      : citation.passage.slice(0, PREVIEW_CHARS).trimEnd() + "…";

  return (
    <div
      id={`citation-${citation.index}`}
      className="rounded-r-lg border-l-4 border-l-[#C9873A] bg-[#F7F8FA] py-4 pl-4 pr-5"
    >
      {/* Badge */}
      <span className="inline-block rounded px-1.5 py-0.5 text-[11px] font-semibold text-white bg-[#C9873A] mb-2">
        [{citation.index}]
      </span>

      {/* Passage text */}
      <p className="font-mono text-[13px] leading-[1.6] text-[#374151]">
        {displayed}
      </p>

      {/* Fix 1: expand / collapse toggle */}
      {isLong && (
        <button
          onClick={() => setExpanded((v) => !v)}
          className="mt-2 text-[12px] font-medium text-[#C9873A] hover:text-[#b5762e] transition-colors"
          aria-expanded={expanded}
        >
          {expanded ? "Show less ↑" : "Show more ↓"}
        </button>
      )}

      {/* Document label */}
      <p className="mt-2 text-[11px] font-medium text-[#9CA3AF]">
        {citation.document_name}
      </p>
    </div>
  );
}
