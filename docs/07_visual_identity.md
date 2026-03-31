# LegalBridge — Visual Identity Guide
Version: v1.0, 2026-03-31

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

## Color System

### Primary Palette

| Token | Name | Hex | Use |
|-------|------|-----|-----|
| `--color-primary` | Deep Navy | `#1B3A5C` | Primary actions, header background, active states |
| `--color-primary-hover` | Navy Hover | `#162F4A` | Hover state on primary elements |
| `--color-accent` | Warm Amber | `#C9873A` | Highlights, call-to-action accents, citation markers |
| `--color-accent-hover` | Amber Hover | `#B0732E` | Hover on accent elements |

**Rationale:** Deep Navy communicates legal authority and trust — the color of formal documents, court robes, and institutional credibility. Warm Amber draws from West African craft and textile heritage: gold, kente, and terracotta. Together they project trustworthy professionalism with cultural warmth.

### Surface Palette

| Token | Name | Hex | Use |
|-------|------|-----|-----|
| `--color-background` | Parchment | `#F7F8FA` | Page backgrounds |
| `--color-surface` | White | `#FFFFFF` | Cards, panels, input fields |
| `--color-surface-secondary` | Light Gray | `#F1F3F5` | Inactive tabs, secondary panels |
| `--color-border` | Slate Border | `#D1D5DB` | Input borders, card outlines |

### Semantic Colors

| Token | Hex | Use |
|-------|-----|-----|
| `--color-success` | `#16A34A` | Successful ingestion, upload confirmation |
| `--color-success-bg` | `#F0FDF4` | Success state backgrounds |
| `--color-warning` | `#D97706` | Partial results, degraded service warnings |
| `--color-warning-bg` | `#FFFBEB` | Warning state backgrounds |
| `--color-error` | `#DC2626` | Upload failures, API errors |
| `--color-error-bg` | `#FEF2F2` | Error state backgrounds |
| `--color-info` | `#2563EB` | Informational tooltips, inline help |

### Neutral Scale

| Token | Hex | Use |
|-------|-----|-----|
| `--color-text-primary` | `#111827` | Body text, headings |
| `--color-text-secondary` | `#4B5563` | Supporting text, labels |
| `--color-text-muted` | `#9CA3AF` | Placeholders, disabled states |
| `--color-text-inverse` | `#FFFFFF` | Text on dark backgrounds |

### CSS Custom Properties

```css
:root {
  --color-primary:           #1B3A5C;
  --color-primary-hover:     #162F4A;
  --color-accent:            #C9873A;
  --color-accent-hover:      #B0732E;
  --color-background:        #F7F8FA;
  --color-surface:           #FFFFFF;
  --color-surface-secondary: #F1F3F5;
  --color-border:            #D1D5DB;
  --color-success:           #16A34A;
  --color-success-bg:        #F0FDF4;
  --color-warning:           #D97706;
  --color-warning-bg:        #FFFBEB;
  --color-error:             #DC2626;
  --color-error-bg:          #FEF2F2;
  --color-info:              #2563EB;
  --color-text-primary:      #111827;
  --color-text-secondary:    #4B5563;
  --color-text-muted:        #9CA3AF;
  --color-text-inverse:      #FFFFFF;
}
```

---

## Typography

| Role | Font | Weights | Use |
|------|------|---------|-----|
| Display | Playfair Display | 600, 700 | Hero text, product name, page titles |
| Body | Inter | 400, 500, 600 | All body copy, labels, UI text |
| Monospace | JetBrains Mono | 400 | Citation passage text, document excerpts |

**Rationale:** Playfair Display carries the gravitas of legal publishing — serif typefaces signal seriousness, authority, and durability. Inter is maximally legible in UI contexts at any screen size. JetBrains Mono distinguishes citation text from synthesized content, reinforcing that citations are source material, not AI-generated.

**Import (Google Fonts):**
```html
<link href="https://fonts.googleapis.com/css2?family=Playfair+Display:wght@600;700&family=Inter:wght@400;500;600&family=JetBrains+Mono&display=swap" rel="stylesheet">
```

### Type Scale

| Name | Size | Line Height | Letter Spacing | Use |
|------|------|-------------|----------------|-----|
| `text-xs` | 12px | 1.4 | 0.02em | Labels, badges, captions |
| `text-sm` | 14px | 1.5 | 0 | Secondary UI text, helper text |
| `text-base` | 16px | 1.6 | 0 | Body copy, form labels |
| `text-lg` | 18px | 1.5 | 0 | Section subheadings |
| `text-xl` | 20px | 1.4 | -0.01em | Page section headers |
| `text-2xl` | 24px | 1.3 | -0.01em | Card titles, panel headers |
| `text-4xl` | 36px | 1.2 | -0.02em | Page title (Display font) |
| `text-5xl` | 48px | 1.1 | -0.02em | Hero headline (Display font) |

