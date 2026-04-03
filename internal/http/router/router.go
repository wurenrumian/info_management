package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"manage/internal/config"
	"manage/internal/http/handler"
	"manage/internal/http/middleware"
	"manage/internal/service/notification"
)

func New(db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", handler.Health)

	// Unified static file serving
	uploadDir := config.DocumentUploadDir()
	r.Static("/uploads/documents", uploadDir)

	// Backward compat: knowledge uploads
	knowledgeDir := config.KnowledgeUploadDir()
	if knowledgeDir != "" {
		r.Static("/uploads/knowledge", knowledgeDir)
	}

	jwtSecret := config.JWTSecret()
	appID := config.WechatAppID()
	appSecret := config.WechatAppSecret()

	api := r.Group("/api/v1")

	notifSvc := initNotificationSvc(db)
	wechatHandler := handler.NewWechatHandler(db, appID, appSecret, jwtSecret, notifSvc)
	api.POST("/auth/public-register", wechatHandler.PublicRegister)
	api.POST("/wechat/login", wechatHandler.Login)
	api.POST("/wechat/bind", middleware.OptionalJWTAuth(jwtSecret), wechatHandler.Bind)
	api.POST("/dev/register-or-login", wechatHandler.DevRegisterOrLogin)
	api.POST("/dev/login-and-send-subscribe-check", wechatHandler.DevLoginAndSendSubscribeCheck)

	subscribeHandler := handler.NewSubscribeHandler(db)
	api.POST("/wechat/callback", subscribeHandler.WechatCallback)

	api.Use(middleware.JWTAuth(jwtSecret))

	meHandler := handler.NewMeHandler(db)
	knowledgeHandler := handler.NewKnowledgeHandler(db)
	adminUserHandler := handler.NewAdminUserHandler(db)
	adminClassHandler := handler.NewAdminClassHandler(db)
	adminLogHandler := handler.NewAdminLogHandler(db)
	adminKnowledgeHandler := handler.NewAdminKnowledgeHandler(db)
	fileHandler := handler.NewFileHandler(db)

	api.GET("/me", meHandler.GetMe)
	api.PATCH("/me", meHandler.PatchMe)
	api.GET("/profile/home", meHandler.GetProfileHome)
	api.GET("/knowledge/search", knowledgeHandler.Search)

	// File APIs
	api.POST("/files/upload", fileHandler.Upload)
	api.GET("/files", fileHandler.List)
	api.GET("/files/:id", fileHandler.Get)
	api.GET("/files/:id/download", fileHandler.Download)
	api.DELETE("/files/:id", fileHandler.Delete)

	api.POST("/user/subscribe/report", subscribeHandler.ReportSubscribe)

	admin := api.Group("/admin")
	admin.GET("/users", adminUserHandler.ListUsers)
	admin.GET("/users/:id", adminUserHandler.GetUser)
	admin.PATCH("/users/:id", adminUserHandler.PatchUser)

	admin.GET("/classes", adminClassHandler.ListClasses)
	admin.GET("/classes/:id", adminClassHandler.GetClass)
	admin.POST("/classes", adminClassHandler.CreateClass)
	admin.PATCH("/classes/:id", adminClassHandler.PatchClass)

	admin.GET("/logs", adminLogHandler.ListLogs)
	admin.GET("/knowledge", adminKnowledgeHandler.ListKnowledge)
	admin.GET("/knowledge/:id", adminKnowledgeHandler.GetKnowledge)
	admin.POST("/knowledge", adminKnowledgeHandler.CreateKnowledge)
	admin.POST("/knowledge/import", adminKnowledgeHandler.ImportKnowledge)
	admin.PATCH("/knowledge/:id", adminKnowledgeHandler.PatchKnowledge)
	admin.DELETE("/knowledge/:id", adminKnowledgeHandler.DeleteKnowledge)

	// Notification routes
	notifHandler := handler.NewNotificationHandler(notifSvc)

	admin.POST("/notification/templates", notifHandler.CreateTemplate)
	admin.GET("/notification/templates/:code", notifHandler.GetTemplate)
	admin.GET("/notification/logs", notifHandler.ListLogs)
	api.GET("/notifications/unread/count", notifHandler.UnreadCount)

	return r
}

func initNotificationSvc(db *gorm.DB) *notification.Service {
	appID := config.WechatAppID()
	appSecret := config.WechatAppSecret()
	tokenCache := notification.NewTokenCache(appID, appSecret, nil)
	wechatClient := notification.NewWechatClient(nil, tokenCache)
	repo := notification.NewRepo(db)
	userRepo := notification.NewGormUserRepo(db)
	return notification.NewService(wechatClient, repo, userRepo)
}
