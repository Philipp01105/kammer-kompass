package rbac

type ResourceScope struct {
	IHKID string
	State string
}

type ScopeType string

const (
	ScopeGlobal ScopeType = "global"
	ScopeState  ScopeType = "state"
	ScopeIHK    ScopeType = "ihk"
)

type Assignment struct {
	ScopeType ScopeType
	ScopeID   string
	AllowMask Permission
	DenyMask  Permission
}

// MatchesScope returns true if the assignment matches the given scope.
func MatchesScope(a Assignment, s ResourceScope) bool {
	switch a.ScopeType {
	case ScopeGlobal:
		return true
	case ScopeState:
		return a.ScopeID != "" && s.State != "" && a.ScopeID == s.State
	case ScopeIHK:
		return a.ScopeID != "" && s.IHKID != "" && a.ScopeID == s.IHKID
	default:
		return false
	}
}

// EffectiveMask returns the intersection of all allow and deny masks for the given scope.
func EffectiveMask(assignments []Assignment, scope ResourceScope) Permission {
	var allow Permission
	var deny Permission
	for _, a := range assignments {
		if !MatchesScope(a, scope) {
			continue
		}
		allow |= a.AllowMask
		deny |= a.DenyMask
	}
	return allow &^ deny
}

// HasInAnyAssignment returns true if any of the given assignments has the required permission.
func HasInAnyAssignment(assignments []Assignment, required Permission) bool {
	for _, a := range assignments {
		mask := a.AllowMask &^ a.DenyMask
		if HasAll(mask, required) {
			return true
		}
	}
	return false
}
