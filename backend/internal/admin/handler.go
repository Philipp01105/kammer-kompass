package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Philipp01105/kammer-kompass/backend/internal/audit"
	"github.com/Philipp01105/kammer-kompass/backend/internal/db/sqlc"
	"github.com/Philipp01105/kammer-kompass/backend/internal/httpx"
	appmw "github.com/Philipp01105/kammer-kompass/backend/internal/middleware"
	"github.com/Philipp01105/kammer-kompass/backend/internal/moderation"
	"github.com/Philipp01105/kammer-kompass/backend/internal/rbac"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	db         *pgxpool.Pool
	q          *sqlc.Queries
	rbac       *rbac.Service
	audit      *audit.Writer
	validate   *validator.Validate
	secretSalt string
}

func NewHandler(db *pgxpool.Pool, q *sqlc.Queries, rbacSvc *rbac.Service, auditWriter *audit.Writer, secretSalt string) *Handler {
	return &Handler{
		db:         db,
		q:          q,
		rbac:       rbacSvc,
		audit:      auditWriter,
		validate:   validator.New(),
		secretSalt: secretSalt,
	}
}

// Me returns the authenticated admin user's profile and coarse-grained UI abilities
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

	httpx.JSON(w, http.StatusOK, map[string]any{
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
			"canManagePermissionRequests": rbac.HasAll(mask, rbac.ActionAssignRole) ||
				rbac.HasAll(mask, rbac.PermUserRead|rbac.PermUserUpdate|rbac.PermRoleAssign|rbac.PermAuditWrite),
		},
	})
}

