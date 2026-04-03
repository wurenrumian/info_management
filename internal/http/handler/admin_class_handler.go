package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/audit"
	"manage/internal/service/authz"
	gradesvc "manage/internal/service/grade"
)

type AdminClassHandler struct {
	classRepo   *repo.ClassRepo
	auditLogger *audit.Logger
	gradeSvc    *gradesvc.Service
}

func NewAdminClassHandler(db *gorm.DB) *AdminClassHandler {
	return &AdminClassHandler{
		classRepo:   repo.NewClassRepo(db),
		auditLogger: audit.NewLogger(repo.NewAdminLogRepo(db)),
		gradeSvc:    gradesvc.NewService(db),
	}
}

func (h *AdminClassHandler) ListClasses(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionClassesList) {
		response.Error(c, 403, "forbidden")
		return
	}
	items, total, err := h.classRepo.ListByScopeWithTotal(authz.BuildScope(actor), 20, 0)
	if err != nil {
		response.Error(c, 500, "list classes failed")
		return
	}
	response.List(c, items, total)
}

func (h *AdminClassHandler) GetClass(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionClassesGet) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "invalid id")
		return
	}
	item, err := h.classRepo.GetByIDInScope(authz.BuildScope(actor), uint(id))
	if err != nil {
		response.Error(c, 404, "class not found")
		return
	}
	response.OK(c, item)
}

type createClassReq struct {
	ClassName string `json:"class_name"`
	Grade     string `json:"grade"`
	Major     string `json:"major"`
}

func (h *AdminClassHandler) CreateClass(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionClassesCreate) {
		response.Error(c, 403, "forbidden")
		return
	}
	var req createClassReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}
	if req.ClassName == "" || req.Grade == "" || req.Major == "" {
		response.Error(c, 400, "missing fields")
		return
	}
	item := model.Class{ClassName: req.ClassName, Grade: req.Grade, Major: req.Major}
	if err := h.classRepo.Create(&item); err != nil {
		response.Error(c, 500, "create class failed")
		return
	}
	h.auditLogger.Log(c, actor, "classes.create", "class", item.ID)
	response.OK(c, item)
}

type patchClassReq struct {
	ClassName *string `json:"class_name"`
	Grade     *string `json:"grade"`
	Major     *string `json:"major"`
}

func (h *AdminClassHandler) PatchClass(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionClassesPatch) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "invalid id")
		return
	}
	before, err := h.classRepo.GetByIDInScope(authz.BuildScope(actor), uint(id))
	if err != nil {
		response.Error(c, 404, "class not found")
		return
	}
	var req patchClassReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}
	updates := map[string]any{}
	if req.ClassName != nil {
		updates["class_name"] = *req.ClassName
	}
	if req.Grade != nil {
		updates["grade"] = *req.Grade
	}
	if req.Major != nil {
		updates["major"] = *req.Major
	}
	if len(updates) == 0 {
		response.Error(c, 400, "empty patch")
		return
	}
	if err := h.classRepo.UpdateByID(uint(id), updates); err != nil {
		response.Error(c, 500, "update class failed")
		return
	}
	if req.Grade != nil && *req.Grade != before.Grade {
		if _, err := h.gradeSvc.SyncUsersGradeByClassID(uint(id)); err != nil {
			response.Error(c, 500, "sync users grade failed")
			return
		}
	}
	h.auditLogger.Log(c, actor, "classes.patch", "class", uint(id))
	response.OK(c, gin.H{"updated": true})
}
