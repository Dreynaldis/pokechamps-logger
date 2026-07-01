package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/dreynaldis/pokechamps-logger/internal/config"
)

type contextKey string

// ContextKeyUserID is the key used to store the authenticated user's ID in the request context.
const ContextKeyUserID contextKey = "userID"

// Middleware validates the Bearer token and injects the user ID into the request context.
// Returns 401 if the token is missing, malformed, or expired.
func Middleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, `{"error":"missing or invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			claims, err := ParseAccessToken(strings.TrimPrefix(header, "Bearer "), cfg)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
