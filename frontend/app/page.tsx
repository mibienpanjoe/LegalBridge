"use client";

import { useCallback, useRef, useState } from "react";
import Disclaimer from "@/components/Disclaimer";
import DocumentUpload, { type UploadStatus } from "@/components/DocumentUpload";
import QueryInput from "@/components/QueryInput";
import AnswerThread from "@/components/AnswerThread";
import { ingestDocument, queryDocument, type Citation } from "@/lib/api";

// ─── State types ──────────────────────────────────────────────────────────────

type UploadState =
  | { status: "idle" }
  | { status: "uploading" }
  | { status: "success"; documentId: string; chunkCount: number; filename: string }
  | { status: "error"; message: string };

type QueryState =
  | { status: "idle" }
  | { status: "searching"; question: string }
  | { status: "error"; message: string };

export type HistoryEntry = {
  question: string;
  answer: string;
  citations: Citation[];
  noResults: boolean;
};

// ─── Suggested questions (generic legal, bilingual) ───────────────────────────

const SUGGESTED_QUESTIONS = [
  "Quels sont les droits et obligations des parties ?",
  "Quelles sont les conditions requises ?",
  "Quelles sont les sanctions prévues ?",
];

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function Home() {
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
      });
    } catch (err) {
      setUploadState({
        status: "error",
        message: err instanceof Error ? err.message : "Upload failed.",
      });
    }
  }, []);

  const handleQuery = useCallback(async () => {
    const q = question.trim();
    if (!q) return;

    // Fix 4: clear input immediately after submit
    setQuestion("");
    setQueryState({ status: "searching", question: q });

    // Fix 6: scroll to the answer area
    setTimeout(() => {
      answerRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" });
    }, 80);

    try {
      const res = await queryDocument(q);
      // Fix 3: push to history instead of replacing
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
      setQueryState({
        status: "error",
        message: err instanceof Error ? err.message : "Query failed.",
      });
    }
  }, [question]);

  const uploadStatus: UploadStatus =
    uploadState.status === "idle" ||
    uploadState.status === "uploading" ||
    uploadState.status === "error"
      ? uploadState.status
      : "success";

  const showQuery = uploadState.status === "success";
  const showThread =
    history.length > 0 ||
    queryState.status === "searching" ||
    queryState.status === "error";

  return (
    <div className="flex min-h-screen flex-col">
      {/* ── Header ─────────────────────────────────────────────────────────── */}
      <header className="bg-[#1B3A5C] px-4 py-6 md:px-12">
        <div className="mx-auto max-w-4xl">
          <h1 className="font-heading text-2xl font-bold text-white">
            LegalBridge
          </h1>
          <p className="mt-1 text-sm text-white/70">
            Your legal documents, explained.
          </p>
        </div>
      </header>

      {/* ── Main content ───────────────────────────────────────────────────── */}
      <main className="mx-auto w-full max-w-4xl flex-1 space-y-6 px-4 py-8 md:px-12">
        {/* Fix 7: dismissible disclaimer */}
        <Disclaimer />

        {/* Upload zone */}
        <section aria-labelledby="upload-heading">
          <h2
            id="upload-heading"
            className="mb-3 text-xs font-semibold uppercase tracking-widest text-[#4B5563]"
          >
            Upload document
          </h2>
          <DocumentUpload
            status={uploadStatus}
            chunkCount={
              uploadState.status === "success" ? uploadState.chunkCount : undefined
            }
            filename={
              uploadState.status === "success" ? uploadState.filename : undefined
            }
            error={
              uploadState.status === "error" ? uploadState.message : undefined
            }
            onUpload={handleUpload}
          />
        </section>

        {/* Query input — only visible after a successful upload */}
        {showQuery && (
          <QueryInput
            question={question}
            onQuestionChange={setQuestion}
            onSubmit={handleQuery}
            isLoading={queryState.status === "searching"}
            error={
              queryState.status === "error" ? queryState.message : undefined
            }
            suggestedQuestions={SUGGESTED_QUESTIONS}
          />
        )}

        {/* Fix 3: conversation thread */}
        {showThread && (
          <div ref={answerRef}>
            <AnswerThread
              history={history}
              isSearching={queryState.status === "searching"}
              searchingQuestion={
                queryState.status === "searching" ? queryState.question : undefined
              }
            />
          </div>
        )}
      </main>

      {/* ── Footer ─────────────────────────────────────────────────────────── */}
      <footer className="border-t border-[#D1D5DB] px-4 py-6 text-center text-sm text-[#9CA3AF]">
        Powered by open-source AI · Built for West Africa
      </footer>
    </div>
  );
}
