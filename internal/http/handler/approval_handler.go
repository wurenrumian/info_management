package handler

import (
	"encoding/json"
	"errors"
	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/repo"
	approvals "manage/internal/service/approvals"
	"manage/internal/service/audit"
	"manage/internal/service/authz"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ApprovalHandler struct {
	svc         *approvals.Service
	auditLogger *audit.Logger
}

func NewApprovalHandler(db *gorm.DB) *ApprovalHandler {
	return &ApprovalHandler{
		svc:         approvals.NewService(db),
		auditLogger: audit.NewLogger(repo.NewAdminLogRepo(db)),
	}
}

type createApprovalReq struct {
	ApprovalType      string          `json:"approval_type" binding:"required"`
	Title             string          `json:"title" binding:"required"`
	FormData          json.RawMessage `json:"form_data" binding:"required"`
	AttachmentFileIDs []uint          `json:"attachment_file_ids"`
	TemplateFileID    *uint           `json:"template_file_id"`
	Semester          string          `json:"semester"`
}
type reviewApprovalReq struct {
	Action  string `json:"action" binding:"required"`
	Comment string `json:"comment"`
}
type assignApprovalReq struct {
	ApproverID uint   `json:"approver_id" binding:"required"`
	Comment    string `json:"comment"`
}
type remindApprovalReq struct {
	Comment string `json:"comment"`
}

func (h *ApprovalHandler) Create(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionApprovalsCreate) {
		response.Error(c, 403, "forbidden")
		return
	}

	var req createApprovalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid request")
		return
	}

	item, err := h.svc.Create(actor, approvals.CreateRequest{
		ApprovalType: req.ApprovalType,
		Title:        req.Title,
		FormData:     req.FormData,
		AttachmentFileIDs: req.AttachmentFileIDs,
		TemplateFileID:    req.TemplateFileID,
		Semester:          req.Semester,
	})
	if err != nil {
		h.writeSvcErr(c, err)
		return
	}
	response.OK(c, item)
}

func (h *ApprovalHandler) ListMine(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionApprovalsMyList) {
		response.Error(c, 403, "forbidden")
		return
	}
	limit, offset := parsePage(c)
	list, total, err := h.svc.ListMine(actor, limit, offset)
	if err != nil {
		response.Error(c, 500, "query failed")
		return
	}
	response.List(c, list, total)
}

func (h *ApprovalHandler) Get(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionApprovalsGet) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}
	detail, err := h.svc.Get(actor, id)
	if err != nil {
		h.writeSvcErr(c, err)
		return
	}
	response.OK(c, detail)
}

func (h *ApprovalHandler) Withdraw(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionApprovalsWithdraw) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}
	if err := h.svc.Withdraw(actor, id); err != nil {
		h.writeSvcErr(c, err)
		return
	}
	response.OK(c, gin.H{"withdrawn": true})
}

func (h *ApprovalHandler) ListAdmin(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionApprovalsList) {
		response.Error(c, 403, "forbidden")
		return
	}
	limit, offset := parsePage(c)
	list, total, err := h.svc.ListAdmin(actor, approvals.ListAdminRequest{
		Status: c.Query("status"), ApprovalType: c.Query("approval_type"), Limit: limit, Offset: offset,
	})
	if err != nil {
		h.writeSvcErr(c, err)
		return
	}
	response.List(c, list, total)
}

func (h *ApprovalHandler) Review(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionApprovalsReview) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}
	var req reviewApprovalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid request")
		return
	}
	if err := h.svc.Review(actor, id, approvals.ReviewRequest{Action: req.Action, Comment: req.Comment}); err != nil {
		h.writeSvcErr(c, err)
		return
	}
	h.auditLogger.Log(c, actor, "approvals.review", "approval", id)
	response.OK(c, gin.H{"reviewed": true})
}

func (h *ApprovalHandler) Assign(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionApprovalsAssign) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}
	var req assignApprovalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid request")
		return
	}
	if err := h.svc.Assign(actor, id, approvals.AssignRequest{ApproverID: req.ApproverID, Comment: req.Comment}); err != nil {
		h.writeSvcErr(c, err)
		return
	}
	h.auditLogger.Log(c, actor, "approvals.assign", "approval", id)
	response.OK(c, gin.H{"assigned": true})
}

func (h *ApprovalHandler) Remind(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionApprovalsRemind) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}
	var req remindApprovalReq
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.Remind(actor, id, req.Comment); err != nil {
		h.writeSvcErr(c, err)
		return
	}
	h.auditLogger.Log(c, actor, "approvals.remind", "approval", id)
	response.OK(c, gin.H{"reminded": true})
}

func (h *ApprovalHandler) ScanOverdue(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionApprovalsOverdueScan) {
		response.Error(c, 403, "forbidden")
		return
	}
	out, err := h.svc.ScanAndRemindOverdue(c.Request.Context(), time.Now())
	if err != nil {
		response.Error(c, 500, "scan failed")
		return
	}
	h.auditLogger.Log(c, actor, "approvals.overdue.scan", "approval", 0)
	response.OK(c, out)
}

func parseID(c *gin.Context, key string) (uint, bool) {
	v, err := strconv.ParseUint(c.Param(key), 10, 64)
	return uint(v), err == nil && v > 0
}
func parsePage(c *gin.Context) (int, int) {
	limit, offset := 20, 0
	if v, err := strconv.Atoi(strings.TrimSpace(c.Query("limit"))); err == nil && v > 0 {
		limit = v
	}
	if v, err := strconv.Atoi(strings.TrimSpace(c.Query("offset"))); err == nil && v >= 0 {
		offset = v
	}
	return limit, offset
}
func (h *ApprovalHandler) writeSvcErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.Error(c, 404, "approval not found")
	case errors.Is(err, approvals.ErrForbidden):
		response.Error(c, 403, "forbidden")
	case errors.Is(err, approvals.ErrInvalidApprovalType),
		errors.Is(err, approvals.ErrInvalidFormData),
		errors.Is(err, approvals.ErrInvalidState):
		response.Error(c, 400, err.Error())
	default:
		response.Error(c, 500, "operation failed")
	}
}
