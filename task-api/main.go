package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/asdlc/task-api/internal/handlers"
	"github.com/asdlc/task-api/internal/httputil"
	"github.com/asdlc/task-api/internal/middleware"
	"github.com/asdlc/task-api/internal/store"
)

const defaultPort = "9090"

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func main() {
	port := envOr("PORT", defaultPort)
	frontendOrigin := envOr("FRONTEND_ORIGIN", "*")
	secureCookies := envOr("SECURE_COOKIES", "false") == "true"

	st := store.NewMemoryStore()
	authH := handlers.NewAuthHandler(st, secureCookies)
	taskH := handlers.NewTaskHandler(st)
	catH := handlers.NewCategoryHandler(st)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Auth routes (public)
	mux.HandleFunc("/auth/register", methodOnly(http.MethodPost, authH.Register))
	mux.HandleFunc("/auth/login", methodOnly(http.MethodPost, authH.Login))
	mux.HandleFunc("/auth/logout", methodOnly(http.MethodPost, authH.Logout))
	mux.HandleFunc("/auth/password-reset", methodOnly(http.MethodPost, authH.PasswordReset))
	mux.HandleFunc("/auth/password-reset/confirm", methodOnly(http.MethodPost, authH.PasswordResetConfirm))
	mux.Handle("/auth/me", middleware.AuthRequired(st)(http.HandlerFunc(methodOnly(http.MethodGet, authH.Me))))

	// Protected routes
	authRequired := middleware.AuthRequired(st)

	mux.Handle("/tasks", authRequired(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			taskH.List(w, r)
		case http.MethodPost:
			taskH.Create(w, r)
		default:
			httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})))

	mux.Handle("/tasks/", authRequired(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/tasks/")
		parts := strings.Split(path, "/")
		if len(parts) == 0 || parts[0] == "" {
			httputil.WriteError(w, http.StatusNotFound, "not found")
			return
		}
		id := parts[0]
		if len(parts) == 1 {
			switch r.Method {
			case http.MethodPut:
				taskH.Update(w, r, id)
			case http.MethodDelete:
				taskH.Delete(w, r, id)
			default:
				httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		}
		if len(parts) == 2 && r.Method == http.MethodPost {
			switch parts[1] {
			case "complete":
				taskH.Complete(w, r, id)
				return
			case "incomplete":
				taskH.Incomplete(w, r, id)
				return
			}
		}
		httputil.WriteError(w, http.StatusNotFound, "not found")
	})))

	mux.Handle("/categories", authRequired(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			catH.List(w, r)
		case http.MethodPost:
			catH.Create(w, r)
		default:
			httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})))

	mux.Handle("/categories/", authRequired(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/categories/")
		parts := strings.Split(path, "/")
		if len(parts) != 1 || parts[0] == "" {
			httputil.WriteError(w, http.StatusNotFound, "not found")
			return
		}
		id := parts[0]
		switch r.Method {
		case http.MethodPut:
			catH.Update(w, r, id)
		case http.MethodDelete:
			catH.Delete(w, r, id)
		default:
			httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})))

	handler := middleware.CORS(frontendOrigin)(mux)

	addr := ":" + port
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("task-api listening on %s (frontend_origin=%s secure_cookies=%v)", addr, frontendOrigin, secureCookies)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutdown signal received")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func methodOnly(method string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h(w, r)
	}
}
