package public

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Philipp01105/kammer-kompass/backend/internal/db/sqlc"
	"github.com/Philipp01105/kammer-kompass/backend/internal/httpx"
	"github.com/Philipp01105/kammer-kompass/backend/internal/middleware"
	"github.com/Philipp01105/kammer-kompass/backend/internal/moderation"
	"github.com/Philipp01105/kammer-kompass/backend/internal/netx"
	"github.com/Philipp01105/kammer-kompass/backend/internal/rate_limit"
	"github.com/Philipp01105/kammer-kompass/backend/internal/security"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	q        *sqlc.Queries
	validate *validator.Validate

	limiter      *rate_limit.Limiter
	secretSalt   string
	langDetect   moderation.LanguageDetector
	maxLinkCount int
}

func NewHandler(q *sqlc.Queries, limiter *rate_limit.Limiter, secretSalt string, detector moderation.LanguageDetector) (*Handler, error) {
	if limiter == nil {
		return nil, errors.New("rate limiter must not be nil")
	}
	return &Handler{
		q:            q,
		validate:     validator.New(),
		limiter:      limiter,
		secretSalt:   secretSalt,
		langDetect:   detector,
		maxLinkCount: 3,
	}, nil
}

func (h *Handler) ListIHKs(w http.ResponseWriter, r *http.Request) {
	ip := netx.ClientIP(r)
	ipHash := security.Sha256Hex(ip + h.secretSalt)
	if allowed, err := h.limiter.Allow(r.Context(), "rl:public:list_ihks:"+ipHash, time.Minute, 30); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	} else if !allowed {
		httpx.JSON(w, http.StatusTooManyRequests, map[string]any{"ok": false, "message": "Rate limit exceeded"})
		return
	}

	state := strings.TrimSpace(r.URL.Query().Get("state"))
	query := strings.TrimSpace(r.URL.Query().Get("query"))

	includePending := false
	if v := strings.TrimSpace(r.URL.Query().Get("includePending")); v != "" {
		includePending = v == "true" || v == "1"
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
		name, id, err := decodeCursor(cur)
		if err == nil {
			cursorName = &name
			cursorID = pgtype.UUID{Bytes: id, Valid: true}
		}
	}

	var statePtr *string
	if state != "" {
		statePtr = &state
	}
	var queryPtr *string
	if query != "" {
		queryPtr = &query
	}

	rows, err := h.q.ListPublicIHKs(r.Context(), sqlc.ListPublicIHKsParams{
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

	pendingByIHK := map[pgtype.UUID][]any{}
	if includePending && len(rows) > 0 {
		ihkIDs := make([]pgtype.UUID, 0, len(rows))
		for _, row := range rows {
			ihkIDs = append(ihkIDs, row.ID)
		}
		hints, err := h.q.ListPendingHintsByIHKIDs(r.Context(), sqlc.ListPendingHintsByIHKIDsParams{
			IhkIds:      ihkIDs,
			PerIhkLimit: 5,
		})
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
			return
		}
		for _, hint := range hints {
			pendingByIHK[hint.IhkID] = append(pendingByIHK[hint.IhkID], map[string]any{
				"id":                uuidString(hint.ID),
				"publicPendingText": hint.PublicPendingText,
				"sourceUrl":         hint.SourceUrl,
				"sourceNote":        hint.SourceNote,
				"createdAt":         hint.CreatedAt.Time.UTC().Format(time.RFC3339),
			})
		}
	}

	items := make([]any, 0, len(rows))
	for _, row := range rows {
		pendingHints := []any(nil)
		if includePending {
			pendingHints = pendingByIHK[row.ID]
			if pendingHints == nil {
				pendingHints = []any{}
			}
		}

		currentText := ""
		if row.CurrentText != nil {
			currentText = *row.CurrentText
		}
		confidenceLevel := "low"
		if row.ConfidenceLevel != nil && *row.ConfidenceLevel != "" {
			confidenceLevel = *row.ConfidenceLevel
		}

		updatedAt := ""
		if row.InfoUpdatedAt.Valid {
			updatedAt = row.InfoUpdatedAt.Time.UTC().Format(time.RFC3339)
		}

		info := map[string]any{
			"currentText":     currentText,
			"confidenceLevel": confidenceLevel,
			"sourceSummary":   row.SourceSummary,
			"updatedAt":       updatedAt,
		}

		items = append(items, map[string]any{
			"id":           uuidString(row.ID),
			"name":         row.Name,
			"slug":         row.Slug,
			"city":         row.City,
			"state":        row.State,
			"officialUrl":  row.OfficialUrl,
			"info":         info,
			"pendingHints": pendingHints,
		})
	}

	var nextCursor any = nil
	if len(rows) == limit {
		last := rows[len(rows)-1]
		if last.ID.Valid {
			nextCursor = encodeCursor(last.Name, last.ID.Bytes)
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"items":      items,
		"nextCursor": nextCursor,
	})
}

