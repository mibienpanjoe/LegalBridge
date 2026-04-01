"use client";

import { useCallback, useState } from "react";
import Disclaimer from "@/components/Disclaimer";
import DocumentUpload, { type UploadStatus } from "@/components/DocumentUpload";
import QueryInput from "@/components/QueryInput";
import AnswerDisplay, { type AnswerStatus } from "@/components/AnswerDisplay";
import { ingestDocument, queryDocument, type QueryResponse } from "@/lib/api";

// ─── State types ──────────────────────────────────────────────────────────────

type UploadState =
  | { status: "idle" }
  | { status: "uploading" }
  | { status: "success"; documentId: string; chunkCount: number; filename: string }
  | { status: "error"; message: string };

type QueryState =
  | { status: "idle" }
  | { status: "searching" }
  | { status: "success"; result: QueryResponse }
  | { status: "error"; message: string };

// ─── Suggested questions for the demo ─────────────────────────────────────────

const SUGGESTED_QUESTIONS = [
  "What are the requirements to register a company?",
  "How long does registration take?",
  "What documents are needed for registration?",
];

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function Home() {
  const [uploadState, setUploadState] = useState<UploadState>({ status: "idle" });
  const [queryState, setQueryState] = useState<QueryState>({ status: "idle" });
  const [question, setQuestion] = useState("");

  const handleUpload = useCallback(async (file: File) => {
    setUploadState({ status: "uploading" });
    setQueryState({ status: "idle" });
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
    setQueryState({ status: "searching" });
    try {
      const res = await queryDocument(q);
      setQueryState({ status: "success", result: res });
    } catch (err) {
      setQueryState({
        status: "error",
        message: err instanceof Error ? err.message : "Query failed.",
      });
    }
  }, [question]);

  const uploadStatus: UploadStatus =
    uploadState.status === "idle" || uploadState.status === "uploading" || uploadState.status === "error"
      ? uploadState.status
      : "success";

  const showQuery = uploadState.status === "success";
  const showAnswer =
    queryState.status === "searching" || queryState.status === "success";

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
        {/* BR-01: always visible */}
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

        {/* Answer display */}
        {showAnswer && (
          <AnswerDisplay
            status={queryState.status as AnswerStatus}
            answer={
              queryState.status === "success"
                ? queryState.result.answer
                : undefined
            }
            citations={
              queryState.status === "success"
                ? queryState.result.citations
                : undefined
            }
            noResults={
              queryState.status === "success"
                ? queryState.result.no_results
                : undefined
            }
          />
        )}
      </main>

      {/* ── Footer ─────────────────────────────────────────────────────────── */}
      <footer className="border-t border-[#D1D5DB] px-4 py-6 text-center text-sm text-[#9CA3AF]">
        Powered by open-source AI · Built for West Africa
      </footer>
    </div>
  );
}
