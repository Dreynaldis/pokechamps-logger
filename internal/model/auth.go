package model

import "time"

type User struct {
	ID           string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Email        string    `gorm:"uniqueIndex;not null"`
	PasswordHash string    // null for OAuth-only accounts
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// OAuthAccount links a User to a third-party provider identity.
// One user can have multiple rows (Google + Discord both linked).
type OAuthAccount struct {
	ID                string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID            string    `gorm:"type:uuid;not null;index"`
	Provider          string    `gorm:"not null"` // "google" | "discord"
	ProviderAccountID string    `gorm:"not null"` // stable ID from the provider
	CreatedAt         time.Time
}

// RefreshToken stores a bcrypt hash of the raw token, never the raw value.
// Deleted on logout or rotation.
type RefreshToken struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    string    `gorm:"type:uuid;not null;index"`
	TokenHash string    `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}