---

## Spacing & Layout

**Base unit:** 4px
**Spacing scale:** 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 80, 96px

**Responsive breakpoints:**
| Name | Min width | Use |
|------|-----------|-----|
| `sm` | 640px | Mobile landscape |
| `md` | 768px | Tablet |
| `lg` | 1024px | Desktop |
| `xl` | 1280px | Wide desktop |

**Max content width:** 1024px, centered
**Page horizontal padding:** 16px (mobile), 24px (tablet), 48px (desktop)

---

## Component Styling

### Buttons

**Primary:**
```
Background: #1B3A5C
Text: #FFFFFF, 14px Inter 500
Border radius: 8px
Padding: 10px 20px
Hover: background #162F4A, translateY(-1px), box-shadow 0 4px 12px rgba(27, 58, 92, 0.3)
Active: background #112540, translateY(0)
Focus: box-shadow 0 0 0 3px rgba(27, 58, 92, 0.25)
Disabled: background #9CA3AF, cursor not-allowed
```

**Accent (CTA):**
```
Background: #C9873A
Text: #FFFFFF, 14px Inter 500
Border radius: 8px
Padding: 10px 20px
Hover: background #B0732E, translateY(-1px)
```

**Secondary (Outline):**
```
Background: transparent
Border: 1.5px solid #1B3A5C
Text: #1B3A5C, 14px Inter 500
Border radius: 8px
Padding: 10px 20px
Hover: background rgba(27, 58, 92, 0.06)
```

**Ghost:**
```
Background: transparent
Text: #4B5563, 14px Inter 500
No border
Hover: background #F1F3F5
```

**Destructive:**
```
Background: #DC2626
Text: #FFFFFF
Hover: background #B91C1C
```

---

### Cards

```
Background: #FFFFFF
Border: 1px solid #D1D5DB
Border radius: 12px
Padding: 24px
Box shadow: 0 1px 3px rgba(0, 0, 0, 0.08)
Hover (interactive cards): box-shadow 0 4px 12px rgba(0, 0, 0, 0.12), translateY(-1px)
```

---

### Citation Block

The citation block is the most distinctive component — it must visually communicate "this is source material, not AI synthesis."

```
Background: #F7F8FA
Border-left: 4px solid #C9873A (Warm Amber — signals provenance)
Border radius: 0 8px 8px 0
Padding: 16px
Margin-top: 12px

Citation number badge:
  Background: #C9873A
  Text: #FFFFFF, 11px Inter 600
  Border radius: 4px
  Padding: 2px 7px
  Margin-bottom: 8px

Passage text:
  Font: JetBrains Mono 13px
  Color: #374151
  Line height: 1.6

Document label:
  Font: Inter 11px, 500
  Color: #9CA3AF
  Margin-top: 8px
```

---

### Document Upload Zone

```
Border: 2px dashed #D1D5DB
Border radius: 12px
Padding: 40px
Background: #F7F8FA
Text-align: center

Active (drag-over):
  Border-color: #C9873A
  Background: rgba(201, 135, 58, 0.05)

Accepted:
  Border-color: #16A34A
  Background: #F0FDF4

Rejected:
  Border-color: #DC2626
  Background: #FEF2F2
```

---

### Query Input

```
Input field:
  Border: 1.5px solid #D1D5DB
  Border radius: 8px
  Padding: 12px 16px
  Font: Inter 15px 400
  Background: #FFFFFF
  Color: #111827

  Focus: border-color #1B3A5C, box-shadow 0 0 0 3px rgba(27, 58, 92, 0.15)
  Placeholder: color #9CA3AF

Submit button:
  Attached to right edge of input (or below on mobile)
  Uses Primary button style
  Icon: arrow-right or send glyph
```

---

### Answer Display

```
Answer text container:
  Font: Inter 15px 400
  Color: #111827
  Line height: 1.7
  Max width: 720px

Citation inline marker (e.g., [1]):
  Color: #C9873A
  Font-weight: 600
  Cursor: pointer (scrolls to citation block below)

Citations section heading:
  Font: Inter 13px 600
  Color: #4B5563
  Text-transform: uppercase
  Letter-spacing: 0.06em
  Margin-top: 24px
  Margin-bottom: 12px
```

---

### Legal Disclaimer Banner

```
Background: #FFFBEB
Border: 1px solid #D97706
Border radius: 8px
Padding: 12px 16px
Font: Inter 13px 400
Color: #92400E
Icon: ⚠ (warning) — color #D97706

Text: "LegalBridge provides legal information, not legal advice.
      Always consult a qualified lawyer for legal counsel."
```

---

### Loading States

