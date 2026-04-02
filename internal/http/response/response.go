package response

import "github.com/gin-gonic/gin"

func OK(c *gin.Context, data any) {
	c.JSON(200, gin.H{"data": data})
}

func List(c *gin.Context, data any, total int64) {
	c.JSON(200, gin.H{"data": data, "total": total})
}

func Error(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}
