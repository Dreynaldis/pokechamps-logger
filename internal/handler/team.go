package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dreynaldis/pokechamps-logger/internal/auth"
	"github.com/dreynaldis/pokechamps-logger/internal/model"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Output types
// ---------------------------------------------------------------------------

type pokemonRef struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	DisplayName string          `json:"display_name"`
	SpriteURL   string          `json:"sprite_url,omitempty"`
	Types       json.RawMessage `json:"types"`
}

type abilityRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

type itemRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

type moveRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

type slotMoveOut struct {
	Slot int     `json:"slot"`
	Move moveRef `json:"move"`
}

type slotOut struct {
	ID      string        `json:"id"`
	Slot    int           `json:"slot"`
	Pokemon pokemonRef    `json:"pokemon"`
	Ability *abilityRef   `json:"ability,omitempty"`
	Item    *itemRef      `json:"item,omitempty"`
	Nature  *string       `json:"nature,omitempty"`
	Moves   []slotMoveOut `json:"moves"`
}

type teamOut struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	IsActive bool      `json:"is_active"`
	Slots    []slotOut `json:"slots,omitempty"`
}

// ---------------------------------------------------------------------------
// Input types
// ---------------------------------------------------------------------------

type slotMoveInput struct {
	Slot   int    `json:"slot"`
	MoveID string `json:"move_id"`
}

type slotInput struct {
	Slot      int             `json:"slot"`
	PokemonID string          `json:"pokemon_id"`
	AbilityID *string         `json:"ability_id"`
	ItemID    *string         `json:"item_id"`
	Nature    *string         `json:"nature"`
	Moves     []slotMoveInput `json:"moves"`
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func toTeamOut(t model.Team, withSlots bool) teamOut {
	out := teamOut{ID: t.ID, Name: t.Name, IsActive: t.IsActive}
	if !withSlots {
		return out
	}
	out.Slots = make([]slotOut, 0, len(t.Slots))
	for _, s := range t.Slots {
		so := slotOut{
			ID:   s.ID,
			Slot: s.Slot,
			Pokemon: pokemonRef{
				ID:          s.Pokemon.ID,
				Name:        s.Pokemon.Name,
				DisplayName: s.Pokemon.DisplayName,
				SpriteURL:   s.Pokemon.SpriteURL,
				Types:       json.RawMessage(s.Pokemon.Types),
			},
			Nature: s.Nature,
		}
		if s.Ability != nil {
			so.Ability = &abilityRef{ID: s.Ability.ID, Name: s.Ability.Name, DisplayName: s.Ability.DisplayName}
		}
		if s.Item != nil {
			so.Item = &itemRef{ID: s.Item.ID, Name: s.Item.Name, DisplayName: s.Item.DisplayName}
		}
		so.Moves = make([]slotMoveOut, 0, len(s.Moves))
		for _, m := range s.Moves {
			so.Moves = append(so.Moves, slotMoveOut{
				Slot: m.Slot,
				Move: moveRef{ID: m.Move.ID, Name: m.Move.Name, DisplayName: m.Move.DisplayName, Type: m.Move.Type},
			})
		}
		out.Slots = append(out.Slots, so)
	}
	return out
}

func loadTeamFull(db *gorm.DB, teamID, userID string) (model.Team, error) {
	var team model.Team
	err := db.
		Where("id = ? AND user_id = ?", teamID, userID).
		Preload("Slots", func(db *gorm.DB) *gorm.DB { return db.Order("slot asc") }).
		Preload("Slots.Pokemon").
		Preload("Slots.Ability").
		Preload("Slots.Item").
		Preload("Slots.Moves", func(db *gorm.DB) *gorm.DB { return db.Order("slot asc") }).
		Preload("Slots.Moves.Move").
		First(&team).Error
	return team, err
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// ListTeams handles GET /api/v1/teams
func (h *Handler) ListTeams(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(auth.ContextKeyUserID).(string)

	var teams []model.Team
	h.DB.Where("user_id = ?", userID).Order("created_at asc").Find(&teams)

	out := make([]teamOut, len(teams))
	for i, t := range teams {
		out[i] = toTeamOut(t, false)
	}
	writeJSON(w, http.StatusOK, out)
}

// CreateTeam handles POST /api/v1/teams
func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(auth.ContextKeyUserID).(string)

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Name == "" || len(body.Name) > 50 {
		writeError(w, http.StatusUnprocessableEntity, "name is required and must be under 50 characters")
		return
	}

	team := model.Team{UserID: userID, Name: body.Name}
	if err := h.DB.Create(&team).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not create team")
		return
	}
	writeJSON(w, http.StatusCreated, toTeamOut(team, false))
}

// GetTeam handles GET /api/v1/teams/:id
func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(auth.ContextKeyUserID).(string)
	teamID := chi.URLParam(r, "id")

	team, err := loadTeamFull(h.DB, teamID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "team not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toTeamOut(team, true))
}

