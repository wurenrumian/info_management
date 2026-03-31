package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"manage/internal/http/handler"
	"manage/internal/http/middleware"
)

func New(db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", handler.Health)
	api := r.Group("/api/v1")
	api.Use(middleware.IdentityFromHeaders())
	return r
}
