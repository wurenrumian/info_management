package handler

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"

	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/repo"
	annsvc "manage/internal/service/announcements"
	"manage/internal/service/audit"
	"manage/internal/service/authz"
	"manage/internal/service/notification"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AnnouncementHandler handles announcement related HTTP requests.
type AnnouncementHandler struct {
	svc         *annsvc.Service
	auditLogger *audit.Logger
}

// NewAnnouncementHandler creates a new AnnouncementHandler.
func NewAnnouncementHandler(db *gorm.DB, notifSvc *notification.Service) *AnnouncementHandler {
	return &AnnouncementHandler{
		svc:         annsvc.NewService(db, notifSvc),
		auditLogger: audit.NewLogger(repo.NewAdminLogRepo(db)),
	}
}

// CreateAnnouncementReq defines the request structure for creating an announcement.
type CreateAnnouncementReq struct {
	Title             string           `json:"title" binding:"required"`
	Content           string           `json:"content" binding:"required"`
	AudienceType      string           `json:"audience_type"`
	TargetScope       map[string]any   `json:"target_scope"`
	Tags              []string         `json:"tags"`
	AttachmentFileIDs []uint           `json:"attachment_file_ids"`
	ExternalLinks     []map[string]any `json:"external_links"`
}

type PublishAnnouncementReq struct {
	SendNotification bool   `json:"send_notification"`
	TemplateCode     string `json:"template_code"`
}

type PatchAnnouncementReq struct {
	Title             *string           `json:"title"`
	Content           *string           `json:"content"`
	AudienceType      *string           `json:"audience_type"`
	TargetScope       *map[string]any   `json:"target_scope"`
	Tags              *[]string         `json:"tags"`
	AttachmentFileIDs *[]uint           `json:"attachment_file_ids"`
	ExternalLinks     *[]map[string]any `json:"external_links"`
}

func parseUintParam(c *gin.Context, key string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(key), 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

func parsePagination(c *gin.Context) (int, int) {
	limit := 20
	offset := 0
	if s := c.Query("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			limit = v
		}
	}
	if s := c.Query("offset"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			offset = v
		}
	}
	return limit, offset
}

// Create creates a new announcement.
func (h *AnnouncementHandler) Create(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}

	// Check authorization
	if !authz.Authorize(actor.Role, authz.ActionAnnouncementsCreate) {
		response.Error(c, 403, "forbidden")
		return
	}

	var req CreateAnnouncementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid request")
		return
	}

	item, err := h.svc.Create(actor, annsvc.CreateRequest{
		Title:             req.Title,
		Content:           req.Content,
		AudienceType:      req.AudienceType,
		TargetScope:       mustMarshalJSON(req.TargetScope),
		Tags:              mustMarshalJSON(req.Tags),
		AttachmentFileIDs: mustMarshalJSON(req.AttachmentFileIDs),
		ExternalLinks:     mustMarshalJSON(req.ExternalLinks),
	})
	if err != nil {
		if err == annsvc.ErrInvalidAudienceType {
			response.Error(c, 400, err.Error())
			return
		}
		response.Error(c, 500, "create failed")
		return
	}
	h.auditLogger.Log(c, actor, "announcements.create", "announcement", item.ID)
	response.OK(c, item)
}

// List fetches the list of published announcements.
func (h *AnnouncementHandler) List(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionAnnouncementsList) {
		response.Error(c, 403, "forbidden")
		return
	}
	limit, offset := parsePagination(c)
	log.Printf("[announcements] list request: user_id=%d role=%d limit=%d offset=%d", actor.UserID, actor.Role, limit, offset)
	list, total, err := h.svc.ListForStudent(actor, limit, offset)
	if err != nil {
		log.Printf("[announcements] list error: user_id=%d err=%v", actor.UserID, err)
		response.Error(c, 500, "query failed")
		return
	}
	log.Printf("[announcements] list success: user_id=%d total=%d", actor.UserID, total)
	response.List(c, list, total)
}

// ListAllPublished fetches all published announcements without audience scope filtering.
// Access is limited to privileged roles.
func (h *AnnouncementHandler) ListAllPublished(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionAnnouncementsListAll) {
		response.Error(c, 403, "forbidden")
		return
	}
	limit, offset := parsePagination(c)
	list, total, err := h.svc.ListAllPublished(limit, offset)
	if err != nil {
		response.Error(c, 500, "query failed")
		return
	}
	response.List(c, list, total)
}

// GetAllPublishedByID returns one published announcement without audience scope filtering.
// Access is limited to privileged roles.
func (h *AnnouncementHandler) GetAllPublishedByID(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionAnnouncementsGetAll) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}
	item, err := h.svc.GetAllPublishedByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "announcement not found")
			return
		}
		response.Error(c, 500, "query failed")
		return
	}
	response.OK(c, item)
}

