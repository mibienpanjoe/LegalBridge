"use client";

import { useState } from "react";
import type { HistoryEntry } from "@/app/page";
import { citationLocation, type Citation, type QueryStage } from "@/lib/api";
import { useLocale } from "@/lib/i18n";

interface Props {
  history: HistoryEntry[];
  isSearching: boolean;
  searchingQuestion?: string;
  searchingStage?: QueryStage;
}

export default function AnswerThread({
  history,
  isSearching,
  searchingQuestion,
  searchingStage,
}: Props) {
  const { t } = useLocale();

  const stageLabel = (stage?: QueryStage) => {
    if (stage?.stage === "generating") return t.stageGenerating;
    if (stage?.stage === "sources") {
      const where = stage.passages
        .map((p) => p.section)
        .filter(Boolean)
        .join(", ");
      return t.stageSources(stage.passages.length, where);
    }
    return t.stageSearching;
  };

  return (
    <section aria-label={t.askHeading} className="space-y-10">
      {history.map((entry, i) => (
        <AnswerEntry key={i} entry={entry} entryIndex={i} />
      ))}

      {isSearching && (
        <div aria-live="polite" aria-busy="true">
          {searchingQuestion && <QuestionLine question={searchingQuestion} />}
          <div className="mt-4 space-y-2.5" aria-hidden="true">
            <div className="h-3.5 w-3/4 animate-pulse rounded bg-muted" />
            <div className="h-3.5 w-full animate-pulse rounded bg-muted" />
            <div className="h-3.5 w-5/6 animate-pulse rounded bg-muted" />
          </div>
          <p className="mt-3 text-sm text-muted-foreground">{stageLabel(searchingStage)}</p>
        </div>
      )}
    </section>
  );
}

// ─── Single Q&A ───────────────────────────────────────────────────────────────

function AnswerEntry({ entry, entryIndex }: { entry: HistoryEntry; entryIndex: number }) {
  const { t } = useLocale();

  if (entry.noResults) {
    return (
      <div>
        <QuestionLine question={entry.question} />
        <div className="mt-4 border-l-2 border-border pl-4">
          <p className="font-medium">{t.noResultsTitle}</p>
          <p className="mt-1 text-sm text-muted-foreground">{t.noResultsBody}</p>
        </div>
      </div>
    );
  }

  return (
    <article className="animate-in fade-in-0 duration-300">
      <QuestionLine question={entry.question} />

      <p className="mt-4 max-w-[68ch] font-serif text-[17px] leading-[1.7]">
        <AnswerText text={entry.answer} citations={entry.citations} entryIndex={entryIndex} />
      </p>

      {entry.citations && entry.citations.length > 0 && (
        <div className="mt-6">
          <h3 className="mb-3 text-sm font-medium text-muted-foreground">{t.sources}</h3>
          <ol className="space-y-3">
            {entry.citations.map((c) => (
              <li key={c.index}>
                <CitationBlock citation={c} entryIndex={entryIndex} />
              </li>
            ))}
          </ol>
        </div>
      )}
    </article>
  );
}

function QuestionLine({ question }: { question: string }) {
  const { t } = useLocale();
  return (
    <p className="text-[15px] font-medium">
      <span className="sr-only">{t.you}: </span>
      {question}
    </p>
  );
}

// ─── Answer text with clickable [N] refs ──────────────────────────────────────

// Citation anchors are namespaced by entry so [1] in a later answer does not
// jump to [1] of an earlier one.
function citationId(entryIndex: number, idx: number) {
  return `citation-${entryIndex}-${idx}`;
}

function AnswerText({
  text,
  citations,
  entryIndex,
}: {
  text: string;
  citations: Citation[];
  entryIndex: number;
}) {
  const { t } = useLocale();
  const parts = text.split(/(\[\d+\])/g);

  return (
    <>
      {parts.map((part, i) => {
        const match = part.match(/^\[(\d+)\]$/);
        if (match) {
          const idx = parseInt(match[1]);
          if (citations.some((c) => c.index === idx)) {
            return (
              <button
                key={i}
                type="button"
                onClick={() =>
                  document
                    .getElementById(citationId(entryIndex, idx))
                    ?.scrollIntoView({ behavior: "smooth", block: "center" })
                }
                className="highlighter mx-0.5 rounded-sm px-1 font-sans text-[13px] font-semibold text-highlight-ink align-[1px] transition-colors hover:bg-highlight-soft focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-ring"
                aria-label={t.jumpToSource(idx)}
              >
                {idx}
              </button>
            );
          }
        }
        return <span key={i}>{part}</span>;
      })}
    </>
  );
}

// ─── Source passage (collapsible) ─────────────────────────────────────────────

const PREVIEW_CHARS = 280;

function CitationBlock({ citation, entryIndex }: { citation: Citation; entryIndex: number }) {
  const { t } = useLocale();
  const [expanded, setExpanded] = useState(false);
  const isLong = citation.passage.length > PREVIEW_CHARS;
  const displayed =
    expanded || !isLong
      ? citation.passage
      : citation.passage.slice(0, PREVIEW_CHARS).trimEnd() + "…";

  return (
    <figure
      id={citationId(entryIndex, citation.index)}
      className="scroll-mt-24 rounded-lg border bg-card p-4"
    >
      <figcaption className="mb-2 flex items-baseline gap-2 text-xs text-muted-foreground">
        <span className="highlighter rounded-sm px-1 font-semibold text-highlight-ink">
          {citation.index}
        </span>
        <span>{citationLocation(citation)}</span>
      </figcaption>
      <blockquote className="whitespace-pre-line font-serif text-[15px] leading-[1.65]">
        {displayed}
      </blockquote>
      {isLong && (
        <button
          type="button"
          onClick={() => setExpanded((v) => !v)}
          className="mt-2 min-h-9 text-sm font-medium text-highlight-ink underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          aria-expanded={expanded}
        >
          {expanded ? t.showLess : t.showMore}
        </button>
      )}
    </figure>
  );
}
