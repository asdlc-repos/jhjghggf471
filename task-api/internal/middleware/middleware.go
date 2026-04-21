package middleware

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/asdlc/task-api/internal/httputil"
	"github.com/asdlc/task-api/internal/store"
)

type ctxKey int

const (
	userIDKey ctxKey = iota
)

var stderr = log.New(os.Stderr, "[auth] ", log.LstdFlags)

// CORS returns a middleware that adds CORS headers for the given origin.
func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if allowedOrigin == "*" || allowedOrigin == "" {
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				}
			} else if origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "600")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AuthRequired reads the session cookie, validates it, and attaches userId to the request context.
func AuthRequired(st store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil || cookie.Value == "" {
				stderr.Printf("unauthorized: no session cookie from %s %s", r.Method, r.URL.Path)
				httputil.WriteError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			sess, err := st.GetSession(cookie.Value)
			if err != nil {
				stderr.Printf("unauthorized: invalid session from %s %s", r.Method, r.URL.Path)
				httputil.WriteError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, sess.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserID extracts the authenticated user id from the request context.
func UserID(r *http.Request) string {
	if v, ok := r.Context().Value(userIDKey).(string); ok {
		return v
	}
	return ""
}
