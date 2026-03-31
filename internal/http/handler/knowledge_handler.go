package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/service/authz"
	knowledgeSvc "manage/internal/service/knowledge"
)

// KnowledgeHandler handles student-facing knowledge search.
type KnowledgeHandler struct {
	svc *knowledgeSvc.Service
}

// NewKnowledgeHandler creates a knowledge search handler.
func NewKnowledgeHandler(db *gorm.DB) *KnowledgeHandler {
	return &KnowledgeHandler{svc: knowledgeSvc.NewService(db)}
}

// Search returns knowledge items by keyword.
func (h *KnowledgeHandler) Search(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionKnowledgeSearch) {
		response.Error(c, 403, "forbidden")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	items, total, err := h.svc.Search(c.Query("q"), limit, offset)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	c.JSON(200, gin.H{"data": items, "total": total})
}