func (h *Handler) GetIHKBySlug(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(chi.URLParam(r, "slug"))
	if slug == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Missing slug"})
		return
	}

	row, err := h.q.GetPublicIHKBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"ok": false, "message": "Not found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	hints, err := h.q.ListPendingHintsByIHKID(r.Context(), sqlc.ListPendingHintsByIHKIDParams{
		IhkID: row.ID,
		Limit: 20,
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	pendingHints := make([]any, 0, len(hints))
	for _, hint := range hints {
		pendingHints = append(pendingHints, map[string]any{
			"id":                uuidString(hint.ID),
			"publicPendingText": hint.PublicPendingText,
			"sourceUrl":         hint.SourceUrl,
			"sourceNote":        hint.SourceNote,
			"createdAt":         hint.CreatedAt.Time.UTC().Format(time.RFC3339),
		})
	}

	currentText := ""
	if row.CurrentText != nil {
		currentText = *row.CurrentText
	}
	confidenceLevel := "low"
	if row.ConfidenceLevel != nil && *row.ConfidenceLevel != "" {
		confidenceLevel = *row.ConfidenceLevel
	}

	updatedAt := ""
	if row.InfoUpdatedAt.Valid {
		updatedAt = row.InfoUpdatedAt.Time.UTC().Format(time.RFC3339)
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"id":          uuidString(row.ID),
		"name":        row.Name,
		"slug":        row.Slug,
		"city":        row.City,
		"state":       row.State,
		"officialUrl": row.OfficialUrl,
		"info": map[string]any{
			"currentText":     currentText,
			"confidenceLevel": confidenceLevel,
			"sourceSummary":   row.SourceSummary,
			"updatedAt":       updatedAt,
		},
		"pendingHints": pendingHints,
	})
}

type submitInfoSuggestionRequest struct {
	IhkID           string  `json:"ihkId" validate:"required,uuid"`
	SuggestedChange string  `json:"suggestedChange" validate:"required,min=20,max=3000"`
	Reason          *string `json:"reason" validate:"omitempty,max=2000"`
	SourceURL       *string `json:"sourceUrl" validate:"omitempty,max=2000"`
	SourceNote      *string `json:"sourceNote" validate:"omitempty,max=2000"`
	SubmittedEmail  *string `json:"submittedEmail" validate:"omitempty,email,max=320"`
	Honeypot        string  `json:"honeypot"`
}

