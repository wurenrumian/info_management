package handler

import (
	"context"
	"strconv"
	"time"

	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/authz"
	"manage/internal/service/notification"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AnnouncementHandler struct {
	db       *gorm.DB
	repo     *repo.AnnouncementRepo
	notifSvc *notification.Service
}

func NewAnnouncementHandler(db *gorm.DB, notifSvc *notification.Service) *AnnouncementHandler {
	return &AnnouncementHandler{
		db:       db,
		repo:     repo.NewAnnouncementRepo(db),
		notifSvc: notifSvc,
	}
}

type CreateAnnouncementReq struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

func parseID(c *gin.Context) uint {
	id, _ := strconv.Atoi(c.Param("id"))
	return uint(id)
}

func (h *AnnouncementHandler) Create(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}

	if !authz.Authorize(actor.Role, "announcement:create") {
		response.Error(c, 403, "forbidden")
		return
	}

	var req CreateAnnouncementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid request")
		return
	}

	a := model.Announcement{
		Title:     req.Title,
		Content:   req.Content,
		Status:    "draft",
		CreatedBy: actor.UserID,
	}

	if err := h.repo.Create(&a); err != nil {
		response.Error(c, 500, "create failed")
		return
	}

	response.OK(c, a)
}

func (h *AnnouncementHandler) List(c *gin.Context) {
	list, total, err := h.repo.ListWithTotal("published", 20, 0)
	if err != nil {
		response.Error(c, 500, "query failed")
		return
	}
	response.List(c, list, total)
}

func (h *AnnouncementHandler) Publish(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}

	if !authz.Authorize(actor.Role, "announcement:publish") {
		response.Error(c, 403, "forbidden")
		return
	}

	id := parseID(c)

	now := time.Now()

	err := h.repo.UpdateByID(id, map[string]any{
		"status":       "published",
		"published_at": now,
	})
	if err != nil {
		response.Error(c, 500, "publish failed")
		return
	}

	a, _ := h.repo.GetByID(id)

	go h.sendNotification(a)

	response.OK(c, gin.H{"ok": true})
}

func (h *AnnouncementHandler) sendNotification(a *model.Announcement) {
	var subs []model.UserSubscribe

	h.db.Where("template_code = ? AND status = ?", "announcement_publish", "subscribed").
		Find(&subs)

	for _, sub := range subs {

		if sub.GrantedCount-sub.ConsumedCount <= 0 {
			continue
		}

		err := h.notifSvc.Send(context.Background(), notification.SendRequest{
			UserID:       sub.UserID,
			TemplateCode: "announcement_publish",
			Page:         "/pages/announcement/detail?id=" + strconv.Itoa(int(a.ID)),
			TemplateData: map[string]interface{}{
				"thing1": map[string]string{"value": a.Title},
				"time2":  map[string]string{"value": a.PublishedAt.Format("2006-01-02 15:04")},
			},
		})

		if err == nil {
			h.db.Model(&model.UserSubscribe{}).
				Where("id = ?", sub.ID).
				Update("consumed_count", gorm.Expr("consumed_count + 1"))
		}
	}
}
