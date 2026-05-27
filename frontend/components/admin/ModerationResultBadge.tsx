import type { PreModerationStatus } from "@/types/api";

const labels: Record<PreModerationStatus, string> = {
  passed: "Automatisch bestanden",
  blocked_language: "Sprache blockiert",
  blocked_word_filter: "Wortfilter blockiert",
  blocked_html: "HTML blockiert",
  blocked_url: "URL blockiert",
  blocked_length: "Länge blockiert",
};

export function ModerationResultBadge({ status }: { status: PreModerationStatus }) {
  const ok = status === "passed";
  return (
    <span
      className={
        ok
          ? "inline-flex rounded-md bg-emerald-100 px-2 py-1 text-xs font-medium text-emerald-900"
          : "inline-flex rounded-md bg-destructive/10 px-2 py-1 text-xs font-medium text-destructive"
      }
    >
      {labels[status] ?? status}
    </span>
  );
}
