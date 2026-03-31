package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"manage/internal/http/handler"
)

func New(db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", handler.Health)
	return r
}
