import { useQuery } from "@tanstack/react-query";
import { adminApi } from "@/lib/api";
import type { AdminAbility, AdminMeResponse } from "@/types/api";

export const Permission = {
  publicRead: 2 ** 0,
  ihkRead: 2 ** 1,
  ihkCreate: 2 ** 2,
  ihkUpdate: 2 ** 3,
  ihkDelete: 2 ** 4,
  infoRead: 2 ** 5,
  infoSuggest: 2 ** 6,
  infoPublish: 2 ** 7,
  infoRollback: 2 ** 8,
  versionRead: 2 ** 9,
  versionCreate: 2 ** 10,
  infoSuggestionRead: 2 ** 11,
  infoSuggestionComment: 2 ** 12,
  infoSuggestionTriage: 2 ** 13,
  infoSuggestionAccept: 2 ** 14,
  infoSuggestionReject: 2 ** 15,
  infoSuggestionApply: 2 ** 16,
  pendingHintRead: 2 ** 17,
  pendingHintHide: 2 ** 18,
  pendingHintModerate: 2 ** 19,
  moderationTermRead: 2 ** 20,
  moderationTermCreate: 2 ** 21,
  moderationTermUpdate: 2 ** 22,
  moderationTermDelete: 2 ** 23,
  userRead: 2 ** 24,
  userUpdate: 2 ** 25,
  roleAssign: 2 ** 26,
  roleRevoke: 2 ** 27,
  auditRead: 2 ** 28,
  auditWrite: 2 ** 29,
  spamModerate: 2 ** 30,
  lockOverride: 2 ** 31,
  systemAdmin: 2 ** 32,
} as const;

export function hasEvery(mask: number, permissions: number[]) {
  if (mask < 0) return true;
  return permissions.every((permission) => (Math.floor(mask / permission) % 2) >= 1);
}

export function getAbilities(me?: AdminMeResponse | null): Record<AdminAbility, boolean> {
  const mask = me?.effectiveMask ?? 0;
  if (mask < 0) {
    return {
      canReviewInfoSuggestions: true,
      canApplyInfoSuggestions: true,
      canHidePendingHints: true,
      canManageModerationTerms: true,
      canManageUsers: true,
      canReadAuditLogs: true,
      canManagePermissionRequests: true,
      canPublishIHKInfo: true,
      canRollbackIHKInfo: true,
      canUpdateIHK: true,
    };
  }
  return {
    canReviewInfoSuggestions: hasEvery(mask, [
      Permission.infoSuggestionRead,
      Permission.infoSuggestionTriage,
    ]),
    canApplyInfoSuggestions: hasEvery(mask, [
      Permission.infoSuggestionRead,
      Permission.infoSuggestionApply,
      Permission.infoPublish,
      Permission.versionCreate,
    ]),
    canHidePendingHints:
      Boolean(me?.abilities.canHidePendingHints) ||
      hasEvery(mask, [Permission.infoSuggestionRead, Permission.pendingHintHide]),
    canManageModerationTerms:
      Boolean(me?.abilities.canManageModerationTerms) ||
      hasEvery(mask, [
        Permission.moderationTermRead,
        Permission.moderationTermCreate,
        Permission.moderationTermUpdate,
        Permission.moderationTermDelete,
      ]),
    canManageUsers: hasEvery(mask, [Permission.userRead, Permission.roleAssign]),
    canReadAuditLogs: hasEvery(mask, [Permission.auditRead]),
    canManagePermissionRequests:
      Boolean(me?.abilities.canManagePermissionRequests) ||
      hasEvery(mask, [Permission.userRead, Permission.roleAssign, Permission.auditWrite]),
    canPublishIHKInfo: hasEvery(mask, [
      Permission.ihkRead,
      Permission.infoPublish,
      Permission.versionCreate,
    ]),
    canRollbackIHKInfo: hasEvery(mask, [
      Permission.infoRollback,
      Permission.versionCreate,
    ]),
    canUpdateIHK: hasEvery(mask, [Permission.ihkUpdate, Permission.auditWrite]),
  };
}

export function useAdminMe() {
  return useQuery({
    queryKey: ["admin-me"],
    queryFn: adminApi.me,
    retry: false,
  });
}

export function useAbilities() {
  const query = useAdminMe();
  return { ...query, abilities: getAbilities(query.data) };
}
