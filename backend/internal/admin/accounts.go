package admin

import (
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
	"github.com/Philipp01105/kammer-kompass/backend/internal/rbac"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type accountService struct{ *adminDeps }

func (s *accountService) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	if !s.requireGlobal(w, r, rbac.PermAuditRead) {
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

	rows, err := s.q.ListAuditLogs(r.Context(), sqlc.ListAuditLogsParams{
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

func (s *accountService) ListUsers(w http.ResponseWriter, r *http.Request) {
	if !s.requireGlobal(w, r, rbac.PermUserRead) {
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

	rows, err := s.q.ListUsers(r.Context(), sqlc.ListUsersParams{
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

type updateUserStatusRequest struct {
	IsActive bool `json:"isActive"`
}

func (s *accountService) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	if !s.requireGlobal(w, r, rbac.PermUserUpdate|rbac.PermAuditWrite) {
		return
	}
	actorUserID, _ := appmw.UserID(r)

	targetID, ok := parseURLUUID(w, r, "id")
	if !ok {
		return
	}

	var req updateUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
		return
	}

	if !s.canChangeUserStatus(r, actorUserID, targetID, req.IsActive) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}

	updated, err := s.q.SetUserActive(r.Context(), sqlc.SetUserActiveParams{
		ID:       pgtype.UUID{Bytes: targetID, Valid: true},
		IsActive: req.IsActive,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "User not found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	if !req.IsActive && s.sessions != nil {
		if err := s.sessions.ClearUserSessions(r.Context(), targetID.String()); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
			return
		}
	}

	if err := s.audit.Log(r.Context(), r, actorUserID, "user.update_status", "user", &targetID, ptr("global"), nil,
		nil, map[string]any{"is_active": req.IsActive}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"user": map[string]any{
			"id":       uuid.UUID(updated.ID.Bytes).String(),
			"isActive": updated.IsActive,
		},
	})
}

func (s *accountService) canChangeUserStatus(r *http.Request, actorUserID string, targetID uuid.UUID, isActive bool) bool {
	isSelf := strings.EqualFold(strings.TrimSpace(actorUserID), targetID.String())
	disabling := !isActive
	if !canChangeUserStatus(isSelf, disabling, false, false) {
		return false
	}
	if !s.hasGlobalRole(r.Context(), targetID.String(), "super_admin") {
		return true
	}
	return canChangeUserStatus(isSelf, disabling, s.actorHasSuperAdminProtection(r, actorUserID), true)
}

func canChangeUserStatus(isSelf bool, disabling bool, actorIsProtectedSuperAdmin bool, targetIsSuperAdmin bool) bool {
	if isSelf && disabling {
		return false
	}
	if targetIsSuperAdmin && !actorIsProtectedSuperAdmin {
		return false
	}
	return true
}

func (s *accountService) actorHasSuperAdminProtection(r *http.Request, actorUserID string) bool {
	mask, err := s.rbac.EffectiveMask(r.Context(), actorUserID, rbac.ResourceScope{})
	if err != nil || !rbac.HasAll(mask, rbac.PermSystemAdmin) {
		return false
	}
	return s.hasGlobalRole(r.Context(), actorUserID, "super_admin")
}

type createUserRequest struct {
	Email          string  `json:"email" validate:"required,email,max=320"`
	DisplayName    string  `json:"displayName" validate:"required,min=2,max=100"`
	Password       string  `json:"password" validate:"required,min=10,max=256"`
	RoleTemplateID *string `json:"roleTemplateId" validate:"omitempty,uuid"`
	ScopeType      *string `json:"scopeType" validate:"omitempty,oneof=global state ihk"`
	ScopeID        *string `json:"scopeId" validate:"omitempty,max=200"`
}

func (s *accountService) CreateUser(w http.ResponseWriter, r *http.Request) {
	if !s.requireSuperAdmin(w, r, rbac.PermSystemAdmin|rbac.PermUserRead|rbac.PermUserUpdate|rbac.PermAuditWrite) {
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
	if err := s.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}
	if req.RoleTemplateID != nil && !s.requireGlobal(w, r, rbac.ActionAssignRole) {
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

	tx, err := s.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	qtx := s.q.WithTx(tx)
	audittx := audit.NewWriter(qtx, s.secretSalt)

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
		actorMask, err := s.rbac.EffectiveMask(r.Context(), actorUserID, rbac.ResourceScope{})
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
			return
		}
		if rbac.Permission(role.AllowMask)&^actorMask != 0 {
			httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Cannot grant permissions you do not hold"})
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
	if err := audittx.Log(r.Context(), r, actorUserID, "user.create", "user", &userID, ptr("global"), nil, nil, map[string]any{
		"id":           userID.String(),
		"email":        user.Email,
		"display_name": user.DisplayName,
		"assignment":   assignment,
	}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"ok": true, "id": userID.String()})
}

func (s *accountService) ListRoleTemplates(w http.ResponseWriter, r *http.Request) {
	if !s.requireGlobal(w, r, rbac.PermUserRead) {
		return
	}

	rows, err := s.q.ListRoleTemplates(r.Context())
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

func (s *accountService) ListUserRoleAssignments(w http.ResponseWriter, r *http.Request) {
	if !s.requireGlobal(w, r, rbac.PermUserRead) {
		return
	}

	targetUserID, ok := parseURLUUID(w, r, "id")
	if !ok {
		return
	}

	rows, err := s.q.ListUserRoleAssignmentsDetailed(r.Context(), pgtype.UUID{Bytes: targetUserID, Valid: true})
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

func (s *accountService) AssignUserRole(w http.ResponseWriter, r *http.Request) {
	if !s.requireGlobal(w, r, rbac.ActionAssignRole) {
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
	if err := s.validate.Struct(req); err != nil {
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

	role, err := s.q.GetRoleTemplateByID(r.Context(), pgtype.UUID{Bytes: roleID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Role template not found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	actorMask, err := s.rbac.EffectiveMask(r.Context(), actorUserID, rbac.ResourceScope{})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	if rbac.Permission(role.AllowMask)&^actorMask != 0 {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Cannot grant permissions you do not hold"})
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

	assignment, err := s.q.CreateUserRoleAssignment(r.Context(), sqlc.CreateUserRoleAssignmentParams{
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
	if err := s.audit.Log(r.Context(), r, actorUserID, "role.assign", "user_role_assignment", &assignmentID, ptr(req.ScopeType), req.ScopeID, nil, assignment); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"ok": true, "id": assignmentID.String()})
}

func (s *accountService) RevokeUserRole(w http.ResponseWriter, r *http.Request) {
	if !s.requireGlobal(w, r, rbac.PermUserRead|rbac.PermRoleRevoke|rbac.PermAuditWrite) {
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

	deleted, err := s.q.DeleteUserRoleAssignment(r.Context(), sqlc.DeleteUserRoleAssignmentParams{
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

	if err := s.audit.Log(r.Context(), r, actorUserID, "role.revoke", "user_role_assignment", &assignmentID, ptr(deleted.ScopeType), deleted.ScopeID, deleted, nil); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