// ListModerationTerms manages the word filter used by public submissions
func (h *Handler) ListModerationTerms(w http.ResponseWriter, r *http.Request) {
	if !h.requireGlobal(w, r, rbac.ActionManageModerationTerms) {
		return
	}

	terms, err := h.q.ListActiveModerationTerms(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	items := make([]any, 0, len(terms))
	for _, t := range terms {
		items = append(items, map[string]any{
			"id":             uuid.UUID(t.ID.Bytes).String(),
			"term":           t.Term,
			"normalizedTerm": t.NormalizedTerm,
			"category":       t.Category,
			"severity":       t.Severity,
			"isActive":       t.IsActive,
			"createdAt":      t.CreatedAt.Time.UTC().Format(time.RFC3339),
			"updatedAt":      t.UpdatedAt.Time.UTC().Format(time.RFC3339),
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

type createModerationTermRequest struct {
	Term     string `json:"term" validate:"required,min=1,max=200"`
	Category string `json:"category" validate:"required,oneof=insult slur threat sexual spam other"`
	Severity string `json:"severity" validate:"required,oneof=low medium high"`
}

func (h *Handler) CreateModerationTerm(w http.ResponseWriter, r *http.Request) {
	if !h.requireGlobal(w, r, rbac.ActionManageModerationTerms) {
		return
	}

	userID, _ := appmw.UserID(r)
	uid, _ := uuid.Parse(userID)

	var req createModerationTermRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
		return
	}
	req.Term = strings.TrimSpace(req.Term)
	if err := h.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}

	normalized := moderation.Normalize(req.Term)
	created, err := h.q.CreateModerationTerm(r.Context(), sqlc.CreateModerationTermParams{
		Term:           req.Term,
		NormalizedTerm: normalized,
		Category:       req.Category,
		Severity:       req.Severity,
		CreatedBy:      pgtype.UUID{Bytes: uid, Valid: true},
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	_ = h.audit.CreateAuditLog(r.Context(), r, userID, "moderation_term.create", "moderation_term", uuidPtr(created.ID), ptr("global"), nil, map[string]any{"id": created.ID.Bytes}, created)

	httpx.JSON(w, http.StatusCreated, map[string]any{"ok": true, "id": uuid.UUID(created.ID.Bytes).String()})
}

type updateModerationTermRequest struct {
	Term     *string `json:"term" validate:"omitempty,min=1,max=200"`
	Category *string `json:"category" validate:"omitempty,oneof=insult slur threat sexual spam other"`
	Severity *string `json:"severity" validate:"omitempty,oneof=low medium high"`
	IsActive *bool   `json:"isActive"`
}

func (h *Handler) UpdateModerationTerm(w http.ResponseWriter, r *http.Request) {
	if !h.requireGlobal(w, r, rbac.ActionManageModerationTerms) {
		return
	}

	userID, _ := appmw.UserID(r)

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(strings.TrimSpace(idStr))
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid id"})
		return
	}

	var req updateModerationTermRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
		return
	}
	if req.Term != nil {
		t := strings.TrimSpace(*req.Term)
		req.Term = &t
	}

	if err := h.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}

	var normalized *string
	if req.Term != nil {
		n := moderation.Normalize(*req.Term)
		normalized = &n
	}

	updated, err := h.q.UpdateModerationTerm(r.Context(), sqlc.UpdateModerationTermParams{
		Term:           req.Term,
		NormalizedTerm: normalized,
		Category:       req.Category,
		Severity:       req.Severity,
		IsActive:       req.IsActive,
		ID:             pgtype.UUID{Bytes: id, Valid: true},
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	_ = h.audit.CreateAuditLog(r.Context(), r, userID, "moderation_term.update", "moderation_term", &id, ptr("global"), nil, nil, updated)

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) DeleteModerationTerm(w http.ResponseWriter, r *http.Request) {
	if !h.requireGlobal(w, r, rbac.ActionManageModerationTerms) {
		return
	}
	userID, _ := appmw.UserID(r)

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(strings.TrimSpace(idStr))
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid id"})
		return
	}

	if err := h.q.SoftDeleteModerationTerm(r.Context(), pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	_ = h.audit.CreateAuditLog(r.Context(), r, userID, "moderation_term.delete", "moderation_term", &id, ptr("global"), nil, nil, map[string]any{"id": id.String(), "is_active": false})

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ListAuditLogs exposes the immutable admin activity stream with cursor pagination
func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	if !h.requireGlobal(w, r, rbac.PermAuditRead) {
		return
	}

	limit := 50
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	var cursorCreatedAt pgtype.Timestamptz
	var cursorID pgtype.UUID
	if cur := strings.TrimSpace(r.URL.Query().Get("cursor")); cur != "" {
		parts := strings.SplitN(cur, "|", 2)
		if len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
				cursorCreatedAt = pgtype.Timestamptz{Time: t, Valid: true}
			}
			if id, err := uuid.Parse(parts[1]); err == nil {
				cursorID = pgtype.UUID{Bytes: id, Valid: true}
			}
		}
	}

	rows, err := h.q.ListAuditLogs(r.Context(), sqlc.ListAuditLogsParams{
		CursorCreatedAt: cursorCreatedAt,
		CursorID:        cursorID,
		Limit:           int32(limit),
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	items := make([]any, 0, len(rows))
	for _, a := range rows {
		items = append(items, map[string]any{
			"id":            uuid.UUID(a.ID.Bytes).String(),
			"actorUserId":   uuidStringOrNil(a.ActorUserID),
			"action":        a.Action,
			"resourceType":  a.ResourceType,
			"resourceId":    uuidStringOrNil(a.ResourceID),
			"scopeType":     a.ScopeType,
			"scopeId":       a.ScopeID,
			"oldValue":      a.OldValue,
			"newValue":      a.NewValue,
			"ipHash":        a.IpHash,
			"userAgentHash": a.UserAgentHash,
			"createdAt":     a.CreatedAt.Time.UTC().Format(time.RFC3339),
		})
	}

	var nextCursor any = nil
	if len(rows) == limit {
		last := rows[len(rows)-1]
		nextCursor = last.CreatedAt.Time.UTC().Format(time.RFC3339) + "|" + uuid.UUID(last.ID.Bytes).String()
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "nextCursor": nextCursor})
}

// ListUsers covers admin-created users and direct role assignments
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if !h.requireGlobal(w, r, rbac.PermUserRead) {
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("query"))
	var queryPtr *string
	if query != "" {
		queryPtr = &query
	}

	limit := 50
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	var cursorCreatedAt pgtype.Timestamptz
	var cursorID pgtype.UUID
	if cur := strings.TrimSpace(r.URL.Query().Get("cursor")); cur != "" {
		parts := strings.SplitN(cur, "|", 2)
		if len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
				cursorCreatedAt = pgtype.Timestamptz{Time: t, Valid: true}
			}
			if id, err := uuid.Parse(parts[1]); err == nil {
				cursorID = pgtype.UUID{Bytes: id, Valid: true}
			}
		}
	}

	rows, err := h.q.ListUsers(r.Context(), sqlc.ListUsersParams{
		Query:           queryPtr,
		CursorCreatedAt: cursorCreatedAt,
		CursorID:        cursorID,
		Limit:           int32(limit),
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	items := make([]any, 0, len(rows))
	for _, u := range rows {
		items = append(items, map[string]any{
			"id":          uuid.UUID(u.ID.Bytes).String(),
			"email":       u.Email,
			"displayName": u.DisplayName,
			"isVerified":  u.IsVerified,
			"isActive":    u.IsActive,
			"createdAt":   u.CreatedAt.Time.UTC().Format(time.RFC3339),
			"updatedAt":   u.UpdatedAt.Time.UTC().Format(time.RFC3339),
		})
	}

	var nextCursor any = nil
	if len(rows) == limit {
		last := rows[len(rows)-1]
		nextCursor = last.CreatedAt.Time.UTC().Format(time.RFC3339) + "|" + uuid.UUID(last.ID.Bytes).String()
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "nextCursor": nextCursor})
}

type createUserRequest struct {
	Email          string  `json:"email" validate:"required,email,max=320"`
	DisplayName    string  `json:"displayName" validate:"required,min=2,max=100"`
	Password       string  `json:"password" validate:"required,min=10,max=256"`
	RoleTemplateID *string `json:"roleTemplateId" validate:"omitempty,uuid"`
	ScopeType      *string `json:"scopeType" validate:"omitempty,oneof=global state ihk"`
	ScopeID        *string `json:"scopeId" validate:"omitempty,max=200"`
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireGlobal(w, r, rbac.PermUserRead|rbac.PermUserUpdate|rbac.PermAuditWrite) {
		return
	}
	actorUserID, _ := appmw.UserID(r)
	actorID, _ := uuid.Parse(actorUserID)

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.ScopeID != nil {
		scope := strings.TrimSpace(*req.ScopeID)
		req.ScopeID = &scope
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}
	if req.RoleTemplateID != nil && !h.requireGlobal(w, r, rbac.ActionAssignRole) {
		return
	}
	if req.RoleTemplateID != nil {
		if req.ScopeType == nil || strings.TrimSpace(*req.ScopeType) == "" {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "scopeType is required when assigning a role"})
			return
		}
		if *req.ScopeType == "global" {
			req.ScopeID = nil
		} else if req.ScopeID == nil || *req.ScopeID == "" {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "scopeId is required for state and ihk scopes"})
			return
		}
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	tx, err := h.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	qtx := h.q.WithTx(tx)
	audittx := audit.NewWriter(qtx, h.secretSalt)

	user, err := qtx.CreateUser(r.Context(), sqlc.CreateUserParams{
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		httpx.JSON(w, http.StatusConflict, map[string]any{"ok": false, "message": "User already exists or cannot be created"})
		return
	}

	var assignment any
	if req.RoleTemplateID != nil {
		roleID, err := uuid.Parse(*req.RoleTemplateID)
		if err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid roleTemplateId"})
			return
		}
		role, err := qtx.GetRoleTemplateByID(r.Context(), pgtype.UUID{Bytes: roleID, Valid: true})
		if err != nil {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Role template not found"})
			return
		}
		createdAssignment, err := qtx.CreateUserRoleAssignment(r.Context(), sqlc.CreateUserRoleAssignmentParams{
			UserID:         user.ID,
			RoleTemplateID: role.ID,
			ScopeType:      *req.ScopeType,
			ScopeID:        req.ScopeID,
			AllowMask:      role.AllowMask,
			DenyMask:       0,
			GrantedBy:      pgtype.UUID{Bytes: actorID, Valid: true},
			ExpiresAt:      pgtype.Timestamptz{Valid: false},
		})
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
			return
		}
		assignment = createdAssignment
	}

	userID := uuid.UUID(user.ID.Bytes)
	_ = audittx.CreateAuditLog(r.Context(), r, actorUserID, "user.create", "user", &userID, ptr("global"), nil, nil, map[string]any{
		"id":           userID.String(),
		"email":        user.Email,
		"display_name": user.DisplayName,
		"assignment":   assignment,
	})

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"ok": true, "id": userID.String()})
}

