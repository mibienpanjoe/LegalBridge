"use client";

import { useCallback } from "react";
import { ArrowRight, Loader2 } from "lucide-react";

interface Props {
  question: string;
  onQuestionChange: (q: string) => void;
  onSubmit: () => void;
  isLoading: boolean;
  error?: string;
  suggestedQuestions?: string[];
}

export default function QueryInput({
  question,
  onQuestionChange,
  onSubmit,
  isLoading,
  error,
  suggestedQuestions,
}: Props) {
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      // Submit on Enter (without Shift).
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        if (!isLoading && question.trim()) onSubmit();
      }
    },
    [isLoading, question, onSubmit]
  );

  const canSubmit = !isLoading && question.trim().length > 0;

  return (
    <section aria-labelledby="query-heading">
      <h2
        id="query-heading"
        className="mb-3 text-xs font-semibold uppercase tracking-widest text-[#4B5563]"
      >
        Ask a question
      </h2>

      {/* Input + button row */}
      <div className="flex gap-2 items-end">
        <div className="relative flex-1">
          <label htmlFor="question-input" className="sr-only">
            Your question
          </label>
          <textarea
            id="question-input"
            rows={2}
            value={question}
            onChange={(e) => onQuestionChange(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={isLoading}
            placeholder="What would you like to know?"
            className="w-full resize-none rounded-lg border border-[#D1D5DB] bg-white px-4 py-3 text-[15px] text-[#111827] placeholder-[#9CA3AF] outline-none transition-shadow focus:border-[#1B3A5C] focus:ring-3 focus:ring-[rgba(27,58,92,0.15)] disabled:cursor-not-allowed disabled:opacity-60"
          />
        </div>
        <button
          onClick={onSubmit}
          disabled={!canSubmit}
          aria-label="Submit question"
          className="flex h-[72px] min-w-[48px] items-center justify-center rounded-lg bg-[#1B3A5C] px-4 text-white transition-all hover:bg-[#162F4A] hover:-translate-y-px hover:shadow-[0_4px_12px_rgba(27,58,92,0.3)] focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-[rgba(27,58,92,0.35)] active:translate-y-0 disabled:cursor-not-allowed disabled:bg-[#9CA3AF] disabled:shadow-none"
        >
          {isLoading ? (
            <Loader2 className="size-5 animate-spin" aria-label="Searching…" />
          ) : (
            <ArrowRight className="size-5" aria-hidden="true" />
          )}
        </button>
      </div>

      {/* Error message */}
      {error && (
        <p role="alert" className="mt-2 flex items-center gap-1.5 text-sm text-[#DC2626]">
          <span aria-hidden="true">✕</span>
          {error}
        </p>
      )}

      {/* Suggested questions */}
      {suggestedQuestions && suggestedQuestions.length > 0 && (
        <div
          className="mt-3 flex flex-wrap gap-2"
          aria-label="Suggested questions"
        >
          {suggestedQuestions.map((q) => (
            <button
              key={q}
              onClick={() => {
                onQuestionChange(q);
              }}
              className="rounded-full border border-[#D1D5DB] bg-white px-3 py-1.5 text-xs text-[#4B5563] transition-colors hover:border-[#C9873A] hover:text-[#C9873A]"
            >
              {q}
            </button>
          ))}
        </div>
      )}
    </section>
  );
}
