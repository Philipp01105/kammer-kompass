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
	"github.com/Philipp01105/kammer-kompass/backend/internal/moderation"
	"github.com/Philipp01105/kammer-kompass/backend/internal/rbac"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type infoSuggestionService struct{ *adminDeps }

func (s *infoSuggestionService) ListInfoSuggestions(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmw.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}

	assignments, err := s.rbac.ListAssignments(r.Context(), userID)
	if err != nil || !rbac.HasInAnyAssignment(assignments, rbac.PermInfoSuggestionRead) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}
	actorID, err := uuid.Parse(userID)
	if err != nil {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
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

	rows, err := s.q.ListAdminInfoSuggestions(r.Context(), sqlc.ListAdminInfoSuggestionsParams{
		ActorUserID:          pgtype.UUID{Bytes: actorID, Valid: true},
		Status:               statusPtr,
		IhkID:                ihkID,
		PublicPendingVisible: pendingPtr,
		RequiredMask:         int64(rbac.PermInfoSuggestionRead),
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

func (s *infoSuggestionService) GetInfoSuggestion(w http.ResponseWriter, r *http.Request) {
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

	row, err := s.q.GetAdminInfoSuggestionByID(r.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	scope := rbac.ResourceScope{IHKID: uuid.UUID(row.IhkID.Bytes).String(), State: row.IhkState}
	mask, err := s.rbac.EffectiveMask(r.Context(), userID, scope)
	if err != nil || !rbac.HasAll(mask, rbac.PermInfoSuggestionRead) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}

	events, err := s.q.ListReviewEvents(r.Context(), sqlc.ListReviewEventsParams{
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

func (s *infoSuggestionService) StartReviewInfoSuggestion(w http.ResponseWriter, r *http.Request) {
	s.changeInfoSuggestionStatus(w, r, "under_review", rbac.ActionReviewInfoSuggestion, true, "start_review")
}

func (s *infoSuggestionService) AcceptInfoSuggestion(w http.ResponseWriter, r *http.Request) {
	s.changeInfoSuggestionStatus(w, r, "accepted", rbac.ActionAcceptInfoSuggestion, false, "accept")
}

func (s *infoSuggestionService) RejectInfoSuggestion(w http.ResponseWriter, r *http.Request) {
	s.changeInfoSuggestionStatus(w, r, "rejected", rbac.ActionRejectInfoSuggestion, false, "reject")
}

func (s *infoSuggestionService) NeedsMoreInfoSuggestion(w http.ResponseWriter, r *http.Request) {
	s.changeInfoSuggestionStatus(w, r, "needs_more_info", rbac.ActionRejectInfoSuggestion, false, "needs_more_info")
}

func (s *infoSuggestionService) MarkSpamInfoSuggestion(w http.ResponseWriter, r *http.Request) {
	s.changeInfoSuggestionStatus(w, r, "spam",
		rbac.PermInfoSuggestionRead|rbac.PermSpamModerate|rbac.PermAuditWrite,
		false, "spam")
}

func (s *infoSuggestionService) ReopenInfoSuggestion(w http.ResponseWriter, r *http.Request) {
	s.changeInfoSuggestionStatus(w, r, "under_review", rbac.ActionReviewInfoSuggestion, false, "reopen")
}

func (s *infoSuggestionService) HidePendingInfoSuggestion(w http.ResponseWriter, r *http.Request) {
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
	if err := s.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}

	row, err := s.q.GetAdminInfoSuggestionByID(r.Context(), pgtype.UUID{Bytes: sid, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
		return
	}
	scope := rbac.ResourceScope{IHKID: uuid.UUID(row.IhkID.Bytes).String(), State: row.IhkState}

	mask, err := s.rbac.EffectiveMask(r.Context(), userID, scope)
	if err != nil || !rbac.HasAll(mask, rbac.ActionHidePendingHint) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}

	updated, err := s.q.HideInfoSuggestionPending(r.Context(), sqlc.HideInfoSuggestionPendingParams{
		ID:                      pgtype.UUID{Bytes: sid, Valid: true},
		PublicPendingHiddenBy:   pgtype.UUID{Bytes: uid, Valid: true},
		PublicPendingHideReason: &req.Reason,
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	if err := s.audit.LogReview(r.Context(), "info_suggestion", sid, userID, "hide_pending", nil, nil, &req.Reason); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}
	if err := s.audit.Log(r.Context(), r, userID, "info_suggestion.hide_pending", "info_suggestion", &sid, ptr("ihk"), ptr(uuid.UUID(updated.IhkID.Bytes).String()), nil, map[string]any{"public_pending_visible": false, "reason": req.Reason}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

type applyInfoSuggestionRequest struct {
	NewText         string  `json:"newText" validate:"required,min=1,max=20000"`
	ConfidenceLevel string  `json:"confidenceLevel" validate:"required,oneof=low medium high"`
	SourceSummary   *string `json:"sourceSummary" validate:"omitempty,max=2000"`
	ChangeSummary   string  `json:"changeSummary" validate:"required,min=1,max=2000"`
}

// ApplyInfoSuggestion publishes accepted suggestion text and marks the suggestion applied
func (s *infoSuggestionService) ApplyInfoSuggestion(w http.ResponseWriter, r *http.Request) {
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
	if err := s.validate.Struct(req); err != nil {
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

	row, err := s.q.GetAdminInfoSuggestionByID(r.Context(), pgtype.UUID{Bytes: sid, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
		return
	}
	scope := rbac.ResourceScope{IHKID: uuid.UUID(row.IhkID.Bytes).String(), State: row.IhkState}

	mask, err := s.rbac.EffectiveMask(r.Context(), userID, scope)
	if err != nil || !rbac.HasAll(mask, rbac.ActionApplyInfoSuggestion) {
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

	if err := audittx.LogReview(r.Context(), "info_suggestion", sid, userID, "apply", ptr("accepted"), ptr("applied"), nil); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}
	if err := audittx.Log(r.Context(), r, userID, "info_suggestion.apply", "info_suggestion", &sid, ptr("ihk"), ptr(uuid.UUID(sug.IhkID.Bytes).String()), map[string]any{
		"status":       "accepted",
		"current_text": page.CurrentText,
	}, map[string]any{
		"status":           "applied",
		"current_text":     req.NewText,
		"version_id":       uuid.UUID(version.ID.Bytes).String(),
		"version_number":   version.VersionNumber,
		"confidence_level": req.ConfidenceLevel,
	}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}
	if err := audittx.Log(r.Context(), r, userID, "live_info.publish", "ihk_info_page", uuidPtr(page.ID), ptr("ihk"), ptr(uuid.UUID(sug.IhkID.Bytes).String()), map[string]any{
		"current_text": page.CurrentText,
	}, map[string]any{
		"current_text":    req.NewText,
		"last_version_id": uuid.UUID(version.ID.Bytes).String(),
	}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// changeInfoSuggestionStatus centralizes review status transitions and event recording
func (s *infoSuggestionService) changeInfoSuggestionStatus(w http.ResponseWriter, r *http.Request, newStatus string, required rbac.Permission, keepPending bool, action string) {
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

	row, err := s.q.GetAdminInfoSuggestionByID(r.Context(), pgtype.UUID{Bytes: sid, Valid: true})
	if err != nil {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
		return
	}
	scope := rbac.ResourceScope{IHKID: uuid.UUID(row.IhkID.Bytes).String(), State: row.IhkState}

	mask, err := s.rbac.EffectiveMask(r.Context(), userID, scope)
	if err != nil || !rbac.HasAll(mask, required) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Forbidden"})
		return
	}

	var req statusChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
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
		if oldStatus != "submitted" && oldStatus != "under_review" {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid status transition"})
			return
		}
	default:
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Unknown target status"})
		return
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

	if err := audittx.LogReview(r.Context(), "info_suggestion", sid, userID, action, &oldStatus, &newStatus, req.Comment); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}
	if newStatus == "accepted" || newStatus == "rejected" || action == "reopen" {
		if err := audittx.Log(r.Context(), r, userID, "info_suggestion."+action, "info_suggestion", &sid, ptr("ihk"), ptr(uuid.UUID(sug.IhkID.Bytes).String()), map[string]any{
			"status":                 oldStatus,
			"public_pending_visible": sug.PublicPendingVisible,
		}, map[string]any{
			"status":                 newStatus,
			"public_pending_visible": publicPendingVisible,
		}); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	_ = updated
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
