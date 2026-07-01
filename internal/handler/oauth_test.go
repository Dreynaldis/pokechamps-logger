package handler_test

import (
	"testing"

	"github.com/dreynaldis/pokechamps-logger/internal/config"
	"github.com/dreynaldis/pokechamps-logger/internal/handler"
	"github.com/dreynaldis/pokechamps-logger/internal/model"
	"github.com/dreynaldis/pokechamps-logger/internal/testutil"
	"github.com/markbates/goth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newOAuthHandler(t *testing.T) *handler.Handler {
	t.Helper()
	return &handler.Handler{
		DB:     testutil.SetupDB(t),
		Config: &config.Config{AuthSecret: "test-secret-that-is-long-enough-32b"},
	}
}

func fakeGothUser(provider, providerID, email string) goth.User {
	return goth.User{
		Provider: provider,
		UserID:   providerID,
		Email:    email,
	}
}

// UpsertOAuthUser is exported for testing. See handler/oauth.go.
// We call it via the exported method on Handler.

func TestUpsertOAuthUser(t *testing.T) {
	t.Run("new user creates User and OAuthAccount rows", func(t *testing.T) {
		h := newOAuthHandler(t)
		gothUser := fakeGothUser("google", "google-uid-001", "newuser@example.com")

		userID, err := h.UpsertOAuthUser(gothUser)
		require.NoError(t, err)
		assert.NotEmpty(t, userID)

		// User row should exist
		var user model.User
		require.NoError(t, h.DB.First(&user, "id = ?", userID).Error)
		assert.Equal(t, "newuser@example.com", user.Email)
		assert.Empty(t, user.PasswordHash) // OAuth-only: no password

		// OAuthAccount row should be linked
		var oa model.OAuthAccount
		require.NoError(t, h.DB.Where("provider = ? AND provider_account_id = ?", "google", "google-uid-001").First(&oa).Error)
		assert.Equal(t, userID, oa.UserID)
	})

	t.Run("returning user via same provider returns same userID", func(t *testing.T) {
		h := newOAuthHandler(t)
		gothUser := fakeGothUser("google", "google-uid-002", "returning@example.com")

		first, err := h.UpsertOAuthUser(gothUser)
		require.NoError(t, err)

		second, err := h.UpsertOAuthUser(gothUser)
		require.NoError(t, err)

		assert.Equal(t, first, second)

		// Should still be exactly one OAuthAccount row
		var count int64
		h.DB.Model(&model.OAuthAccount{}).
			Where("provider = ? AND provider_account_id = ?", "google", "google-uid-002").
			Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("OAuth user with existing email/password account gets linked, no new User created", func(t *testing.T) {
		h := newOAuthHandler(t)

		// Register via email/password first
		regW := postJSON(t, h.Register, map[string]string{
			"email": "linked@example.com", "password": "password123",
		})
		require.Equal(t, 201, regW.Code)

		// Now OAuth login with same email
		gothUser := fakeGothUser("google", "google-uid-003", "linked@example.com")
		userID, err := h.UpsertOAuthUser(gothUser)
		require.NoError(t, err)

		// Should be the same user, not a new one
		var count int64
		h.DB.Model(&model.User{}).Where("email = ?", "linked@example.com").Count(&count)
		assert.Equal(t, int64(1), count)

		// OAuthAccount should point at that existing user
		var oa model.OAuthAccount
		require.NoError(t, h.DB.Where("provider = ? AND provider_account_id = ?", "google", "google-uid-003").First(&oa).Error)
		assert.Equal(t, userID, oa.UserID)

		// Original user still has a password hash (not wiped)
		var user model.User
		require.NoError(t, h.DB.First(&user, "id = ?", userID).Error)
		assert.NotEmpty(t, user.PasswordHash)
	})

	t.Run("two different providers with same email link to the same User", func(t *testing.T) {
		h := newOAuthHandler(t)

		googleUser := fakeGothUser("google", "google-uid-004", "multiauth@example.com")
		discordUser := fakeGothUser("discord", "discord-uid-004", "multiauth@example.com")

		googleUserID, err := h.UpsertOAuthUser(googleUser)
		require.NoError(t, err)

		discordUserID, err := h.UpsertOAuthUser(discordUser)
		require.NoError(t, err)

		assert.Equal(t, googleUserID, discordUserID)

		// Two OAuthAccount rows, one User row
		var oaCount int64
		h.DB.Model(&model.OAuthAccount{}).Where("user_id = ?", googleUserID).Count(&oaCount)
		assert.Equal(t, int64(2), oaCount)

		var userCount int64
		h.DB.Model(&model.User{}).Where("email = ?", "multiauth@example.com").Count(&userCount)
		assert.Equal(t, int64(1), userCount)
	})
}
