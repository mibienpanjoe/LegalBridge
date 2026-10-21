"use client";

import { Scale } from "lucide-react";
import { useLocale } from "@/lib/i18n";

// BR-01 / INV-03: always visible, never dismissible, so no visit misses it.
export default function Disclaimer({ className = "" }: { className?: string }) {
  const { t } = useLocale();
  return (
    <p role="note" className={`flex items-start gap-2 text-sm text-muted-foreground ${className}`}>
      <Scale className="mt-0.5 size-4 shrink-0" strokeWidth={1.75} aria-hidden="true" />
      <span>
        <strong className="font-medium text-foreground">{t.disclaimerStrong}</strong>{" "}
        {t.disclaimerRest}
      </span>
    </p>
  );
}