// GetByID returns one published announcement for student side.
func (h *AnnouncementHandler) GetByID(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionAnnouncementsGet) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}
	item, err := h.svc.GetForStudent(actor, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "announcement not found")
			return
		}
		response.Error(c, 500, "query failed")
		return
	}
	response.OK(c, item)
}

// ListAdmin lists announcements for admin side.
func (h *AnnouncementHandler) ListAdmin(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionAnnouncementsAdminList) {
		response.Error(c, 403, "forbidden")
		return
	}
	limit, offset := parsePagination(c)
	list, total, err := h.svc.ListForAdmin(annsvc.ListRequest{
		Status: c.Query("status"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		if errors.Is(err, annsvc.ErrInvalidStatus) {
			response.Error(c, 400, err.Error())
			return
		}
		response.Error(c, 500, "query failed")
		return
	}
	response.List(c, list, total)
}

// GetAdmin returns one announcement for admin side.
func (h *AnnouncementHandler) GetAdmin(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionAnnouncementsAdminGet) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}
	item, err := h.svc.GetForAdmin(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "announcement not found")
			return
		}
		response.Error(c, 500, "query failed")
		return
	}
	response.OK(c, item)
}

// Patch updates one announcement.
func (h *AnnouncementHandler) Patch(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionAnnouncementsPatch) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}

	var req PatchAnnouncementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid request")
		return
	}

	updates := map[string]any{}
	if req.Title != nil {
		updates["title"] = strings.TrimSpace(*req.Title)
	}
	if req.Content != nil {
		updates["content"] = strings.TrimSpace(*req.Content)
	}
	if req.AudienceType != nil {
		v := strings.TrimSpace(*req.AudienceType)
		if v != annsvc.AudienceAll && v != annsvc.AudienceTargeted {
			response.Error(c, 400, annsvc.ErrInvalidAudienceType.Error())
			return
		}
		updates["audience_type"] = v
		if v == annsvc.AudienceAll {
			updates["target_scope"] = mustMarshalJSON(map[string]any{})
		}
	}
	if req.TargetScope != nil {
		updates["target_scope"] = mustMarshalJSON(*req.TargetScope)
	}
	if req.Tags != nil {
		updates["tags"] = mustMarshalJSON(*req.Tags)
	}
	if req.AttachmentFileIDs != nil {
		updates["attachment_file_ids"] = mustMarshalJSON(*req.AttachmentFileIDs)
	}
	if req.ExternalLinks != nil {
		updates["external_links"] = mustMarshalJSON(*req.ExternalLinks)
	}
	if len(updates) == 0 {
		response.Error(c, 400, "invalid request")
		return
	}

	if err := h.svc.Patch(id, updates); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "announcement not found")
			return
		}
		response.Error(c, 500, "patch failed")
		return
	}
	item, err := h.svc.GetForAdmin(id)
	if err != nil {
		response.Error(c, 500, "query failed")
		return
	}
	h.auditLogger.Log(c, actor, "announcements.patch", "announcement", id)
	response.OK(c, item)
}

// Publish publishes an announcement.
func (h *AnnouncementHandler) Publish(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}

	// Check authorization
	if !authz.Authorize(actor.Role, authz.ActionAnnouncementsPublish) {
		response.Error(c, 403, "forbidden")
		return
	}

	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}
	var req PublishAnnouncementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// Keep compatibility: allow empty body.
		req = PublishAnnouncementReq{}
	}
	item, notifSummary, err := h.svc.Publish(c.Request.Context(), id, annsvc.PublishRequest{
		SendNotification: req.SendNotification,
		TemplateCode:     req.TemplateCode,
	})
	if err != nil {
		response.Error(c, 500, "publish failed")
		return
	}
	h.auditLogger.Log(c, actor, "announcements.publish", "announcement", item.ID)
	response.OK(c, gin.H{
		"id":                   item.ID,
		"status":               item.Status,
		"published_at":         item.PublishedAt,
		"notification_summary": notifSummary,
	})
}

// Archive marks one announcement as archived.
func (h *AnnouncementHandler) Archive(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionAnnouncementsArchive) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}
	if err := h.svc.Archive(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "announcement not found")
			return
		}
		response.Error(c, 500, "archive failed")
		return
	}
	item, err := h.svc.GetForAdmin(id)
	if err != nil {
		response.Error(c, 500, "query failed")
		return
	}
	h.auditLogger.Log(c, actor, "announcements.archive", "announcement", id)
	response.OK(c, gin.H{
		"id":         item.ID,
		"status":     item.Status,
		"updated_at": item.UpdatedAt,
	})
}

func mustMarshalJSON(v any) []byte {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}
