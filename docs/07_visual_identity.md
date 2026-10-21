# LegalBridge — Visual Identity Guide
Version: v2.0, 2026-10-21 (v1.0, 2026-03-31)

## Brand Essence

**Name:** LegalBridge — "Bridge" between complex legal text and plain-language understanding. The name signals access: closing the gap between the law as written and the business owner who needs to understand it.

**Tagline:** *Your legal documents, explained.*

**Personality traits:** Trustworthy · Precise · Approachable · Grounded · Empowering

**Design principles:**
1. **Trust through transparency** — Every answer is cited. The interface must look authoritative, not casual. Users should feel they are consulting a well-organized legal library, not a chatbot.
2. **Clarity for non-lawyers** — SME users are not legal professionals. UI labels, button text, and error messages must be in plain language. Avoid legal jargon in the interface itself.
3. **Multilingual readiness** — Layouts must accommodate French and English without breaking. French text is typically 15–20% longer than its English equivalent — buttons and labels must not clip.
4. **Professional with warmth** — Legal tools often feel cold and sterile. LegalBridge serves West African businesses; the palette draws on regional identity to feel grounded rather than generically corporate.
5. **Accessibility over decoration** — No animations that block interaction. No color-as-sole-state indicator. No touch targets under 44px.

---

## Signature: the amber highlighter

One element carries the identity: **cited law is marked with an amber highlighter**. Citation chips in an answer and the numbers on source passages use a soft amber band behind the text, like a marker pen on a printed statute (`highlighter` utility in `app/globals.css`). Everything around it stays quiet (navy ink on cool paper-white, thin rules, no drop shadows) so that the marks are what the eye follows from answer to source.

---

## Color System

Tokens are defined in OKLCH in `app/globals.css` and mapped onto the shadcn variable names (`--background`, `--primary`, …), so shadcn/Base UI components inherit the brand. Never hard-code hex values in components; use the Tailwind token classes (`bg-background`, `text-muted-foreground`, `bg-highlight-soft`, `text-highlight-ink`…).

### Light

| Token | Name | Hex | OKLCH | Use |
|-------|------|-----|-------|-----|
| `--background` | Paper | `#F6F7F9` | `0.976 0.003 264.5` | Page background |
| `--card` | Page | `#FFFFFF` | `1 0 0` | Cards, document, inputs |
| `--foreground` | Ink | `#14263D` | `0.266 0.050 255.2` | Body text |
| `--primary` | Navy | `#1B3A5C` | `0.343 0.070 252.4` | Wordmark, primary actions, focus ring |
| `--muted-foreground` | Slate | `#5B6675` | `0.506 0.027 256.2` | Secondary text (5.4:1 on paper) |
| `--border` | Rule | `#DDE1E7` | `0.908 0.009 258.3` | Card outlines, dividers |
| `--highlight` | Amber | `#C9873A` | `0.677 0.122 67.0` | Highlighter fills and drop-zone hover. **Never text** (3.0:1) |
| `--highlight-soft` | Amber 28% | — | amber / 0.28 | Highlighter band |
| `--highlight-ink` | Amber ink | `#8A5A1F` | `0.509 0.096 67.6` | Amber text: citation numbers, links in the upload zone (5.9:1) |
| `--destructive` | Error | `#B42318` | `0.500 0.182 29.5` | Error text |

### Dark

Dark mode follows the system preference, can be overridden with the header toggle (saved in `localStorage`), and is applied before first paint by an inline script, so there's no flash.

| Token | Hex | OKLCH |
|-------|-----|-------|
| `--background` (Night) | `#0E1726` | `0.204 0.033 260.3` |
| `--card` | `#152238` | `0.252 0.046 260.5` |
| `--foreground` | `#E6EAF0` | `0.936 0.009 258.3` |
| `--primary` (Navy, lifted) | `#8FB3DC` | `0.755 0.071 252.0` |
| `--muted-foreground` | `#9AA6B6` | `0.721 0.027 255.6` |
| `--border` | `#24324A` | `0.316 0.047 260.8` |
| `--highlight` / `--highlight-ink` | `#E0A458` | `0.761 0.117 70.9` |

**Rationale:** Navy carries the authority of formal documents; amber draws on West African gold and textile heritage and works as a highlighter. The paper is cool, not cream, to stay clear of the generic warm-cream "AI" look.

---

## Typography

