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
)

type ihkService struct{ *adminDeps }

func (s *ihkService) ListIHKs(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}
	assignments, err := s.rbac.ListAssignments(r.Context(), userID)
	if err != nil || !rbac.HasInAnyAssignment(assignments, rbac.PermIHKRead) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}
	actorID, err := uuid.Parse(userID)
	if err != nil {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
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

	rows, err := s.q.ListAdminIHKs(r.Context(), sqlc.ListAdminIHKsParams{
		ActorUserID:  pgtype.UUID{Bytes: actorID, Valid: true},
		State:        statePtr,
		Query:        queryPtr,
		RequiredMask: int64(rbac.PermIHKRead),
		CursorName:   cursorName,
		CursorID:     cursorID,
		Limit:        int32(limit),
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	items := make([]any, 0, len(rows))
	for _, ihk := range rows {
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

func (s *ihkService) UpdateIHK(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}
	if !s.requireSuperAdmin(w, r, rbac.PermIHKUpdate|rbac.PermAuditWrite) {
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
	if err := s.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}
	if err := ensureAdminHTTPURL(req.OfficialURL); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid officialUrl"})
		return
	}

	oldIHK, err := s.q.GetIHKByID(r.Context(), pgtype.UUID{Bytes: ihkID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	updated, err := s.q.UpdateIHKOfficialURL(r.Context(), sqlc.UpdateIHKOfficialURLParams{
		ID:          pgtype.UUID{Bytes: ihkID, Valid: true},
		OfficialUrl: req.OfficialURL,
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	if err := s.audit.Log(r.Context(), r, userID, "ihk.update", "ihk", &ihkID, ptr("ihk"), ptr(ihkID.String()), map[string]any{
		"official_url": oldIHK.OfficialUrl,
	}, map[string]any{
		"official_url": updated.OfficialUrl,
	}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

type publishIHKInfoRequest struct {
	NewText         string  `json:"newText" validate:"required,min=1,max=20000"`
	ConfidenceLevel string  `json:"confidenceLevel" validate:"required,oneof=low medium high"`
	SourceSummary   *string `json:"sourceSummary" validate:"omitempty,max=2000"`
	ChangeSummary   string  `json:"changeSummary" validate:"required,min=1,max=2000"`
}

// PublishIHKInfo creates a new live-info version and updates the current page in one transaction
func (s *ihkService) PublishIHKInfo(w http.ResponseWriter, r *http.Request) {
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
	if err := s.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}
	if err := validateLiveInfoText(req.NewText); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		return
	}

	ihk, err := s.q.GetIHKByID(r.Context(), pgtype.UUID{Bytes: ihkID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	scope := rbac.ResourceScope{IHKID: ihkID.String(), State: ihk.State}
	mask, err := s.rbac.EffectiveMask(r.Context(), userID, scope)
	if err != nil || !rbac.HasAll(mask, rbac.PermIHKRead|rbac.PermInfoPublish|rbac.PermVersionCreate|rbac.PermAuditWrite) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
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

	if err := audittx.Log(r.Context(), r, userID, "live_info.publish", "ihk_info_page", uuidPtr(updatedPage.ID), ptr("ihk"), ptr(ihkID.String()), map[string]any{
		"current_text": page.CurrentText,
	}, map[string]any{
		"current_text":     req.NewText,
		"last_version_id":  uuid.UUID(version.ID.Bytes).String(),
		"version_number":   version.VersionNumber,
		"confidence_level": req.ConfidenceLevel,
	}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "versionId": uuid.UUID(version.ID.Bytes).String(), "versionNumber": version.VersionNumber})
}

func (s *ihkService) ListIHKInfoVersions(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}
	ihkID, ok := parseURLUUID(w, r, "id")
	if !ok {
		return
	}
	ihk, err := s.q.GetIHKByID(r.Context(), pgtype.UUID{Bytes: ihkID, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
		return
	}
	mask, err := s.rbac.EffectiveMask(r.Context(), userID, rbac.ResourceScope{IHKID: ihkID.String(), State: ihk.State})
	if err != nil || !rbac.HasAll(mask, rbac.PermVersionRead) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}

	rows, err := s.q.ListIHKInfoVersionsByIHKID(r.Context(), pgtype.UUID{Bytes: ihkID, Valid: true})
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
func (s *ihkService) RollbackIHKInfo(w http.ResponseWriter, r *http.Request) {
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
	if err := s.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}
	versionID, _ := uuid.Parse(req.VersionID)

	ihk, err := s.q.GetIHKByID(r.Context(), pgtype.UUID{Bytes: ihkID, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
		return
	}
	scope := rbac.ResourceScope{IHKID: ihkID.String(), State: ihk.State}
	mask, err := s.rbac.EffectiveMask(r.Context(), userID, scope)
	if err != nil || !rbac.HasAll(mask, rbac.PermInfoRollback|rbac.PermVersionRead|rbac.PermVersionCreate|rbac.PermAuditWrite) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
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

	if err := audittx.Log(r.Context(), r, userID, "live_info.rollback", "ihk_info_page", uuidPtr(updatedPage.ID), ptr("ihk"), ptr(ihkID.String()), map[string]any{
		"current_text": page.CurrentText,
	}, map[string]any{
		"current_text":       target.NewText,
		"rolled_back_to":     req.VersionID,
		"last_version_id":    uuid.UUID(version.ID.Bytes).String(),
		"new_version_number": version.VersionNumber,
	}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "versionId": uuid.UUID(version.ID.Bytes).String(), "versionNumber": version.VersionNumber})
}
