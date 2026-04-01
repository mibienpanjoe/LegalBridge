"use client";

import { useCallback, useRef, useState } from "react";
import { Upload, CheckCircle, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

export type UploadStatus = "idle" | "uploading" | "success" | "error";

interface Props {
  status: UploadStatus;
  chunkCount?: number;
  filename?: string;
  error?: string;
  onUpload: (file: File) => void;
}

export default function DocumentUpload({
  status,
  chunkCount,
  filename,
  error,
  onUpload,
}: Props) {
  const [isDragOver, setIsDragOver] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleFile = useCallback(
    (file: File) => {
      onUpload(file);
    },
    [onUpload]
  );

  const handleDrop = useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      setIsDragOver(false);
      const file = e.dataTransfer.files[0];
      if (file) handleFile(file);
    },
    [handleFile]
  );

  const handleDragOver = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragOver(true);
  }, []);

  const handleDragLeave = useCallback(() => {
    setIsDragOver(false);
  }, []);

  const handleInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (file) handleFile(file);
      // Reset input so the same file can be re-selected after an error.
      e.target.value = "";
    },
    [handleFile]
  );

  // Success state — show confirmation instead of the drop zone.
  if (status === "success") {
    return (
      <div className="flex items-center gap-3 rounded-lg border border-[#16A34A] bg-[#F0FDF4] px-5 py-4">
        <CheckCircle className="size-5 shrink-0 text-[#16A34A]" aria-hidden="true" />
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-[#15803D]">
            Document loaded
            {filename && (
              <span className="font-normal text-[#166534]"> — {filename}</span>
            )}
          </p>
          {chunkCount !== undefined && (
            <p className="text-xs text-[#4B5563] mt-0.5">
              {chunkCount} passage{chunkCount !== 1 ? "s" : ""} indexed
            </p>
          )}
        </div>
        <button
          onClick={() => inputRef.current?.click()}
          className="text-xs text-[#1B3A5C] underline underline-offset-2 hover:text-[#162F4A] transition-colors shrink-0"
        >
          Replace
        </button>
        <input
          ref={inputRef}
          type="file"
          accept=".pdf,application/pdf"
          className="sr-only"
          onChange={handleInputChange}
          aria-label="Replace PDF document"
        />
      </div>
    );
  }

  const zoneStyle = cn(
    "relative flex flex-col items-center justify-center gap-3 rounded-xl border-2 border-dashed px-8 py-10 text-center transition-colors duration-200 cursor-pointer select-none",
    isDragOver && "border-[#C9873A] bg-[rgba(201,135,58,0.05)]",
    !isDragOver && "border-[#D1D5DB] bg-[#F7F8FA] hover:border-[#C9873A] hover:bg-[rgba(201,135,58,0.03)]",
    status === "uploading" && "pointer-events-none opacity-75"
  );

  return (
    <div>
      <div
        className={zoneStyle}
        onClick={() => status !== "uploading" && inputRef.current?.click()}
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        role="button"
        tabIndex={0}
        aria-label="Upload PDF document"
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            inputRef.current?.click();
          }
        }}
      >
        <input
          ref={inputRef}
          type="file"
          accept=".pdf,application/pdf"
          className="sr-only"
          onChange={handleInputChange}
          aria-label="Select PDF document"
          disabled={status === "uploading"}
        />

        {status === "uploading" ? (
          <>
            <Loader2
              className="size-8 text-[#C9873A] animate-spin"
              aria-hidden="true"
            />
            <p className="text-sm font-medium text-[#4B5563]">
              Processing document…
            </p>
          </>
        ) : (
          <>
            <Upload
              className="size-8 text-[#9CA3AF]"
              aria-hidden="true"
            />
            <div>
              <p className="text-sm font-medium text-[#111827]">
                Drop a PDF here, or{" "}
                <span className="text-[#C9873A] underline underline-offset-2">
                  browse
                </span>
              </p>
              <p className="mt-1 text-xs text-[#9CA3AF]">PDF only · max 20 MB</p>
            </div>
          </>
        )}
      </div>

      {status === "error" && error && (
        <p role="alert" className="mt-2 flex items-center gap-1.5 text-sm text-[#DC2626]">
          <span aria-hidden="true">✕</span>
          {error}
        </p>
      )}
    </div>
  );
}
