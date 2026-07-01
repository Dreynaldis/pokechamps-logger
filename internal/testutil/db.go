// Package testutil provides helpers for integration tests that need a real DB.
// Tests using SetupDB require the pokechamps Postgres container to be running
// (docker compose up -d) and DATABASE_URL set in the environment or .env.
package testutil

import (
	"os"
	"testing"

	"github.com/dreynaldis/pokechamps-logger/internal/database"
	"github.com/dreynaldis/pokechamps-logger/internal/model"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

// SetupDB opens a connection to the real DB, runs migrations, and returns a
// *gorm.DB scoped to a transaction that is rolled back at test end.
// This means each test starts clean without truncating tables between runs.
func SetupDB(t *testing.T) *gorm.DB {
	t.Helper()

	_ = godotenv.Load("../../.env") // best-effort; CI uses real env vars

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set -- skipping integration test")
	}

	db, err := database.Connect2(dsn)
	if err != nil {
		t.Fatalf("testutil: db connect: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		t.Fatalf("testutil: migrate: %v", err)
	}

	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })

	return tx
}

// CleanUsers deletes all users (and cascades to oauth_accounts, refresh_tokens)
// by email prefix. Useful when a test must commit (e.g. testing refresh token rotation)
// and cannot rely on the rollback cleanup.
func CleanUsers(t *testing.T, db *gorm.DB, emails ...string) {
	t.Helper()
	for _, e := range emails {
		db.Where("email = ?", e).Delete(&model.User{})
	}
}