**Query in progress:**
```
Answer area: animated pulse placeholder (3 lines, gray skeleton)
Submit button: disabled, "Searching..." label with spinner icon
Duration token: --duration-slow (300ms pulse cycle)
```

**Ingestion in progress:**
```
Upload zone: progress indicator
"Processing document..." text below upload zone
Duration: until ingest API returns
```

---

## Motion & Animation

| Token | Value | Use |
|-------|-------|-----|
| `--duration-fast` | 100ms | Hover state color transitions |
| `--duration-base` | 200ms | Button clicks, focus rings, small transitions |
| `--duration-slow` | 300ms | Card entrance, answer reveal, loading pulse |
| `--easing-out` | `cubic-bezier(0, 0, 0.2, 1)` | Elements entering the screen |
| `--easing-in-out` | `cubic-bezier(0.4, 0, 0.2, 1)` | Elements moving within the screen |

**Answer reveal:** When the query response is received, the answer container fades in over 300ms with a slight `translateY(4px → 0)`. Citations appear 100ms after the answer.

**Accessibility:** All transitions MUST be gated on `prefers-reduced-motion`:
```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

---

## Page-Level Patterns

### Main Application Page (Single Page)

```
┌─────────────────────────────────────────────────────┐
│  HEADER: LegalBridge logo + tagline                 │
│  Background: #1B3A5C (Deep Navy)                    │
│  Text: #FFFFFF                                      │
├─────────────────────────────────────────────────────┤
│  DISCLAIMER BANNER (always visible)                 │
│  "LegalBridge provides information, not legal       │
│   advice. Consult a qualified lawyer."              │
├─────────────────────────────────────────────────────┤
│  UPLOAD ZONE (or "Document loaded" confirmation)    │
│  Drag-and-drop PDF upload                           │
│  OR: "Demo document loaded: Ghana Companies Act"    │
├─────────────────────────────────────────────────────┤
│  QUERY SECTION                                      │
│  ┌─────────────────────────────────────────────┐   │
│  │  What would you like to know?               │   │
│  │  [text input                    ] [Ask →]   │   │
│  └─────────────────────────────────────────────┘   │
│  Suggested questions (pills, clickable)             │
├─────────────────────────────────────────────────────┤
│  ANSWER SECTION (empty until first query)           │
│  Answer text...                                     │
│  ──────────────────────────────────────────────     │
│  SOURCES                                            │
│  ┌──────────────────────────────────────────────┐  │
│  │ [1] ghana_companies_act.pdf                  │  │
│  │     "Verbatim passage text from document..." │  │
│  └──────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────┤
│  FOOTER: "Powered by open-source AI · Built for West Africa"│
└─────────────────────────────────────────────────────┘
```

---

## Accessibility Checklist

- [ ] All text meets WCAG 2.1 AA contrast ratios: ≥ 4.5:1 for body text, ≥ 3:1 for large text (≥18px bold or ≥24px regular)
- [ ] Deep Navy `#1B3A5C` on White `#FFFFFF`: contrast ratio 9.5:1 ✓
- [ ] Warm Amber `#C9873A` on White `#FFFFFF`: contrast ratio 3.1:1 — use only for large text or decorative elements, not body copy
- [ ] All interactive elements have visible focus indicators (3px ring)
- [ ] All touch targets are ≥ 44×44px (mobile)
- [ ] Color is never the sole indicator of state (error states use icon + border + text, not color alone)
- [ ] `prefers-reduced-motion` is respected for all animations
- [ ] All images and icons have `aria-label` or `alt` text
- [ ] Semantic HTML: `<button>`, `<nav>`, `<main>`, `<section>`, `<header>`, `<footer>`
- [ ] Loading states communicate via `aria-live="polite"` regions
- [ ] Form inputs have associated `<label>` elements

---

## Language & Tone Guidelines

**For UI labels and buttons:**
- Use direct verbs: "Ask", "Upload", "View source" — not "Submit Query" or "Initiate Document Ingestion"
- Keep button labels under 20 characters
- In French: "Poser une question", "Télécharger", "Voir la source"

**For answer synthesis (Claude prompt instructions):**
- Plain language over legal jargon
- Active voice where possible
- Precise: quote the specific section number when available

**For error messages:**
- Describe what happened: "This file is not a PDF."
- Suggest a next action: "Please upload a PDF document."
- Never blame the user: not "Invalid file type" alone — add context

**For the disclaimer:**
- English: "LegalBridge provides legal information, not legal advice. Consult a qualified lawyer for legal counsel."
- French: "LegalBridge fournit des informations juridiques, et non des conseils juridiques. Consultez un avocat qualifié pour obtenir des conseils."

**Brand voice adjectives:** Authoritative · Clear · Accessible · Grounded
**Avoid:** Casual slang, excessive exclamation, vague legal terms used as decoration, English-only assumptions
