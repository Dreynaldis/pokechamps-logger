// cmd/seed loads all seed JSON files into the database.
// Run after scrape + enrich. Safe to re-run (upserts on unique name constraints).
//
// Usage: go run ./cmd/seed
package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/dreynaldis/pokechamps-logger/internal/config"
	"github.com/dreynaldis/pokechamps-logger/internal/database"
	"github.com/dreynaldis/pokechamps-logger/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- JSON input types ---

type rawBaseStats struct {
	HP      int `json:"hp"`
	Attack  int `json:"attack"`
	Defense int `json:"defense"`
	SpAtk   int `json:"sp_atk"`
	SpDef   int `json:"sp_def"`
	Speed   int `json:"speed"`
}

type rawAbility struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

type rawMove struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	Category    string `json:"category"`
	Power       *int   `json:"power"`
	Accuracy    *int   `json:"accuracy"`
	PP          int    `json:"pp"`
}

type rawPokemon struct {
	Slug        string       `json:"slug"`
	DexNumber   int          `json:"dex_number"`
	DisplayName string       `json:"display_name"`
	SpriteURL   string       `json:"sprite_url"`
	Types       []string     `json:"types"`
	BaseStats   rawBaseStats `json:"base_stats"`
	Abilities   []rawAbility `json:"abilities"`
	Moves       []rawMove    `json:"moves"`
}

type rawOutput struct {
	Pokemon []rawPokemon `json:"pokemon"`
}

type enrichedMove struct {
	Name               string `json:"name"`
	Priority           int    `json:"priority"`
	Target             string `json:"target"`
	ShortEffect        string `json:"short_effect"`
	EffectChance       *int   `json:"effect_chance"`
	HasSecondaryEffect bool   `json:"has_secondary_effect"`
	IsPivot            bool   `json:"is_pivot"`
}

type enrichedAbility struct {
	Name        string `json:"name"`
	ShortEffect string `json:"short_effect"`
}

type overrideMovePatch struct {
	Name  string          `json:"name"`
	Patch map[string]any  `json:"patch"`
}

type overrideAbilityPatch struct {
	Name  string          `json:"name"`
	Patch map[string]any  `json:"patch"`
}

type overrideMegaForm struct {
	BasePokemon string `json:"base_pokemon"`
	Item        string `json:"item"`
	FormName    string `json:"form_name"`
	SpriteURL   string `json:"sprite_url"`
}

type overrides struct {
	MovePatches     []overrideMovePatch    `json:"move_patches"`
	AbilityPatches  []overrideAbilityPatch `json:"ability_patches"`
	MegaForms       []overrideMegaForm     `json:"mega_forms"`
}

