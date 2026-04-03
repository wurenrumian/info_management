package testutil

import (
	"manage/internal/config"
	jwtauth "manage/internal/service/auth"
)

func GenerateTestToken(userID uint, role int, classID uint, grade string) string {
	secret := config.JWTSecret()
	token, _ := jwtauth.GenerateToken(userID, role, classID, grade, secret)
	return token
}
