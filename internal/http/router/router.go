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
	r.Static("/uploads", uploadDir)

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
	announcementHandler := handler.NewAnnouncementHandler(db, notifSvc)
	approvalHandler := handler.NewApprovalHandler(db)
	partyflowHandler := handler.NewPartyflowHandler(db)

	api.GET("/me", meHandler.GetMe)
	api.PATCH("/me", meHandler.PatchMe)
	api.GET("/profile/home", meHandler.GetProfileHome)
	api.GET("/knowledge/search", knowledgeHandler.Search)
	api.GET("/knowledge/:id", knowledgeHandler.GetByID)
	api.GET("/announcements", announcementHandler.List)
	api.GET("/announcements/all", announcementHandler.ListAllPublished)
	api.GET("/announcements/all/:id", announcementHandler.GetAllPublishedByID)
	api.GET("/announcements/:id", announcementHandler.GetByID)
	api.POST("/approvals", approvalHandler.Create)
	api.GET("/approvals/me", approvalHandler.ListMine)
	api.GET("/approvals/:id", approvalHandler.Get)
	api.POST("/approvals/:id/withdraw", approvalHandler.Withdraw)
	api.GET("/partyflow/me", partyflowHandler.GetMe)

	// File APIs
	api.POST("/files/upload", fileHandler.Upload)
	api.GET("/files/search", fileHandler.Search)
	api.GET("/files", fileHandler.List)
	api.GET("/files/:id", fileHandler.Get)
	api.GET("/files/:id/download", fileHandler.Download)
	api.DELETE("/files/:id", fileHandler.Delete)

	api.POST("/user/subscribe/report", subscribeHandler.ReportSubscribe)

	admin := api.Group("/admin")
	admin.GET("/users", adminUserHandler.ListUsers)
	admin.GET("/users/:id", adminUserHandler.GetUser)
	admin.PATCH("/users/:id", adminUserHandler.PatchUser)
	admin.POST("/users/import", adminUserHandler.ImportUsers)

	admin.GET("/classes", adminClassHandler.ListClasses)
	admin.GET("/classes/:id", adminClassHandler.GetClass)
	admin.POST("/classes", adminClassHandler.CreateClass)
	admin.PATCH("/classes/:id", adminClassHandler.PatchClass)

	admin.GET("/logs", adminLogHandler.ListLogs)
	admin.GET("/knowledge", adminKnowledgeHandler.ListKnowledge)
	admin.GET("/knowledge/:id", adminKnowledgeHandler.GetKnowledge)
	admin.POST("/knowledge", adminKnowledgeHandler.CreateKnowledge)
	admin.POST("/knowledge/qa-generate-preview", adminKnowledgeHandler.GenerateQAPreview)
	admin.POST("/knowledge/qa-generate-preview/stream", adminKnowledgeHandler.GenerateQAPreviewStream)
	admin.POST("/knowledge/batch", adminKnowledgeHandler.BatchCreateKnowledge)
	admin.POST("/knowledge/:id/attachments", adminKnowledgeHandler.BindAttachments)
	admin.GET("/knowledge/:id/attachments", adminKnowledgeHandler.ListAttachments)
	admin.DELETE("/knowledge/:id/attachments/:file_id", adminKnowledgeHandler.DeleteAttachment)
	admin.PATCH("/knowledge/:id", adminKnowledgeHandler.PatchKnowledge)
	admin.DELETE("/knowledge/:id", adminKnowledgeHandler.DeleteKnowledge)

	// Notification routes
	notifHandler := handler.NewNotificationHandler(notifSvc)

	admin.POST("/notification/templates", notifHandler.CreateTemplate)
	admin.GET("/notification/templates/:code", notifHandler.GetTemplate)
	admin.GET("/notification/logs", notifHandler.ListLogs)
	api.GET("/notifications/unread/count", notifHandler.UnreadCount)
	admin.GET("/announcements", announcementHandler.ListAdmin)
	admin.GET("/announcements/:id", announcementHandler.GetAdmin)
	admin.POST("/announcements", announcementHandler.Create)
	admin.PATCH("/announcements/:id", announcementHandler.Patch)
	admin.POST("/announcements/:id/publish", announcementHandler.Publish)
	admin.POST("/announcements/:id/archive", announcementHandler.Archive)
	admin.GET("/approvals", approvalHandler.ListAdmin)
	admin.POST("/approvals/:id/review", approvalHandler.Review)
	admin.POST("/approvals/:id/assign", approvalHandler.Assign)
	admin.POST("/approvals/:id/remind", approvalHandler.Remind)
	admin.POST("/approvals/scan-overdue", approvalHandler.ScanOverdue)
	admin.GET("/partyflow/statuses", partyflowHandler.ListAdminStatuses)
	admin.GET("/partyflow/statuses/:id", partyflowHandler.GetAdminStatus)
	admin.POST("/partyflow/statuses", partyflowHandler.CreateAdminStatus)
	admin.PATCH("/partyflow/statuses/:id", partyflowHandler.PatchAdminStatus)
	admin.POST("/partyflow/statuses/import", partyflowHandler.ImportAdminStatuses)
	admin.POST("/partyflow/statuses/:id/events", partyflowHandler.CreateAdminEvent)
	admin.GET("/partyflow/reminder-rules", partyflowHandler.ListReminderRules)
	admin.PATCH("/partyflow/reminder-rules/:id", partyflowHandler.PatchReminderRule)
	admin.POST("/partyflow/reminders/scan", partyflowHandler.ScanReminders)

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
