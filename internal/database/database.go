package database

import (
	"fmt"

	"github.com/dreynaldis/pokechamps-logger/internal/config"
	"github.com/dreynaldis/pokechamps-logger/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// Migrate runs AutoMigrate for all registered models.
// Models are added here as each phase introduces them.
func Migrate(db *gorm.DB) error {
	// Enable pg_trgm for autocomplete (Phase 3+). Safe to run repeatedly.
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm").Error; err != nil {
		return fmt.Errorf("pg_trgm extension: %w", err)
	}

	return db.AutoMigrate(
		// Phase 1 -- seed / reference tables
		&model.Ability{},
		&model.Item{},
		&model.Pokemon{},
		&model.Move{},
		&model.PokemonAbility{},
		&model.PokemonLearnset{},
		&model.MegaForm{},
	)
}
