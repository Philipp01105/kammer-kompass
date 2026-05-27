package rbac

type Permission uint64

const (
	PermPublicRead Permission = 1 << iota

	PermIHKRead
	PermIHKCreate
	PermIHKUpdate
	PermIHKDelete

	PermInfoRead
	PermInfoSuggest
	PermInfoPublish
	PermInfoRollback

	PermVersionRead
	PermVersionCreate

	PermInfoSuggestionRead
	PermInfoSuggestionComment
	PermInfoSuggestionTriage
	PermInfoSuggestionAccept
	PermInfoSuggestionReject
	PermInfoSuggestionApply

	PermPendingHintRead
	PermPendingHintHide
	PermPendingHintModerate

	PermModerationTermRead
	PermModerationTermCreate
	PermModerationTermUpdate
	PermModerationTermDelete

	PermUserRead
	PermUserUpdate
	PermRoleAssign
	PermRoleRevoke

	PermAuditRead
	PermAuditWrite

	PermSpamModerate
	PermLockOverride
	PermSystemAdmin
)

func Has(mask Permission, perm Permission) bool {
	return mask&perm == perm
}

func HasAll(mask Permission, required Permission) bool {
	return mask&required == required
}

func HasAny(mask Permission, options Permission) bool {
	return mask&options != 0
}
