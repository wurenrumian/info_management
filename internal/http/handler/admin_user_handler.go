package handler

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/repo"
	"manage/internal/service/audit"
	"manage/internal/service/authz"
	gradesvc "manage/internal/service/grade"
)

type AdminUserHandler struct {
	userRepo    *repo.UserRepo
	auditLogger *audit.Logger
	gradeSvc    *gradesvc.Service
}

func NewAdminUserHandler(db *gorm.DB) *AdminUserHandler {
	return &AdminUserHandler{
		userRepo:    repo.NewUserRepo(db),
		auditLogger: audit.NewLogger(repo.NewAdminLogRepo(db)),
		gradeSvc:    gradesvc.NewService(db),
	}
}

func (h *AdminUserHandler) ListUsers(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionUsersList) {
		response.Error(c, 403, "forbidden")
		return
	}
	users, total, err := h.userRepo.ListByScopeWithTotal(authz.BuildScope(actor), 20, 0)
	if err != nil {
		response.Error(c, 500, "list users failed")
		return
	}
	response.List(c, users, total)
}

func (h *AdminUserHandler) GetUser(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionUsersGet) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "invalid id")
		return
	}
	user, err := h.userRepo.GetByIDInScope(authz.BuildScope(actor), uint(id))
	if err != nil {
		response.Error(c, 404, "user not found")
		return
	}
	response.OK(c, user)
}

type patchUserReq struct {
	Name         *string          `json:"name"`
	Role         *int             `json:"role"`
	ClassID      *uint            `json:"class_id"`
	Grade        *string          `json:"grade"`
	Major        *string          `json:"major"`
	ExtraAttrs   *json.RawMessage `json:"extra_attrs"`
	ProfileAttrs *json.RawMessage `json:"profile_attrs"`
}

func (h *AdminUserHandler) PatchUser(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionUsersPatch) {
		response.Error(c, 403, "forbidden")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "invalid id")
		return
	}
	var req patchUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}
	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Role != nil {
		updates["role"] = *req.Role
	}
	if req.ClassID != nil {
		updates["class_id"] = *req.ClassID
	}
	if req.Grade != nil {
		response.Error(c, 400, "grade is system-managed")
		return
	}
	if req.Major != nil {
		updates["major"] = *req.Major
	}
	if req.ExtraAttrs != nil {
		updates["extra_attrs"] = datatypes.JSON(*req.ExtraAttrs)
	}
	if req.ProfileAttrs != nil {
		updates["profile_attrs"] = datatypes.JSON(*req.ProfileAttrs)
	}
	if len(updates) == 0 {
		response.Error(c, 400, "empty patch")
		return
	}
	if err := h.userRepo.UpdateByID(uint(id), updates); err != nil {
		response.Error(c, 500, "update user failed")
		return
	}
	if req.ClassID != nil {
		if err := h.gradeSvc.SyncUserGradeByClassID(uint(id), *req.ClassID); err != nil {
			response.Error(c, 500, "sync user grade failed")
			return
		}
	}
	h.auditLogger.Log(c, actor, "users.patch", "user", uint(id))
	response.OK(c, gin.H{"updated": true})
}
