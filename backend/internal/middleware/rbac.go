package middleware

import (
	"errors"
	"net/http"

	"github.com/Philipp01105/kammer-kompass/backend/internal/httpx"
	"github.com/Philipp01105/kammer-kompass/backend/internal/rbac"
)

type ScopeResolver func(*http.Request) (rbac.ResourceScope, error)

// RequirePermissions middleware requires the user to have the specified permission
func RequirePermissions(svc *rbac.Service, required rbac.Permission, resolveScope ScopeResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserID(r)
			if !ok {
				httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
				return
			}

			scope, err := resolveScope(r)
			if err != nil {
				httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid scope"})
				return
			}

			mask, err := svc.EffectiveMask(r.Context(), userID, scope)
			if err != nil {
				if errors.Is(err, rbac.ErrInvalidUserID) {
					httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
					return
				}
				httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
				return
			}

			if !rbac.HasAll(mask, required) {
				httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
				return
			}

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// RequireAnyAssignmentPermissions requires the user to hold the given
// permission set in at least one active assignment. It is a coarse route-level
// gate for endpoints whose exact resource scope must still be resolved inside
// the handler before the final RBAC decision.
func RequireAnyAssignmentPermissions(svc *rbac.Service, required rbac.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserID(r)
			if !ok {
				httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
				return
			}

			assignments, err := svc.ListAssignments(r.Context(), userID)
			if err != nil {
				if errors.Is(err, rbac.ErrInvalidUserID) {
					httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
					return
				}
				httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
				return
			}

			if !rbac.HasInAnyAssignment(assignments, required) {
				httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
				return
			}

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
