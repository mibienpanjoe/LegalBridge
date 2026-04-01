"use client";

import { useEffect, useState } from "react";

const STORAGE_KEY = "lb_disclaimer_dismissed";

// BR-01: legal disclaimer — must be visible on every page load unless dismissed.
export default function Disclaimer() {
  const [dismissed, setDismissed] = useState(true); // hidden until hydrated

  useEffect(() => {
    setDismissed(localStorage.getItem(STORAGE_KEY) === "1");
  }, []);

  if (dismissed) return null;

  return (
    <div
      role="note"
      aria-label="Legal disclaimer"
      className="flex items-start gap-3 rounded-lg border border-[#D97706] bg-[#FFFBEB] px-4 py-3"
    >
      <span className="mt-0.5 shrink-0 text-[#D97706]" aria-hidden="true">
        ⚠
      </span>
      <p className="flex-1 text-sm text-[#92400E]">
        <strong className="font-semibold">
          LegalBridge provides legal information, not legal advice.
        </strong>{" "}
        Always consult a qualified lawyer for legal counsel.
      </p>
      <button
        onClick={() => {
          localStorage.setItem(STORAGE_KEY, "1");
          setDismissed(true);
        }}
        aria-label="Dismiss disclaimer"
        className="shrink-0 text-[#D97706] hover:text-[#92400E] transition-colors text-lg leading-none"
      >
        ×
      </button>
    </div>
  );
}
