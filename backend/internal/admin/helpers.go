package admin

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Philipp01105/kammer-kompass/backend/internal/db/sqlc"
	"github.com/Philipp01105/kammer-kompass/backend/internal/httpx"
	appmw "github.com/Philipp01105/kammer-kompass/backend/internal/middleware"
	"github.com/Philipp01105/kammer-kompass/backend/internal/moderation"
	"github.com/Philipp01105/kammer-kompass/backend/internal/rbac"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// requireGlobal checks the effective permission mask without enforcing a role name
func (d *adminDeps) requireGlobal(w http.ResponseWriter, r *http.Request, required rbac.Permission) bool {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return false
	}
	mask, err := d.rbac.EffectiveMask(r.Context(), userID, rbac.ResourceScope{})
	if err != nil || !rbac.HasAll(mask, required) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return false
	}
	return true
}

// requireSuperAdmin requires the permission mask and the explicit super_admin role
func (d *adminDeps) requireSuperAdmin(w http.ResponseWriter, r *http.Request, required rbac.Permission) bool {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return false
	}
	mask, err := d.rbac.EffectiveMask(r.Context(), userID, rbac.ResourceScope{})
	if err != nil || !rbac.HasAll(mask, required) || !d.hasGlobalRole(r.Context(), userID, "super_admin") {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return false
	}
	return true
}

// hasGlobalRole checks for a non-expired global assignment with the given role name
func (d *adminDeps) hasGlobalRole(ctx context.Context, userID string, roleName string) bool {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false
	}
	assignments, err := d.q.ListUserRoleAssignmentsDetailed(ctx, pgtype.UUID{Bytes: uid, Valid: true})
	if err != nil {
		return false
	}
	now := time.Now()
	for _, assignment := range assignments {
		if assignment.ScopeType != "global" || assignment.RoleName != roleName {
			continue
		}
		if assignment.ExpiresAt.Valid && !assignment.ExpiresAt.Time.After(now) {
			continue
		}
		return true
	}
	return false
}

// publishInfoPage appends a version and moves the live page pointer to it
func publishInfoPage(ctx context.Context, qtx *sqlc.Queries, page sqlc.IhkInfoPage, text string, confidenceLevel string, sourceSummary *string, changeSummary string, userID uuid.UUID, suggestionID pgtype.UUID) (sqlc.IhkInfoVersion, sqlc.IhkInfoPage, error) {
	nextVersion, err := qtx.GetNextInfoVersionNumber(ctx, page.ID)
	if err != nil {
		return sqlc.IhkInfoVersion{}, sqlc.IhkInfoPage{}, err
	}
	version, err := qtx.CreateIHKInfoVersion(ctx, sqlc.CreateIHKInfoVersionParams{
		IhkInfoPageID:           page.ID,
		IhkID:                   page.IhkID,
		VersionNumber:           nextVersion,
		OldText:                 page.CurrentText,
		NewText:                 text,
		ChangeSummary:           changeSummary,
		ChangedBy:               pgtype.UUID{Bytes: userID, Valid: true},
		BasedOnInfoSuggestionID: suggestionID,
	})
	if err != nil {
		return sqlc.IhkInfoVersion{}, sqlc.IhkInfoPage{}, err
	}
	updatedPage, err := qtx.UpdateIHKInfoPage(ctx, sqlc.UpdateIHKInfoPageParams{
		ID:              page.ID,
		CurrentText:     text,
		ConfidenceLevel: confidenceLevel,
		SourceSummary:   sourceSummary,
		LastVersionID:   version.ID,
		UpdatedBy:       pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		return sqlc.IhkInfoVersion{}, sqlc.IhkInfoPage{}, err
	}
	return version, updatedPage, nil
}

// validateLiveInfoText applies the safety checks required before publishing live content
func validateLiveInfoText(text string) error {
	if moderation.ContainsBlockedHTML(text) {
		return errors.New("text contains blocked HTML")
	}
	if err := moderation.CheckURLSafety(text, 0); err != nil {
		return errors.New("text contains blocked URL scheme")
	}
	return nil
}

func ptr[T any](v T) *T { return &v }

func optionalTrimmedString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// parseURLUUID reads a UUID path parameter and writes the client error response on failure
func parseURLUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, param)))
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid " + param})
		return uuid.UUID{}, false
	}
	return id, true
}

// uuidPtr converts a nullable database UUID to a pointer for audit payloads
func uuidPtr(id pgtype.UUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}
	u := uuid.UUID(id.Bytes)
	return &u
}

// uuidStringOrNil converts a nullable database UUID to either a string or nil JSON value
func uuidStringOrNil(id pgtype.UUID) any {
	if !id.Valid {
		return nil
	}
	return uuid.UUID(id.Bytes).String()
}

// timeOrNil converts a nullable database timestamp to an RFC3339 string or nil JSON value
func timeOrNil(t pgtype.Timestamptz) any {
	if !t.Valid {
		return nil
	}
	return t.Time.UTC().Format(time.RFC3339)
}

// ensureAdminHTTPURL accepts empty URLs and rejects non-HTTP schemes
func ensureAdminHTTPURL(raw *string) error {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	u, err := http.NewRequest(http.MethodGet, strings.TrimSpace(*raw), nil)
	if err != nil {
		return err
	}
	if u.URL.Scheme != "http" && u.URL.Scheme != "https" {
		return errors.New("scheme not allowed")
	}
	return nil
}
