"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";

export type Locale = "fr" | "en";

const STORAGE_KEY = "lb_locale";

const en = {
  tagline: "Your legal documents, explained.",
  footer: "Built for West African businesses, on open-source models.",

  disclaimerStrong: "LegalBridge gives legal information, not legal advice.",
  disclaimerRest: "For advice on your situation, consult a qualified lawyer.",
  disclaimerDismiss: "Hide this notice",

  uploadHeading: "Your document",
  uploadPrompt: "Drop a legal PDF here, or",
  uploadBrowse: "choose a file",
  uploadHint: "PDF up to 20 MB, in French or English",
  uploadZoneLabel: "Upload a PDF document",
  uploadProcessing: "Reading and indexing your document…",
  uploadReady: "Ready",
  uploadReplace: "Replace",
  uploadReplaceLabel: "Replace the PDF document",
  passages: (n: number) => `${n} passage${n === 1 ? "" : "s"} indexed`,
  deletedAt: (when: string) => `deleted automatically ${when}`,

  askHeading: "Ask about this document",
  askLabel: "Your question",
  askPlaceholder: "What would you like to know about this document?",
  askSubmit: "Ask",
  suggestions: [
    "What are the obligations of each party?",
    "What conditions must be met?",
    "What penalties does it set out?",
  ],

  you: "You",
  sources: "Sources",
  jumpToSource: (n: number) => `Go to source ${n}`,
  showMore: "Show the full passage",
  showLess: "Show less",
  noResultsTitle: "This document doesn't cover that",
  noResultsBody:
    "No passage is close enough to your question to answer it reliably. Try other words, or ask about something the document deals with.",
  stageSearching: "Searching the document…",
  stageSources: (n: number, where: string) =>
    `Found ${n} passage${n === 1 ? "" : "s"}${where ? ` in ${where}` : ""}`,
  stageGenerating: "Writing a cited answer…",

  newDocument: "New document",
  heroTitle: "Ask your legal document a question.",
  heroBody:
    "Upload a law, a code or a contract. Each answer points to the article it comes from, highlighted in the document itself.",
  exampleLabel: "Example from the Burkina Faso Labour Code",
  exampleQuestion: "Who does the labour code apply to?",
  exampleAnswer: "To workers and employers carrying out their professional activity in Burkina Faso",
  emptyThreadTitle: "Ask your first question",
  emptyThreadBody: "Answers cite the passages they rely on. Select a citation to see it in the document.",
  showInDocument: "Show in document",
  viewerLabel: "PDF document",
  viewerLoading: "Opening the document…",
  viewerError: "This PDF can't be displayed here, but you can still ask questions about it.",
  viewerPage: (n: number, total: number) => `Page ${n} of ${total}`,
  viewerClose: "Back to answers",
  viewerOpen: "View document",

  themeToLight: "Switch to light theme",
  themeToDark: "Switch to dark theme",
  localeSwitch: "Lire en français",

  errors: {
    DOCUMENT_NOT_FOUND:
      "This document was deleted after 24 hours. Upload it again to keep asking questions.",
    INVALID_FILE_TYPE: "This file is not a PDF. Choose a PDF document.",
    FILE_TOO_LARGE: "This PDF is larger than 20 MB. Choose a smaller file.",
    EXTRACTION_FAILED:
      "No text could be read from this PDF. Scanned documents need text recognition (OCR) first.",
    LLM_UNAVAILABLE: "The answer service is not responding. Try again in a moment.",
    EMBEDDING_UNAVAILABLE: "The search service is not responding. Try again in a moment.",
    DATABASE_UNAVAILABLE: "The document store is not responding. Try again in a moment.",
  } as Record<string, string>,
};

type Dictionary = typeof en;

