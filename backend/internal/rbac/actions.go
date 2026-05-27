package rbac

var ActionReviewInfoSuggestion = PermInfoSuggestionRead |
	PermInfoSuggestionTriage |
	PermAuditWrite

var ActionAcceptInfoSuggestion = PermInfoSuggestionRead |
	PermInfoSuggestionAccept |
	PermAuditWrite

var ActionRejectInfoSuggestion = PermInfoSuggestionRead |
	PermInfoSuggestionReject |
	PermAuditWrite

var ActionApplyInfoSuggestion = PermInfoSuggestionRead |
	PermInfoSuggestionApply |
	PermInfoPublish |
	PermVersionCreate |
	PermAuditWrite

var ActionHidePendingHint = PermInfoSuggestionRead |
	PermPendingHintHide |
	PermAuditWrite

var ActionManageModerationTerms = PermModerationTermRead |
	PermModerationTermCreate |
	PermModerationTermUpdate |
	PermModerationTermDelete |
	PermAuditWrite

var ActionAssignRole = PermUserRead |
	PermRoleAssign |
	PermAuditWrite
