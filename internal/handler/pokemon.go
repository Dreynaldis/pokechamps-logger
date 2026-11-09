package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dreynaldis/pokechamps-logger/internal/model"
	"github.com/go-chi/chi/v5"
)

// ListPokemon handles GET /api/v1/pokemon?q=&limit=
// Returns a summary list for the search/autocomplete UI.
func (h *Handler) ListPokemon(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit := 20
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 300 {
		limit = l
	}

	var pokemon []model.Pokemon
	query := h.DB.Select("id, name, display_name, types, sprite_url").Where("name NOT LIKE ?", "%-mega%")
	if q != "" {
		query = query.Where("display_name ILIKE ?", "%"+q+"%")
	}
	if err := query.Limit(limit).Order("display_name").Find(&pokemon).Error; err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type summary struct {
		ID          string          `json:"id"`
		Name        string          `json:"name"`
		DisplayName string          `json:"display_name"`
		Types       json.RawMessage `json:"types"`
		SpriteURL   string          `json:"sprite_url"`
	}

	result := make([]summary, len(pokemon))
	for i, p := range pokemon {
		result[i] = summary{
			ID:          p.ID,
			Name:        p.Name,
			DisplayName: p.DisplayName,
			Types:       json.RawMessage(p.Types),
			SpriteURL:   p.SpriteURL,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ListItems handles GET /api/v1/items?q=&limit=
// Returns a searchable list for the held-item picker UI.
func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit := 200
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 200 {
		limit = l
	}

	var items []model.Item
	query := h.DB.Select("id, name, display_name, category, short_effect, sprite_url")
	if q != "" {
		query = query.Where("display_name ILIKE ?", "%"+q+"%")
	}
	if err := query.Limit(limit).Order("display_name").Find(&items).Error; err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type summary struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Category    string `json:"category"`
		ShortEffect string `json:"short_effect"`
		SpriteURL   string `json:"sprite_url"`
	}

	result := make([]summary, len(items))
	for i, it := range items {
		result[i] = summary{
			ID:          it.ID,
			Name:        it.Name,
			DisplayName: it.DisplayName,
			Category:    it.Category,
			ShortEffect: it.ShortEffect,
			SpriteURL:   it.SpriteURL,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetPokemon handles GET /api/v1/pokemon/:name
// Returns full Pokemon detail with abilities and moves for Phase 1 verification.
func (h *Handler) GetPokemon(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var p model.Pokemon
	err := h.DB.
		Where("name = ?", name).
		Preload("PokemonAbilities.Ability").
		Preload("Learnset.Move").
		First(&p).Error
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	type abilityOut struct {
		ID                 string `json:"id"`
		Name               string `json:"name"`
		DisplayName        string `json:"display_name"`
		ShortEffect        string `json:"short_effect"`
		GrantsImmunityType string `json:"grants_immunity_type,omitempty"`
	}

	type moveOut struct {
		ID                 string `json:"id"`
		Name               string `json:"name"`
		DisplayName        string `json:"display_name"`
		Type               string `json:"type"`
		Category           string `json:"category"`
		Power              *int   `json:"power"`
		Accuracy           *int   `json:"accuracy"`
		PP                 int    `json:"pp"`
		Priority           int    `json:"priority"`
		HasSecondaryEffect bool   `json:"has_secondary_effect"`
		IsPivot            bool   `json:"is_pivot"`
	}

	type detail struct {
		ID          string          `json:"id"`
		Name        string          `json:"name"`
		DisplayName string          `json:"display_name"`
		SpriteURL   string          `json:"sprite_url"`
		Types       json.RawMessage `json:"types"`
		BaseStats   json.RawMessage `json:"base_stats"`
		Abilities   []abilityOut    `json:"abilities"`
		Moves       []moveOut       `json:"moves"`
	}

	abilities := make([]abilityOut, len(p.PokemonAbilities))
	for i, pa := range p.PokemonAbilities {
		abilities[i] = abilityOut{
			ID:                 pa.Ability.ID,
			Name:               pa.Ability.Name,
			DisplayName:        pa.Ability.DisplayName,
			ShortEffect:        pa.Ability.ShortEffect,
			GrantsImmunityType: pa.Ability.GrantsImmunityType,
		}
	}

	moves := make([]moveOut, len(p.Learnset))
	for i, ls := range p.Learnset {
		moves[i] = moveOut{
			ID:                 ls.Move.ID,
			Name:               ls.Move.Name,
			DisplayName:        ls.Move.DisplayName,
			Type:               ls.Move.Type,
			Category:           ls.Move.Category,
			Power:              ls.Move.Power,
			Accuracy:           ls.Move.Accuracy,
			PP:                 ls.Move.PP,
			Priority:           ls.Move.Priority,
			HasSecondaryEffect: ls.Move.HasSecondaryEffect,
			IsPivot:            ls.Move.IsPivot,
		}
	}

	out := detail{
		ID:          p.ID,
		Name:        p.Name,
		DisplayName: p.DisplayName,
		SpriteURL:   p.SpriteURL,
		Types:       json.RawMessage(p.Types),
		BaseStats:   json.RawMessage(p.BaseStats),
		Abilities:   abilities,
		Moves:       moves,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
