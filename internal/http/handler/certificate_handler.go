package handler

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/service/authz"
	certificates "manage/internal/service/certificates"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CertificateHandler struct {
	svc *certificates.Service
}

func NewCertificateHandler(db *gorm.DB) *CertificateHandler {
	return &CertificateHandler{svc: certificates.NewService(db)}
}

func (h *CertificateHandler) ListMine(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionCertificatesMyList) {
		response.Error(c, 403, "forbidden")
		return
	}

	limit, offset := parseCertPage(c)
	list, total, err := h.svc.ListMine(actor, strings.TrimSpace(c.Query("approval_type")), limit, offset)
	if err != nil {
		response.Error(c, 500, "query failed")
		return
	}
	response.List(c, list, total)
}

func (h *CertificateHandler) Get(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionCertificatesGet) {
		response.Error(c, 403, "forbidden")
		return
	}

	id, ok := parseCertID(c)
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}

	item, err := h.svc.Get(actor, id)
	if err != nil {
		h.writeSvcErr(c, err)
		return
	}
	response.OK(c, item)
}

func (h *CertificateHandler) Verify(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		response.Error(c, 400, "invalid verification code")
		return
	}

	out, err := h.svc.VerifyByCode(code)
	if err != nil {
		h.writeSvcErr(c, err)
		return
	}
	response.OK(c, out)
}

func (h *CertificateHandler) ListAdminTemplates(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionCertificatesTemplateAdminList) {
		response.Error(c, 403, "forbidden")
		return
	}
	list, err := h.svc.ListTemplates()
	if err != nil {
		response.Error(c, 500, "query failed")
		return
	}
	response.List(c, list, int64(len(list)))
}

func (h *CertificateHandler) ToggleTemplate(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionCertificatesTemplateToggle) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, ok := parseCertID(c)
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}
	item, err := h.svc.ToggleTemplate(id, strings.HasSuffix(c.Request.URL.Path, "/activate"))
	if err != nil {
		h.writeSvcErr(c, err)
		return
	}
	response.OK(c, item)
}

func (h *CertificateHandler) RegenerateApplicationPDF(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionCertificatesApplicationRegenerate) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, ok := parseCertID(c)
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}
	item, err := h.svc.RegenerateApplicationPDF(context.Background(), id)
	if err != nil {
		h.writeSvcErr(c, err)
		return
	}
	response.OK(c, item)
}

func (h *CertificateHandler) RegenerateApprovalCertificate(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionCertificatesCertificateRegenerate) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, ok := parseCertID(c)
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}
	item, err := h.svc.RegenerateApprovalCertificate(context.Background(), id)
	if err != nil {
		h.writeSvcErr(c, err)
		return
	}
	response.OK(c, item)
}

func (h *CertificateHandler) Revoke(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionCertificatesRevoke) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, ok := parseCertID(c)
	if !ok {
		response.Error(c, 400, "invalid id")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}
	item, err := h.svc.Revoke(context.Background(), id, req.Reason)
	if err != nil {
		h.writeSvcErr(c, err)
		return
	}
	response.OK(c, item)
}

func parseCertID(c *gin.Context) (uint, bool) {
	v, err := strconv.ParseUint(c.Param("id"), 10, 64)
	return uint(v), err == nil && v > 0
}

func parseCertPage(c *gin.Context) (int, int) {
	limit, offset := 20, 0
	if v, err := strconv.Atoi(strings.TrimSpace(c.Query("limit"))); err == nil && v > 0 {
		limit = v
	}
	if v, err := strconv.Atoi(strings.TrimSpace(c.Query("offset"))); err == nil && v >= 0 {
		offset = v
	}
	return limit, offset
}

func (h *CertificateHandler) writeSvcErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.Error(c, 404, "certificate not found")
	case errors.Is(err, certificates.ErrInvalidVerificationCode), errors.Is(err, certificates.ErrApprovalNotApproved):
		response.Error(c, 400, err.Error())
	default:
		response.Error(c, 500, "operation failed")
	}
}
