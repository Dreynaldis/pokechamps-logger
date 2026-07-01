package handler_test

import (
	"bytes"
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

func newTestHandler(t *testing.T) *handler.Handler {
	t.Helper()
	return &handler.Handler{
		DB:     testutil.SetupDB(t),
		Config: &config.Config{AuthSecret: "test-secret-that-is-long-enough-32b"},
	}
}

func postJSON(t *testing.T, fn http.HandlerFunc, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	fn(w, r)
	return w
}

func refreshCookieFrom(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == "refresh_token" {
			return c
		}
	}
	t.Fatal("no refresh_token cookie in response")
	return nil
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

func TestRegister(t *testing.T) {
	h := newTestHandler(t)

	t.Run("success returns 201 with access token", func(t *testing.T) {
		w := postJSON(t, h.Register, map[string]string{"email": "reg@example.com", "password": "password123"})
		require.Equal(t, http.StatusCreated, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEmpty(t, resp["access_token"])
		assert.NotNil(t, refreshCookieFrom(t, w))
	})

	t.Run("duplicate email returns 409", func(t *testing.T) {
		postJSON(t, h.Register, map[string]string{"email": "dup@example.com", "password": "password123"})
		w := postJSON(t, h.Register, map[string]string{"email": "dup@example.com", "password": "password123"})
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("invalid email returns 422", func(t *testing.T) {
		w := postJSON(t, h.Register, map[string]string{"email": "notanemail", "password": "password123"})
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("password too short returns 422", func(t *testing.T) {
		w := postJSON(t, h.Register, map[string]string{"email": "short@example.com", "password": "abc"})
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestLogin(t *testing.T) {
	h := newTestHandler(t)
	postJSON(t, h.Register, map[string]string{"email": "login@example.com", "password": "password123"})

	t.Run("correct credentials returns 200 with tokens", func(t *testing.T) {
		w := postJSON(t, h.Login, map[string]string{"email": "login@example.com", "password": "password123"})
		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEmpty(t, resp["access_token"])
		assert.NotNil(t, refreshCookieFrom(t, w))
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		w := postJSON(t, h.Login, map[string]string{"email": "login@example.com", "password": "wrongpassword"})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("unknown email returns 401 (same as wrong password -- no enumeration)", func(t *testing.T) {
		w := postJSON(t, h.Login, map[string]string{"email": "nobody@example.com", "password": "password123"})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// ---------------------------------------------------------------------------
// Refresh
// ---------------------------------------------------------------------------

func TestRefresh(t *testing.T) {
	h := newTestHandler(t)
	regW := postJSON(t, h.Register, map[string]string{"email": "refresh@example.com", "password": "password123"})
	require.Equal(t, http.StatusCreated, regW.Code)
	cookie := refreshCookieFrom(t, regW)

	t.Run("valid cookie returns 200 with new access token", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		h.Refresh(w, r)
		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEmpty(t, resp["access_token"])
	})

	t.Run("missing cookie returns 401", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		w := httptest.NewRecorder()
		h.Refresh(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid cookie value returns 401", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		r.AddCookie(&http.Cookie{Name: "refresh_token", Value: "totallyfakevalue"})
		w := httptest.NewRecorder()
		h.Refresh(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// ---------------------------------------------------------------------------
// Logout
// ---------------------------------------------------------------------------

func TestLogout(t *testing.T) {
	h := newTestHandler(t)
	regW := postJSON(t, h.Register, map[string]string{"email": "logout@example.com", "password": "password123"})
	require.Equal(t, http.StatusCreated, regW.Code)
	cookie := refreshCookieFrom(t, regW)

	t.Run("with cookie returns 204 and clears cookie", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		h.Logout(w, r)
		assert.Equal(t, http.StatusNoContent, w.Code)
		// cleared cookie should have MaxAge -1
		var cleared *http.Cookie
		for _, c := range w.Result().Cookies() {
			if c.Name == "refresh_token" {
				cleared = c
			}
		}
		require.NotNil(t, cleared)
		assert.Equal(t, -1, cleared.MaxAge)
	})

	t.Run("without cookie returns 204 (already logged out)", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		w := httptest.NewRecorder()
		h.Logout(w, r)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

// ---------------------------------------------------------------------------
// Me
// ---------------------------------------------------------------------------

func TestMe(t *testing.T) {
	h := newTestHandler(t)
	regW := postJSON(t, h.Register, map[string]string{"email": "me@example.com", "password": "password123"})
	require.Equal(t, http.StatusCreated, regW.Code)
	var regResp map[string]any
	require.NoError(t, json.Unmarshal(regW.Body.Bytes(), &regResp))
	userID := regResp["user"].(map[string]any)["id"].(string)

	t.Run("with user ID in context returns 200", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		r = r.WithContext(context.WithValue(r.Context(), auth.ContextKeyUserID, userID))
		w := httptest.NewRecorder()
		h.Me(w, r)
		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "me@example.com", resp["email"])
		assert.Equal(t, userID, resp["id"])
	})
}

// ---------------------------------------------------------------------------
// JWT Middleware
// ---------------------------------------------------------------------------

func TestMiddleware(t *testing.T) {
	cfg := &config.Config{AuthSecret: "test-secret-that-is-long-enough-32b"}
	mw := auth.Middleware(cfg)

	sentinel := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := r.Context().Value(auth.ContextKeyUserID).(string)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(uid))
	})
	protected := mw(sentinel)

	// Issue a real token using the same config
	h := &handler.Handler{
		DB:     testutil.SetupDB(t),
		Config: cfg,
	}
	regW := postJSON(t, h.Register, map[string]string{"email": "mw@example.com", "password": "password123"})
	require.Equal(t, http.StatusCreated, regW.Code)
	var regResp map[string]any
	require.NoError(t, json.Unmarshal(regW.Body.Bytes(), &regResp))
	token := regResp["access_token"].(string)

	t.Run("valid Bearer token passes through", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("missing header returns 401", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("malformed token returns 401", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer notavalidjwt")
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("wrong secret returns 401", func(t *testing.T) {
		// token signed with a different secret
		badCfg := &config.Config{AuthSecret: "completely-different-secret-value"}
		badMw := auth.Middleware(badCfg)
		protected2 := badMw(sentinel)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		protected2.ServeHTTP(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
