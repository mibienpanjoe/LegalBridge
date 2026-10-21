"use client";

import DocumentUpload, { type UploadStatus } from "@/components/DocumentUpload";
import Disclaimer from "@/components/Disclaimer";
import { useLocale } from "@/lib/i18n";

interface Props {
  status: UploadStatus;
  error?: string;
  onUpload: (file: File) => void;
}

/** The first screen: what LegalBridge does, shown on a real article, and the upload. */
export default function Landing({ status, error, onUpload }: Props) {
  const { t } = useLocale();

  return (
    <main className="mx-auto grid w-full max-w-6xl flex-1 items-center gap-12 px-4 py-10 md:px-8 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)] lg:py-16">
      <section>
        <h2 className="max-w-[18ch] font-serif text-4xl font-semibold leading-[1.1] tracking-tight text-primary md:text-5xl">
          {t.heroTitle}
        </h2>
        <p className="mt-5 max-w-[52ch] text-[17px] leading-relaxed text-muted-foreground">
          {t.heroBody}
        </p>
        <div className="mt-8">
          <DocumentUpload status={status} error={error} onUpload={onUpload} />
        </div>
        <Disclaimer className="mt-6" />
      </section>

      <Specimen />
    </main>
  );
}

// A worked example on the real text of the Burkina Faso Labour Code
// (Loi n° 028-2008/AN, Articles 1–2), showing the answer → highlighted article link.
function Specimen() {
  const { t } = useLocale();
  return (
    <figure aria-label={t.exampleLabel} className="relative">
      <div className="rounded-xl border bg-card p-6 md:p-8">
        <p className="text-[15px] font-medium">{t.exampleQuestion}</p>
        <p className="mt-3 font-serif text-[17px] leading-[1.7]">
          {t.exampleAnswer}
          <span className="highlighter mx-1 rounded-sm px-1 align-[1px] font-sans text-[13px] font-semibold text-highlight-ink">
            1
          </span>
          .
        </p>

        <div
          lang="fr"
          className="mt-6 rounded-lg border bg-background p-5 font-serif text-[15px] leading-[1.65]"
        >
          <p className="mb-3 font-sans text-xs text-muted-foreground">
            Code du travail, Titre I, p. 3
          </p>
          <p>
            <span className="highlighter">
              Article 1 : La présente loi est applicable aux travailleurs et aux employeurs
              exerçant leur activité professionnelle au Burkina Faso.
            </span>
          </p>
          <p className="mt-3 text-muted-foreground">
            Article 2 : Est considérée comme travailleur, au sens de la présente loi, toute
            personne qui s’est engagée à mettre son activité professionnelle moyennant
            rémunération…
          </p>
        </div>
      </div>
      <figcaption className="mt-3 text-sm text-muted-foreground">{t.exampleLabel}</figcaption>
    </figure>
  );
}
