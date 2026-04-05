package handler

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/authz"
	ksvc "manage/internal/service/knowledge"
)

// KnowledgeHandler handles student-facing knowledge search.
type KnowledgeHandler struct {
	svc            *ksvc.Service
	documentRepo   *repo.DocumentRepo
	attachmentRepo *repo.KnowledgeAttachmentRepo
}

// NewKnowledgeHandler creates a knowledge search handler.
func NewKnowledgeHandler(db *gorm.DB) *KnowledgeHandler {
	return &KnowledgeHandler{
		svc:            ksvc.NewService(db),
		documentRepo:   repo.NewDocumentRepo(db),
		attachmentRepo: repo.NewKnowledgeAttachmentRepo(db),
	}
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
		response.Error(c, 500, "search knowledge failed")
		return
	}
	if err := h.hydrateKnowledgeItemsAttachments(items); err != nil {
		response.Error(c, 500, "search knowledge failed")
		return
	}
	response.List(c, items, total)
}

// GetByID returns one knowledge item by id for student-facing detail page.
func (h *KnowledgeHandler) GetByID(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionKnowledgeSearch) {
		response.Error(c, 403, "forbidden")
		return
	}

	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "invalid id")
		return
	}
	item, err := h.svc.GetByID(uint(id64))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "knowledge not found")
			return
		}
		response.Error(c, 500, "get knowledge failed")
		return
	}
	if err := h.hydrateKnowledgeItemAttachments(item); err != nil {
		response.Error(c, 500, "get knowledge failed")
		return
	}
	response.OK(c, item)
}

func (h *KnowledgeHandler) hydrateKnowledgeItemAttachments(item *model.KnowledgeItem) error {
	rows, err := h.attachmentRepo.ListByKnowledgeID(item.ID)
	if err != nil {
		return err
	}
	fileIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		fileIDs = append(fileIDs, row.FileID)
	}
	docs, err := h.documentRepo.ListByIDs(fileIDs)
	if err != nil {
		return err
	}
	docByID := make(map[uint]model.Document, len(docs))
	for _, d := range docs {
		docByID[d.ID] = d
	}
	attachments := make([]attachmentResp, 0, len(rows))
	for _, row := range rows {
		doc, ok := docByID[row.FileID]
		if !ok {
			continue
		}
		attachments = append(attachments, toAttachmentResp(doc))
	}
	b, err := json.Marshal(attachments)
	if err != nil {
		return err
	}
	item.Attachments = datatypes.JSON(b)
	return nil
}

func (h *KnowledgeHandler) hydrateKnowledgeItemsAttachments(items []model.KnowledgeItem) error {
	if len(items) == 0 {
		return nil
	}

	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	rows, err := h.attachmentRepo.ListByKnowledgeIDs(ids)
	if err != nil {
		return err
	}
	fileIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		fileIDs = append(fileIDs, row.FileID)
	}
	docs, err := h.documentRepo.ListByIDs(fileIDs)
	if err != nil {
		return err
	}
	docByID := make(map[uint]model.Document, len(docs))
	for _, d := range docs {
		docByID[d.ID] = d
	}
	byKnowledge := make(map[uint][]attachmentResp)
	for _, row := range rows {
		doc, ok := docByID[row.FileID]
		if !ok {
			continue
		}
		byKnowledge[row.KnowledgeID] = append(byKnowledge[row.KnowledgeID], toAttachmentResp(doc))
	}

	for i := range items {
		attachments := byKnowledge[items[i].ID]
		if attachments == nil {
			attachments = []attachmentResp{}
		}
		b, err := json.Marshal(attachments)
		if err != nil {
			return err
		}
		items[i].Attachments = datatypes.JSON(b)
	}
	return nil
}
