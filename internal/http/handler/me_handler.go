package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/repo"
	"manage/internal/service/authz"
)

type MeHandler struct {
	userRepo *repo.UserRepo
}

func NewMeHandler(db *gorm.DB) *MeHandler {
	return &MeHandler{userRepo: repo.NewUserRepo(db)}
}

func (h *MeHandler) GetMe(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionGetMe) {
		response.Error(c, 403, "forbidden")
		return
	}
	scope := authz.BuildScope(actor)
	users, err := h.userRepo.ListByScope(scope, 1, 0)
	if err != nil {
		response.Error(c, 500, "query me failed")
		return
	}
	if len(users) == 0 {
		response.Error(c, 404, "user not found")
		return
	}
	response.OK(c, users[0])
}
