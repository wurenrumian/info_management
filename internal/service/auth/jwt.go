package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents custom JWT claims used by this project.
type Claims struct {
	UserID  uint   `json:"sub"`
	Role    int    `json:"role"`
	ClassID uint   `json:"class_id"`
	Grade   string `json:"grade"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT for the given user identity.
func GenerateToken(userID uint, role int, classID uint, grade string, secret string) (string, error) {
	claims := Claims{
		UserID:  userID,
		Role:    role,
		ClassID: classID,
		Grade:   grade,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken validates a signed JWT and returns parsed claims.
func ParseToken(tokenString string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
