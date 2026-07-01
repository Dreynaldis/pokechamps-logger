package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/dreynaldis/pokechamps-logger/internal/auth"
	"github.com/dreynaldis/pokechamps-logger/internal/model"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var validate = validator.New()

// Register handles POST /auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"    validate:"required,email"`
		Password string `json:"password" validate:"required,min=8"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := validate.Struct(body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 12)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	user := model.User{
		Email:        strings.ToLower(body.Email),
		PasswordHash: string(hash),
	}
	if err := h.DB.Create(&user).Error; err != nil {
		// unique constraint on email
		writeError(w, http.StatusConflict, "email already registered")
		return
	}

	accessToken, err := auth.IssueTokens(w, h.DB, h.Config, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue tokens")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"access_token": accessToken,
		"user": map[string]any{
			"id":    user.ID,
			"email": user.Email,
		},
	})
}

// Login handles POST /auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"    validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := validate.Struct(body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	var user model.User
	if err := h.DB.Where("email = ?", strings.ToLower(body.Email)).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// same response as wrong password -- don't leak whether email exists
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if user.PasswordHash == "" {
		// OAuth-only account -- no password set
		writeError(w, http.StatusUnauthorized, "account uses social login")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	accessToken, err := auth.IssueTokens(w, h.DB, h.Config, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue tokens")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": accessToken,
		"user": map[string]any{
			"id":    user.ID,
			"email": user.Email,
		},
	})
}

// Refresh handles POST /auth/refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	raw, err := auth.ReadRefreshCookie(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	// Find all non-expired tokens and bcrypt-compare -- there's typically only 1 per device
	var candidates []model.RefreshToken
	h.DB.Where("expires_at > ?", time.Now()).Find(&candidates)

	var matched *model.RefreshToken
	for i := range candidates {
		if bcrypt.CompareHashAndPassword([]byte(candidates[i].TokenHash), []byte(raw)) == nil {
			matched = &candidates[i]
			break
		}
	}
	if matched == nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	// Rotate -- delete old, issue new
	h.DB.Delete(matched)

	accessToken, err := auth.IssueTokens(w, h.DB, h.Config, matched.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue tokens")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"access_token": accessToken})
}

// Logout handles POST /auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	raw, err := auth.ReadRefreshCookie(r)
	if err != nil {
		// No cookie -- already logged out
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var candidates []model.RefreshToken
	h.DB.Where("expires_at > ?", time.Now()).Find(&candidates)

	for i := range candidates {
		if bcrypt.CompareHashAndPassword([]byte(candidates[i].TokenHash), []byte(raw)) == nil {
			h.DB.Delete(&candidates[i])
			break
		}
	}

	auth.ClearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// Me handles GET /auth/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(auth.ContextKeyUserID).(string)

	var user model.User
	if err := h.DB.First(&user, "id = ?", userID).Error; err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    user.ID,
		"email": user.Email,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
