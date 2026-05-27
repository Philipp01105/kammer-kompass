package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/Philipp01105/kammer-kompass/backend/internal/config"
	"github.com/gorilla/sessions"
)

const sessionUserIDKey = "user_id"

type SessionManager struct {
	store      *sessions.CookieStore
	cookieName string
}

func NewSessionManager(cfg config.SessionConfig) (*SessionManager, error) {
	if len(cfg.AuthKey) < 32 {
		return nil, errors.New("SESSION_AUTH_KEY must be at least 32 bytes (after base64 decode)")
	}
	if len(cfg.EncKey) != 0 && len(cfg.EncKey) != 16 && len(cfg.EncKey) != 24 && len(cfg.EncKey) != 32 {
		return nil, errors.New("SESSION_ENC_KEY must decode to exactly 16, 24, or 32 bytes")
	}

	store := sessions.NewCookieStore(cfg.AuthKey, cfg.EncKey)
	store.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.Secure,
		MaxAge:   int((30 * 24 * time.Hour).Seconds()),
	}

	return &SessionManager{
		store:      store,
		cookieName: cfg.CookieName,
	}, nil
}

func (s *SessionManager) Get(r *http.Request) (*sessions.Session, error) {
	return s.store.Get(r, s.cookieName)
}

// SetUserID sets the user id of the user of the session
func (s *SessionManager) SetUserID(w http.ResponseWriter, r *http.Request, userID string) error {
	sess, err := s.Get(r)
	if err != nil {
		return err
	}
	sess.Values[sessionUserIDKey] = userID
	return sess.Save(r, w)
}

// Clear removes the session cookie
func (s *SessionManager) Clear(w http.ResponseWriter, r *http.Request) error {
	sess, err := s.Get(r)
	if err != nil {
		return err
	}
	sess.Options.MaxAge = -1
	return sess.Save(r, w)
}

// UserID returns the user id of the user of the session
func (s *SessionManager) UserID(r *http.Request) (string, bool) {
	sess, err := s.Get(r)
	if err != nil {
		return "", false
	}
	raw, ok := sess.Values[sessionUserIDKey]
	if !ok {
		return "", false
	}
	id, ok := raw.(string)
	return id, ok && id != ""
}
