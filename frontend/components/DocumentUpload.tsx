"use client";

import { useCallback, useRef, useState } from "react";
import { FileText, Loader2, Upload } from "lucide-react";
import { cn } from "@/lib/utils";
import { useLocale } from "@/lib/i18n";

export type UploadStatus = "idle" | "uploading" | "success" | "error";

interface Props {
  status: UploadStatus;
  chunkCount?: number;
  filename?: string;
  expiresAt?: string;
  error?: string;
  onUpload: (file: File) => void;
}

export default function DocumentUpload({
  status,
  chunkCount,
  filename,
  expiresAt,
  error,
  onUpload,
}: Props) {
  const { t, locale } = useLocale();
  const [isDragOver, setIsDragOver] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleDrop = useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      setIsDragOver(false);
      const file = e.dataTransfer.files[0];
      if (file) onUpload(file);
    },
    [onUpload]
  );

  const handleInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (file) onUpload(file);
      // Reset input so the same file can be re-selected after an error.
      e.target.value = "";
    },
    [onUpload]
  );

  const input = (label: string) => (
    <input
      ref={inputRef}
      type="file"
      accept=".pdf,application/pdf"
      className="sr-only"
      onChange={handleInputChange}
      aria-label={label}
      disabled={status === "uploading"}
    />
  );

  // Success state — the loaded document, with a way to replace it.
  if (status === "success") {
    const when =
      expiresAt &&
      new Date(expiresAt).toLocaleString(locale, {
        weekday: "long",
        hour: "2-digit",
        minute: "2-digit",
      });
    return (
      <div className="flex items-center gap-3 rounded-lg border bg-card px-4 py-3">
        <FileText className="size-5 shrink-0 text-primary" strokeWidth={1.75} aria-hidden="true" />
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium">{filename}</p>
          <p className="text-xs text-muted-foreground">
            {chunkCount !== undefined && t.passages(chunkCount)}
            {when && <>, {t.deletedAt(when)}</>}
          </p>
        </div>
        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          className="h-11 shrink-0 rounded-md px-3 text-sm font-medium text-primary transition-colors hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
        >
          {t.uploadReplace}
        </button>
        {input(t.uploadReplaceLabel)}
      </div>
    );
  }

  return (
    <div>
      <div
        className={cn(
          "flex cursor-pointer select-none flex-col items-center justify-center gap-3 rounded-xl border border-dashed px-8 py-12 text-center transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring",
          isDragOver
            ? "border-highlight bg-highlight-soft/40"
            : "border-border bg-card hover:border-highlight",
          status === "uploading" && "pointer-events-none"
        )}
        onClick={() => status !== "uploading" && inputRef.current?.click()}
        onDrop={handleDrop}
        onDragOver={(e) => {
          e.preventDefault();
          setIsDragOver(true);
        }}
        onDragLeave={() => setIsDragOver(false)}
        role="button"
        tabIndex={0}
        aria-label={t.uploadZoneLabel}
        aria-busy={status === "uploading"}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            inputRef.current?.click();
          }
        }}
      >
        {input(t.uploadZoneLabel)}
        {status === "uploading" ? (
          <>
            <Loader2 className="size-7 animate-spin text-highlight" aria-hidden="true" />
            <p className="text-sm font-medium text-muted-foreground">{t.uploadProcessing}</p>
          </>
        ) : (
          <>
            <Upload className="size-7 text-muted-foreground" strokeWidth={1.5} aria-hidden="true" />
            <div>
              <p className="text-sm font-medium">
                {t.uploadPrompt}{" "}
                <span className="text-highlight-ink underline underline-offset-4">{t.uploadBrowse}</span>
              </p>
              <p className="mt-1 text-xs text-muted-foreground">{t.uploadHint}</p>
            </div>
          </>
        )}
      </div>

      {status === "error" && error && (
        <p role="alert" className="mt-2 text-sm text-destructive">
          {error}
        </p>
      )}
    </div>
  );
}
