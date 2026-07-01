package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dreynaldis/pokechamps-logger/internal/auth"
	"github.com/dreynaldis/pokechamps-logger/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"gorm.io/gorm"
)

// BeginOAuth redirects the browser to the provider's consent screen.
// Route: GET /auth/{provider}
func (h *Handler) BeginOAuth(w http.ResponseWriter, r *http.Request) {
	r = withProvider(r)
	gothic.BeginAuthHandler(w, r)
}

// OAuthCallback completes the OAuth flow, upserts user/oauth_account rows,
// issues tokens, and redirects to the frontend with the access token in the
// URL hash so it never appears in server logs.
// Route: GET /auth/{provider}/callback
func (h *Handler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	r = withProvider(r)

	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		http.Redirect(w, r, h.Config.FrontendOrigin+"/?error=oauth_failed", http.StatusFound)
		return
	}

	userID, err := h.UpsertOAuthUser(gothUser)
	if err != nil {
		http.Redirect(w, r, h.Config.FrontendOrigin+"/?error=oauth_failed", http.StatusFound)
		return
	}

	accessToken, err := auth.IssueTokens(w, h.DB, h.Config, userID)
	if err != nil {
		http.Redirect(w, r, h.Config.FrontendOrigin+"/?error=token_failed", http.StatusFound)
		return
	}

	http.Redirect(w, r, h.Config.FrontendOrigin+"/auth/callback#access_token="+accessToken, http.StatusFound)
}

// upsertOAuthUser finds or creates the User and OAuthAccount rows in a
// single transaction. If the oauth_accounts row already exists, just returns
// the associated user ID. If the email is already registered (email/password
// account), links it rather than creating a duplicate user.
func (h *Handler) UpsertOAuthUser(gothUser goth.User) (string, error) {
	var userID string

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Check for existing oauth_accounts row (returning user via same provider)
		var existing model.OAuthAccount
		err := tx.Where("provider = ? AND provider_account_id = ?",
			gothUser.Provider, gothUser.UserID).First(&existing).Error

		if err == nil {
			userID = existing.UserID
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 2. No oauth_accounts row -- find or create the User
		var user model.User
		err = tx.Where("email = ?", strings.ToLower(gothUser.Email)).First(&user).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user = model.User{Email: strings.ToLower(gothUser.Email)}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		// 3. Link this provider to the user
		oauthAccount := model.OAuthAccount{
			UserID:            user.ID,
			Provider:          gothUser.Provider,
			ProviderAccountID: gothUser.UserID,
		}
		if err := tx.Create(&oauthAccount).Error; err != nil {
			return err
		}

		userID = user.ID
		return nil
	})

	return userID, err
}

// withProvider copies the {provider} URL param into the query string so
// gothic can find it via its default GetProviderName implementation.
func withProvider(r *http.Request) *http.Request {
	provider := chi.URLParam(r, "provider")
	q := r.URL.Query()
	q.Set("provider", provider)
	r.URL.RawQuery = q.Encode()
	return r
}
