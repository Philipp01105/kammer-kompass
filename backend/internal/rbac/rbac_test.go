package rbac

import "testing"

func TestPermissionHelpers(t *testing.T) {
	mask := PermInfoSuggestionRead | PermInfoSuggestionAccept | PermAuditWrite

	if !Has(mask, PermInfoSuggestionRead) {
		t.Fatal("Has() did not find present permission")
	}
	if !HasAll(mask, ActionAcceptInfoSuggestion) {
		t.Fatal("HasAll() did not match compound action")
	}
	if HasAll(mask, ActionApplyInfoSuggestion) {
		t.Fatal("HasAll() matched missing apply action")
	}
	if !HasAny(mask, PermInfoPublish|PermAuditWrite) {
		t.Fatal("HasAny() did not match one present permission")
	}
}

func TestEffectiveMaskHonorsScopeAndDeny(t *testing.T) {
	assignments := []Assignment{
		{ScopeType: ScopeGlobal, AllowMask: PermPublicRead},
		{ScopeType: ScopeState, ScopeID: "Bayern", AllowMask: PermInfoSuggestionRead | PermInfoSuggestionAccept, DenyMask: PermInfoSuggestionAccept},
		{ScopeType: ScopeIHK, ScopeID: "ihk-1", AllowMask: PermInfoSuggestionApply},
	}

	mask := EffectiveMask(assignments, ResourceScope{State: "Bayern", IHKID: "ihk-1"})
	if !HasAll(mask, PermPublicRead|PermInfoSuggestionRead|PermInfoSuggestionApply) {
		t.Fatalf("EffectiveMask() missing expected scoped permissions: %064b", mask)
	}
	if Has(mask, PermInfoSuggestionAccept) {
		t.Fatal("EffectiveMask() retained denied permission")
	}

	other := EffectiveMask(assignments, ResourceScope{State: "Hessen", IHKID: "ihk-2"})
	if other != PermPublicRead {
		t.Fatalf("EffectiveMask() for unrelated scope = %064b, want only public read", other)
	}
}
