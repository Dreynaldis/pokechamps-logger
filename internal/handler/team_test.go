package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dreynaldis/pokechamps-logger/internal/auth"
	"github.com/dreynaldis/pokechamps-logger/internal/config"
	"github.com/dreynaldis/pokechamps-logger/internal/handler"
	"github.com/dreynaldis/pokechamps-logger/internal/model"
	"github.com/dreynaldis/pokechamps-logger/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// withUser injects an authenticated user ID into the request context.
func withUser(r *http.Request, userID string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), auth.ContextKeyUserID, userID))
}

// withID injects a Chi URL param "id" so handlers can call chi.URLParam(r, "id").
func withID(r *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// teamRequest combines withUser + withID and issues the request to fn.
func teamRequest(t *testing.T, fn http.HandlerFunc, method, teamID, userID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, "/", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, "/", nil)
	}
	req = withUser(withID(req, teamID), userID)
	w := httptest.NewRecorder()
	fn(w, req)
	return w
}

// seed fixtures grabbed from committed seed data, visible inside the test tx.
type fixtures struct {
	pokemonID string
	abilityID string
	moveID    string
}

func getFixtures(t *testing.T, h *handler.Handler) fixtures {
	t.Helper()
	var p model.Pokemon
	require.NoError(t, h.DB.First(&p).Error, "no pokemon in DB -- run seed first")
	var a model.Ability
	require.NoError(t, h.DB.First(&a).Error)
	var m model.Move
	require.NoError(t, h.DB.First(&m).Error)
	return fixtures{pokemonID: p.ID, abilityID: a.ID, moveID: m.ID}
}

func registerUser(t *testing.T, h *handler.Handler, email string) string {
	t.Helper()
	w := postJSON(t, h.Register, map[string]string{"email": email, "password": "password123"})
	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp["user"].(map[string]any)["id"].(string)
}

func newTeamHandler(t *testing.T) *handler.Handler {
	t.Helper()
	return &handler.Handler{
		DB:     testutil.SetupDB(t),
		Config: &config.Config{AuthSecret: "test-secret-that-is-long-enough-32b"},
	}
}

// ---------------------------------------------------------------------------
// CRUD
// ---------------------------------------------------------------------------

func TestTeamCRUD(t *testing.T) {
	h := newTeamHandler(t)
	fx := getFixtures(t, h)
	userID := registerUser(t, h, "teamcrud@example.com")

	// --- Create ---
	t.Run("create returns 201 with id and name", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/teams", jsonBody(`{"name":"Alpha"}`))
		r.Header.Set("Content-Type", "application/json")
		r = withUser(r, userID)
		w := httptest.NewRecorder()
		h.CreateTeam(w, r)
		require.Equal(t, http.StatusCreated, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Alpha", resp["name"])
		assert.Equal(t, false, resp["is_active"])
	})

	// --- List ---
	t.Run("list returns all teams for user, no slots", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/teams", nil)
		r = withUser(r, userID)
		w := httptest.NewRecorder()
		h.ListTeams(w, r)
		require.Equal(t, http.StatusOK, w.Code)
		var resp []map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Len(t, resp, 1)
		assert.Nil(t, resp[0]["slots"]) // slots omitted on list
	})

	// Create a team and get its ID for remaining sub-tests
	teamID := createTeam(t, h, userID, "Beta")

	// --- Get (empty) ---
	t.Run("get returns team with empty slots", func(t *testing.T) {
		w := teamRequest(t, h.GetTeam, http.MethodGet, teamID, userID, nil)
		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, teamID, resp["id"])
		assert.Empty(t, resp["slots"])
	})

	// --- Patch: name only ---
	t.Run("patch name only leaves slots unchanged", func(t *testing.T) {
		w := teamRequest(t, h.PatchTeam, http.MethodPatch, teamID, userID, map[string]any{"name": "Gamma"})
		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Gamma", resp["name"])
		assert.Empty(t, resp["slots"]) // still no slots
	})

	// --- Patch: add slots ---
	t.Run("patch slots replaces team contents", func(t *testing.T) {
		body := map[string]any{
			"slots": []map[string]any{
				{
					"slot":       1,
					"pokemon_id": fx.pokemonID,
					"ability_id": fx.abilityID,
					"nature":     "Jolly",
					"moves":      []map[string]any{{"slot": 1, "move_id": fx.moveID}},
				},
			},
		}
		w := teamRequest(t, h.PatchTeam, http.MethodPatch, teamID, userID, body)
		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		slots := resp["slots"].([]any)
		require.Len(t, slots, 1)
		slot := slots[0].(map[string]any)
		assert.Equal(t, float64(1), slot["slot"])
		assert.Equal(t, "Jolly", slot["nature"])
		assert.NotNil(t, slot["ability"])
		moves := slot["moves"].([]any)
		require.Len(t, moves, 1)
		assert.Equal(t, float64(1), moves[0].(map[string]any)["slot"])
	})

	// --- Patch: empty slots clears ---
	t.Run("patch empty slots clears all slots", func(t *testing.T) {
		w := teamRequest(t, h.PatchTeam, http.MethodPatch, teamID, userID, map[string]any{"slots": []any{}})
		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Empty(t, resp["slots"])
	})

	// --- Delete ---
	t.Run("delete returns 204 and team is gone", func(t *testing.T) {
		w := teamRequest(t, h.DeleteTeam, http.MethodDelete, teamID, userID, nil)
		assert.Equal(t, http.StatusNoContent, w.Code)

		w2 := teamRequest(t, h.GetTeam, http.MethodGet, teamID, userID, nil)
		assert.Equal(t, http.StatusNotFound, w2.Code)
	})
}

// ---------------------------------------------------------------------------
// Activation
// ---------------------------------------------------------------------------

