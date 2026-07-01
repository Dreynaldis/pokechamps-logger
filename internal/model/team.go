package model

import (
	"time"

	"gorm.io/datatypes"
)

// Team is one saved team belonging to a user. Only one team per user can have
// is_active=true at a time; the constraint is enforced in the application layer
// via a transaction in the activate endpoint.
type Team struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    string    `gorm:"type:uuid;not null;index"`
	Name      string    `gorm:"not null"`
	IsActive  bool      `gorm:"not null;default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Slots []TeamSlot `gorm:"foreignKey:TeamID"`
}

// TeamSlot is one of the six positions in a team.
// PokemonID is the only required field; ability, item, nature, and moves are
// optional and can be added incrementally as the team is built.
type TeamSlot struct {
	ID        string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TeamID    string         `gorm:"type:uuid;not null;uniqueIndex:idx_team_slot"`
	Slot      int            `gorm:"not null;uniqueIndex:idx_team_slot;check:slot >= 1 AND slot <= 6"`
	PokemonID string         `gorm:"type:uuid;not null"`
	AbilityID *string        `gorm:"type:uuid"`
	ItemID    *string        `gorm:"type:uuid"`
	Nature    *string
	// TrainingPoints stores EV-equivalent values; exact schema TBD.
	TrainingPoints datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt      time.Time

	Pokemon Pokemon  `gorm:"foreignKey:PokemonID"`
	Ability *Ability `gorm:"foreignKey:AbilityID"`
	Item    *Item    `gorm:"foreignKey:ItemID"`
	Moves   []TeamSlotMove `gorm:"foreignKey:TeamSlotID"`
}

// TeamSlotMove is one move in a slot. Composite PK enforces uniqueness of
// (team_slot_id, slot) so each move position can only hold one move.
type TeamSlotMove struct {
	TeamSlotID string `gorm:"primaryKey;type:uuid;not null;column:team_slot_id"`
	Slot       int    `gorm:"primaryKey;not null;check:slot >= 1 AND slot <= 4"`
	MoveID     string `gorm:"type:uuid;not null"`

	Move Move `gorm:"foreignKey:MoveID"`
}
