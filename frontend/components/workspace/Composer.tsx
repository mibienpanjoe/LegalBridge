"use client";

import { ArrowUp, Loader2 } from "lucide-react";
import { useLocale } from "@/lib/i18n";

interface Props {
  question: string;
  onQuestionChange: (q: string) => void;
  onSubmit: () => void;
  isLoading: boolean;
  error?: string;
}

export default function Composer({
  question,
  onQuestionChange,
  onSubmit,
  isLoading,
  error,
}: Props) {
  const { t } = useLocale();
  const canSubmit = !isLoading && question.trim().length > 0;

  return (
    <div>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (canSubmit) onSubmit();
        }}
        className="flex items-end gap-2 rounded-xl border bg-card p-2 transition-shadow focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/15"
      >
        <label htmlFor="question-input" className="sr-only">
          {t.askLabel}
        </label>
        <textarea
          id="question-input"
          rows={2}
          value={question}
          onChange={(e) => onQuestionChange(e.target.value)}
          onKeyDown={(e) => {
            // Enter submits; Shift+Enter adds a line.
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              if (canSubmit) onSubmit();
            }
          }}
          disabled={isLoading}
          placeholder={t.askPlaceholder}
          className="min-h-11 flex-1 resize-none bg-transparent px-2 py-2 text-[15px] outline-none placeholder:text-muted-foreground disabled:opacity-60"
        />
        <button
          type="submit"
          disabled={!canSubmit}
          aria-label={t.askSubmit}
          className="inline-flex size-11 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground transition-colors hover:bg-primary/90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring disabled:bg-muted disabled:text-muted-foreground"
        >
          {isLoading ? (
            <Loader2 className="size-5 animate-spin" aria-hidden="true" />
          ) : (
            <ArrowUp className="size-5" aria-hidden="true" />
          )}
        </button>
      </form>

      {error && (
        <p role="alert" className="mt-2 text-sm text-destructive">
          {error}
        </p>
      )}
    </div>
  );
}