func (h *Handler) SubmitInfoSuggestion(w http.ResponseWriter, r *http.Request) {
	var req submitInfoSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
		return
	}
	if req.Honeypot != "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}
	req.SuggestedChange = strings.TrimSpace(req.SuggestedChange)
	req.Reason = optionalTrimmedString(req.Reason)
	req.SourceURL = optionalTrimmedString(req.SourceURL)
	req.SourceNote = optionalTrimmedString(req.SourceNote)
	req.SubmittedEmail = optionalTrimmedString(req.SubmittedEmail)

	if err := h.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}
	if err := isHTTP(req.SourceURL); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid sourceUrl"})
		return
	}

	ip := netx.ClientIP(r)
	ipHash := security.Sha256Hex(ip + h.secretSalt)

	allowed, err := h.limiter.Allow(r.Context(), "rl:public:info_suggestions:"+ipHash, time.Hour, 5)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	if !allowed {
		httpx.JSON(w, http.StatusTooManyRequests, map[string]any{"ok": false, "message": "Rate limit exceeded"})
		return
	}

	if moderation.ContainsBlockedHTML(req.SuggestedChange) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "code": "HTML_BLOCKED", "message": "Bitte entferne HTML oder Skripte."})
		return
	}
	if err := moderation.CheckURLSafety(req.SuggestedChange, h.maxLinkCount); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "code": "URL_BLOCKED", "message": "Bitte entferne unsichere Links."})
		return
	}

	conf := moderation.DetectGermanConfidence(req.SuggestedChange, h.langDetect)
	if conf < 0.70 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "code": "LANGUAGE_NOT_GERMAN", "message": "Bitte reiche Hinweise auf Deutsch ein."})
		return
	}

	terms, err := h.q.ListActiveModerationNormalizedTerms(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	blocked, _ := moderation.CheckWordFilter(req.SuggestedChange, terms)
	if blocked {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "code": "WORD_FILTER_BLOCKED", "message": "Dein Hinweis enthält Begriffe, die nicht öffentlich eingereicht werden können."})
		return
	}

	ihkUUID, _ := uuid.Parse(req.IhkID)
	ihkID := pgtype.UUID{Bytes: ihkUUID, Valid: true}

	infoPage, err := h.q.GetIHKInfoPageByIHKID(r.Context(), ihkID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			infoPage, err = h.ensureInfoPage(r.Context(), ihkID)
		}
		if err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Unknown ihkId"})
			return
		}
	}

	var langNumeric pgtype.Numeric
	_ = langNumeric.Scan(fmt.Sprintf("%.3f", conf))

	submittedBy := h.submittedByUserID(r)

	ipHashStr := ipHash
	now := time.Now()
	_, err = h.q.CreateInfoSuggestion(r.Context(), sqlc.CreateInfoSuggestionParams{
		IhkID:                  ihkID,
		CurrentTextSnapshot:    infoPage.CurrentText,
		SuggestedChange:        req.SuggestedChange,
		PublicPendingText:      req.SuggestedChange,
		Reason:                 req.Reason,
		SourceUrl:              req.SourceURL,
		SourceNote:             req.SourceNote,
		LanguageCode:           "de",
		LanguageConfidence:     langNumeric,
		PreModerationStatus:    "passed",
		ModerationFlags:        []byte("[]"),
		PublicPendingVisible:   true,
		PublicPendingCreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
		SubmittedByUserID:      submittedBy,
		SubmittedEmail:         req.SubmittedEmail,
		IpHash:                 &ipHashStr,
		Status:                 "submitted",
	})
	if err != nil {
		slog.Error("create info suggestion failed", "error", err)
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "Dein Hinweis wurde eingereicht",
	})
}

// isHTTP returns a error if the given string is not a valid URL
func isHTTP(raw *string) error {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	u, err := url.Parse(strings.TrimSpace(*raw))
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("scheme not allowed")
	}
	return nil
}

// optionalTrimmedString returns the given string if it is not empty after trimming
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

// submittedByUserID returns the user ID of the user who submitted the request
// if the userid does not exist or is invalid we return a UUID with Valid=false
func (h *Handler) submittedByUserID(r *http.Request) pgtype.UUID {
	userID, hasUser := middleware.UserID(r)
	if !hasUser {
		return pgtype.UUID{Valid: false}
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return pgtype.UUID{Valid: false}
	}
	submittedBy := pgtype.UUID{Bytes: uid, Valid: true}
	if _, err := h.q.GetUserByID(r.Context(), submittedBy); err != nil {
		return pgtype.UUID{Valid: false}
	}
	return submittedBy
}

// encodeCursor serializes the last row's sort key into a URL-safe cursor
func encodeCursor(name string, id uuid.UUID) string {
	payload := name + "\x00" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

// decodeCursor restores the name and UUID from a cursor produced by encodeCursor
func decodeCursor(cursor string) (string, uuid.UUID, error) {
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", uuid.UUID{}, err
	}
	parts := strings.SplitN(string(b), "\x00", 2)
	if len(parts) != 2 {
		return "", uuid.UUID{}, errors.New("invalid cursor")
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return "", uuid.UUID{}, err
	}
	return parts[0], id, nil
}

func uuidString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return uuid.UUID(id.Bytes).String()
}

func (h *Handler) ensureInfoPage(ctx context.Context, ihkID pgtype.UUID) (sqlc.IhkInfoPage, error) {
	_, err := h.q.GetIHKByID(ctx, ihkID)
	if err != nil {
		return sqlc.IhkInfoPage{}, err
	}
	emptyText := ""
	conf := "low"
	created, err := h.q.CreateIHKInfoPage(ctx, sqlc.CreateIHKInfoPageParams{
		IhkID:           ihkID,
		CurrentText:     emptyText,
		ConfidenceLevel: conf,
		SourceSummary:   nil,
		LastVersionID:   pgtype.UUID{Valid: false},
		Locked:          false,
		UpdatedBy:       pgtype.UUID{Valid: false},
	})
	return created, err
}
