package middleware

import (
	"context"
	"net/http"

	"github.com/Philipp01105/kammer-kompass/backend/internal/auth"
)

type ctxKey int

const userIDKey ctxKey = 1

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func UserID(r *http.Request) (string, bool) {
	v := r.Context().Value(userIDKey)
	id, ok := v.(string)
	return id, ok && id != ""
}

// OptionalAuth middleware sets the user ID in the context if the user is authenticated.
func OptionalAuth(sm *auth.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if id, ok := sm.UserID(r); ok {
				r = r.WithContext(WithUserID(r.Context(), id))
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// RequireAuth middleware requires the user to be authenticated.
func RequireAuth(sm *auth.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			id, ok := sm.UserID(r)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), id)))
		}
		return http.HandlerFunc(fn)
	}
}
