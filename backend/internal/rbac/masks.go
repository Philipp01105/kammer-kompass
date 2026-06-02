package rbac

type RoleTemplateDefinition struct {
	Name        string
	Description string
	AllowMask   Permission
}

var AnonymousMask = PermPublicRead |
	PermIHKRead |
	PermInfoRead |
	PermInfoSuggest |
	PermPendingHintRead

var RegisteredUserMask = AnonymousMask

var ContributorMask = RegisteredUserMask |
	PermInfoSuggestionRead |
	PermInfoSuggestionComment

var ReviewerMask = PermPublicRead |
	PermIHKRead |
	PermInfoRead |
	PermPendingHintRead |
	PermPendingHintHide |
	PermInfoSuggestionRead |
	PermInfoSuggestionComment |
	PermInfoSuggestionTriage |
	PermInfoSuggestionAccept |
	PermInfoSuggestionReject |
	PermVersionRead |
	PermAuditWrite

var WriterMask = PermPublicRead |
	PermIHKRead |
	PermInfoRead |
	PermPendingHintRead |
	PermInfoSuggestionRead |
	PermInfoSuggestionComment |
	PermInfoSuggestionApply |
	PermInfoPublish |
	PermVersionRead |
	PermVersionCreate |
	PermAuditWrite

var RegionalLeadMask = ReviewerMask | WriterMask

var AdminMask = PermPublicRead |
	PermIHKRead |
	PermIHKUpdate |
	PermInfoRead |
	PermInfoPublish |
	PermInfoRollback |
	PermVersionRead |
	PermVersionCreate |
	PermPendingHintRead |
	PermPendingHintHide |
	PermPendingHintModerate |
	PermModerationTermRead |
	PermModerationTermCreate |
	PermModerationTermUpdate |
	PermModerationTermDelete |
	PermInfoSuggestionRead |
	PermInfoSuggestionTriage |
	PermInfoSuggestionAccept |
	PermInfoSuggestionReject |
	PermInfoSuggestionApply |
	PermUserRead |
	PermUserUpdate |
	PermRoleAssign |
	PermRoleRevoke |
	PermAuditRead |
	PermAuditWrite |
	PermSpamModerate

// AllPermissions is the explicit union of every defined permission bit.
// SuperAdminMask is derived from this — update here when adding new permission constants
// so the DB stores a non-negative value and the API returns a meaningful positive mask.
var AllPermissions = PermPublicRead | PermIHKRead | PermIHKCreate | PermIHKUpdate | PermIHKDelete |
	PermInfoRead | PermInfoSuggest | PermInfoPublish | PermInfoRollback |
	PermVersionRead | PermVersionCreate |
	PermInfoSuggestionRead | PermInfoSuggestionComment | PermInfoSuggestionTriage |
	PermInfoSuggestionAccept | PermInfoSuggestionReject | PermInfoSuggestionApply |
	PermPendingHintRead | PermPendingHintHide | PermPendingHintModerate |
	PermModerationTermRead | PermModerationTermCreate | PermModerationTermUpdate | PermModerationTermDelete |
	PermUserRead | PermUserUpdate | PermRoleAssign | PermRoleRevoke |
	PermAuditRead | PermAuditWrite |
	PermSpamModerate | PermLockOverride | PermSystemAdmin

var SuperAdminMask = AllPermissions

func RoleTemplateDefinitions() []RoleTemplateDefinition {
	return []RoleTemplateDefinition{
		{
			Name:        "anonymous",
			Description: "Public read + submit suggestions/proposals",
			AllowMask:   AnonymousMask,
		},
		{
			Name:        "registered_user",
			Description: "Like anonymous but trackable",
			AllowMask:   RegisteredUserMask,
		},
		{
			Name:        "contributor",
			Description: "Can read/comment on suggestions",
			AllowMask:   ContributorMask,
		},
		{
			Name:        "reviewer",
			Description: "Can triage/accept/reject info suggestions",
			AllowMask:   ReviewerMask,
		},
		{
			Name:        "writer",
			Description: "Can apply accepted info suggestions",
			AllowMask:   WriterMask,
		},
		{
			Name:        "regional_lead",
			Description: "Regional reviewer + writer",
			AllowMask:   RegionalLeadMask,
		},
		{
			Name:        "admin",
			Description: "Manage system incl proposals/terms/users",
			AllowMask:   AdminMask,
		},
		{
			Name:        "super_admin",
			Description: "Full technical access",
			AllowMask:   SuperAdminMask,
		},
	}
}
