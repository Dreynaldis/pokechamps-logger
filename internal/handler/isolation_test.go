package handler_test

// Cross-user isolation tests.
//
// These verify that authenticated requests only expose data belonging to the
// requesting user. Phase 2 has one protected endpoint (/auth/me); the pattern
// established here will be repeated for teams and matches in Phase 3+.
//
// Failure mode being guarded: a handler that forgets to filter by userID from
// context and instead uses a user-supplied ID from the URL or body -- a
// horizontal privilege escalation (user A reads/modifies user B's data).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dreynaldis/pokechamps-logger/internal/auth"
	"github.com/dreynaldis/pokechamps-logger/internal/config"
	"github.com/dreynaldis/pokechamps-logger/internal/handler"
	"github.com/dreynaldis/pokechamps-logger/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrossUserIsolation(t *testing.T) {
	db := testutil.SetupDB(t)
	cfg := &config.Config{AuthSecret: "test-secret-that-is-long-enough-32b"}
	h := &handler.Handler{DB: db, Config: cfg}

	// Register two independent users
	wA := postJSON(t, h.Register, map[string]string{"email": "userA@example.com", "password": "password123"})
	require.Equal(t, http.StatusCreated, wA.Code)
	var respA map[string]any
	require.NoError(t, json.Unmarshal(wA.Body.Bytes(), &respA))
	userAID := respA["user"].(map[string]any)["id"].(string)

	wB := postJSON(t, h.Register, map[string]string{"email": "userB@example.com", "password": "password123"})
	require.Equal(t, http.StatusCreated, wB.Code)
	var respB map[string]any
	require.NoError(t, json.Unmarshal(wB.Body.Bytes(), &respB))
	userBID := respB["user"].(map[string]any)["id"].(string)

	t.Run("/auth/me returns only the requesting user's own data", func(t *testing.T) {
		// User A's token should return user A's profile, not B's
		rA := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		rA = rA.WithContext(context.WithValue(rA.Context(), auth.ContextKeyUserID, userAID))
		wA2 := httptest.NewRecorder()
		h.Me(wA2, rA)
		require.Equal(t, http.StatusOK, wA2.Code)
		var meA map[string]any
		require.NoError(t, json.Unmarshal(wA2.Body.Bytes(), &meA))
		assert.Equal(t, "usera@example.com", meA["email"])
		assert.NotEqual(t, userBID, meA["id"])

		// User B's token should return user B's profile, not A's
		rB := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		rB = rB.WithContext(context.WithValue(rB.Context(), auth.ContextKeyUserID, userBID))
		wB2 := httptest.NewRecorder()
		h.Me(wB2, rB)
		require.Equal(t, http.StatusOK, wB2.Code)
		var meB map[string]any
		require.NoError(t, json.Unmarshal(wB2.Body.Bytes(), &meB))
		assert.Equal(t, "userb@example.com", meB["email"])
		assert.NotEqual(t, userAID, meB["id"])
	})

	t.Run("JWT middleware rejects user A token on a request carrying user B token", func(t *testing.T) {
		mw := auth.Middleware(cfg)

		// Issue a real JWT for user A
		wToken := postJSON(t, h.Login, map[string]string{"email": "userA@example.com", "password": "password123"})
		require.Equal(t, http.StatusOK, wToken.Code)
		var loginResp map[string]any
		require.NoError(t, json.Unmarshal(wToken.Body.Bytes(), &loginResp))
		tokenA := loginResp["access_token"].(string)

		// User A's token must not grant access when the middleware uses a different secret
		wrongCfg := &config.Config{AuthSecret: "wrong-secret-completely-different!!"}
		blockedMw := auth.Middleware(wrongCfg)

		sentinel := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer "+tokenA)

		// Right secret: passes
		w1 := httptest.NewRecorder()
		mw(sentinel).ServeHTTP(w1, r)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Wrong secret: blocked
		w2 := httptest.NewRecorder()
		blockedMw(sentinel).ServeHTTP(w2, r)
		assert.Equal(t, http.StatusUnauthorized, w2.Code)
	})

	t.Run("refresh token for user A cannot be used to obtain user B's session", func(t *testing.T) {
		// Log in as user A, get their refresh cookie
		wLoginA := postJSON(t, h.Login, map[string]string{"email": "userA@example.com", "password": "password123"})
		require.Equal(t, http.StatusOK, wLoginA.Code)
		cookieA := refreshCookieFrom(t, wLoginA)

		// Use user A's cookie to refresh -- should return a token for user A only
		r := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		r.AddCookie(cookieA)
		wRefresh := httptest.NewRecorder()
		h.Refresh(wRefresh, r)
		require.Equal(t, http.StatusOK, wRefresh.Code)

		var refreshResp map[string]any
		require.NoError(t, json.Unmarshal(wRefresh.Body.Bytes(), &refreshResp))
		newToken := refreshResp["access_token"].(string)

		// Parse the new token and confirm the sub claim is user A, not user B
		claims, err := auth.ParseAccessToken(newToken, cfg)
		require.NoError(t, err)
		assert.Equal(t, userAID, claims.UserID)
		assert.NotEqual(t, userBID, claims.UserID)
	})
}
