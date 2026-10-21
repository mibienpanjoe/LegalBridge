"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import dynamic from "next/dynamic";
import { ArrowLeft, FileText } from "lucide-react";
import Landing from "@/components/workspace/Landing";
import AssistantPanel, { type HistoryEntry } from "@/components/workspace/AssistantPanel";
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

// pdf.js needs the browser; never render the viewer on the server.
const DocumentViewer = dynamic(() => import("@/components/workspace/DocumentViewer"), {
  ssr: false,
});

// ─── State types ──────────────────────────────────────────────────────────────

// Errors keep the API code so the message is localized at render time.
type Failure = { message: string; code?: string };

type UploadState =
  | { status: "idle" }
  | { status: "uploading" }
  | {
      status: "success";
      file: File;
      documentId: string;
      chunkCount: number;
      expiresAt?: string;
    }
  | ({ status: "error" } & Failure);

type QueryState =
  | { status: "idle" }
  | { status: "searching"; question: string; stage?: QueryStage; passages?: Citation[] }
  | ({ status: "error" } & Failure);

function toFailure(err: unknown, fallback: string): Failure {
  if (err instanceof ApiError) return { message: err.message, code: err.code };
  return { message: err instanceof Error ? err.message : fallback };
}

// Matches Tailwind's lg breakpoint, where the viewer sits beside the answers.
const SPLIT_QUERY = "(min-width: 1024px)";

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function Home() {
  const { t, locale } = useLocale();
  const [uploadState, setUploadState] = useState<UploadState>({ status: "idle" });
  const [queryState, setQueryState] = useState<QueryState>({ status: "idle" });
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const [question, setQuestion] = useState("");
  const [active, setActive] = useState<Citation>();
  const [sheetOpen, setSheetOpen] = useState(false); // viewer on small screens
  const replaceRef = useRef<HTMLInputElement>(null);

  const handleUpload = useCallback(async (file: File) => {
    setUploadState({ status: "uploading" });
    setQueryState({ status: "idle" });
    setHistory([]);
    setActive(undefined);
    try {
      const res = await ingestDocument(file);
      setUploadState({
        status: "success",
        file,
        documentId: res.document_id,
        chunkCount: res.chunk_count,
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

    try {
      const res = await queryDocumentStream(documentId, q, (stage) =>
        setQueryState((prev) => ({
          status: "searching",
          question: q,
          stage,
          passages:
            stage.stage === "sources"
              ? stage.passages
              : prev.status === "searching"
                ? prev.passages
                : undefined,
        }))
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
      // Point the document at the first source of the new answer.
      if (res.citations[0]?.page_start) setActive(res.citations[0]);
    } catch (err) {
      const failure = toFailure(err, "Query failed.");
      // The upload expired server-side: go back to the upload step.
      if (failure.code === "DOCUMENT_NOT_FOUND") {
        setUploadState({ status: "error", ...failure });
        setQueryState({ status: "idle" });
        setHistory([]);
        setActive(undefined);
        return;
      }
      setQueryState({ status: "error", ...failure });
    }
  }, [question, uploadState]);

  const showSource = useCallback((c: Citation) => {
    setActive(c);
    if (!window.matchMedia(SPLIT_QUERY).matches) setSheetOpen(true);
  }, []);

  // Close the small-screen viewer with Escape.
  useEffect(() => {
    if (!sheetOpen) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setSheetOpen(false);
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [sheetOpen]);

  const toggles = (
    <div className="flex shrink-0 items-center">
      <LocaleToggle />
      <ThemeToggle />
    </div>
  );

  // ── Before a document is loaded ────────────────────────────────────────────
  if (uploadState.status !== "success") {
    return (
      <div className="flex min-h-dvh flex-col">
        <header className="border-b">
          <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-2 md:px-8">
            <h1 className="font-heading text-xl font-semibold tracking-tight text-primary">
              LegalBridge
            </h1>
            {toggles}
          </div>
        </header>
        <Landing
          status={uploadState.status}
          error={
            uploadState.status === "error"
              ? errorMessage(t, uploadState.code, uploadState.message)
              : undefined
          }
          onUpload={handleUpload}
        />
        <footer className="border-t px-4 py-5 text-center text-sm text-muted-foreground">
          {t.footer}
        </footer>
      </div>
    );
  }

  // ── Workspace ──────────────────────────────────────────────────────────────
  const { file, chunkCount, expiresAt } = uploadState;
  const when =
    expiresAt &&
    new Date(expiresAt).toLocaleString(locale, {
      weekday: "long",
      hour: "2-digit",
      minute: "2-digit",
    });

  return (
    <div className="flex h-dvh flex-col">
      <header className="flex items-center gap-3 border-b px-4 py-2">
        <h1 className="font-heading text-lg font-semibold tracking-tight text-primary">
          LegalBridge
        </h1>
        <div className="flex min-w-0 flex-1 items-center gap-2 border-l pl-3">
          <FileText className="size-4 shrink-0 text-muted-foreground" strokeWidth={1.75} aria-hidden="true" />
          <div className="min-w-0">
            <p className="truncate text-sm font-medium">{file.name}</p>
            <p className="truncate text-xs text-muted-foreground">
              {t.passages(chunkCount)}
              {when && <>, {t.deletedAt(when)}</>}
            </p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => setSheetOpen(true)}
          className="h-11 shrink-0 rounded-md px-3 text-sm font-medium text-primary hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring lg:hidden"
        >
          {t.viewerOpen}
        </button>
        <button
          type="button"
          onClick={() => replaceRef.current?.click()}
          className="hidden h-11 shrink-0 rounded-md px-3 text-sm font-medium text-primary hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring sm:block"
        >
          {t.newDocument}
        </button>
        <input
          ref={replaceRef}
          type="file"
          accept=".pdf,application/pdf"
          className="sr-only"
          aria-label={t.uploadReplaceLabel}
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) handleUpload(f);
            e.target.value = "";
          }}
        />
        {toggles}
      </header>

      <div className="grid min-h-0 flex-1 lg:grid-cols-[minmax(0,1fr)_minmax(400px,42%)]">
        {/* Document: beside the answers on large screens, a sheet on small ones. */}
        <div
          className={
            sheetOpen
              ? "fixed inset-0 z-40 flex flex-col bg-background lg:static lg:z-auto"
              : "hidden min-h-0 lg:block"
          }
          role={sheetOpen ? "dialog" : undefined}
          aria-modal={sheetOpen || undefined}
          aria-label={t.viewerLabel}
        >
          {sheetOpen && (
            <div className="flex items-center border-b px-2 py-1 lg:hidden">
              <button
                type="button"
                onClick={() => setSheetOpen(false)}
                className="inline-flex h-11 items-center gap-2 rounded-md px-3 text-sm font-medium hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
              >
                <ArrowLeft className="size-4" aria-hidden="true" />
                {t.viewerClose}
              </button>
            </div>
          )}
          <div className="min-h-0 flex-1 lg:h-full">
            <DocumentViewer file={file} active={active} />
          </div>
        </div>

        <div className="min-h-0 border-l">
          <AssistantPanel
            history={history}
            pending={
              queryState.status === "searching"
                ? {
                    question: queryState.question,
                    stage: queryState.stage,
                    passages: queryState.passages,
                  }
                : undefined
            }
            question={question}
            onQuestionChange={setQuestion}
            onSubmit={handleQuery}
            error={
              queryState.status === "error"
                ? errorMessage(t, queryState.code, queryState.message)
                : undefined
            }
            active={active}
            onShowSource={showSource}
          />
        </div>
      </div>
    </div>
  );
}
