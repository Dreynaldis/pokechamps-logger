package model

import "gorm.io/datatypes"

// Pokemon is a Champions species (310 total).
type Pokemon struct {
	ID          string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DexNumber   int            `gorm:"uniqueIndex;not null"`
	Name        string         `gorm:"uniqueIndex;not null"` // lowercase-hyphenated slug
	DisplayName string         `gorm:"not null"`
	SpriteURL   string
	Types       datatypes.JSON `gorm:"type:jsonb;not null"` // []string
	BaseStats   datatypes.JSON `gorm:"type:jsonb;not null"` // BaseStatsJSON

	PokemonAbilities []PokemonAbility `gorm:"foreignKey:PokemonID"`
	Learnset         []PokemonLearnset `gorm:"foreignKey:PokemonID"`
	MegaForms        []MegaForm       `gorm:"foreignKey:BasePokemonID"`
}

func (Pokemon) TableName() string { return "pokemon" }

// BaseStatsJSON is the shape stored in pokemon.base_stats JSONB.
type BaseStatsJSON struct {
	HP      int `json:"hp"`
	Attack  int `json:"attack"`
	Defense int `json:"defense"`
	SpAtk   int `json:"sp_atk"`
	SpDef   int `json:"sp_def"`
	Speed   int `json:"speed"`
}

// Ability is one ability entity shared across Pokemon.
type Ability struct {
	ID                 string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name               string `gorm:"uniqueIndex;not null"`
	DisplayName        string `gorm:"not null"`
	ShortEffect        string
	GrantsImmunityType string
}

// PokemonAbility is the join table; PK is (pokemon_id, ability_id).
type PokemonAbility struct {
	PokemonID string  `gorm:"primaryKey;type:uuid;not null;column:pokemon_id"`
	AbilityID string  `gorm:"primaryKey;type:uuid;not null;column:ability_id"`
	Ability   Ability `gorm:"foreignKey:AbilityID"`
}

// Move is a move entity with stats from pokebase + PokeAPI enrichment.
type Move struct {
	ID                 string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name               string `gorm:"uniqueIndex;not null"`
	DisplayName        string `gorm:"not null"`
	Type               string `gorm:"not null"`
	Category           string `gorm:"not null"` // physical | special | status
	Power              *int
	Accuracy           *int
	PP                 int    `gorm:"not null"`
	Priority           int    `gorm:"not null;default:0"`
	Target             string `gorm:"not null;default:'selected-pokemon'"`
	ShortEffect        string
	EffectChance       *int
	HasSecondaryEffect bool   `gorm:"not null;default:false"`
	IsPivot            bool   `gorm:"not null;default:false"`
}

// PokemonLearnset is the Champions-specific learnset join table.
type PokemonLearnset struct {
	PokemonID string `gorm:"primaryKey;type:uuid;not null;column:pokemon_id"`
	MoveID    string `gorm:"primaryKey;type:uuid;not null;column:move_id"`
	Move      Move   `gorm:"foreignKey:MoveID"`
}

// Item covers mega stones and held items.
type Item struct {
	ID                  string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name                string `gorm:"uniqueIndex;not null"`
	DisplayName         string `gorm:"not null"`
	Category            string `gorm:"not null"`
	ShortEffect         string
	FieldEffectDuration *int
}

// MegaForm is derived from Pokemon + Item combination.
type MegaForm struct {
	ID            string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	BasePokemonID string `gorm:"type:uuid;not null;uniqueIndex:idx_mega_base_item"`
	ItemID        string `gorm:"type:uuid;not null;uniqueIndex:idx_mega_base_item"`
	FormName      string `gorm:"not null"`
	SpriteURL     string
}
