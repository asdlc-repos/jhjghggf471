package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/asdlc/task-api/internal/httputil"
	"github.com/asdlc/task-api/internal/models"
	"github.com/asdlc/task-api/internal/store"
	"github.com/asdlc/task-api/internal/validation"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	maxLoginAttempts = 3
	lockoutDuration  = 15 * time.Minute
	sessionDuration  = 24 * time.Hour
	resetTokenTTL    = 1 * time.Hour
	bcryptCost       = 10
)

type AuthHandler struct {
	Store      store.Store
	CookieName string
	Secure     bool
}

func NewAuthHandler(st store.Store, secure bool) *AuthHandler {
	return &AuthHandler{Store: st, CookieName: "session", Secure: secure}
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	sameSite := http.SameSiteLaxMode
	if h.Secure {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     h.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(sessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   h.Secure,
		SameSite: sameSite,
	})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	sameSite := http.SameSiteLaxMode
	if h.Secure {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     h.CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.Secure,
		SameSite: sameSite,
	})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Email = store.NormalizeEmail(req.Email)
	if err := validation.ValidateEmail(req.Email); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validation.ValidatePassword(req.Password); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	user := &models.User{
		ID:           uuid.NewString(),
		Email:        req.Email,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}
	if err := h.Store.CreateUser(user); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			httputil.WriteError(w, http.StatusBadRequest, "email already registered")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create user")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, userResponse{ID: user.ID, Email: user.Email})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	email := store.NormalizeEmail(req.Email)
	if email == "" || req.Password == "" {
		httputil.WriteError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	attempt := h.Store.GetLoginAttempt(email)
	if attempt != nil && time.Now().Before(attempt.LockedUntil) {
		w.Header().Set("Retry-After", time.Until(attempt.LockedUntil).Round(time.Second).String())
		httputil.WriteError(w, http.StatusLocked, "account locked due to too many failed login attempts")
		return
	}

	user, err := h.Store.GetUserByEmail(email)
	if err != nil || bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(req.Password)) != nil {
		h.recordFailedAttempt(email, attempt)
		httputil.WriteError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	h.Store.ClearLoginAttempt(email)

	token, err := randomToken()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	expires := time.Now().Add(sessionDuration)
	if err := h.Store.CreateSession(&models.Session{Token: token, UserID: user.ID, ExpiresAt: expires}); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	h.setSessionCookie(w, token, expires)
	httputil.WriteJSON(w, http.StatusOK, userResponse{ID: user.ID, Email: user.Email})
}

func (h *AuthHandler) recordFailedAttempt(email string, prev *models.LoginAttempt) {
	a := &models.LoginAttempt{}
	if prev != nil {
		a.Count = prev.Count
		a.LockedUntil = prev.LockedUntil
	}
	a.Count++
	if a.Count >= maxLoginAttempts {
		a.LockedUntil = time.Now().Add(lockoutDuration)
	}
	h.Store.SetLoginAttempt(email, a)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(h.CookieName); err == nil {
		_ = h.Store.DeleteSession(cookie.Value)
	}
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.CookieName)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	sess, err := h.Store.GetSession(cookie.Value)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	user, err := h.Store.GetUserByID(sess.UserID)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, userResponse{ID: user.ID, Email: user.Email})
}

type passwordResetRequest struct {
	Email string `json:"email"`
}

func (h *AuthHandler) PasswordReset(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	email := store.NormalizeEmail(req.Email)
	response := map[string]string{"message": "if an account exists for that email, a reset link has been sent"}
	if email == "" {
		httputil.WriteJSON(w, http.StatusOK, response)
		return
	}
	user, err := h.Store.GetUserByEmail(email)
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, response)
		return
	}
	token, err := randomToken()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create reset token")
		return
	}
	_ = h.Store.CreateResetToken(&models.PasswordResetToken{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(resetTokenTTL),
	})
	log.Printf("[password-reset] token issued for %s: %s", email, token)
	httputil.WriteJSON(w, http.StatusOK, response)
}

type passwordResetConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

func (h *AuthHandler) PasswordResetConfirm(w http.ResponseWriter, r *http.Request) {
	var req passwordResetConfirmRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Token == "" {
		httputil.WriteError(w, http.StatusBadRequest, "token is required")
		return
	}
	if err := validation.ValidatePassword(req.NewPassword); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := h.Store.GetResetToken(req.Token)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid or expired token")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcryptCost)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	if err := h.Store.UpdateUserPassword(t.UserID, hash); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update password")
		return
	}
	h.Store.DeleteResetToken(req.Token)
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}
