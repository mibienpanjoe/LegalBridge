"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Document, Page, pdfjs } from "react-pdf";
import "react-pdf/dist/Page/TextLayer.css";
import type { Citation } from "@/lib/api";
import { useLocale } from "@/lib/i18n";

// Must be set in the module that renders <Document> (react-pdf README).
pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  "pdfjs-dist/build/pdf.worker.min.mjs",
  import.meta.url
).toString();

/** Letters and digits only, lowercased and accent-free, so a text-layer item
 *  can be matched against the stored passage whatever the spacing. */
function fold(s: string): string {
  return s
    .normalize("NFD")
    .replace(/\p{Mn}/gu, "")
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, "");
}

function escapeHtml(s: string): string {
  return s.replace(/[&<>"']/g, (c) => `&#${c.charCodeAt(0)};`);
}

// Text items shorter than this are too generic ("Article", "1.") to be
// highlighted on their own.
const MIN_MATCH_CHARS = 12;

interface Props {
  file: File;
  /** The passage to show: the viewer scrolls to its page and highlights it. */
  active?: Citation;
}

export default function DocumentViewer({ file, active }: Props) {
  const { t } = useLocale();
  const containerRef = useRef<HTMLDivElement>(null);
  const [numPages, setNumPages] = useState(0);
  const [width, setWidth] = useState(0);
  const [aspect, setAspect] = useState(1.414); // A4 until the first page is measured
  const [visible, setVisible] = useState<Set<number>>(() => new Set([1, 2]));
  const pageRefs = useRef(new Map<number, HTMLDivElement>());

  // Render pages at the container width.
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const ro = new ResizeObserver(([entry]) => {
      setWidth(Math.min(Math.floor(entry.contentRect.width) - 32, 900));
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  // Only rasterize pages near the viewport; a 100-page code would otherwise
  // render every canvas up front.
  useEffect(() => {
    const root = containerRef.current;
    if (!root || numPages === 0) return;
    const io = new IntersectionObserver(
      (entries) => {
        setVisible((prev) => {
          const next = new Set(prev);
          for (const e of entries) {
            const n = Number((e.target as HTMLElement).dataset.page);
            if (e.isIntersecting) next.add(n);
          }
          return next.size === prev.size ? prev : next;
        });
      },
      { root, rootMargin: "800px 0px" }
    );
    pageRefs.current.forEach((el) => io.observe(el));
    return () => io.disconnect();
  }, [numPages]);

  // Scroll to the active passage's first page.
  useEffect(() => {
    if (!active?.page_start) return;
    const page = active.page_start;
    // eslint-disable-next-line react-hooks/set-state-in-effect -- ensure the target page renders before scrolling
    setVisible((prev) => (prev.has(page) ? prev : new Set(prev).add(page)));
    pageRefs.current.get(page)?.scrollIntoView({ behavior: "smooth", block: "start" });
  }, [active]);

  const foldedPassage = useMemo(() => (active ? fold(active.passage) : ""), [active]);
  const pageStart = active?.page_start ?? 0;
  const pageEnd = active?.page_end || pageStart;

  const renderText = useCallback(
    (pageNumber: number) =>
      ({ str }: { str: string }) => {
        const safe = escapeHtml(str);
        if (!foldedPassage || pageNumber < pageStart || pageNumber > pageEnd) return safe;
        const f = fold(str);
        return f.length >= MIN_MATCH_CHARS && foldedPassage.includes(f)
          ? `<mark class="lb-pdf-highlight">${safe}</mark>`
          : safe;
      },
    [foldedPassage, pageStart, pageEnd]
  );

  return (
    <div
      ref={containerRef}
      className="h-full overflow-y-auto bg-muted/60 px-4 py-6"
      aria-label={t.viewerLabel}
      role="region"
    >
      <Document
        file={file}
        onLoadSuccess={async (pdf) => {
          setNumPages(pdf.numPages);
          const first = await pdf.getPage(1);
          const vp = first.getViewport({ scale: 1 });
          setAspect(vp.height / vp.width);
        }}
        loading={<p className="p-6 text-center text-sm text-muted-foreground">{t.viewerLoading}</p>}
        error={<p className="p-6 text-center text-sm text-destructive">{t.viewerError}</p>}
        className="mx-auto flex w-fit flex-col items-center gap-4"
      >
        {width > 0 &&
          Array.from({ length: numPages }, (_, i) => i + 1).map((n) => (
            <div
              key={n}
              data-page={n}
              ref={(el) => {
                if (el) pageRefs.current.set(n, el);
                else pageRefs.current.delete(n);
              }}
              className="scroll-mt-4 bg-white shadow-[0_0_0_1px_var(--border)]"
              style={{ width, minHeight: Math.round(width * aspect) }}
              aria-label={t.viewerPage(n, numPages)}
            >
              {visible.has(n) && (
                <Page
                  pageNumber={n}
                  width={width}
                  renderAnnotationLayer={false}
                  customTextRenderer={renderText(n)}
                />
              )}
            </div>
          ))}
      </Document>
    </div>
  );
}
