package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"manage/internal/auth"
	jwtauth "manage/internal/service/auth"
)

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			c.Abort()
			return
		}

		claims, err := jwtauth.ParseToken(parts[1], secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		actor := auth.Actor{
			UserID:  claims.UserID,
			Role:    claims.Role,
			ClassID: claims.ClassID,
			Grade:   claims.Grade,
		}
		auth.SetActor(c, actor)
		c.Next()
	}
}

func OptionalJWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		claims, err := jwtauth.ParseToken(parts[1], secret)
		if err != nil {
			c.Next()
			return
		}

		actor := auth.Actor{
			UserID:  claims.UserID,
			Role:    claims.Role,
			ClassID: claims.ClassID,
			Grade:   claims.Grade,
		}
		auth.SetActor(c, actor)
		c.Next()
	}
}
