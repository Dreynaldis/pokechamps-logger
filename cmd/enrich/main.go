// cmd/enrich fetches move and ability metadata from PokeAPI.
// It reads seed/pokebase-raw.json and outputs:
//   seed/moves-enriched.json
//   seed/abilities-enriched.json
//
// Usage: go run ./cmd/enrich
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const pokeAPIBase = "https://pokeapi.co/api/v2"

// --- Input types (from pokebase-raw.json) ---

type RawMove struct {
	Name     string `json:"name"`
	MoveType string `json:"type"`
}

type RawAbility struct {
	Name string `json:"name"`
}

type RawPokemon struct {
	Abilities []RawAbility `json:"abilities"`
	Moves     []RawMove    `json:"moves"`
}

type RawOutput struct {
	Pokemon []RawPokemon `json:"pokemon"`
}

// --- PokeAPI response types ---

type pokeAPIMove struct {
	Priority     int    `json:"priority"`
	Target       struct{ Name string `json:"name"` } `json:"target"`
	EffectChance *int   `json:"effect_chance"`
	Meta         struct {
		Ailment     struct{ Name string `json:"name"` } `json:"ailment"`
		FlinchChance int `json:"flinch_chance"`
		Drain        int `json:"drain"`
		Healing      int `json:"healing"`
	} `json:"meta"`
	StatChanges  []struct{ Change int `json:"change"` } `json:"stat_changes"`
	EffectEntries []struct {
		Effect   string `json:"effect"`
		Language struct{ Name string `json:"name"` } `json:"language"`
	} `json:"effect_entries"`
}

type pokeAPIAbility struct {
	EffectEntries []struct {
		ShortEffect string `json:"short_effect"`
		Language    struct{ Name string `json:"name"` } `json:"language"`
	} `json:"effect_entries"`
}

// --- Output types ---

type EnrichedMove struct {
	Name               string `json:"name"`
	Priority           int    `json:"priority"`
	Target             string `json:"target"`
	ShortEffect        string `json:"short_effect"`
	EffectChance       *int   `json:"effect_chance"`
	HasSecondaryEffect bool   `json:"has_secondary_effect"`
	IsPivot            bool   `json:"is_pivot"`
}

type EnrichedAbility struct {
	Name        string `json:"name"`
	ShortEffect string `json:"short_effect"`
}

type EnrichedOutput struct {
	Moves     []EnrichedMove    `json:"moves"`
	Abilities []EnrichedAbility `json:"abilities"`
}

var pivotMoves = map[string]bool{
	"flip-turn": true, "volt-switch": true, "u-turn": true, "parting-shot": true,
}

func main() {
	raw, err := loadRaw("seed/pokebase-raw.json")
	if err != nil {
		log.Fatalf("load raw: %v", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	throttle := time.NewTicker(time.Second)
	defer throttle.Stop()

	// Collect unique move and ability names
	moveNames := unique(collectMoveNames(raw))
	abilityNames := unique(collectAbilityNames(raw))

	log.Printf("enriching %d moves and %d abilities from PokeAPI", len(moveNames), len(abilityNames))

	// Enrich moves
	moves := make([]EnrichedMove, 0, len(moveNames))
	for i, name := range moveNames {
		<-throttle.C
		log.Printf("[move %d/%d] %s", i+1, len(moveNames), name)
		m, err := fetchMove(client, name)
		if err != nil {
			log.Printf("  WARN: %v", err)
			moves = append(moves, EnrichedMove{Name: name, Target: "selected-pokemon", IsPivot: pivotMoves[name]})
			continue
		}
		moves = append(moves, *m)
	}

	// Enrich abilities
	abilities := make([]EnrichedAbility, 0, len(abilityNames))
	for i, name := range abilityNames {
		<-throttle.C
		log.Printf("[ability %d/%d] %s", i+1, len(abilityNames), name)
		a, err := fetchAbility(client, name)
		if err != nil {
			log.Printf("  WARN: %v", err)
			abilities = append(abilities, EnrichedAbility{Name: name})
			continue
		}
		abilities = append(abilities, *a)
	}

	out := EnrichedOutput{Moves: moves, Abilities: abilities}
	if err := writeJSON("seed/moves-enriched.json", out.Moves); err != nil {
		log.Fatalf("write moves: %v", err)
	}
	if err := writeJSON("seed/abilities-enriched.json", out.Abilities); err != nil {
		log.Fatalf("write abilities: %v", err)
	}
	log.Println("done.")
}

func fetchMove(client *http.Client, name string) (*EnrichedMove, error) {
	var api pokeAPIMove
	if err := getJSON(client, fmt.Sprintf("%s/move/%s", pokeAPIBase, name), &api); err != nil {
		return nil, fmt.Errorf("move %s: %w", name, err)
	}

	hasSec := api.EffectChance != nil ||
		(api.Meta.Ailment.Name != "" && api.Meta.Ailment.Name != "none") ||
		api.Meta.FlinchChance > 0 ||
		len(api.StatChanges) > 0 ||
		api.Meta.Drain != 0 ||
		api.Meta.Healing != 0

	shortEffect := ""
	for _, e := range api.EffectEntries {
		if e.Language.Name == "en" {
			shortEffect = e.Effect
			break
		}
	}

	return &EnrichedMove{
		Name:               name,
		Priority:           api.Priority,
		Target:             api.Target.Name,
		ShortEffect:        shortEffect,
		EffectChance:       api.EffectChance,
		HasSecondaryEffect: hasSec,
		IsPivot:            pivotMoves[name],
	}, nil
}

func fetchAbility(client *http.Client, name string) (*EnrichedAbility, error) {
	var api pokeAPIAbility
	if err := getJSON(client, fmt.Sprintf("%s/ability/%s", pokeAPIBase, name), &api); err != nil {
		return nil, fmt.Errorf("ability %s: %w", name, err)
	}

	shortEffect := ""
	for _, e := range api.EffectEntries {
		if e.Language.Name == "en" {
			shortEffect = e.ShortEffect
			break
		}
	}

	return &EnrichedAbility{Name: name, ShortEffect: shortEffect}, nil
}

func getJSON(client *http.Client, url string, out interface{}) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found: %s", url)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func loadRaw(path string) (*RawOutput, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out RawOutput
	return &out, json.NewDecoder(f).Decode(&out)
}

func collectMoveNames(raw *RawOutput) []string {
	var names []string
	for _, p := range raw.Pokemon {
		for _, m := range p.Moves {
			names = append(names, m.Name)
		}
	}
	return names
}

func collectAbilityNames(raw *RawOutput) []string {
	var names []string
	for _, p := range raw.Pokemon {
		for _, a := range p.Abilities {
			names = append(names, a.Name)
		}
	}
	return names
}

func unique(names []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, n := range names {
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

func writeJSON(path string, v interface{}) error {
	if err := os.MkdirAll("seed", 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func slugify(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", "-"))
}

var _ = slugify // suppress unused warning