// PatchTeam handles PATCH /api/v1/teams/:id
// Accepts { name?, slots? }. If slots is provided (even empty []), it fully
// replaces all existing slots for the team. If slots is omitted, slots are
// left unchanged. This matches the auto-save model: client owns the full state.
func (h *Handler) PatchTeam(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(auth.ContextKeyUserID).(string)
	teamID := chi.URLParam(r, "id")

	var body struct {
		Name  *string      `json:"name"`
		Slots *[]slotInput `json:"slots"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if body.Name != nil && (len(*body.Name) == 0 || len(*body.Name) > 50) {
		writeError(w, http.StatusUnprocessableEntity, "name must be 1-50 characters")
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// Verify ownership
		var team model.Team
		if err := tx.Where("id = ? AND user_id = ?", teamID, userID).First(&team).Error; err != nil {
			return err
		}

		if body.Name != nil {
			if err := tx.Model(&team).Update("name", *body.Name).Error; err != nil {
				return err
			}
		}

		if body.Slots != nil {
			if err := validateSlots(*body.Slots); err != nil {
				return err
			}

			// Full replace: delete existing slot moves then slots
			var slotIDs []string
			tx.Model(&model.TeamSlot{}).Where("team_id = ?", teamID).Pluck("id", &slotIDs)
			if len(slotIDs) > 0 {
				if err := tx.Where("team_slot_id IN ?", slotIDs).Delete(&model.TeamSlotMove{}).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("team_id = ?", teamID).Delete(&model.TeamSlot{}).Error; err != nil {
				return err
			}

			// Insert new slots and moves
			for _, si := range *body.Slots {
				slot := model.TeamSlot{
					TeamID:    teamID,
					Slot:      si.Slot,
					PokemonID: si.PokemonID,
					AbilityID: si.AbilityID,
					ItemID:    si.ItemID,
					Nature:    si.Nature,
				}
				if err := tx.Create(&slot).Error; err != nil {
					return err
				}
				for _, mi := range si.Moves {
					move := model.TeamSlotMove{TeamSlotID: slot.ID, Slot: mi.Slot, MoveID: mi.MoveID}
					if err := tx.Create(&move).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "team not found")
			return
		}
		if err.Error() == "invalid slots" {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	team, _ := loadTeamFull(h.DB, teamID, userID)
	writeJSON(w, http.StatusOK, toTeamOut(team, true))
}

// DeleteTeam handles DELETE /api/v1/teams/:id
func (h *Handler) DeleteTeam(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(auth.ContextKeyUserID).(string)
	teamID := chi.URLParam(r, "id")

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var team model.Team
		if err := tx.Where("id = ? AND user_id = ?", teamID, userID).First(&team).Error; err != nil {
			return err
		}
		// Delete moves, slots, then team (AutoMigrate doesn't guarantee cascade DDL)
		var slotIDs []string
		tx.Model(&model.TeamSlot{}).Where("team_id = ?", teamID).Pluck("id", &slotIDs)
		if len(slotIDs) > 0 {
			tx.Where("team_slot_id IN ?", slotIDs).Delete(&model.TeamSlotMove{})
		}
		tx.Where("team_id = ?", teamID).Delete(&model.TeamSlot{})
		return tx.Delete(&team).Error
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "team not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ActivateTeam handles POST /api/v1/teams/:id/activate
// Wraps in a transaction: deactivate all user teams, then activate this one.
func (h *Handler) ActivateTeam(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(auth.ContextKeyUserID).(string)
	teamID := chi.URLParam(r, "id")

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var team model.Team
		if err := tx.Where("id = ? AND user_id = ?", teamID, userID).First(&team).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Team{}).Where("user_id = ?", userID).Update("is_active", false).Error; err != nil {
			return err
		}
		return tx.Model(&team).Update("is_active", true).Error
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "team not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

type validationError string

func (e validationError) Error() string { return string(e) }

func validateSlots(slots []slotInput) error {
	if len(slots) > 6 {
		return validationError("invalid slots")
	}
	seen := map[int]bool{}
	for _, s := range slots {
		if s.Slot < 1 || s.Slot > 6 {
			return validationError("invalid slots")
		}
		if seen[s.Slot] {
			return validationError("invalid slots")
		}
		seen[s.Slot] = true
		if s.PokemonID == "" {
			return validationError("invalid slots")
		}
		if len(s.Moves) > 4 {
			return validationError("invalid slots")
		}
		seenMove := map[int]bool{}
		for _, m := range s.Moves {
			if m.Slot < 1 || m.Slot > 4 || m.MoveID == "" || seenMove[m.Slot] {
				return validationError("invalid slots")
			}
			seenMove[m.Slot] = true
		}
	}
	return nil
}
