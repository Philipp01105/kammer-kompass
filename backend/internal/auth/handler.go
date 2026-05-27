package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Philipp01105/kammer-kompass/backend/internal/config"
	"github.com/Philipp01105/kammer-kompass/backend/internal/httpx"
	"github.com/Philipp01105/kammer-kompass/backend/internal/netx"
	"github.com/Philipp01105/kammer-kompass/backend/internal/rate_limit"
	"github.com/Philipp01105/kammer-kompass/backend/internal/security"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// 2 mebibytes
const maxProofUploadBytes = 2 * 1024 * 1024

var allowedProofMimeTypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
}

type Handler struct {
	store      *Store
	sessions   *SessionManager
	validate   *validator.Validate
	limiter    *rate_limit.Limiter
	secretSalt string
}

func NewHandler(db *pgxpool.Pool, sessionCfg config.SessionConfig, limiter *rate_limit.Limiter, secretSalt string) (*Handler, error) {
	sessions, err := NewSessionManager(sessionCfg)
	if err != nil {
		return nil, err
	}
	return &Handler{
		store:      NewStore(db),
		sessions:   sessions,
		validate:   validator.New(),
		limiter:    limiter,
		secretSalt: secretSalt,
	}, nil
}

type registerRequest struct {
	Email                 string  `json:"email" validate:"required,email,max=320"`
	DisplayName           string  `json:"displayName" validate:"required,min=2,max=100"`
	Password              string  `json:"password" validate:"required,min=10,max=256"`
	RequestedRoleTemplate string  `json:"requestedRoleTemplateId" validate:"omitempty,uuid"`
	RequestedScopeType    string  `json:"requestedScopeType" validate:"omitempty,oneof=global state ihk"`
	RequestedScopeID      *string `json:"requestedScopeId" validate:"omitempty,max=200"`
	ProofFileName         *string `json:"proofFileName" validate:"omitempty,max=255"`
	ProofMimeType         *string `json:"proofMimeType" validate:"omitempty,max=100"`
	ProofContentBase64    *string `json:"proofContentBase64" validate:"omitempty,max=2000000"`
	ProofNote             *string `json:"proofNote" validate:"omitempty,max=2000"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.RequestedRoleTemplate = strings.TrimSpace(req.RequestedRoleTemplate)
	req.RequestedScopeType = strings.TrimSpace(req.RequestedScopeType)
	req.RequestedScopeID = optionalTrimmedString(req.RequestedScopeID)
	req.ProofFileName = optionalTrimmedString(req.ProofFileName)
	req.ProofMimeType = optionalTrimmedString(req.ProofMimeType)
	req.ProofContentBase64 = optionalTrimmedString(req.ProofContentBase64)
	req.ProofNote = optionalTrimmedString(req.ProofNote)

	if err := h.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}
	if err := validateProofUpload(req.ProofMimeType, req.ProofContentBase64); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		return
	}
	requestsRights := req.RequestedRoleTemplate != ""
	limitKey := "rl:auth:register:"
	limitWindow := time.Hour
	limitCount := 5
	if requestsRights {
		limitKey = "rl:auth:register_permission:"
		limitWindow = 24 * time.Hour
		limitCount = 3
	}
	if !h.allowRate(w, r, limitKey+security.Sha256Hex(netx.ClientIP(r)+h.secretSalt), limitWindow, limitCount) {
		return
	}
	if requestsRights {
		if req.RequestedScopeType == "" {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "requestedScopeType is required"})
			return
		}
		if req.RequestedScopeType == "global" {
			req.RequestedScopeID = nil
		} else if req.RequestedScopeID == nil || *req.RequestedScopeID == "" {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "requestedScopeId is required for state and ihk scopes"})
			return
		}
	}

	pwHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	var user User
	if requestsRights {
		user, err = h.store.RegisterUserWithPermissionRequest(r.Context(), req.Email, req.DisplayName, string(pwHash), req.RequestedRoleTemplate, req.RequestedScopeType, req.RequestedScopeID, req.ProofFileName, req.ProofMimeType, req.ProofContentBase64, req.ProofNote)
	} else {
		user, err = h.store.RegisterUser(r.Context(), req.Email, req.DisplayName, string(pwHash))
	}
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			httpx.JSON(w, http.StatusConflict, map[string]any{"ok": false, "message": "Email already registered"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}

	if requestsRights {
		httpx.JSON(w, http.StatusAccepted, map[string]any{
			"ok":      true,
			"message": "Deine Registrierung wurde eingereicht und muss von einem Admin geprüft werden.",
			"user": map[string]any{
				"id":          user.ID,
				"email":       user.Email,
				"displayName": user.DisplayName,
				"isVerified":  user.IsVerified,
			},
		})
		return
	}

	_ = h.sessions.SetUserID(w, r, user.ID)
	httpx.JSON(w, http.StatusCreated, map[string]any{
		"ok": true,
		"user": map[string]any{
			"id":          user.ID,
			"email":       user.Email,
			"displayName": user.DisplayName,
			"isVerified":  user.IsVerified,
		},
	})
}

type permissionRequestRequest struct {
	RequestedRoleTemplate string  `json:"requestedRoleTemplateId" validate:"required,uuid"`
	RequestedScopeType    string  `json:"requestedScopeType" validate:"required,oneof=global state ihk"`
	RequestedScopeID      *string `json:"requestedScopeId" validate:"omitempty,max=200"`
	ProofFileName         *string `json:"proofFileName" validate:"omitempty,max=255"`
	ProofMimeType         *string `json:"proofMimeType" validate:"omitempty,max=100"`
	ProofContentBase64    *string `json:"proofContentBase64" validate:"omitempty,max=2000000"`
	ProofNote             *string `json:"proofNote" validate:"omitempty,max=2000"`
}

func (h *Handler) RequestPermissions(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.sessions.UserID(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "message": "Not authenticated"})
		return
	}

	var req permissionRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid JSON"})
		return
	}
	req.RequestedRoleTemplate = strings.TrimSpace(req.RequestedRoleTemplate)
	req.RequestedScopeType = strings.TrimSpace(req.RequestedScopeType)
	req.RequestedScopeID = optionalTrimmedString(req.RequestedScopeID)
	req.ProofFileName = optionalTrimmedString(req.ProofFileName)
	req.ProofMimeType = optionalTrimmedString(req.ProofMimeType)
	req.ProofContentBase64 = optionalTrimmedString(req.ProofContentBase64)
	req.ProofNote = optionalTrimmedString(req.ProofNote)
	if err := h.validate.Struct(req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "Invalid input"})
		return
	}
	if err := validateProofUpload(req.ProofMimeType, req.ProofContentBase64); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
		return
	}
	ipHash := security.Sha256Hex(netx.ClientIP(r) + h.secretSalt)
	userHash := security.Sha256Hex(userID + h.secretSalt)
	if !h.allowRate(w, r, "rl:auth:permission_request:ip:"+ipHash, 24*time.Hour, 5) {
		return
	}
	if !h.allowRate(w, r, "rl:auth:permission_request:user:"+userHash, 24*time.Hour, 3) {
		return
	}
	if req.RequestedScopeType == "global" {
		req.RequestedScopeID = nil
	} else if req.RequestedScopeID == nil || *req.RequestedScopeID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "requestedScopeId is required for state and ihk scopes"})
		return
	}

	if err := h.store.CreatePermissionRequestForUser(r.Context(), userID, req.RequestedRoleTemplate, req.RequestedScopeType, req.RequestedScopeID, req.ProofFileName, req.ProofMimeType, req.ProofContentBase64, req.ProofNote); err != nil {
		if errors.Is(err, ErrPermissionRequestAlreadyPending) {
			httpx.JSON(w, http.StatusConflict, map[string]any{"ok": false, "code": "PERMISSION_REQUEST_ALREADY_PENDING", "message": "Für diese Rolle und diesen Scope liegt bereits eine offene Rechteanfrage vor."})
			return
		}
		if errors.Is(err, ErrPermissionAlreadyGranted) {
			httpx.JSON(w, http.StatusConflict, map[string]any{"ok": false, "code": "PERMISSION_ALREADY_GRANTED", "message": "Du hast diese Rechte im gewählten Scope bereits."})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	httpx.JSON(w, http.StatusAccepted, map[string]any{"ok": true, "message": "Deine Rechteanfrage wurde eingereicht."})
}

func (h *Handler) ListRequestableRoleTemplates(w http.ResponseWriter, r *http.Request) {
	roles, err := h.store.ListRequestableRoleTemplates(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "message": "Server error"})
		return
	}
	items := make([]any, 0, len(roles))
	for _, role := range roles {
		items = append(items, map[string]any{
			"id":          role.ID,
			"name":        role.Name,
			"description": role.Description,
			"allowMask":   role.AllowMask,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
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
	if h.limiter == nil {
		return true
	}
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
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "super_admin" {
		return "super_admin@local.invalid"
	}
	return value
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

func validateProofUpload(mimeType *string, contentBase64 *string) error {
	if contentBase64 == nil || *contentBase64 == "" {
		return nil
	}
	if mimeType == nil || !allowedProofMimeTypes[*mimeType] {
		return errors.New("nachweis-Dateityp ist nicht erlaubt. Erlaubt sind PDF, JPG, PNG und WebP")
	}
	decoded, err := base64.StdEncoding.DecodeString(*contentBase64)
	if err != nil {
		return errors.New("nachweis-Datei ist ungültig kodiert")
	}
	if len(decoded) > maxProofUploadBytes {
		return errors.New("nachweis-Datei darf maximal 2 MB groß sein")
	}
	return nil
}