func TestTeamActivation(t *testing.T) {
	h := newTeamHandler(t)
	userID := registerUser(t, h, "activate@example.com")

	teamA := createTeam(t, h, userID, "Team A")
	teamB := createTeam(t, h, userID, "Team B")

	t.Run("activating team A sets only A active", func(t *testing.T) {
		w := teamRequest(t, h.ActivateTeam, http.MethodPost, teamA, userID, nil)
		require.Equal(t, http.StatusNoContent, w.Code)

		teams := listTeams(t, h, userID)
		for _, team := range teams {
			id := team["id"].(string)
			if id == teamA {
				assert.True(t, team["is_active"].(bool), "team A should be active")
			} else {
				assert.False(t, team["is_active"].(bool), "team B should be inactive")
			}
		}
	})

	t.Run("activating team B deactivates A", func(t *testing.T) {
		w := teamRequest(t, h.ActivateTeam, http.MethodPost, teamB, userID, nil)
		require.Equal(t, http.StatusNoContent, w.Code)

		teams := listTeams(t, h, userID)
		activeCount := 0
		for _, team := range teams {
			if team["is_active"].(bool) {
				activeCount++
				assert.Equal(t, teamB, team["id"].(string))
			}
		}
		assert.Equal(t, 1, activeCount, "exactly one team should be active")
	})
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func TestTeamValidation(t *testing.T) {
	h := newTeamHandler(t)
	fx := getFixtures(t, h)
	userID := registerUser(t, h, "teamval@example.com")

	t.Run("empty name returns 422", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", jsonBody(`{"name":""}`))
		r.Header.Set("Content-Type", "application/json")
		r = withUser(r, userID)
		w := httptest.NewRecorder()
		h.CreateTeam(w, r)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("name over 50 chars returns 422", func(t *testing.T) {
		longName := fmt.Sprintf("%051d", 0) // 51-char numeric string
		r := httptest.NewRequest(http.MethodPost, "/", jsonBody(fmt.Sprintf(`{"name":%q}`, longName)))
		r.Header.Set("Content-Type", "application/json")
		r = withUser(r, userID)
		w := httptest.NewRecorder()
		h.CreateTeam(w, r)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	teamID := createTeam(t, h, userID, "Validation Team")

	t.Run("slot number out of range returns 422", func(t *testing.T) {
		body := map[string]any{"slots": []map[string]any{{"slot": 7, "pokemon_id": fx.pokemonID}}}
		w := teamRequest(t, h.PatchTeam, http.MethodPatch, teamID, userID, body)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("duplicate slot numbers returns 422", func(t *testing.T) {
		body := map[string]any{"slots": []map[string]any{
			{"slot": 1, "pokemon_id": fx.pokemonID},
			{"slot": 1, "pokemon_id": fx.pokemonID},
		}}
		w := teamRequest(t, h.PatchTeam, http.MethodPatch, teamID, userID, body)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("more than 4 moves on a slot returns 422", func(t *testing.T) {
		moves := make([]map[string]any, 5)
		for i := range moves {
			moves[i] = map[string]any{"slot": i + 1, "move_id": fx.moveID}
		}
		body := map[string]any{"slots": []map[string]any{{"slot": 1, "pokemon_id": fx.pokemonID, "moves": moves}}}
		w := teamRequest(t, h.PatchTeam, http.MethodPatch, teamID, userID, body)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("missing pokemon_id on slot returns 422", func(t *testing.T) {
		body := map[string]any{"slots": []map[string]any{{"slot": 1, "pokemon_id": ""}}}
		w := teamRequest(t, h.PatchTeam, http.MethodPatch, teamID, userID, body)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

// ---------------------------------------------------------------------------
// Cross-user isolation
// ---------------------------------------------------------------------------

func TestTeamIsolation(t *testing.T) {
	h := newTeamHandler(t)
	userA := registerUser(t, h, "teamisoA@example.com")
	userB := registerUser(t, h, "teamisoB@example.com")
	fx := getFixtures(t, h)

	teamID := createTeam(t, h, userA, "User A Team")

	t.Run("user B cannot GET user A's team", func(t *testing.T) {
		w := teamRequest(t, h.GetTeam, http.MethodGet, teamID, userB, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("user B cannot PATCH user A's team", func(t *testing.T) {
		body := map[string]any{"slots": []map[string]any{{"slot": 1, "pokemon_id": fx.pokemonID}}}
		w := teamRequest(t, h.PatchTeam, http.MethodPatch, teamID, userB, body)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("user B cannot DELETE user A's team", func(t *testing.T) {
		w := teamRequest(t, h.DeleteTeam, http.MethodDelete, teamID, userB, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("user B cannot ACTIVATE user A's team", func(t *testing.T) {
		w := teamRequest(t, h.ActivateTeam, http.MethodPost, teamID, userB, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("user A's team is unmodified after all B's attempts", func(t *testing.T) {
		w := teamRequest(t, h.GetTeam, http.MethodGet, teamID, userA, nil)
		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "User A Team", resp["name"])
	})
}

// ---------------------------------------------------------------------------
// Local helpers
// ---------------------------------------------------------------------------

func createTeam(t *testing.T, h *handler.Handler, userID, name string) string {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/", jsonBody(fmt.Sprintf(`{"name":%q}`, name)))
	r.Header.Set("Content-Type", "application/json")
	r = withUser(r, userID)
	w := httptest.NewRecorder()
	h.CreateTeam(w, r)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp["id"].(string)
}

func listTeams(t *testing.T, h *handler.Handler, userID string) []map[string]any {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = withUser(r, userID)
	w := httptest.NewRecorder()
	h.ListTeams(w, r)
	require.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

func jsonBody(s string) *bytes.Reader {
	return bytes.NewReader([]byte(s))
}
