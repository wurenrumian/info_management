package handler

import (
	"net/http"
	"strconv"

	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/model"
	"manage/internal/service/authz"
	"manage/internal/service/notification"

	"github.com/gin-gonic/gin"
)

// NotificationHandler handles HTTP requests for notification template and log management.
type NotificationHandler struct {
	svc *notification.Service
}

// NewNotificationHandler creates a NotificationHandler with the given service.
func NewNotificationHandler(svc *notification.Service) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// CreateTemplate handles POST /admin/notification/templates.
func (h *NotificationHandler) CreateTemplate(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok || !authz.Authorize(actor.Role, authz.ActionNotifTplCreate) {
		response.Error(c, http.StatusForbidden, "forbidden")
		return
	}

	var req struct {
		Code             string `json:"code" binding:"required"`
		WechatTemplateID string `json:"wechat_template_id" binding:"required"`
		Name             string `json:"name" binding:"required"`
		Fields           string `json:"fields"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	tmpl := &model.NotificationTemplate{
		Code:             req.Code,
		WechatTemplateID: req.WechatTemplateID,
		Name:             req.Name,
		Fields:           req.Fields,
	}

	if err := h.svc.CreateTemplate(tmpl); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": tmpl})
}

// GetTemplate handles GET /admin/notification/templates/:code.
func (h *NotificationHandler) GetTemplate(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok || !authz.Authorize(actor.Role, authz.ActionNotifTplGet) {
		response.Error(c, http.StatusForbidden, "forbidden")
		return
	}

	code := c.Param("code")
	tmpl, err := h.svc.GetTemplate(code)
	if err != nil {
		response.Error(c, http.StatusNotFound, "template not found")
		return
	}
	response.OK(c, tmpl)
}

// ListLogs handles GET /admin/notification/logs.
func (h *NotificationHandler) ListLogs(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok || !authz.Authorize(actor.Role, authz.ActionNotifLogsList) {
		response.Error(c, http.StatusForbidden, "forbidden")
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
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, gin.H{
		"data":  logs,
		"total": total,
	})
}
