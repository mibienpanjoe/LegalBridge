"use client";

import { useLocale } from "@/lib/i18n";

export default function LocaleToggle() {
  const { locale, setLocale, t } = useLocale();
  const next = locale === "fr" ? "en" : "fr";
  return (
    <button
      type="button"
      onClick={() => setLocale(next)}
      lang={next}
      aria-label={t.localeSwitch}
      title={t.localeSwitch}
      className="inline-flex h-11 min-w-11 items-center justify-center rounded-md px-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
    >
      {next.toUpperCase()}
    </button>
  );
}
