// BR-01: legal disclaimer — must be visible on every page load.
export default function Disclaimer() {
  return (
    <div
      role="note"
      aria-label="Legal disclaimer"
      className="flex items-start gap-3 rounded-lg border border-[#D97706] bg-[#FFFBEB] px-4 py-3"
    >
      <span className="mt-0.5 shrink-0 text-[#D97706]" aria-hidden="true">
        ⚠
      </span>
      <p className="text-sm text-[#92400E]">
        <strong className="font-semibold">
          LegalBridge provides legal information, not legal advice.
        </strong>{" "}
        Always consult a qualified lawyer for legal counsel.
      </p>
    </div>
  );
}