func (h *Handler) ListRoleTemplates(w http.ResponseWriter, r *http.Request) {
	if !h.requireGlobal(w, r, rbac.PermUserRead) {
		return
	}

	rows, err := h.q.ListRoleTemplates(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	items := make([]any, 0, len(rows))
	for _, role := range rows {
		items = append(items, map[string]any{
			"id":          uuid.UUID(role.ID.Bytes).String(),
			"name":        role.Name,
			"description": role.Description,
			"allowMask":   role.AllowMask,
			"createdAt":   role.CreatedAt.Time.UTC().Format(time.RFC3339),
			"updatedAt":   role.UpdatedAt.Time.UTC().Format(time.RFC3339),
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) ListUserRoleAssignments(w http.ResponseWriter, r *http.Request) {
	if !h.requireGlobal(w, r, rbac.PermUserRead) {
		return
	}

	targetUserID, ok := parseURLUUID(w, r, "id")
	if !ok {
		return
	}

	rows, err := h.q.ListUserRoleAssignmentsDetailed(r.Context(), pgtype.UUID{Bytes: targetUserID, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	items := make([]any, 0, len(rows))
	for _, a := range rows {
		items = append(items, map[string]any{
			"id":             uuid.UUID(a.ID.Bytes).String(),
			"roleTemplateId": uuid.UUID(a.RoleTemplateID.Bytes).String(),
			"roleName":       a.RoleName,
			"scopeType":      a.ScopeType,
			"scopeId":        a.ScopeID,
			"allowMask":      a.AllowMask,
			"denyMask":       a.DenyMask,
			"grantedBy":      uuidStringOrNil(a.GrantedBy),
			"expiresAt":      timeOrNil(a.ExpiresAt),
			"createdAt":      a.CreatedAt.Time.UTC().Format(time.RFC3339),
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

type assignUserRoleRequest struct {
	RoleTemplateID string  `json:"roleTemplateId" validate:"required,uuid"`
	ScopeType      string  `json:"scopeType" validate:"required,oneof=global state ihk"`
	ScopeID        *string `json:"scopeId" validate:"omitempty,max=200"`
	ExpiresAt      *string `json:"expiresAt" validate:"omitempty"`
}

func (h *Handler) AssignUserRole(w http.ResponseWriter, r *http.Request) {
	if !h.requireGlobal(w, r, rbac.ActionAssignRole) {
		return
	}
	actorUserID, _ := appmw.UserID(r)
	actorID, _ := uuid.Parse(actorUserID)

	targetUserID, ok := parseURLUUID(w, r, "id")
	if !ok {
		return
	}

	var req assignUserRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
		return
	}
	if req.ScopeID != nil {
		scope := strings.TrimSpace(*req.ScopeID)
		req.ScopeID = &scope
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}
	if req.ScopeType == "global" {
		req.ScopeID = nil
	} else if req.ScopeID == nil || *req.ScopeID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "scopeId is required for state and ihk scopes"})
		return
	}

	roleID, err := uuid.Parse(req.RoleTemplateID)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid roleTemplateId"})
		return
	}

	role, err := h.q.GetRoleTemplateByID(r.Context(), pgtype.UUID{Bytes: roleID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Role template not found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	var expiresAt pgtype.Timestamptz
	if req.ExpiresAt != nil && strings.TrimSpace(*req.ExpiresAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.ExpiresAt))
		if err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid expiresAt"})
			return
		}
		expiresAt = pgtype.Timestamptz{Time: t, Valid: true}
	}

	assignment, err := h.q.CreateUserRoleAssignment(r.Context(), sqlc.CreateUserRoleAssignmentParams{
		UserID:         pgtype.UUID{Bytes: targetUserID, Valid: true},
		RoleTemplateID: role.ID,
		ScopeType:      req.ScopeType,
		ScopeID:        req.ScopeID,
		AllowMask:      role.AllowMask,
		DenyMask:       0,
		GrantedBy:      pgtype.UUID{Bytes: actorID, Valid: true},
		ExpiresAt:      expiresAt,
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	assignmentID := uuid.UUID(assignment.ID.Bytes)
	_ = h.audit.CreateAuditLog(r.Context(), r, actorUserID, "role.assign", "user_role_assignment", &assignmentID, ptr(req.ScopeType), req.ScopeID, nil, assignment)
	httpx.JSON(w, http.StatusCreated, map[string]any{"ok": true, "id": assignmentID.String()})
}

func (h *Handler) RevokeUserRole(w http.ResponseWriter, r *http.Request) {
	if !h.requireGlobal(w, r, rbac.PermUserRead|rbac.PermRoleRevoke|rbac.PermAuditWrite) {
		return
	}
	actorUserID, _ := appmw.UserID(r)

	targetUserID, ok := parseURLUUID(w, r, "id")
	if !ok {
		return
	}
	assignmentID, ok := parseURLUUID(w, r, "assignmentId")
	if !ok {
		return
	}

	deleted, err := h.q.DeleteUserRoleAssignment(r.Context(), sqlc.DeleteUserRoleAssignmentParams{
		ID:     pgtype.UUID{Bytes: assignmentID, Valid: true},
		UserID: pgtype.UUID{Bytes: targetUserID, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Role assignment not found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	_ = h.audit.CreateAuditLog(r.Context(), r, actorUserID, "role.revoke", "user_role_assignment", &assignmentID, ptr(deleted.ScopeType), deleted.ScopeID, deleted, nil)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ListPermissionRequests reviews self-service registration and role requests
func (h *Handler) ListPermissionRequests(w http.ResponseWriter, r *http.Request) {
	if !h.requireGlobal(w, r, rbac.ActionAssignRole) {
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}
	limit := 50
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	rows, err := h.q.ListPermissionRequests(r.Context(), sqlc.ListPermissionRequestsParams{
		Status: statusPtr,
		Limit:  int32(limit),
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	items := make([]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, map[string]any{
			"id":                      uuid.UUID(row.ID.Bytes).String(),
			"userId":                  uuid.UUID(row.UserID.Bytes).String(),
			"email":                   row.Email,
			"displayName":             row.DisplayName,
			"requestType":             row.RequestType,
			"requestedRoleTemplateId": uuid.UUID(row.RequestedRoleTemplateID.Bytes).String(),
			"requestedRoleName":       row.RequestedRoleName,
			"requestedScopeType":      row.RequestedScopeType,
			"requestedScopeId":        row.RequestedScopeID,
			"proofFileName":           row.ProofFileName,
			"proofMimeType":           row.ProofMimeType,
			"proofNote":               row.ProofNote,
			"status":                  row.Status,
			"createdAt":               row.CreatedAt.Time.UTC().Format(time.RFC3339),
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) GetPermissionRequest(w http.ResponseWriter, r *http.Request) {
	if !h.requireGlobal(w, r, rbac.ActionAssignRole) {
		return
	}
	id, ok := parseURLUUID(w, r, "id")
	if !ok {
		return
	}
	row, err := h.q.GetPermissionRequestByID(r.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	activities, err := h.permissionRequestActivity(r.Context(), row.UserID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"id":                      uuid.UUID(row.ID.Bytes).String(),
		"userId":                  uuid.UUID(row.UserID.Bytes).String(),
		"email":                   row.Email,
		"displayName":             row.DisplayName,
		"requestType":             row.RequestType,
		"requestedRoleTemplateId": uuid.UUID(row.RequestedRoleTemplateID.Bytes).String(),
		"requestedRoleName":       row.RequestedRoleName,
		"requestedAllowMask":      row.RequestedAllowMask,
		"requestedScopeType":      row.RequestedScopeType,
		"requestedScopeId":        row.RequestedScopeID,
		"proofFileName":           row.ProofFileName,
		"proofMimeType":           row.ProofMimeType,
		"proofContentBase64":      row.ProofContentBase64,
		"proofNote":               row.ProofNote,
		"status":                  row.Status,
		"reviewedBy":              uuidStringOrNil(row.ReviewedBy),
		"reviewedAt":              timeOrNil(row.ReviewedAt),
		"decisionNote":            row.DecisionNote,
		"createdAt":               row.CreatedAt.Time.UTC().Format(time.RFC3339),
		"updatedAt":               row.UpdatedAt.Time.UTC().Format(time.RFC3339),
		"activities":              activities,
	})
}

type permissionRequestDecisionRequest struct {
	Note *string `json:"note" validate:"omitempty,max=2000"`
}

func (h *Handler) ApprovePermissionRequest(w http.ResponseWriter, r *http.Request) {
	h.decidePermissionRequest(w, r, true)
}

func (h *Handler) RejectPermissionRequest(w http.ResponseWriter, r *http.Request) {
	h.decidePermissionRequest(w, r, false)
}

// decidePermissionRequest applies the review decision atomically before audit logging
// Registration rejections delete the pending account; approvals activate and grant the role
func (h *Handler) decidePermissionRequest(w http.ResponseWriter, r *http.Request, approve bool) {
	if !h.requireGlobal(w, r, rbac.ActionAssignRole) {
		return
	}
	actorUserID, _ := appmw.UserID(r)
	actorID, _ := uuid.Parse(actorUserID)
	requestID, ok := parseURLUUID(w, r, "id")
	if !ok {
		return
	}
	var req permissionRequestDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
		return
	}
	if req.Note != nil {
		note := strings.TrimSpace(*req.Note)
		req.Note = &note
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}

	tx, err := h.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	qtx := h.q.WithTx(tx)

	locked, err := qtx.LockPermissionRequestByID(r.Context(), pgtype.UUID{Bytes: requestID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	if locked.Status != "pending" {
		httpx.JSON(w, http.StatusConflict, map[string]any{"ok": false, "message": "Request is already decided"})
		return
	}
	user, err := qtx.GetUserByID(r.Context(), locked.UserID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	role, err := qtx.GetRoleTemplateByID(r.Context(), locked.RequestedRoleTemplateID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	status := "rejected"
	if approve {
		status = "approved"
	}
	if approve {
		if locked.RequestType == "registration" {
			if _, err := qtx.SetUserActiveVerified(r.Context(), sqlc.SetUserActiveVerifiedParams{
				ID:         locked.UserID,
				IsActive:   true,
				IsVerified: true,
			}); err != nil {
				httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
				return
			}
		}
		if _, err := qtx.CreateUserRoleAssignment(r.Context(), sqlc.CreateUserRoleAssignmentParams{
			UserID:         locked.UserID,
			RoleTemplateID: role.ID,
			ScopeType:      locked.RequestedScopeType,
			ScopeID:        locked.RequestedScopeID,
			AllowMask:      role.AllowMask,
			DenyMask:       0,
			GrantedBy:      pgtype.UUID{Bytes: actorID, Valid: true},
			ExpiresAt:      pgtype.Timestamptz{Valid: false},
		}); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
			return
		}
	}

	if _, err := qtx.UpdatePermissionRequestDecision(r.Context(), sqlc.UpdatePermissionRequestDecisionParams{
		ID:           locked.ID,
		Status:       status,
		ReviewedBy:   pgtype.UUID{Bytes: actorID, Valid: true},
		DecisionNote: req.Note,
	}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	if !approve && locked.RequestType == "registration" {
		if err := qtx.DeleteUserByID(r.Context(), locked.UserID); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	_ = h.audit.CreateAuditLog(r.Context(), r, actorUserID, "permission_request."+status, "permission_request", &requestID, ptr("global"), nil, nil, map[string]any{
		"user_id": user.ID.Bytes,
		"role":    role.Name,
		"status":  status,
	})

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// permissionRequestActivity returns recent public submissions by the requesting user
func (h *Handler) permissionRequestActivity(ctx context.Context, userID pgtype.UUID) ([]any, error) {
	var items []any
	infoRows, err := h.q.ListPermissionRequestInfoSuggestions(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, row := range infoRows {
		id := uuid.UUID(row.ID.Bytes).String()
		items = append(items, map[string]any{
			"type":      "info_suggestion",
			"id":        id,
			"status":    row.Status,
			"href":      "/admin/info-suggestions/" + id,
			"createdAt": row.CreatedAt.Time.UTC().Format(time.RFC3339),
		})
	}
	return items, nil
}

// ListIHKs manages IHK records and the live info pages attached to them
func (h *Handler) ListIHKs(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}
	assignments, err := h.rbac.ListAssignments(r.Context(), userID)
	if err != nil || !rbac.HasInAnyAssignment(assignments, rbac.PermIHKRead) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}

	state := strings.TrimSpace(r.URL.Query().Get("state"))
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	var statePtr *string
	if state != "" {
		statePtr = &state
	}
	var queryPtr *string
	if query != "" {
		queryPtr = &query
	}

	limit := 50
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	var cursorName *string
	var cursorID pgtype.UUID
	if cur := strings.TrimSpace(r.URL.Query().Get("cursor")); cur != "" {
		parts := strings.SplitN(cur, "|", 2)
		if len(parts) == 2 {
			cursorName = &parts[0]
			if id, err := uuid.Parse(parts[1]); err == nil {
				cursorID = pgtype.UUID{Bytes: id, Valid: true}
			}
		}
	}

	rows, err := h.q.ListAdminIHKs(r.Context(), sqlc.ListAdminIHKsParams{
		State:      statePtr,
		Query:      queryPtr,
		CursorName: cursorName,
		CursorID:   cursorID,
		Limit:      int32(limit),
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	items := make([]any, 0, len(rows))
	for _, ihk := range rows {
		scope := rbac.ResourceScope{IHKID: uuid.UUID(ihk.ID.Bytes).String(), State: ihk.State}
		if !rbac.HasAll(rbac.EffectiveMask(assignments, scope), rbac.PermIHKRead) {
			continue
		}
		items = append(items, map[string]any{
			"id":          uuid.UUID(ihk.ID.Bytes).String(),
			"name":        ihk.Name,
			"slug":        ihk.Slug,
			"city":        ihk.City,
			"state":       ihk.State,
			"officialUrl": ihk.OfficialUrl,
			"isActive":    ihk.IsActive,
			"createdAt":   ihk.CreatedAt.Time.UTC().Format(time.RFC3339),
			"updatedAt":   ihk.UpdatedAt.Time.UTC().Format(time.RFC3339),
		})
	}

	var nextCursor any = nil
	if len(rows) == limit {
		last := rows[len(rows)-1]
		nextCursor = last.Name + "|" + uuid.UUID(last.ID.Bytes).String()
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "nextCursor": nextCursor})
}

type updateAdminIHKRequest struct {
	OfficialURL *string `json:"officialUrl" validate:"omitempty,max=2000"`
}

func (h *Handler) UpdateIHK(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}
	if !h.requireSuperAdmin(w, r, rbac.PermIHKUpdate|rbac.PermAuditWrite) {
		return
	}
	ihkID, ok := parseURLUUID(w, r, "id")
	if !ok {
		return
	}

	var req updateAdminIHKRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
		return
	}
	req.OfficialURL = optionalTrimmedString(req.OfficialURL)
	if err := h.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}
	if err := ensureAdminHTTPURL(req.OfficialURL); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid officialUrl"})
		return
	}

	oldIHK, err := h.q.GetIHKByID(r.Context(), pgtype.UUID{Bytes: ihkID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	updated, err := h.q.UpdateIHKOfficialURL(r.Context(), sqlc.UpdateIHKOfficialURLParams{
		ID:          pgtype.UUID{Bytes: ihkID, Valid: true},
		OfficialUrl: req.OfficialURL,
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	_ = h.audit.CreateAuditLog(r.Context(), r, userID, "ihk.update", "ihk", &ihkID, ptr("ihk"), ptr(ihkID.String()), map[string]any{
		"official_url": oldIHK.OfficialUrl,
	}, map[string]any{
		"official_url": updated.OfficialUrl,
	})

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

type publishIHKInfoRequest struct {
	NewText         string  `json:"newText" validate:"required,min=1,max=20000"`
	ConfidenceLevel string  `json:"confidenceLevel" validate:"required,oneof=low medium high"`
	SourceSummary   *string `json:"sourceSummary" validate:"omitempty,max=2000"`
	ChangeSummary   string  `json:"changeSummary" validate:"required,min=1,max=2000"`
}

// PublishIHKInfo creates a new live-info version and updates the current page in one transaction
func (h *Handler) PublishIHKInfo(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}
	uid, _ := uuid.Parse(userID)
	ihkID, ok := parseURLUUID(w, r, "id")
	if !ok {
		return
	}

	var req publishIHKInfoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
		return
	}
	req.NewText = strings.TrimSpace(req.NewText)
	req.ChangeSummary = strings.TrimSpace(req.ChangeSummary)
	if err := h.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}
	if err := validateLiveInfoText(req.NewText); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		return
	}

	ihk, err := h.q.GetIHKByID(r.Context(), pgtype.UUID{Bytes: ihkID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	scope := rbac.ResourceScope{IHKID: ihkID.String(), State: ihk.State}
	mask, err := h.rbac.EffectiveMask(r.Context(), userID, scope)
	if err != nil || !rbac.HasAll(mask, rbac.PermIHKRead|rbac.PermInfoPublish|rbac.PermVersionCreate|rbac.PermAuditWrite) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}

	tx, err := h.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	qtx := h.q.WithTx(tx)
	audittx := audit.NewWriter(qtx, h.secretSalt)

	page, err := qtx.LockIHKInfoPageByIHKID(r.Context(), pgtype.UUID{Bytes: ihkID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			page, err = qtx.CreateIHKInfoPage(r.Context(), sqlc.CreateIHKInfoPageParams{
				IhkID:           pgtype.UUID{Bytes: ihkID, Valid: true},
				CurrentText:     "",
				ConfidenceLevel: "low",
				SourceSummary:   nil,
				LastVersionID:   pgtype.UUID{Valid: false},
				Locked:          false,
				UpdatedBy:       pgtype.UUID{Bytes: uid, Valid: true},
			})
		}
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
			return
		}
	}
	if page.Locked && !rbac.Has(mask, rbac.PermLockOverride) {
		httpx.JSON(w, http.StatusConflict, map[string]any{"ok": false, "message": "Info page is locked"})
		return
	}

	version, updatedPage, err := publishInfoPage(r.Context(), qtx, page, req.NewText, req.ConfidenceLevel, req.SourceSummary, req.ChangeSummary, uid, pgtype.UUID{Valid: false})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	_ = audittx.CreateAuditLog(r.Context(), r, userID, "live_info.publish", "ihk_info_page", uuidPtr(updatedPage.ID), ptr("ihk"), ptr(ihkID.String()), map[string]any{
		"current_text": page.CurrentText,
	}, map[string]any{
		"current_text":     req.NewText,
		"last_version_id":  uuid.UUID(version.ID.Bytes).String(),
		"version_number":   version.VersionNumber,
		"confidence_level": req.ConfidenceLevel,
	})

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "versionId": uuid.UUID(version.ID.Bytes).String(), "versionNumber": version.VersionNumber})
}

func (h *Handler) ListIHKInfoVersions(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}
	ihkID, ok := parseURLUUID(w, r, "id")
	if !ok {
		return
	}
	ihk, err := h.q.GetIHKByID(r.Context(), pgtype.UUID{Bytes: ihkID, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
		return
	}
	mask, err := h.rbac.EffectiveMask(r.Context(), userID, rbac.ResourceScope{IHKID: ihkID.String(), State: ihk.State})
	if err != nil || !rbac.HasAll(mask, rbac.PermVersionRead) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}

	rows, err := h.q.ListIHKInfoVersionsByIHKID(r.Context(), pgtype.UUID{Bytes: ihkID, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	items := make([]any, 0, len(rows))
	for _, v := range rows {
		items = append(items, map[string]any{
			"id":            uuid.UUID(v.ID.Bytes).String(),
			"versionNumber": v.VersionNumber,
			"changeSummary": v.ChangeSummary,
			"changedBy":     uuidStringOrNil(v.ChangedBy),
			"createdAt":     v.CreatedAt.Time.UTC().Format(time.RFC3339),
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

type rollbackIHKInfoRequest struct {
	VersionID       string  `json:"versionId" validate:"required,uuid"`
	ConfidenceLevel *string `json:"confidenceLevel" validate:"omitempty,oneof=low medium high"`
	SourceSummary   *string `json:"sourceSummary" validate:"omitempty,max=2000"`
	ChangeSummary   string  `json:"changeSummary" validate:"required,min=1,max=2000"`
}

// RollbackIHKInfo restores a previous version while recording the rollback as a new version
func (h *Handler) RollbackIHKInfo(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}
	uid, _ := uuid.Parse(userID)
	ihkID, ok := parseURLUUID(w, r, "id")
	if !ok {
		return
	}

	var req rollbackIHKInfoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
		return
	}
	req.ChangeSummary = strings.TrimSpace(req.ChangeSummary)
	if err := h.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}
	versionID, _ := uuid.Parse(req.VersionID)

	ihk, err := h.q.GetIHKByID(r.Context(), pgtype.UUID{Bytes: ihkID, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
		return
	}
	scope := rbac.ResourceScope{IHKID: ihkID.String(), State: ihk.State}
	mask, err := h.rbac.EffectiveMask(r.Context(), userID, scope)
	if err != nil || !rbac.HasAll(mask, rbac.PermInfoRollback|rbac.PermVersionRead|rbac.PermVersionCreate|rbac.PermAuditWrite) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}

	tx, err := h.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	qtx := h.q.WithTx(tx)
	audittx := audit.NewWriter(qtx, h.secretSalt)

	target, err := qtx.GetIHKInfoVersionByIDForIHK(r.Context(), sqlc.GetIHKInfoVersionByIDForIHKParams{
		ID:    pgtype.UUID{Bytes: versionID, Valid: true},
		IhkID: pgtype.UUID{Bytes: ihkID, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Version not found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	page, err := qtx.LockIHKInfoPageByIHKID(r.Context(), pgtype.UUID{Bytes: ihkID, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	if page.Locked && !rbac.Has(mask, rbac.PermLockOverride) {
		httpx.JSON(w, http.StatusConflict, map[string]any{"ok": false, "message": "Info page is locked"})
		return
	}
	confidenceLevel := page.ConfidenceLevel
	if req.ConfidenceLevel != nil {
		confidenceLevel = *req.ConfidenceLevel
	}
	sourceSummary := page.SourceSummary
	if req.SourceSummary != nil {
		sourceSummary = req.SourceSummary
	}

	version, updatedPage, err := publishInfoPage(r.Context(), qtx, page, target.NewText, confidenceLevel, sourceSummary, req.ChangeSummary, uid, pgtype.UUID{Valid: false})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	_ = audittx.CreateAuditLog(r.Context(), r, userID, "live_info.rollback", "ihk_info_page", uuidPtr(updatedPage.ID), ptr("ihk"), ptr(ihkID.String()), map[string]any{
		"current_text": page.CurrentText,
	}, map[string]any{
		"current_text":       target.NewText,
		"rolled_back_to":     req.VersionID,
		"last_version_id":    uuid.UUID(version.ID.Bytes).String(),
		"new_version_number": version.VersionNumber,
	})

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "versionId": uuid.UUID(version.ID.Bytes).String(), "versionNumber": version.VersionNumber})
}

// ListInfoSuggestions lists public info suggestions
func (h *Handler) ListInfoSuggestions(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}

	assignments, err := h.rbac.ListAssignments(r.Context(), userID)
	if err != nil || !rbac.HasInAnyAssignment(assignments, rbac.PermInfoSuggestionRead) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	var ihkID pgtype.UUID
	if v := strings.TrimSpace(r.URL.Query().Get("ihkId")); v != "" {
		uid, err := uuid.Parse(v)
		if err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid ihkId"})
			return
		}
		ihkID = pgtype.UUID{Bytes: uid, Valid: true}
	} else {
		ihkID = pgtype.UUID{Valid: false}
	}

	var pendingPtr *bool
	if v := strings.TrimSpace(r.URL.Query().Get("publicPendingVisible")); v != "" {
		b := v == "true" || v == "1"
		pendingPtr = &b
	}

	limit := 50
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	var cursorCreatedAt pgtype.Timestamptz
	var cursorID pgtype.UUID
	if cur := strings.TrimSpace(r.URL.Query().Get("cursor")); cur != "" {
		parts := strings.SplitN(cur, "|", 2)
		if len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
				cursorCreatedAt = pgtype.Timestamptz{Time: t, Valid: true}
			}
			if id, err := uuid.Parse(parts[1]); err == nil {
				cursorID = pgtype.UUID{Bytes: id, Valid: true}
			}
		}
	}

	rows, err := h.q.ListAdminInfoSuggestions(r.Context(), sqlc.ListAdminInfoSuggestionsParams{
		Status:               statusPtr,
		IhkID:                ihkID,
		PublicPendingVisible: pendingPtr,
		CursorCreatedAt:      cursorCreatedAt,
		CursorID:             cursorID,
		Limit:                int32(limit),
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	items := make([]any, 0, len(rows))
	for _, s := range rows {
		scope := rbac.ResourceScope{IHKID: uuid.UUID(s.IhkID.Bytes).String(), State: s.IhkState}
		mask := rbac.EffectiveMask(assignments, scope)
		if !rbac.HasAll(mask, rbac.PermInfoSuggestionRead) {
			continue
		}
		items = append(items, map[string]any{
			"id":                   uuid.UUID(s.ID.Bytes).String(),
			"ihkId":                uuid.UUID(s.IhkID.Bytes).String(),
			"status":               s.Status,
			"publicPendingVisible": s.PublicPendingVisible,
			"createdAt":            s.CreatedAt.Time.UTC().Format(time.RFC3339),
		})
	}

	var nextCursor any = nil
	if len(rows) == limit {
		last := rows[len(rows)-1]
		nextCursor = last.CreatedAt.Time.UTC().Format(time.RFC3339) + "|" + uuid.UUID(last.ID.Bytes).String()
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "nextCursor": nextCursor})
}

func (h *Handler) GetInfoSuggestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(strings.TrimSpace(idStr))
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid id"})
		return
	}

	row, err := h.q.GetAdminInfoSuggestionByID(r.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	scope := rbac.ResourceScope{IHKID: uuid.UUID(row.IhkID.Bytes).String(), State: row.IhkState}
	mask, err := h.rbac.EffectiveMask(r.Context(), userID, scope)
	if err != nil || !rbac.HasAll(mask, rbac.PermInfoSuggestionRead) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}

	events, err := h.q.ListReviewEvents(r.Context(), sqlc.ListReviewEventsParams{
		TargetType: "info_suggestion",
		TargetID:   pgtype.UUID{Bytes: id, Valid: true},
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	reviewEvents := make([]any, 0, len(events))
	for _, e := range events {
		reviewEvents = append(reviewEvents, map[string]any{
			"id":        uuid.UUID(e.ID.Bytes).String(),
			"action":    e.Action,
			"oldStatus": e.OldStatus,
			"newStatus": e.NewStatus,
			"comment":   e.Comment,
			"createdAt": e.CreatedAt.Time.UTC().Format(time.RFC3339),
		})
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"id":                   uuid.UUID(row.ID.Bytes).String(),
		"ihkId":                uuid.UUID(row.IhkID.Bytes).String(),
		"ihk":                  map[string]any{"name": row.IhkName, "slug": row.IhkSlug, "state": row.IhkState},
		"currentTextSnapshot":  row.CurrentTextSnapshot,
		"suggestedChange":      row.SuggestedChange,
		"publicPendingText":    row.PublicPendingText,
		"publicPendingVisible": row.PublicPendingVisible,
		"preModerationStatus":  row.PreModerationStatus,
		"languageConfidence":   row.LanguageConfidence,
		"status":               row.Status,
		"liveCurrentText":      row.LiveCurrentText,
		"reviewEvents":         reviewEvents,
	})
}

type statusChangeRequest struct {
	Comment *string `json:"comment" validate:"omitempty,max=2000"`
}

func (h *Handler) StartReviewInfoSuggestion(w http.ResponseWriter, r *http.Request) {
	h.changeInfoSuggestionStatus(w, r, "under_review", rbac.ActionReviewInfoSuggestion, true, "start_review")
}

func (h *Handler) AcceptInfoSuggestion(w http.ResponseWriter, r *http.Request) {
	h.changeInfoSuggestionStatus(w, r, "accepted", rbac.ActionAcceptInfoSuggestion, false, "accept")
}

func (h *Handler) RejectInfoSuggestion(w http.ResponseWriter, r *http.Request) {
	h.changeInfoSuggestionStatus(w, r, "rejected", rbac.ActionRejectInfoSuggestion, false, "reject")
}

func (h *Handler) NeedsMoreInfoSuggestion(w http.ResponseWriter, r *http.Request) {
	h.changeInfoSuggestionStatus(w, r, "needs_more_info", rbac.ActionRejectInfoSuggestion, false, "needs_more_info")
}

func (h *Handler) MarkSpamInfoSuggestion(w http.ResponseWriter, r *http.Request) {
	h.changeInfoSuggestionStatus(w, r, "spam", rbac.ActionRejectInfoSuggestion, false, "spam")
}

func (h *Handler) ReopenInfoSuggestion(w http.ResponseWriter, r *http.Request) {
	h.changeInfoSuggestionStatus(w, r, "under_review", rbac.ActionReviewInfoSuggestion, false, "reopen")
}

func (h *Handler) HidePendingInfoSuggestion(w http.ResponseWriter, r *http.Request) {
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

	idStr := chi.URLParam(r, "id")
	sid, err := uuid.Parse(strings.TrimSpace(idStr))
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid id"})
		return
	}

	var req struct {
		Reason string `json:"reason" validate:"required,min=1,max=2000"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if err := h.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}

	row, err := h.q.GetAdminInfoSuggestionByID(r.Context(), pgtype.UUID{Bytes: sid, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
		return
	}
	scope := rbac.ResourceScope{IHKID: uuid.UUID(row.IhkID.Bytes).String(), State: row.IhkState}

	mask, err := h.rbac.EffectiveMask(r.Context(), userID, scope)
	if err != nil || !rbac.HasAll(mask, rbac.ActionHidePendingHint) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}

	updated, err := h.q.HideInfoSuggestionPending(r.Context(), sqlc.HideInfoSuggestionPendingParams{
		ID:                      pgtype.UUID{Bytes: sid, Valid: true},
		PublicPendingHiddenBy:   pgtype.UUID{Bytes: uid, Valid: true},
		PublicPendingHideReason: &req.Reason,
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	_ = h.audit.CreateReviewEvent(r.Context(), "info_suggestion", sid, userID, "hide_pending", nil, nil, &req.Reason)
	_ = h.audit.CreateAuditLog(r.Context(), r, userID, "info_suggestion.hide_pending", "info_suggestion", &sid, ptr("ihk"), ptr(uuid.UUID(updated.IhkID.Bytes).String()), nil, map[string]any{"public_pending_visible": false, "reason": req.Reason})

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

type applyInfoSuggestionRequest struct {
	NewText         string  `json:"newText" validate:"required,min=1,max=20000"`
	ConfidenceLevel string  `json:"confidenceLevel" validate:"required,oneof=low medium high"`
	SourceSummary   *string `json:"sourceSummary" validate:"omitempty,max=2000"`
	ChangeSummary   string  `json:"changeSummary" validate:"required,min=1,max=2000"`
}

// ApplyInfoSuggestion publishes accepted suggestion text and marks the suggestion applied
func (h *Handler) ApplyInfoSuggestion(w http.ResponseWriter, r *http.Request) {
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

	idStr := chi.URLParam(r, "id")
	sid, err := uuid.Parse(strings.TrimSpace(idStr))
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid id"})
		return
	}

	var req applyInfoSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
		return
	}
	req.NewText = strings.TrimSpace(req.NewText)
	req.ChangeSummary = strings.TrimSpace(req.ChangeSummary)
	if err := h.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}

	if moderation.ContainsBlockedHTML(req.NewText) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "newText contains blocked HTML"})
		return
	}
	if err := moderation.CheckURLSafety(req.NewText, 0); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "newText contains blocked URL scheme"})
		return
	}

	row, err := h.q.GetAdminInfoSuggestionByID(r.Context(), pgtype.UUID{Bytes: sid, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
		return
	}
	scope := rbac.ResourceScope{IHKID: uuid.UUID(row.IhkID.Bytes).String(), State: row.IhkState}

	mask, err := h.rbac.EffectiveMask(r.Context(), userID, scope)
	if err != nil || !rbac.HasAll(mask, rbac.ActionApplyInfoSuggestion) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}

	tx, err := h.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	qtx := h.q.WithTx(tx)
	audittx := audit.NewWriter(qtx, h.secretSalt)

	sug, err := qtx.LockInfoSuggestionByID(r.Context(), pgtype.UUID{Bytes: sid, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	if sug.Status != "accepted" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Suggestion must be accepted before apply"})
		return
	}

	page, err := qtx.LockIHKInfoPageByIHKID(r.Context(), sug.IhkID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	if page.Locked {
		httpx.JSON(w, http.StatusConflict, map[string]any{"ok": false, "message": "Info page is locked"})
		return
	}

	nextVersion, err := qtx.GetNextInfoVersionNumber(r.Context(), page.ID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	version, err := qtx.CreateIHKInfoVersion(r.Context(), sqlc.CreateIHKInfoVersionParams{
		IhkInfoPageID:           page.ID,
		IhkID:                   sug.IhkID,
		VersionNumber:           nextVersion,
		OldText:                 page.CurrentText,
		NewText:                 req.NewText,
		ChangeSummary:           req.ChangeSummary,
		ChangedBy:               pgtype.UUID{Bytes: uid, Valid: true},
		BasedOnInfoSuggestionID: pgtype.UUID{Bytes: sid, Valid: true},
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	_, err = qtx.UpdateIHKInfoPage(r.Context(), sqlc.UpdateIHKInfoPageParams{
		ID:              page.ID,
		CurrentText:     req.NewText,
		ConfidenceLevel: req.ConfidenceLevel,
		SourceSummary:   req.SourceSummary,
		LastVersionID:   version.ID,
		UpdatedBy:       pgtype.UUID{Bytes: uid, Valid: true},
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	_, err = qtx.UpdateInfoSuggestionStatus(r.Context(), sqlc.UpdateInfoSuggestionStatusParams{
		ID:                   sug.ID,
		Status:               "applied",
		PublicPendingVisible: false,
		AcceptedBy:           sug.AcceptedBy,
		AppliedBy:            pgtype.UUID{Bytes: uid, Valid: true},
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	_ = audittx.CreateReviewEvent(r.Context(), "info_suggestion", sid, userID, "apply", ptr("accepted"), ptr("applied"), nil)
	_ = audittx.CreateAuditLog(r.Context(), r, userID, "info_suggestion.apply", "info_suggestion", &sid, ptr("ihk"), ptr(uuid.UUID(sug.IhkID.Bytes).String()), map[string]any{
		"status":       "accepted",
		"current_text": page.CurrentText,
	}, map[string]any{
		"status":           "applied",
		"current_text":     req.NewText,
		"version_id":       uuid.UUID(version.ID.Bytes).String(),
		"version_number":   version.VersionNumber,
		"confidence_level": req.ConfidenceLevel,
	})
	_ = audittx.CreateAuditLog(r.Context(), r, userID, "live_info.publish", "ihk_info_page", uuidPtr(page.ID), ptr("ihk"), ptr(uuid.UUID(sug.IhkID.Bytes).String()), map[string]any{
		"current_text": page.CurrentText,
	}, map[string]any{
		"current_text":    req.NewText,
		"last_version_id": uuid.UUID(version.ID.Bytes).String(),
	})

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// changeInfoSuggestionStatus centralizes review status transitions and event recording
func (h *Handler) changeInfoSuggestionStatus(w http.ResponseWriter, r *http.Request, newStatus string, required rbac.Permission, keepPending bool, action string) {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}

	idStr := chi.URLParam(r, "id")
	sid, err := uuid.Parse(strings.TrimSpace(idStr))
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid id"})
		return
	}

	row, err := h.q.GetAdminInfoSuggestionByID(r.Context(), pgtype.UUID{Bytes: sid, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
		return
	}
	scope := rbac.ResourceScope{IHKID: uuid.UUID(row.IhkID.Bytes).String(), State: row.IhkState}

	mask, err := h.rbac.EffectiveMask(r.Context(), userID, scope)
	if err != nil || !rbac.HasAll(mask, required) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}

	var req statusChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, http.ErrBodyNotAllowed) {
		req.Comment = nil
	}

	tx, err := h.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	qtx := h.q.WithTx(tx)
	audittx := audit.NewWriter(qtx, h.secretSalt)

	sug, err := qtx.LockInfoSuggestionByID(r.Context(), pgtype.UUID{Bytes: sid, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	oldStatus := sug.Status

	switch newStatus {
	case "under_review":
		if action == "reopen" {
			if oldStatus != "needs_more_info" && oldStatus != "rejected" {
				httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid status transition"})
				return
			}
			break
		}
		if oldStatus != "submitted" {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid status transition"})
			return
		}
	case "accepted", "rejected", "needs_more_info":
		if oldStatus != "under_review" {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid status transition"})
			return
		}
	case "spam":
	default:
	}

	publicPendingVisible := keepPending
	var acceptedBy = pgtype.UUID{Valid: false}
	if newStatus == "accepted" {
		uid, _ := uuid.Parse(userID)
		acceptedBy = pgtype.UUID{Bytes: uid, Valid: true}
	}

	updated, err := qtx.UpdateInfoSuggestionStatus(r.Context(), sqlc.UpdateInfoSuggestionStatusParams{
		ID:                   sug.ID,
		Status:               newStatus,
		PublicPendingVisible: publicPendingVisible,
		AcceptedBy:           acceptedBy,
		AppliedBy:            pgtype.UUID{Valid: false},
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	_ = audittx.CreateReviewEvent(r.Context(), "info_suggestion", sid, userID, action, &oldStatus, &newStatus, req.Comment)
	if newStatus == "accepted" || newStatus == "rejected" || action == "reopen" {
		_ = audittx.CreateAuditLog(r.Context(), r, userID, "info_suggestion."+action, "info_suggestion", &sid, ptr("ihk"), ptr(uuid.UUID(sug.IhkID.Bytes).String()), map[string]any{
			"status":                 oldStatus,
			"public_pending_visible": sug.PublicPendingVisible,
		}, map[string]any{
			"status":                 newStatus,
			"public_pending_visible": publicPendingVisible,
		})
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	_ = updated
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// requireGlobal checks the effective permission mask without enforcing a role name
func (h *Handler) requireGlobal(w http.ResponseWriter, r *http.Request, required rbac.Permission) bool {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return false
	}
	mask, err := h.rbac.EffectiveMask(r.Context(), userID, rbac.ResourceScope{})
	if err != nil || !rbac.HasAll(mask, required) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return false
	}
	return true
}

// requireGlobalAdminRole requires both the permission mask and a global admin role
func (h *Handler) requireGlobalAdminRole(w http.ResponseWriter, r *http.Request, required rbac.Permission) bool {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return false
	}
	mask, err := h.rbac.EffectiveMask(r.Context(), userID, rbac.ResourceScope{})
	if err != nil || !rbac.HasAll(mask, required) || !h.hasGlobalAdminRole(r.Context(), userID) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return false
	}
	return true
}

// hasGlobalAdminRole accepts global admin and super_admin assignments
func (h *Handler) hasGlobalAdminRole(ctx context.Context, userID string) bool {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false
	}
	assignments, err := h.q.ListUserRoleAssignmentsDetailed(ctx, pgtype.UUID{Bytes: uid, Valid: true})
	if err != nil {
		return false
	}
	now := time.Now()
	for _, assignment := range assignments {
		if assignment.ScopeType != "global" {
			continue
		}
		if assignment.ExpiresAt.Valid && !assignment.ExpiresAt.Time.After(now) {
			continue
		}
		if assignment.RoleName == "admin" || assignment.RoleName == "super_admin" {
			return true
		}
	}
	return false
}

// requireSuperAdmin requires the permission mask and the explicit super_admin role
func (h *Handler) requireSuperAdmin(w http.ResponseWriter, r *http.Request, required rbac.Permission) bool {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return false
	}
	mask, err := h.rbac.EffectiveMask(r.Context(), userID, rbac.ResourceScope{})
	if err != nil || !rbac.HasAll(mask, required) || !h.hasGlobalRole(r.Context(), userID, "super_admin") {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return false
	}
	return true
}

// hasGlobalRole checks for a non-expired global assignment with the given role name
func (h *Handler) hasGlobalRole(ctx context.Context, userID string, roleName string) bool {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false
	}
	assignments, err := h.q.ListUserRoleAssignmentsDetailed(ctx, pgtype.UUID{Bytes: uid, Valid: true})
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
