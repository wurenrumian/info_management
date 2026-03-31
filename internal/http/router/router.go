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

	meHandler := handler.NewMeHandler(db)
	adminUserHandler := handler.NewAdminUserHandler(db)
	adminClassHandler := handler.NewAdminClassHandler(db)
	adminLogHandler := handler.NewAdminLogHandler(db)

	api.GET("/me", meHandler.GetMe)

	admin := api.Group("/admin")
	admin.GET("/users", adminUserHandler.ListUsers)
	admin.GET("/users/:id", adminUserHandler.GetUser)
	admin.PATCH("/users/:id", adminUserHandler.PatchUser)

	admin.GET("/classes", adminClassHandler.ListClasses)
	admin.GET("/classes/:id", adminClassHandler.GetClass)
	admin.POST("/classes", adminClassHandler.CreateClass)
	admin.PATCH("/classes/:id", adminClassHandler.PatchClass)

	admin.GET("/logs", adminLogHandler.ListLogs)

	return r
}