| Role | Typeface | Where |
|------|----------|-------|
| Interface | **Geist** (`--font-sans`) | Labels, buttons, inputs, metadata |
| Legal reading | **Source Serif 4** with optical sizes (`--font-serif`) | Answers, quoted passages, the wordmark |

Both load through `next/font/google` with `latin-ext` (for names and places with extended Latin characters). Answers and passages are set in a serif because they should read like a legal memo, not a chat log. Monospace is not used. JetBrains Mono, Inter and Playfair Display from v1.0 have been removed.

### Type scale

| Element | Font | Size / line height |
|---------|------|--------------------|
| Wordmark | Source Serif 4, 600 | 20px |
| Answer body | Source Serif 4, 400 | 17px / 1.7, max 68ch |
| Quoted passage | Source Serif 4, 400 | 15px / 1.65 |
| Question, body UI | Geist, 500 / 400 | 15px |
| Section heading, metadata | Geist, 500 / 400 | 14px / 12px |

**Rules:** sentence case everywhere; no all-caps tracked labels; headings are plain muted text, not eyebrows.

---

## Components

- **Citation chip**: the number only (`1`, not `[1]`), highlighter band, amber-ink text, 13px Geist semibold. Clicking scrolls to the source passage.
- **Source passage**: card with a caption (`1` highlighter chip + "Article 42–44, p. 14, code.pdf") and the verbatim passage in serif. Line breaks are preserved; passages longer than 280 characters collapse with "Show the full passage".
- **Upload zone**: dashed rule on the page colour; amber border on hover/drag. Once loaded, it becomes a one-line card with the file name, passage count, deletion time and "Replace".
- **Composer**: one bordered field holding the textarea and a 44px navy send button. Enter sends, Shift+Enter adds a line. Suggestions appear as pills only before the first question.
- **Progress**: the streamed pipeline stage replaces the generic loading text ("Searching the document…" → "Found 2 passages: Article 42–44, Article 45–47" → "Writing a cited answer…").
- **Legal disclaimer (BR-01)**: a quiet line with a scales icon, **always visible and not dismissible**, under the composer.

---

## Motion

Motion only answers user actions: the answer fades in (300ms) when it arrives, buttons and borders change colour on hover (150ms). No scroll-triggered or decorative animation. `prefers-reduced-motion` disables all of it, including smooth scrolling.

---

## Page Layout

Single column, max 768px, left-aligned:

```
┌ LegalBridge                                 EN  ☾ ┐
│ Vos documents juridiques, expliqués.              │
├───────────────────────────────────────────────────┤
│ Votre document                                    │
│ [ code-travail.pdf · 98 passages …   Remplacer ]  │
│                                                   │
│ Question                                          │
│ Answer in serif with highlighter chips  1   2     │
│ Sources                                           │
│ [ 1  Article 42–44, p. 14 — passage… ]            │
│                                                   │
│ Interroger ce document                            │
│ [ Que voulez-vous savoir…               (↑) ]     │
│ ⚖ LegalBridge fournit une information juridique…  │
└───────────────────────────────────────────────────┘
```

---

## Accessibility Checklist

- Body text ≥ 4.5:1 in both themes (ink on paper 14.3:1, slate on paper 5.4:1, amber ink on page 5.9:1; dark: text 14.9:1, slate 7.3:1, amber 7.3:1)
- Amber `#C9873A` is never used for text; amber text uses `--highlight-ink`
- Visible focus outline (2px, `--ring`) on every interactive element
- Touch targets ≥ 44×44px (toggles, send, Replace)
- State is never shown by colour alone (errors are text in `role="alert"`)
- Streaming progress is announced through an `aria-live="polite"` region
- `lang` on `<html>` follows the selected language; the language switch is labelled in the target language

---

## Language & Tone

- **French first**: French is the default; the browser language and a saved choice can switch to English. All interface strings live in `lib/i18n.tsx`, and API error codes map to localized messages there.
- Plain verbs and sentence case: "Demander", "Remplacer", "Ask", "Replace".
- Errors say what happened and what to do next: "Ce fichier n’est pas un PDF. Choisissez un document PDF."
- Empty and no-result states give direction, not mood: "Ce document n’aborde pas cette question" followed by what to try.
- Disclaimer: "LegalBridge fournit une information juridique, pas un conseil juridique. Pour un avis sur votre situation, consultez un juriste ou un avocat." / "LegalBridge gives legal information, not legal advice. For advice on your situation, consult a qualified lawyer."
