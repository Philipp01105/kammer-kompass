package admin

import (
	"encoding/json"
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

type moderationTermService struct{ *adminDeps }

func (s *moderationTermService) ListModerationTerms(w http.ResponseWriter, r *http.Request) {
	if !s.requireGlobal(w, r, rbac.ActionManageModerationTerms) {
		return
	}

	terms, err := s.q.ListActiveModerationTerms(r.Context())
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

func (s *moderationTermService) CreateModerationTerm(w http.ResponseWriter, r *http.Request) {
	if !s.requireGlobal(w, r, rbac.ActionManageModerationTerms) {
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
	if err := s.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}

	normalized := moderation.Normalize(req.Term)
	created, err := s.q.CreateModerationTerm(r.Context(), sqlc.CreateModerationTermParams{
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

	if err := s.audit.Log(r.Context(), r, userID, "moderation_term.create", "moderation_term", uuidPtr(created.ID), ptr("global"), nil, map[string]any{"id": created.ID.Bytes}, created); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{"ok": true, "id": uuid.UUID(created.ID.Bytes).String()})
}

type updateModerationTermRequest struct {
	Term     *string `json:"term" validate:"omitempty,min=1,max=200"`
	Category *string `json:"category" validate:"omitempty,oneof=insult slur threat sexual spam other"`
	Severity *string `json:"severity" validate:"omitempty,oneof=low medium high"`
	IsActive *bool   `json:"isActive"`
}

func (s *moderationTermService) UpdateModerationTerm(w http.ResponseWriter, r *http.Request) {
	if !s.requireGlobal(w, r, rbac.ActionManageModerationTerms) {
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

	if err := s.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}

	var normalized *string
	if req.Term != nil {
		n := moderation.Normalize(*req.Term)
		normalized = &n
	}

	updated, err := s.q.UpdateModerationTerm(r.Context(), sqlc.UpdateModerationTermParams{
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

	if err := s.audit.Log(r.Context(), r, userID, "moderation_term.update", "moderation_term", &id, ptr("global"), nil, nil, updated); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *moderationTermService) DeleteModerationTerm(w http.ResponseWriter, r *http.Request) {
	if !s.requireGlobal(w, r, rbac.ActionManageModerationTerms) {
		return
	}
	userID, _ := appmw.UserID(r)

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(strings.TrimSpace(idStr))
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid id"})
		return
	}

	if err := s.q.SoftDeleteModerationTerm(r.Context(), pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	if err := s.audit.Log(r.Context(), r, userID, "moderation_term.delete", "moderation_term", &id, ptr("global"), nil, nil, map[string]any{"id": id.String(), "is_active": false}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Audit log write failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