const fr: Dictionary = {
  tagline: "Vos documents juridiques, expliqués.",
  footer: "Conçu pour les entreprises d’Afrique de l’Ouest, avec des modèles open source.",

  disclaimerStrong: "LegalBridge fournit une information juridique, pas un conseil juridique.",
  disclaimerRest: "Pour un avis sur votre situation, consultez un juriste ou un avocat.",
  disclaimerDismiss: "Masquer cet avertissement",

  uploadHeading: "Votre document",
  uploadPrompt: "Déposez un PDF juridique ici, ou",
  uploadBrowse: "choisissez un fichier",
  uploadHint: "PDF de 20 Mo maximum, en français ou en anglais",
  uploadZoneLabel: "Téléverser un document PDF",
  uploadProcessing: "Lecture et indexation du document…",
  uploadReady: "Prêt",
  uploadReplace: "Remplacer",
  uploadReplaceLabel: "Remplacer le document PDF",
  passages: (n: number) => `${n} passage${n === 1 ? "" : "s"} indexé${n === 1 ? "" : "s"}`,
  deletedAt: (when: string) => `supprimé automatiquement ${when}`,

  askHeading: "Interroger ce document",
  askLabel: "Votre question",
  askPlaceholder: "Que voulez-vous savoir sur ce document ?",
  askSubmit: "Demander",
  suggestions: [
    "Quelles sont les obligations de chaque partie ?",
    "Quelles conditions faut-il remplir ?",
    "Quelles sanctions sont prévues ?",
  ],

  you: "Vous",
  sources: "Sources",
  jumpToSource: (n: number) => `Aller à la source ${n}`,
  showMore: "Afficher tout le passage",
  showLess: "Réduire",
  noResultsTitle: "Ce document n’aborde pas cette question",
  noResultsBody:
    "Aucun passage n’est assez proche de votre question pour y répondre de façon fiable. Reformulez, ou posez une question sur un sujet traité par le document.",
  stageSearching: "Recherche dans le document…",
  stageSources: (n: number, where: string) =>
    `${n} passage${n === 1 ? "" : "s"} trouvé${n === 1 ? "" : "s"}${where ? ` : ${where}` : ""}`,
  stageGenerating: "Rédaction d’une réponse sourcée…",

  newDocument: "Nouveau document",
  heroTitle: "Posez vos questions à votre document juridique.",
  heroBody:
    "Téléversez une loi, un code ou un contrat. Chaque réponse renvoie à l’article dont elle est tirée, surligné dans le document lui-même.",
  exampleLabel: "Exemple tiré du Code du travail du Burkina Faso",
  exampleQuestion: "À qui s’applique le code du travail ?",
  exampleAnswer: "Aux travailleurs et aux employeurs exerçant leur activité professionnelle au Burkina Faso",
  emptyThreadTitle: "Posez votre première question",
  emptyThreadBody:
    "Les réponses citent les passages sur lesquels elles s’appuient. Sélectionnez une citation pour la voir dans le document.",
  showInDocument: "Voir dans le document",
  viewerLabel: "Document PDF",
  viewerLoading: "Ouverture du document…",
  viewerError: "Ce PDF ne peut pas être affiché ici, mais vous pouvez toujours lui poser des questions.",
  viewerPage: (n: number, total: number) => `Page ${n} sur ${total}`,
  viewerClose: "Retour aux réponses",
  viewerOpen: "Voir le document",

  themeToLight: "Passer au thème clair",
  themeToDark: "Passer au thème sombre",
  localeSwitch: "Read in English",

  errors: {
    DOCUMENT_NOT_FOUND:
      "Ce document a été supprimé après 24 heures. Téléversez-le à nouveau pour continuer.",
    INVALID_FILE_TYPE: "Ce fichier n’est pas un PDF. Choisissez un document PDF.",
    FILE_TOO_LARGE: "Ce PDF dépasse 20 Mo. Choisissez un fichier plus léger.",
    EXTRACTION_FAILED:
      "Aucun texte n’a pu être lu dans ce PDF. Un document scanné doit d’abord passer par une reconnaissance de texte (OCR).",
    LLM_UNAVAILABLE: "Le service de réponse ne répond pas. Réessayez dans un instant.",
    EMBEDDING_UNAVAILABLE: "Le service de recherche ne répond pas. Réessayez dans un instant.",
    DATABASE_UNAVAILABLE: "La base de documents ne répond pas. Réessayez dans un instant.",
  },
};

const dictionaries: Record<Locale, Dictionary> = { en, fr };

type LocaleContextValue = {
  locale: Locale;
  t: Dictionary;
  setLocale: (l: Locale) => void;
};

const LocaleContext = createContext<LocaleContextValue | null>(null);

function initialLocale(): Locale {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === "fr" || saved === "en") return saved;
  } catch {
    // storage unavailable (private mode): fall through to the browser language
  }
  return navigator.language.toLowerCase().startsWith("en") ? "en" : "fr";
}

export function LocaleProvider({ children }: { children: React.ReactNode }) {
  // French is the default for West African users; the browser language and a
  // saved choice are applied after hydration.
  const [locale, setLocaleState] = useState<Locale>("fr");

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- reads browser-only state after hydration
    setLocaleState(initialLocale());
  }, []);

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  const setLocale = useCallback((l: Locale) => {
    setLocaleState(l);
    try {
      localStorage.setItem(STORAGE_KEY, l);
    } catch {
      // not persisted; the choice still applies for this visit
    }
  }, []);

  const value = useMemo(
    () => ({ locale, t: dictionaries[locale], setLocale }),
    [locale, setLocale]
  );
  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>;
}

export function useLocale(): LocaleContextValue {
  const ctx = useContext(LocaleContext);
  if (!ctx) throw new Error("useLocale must be used inside <LocaleProvider>");
  return ctx;
}

/** The localized message for an API error code, or the server's message. */
export function errorMessage(t: Dictionary, code: string | undefined, fallback: string) {
  return (code && t.errors[code]) || fallback;
}
