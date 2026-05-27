export type ConfidenceLevel = "low" | "medium" | "high";

export type ModerationStatus =
  | "submitted"
  | "under_review"
  | "needs_more_info"
  | "accepted"
  | "rejected"
  | "applied"
  | "archived"
  | "spam";

export type PreModerationStatus =
  | "passed"
  | "blocked_language"
  | "blocked_word_filter"
  | "blocked_html"
  | "blocked_url"
  | "blocked_length";

export type PendingHint = {
  id: string;
  publicPendingText: string;
  sourceUrl?: string | null;
  sourceNote?: string | null;
  createdAt: string;
};

export type PublicIHKItem = {
  id: string;
  name: string;
  slug: string;
  city?: string | null;
  state: string;
  officialUrl?: string | null;
  info: {
    currentText: string;
    confidenceLevel: ConfidenceLevel;
    sourceSummary?: string | null;
    updatedAt: string;
  };
  pendingHints: PendingHint[];
};

export type ListResponse<T> = {
  items: T[];
  nextCursor?: string | null;
};

export type OKResponse = {
  ok: boolean;
  message?: string;
};

export type AuthMeResponse = {
  ok: boolean;
  user: {
    id: string;
    email: string;
    displayName: string;
    isVerified?: boolean;
  };
};

export type AdminMeResponse = {
  ok: boolean;
  user: {
    id: string;
    email: string;
    displayName: string;
    isVerified?: boolean;
  };
  effectiveMask: number;
  abilities: {
    canHidePendingHints: boolean;
    canManageModerationTerms: boolean;
    canManagePermissionRequests?: boolean;
  };
};

export type AdminAbility =
  | "canReviewInfoSuggestions"
  | "canApplyInfoSuggestions"
  | "canHidePendingHints"
  | "canManageModerationTerms"
  | "canManageUsers"
  | "canReadAuditLogs"
  | "canManagePermissionRequests"
  | "canPublishIHKInfo"
  | "canRollbackIHKInfo"
  | "canUpdateIHK";

export type AdminIHK = {
  id: string;
  name: string;
  slug: string;
  city?: string | null;
  state: string;
  officialUrl?: string | null;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
};

export type ReviewEvent = {
  id: string;
  action: string;
  oldStatus?: string | null;
  newStatus?: string | null;
  comment?: string | null;
  createdAt: string;
};

export type AdminInfoSuggestionListItem = {
  id: string;
  ihkId: string;
  status: ModerationStatus;
  publicPendingVisible: boolean;
  createdAt: string;
};

export type AdminInfoSuggestionDetail = {
  id: string;
  ihkId: string;
  ihk: {
    name: string;
    slug: string;
    state: string;
  };
  currentTextSnapshot: string;
  suggestedChange: string;
  publicPendingText: string;
  publicPendingVisible: boolean;
  preModerationStatus: PreModerationStatus;
  languageConfidence: number | string;
  status: ModerationStatus;
  liveCurrentText: string;
  reviewEvents: ReviewEvent[];
};

export type IHKInfoVersion = {
  id: string;
  versionNumber: number;
  changeSummary: string;
  changedBy?: string | null;
  createdAt: string;
};

export type ModerationTerm = {
  id: string;
  term: string;
  normalizedTerm?: string;
  category: "insult" | "slur" | "threat" | "sexual" | "spam" | "other";
  severity: "low" | "medium" | "high";
  isActive: boolean;
  createdAt?: string;
  updatedAt?: string;
};

export type AdminUser = {
  id: string;
  email: string;
  displayName: string;
  isVerified: boolean;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
};

export type CreateAdminUserResponse = {
  ok: boolean;
  id: string;
};

export type RoleTemplate = {
  id: string;
  name: string;
  description?: string | null;
  allowMask: number;
};

export type UserRoleAssignment = {
  id: string;
  roleTemplateId: string;
  roleName: string;
  scopeType: "global" | "state" | "ihk";
  scopeId?: string | null;
  allowMask: number;
  denyMask: number;
  grantedBy?: string | null;
  expiresAt?: string | null;
  createdAt: string;
};

export type AuditLog = {
  id: string;
  actorUserId?: string | null;
  action: string;
  resourceType: string;
  resourceId?: string | null;
  scopeType?: string | null;
  scopeId?: string | null;
  oldValue?: unknown;
  newValue?: unknown;
  ipHash?: string | null;
  userAgentHash?: string | null;
  createdAt: string;
};

export type PermissionRequestListItem = {
  id: string;
  userId: string;
  email: string;
  displayName: string;
  requestType: "registration" | "role_request";
  requestedRoleTemplateId: string;
  requestedRoleName: string;
  requestedScopeType: "global" | "state" | "ihk";
  requestedScopeId?: string | null;
  proofNote?: string | null;
  status: "pending" | "approved" | "rejected";
  createdAt: string;
};

export type PermissionRequestActivity = {
  type: "info_suggestion";
  id: string;
  status: string;
  href: string;
  createdAt: string;
};

export type PermissionRequestDetail = PermissionRequestListItem & {
  requestedAllowMask: number;
  reviewedBy?: string | null;
  reviewedAt?: string | null;
  decisionNote?: string | null;
  updatedAt: string;
  activities: PermissionRequestActivity[];
};
