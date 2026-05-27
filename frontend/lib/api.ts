import { API_PREFIX, errorMessages } from "@/lib/constants";
import type {
  AdminIHK,
  AdminInfoSuggestionDetail,
  AdminInfoSuggestionListItem,
  AdminMeResponse,
  AdminUser,
  AuthMeResponse,
  AuditLog,
  IHKInfoVersion,
  ListResponse,
  ModerationStatus,
  ModerationTerm,
  OKResponse,
  PermissionRequestDetail,
  PermissionRequestListItem,
  PublicIHKItem,
  RoleTemplate,
  UserRoleAssignment,
} from "@/types/api";

export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

type QueryValue = string | number | boolean | null | undefined;

export function buildQuery(params: Record<string, QueryValue>) {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== "") {
      search.set(key, String(value));
    }
  }
  const value = search.toString();
  return value ? `?${value}` : "";
}

export async function apiFetch<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const headers = new Headers(options?.headers);
  if (!headers.has("Content-Type") && options?.body) {
    headers.set("Content-Type", "application/json");
  }

  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    credentials: "include",
    headers,
  });

  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = (await res.json()) as { code?: string; message?: string };
      message =
        (body.code && errorMessages[body.code]) ||
        body.message ||
        errorMessages[String(res.status)] ||
        message;
    } catch {
      message = await res.text();
    }
    throw new Error(message || "Anfrage fehlgeschlagen.");
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return res.json() as Promise<T>;
}

export const publicApi = {
  listIHKs: (query: string) =>
    apiFetch<ListResponse<PublicIHKItem>>(
      `${API_PREFIX}/public/ihks${buildQuery({ query, includePending: true })}`
    ),
  getIHK: (slug: string) =>
    apiFetch<PublicIHKItem>(`${API_PREFIX}/public/ihks/${slug}`),
  submitInfoSuggestion: (body: unknown) =>
    apiFetch<OKResponse>(`${API_PREFIX}/public/info-suggestions`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
};

export const adminApi = {
  login: (body: { email: string; password: string }) =>
    apiFetch<OKResponse>(`${API_PREFIX}/login`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  register: (body: unknown) =>
    apiFetch<OKResponse>(`${API_PREFIX}/register`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  authMe: () => apiFetch<AuthMeResponse>(`${API_PREFIX}/me`),
  listRequestableRoleTemplates: () =>
    apiFetch<{ items: RoleTemplate[] }>(`${API_PREFIX}/role-templates`),
  requestPermissions: (body: unknown) =>
    apiFetch<OKResponse>(`${API_PREFIX}/permission-requests`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  me: () => apiFetch<AdminMeResponse>(`${API_PREFIX}/admin/me`),
  listInfoSuggestions: (status?: ModerationStatus, publicPendingVisible?: boolean) =>
    apiFetch<ListResponse<AdminInfoSuggestionListItem>>(
      `${API_PREFIX}/admin/info-suggestions${buildQuery({
        status,
        publicPendingVisible,
      })}`
    ),
  getInfoSuggestion: (id: string) =>
    apiFetch<AdminInfoSuggestionDetail>(
      `${API_PREFIX}/admin/info-suggestions/${id}`
    ),
  postInfoSuggestionAction: (id: string, action: string, body?: unknown) =>
    apiFetch<OKResponse>(`${API_PREFIX}/admin/info-suggestions/${id}/${action}`, {
      method: "POST",
      body: body ? JSON.stringify(body) : JSON.stringify({}),
    }),
  applyInfoSuggestion: (id: string, body: unknown) =>
    adminApi.postInfoSuggestionAction(id, "apply", body),
  listIHKs: (query?: string, state?: string) =>
    apiFetch<ListResponse<AdminIHK>>(
      `${API_PREFIX}/admin/ihks${buildQuery({ query, state })}`
    ),
  updateIHK: (id: string, body: unknown) =>
    apiFetch<OKResponse>(`${API_PREFIX}/admin/ihks/${id}`, {
      method: "PATCH",
      body: JSON.stringify(body),
    }),
  publishIHKInfo: (id: string, body: unknown) =>
    apiFetch<OKResponse & { versionId: string; versionNumber: number }>(
      `${API_PREFIX}/admin/ihks/${id}/info/publish`,
      { method: "POST", body: JSON.stringify(body) }
    ),
  listIHKVersions: (id: string) =>
    apiFetch<{ items: IHKInfoVersion[] }>(
      `${API_PREFIX}/admin/ihks/${id}/info/versions`
    ),
  rollbackIHKInfo: (id: string, body: unknown) =>
    apiFetch<OKResponse>(`${API_PREFIX}/admin/ihks/${id}/info/rollback`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  listModerationTerms: () =>
    apiFetch<{ items: ModerationTerm[] }>(`${API_PREFIX}/admin/moderation-terms`),
  createModerationTerm: (body: unknown) =>
    apiFetch<OKResponse>(`${API_PREFIX}/admin/moderation-terms`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  updateModerationTerm: (id: string, body: unknown) =>
    apiFetch<OKResponse>(`${API_PREFIX}/admin/moderation-terms/${id}`, {
      method: "PATCH",
      body: JSON.stringify(body),
    }),
  deleteModerationTerm: (id: string) =>
    apiFetch<OKResponse>(`${API_PREFIX}/admin/moderation-terms/${id}`, {
      method: "DELETE",
    }),
  listUsers: (query?: string) =>
    apiFetch<ListResponse<AdminUser>>(`${API_PREFIX}/admin/users${buildQuery({ query })}`),
  createUser: (body: unknown) =>
    apiFetch<OKResponse & { id: string }>(`${API_PREFIX}/admin/users`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  listRoleTemplates: () =>
    apiFetch<{ items: RoleTemplate[] }>(`${API_PREFIX}/admin/role-templates`),
  listUserRoles: (id: string) =>
    apiFetch<{ items: UserRoleAssignment[] }>(`${API_PREFIX}/admin/users/${id}/roles`),
  assignUserRole: (id: string, body: unknown) =>
    apiFetch<OKResponse>(`${API_PREFIX}/admin/users/${id}/roles`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  revokeUserRole: (userId: string, assignmentId: string) =>
    apiFetch<OKResponse>(
      `${API_PREFIX}/admin/users/${userId}/roles/${assignmentId}`,
      { method: "DELETE" }
    ),
  listAuditLogs: () =>
    apiFetch<ListResponse<AuditLog>>(`${API_PREFIX}/admin/audit-logs`),
  listPermissionRequests: (status?: string) =>
    apiFetch<ListResponse<PermissionRequestListItem>>(
      `${API_PREFIX}/admin/permission-requests${buildQuery({ status })}`
    ),
  getPermissionRequest: (id: string) =>
    apiFetch<PermissionRequestDetail>(`${API_PREFIX}/admin/permission-requests/${id}`),
  decidePermissionRequest: (id: string, action: "approve" | "reject", body?: unknown) =>
    apiFetch<OKResponse>(`${API_PREFIX}/admin/permission-requests/${id}/${action}`, {
      method: "POST",
      body: JSON.stringify(body ?? {}),
    }),
};