func main() {
	cfg := config.Load()
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	raw := mustLoadJSON[rawOutput]("seed/pokebase-raw.json")
	enrichedMoves := mustLoadJSON[[]enrichedMove]("seed/moves-enriched.json")
	enrichedAbilities := mustLoadJSON[[]enrichedAbility]("seed/abilities-enriched.json")
	ov := mustLoadJSON[overrides]("seed/champions-overrides.json")

	// Build lookup maps
	moveEnrichMap := make(map[string]enrichedMove, len(enrichedMoves))
	for _, m := range enrichedMoves {
		moveEnrichMap[m.Name] = m
	}
	abilityEnrichMap := make(map[string]enrichedAbility, len(enrichedAbilities))
	for _, a := range enrichedAbilities {
		abilityEnrichMap[a.Name] = a
	}
	movePatchMap := make(map[string]map[string]any, len(ov.MovePatches))
	for _, p := range ov.MovePatches {
		movePatchMap[p.Name] = p.Patch
	}
	abilityPatchMap := make(map[string]map[string]any, len(ov.AbilityPatches))
	for _, p := range ov.AbilityPatches {
		abilityPatchMap[p.Name] = p.Patch
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		// 1. Upsert abilities
		log.Println("upserting abilities...")
		abilityIDMap := make(map[string]string)
		allAbilities := collectAbilities(raw.Pokemon, abilityEnrichMap, abilityPatchMap)
		for _, a := range allAbilities {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoUpdates: clause.AssignmentColumns([]string{"display_name", "short_effect", "grants_immunity_type"}),
			}).Create(&a).Error; err != nil {
				return err
			}
			abilityIDMap[a.Name] = a.ID
		}

		// 2. Upsert moves
		log.Println("upserting moves...")
		moveIDMap := make(map[string]string)
		allMoves := collectMoves(raw.Pokemon, moveEnrichMap, movePatchMap)
		for _, m := range allMoves {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"display_name", "type", "category", "power", "accuracy", "pp",
					"priority", "target", "short_effect", "effect_chance",
					"has_secondary_effect", "is_pivot",
				}),
			}).Create(&m).Error; err != nil {
				return err
			}
			moveIDMap[m.Name] = m.ID
		}

		// 3. Upsert Pokemon
		log.Println("upserting pokemon...")
		pokemonIDMap := make(map[string]string)
		for i, rp := range raw.Pokemon {
			typesJSON, _ := json.Marshal(rp.Types)
			statsJSON, _ := json.Marshal(rp.BaseStats)
			dex := rp.DexNumber
			if dex == 0 {
				dex = i + 1
			}
			p := model.Pokemon{
				DexNumber:   dex,
				Name:        rp.Slug,
				DisplayName: rp.DisplayName,
				SpriteURL:   rp.SpriteURL,
				Types:       datatypes.JSON(typesJSON),
				BaseStats:   datatypes.JSON(statsJSON),
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoUpdates: clause.AssignmentColumns([]string{"display_name", "types", "base_stats", "sprite_url"}),
			}).Create(&p).Error; err != nil {
				return err
			}
			pokemonIDMap[rp.Slug] = p.ID

			// 4. Pokemon abilities
			for _, ra := range rp.Abilities {
				abID, ok := abilityIDMap[ra.Name]
				if !ok {
					log.Printf("  WARN: ability %q not found for %s, skipping", ra.Name, rp.Slug)
					continue
				}
				pa := model.PokemonAbility{
					PokemonID: p.ID,
					AbilityID: abID,
				}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&pa).Error; err != nil {
					return err
				}
			}

			// 5. Learnset
			for _, rm := range rp.Moves {
				mID, ok := moveIDMap[rm.Name]
				if !ok {
					log.Printf("  WARN: move %q not found for %s, skipping", rm.Name, rp.Slug)
					continue
				}
				ls := model.PokemonLearnset{PokemonID: p.ID, MoveID: mID}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&ls).Error; err != nil {
					return err
				}
			}
		}

		// 6. Mega forms (need item IDs -- items seeded separately; skip if item not found)
		log.Println("skipping mega forms (items not seeded yet in Phase 1)")
		_ = pokemonIDMap

		log.Println("transaction complete")
		return nil
	})
	if err != nil {
		log.Fatalf("seed transaction: %v", err)
	}

	// Quick verification
	var count int64
	db.Model(&model.Pokemon{}).Count(&count)
	log.Printf("done. %d Pokemon in DB.", count)
}

func collectAbilities(pokemon []rawPokemon, enrichMap map[string]enrichedAbility, patchMap map[string]map[string]any) []model.Ability {
	seen := make(map[string]bool)
	var out []model.Ability
	for _, p := range pokemon {
		for _, a := range p.Abilities {
			if seen[a.Name] {
				continue
			}
			seen[a.Name] = true
			ab := model.Ability{
				Name:        a.Name,
				DisplayName: a.DisplayName,
			}
			if e, ok := enrichMap[a.Name]; ok {
				ab.ShortEffect = e.ShortEffect
			}
			if patch, ok := patchMap[a.Name]; ok {
				if v, ok := patch["grants_immunity_type"].(string); ok {
					ab.GrantsImmunityType = v
				}
			}
			out = append(out, ab)
		}
	}
	return out
}

func collectMoves(pokemon []rawPokemon, enrichMap map[string]enrichedMove, patchMap map[string]map[string]any) []model.Move {
	seen := make(map[string]bool)
	var out []model.Move
	for _, p := range pokemon {
		for _, rm := range p.Moves {
			if seen[rm.Name] {
				continue
			}
			seen[rm.Name] = true
			m := model.Move{
				Name:        rm.Name,
				DisplayName: rm.DisplayName,
				Type:        rm.Type,
				Category:    rm.Category,
				Power:       rm.Power,
				Accuracy:    rm.Accuracy,
				PP:          rm.PP,
				Target:      "selected-pokemon",
			}
			if e, ok := enrichMap[rm.Name]; ok {
				m.Priority = e.Priority
				m.Target = e.Target
				m.ShortEffect = e.ShortEffect
				m.EffectChance = e.EffectChance
				m.HasSecondaryEffect = e.HasSecondaryEffect
				m.IsPivot = e.IsPivot
			}
			if patch, ok := patchMap[rm.Name]; ok {
				if v, ok := patch["is_pivot"].(bool); ok {
					m.IsPivot = v
				}
			}
			out = append(out, m)
		}
	}
	return out
}

func mustLoadJSON[T any](path string) T {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	var v T
	if err := json.NewDecoder(f).Decode(&v); err != nil {
		log.Fatalf("decode %s: %v", path, err)
	}
	return v
}
