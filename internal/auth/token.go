package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dreynaldis/pokechamps-logger/internal/config"
	"github.com/dreynaldis/pokechamps-logger/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	AccessTokenTTL  = 24 * time.Hour
	RefreshTokenTTL = 30 * 24 * time.Hour
	bcryptCost      = 12
	refreshCookie   = "refresh_token"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"sub"`
}

// IssueTokens creates a JWT access token and a refresh token, persists the
// refresh token hash to the DB, and writes the refresh cookie to the response.
// Returns the signed access token string for the response body.
func IssueTokens(w http.ResponseWriter, db *gorm.DB, cfg *config.Config, userID string) (string, error) {
	// Access token -- signed JWT, validated by signature alone (no DB hit)
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.AuthSecret))
	if err != nil {
		return "", err
	}

	// Refresh token -- 32 random bytes, stored as bcrypt hash, sent as HttpOnly cookie
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	rawHex := hex.EncodeToString(raw)

	hash, err := bcrypt.GenerateFromPassword([]byte(rawHex), bcryptCost)
	if err != nil {
		return "", err
	}

	rt := model.RefreshToken{
		UserID:    userID,
		TokenHash: string(hash),
		ExpiresAt: time.Now().Add(RefreshTokenTTL),
	}
	if err := db.Create(&rt).Error; err != nil {
		return "", err
	}

	// Cookie value encodes the row ID for O(1) DB lookup on refresh/logout,
	// plus the raw token for the single bcrypt comparison. Format: {id}:{rawHex}
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookie,
		Value:    rt.ID + ":" + rawHex,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  rt.ExpiresAt,
	})

	return accessToken, nil
}

// ParseRefreshCookie splits the cookie into (tokenID, rawHex).
// Cookie format is "{uuid}:{64-char hex}" -- the ID enables O(1) DB lookup
// before the single bcrypt comparison.
func ParseRefreshCookie(r *http.Request) (tokenID, raw string, err error) {
	c, err := r.Cookie(refreshCookie)
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(c.Value, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("malformed refresh cookie")
	}
	return parts[0], parts[1], nil
}

// ClearRefreshCookie overwrites the cookie with an expired one.
func ClearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

// ParseAccessToken validates a JWT string and returns the claims.
func ParseAccessToken(tokenStr string, cfg *config.Config) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(cfg.AuthSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenSignatureInvalid
	}
	return claims, nil
}
