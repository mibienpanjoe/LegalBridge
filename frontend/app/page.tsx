"use client";

import { useCallback, useRef, useState } from "react";
import Disclaimer from "@/components/Disclaimer";
import DocumentUpload, { type UploadStatus } from "@/components/DocumentUpload";
import QueryInput from "@/components/QueryInput";
import AnswerThread from "@/components/AnswerThread";
import ThemeToggle from "@/components/ThemeToggle";
import LocaleToggle from "@/components/LocaleToggle";
import {
  ApiError,
  ingestDocument,
  queryDocumentStream,
  type Citation,
  type QueryStage,
} from "@/lib/api";
import { errorMessage, useLocale } from "@/lib/i18n";

// ─── State types ──────────────────────────────────────────────────────────────

// Errors keep the API code so the message is localized at render time.
type Failure = { message: string; code?: string };

type UploadState =
  | { status: "idle" }
  | { status: "uploading" }
  | {
      status: "success";
      documentId: string;
      chunkCount: number;
      filename: string;
      expiresAt?: string;
    }
  | ({ status: "error" } & Failure);

type QueryState =
  | { status: "idle" }
  | { status: "searching"; question: string; stage?: QueryStage }
  | ({ status: "error" } & Failure);

export type HistoryEntry = {
  question: string;
  answer: string;
  citations: Citation[];
  noResults: boolean;
};

function toFailure(err: unknown, fallback: string): Failure {
  if (err instanceof ApiError) return { message: err.message, code: err.code };
  return { message: err instanceof Error ? err.message : fallback };
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function Home() {
  const { t } = useLocale();
  const [uploadState, setUploadState] = useState<UploadState>({ status: "idle" });
  const [queryState, setQueryState] = useState<QueryState>({ status: "idle" });
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const [question, setQuestion] = useState("");
  const answerRef = useRef<HTMLDivElement>(null);

  const handleUpload = useCallback(async (file: File) => {
    setUploadState({ status: "uploading" });
    setQueryState({ status: "idle" });
    setHistory([]);
    try {
      const res = await ingestDocument(file);
      setUploadState({
        status: "success",
        documentId: res.document_id,
        chunkCount: res.chunk_count,
        filename: file.name,
        expiresAt: res.expires_at,
      });
    } catch (err) {
      setUploadState({ status: "error", ...toFailure(err, "Upload failed.") });
    }
  }, []);

  const handleQuery = useCallback(async () => {
    const q = question.trim();
    if (!q || uploadState.status !== "success") return;
    const { documentId } = uploadState;

    setQuestion("");
    setQueryState({ status: "searching", question: q });
    setTimeout(() => {
      answerRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
    }, 80);

    try {
      const res = await queryDocumentStream(documentId, q, (stage) =>
        setQueryState({ status: "searching", question: q, stage })
      );
      setHistory((prev) => [
        ...prev,
        {
          question: q,
          answer: res.answer,
          citations: res.citations,
          noResults: res.no_results,
        },
      ]);
      setQueryState({ status: "idle" });
    } catch (err) {
      const failure = toFailure(err, "Query failed.");
      // The upload expired server-side: go back to the upload step.
      if (failure.code === "DOCUMENT_NOT_FOUND") {
        setUploadState({ status: "error", ...failure });
        setQueryState({ status: "idle" });
        setHistory([]);
        return;
      }
      setQueryState({ status: "error", ...failure });
    }
  }, [question, uploadState]);

  const uploadStatus: UploadStatus =
    uploadState.status === "success" ? "success" : uploadState.status;

  const showQuery = uploadState.status === "success";
  const showThread = history.length > 0 || queryState.status === "searching";

  return (
    <div className="flex min-h-screen flex-col">
      <header className="border-b">
        <div className="mx-auto flex max-w-3xl items-center justify-between gap-4 px-4 py-3 md:px-6">
          <div className="min-w-0">
            <h1 className="font-heading text-xl font-semibold tracking-tight text-primary">
              LegalBridge
            </h1>
            <p className="truncate text-sm text-muted-foreground">{t.tagline}</p>
          </div>
          <div className="flex shrink-0 items-center">
            <LocaleToggle />
            <ThemeToggle />
          </div>
        </div>
      </header>

      <main className="mx-auto w-full max-w-3xl flex-1 space-y-8 px-4 py-8 md:px-6">
        <section aria-labelledby="upload-heading">
          <h2 id="upload-heading" className="mb-2 text-sm font-medium text-muted-foreground">
            {t.uploadHeading}
          </h2>
          <DocumentUpload
            status={uploadStatus}
            chunkCount={uploadState.status === "success" ? uploadState.chunkCount : undefined}
            filename={uploadState.status === "success" ? uploadState.filename : undefined}
            expiresAt={uploadState.status === "success" ? uploadState.expiresAt : undefined}
            error={
              uploadState.status === "error"
                ? errorMessage(t, uploadState.code, uploadState.message)
                : undefined
            }
            onUpload={handleUpload}
          />
        </section>

        {showThread && (
          <div ref={answerRef}>
            <AnswerThread
              history={history}
              isSearching={queryState.status === "searching"}
              searchingQuestion={
                queryState.status === "searching" ? queryState.question : undefined
              }
              searchingStage={queryState.status === "searching" ? queryState.stage : undefined}
            />
          </div>
        )}

        {showQuery && (
          <QueryInput
            question={question}
            onQuestionChange={setQuestion}
            onSubmit={handleQuery}
            isLoading={queryState.status === "searching"}
            error={
              queryState.status === "error"
                ? errorMessage(t, queryState.code, queryState.message)
                : undefined
            }
            showSuggestions={history.length === 0}
          />
        )}

        <Disclaimer />
      </main>

      <footer className="border-t px-4 py-5 text-center text-sm text-muted-foreground">
        {t.footer}
      </footer>
    </div>
  );
}
