package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/service/authz"
	"manage/internal/service/profile"
)

type MeHandler struct {
	svc *profile.Service
}

func NewMeHandler(db *gorm.DB) *MeHandler {
	return &MeHandler{svc: profile.NewService(db)}
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

	user, err := h.svc.GetMe(actor)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "user not found")
			return
		}
		response.Error(c, 500, "query me failed")
		return
	}
	response.OK(c, user)
}

func (h *MeHandler) GetProfileHome(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionProfileHomeGet) {
		response.Error(c, 403, "forbidden")
		return
	}

	data, err := h.svc.GetHome(actor)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "user not found")
			return
		}
		response.Error(c, 500, "query profile home failed")
		return
	}

	response.OK(c, data)
}

func (h *MeHandler) PatchMe(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionMePatch) {
		response.Error(c, 403, "forbidden")
		return
	}

	var req struct {
		AvatarURL *string `json:"avatar_url"`
		Bio       *string `json:"bio"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}

	if err := h.svc.PatchMe(actor.UserID, req.AvatarURL, req.Bio); err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			response.Error(c, 404, "user not found")
		case err.Error() == "empty patch":
			response.Error(c, 400, "empty patch")
		default:
			response.Error(c, 500, "update me failed")
		}
		return
	}

	response.OK(c, gin.H{"ok": true})
}
