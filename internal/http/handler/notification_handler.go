package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/model"
	"manage/internal/service/authz"
	"manage/internal/service/notification"
)

// NotificationHandler handles HTTP requests for notification template and log management.
type NotificationHandler struct {
	svc *notification.Service
}

type createNotificationTemplateReq struct {
	Code             string `json:"code" binding:"required"`
	WechatTemplateID string `json:"wechat_template_id" binding:"required"`
	Name             string `json:"name" binding:"required"`
}

// NewNotificationHandler creates a NotificationHandler with the given service.
func NewNotificationHandler(svc *notification.Service) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// CreateTemplate handles POST /admin/notification/templates.
func (h *NotificationHandler) CreateTemplate(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionNotifTplCreate) {
		response.Error(c, 403, "forbidden")
		return
	}

	var req createNotificationTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid request")
		return
	}

	tmpl := &model.NotificationTemplate{
		Code:             req.Code,
		WechatTemplateID: req.WechatTemplateID,
		Name:             req.Name,
	}

	if err := h.svc.CreateTemplate(tmpl); err != nil {
		response.Error(c, 500, "create template failed")
		return
	}

	response.OK(c, tmpl)
}

// GetTemplate handles GET /admin/notification/templates/:code.
func (h *NotificationHandler) GetTemplate(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionNotifTplGet) {
		response.Error(c, 403, "forbidden")
		return
	}

	code := c.Param("code")
	tmpl, err := h.svc.GetTemplate(code)
	if err != nil {
		response.Error(c, 404, "template not found")
		return
	}
	response.OK(c, tmpl)
}

// ListLogs handles GET /admin/notification/logs.
func (h *NotificationHandler) ListLogs(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionNotifLogsList) {
		response.Error(c, 403, "forbidden")
		return
	}

	filter := notification.LogFilter{
		Limit: 20,
	}

	if uid := c.Query("user_id"); uid != "" {
		id, err := strconv.ParseUint(uid, 10, 32)
		if err == nil {
			uidUint := uint(id)
			filter.UserID = &uidUint
		}
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}
	if tmpl := c.Query("template_code"); tmpl != "" {
		filter.TemplateCode = &tmpl
	}
	if offset := c.Query("offset"); offset != "" {
		o, _ := strconv.Atoi(offset)
		filter.Offset = o
	}
	if limit := c.Query("limit"); limit != "" {
		l, _ := strconv.Atoi(limit)
		if l > 0 {
			filter.Limit = l
		}
	}

	logs, total, err := h.svc.GetLogs(filter)
	if err != nil {
		response.Error(c, 500, "list logs failed")
		return
	}

	response.OK(c, gin.H{
		"data":  logs,
		"total": total,
	})
}

// UnreadCount handles GET /notifications/unread/count.
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionNotifUnreadGet) {
		response.Error(c, 403, "forbidden")
		return
	}

	count, err := h.svc.GetUnreadCount(actor.UserID)
	if err != nil {
		response.Error(c, 500, "count unread notifications failed")
		return
	}

	response.OK(c, gin.H{"count": count})
}
