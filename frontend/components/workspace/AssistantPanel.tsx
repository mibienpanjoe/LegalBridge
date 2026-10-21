"use client";

import { useState } from "react";
import { BookOpenText, FileSearch, PenLine } from "lucide-react";
import {
  Conversation,
  ConversationContent,
  ConversationScrollButton,
} from "@/components/ai-elements/conversation";
import {
  ChainOfThought,
  ChainOfThoughtSearchResult,
  ChainOfThoughtSearchResults,
  ChainOfThoughtStep,
} from "@/components/ai-elements/chain-of-thought";
import { Suggestion } from "@/components/ai-elements/suggestion";
import Composer from "@/components/workspace/Composer";
import Disclaimer from "@/components/Disclaimer";
import { citationLocation, type Citation, type QueryStage } from "@/lib/api";
import { useLocale } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export type HistoryEntry = {
  question: string;
  answer: string;
  citations: Citation[];
  noResults: boolean;
};

interface Props {
  history: HistoryEntry[];
  /** The question in flight, its current stage and, once found, its passages. */
  pending?: { question: string; stage?: QueryStage; passages?: Citation[] };
  question: string;
  onQuestionChange: (q: string) => void;
  onSubmit: () => void;
  error?: string;
  /** The citation shown in the document viewer, if any. */
  active?: Citation;
  onShowSource: (c: Citation) => void;
}

export default function AssistantPanel({
  history,
  pending,
  question,
  onQuestionChange,
  onSubmit,
  error,
  active,
  onShowSource,
}: Props) {
  const { t } = useLocale();
  const empty = history.length === 0 && !pending;

  return (
    <div className="flex h-full min-h-0 flex-col">
      <Conversation className="min-h-0">
        <ConversationContent className="gap-10 px-5 py-6">
          {empty && (
            <div className="py-6">
              <h2 className="font-serif text-xl font-semibold">{t.emptyThreadTitle}</h2>
              <p className="mt-1 max-w-[52ch] text-sm text-muted-foreground">{t.emptyThreadBody}</p>
              <div className="mt-5 flex flex-col items-start gap-2">
                {t.suggestions.map((s) => (
                  <Suggestion
                    key={s}
                    suggestion={s}
                    onClick={onQuestionChange}
                    className="h-auto min-h-10 whitespace-normal px-4 py-2 text-left font-normal text-muted-foreground hover:border-highlight hover:text-foreground"
                  />
                ))}
              </div>
            </div>
          )}

          {history.map((entry, i) => (
            <AnswerEntry
              key={i}
              entry={entry}
              entryIndex={i}
              active={active}
              onShowSource={onShowSource}
            />
          ))}

          {pending && (
            <PendingEntry
              question={pending.question}
              stage={pending.stage}
              found={pending.passages ?? []}
            />
          )}
        </ConversationContent>
        <ConversationScrollButton />
      </Conversation>

      <div className="border-t bg-background px-5 pb-4 pt-3">
        <Composer
          question={question}
          onQuestionChange={onQuestionChange}
          onSubmit={onSubmit}
          isLoading={!!pending}
          error={error}
        />
        <Disclaimer className="mt-3 text-xs" />
      </div>
    </div>
  );
}

// ─── In-flight question: the streamed pipeline stages ─────────────────────────

const STAGE_ORDER = ["searching", "sources", "generating"] as const;

function PendingEntry({
  question,
  stage,
  found,
}: {
  question: string;
  stage?: QueryStage;
  found: Citation[];
}) {
  const { t } = useLocale();
  const current = STAGE_ORDER.indexOf(stage?.stage ?? "searching");
  const status = (i: number) => (i < current ? "complete" : i === current ? "active" : "pending");

  return (
    <div aria-live="polite" aria-busy="true">
      <QuestionLine question={question} />
      <ChainOfThought defaultOpen className="mt-4 space-y-3">
        <ChainOfThoughtStep icon={FileSearch} label={t.stageSearching} status={status(0)} />
        {current >= 1 && (
          <ChainOfThoughtStep
            icon={BookOpenText}
            label={t.stageSources(found.length, "")}
            status={status(1)}
          >
            {found.some((p) => p.section) && (
              <ChainOfThoughtSearchResults>
                {found.map(
                  (p) =>
                    p.section && (
                      <ChainOfThoughtSearchResult key={p.index} variant="outline">
                        {p.section}
                      </ChainOfThoughtSearchResult>
                    )
                )}
              </ChainOfThoughtSearchResults>
            )}
          </ChainOfThoughtStep>
        )}
        {current >= 2 && (
          <ChainOfThoughtStep icon={PenLine} label={t.stageGenerating} status={status(2)} />
        )}
      </ChainOfThought>
    </div>
  );
}

