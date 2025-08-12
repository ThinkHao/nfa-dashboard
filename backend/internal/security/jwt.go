package security

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"nfa-dashboard/config"
)

// Claims defines custom JWT claims
// Note: use jwt.RegisteredClaims for v5
// We include a token version for future invalidation strategy

type Claims struct {
	UserID       uint64 `json:"user_id"`
	Username     string `json:"username"`
	TokenVersion int    `json:"token_version"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT with the given TTL (minutes)
func GenerateToken(userID uint64, username string, ttlMinutes int) (string, error) {
	secret := []byte(config.GetJWTSecret())
	now := time.Now()
	expires := now.Add(time.Duration(ttlMinutes) * time.Minute)
	claims := Claims{
		UserID:       userID,
		Username:     username,
		TokenVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
			Issuer:    "nfa-dashboard",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseToken validates and parses the token returning Claims
func ParseToken(tokenString string) (*Claims, error) {
	secret := []byte(config.GetJWTSecret())
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}
