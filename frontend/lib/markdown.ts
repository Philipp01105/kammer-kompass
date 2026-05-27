export function confidenceLabel(value: string) {
  if (value === "high") return "hoch";
  if (value === "medium") return "mittel";
  return "niedrig";
}

export function formatDate(value?: string | null) {
  if (!value) return "unbekannt";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "unbekannt";
  return new Intl.DateTimeFormat("de-DE", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}
