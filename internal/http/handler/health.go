package handler

import (
	"github.com/gin-gonic/gin"
	"manage/internal/http/response"
)

func Health(c *gin.Context) {
	response.OK(c, gin.H{"status": "ok"})
}
