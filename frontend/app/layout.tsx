import type { Metadata } from "next";
import { Geist, Source_Serif_4 } from "next/font/google";
import { LocaleProvider } from "@/lib/i18n";
import { themeScript } from "@/components/ThemeToggle";
import "./globals.css";

// Geist for the interface; Source Serif 4 for answers and quoted law, which
// should read like a legal text rather than a chat log.
const geist = Geist({
  subsets: ["latin", "latin-ext"],
  variable: "--font-geist",
});

const sourceSerif = Source_Serif_4({
  subsets: ["latin", "latin-ext"],
  axes: ["opsz"],
  variable: "--font-source-serif",
});

export const metadata: Metadata = {
  title: "LegalBridge",
  description:
    "Ask questions about a legal PDF in French or English and get answers that cite the exact passage.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="fr"
      suppressHydrationWarning // the theme script adds .dark before hydration
      className={`${geist.variable} ${sourceSerif.variable} h-full`}
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className="flex min-h-full flex-col">
        <LocaleProvider>{children}</LocaleProvider>
      </body>
    </html>
  );
}
