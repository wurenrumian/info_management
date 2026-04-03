package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/repo"
	"manage/internal/service/authz"
)

type AdminLogHandler struct {
	logRepo *repo.AdminLogRepo
}

func NewAdminLogHandler(db *gorm.DB) *AdminLogHandler {
	return &AdminLogHandler{logRepo: repo.NewAdminLogRepo(db)}
}

func (h *AdminLogHandler) ListLogs(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionAdminLogsList) {
		response.Error(c, 403, "forbidden")
		return
	}
	logs, total, err := h.logRepo.ListWithTotal(20, 0)
	if err != nil {
		response.Error(c, 500, "list logs failed")
		return
	}
	response.List(c, logs, total)
}
