package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"manage/internal/auth"
)

func IdentityFromHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, ok := auth.ParseUintHeader(c, "X-User-Id")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing X-User-Id"})
			c.Abort()
			return
		}

		roleStr := c.GetHeader("X-User-Role")
		role, err := strconv.Atoi(roleStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid X-User-Role"})
			c.Abort()
			return
		}

		classID, _ := auth.ParseUintHeader(c, "X-User-Class-Id")
		actor := auth.Actor{
			UserID:  uid,
			Role:    role,
			ClassID: classID,
			Grade:   c.GetHeader("X-User-Grade"),
		}
		auth.SetActor(c, actor)
		c.Next()
	}
}

