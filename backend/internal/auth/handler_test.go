package auth

import "testing"

func TestNormalizeLoginIdentifierDoesNotMapSuperAdminAlias(t *testing.T) {
	got := normalizeLoginIdentifier(" super_admin ")
	if got != "super_admin" {
		t.Fatalf("normalizeLoginIdentifier() = %q, want literal super_admin", got)
	}
	if got == "super_admin@local.invalid" {
		t.Fatal("super_admin alias must not map to bootstrap email")
	}
}

func TestNewHandlerRequiresSecurityDependencies(t *testing.T) {
	if _, err := NewHandler(nil, nil, nil, "salt"); err == nil {
		t.Fatal("NewHandler accepted nil limiter/session dependencies")
	}
}
