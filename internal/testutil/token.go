package testutil

import (
	"os"

	jwtauth "manage/internal/service/auth"
)

func GenerateTestToken(userID uint, role int, classID uint, grade string) string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}
	token, _ := jwtauth.GenerateToken(userID, role, classID, grade, secret)
	return token
}
