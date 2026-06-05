package admin

import (
	"net/http"

	"github.com/Philipp01105/kammer-kompass/backend/internal/audit"
	"github.com/Philipp01105/kammer-kompass/backend/internal/auth"
	"github.com/Philipp01105/kammer-kompass/backend/internal/db/sqlc"
	"github.com/Philipp01105/kammer-kompass/backend/internal/httpx"
	appmw "github.com/Philipp01105/kammer-kompass/backend/internal/middleware"
	"github.com/Philipp01105/kammer-kompass/backend/internal/rbac"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type adminDeps struct {
	db         *pgxpool.Pool
	q          *sqlc.Queries
	rbac       *rbac.Service
	audit      *audit.Writer
	sessions   *auth.SessionManager
	validate   *validator.Validate
	secretSalt string
}

type Handler struct {
	*adminDeps
	*moderationTermService
	*accountService
	*ihkService
	*infoSuggestionService
}

func NewHandler(db *pgxpool.Pool, q *sqlc.Queries, rbacSvc *rbac.Service, auditWriter *audit.Writer, sessionMgr *auth.SessionManager, secretSalt string) *Handler {
	deps := &adminDeps{
		db:         db,
		q:          q,
		rbac:       rbacSvc,
		audit:      auditWriter,
		sessions:   sessionMgr,
		validate:   validator.New(),
		secretSalt: secretSalt,
	}
	return &Handler{
		adminDeps:             deps,
		moderationTermService: &moderationTermService{adminDeps: deps},
		accountService:        &accountService{adminDeps: deps},
		ihkService:            &ihkService{adminDeps: deps},
		infoSuggestionService: &infoSuggestionService{adminDeps: deps},
	}
}
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}

	user, err := h.q.GetUserByID(r.Context(), pgtype.UUID{Bytes: uid, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}

	mask, err := h.rbac.EffectiveMask(r.Context(), userID, rbac.ResourceScope{})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	adminAreaPermissions := rbac.PermInfoSuggestionRead |
		rbac.PermModerationTermRead |
		rbac.PermUserRead |
		rbac.PermAuditRead |
		rbac.PermIHKUpdate |
		rbac.PermInfoPublish |
		rbac.PermInfoRollback |
		rbac.PermSystemAdmin
	if !rbac.HasAny(mask, adminAreaPermissions) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}

	httpx.JSON(w, http.StatusOK, adminMeResponse(user, mask))
}

func adminMeResponse(user sqlc.User, mask rbac.Permission) map[string]any {
	return map[string]any{
		"ok": true,
		"user": map[string]any{
			"id":          uuid.UUID(user.ID.Bytes).String(),
			"email":       user.Email,
			"displayName": user.DisplayName,
			"isVerified":  user.IsVerified,
		},
		"effectiveMask": int64(mask),
		"abilities": map[string]any{
			"canHidePendingHints":      rbac.HasAll(mask, rbac.ActionHidePendingHint),
			"canManageModerationTerms": rbac.HasAll(mask, rbac.ActionManageModerationTerms),
		},
	}
}
