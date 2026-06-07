package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Philipp01105/kammer-kompass/backend/internal/config"
	"github.com/google/uuid"
	"github.com/gorilla/securecookie"
	"github.com/redis/go-redis/v9"
)

const sessionTTL = 30 * 24 * time.Hour

// SessionManager stores session tokens in Redis and puts only the encrypted
// token ID in the browser cookie.  Logout or account deactivation can
// immediately invalidate a session by deleting the Redis key — no need to
// wait for cookie expiry.
type SessionManager struct {
	codec      securecookie.Codec
	redis      *redis.Client
	cookieName string
	secure     bool
}

func NewSessionManager(cfg config.SessionConfig, redisClient *redis.Client) (*SessionManager, error) {
	if len(cfg.AuthKey) < 32 {
		return nil, errors.New("SESSION_AUTH_KEY must be at least 32 bytes (after base64 decode)")
	}
	if len(cfg.EncKey) != 0 && len(cfg.EncKey) != 16 && len(cfg.EncKey) != 24 && len(cfg.EncKey) != 32 {
		return nil, errors.New("SESSION_ENC_KEY must decode to exactly 16, 24, or 32 bytes")
	}
	if redisClient == nil {
		return nil, errors.New("redis client must not be nil")
	}

	var codec securecookie.Codec
	if len(cfg.EncKey) > 0 {
		codec = securecookie.New(cfg.AuthKey, cfg.EncKey)
	} else {
		codec = securecookie.New(cfg.AuthKey, nil)
	}

	return &SessionManager{
		codec:      codec,
		redis:      redisClient,
		cookieName: cfg.CookieName,
		secure:     cfg.Secure,
	}, nil
}

// SetUserID creates a new server-side session in Redis and sets the session
// cookie. Any previous session cookie is silently superseded.
func (s *SessionManager) SetUserID(w http.ResponseWriter, r *http.Request, userID string) error {
	sessionID := uuid.NewString()

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.redis.Set(ctx, s.redisKey(sessionID), userID, sessionTTL).Err(); err != nil {
		return err
	}
	if err := s.redis.SAdd(ctx, s.userSessionsKey(userID), sessionID).Err(); err != nil {
		_ = s.redis.Del(ctx, s.redisKey(sessionID)).Err()
		return err
	}
	if err := s.redis.Expire(ctx, s.userSessionsKey(userID), sessionTTL).Err(); err != nil {
		_ = s.redis.Del(ctx, s.redisKey(sessionID)).Err()
		_ = s.redis.SRem(ctx, s.userSessionsKey(userID), sessionID).Err()
		return err
	}

	encoded, err := s.codec.Encode(s.cookieName, sessionID)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
	return nil
}

// UserID reads the session cookie, validates it, and looks up the user ID in
// Redis.  Returns ("", false) if the session is missing, invalid, or expired.
func (s *SessionManager) UserID(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(s.cookieName)
	if err != nil {
		return "", false
	}
	var sessionID string
	if err := s.codec.Decode(s.cookieName, cookie.Value, &sessionID); err != nil {
		return "", false
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	userID, err := s.redis.Get(ctx, s.redisKey(sessionID)).Result()
	if err != nil {
		return "", false
	}
	return userID, userID != ""
}

// Clear deletes the server-side session from Redis and instructs the browser
// to remove the cookie.  Silently ignores missing or invalid cookies.
func (s *SessionManager) Clear(w http.ResponseWriter, r *http.Request) error {
	if cookie, err := r.Cookie(s.cookieName); err == nil {
		var sessionID string
		if s.codec.Decode(s.cookieName, cookie.Value, &sessionID) == nil {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()
			if userID, err := s.redis.Get(ctx, s.redisKey(sessionID)).Result(); err == nil && userID != "" {
				_ = s.redis.SRem(ctx, s.userSessionsKey(userID), sessionID).Err()
			}
			_ = s.redis.Del(ctx, s.redisKey(sessionID)).Err()
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	return nil
}

// ClearUserSessions invalidates all active server-side sessions for a user.
func (s *SessionManager) ClearUserSessions(ctx context.Context, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	setKey := s.userSessionsKey(userID)
	sessionIDs, err := s.redis.SMembers(ctx, setKey).Result()
	if err != nil {
		return err
	}
	if len(sessionIDs) == 0 {
		return s.redis.Del(ctx, setKey).Err()
	}

	keys := make([]string, 0, len(sessionIDs)+1)
	for _, sessionID := range sessionIDs {
		keys = append(keys, s.redisKey(sessionID))
	}
	keys = append(keys, setKey)
	return s.redis.Del(ctx, keys...).Err()
}

func (s *SessionManager) redisKey(sessionID string) string {
	return "sess:" + sessionID
}

func (s *SessionManager) userSessionsKey(userID string) string {
	return "user_sessions:" + userID
}
