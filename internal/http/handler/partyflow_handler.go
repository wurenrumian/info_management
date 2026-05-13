package handler

import (
	"errors"
	"strconv"
	"strings"

	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/repo"
	"manage/internal/service/audit"
	"manage/internal/service/authz"
	partyflowsvc "manage/internal/service/partyflow"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PartyflowHandler handles partyflow APIs.
type PartyflowHandler struct {
	svc         *partyflowsvc.Service
	auditLogger *audit.Logger
}

// NewPartyflowHandler creates a handler instance.
func NewPartyflowHandler(db *gorm.DB) *PartyflowHandler {
	return &PartyflowHandler{
		svc:         partyflowsvc.NewService(db),
		auditLogger: audit.NewLogger(repo.NewAdminLogRepo(db)),
	}
}

// GetMe handles GET /api/v1/partyflow/me
func (h *PartyflowHandler) GetMe(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionPartyflowMeGet) {
		response.Error(c, 403, "forbidden")
		return
	}
	items, err := h.svc.ListMine(actor)
	if err != nil {
		response.Error(c, 500, "query partyflow failed")
		return
	}
	response.OK(c, items)
}

// ListAdminStatuses handles GET /api/v1/admin/partyflow/statuses
func (h *PartyflowHandler) ListAdminStatuses(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionPartyflowStatusesList) {
		response.Error(c, 403, "forbidden")
		return
	}
	limit, offset := 20, 0
	if raw := c.Query("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}
	if raw := c.Query("offset"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			offset = v
		}
	}

	items, total, err := h.svc.ListAdmin(actor, partyflowsvc.AdminListParams{
		OrgType:   c.Query("org_type"),
		Status:    c.Query("status"),
		StudentID: c.Query("student_id"),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		switch {
		case errors.Is(err, partyflowsvc.ErrInvalidOrgType), errors.Is(err, partyflowsvc.ErrInvalidStatus):
			response.Error(c, 400, err.Error())
		default:
			response.Error(c, 500, "query partyflow statuses failed")
		}
		return
	}
	response.List(c, items, total)
}

// GetAdminStatus handles GET /api/v1/admin/partyflow/statuses/:id
func (h *PartyflowHandler) GetAdminStatus(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionPartyflowStatusesGet) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, 400, "invalid id")
		return
	}
	item, err := h.svc.GetAdmin(actor, uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "partyflow status not found")
			return
		}
		response.Error(c, 500, "query partyflow status failed")
		return
	}
	response.OK(c, item)
}

// CreateAdminStatus handles POST /api/v1/admin/partyflow/statuses
func (h *PartyflowHandler) CreateAdminStatus(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionPartyflowStatusesCreate) {
		response.Error(c, 403, "forbidden")
		return
	}
	var req partyflowsvc.CreateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}
	item, err := h.svc.CreateStatus(actor, req)
	if err != nil {
		switch {
		case errors.Is(err, partyflowsvc.ErrInvalidOrgType), errors.Is(err, partyflowsvc.ErrInvalidStatus):
			response.Error(c, 400, err.Error())
		case errors.Is(err, gorm.ErrRecordNotFound):
			response.Error(c, 404, "user not found")
		default:
			response.Error(c, 500, "create partyflow status failed")
		}
		return
	}
	h.auditLogger.Log(c, actor, "partyflow.status_create", "partyflow_status", item.ID)
	response.OK(c, item)
}

// PatchAdminStatus handles PATCH /api/v1/admin/partyflow/statuses/:id
func (h *PartyflowHandler) PatchAdminStatus(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionPartyflowStatusesPatch) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, 400, "invalid id")
		return
	}
	var req partyflowsvc.PatchStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}
	item, err := h.svc.PatchStatus(actor, uint(id), req)
	if err != nil {
		switch {
		case errors.Is(err, partyflowsvc.ErrInvalidStatus):
			response.Error(c, 400, err.Error())
		case errors.Is(err, gorm.ErrRecordNotFound):
			response.Error(c, 404, "partyflow status not found")
		default:
			response.Error(c, 500, "patch partyflow status failed")
		}
		return
	}
	h.auditLogger.Log(c, actor, "partyflow.status_patch", "partyflow_status", uint(id))
	response.OK(c, item)
}

// ImportAdminStatuses handles POST /api/v1/admin/partyflow/statuses/import
func (h *PartyflowHandler) ImportAdminStatuses(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionPartyflowStatusesImport) {
		response.Error(c, 403, "forbidden")
		return
	}
	var req partyflowsvc.ImportStatusesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}
	result, err := h.svc.ImportStatuses(actor, req)
	if err != nil {
		response.Error(c, 500, "import partyflow statuses failed")
		return
	}
	h.auditLogger.Log(c, actor, "partyflow.status_import", "partyflow_status", 0)
	response.OK(c, result)
}

// CreateAdminEvent handles POST /api/v1/admin/partyflow/statuses/:id/events
func (h *PartyflowHandler) CreateAdminEvent(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionPartyflowEventsCreate) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, 400, "invalid id")
		return
	}
	var req partyflowsvc.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}
	item, err := h.svc.CreateEvent(actor, uint(id), req)
	if err != nil {
		switch {
		case errors.Is(err, partyflowsvc.ErrInvalidEventType):
			response.Error(c, 400, err.Error())
		case errors.Is(err, gorm.ErrRecordNotFound):
			response.Error(c, 404, "partyflow status not found")
		default:
			response.Error(c, 500, "create partyflow event failed")
		}
		return
	}
	h.auditLogger.Log(c, actor, "partyflow.event_create", "partyflow_status", uint(id))
	response.OK(c, item)
}

// ListReminderRules handles GET /api/v1/admin/partyflow/reminder-rules
func (h *PartyflowHandler) ListReminderRules(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionPartyflowRulesList) {
		response.Error(c, 403, "forbidden")
		return
	}
	var enabled *bool
	if raw := strings.TrimSpace(c.Query("enabled")); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			enabled = &v
		}
	}
	items, err := h.svc.ListRules(actor, partyflowsvc.RuleListParams{
		OrgType: c.Query("org_type"),
		Enabled: enabled,
	})
	if err != nil {
		if errors.Is(err, partyflowsvc.ErrInvalidOrgType) {
			response.Error(c, 400, err.Error())
			return
		}
		response.Error(c, 500, "query partyflow reminder rules failed")
		return
	}
	response.OK(c, items)
}

// PatchReminderRule handles PATCH /api/v1/admin/partyflow/reminder-rules/:id
func (h *PartyflowHandler) PatchReminderRule(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionPartyflowRulesPatch) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, 400, "invalid id")
		return
	}
	var req partyflowsvc.PatchRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}
	item, err := h.svc.PatchRule(actor, uint(id), req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "partyflow reminder rule not found")
			return
		}
		response.Error(c, 500, "patch partyflow reminder rule failed")
		return
	}
	h.auditLogger.Log(c, actor, "partyflow.rule_patch", "partyflow_rule", uint(id))
	response.OK(c, item)
}

// ScanReminders handles POST /api/v1/admin/partyflow/reminders/scan
func (h *PartyflowHandler) ScanReminders(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionPartyflowRemindersScan) {
		response.Error(c, 403, "forbidden")
		return
	}
	var req partyflowsvc.ScanRemindersRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.Error(c, 400, "invalid body")
		return
	}
	result, err := h.svc.ScanReminders(req)
	if err != nil {
		response.Error(c, 500, "scan partyflow reminders failed")
		return
	}
	h.auditLogger.Log(c, actor, "partyflow.reminder_scan", "partyflow_rule", 0)
	response.OK(c, result)
}
