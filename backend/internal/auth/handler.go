package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Philipp01105/kammer-kompass/backend/internal/httpx"
	"github.com/Philipp01105/kammer-kompass/backend/internal/netx"
	"github.com/Philipp01105/kammer-kompass/backend/internal/rate_limit"
	"github.com/Philipp01105/kammer-kompass/backend/internal/security"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	store      *Store
	sessions   *SessionManager
	validate   *validator.Validate
	limiter    *rate_limit.Limiter
	secretSalt string
}

func NewHandler(db *pgxpool.Pool, sessions *SessionManager, limiter *rate_limit.Limiter, secretSalt string) (*Handler, error) {
	if limiter == nil {
		return nil, errors.New("rate limiter must not be nil")
	}
	if sessions == nil {
		return nil, errors.New("session manager must not be nil")
	}
	return &Handler{
		store:      NewStore(db),
		sessions:   sessions,
		validate:   validator.New(),
		limiter:    limiter,
		secretSalt: secretSalt,
	}, nil
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,max=320"`
	Password string `json:"password" validate:"required,min=1,max=256"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
		return
	}
	req.Email = normalizeLoginIdentifier(req.Email)

	if err := h.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		h.rejectFailedLogin(w, r, req.Email)
		return
	}
	if !user.IsActive {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"ok": false, "message": "Account disabled"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.rejectFailedLogin(w, r, req.Email)
		return
	}

	if err := h.sessions.SetUserID(w, r, user.ID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	_ = h.sessions.Clear(w, r)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.sessions.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}

	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			_ = h.sessions.Clear(w, r)
			httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	if !user.IsActive {
		_ = h.sessions.Clear(w, r)
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"user": map[string]any{
			"id":          user.ID,
			"email":       user.Email,
			"displayName": user.DisplayName,
			"isVerified":  user.IsVerified,
		},
	})
}

func (h *Handler) rejectFailedLogin(w http.ResponseWriter, r *http.Request, email string) {
	if h.limiter != nil {
		ipHash := security.Sha256Hex(netx.ClientIP(r) + h.secretSalt)
		emailHash := security.Sha256Hex(strings.ToLower(strings.TrimSpace(email)) + h.secretSalt)

		ipAllowed, ipErr := h.limiter.Allow(r.Context(), "rl:auth:login_failed:ip:"+ipHash, 15*time.Minute, 5)
		emailAllowed, emailErr := h.limiter.Allow(r.Context(), "rl:auth:login_failed:email:"+emailHash, 15*time.Minute, 5)
		if ipErr != nil || emailErr != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
			return
		}
		if !ipAllowed || !emailAllowed {
			httpx.JSON(w, http.StatusTooManyRequests, map[string]any{"ok": false, "message": "Rate limit exceeded"})
			return
		}
	}

	httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Invalid credentials"})
}

func (h *Handler) allowRate(w http.ResponseWriter, r *http.Request, key string, window time.Duration, limit int) bool {
	allowed, err := h.limiter.Allow(r.Context(), key, window, limit)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return false
	}
	if !allowed {
		httpx.JSON(w, http.StatusTooManyRequests, map[string]any{"ok": false, "code": "RATE_LIMITED", "message": "Rate limit exceeded"})
		return false
	}
	return true
}

func normalizeLoginIdentifier(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

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
