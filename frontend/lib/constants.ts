export const API_PREFIX = "/api/v1";

export const STATUS_OPTIONS = [
  "submitted",
  "under_review",
  "accepted",
  "rejected",
  "needs_more_info",
  "applied",
  "archived",
  "spam",
] as const;

export const CONFIDENCE_OPTIONS = ["low", "medium", "high"] as const;

export const errorMessages: Record<string, string> = {
  LANGUAGE_NOT_GERMAN: "Bitte reiche Hinweise auf Deutsch ein.",
  WORD_FILTER_BLOCKED:
    "Dein Hinweis enthält Begriffe, die nicht öffentlich eingereicht werden können.",
  HTML_BLOCKED: "Bitte entferne HTML, Skripte oder eingebettete Inhalte.",
  URL_BLOCKED: "Bitte verwende nur sichere http/https Links.",
  RATE_LIMITED: "Zu viele Anfragen. Bitte versuche es später erneut.",
  FORBIDDEN: "Du hast dafür keine Berechtigung.",
  UNAUTHORIZED: "Bitte melde dich erneut an.",
};