// ─── Answered question ────────────────────────────────────────────────────────

function AnswerEntry({
  entry,
  entryIndex,
  active,
  onShowSource,
}: {
  entry: HistoryEntry;
  entryIndex: number;
  active?: Citation;
  onShowSource: (c: Citation) => void;
}) {
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

  const select = (c: Citation) => {
    onShowSource(c);
    document
      .getElementById(citationId(entryIndex, c.index))
      ?.scrollIntoView({ behavior: "smooth", block: "nearest" });
  };

  return (
    <article className="animate-in fade-in-0 duration-300">
      <QuestionLine question={entry.question} />

      <p className="mt-4 max-w-[68ch] font-serif text-[17px] leading-[1.7]">
        <AnswerText text={entry.answer} citations={entry.citations} onSelect={select} />
      </p>

      {entry.citations.length > 0 && (
        <div className="mt-6">
          <h3 className="mb-3 text-sm font-medium text-muted-foreground">{t.sources}</h3>
          <ol className="space-y-3">
            {entry.citations.map((c) => (
              <li key={c.index}>
                <SourcePassage
                  citation={c}
                  id={citationId(entryIndex, c.index)}
                  isActive={active === c}
                  onShow={() => onShowSource(c)}
                />
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

// Citation anchors are namespaced by entry so [1] in a later answer does not
// jump to [1] of an earlier one.
function citationId(entryIndex: number, idx: number) {
  return `citation-${entryIndex}-${idx}`;
}

/** The answer with each valid [N] marker turned into a highlighter chip. */
function AnswerText({
  text,
  citations,
  onSelect,
}: {
  text: string;
  citations: Citation[];
  onSelect: (c: Citation) => void;
}) {
  const { t } = useLocale();
  return (
    <>
      {text.split(/(\[\d+\])/g).map((part, i) => {
        const match = part.match(/^\[(\d+)\]$/);
        const citation = match && citations.find((c) => c.index === Number(match[1]));
        if (!citation) return <span key={i}>{part}</span>;
        return (
          <button
            key={i}
            type="button"
            onClick={() => onSelect(citation)}
            className="highlighter mx-0.5 rounded-sm px-1 align-[1px] font-sans text-[13px] font-semibold text-highlight-ink transition-colors hover:bg-highlight-soft focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-ring"
            aria-label={t.jumpToSource(citation.index)}
          >
            {citation.index}
          </button>
        );
      })}
    </>
  );
}

// ─── Source passage ───────────────────────────────────────────────────────────

const PREVIEW_CHARS = 280;

function SourcePassage({
  citation,
  id,
  isActive,
  onShow,
}: {
  citation: Citation;
  id: string;
  isActive: boolean;
  onShow: () => void;
}) {
  const { t } = useLocale();
  const [expanded, setExpanded] = useState(false);
  const isLong = citation.passage.length > PREVIEW_CHARS;
  const displayed =
    expanded || !isLong ? citation.passage : citation.passage.slice(0, PREVIEW_CHARS).trimEnd() + "…";

  return (
    <figure
      id={id}
      className={cn(
        "scroll-mt-4 rounded-lg border bg-card p-4 transition-colors",
        isActive && "border-highlight"
      )}
    >
      <figcaption className="mb-2 flex items-baseline gap-2 text-xs text-muted-foreground">
        <span className="highlighter rounded-sm px-1 font-semibold text-highlight-ink">
          {citation.index}
        </span>
        <span className="min-w-0 flex-1">{citationLocation(citation)}</span>
      </figcaption>
      <blockquote className="whitespace-pre-line font-serif text-[15px] leading-[1.65]">
        {displayed}
      </blockquote>
      <div className="mt-2 flex flex-wrap gap-x-4">
        {citation.page_start ? (
          <button
            type="button"
            onClick={onShow}
            className="min-h-9 text-sm font-medium text-primary underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          >
            {t.showInDocument}
          </button>
        ) : null}
        {isLong && (
          <button
            type="button"
            onClick={() => setExpanded((v) => !v)}
            className="min-h-9 text-sm font-medium text-muted-foreground underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
            aria-expanded={expanded}
          >
            {expanded ? t.showLess : t.showMore}
          </button>
        )}
      </div>
    </figure>
  );
}
