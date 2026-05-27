import type { ModerationStatus } from "@/types/api";

const labels: Record<ModerationStatus, string> = {
  submitted: "Eingereicht",
  under_review: "In Prüfung",
  needs_more_info: "Mehr Infos nötig",
  accepted: "Akzeptiert",
  rejected: "Abgelehnt",
  applied: "Übernommen",
  archived: "Archiviert",
  spam: "Spam",
};

export function StatusBadge({ status }: { status: ModerationStatus }) {
  return (
    <span className="inline-flex rounded-md bg-muted px-2 py-1 text-xs font-medium">
      {labels[status] ?? status}
    </span>
  );
}
