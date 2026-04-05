package handler

import (
	"encoding/json"
	"errors"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/config"
	"manage/internal/http/response"
	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/audit"
	"manage/internal/service/authz"
	ksvc "manage/internal/service/knowledge"
)

// AdminKnowledgeHandler handles knowledge management APIs.
type AdminKnowledgeHandler struct {
	db             *gorm.DB
	svc            *ksvc.Service
	auditLogger    *audit.Logger
	documentRepo   *repo.DocumentRepo
	attachmentRepo *repo.KnowledgeAttachmentRepo
	qaGenerator    ksvc.QAGenerator
	uploadDir      string
}

// NewAdminKnowledgeHandler creates an admin knowledge handler.
func NewAdminKnowledgeHandler(db *gorm.DB) *AdminKnowledgeHandler {
	uploadDir := config.PrimaryUploadDir()
	return &AdminKnowledgeHandler{
		db:             db,
		svc:            ksvc.NewService(db),
		auditLogger:    audit.NewLogger(repo.NewAdminLogRepo(db)),
		documentRepo:   repo.NewDocumentRepo(db),
		attachmentRepo: repo.NewKnowledgeAttachmentRepo(db),
		qaGenerator:    ksvc.NewQAGeneratorFromConfig(),
		uploadDir:      uploadDir,
	}
}

type createKnowledgeReq struct {
	Question    string              `json:"question"`
	Answer      string              `json:"answer"`
	Keywords    []string            `json:"keywords"`
	Attachments []map[string]string `json:"attachments"`
}

type patchKnowledgeReq struct {
	Question    *string              `json:"question"`
	Answer      *string              `json:"answer"`
	Keywords    *[]string            `json:"keywords"`
	Attachments *[]map[string]string `json:"attachments"`
}

type bindAttachmentsReq struct {
	FileIDs []uint `json:"file_ids"`
}

type qaGeneratePreviewReq struct {
	FileIDs      []uint            `json:"file_ids"`
	QACountRange ksvc.QACountRange `json:"qa_count_range"`
}

type batchCreateKnowledgeReq struct {
	Items []ksvc.QADraft `json:"items"`
}

type attachmentResp struct {
	FileID      uint   `json:"file_id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	FileSize    int64  `json:"file_size"`
}

// ListKnowledge lists knowledge items for admins.
func (h *AdminKnowledgeHandler) ListKnowledge(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionKnowledgeList) {
		response.Error(c, 403, "forbidden")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	items, total, err := h.svc.List(c.Query("query"), limit, offset)
	if err != nil {
		response.Error(c, 500, "list knowledge failed")
		return
	}
	if err := h.hydrateKnowledgeItemsAttachments(items); err != nil {
		response.Error(c, 500, "list knowledge failed")
		return
	}
	response.List(c, items, total)
}

// GetKnowledge gets one knowledge item by id.
func (h *AdminKnowledgeHandler) GetKnowledge(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionKnowledgeGet) {
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

// CreateKnowledge creates one knowledge item.
func (h *AdminKnowledgeHandler) CreateKnowledge(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionKnowledgeCreate) {
		response.Error(c, 403, "forbidden")
		return
	}

	var req createKnowledgeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}

	keywords, err := json.Marshal(req.Keywords)
	if err != nil {
		response.Error(c, 400, "invalid keywords")
		return
	}

	emptyAttachments, _ := json.Marshal([]attachmentResp{})
	item := model.KnowledgeItem{
		Question:    strings.TrimSpace(req.Question),
		Answer:      strings.TrimSpace(req.Answer),
		Keywords:    datatypes.JSON(keywords),
		Attachments: datatypes.JSON(emptyAttachments),
		CreatedBy:   actor.UserID,
		UpdatedBy:   actor.UserID,
	}

	if err := h.svc.Create(&item); err != nil {
		response.Error(c, 500, "create knowledge failed")
		return
	}

	h.auditLogger.Log(c, actor, "knowledge.create", "knowledge", item.ID)
	response.OK(c, item)
}

// PatchKnowledge patches a knowledge item.
func (h *AdminKnowledgeHandler) PatchKnowledge(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionKnowledgePatch) {
		response.Error(c, 403, "forbidden")
		return
	}

	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "invalid id")
		return
	}

	var req patchKnowledgeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}

	updates := map[string]any{}
	if req.Question != nil {
		updates["question"] = strings.TrimSpace(*req.Question)
	}
	if req.Answer != nil {
		updates["answer"] = strings.TrimSpace(*req.Answer)
	}
	if req.Keywords != nil {
		keywords, err := json.Marshal(*req.Keywords)
		if err != nil {
			response.Error(c, 400, "invalid keywords")
			return
		}
		updates["keywords"] = datatypes.JSON(keywords)
	}
	if len(updates) == 0 {
		response.Error(c, 400, "empty patch")
		return
	}
	updates["updated_by"] = actor.UserID

	if err := h.svc.Patch(uint(id64), updates); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "knowledge not found")
			return
		}
		response.Error(c, 500, "patch knowledge failed")
		return
	}

	h.auditLogger.Log(c, actor, "knowledge.patch", "knowledge", uint(id64))
	response.OK(c, gin.H{"updated": true})
}

// BindAttachments explicitly binds documents to one knowledge item.
func (h *AdminKnowledgeHandler) BindAttachments(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionKnowledgePatch) {
		response.Error(c, 403, "forbidden")
		return
	}

	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "invalid id")
		return
	}
	knowledgeID := uint(id64)

	item, err := h.svc.GetByID(knowledgeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "knowledge not found")
			return
		}
		response.Error(c, 500, "get knowledge failed")
		return
	}

	var req bindAttachmentsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}
	fileIDs := dedupeUint(req.FileIDs)
	if len(fileIDs) == 0 {
		response.Error(c, 400, "missing file_ids")
		return
	}

	docs, err := h.documentRepo.ListByIDs(fileIDs)
	if err != nil {
		response.Error(c, 500, "get file failed")
		return
	}
	if len(docs) != len(fileIDs) {
		response.Error(c, 400, "file not found")
		return
	}

	existingRows, err := h.attachmentRepo.ListByKnowledgeID(knowledgeID)
	if err != nil {
		response.Error(c, 500, "list attachment failed")
		return
	}
	existing := make(map[uint]struct{}, len(existingRows))
	for _, row := range existingRows {
		existing[row.FileID] = struct{}{}
	}

	toCreate := make([]model.KnowledgeAttachment, 0, len(fileIDs))
	addedCount := 0
	alreadyCount := 0
	newDocs := make([]model.Document, 0, len(fileIDs))
	docByID := make(map[uint]model.Document, len(docs))
	for _, d := range docs {
		docByID[d.ID] = d
	}
	for _, fileID := range fileIDs {
		if _, ok := existing[fileID]; ok {
			alreadyCount++
			continue
		}
		addedCount++
		toCreate = append(toCreate, model.KnowledgeAttachment{KnowledgeID: knowledgeID, FileID: fileID, CreatedBy: actor.UserID})
		newDocs = append(newDocs, docByID[fileID])
	}
	if err := h.attachmentRepo.CreateBatch(toCreate); err != nil {
		response.Error(c, 500, "bind attachment failed")
		return
	}

	if addedCount > 0 {
		additional := extractTextFromDocuments(h.uploadDir, newDocs)
		merged := mergeContentText(item.ContentText, additional)
		if merged != item.ContentText {
			if err := h.svc.Patch(knowledgeID, map[string]any{"content_text": merged, "updated_by": actor.UserID}); err != nil {
				response.Error(c, 500, "patch knowledge failed")
				return
			}
		}
		h.auditLogger.Log(c, actor, "knowledge.attach", "knowledge", knowledgeID)
	}

	attachments, err := h.listAttachmentDetailsByKnowledgeID(knowledgeID)
	if err != nil {
		response.Error(c, 500, "list attachment failed")
		return
	}
	response.OK(c, gin.H{
		"added_count":   addedCount,
		"already_count": alreadyCount,
		"attachments":   attachments,
	})
}

// ListAttachments lists explicit attachment relations for one knowledge item.
func (h *AdminKnowledgeHandler) ListAttachments(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionKnowledgeGet) {
		response.Error(c, 403, "forbidden")
		return
	}

	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "invalid id")
		return
	}
	knowledgeID := uint(id64)

	if _, err := h.svc.GetByID(knowledgeID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "knowledge not found")
			return
		}
		response.Error(c, 500, "get knowledge failed")
		return
	}

	attachments, err := h.listAttachmentDetailsByKnowledgeID(knowledgeID)
	if err != nil {
		response.Error(c, 500, "list attachment failed")
		return
	}
	response.OK(c, attachments)
}

// DeleteAttachment detaches one document from one knowledge item.
func (h *AdminKnowledgeHandler) DeleteAttachment(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionKnowledgeDelete) {
		response.Error(c, 403, "forbidden")
		return
	}

	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "invalid id")
		return
	}
	fileID64, err := strconv.ParseUint(c.Param("file_id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "invalid file_id")
		return
	}
	knowledgeID := uint(id64)
	fileID := uint(fileID64)

	if _, err := h.svc.GetByID(knowledgeID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "knowledge not found")
			return
		}
		response.Error(c, 500, "get knowledge failed")
		return
	}

	if err := h.attachmentRepo.DeleteByKnowledgeAndFileID(knowledgeID, fileID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "attachment not found")
			return
		}
		response.Error(c, 500, "delete attachment failed")
		return
	}

	h.auditLogger.Log(c, actor, "knowledge.detach", "knowledge", knowledgeID)
	response.OK(c, gin.H{"deleted": true})
}

// DeleteKnowledge deletes one knowledge item by id.
func (h *AdminKnowledgeHandler) DeleteKnowledge(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionKnowledgeDelete) {
		response.Error(c, 403, "forbidden")
		return
	}

	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, 400, "invalid id")
		return
	}

	if err := h.svc.Delete(uint(id64)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "knowledge not found")
			return
		}
		response.Error(c, 500, "delete knowledge failed")
		return
	}

	h.auditLogger.Log(c, actor, "knowledge.delete", "knowledge", uint(id64))
	response.OK(c, gin.H{"deleted": true})
}

// GenerateQAPreview generates editable QA drafts from uploaded documents.
func (h *AdminKnowledgeHandler) GenerateQAPreview(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if actor.Role < model.RoleTeacher {
		response.Error(c, 403, "forbidden")
		return
	}

	var req qaGeneratePreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}
	fileIDs := dedupeUint(req.FileIDs)
	if len(fileIDs) == 0 {
		response.Error(c, 400, "missing file_ids")
		return
	}
	if err := ksvc.ValidateCountRange(req.QACountRange); err != nil {
		response.Error(c, 400, "invalid qa_count_range")
		return
	}
	log.Printf("[knowledge-ai] preview request user_id=%d role=%d files=%d range=%d-%d", actor.UserID, actor.Role, len(fileIDs), req.QACountRange.Min, req.QACountRange.Max)

	docs, err := h.documentRepo.ListByIDs(fileIDs)
	if err != nil {
		response.Error(c, 500, "generate preview failed")
		return
	}
	if len(docs) != len(fileIDs) {
		response.Error(c, 400, "file not found")
		return
	}

	docByID := make(map[uint]model.Document, len(docs))
	for _, d := range docs {
		docByID[d.ID] = d
	}
	inputs := make([]ksvc.QADocumentInput, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		doc, ok := docByID[fileID]
		if !ok {
			response.Error(c, 400, "file not found")
			return
		}
		inputs = append(inputs, ksvc.QADocumentInput{
			FileID:      doc.ID,
			Title:       doc.Title,
			FilePath:    doc.FilePath,
			URL:         "/uploads/" + doc.FilePath,
			ContentText: doc.ContentText,
		})
	}

	drafts, err := h.qaGenerator.Generate(c.Request.Context(), inputs, req.QACountRange)
	if err != nil {
		log.Printf("[knowledge-ai] preview failed user_id=%d err=%v", actor.UserID, err)
		response.Error(c, 500, "generate preview failed")
		return
	}
	log.Printf("[knowledge-ai] preview success user_id=%d items=%d", actor.UserID, len(drafts))

	h.auditLogger.Log(c, actor, "knowledge.ai_generate_preview", "knowledge", 0)
	response.List(c, drafts, int64(len(drafts)))
}

// BatchCreateKnowledge creates multiple knowledge items in one transaction.
func (h *AdminKnowledgeHandler) BatchCreateKnowledge(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if actor.Role < model.RoleTeacher {
		response.Error(c, 403, "forbidden")
		return
	}

	var req batchCreateKnowledgeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}
	if len(req.Items) == 0 {
		response.Error(c, 400, "empty items")
		return
	}

	allFileIDs := make([]uint, 0, len(req.Items))
	for _, item := range req.Items {
		allFileIDs = append(allFileIDs, item.AttachmentFileIDs...)
	}
	allFileIDs = dedupeUint(allFileIDs)
	docByID := map[uint]model.Document{}
	if len(allFileIDs) > 0 {
		docs, err := h.documentRepo.ListByIDs(allFileIDs)
		if err != nil {
			response.Error(c, 500, "batch create knowledge failed")
			return
		}
		if len(docs) != len(allFileIDs) {
			response.Error(c, 400, "file not found")
			return
		}
		for _, d := range docs {
			docByID[d.ID] = d
		}
	}

	createdItems := make([]model.KnowledgeItem, 0, len(req.Items))
	err := h.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Items {
			question := strings.TrimSpace(item.Question)
			answer := strings.TrimSpace(item.Answer)
			if question == "" || answer == "" {
				return ksvc.ErrInvalidDraftItem
			}

			keywordsIn := item.Keywords
			if keywordsIn == nil {
				keywordsIn = []string{}
			}
			keywords, err := json.Marshal(keywordsIn)
			if err != nil {
				return ksvc.ErrInvalidDraftItem
			}
			emptyAttachments, _ := json.Marshal([]attachmentResp{})

			record := model.KnowledgeItem{
				Question:    question,
				Answer:      answer,
				Keywords:    datatypes.JSON(keywords),
				Attachments: datatypes.JSON(emptyAttachments),
				CreatedBy:   actor.UserID,
				UpdatedBy:   actor.UserID,
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}

			attachmentIDs := dedupeUint(item.AttachmentFileIDs)
			if len(attachmentIDs) > 0 {
				rels := make([]model.KnowledgeAttachment, 0, len(attachmentIDs))
				attachmentDocs := make([]model.Document, 0, len(attachmentIDs))
				for _, fileID := range attachmentIDs {
					doc, ok := docByID[fileID]
					if !ok {
						return ksvc.ErrInvalidDraftItem
					}
					rels = append(rels, model.KnowledgeAttachment{
						KnowledgeID: record.ID,
						FileID:      fileID,
						CreatedBy:   actor.UserID,
					})
					attachmentDocs = append(attachmentDocs, doc)
				}
				if err := tx.Create(&rels).Error; err != nil {
					return err
				}

				additional := extractTextFromDocuments(h.uploadDir, attachmentDocs)
				merged := mergeContentText(record.ContentText, additional)
				if merged != record.ContentText {
					if err := tx.Model(&model.KnowledgeItem{}).Where("id = ?", record.ID).Updates(map[string]any{
						"content_text": merged,
						"updated_by":   actor.UserID,
					}).Error; err != nil {
						return err
					}
					record.ContentText = merged
				}

				attachments := make([]attachmentResp, 0, len(attachmentDocs))
				for _, doc := range attachmentDocs {
					attachments = append(attachments, toAttachmentResp(doc))
				}
				b, err := json.Marshal(attachments)
				if err != nil {
					return err
				}
				if err := tx.Model(&model.KnowledgeItem{}).Where("id = ?", record.ID).Update("attachments", datatypes.JSON(b)).Error; err != nil {
					return err
				}
				record.Attachments = datatypes.JSON(b)
			}

			createdItems = append(createdItems, record)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ksvc.ErrInvalidDraftItem) {
			response.Error(c, 400, "invalid item")
			return
		}
		response.Error(c, 500, "batch create knowledge failed")
		return
	}

	h.auditLogger.Log(c, actor, "knowledge.batch_create", "knowledge", 0)
	response.List(c, createdItems, int64(len(createdItems)))
}

func (h *AdminKnowledgeHandler) listAttachmentDetailsByKnowledgeID(knowledgeID uint) ([]attachmentResp, error) {
	rows, err := h.attachmentRepo.ListByKnowledgeID(knowledgeID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []attachmentResp{}, nil
	}
	fileIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		fileIDs = append(fileIDs, row.FileID)
	}
	docs, err := h.documentRepo.ListByIDs(fileIDs)
	if err != nil {
		return nil, err
	}
	docByID := make(map[uint]model.Document, len(docs))
	for _, d := range docs {
		docByID[d.ID] = d
	}
	out := make([]attachmentResp, 0, len(rows))
	for _, row := range rows {
		doc, ok := docByID[row.FileID]
		if !ok {
			continue
		}
		out = append(out, toAttachmentResp(doc))
	}
	return out, nil
}

func (h *AdminKnowledgeHandler) hydrateKnowledgeItemAttachments(item *model.KnowledgeItem) error {
	attachments, err := h.listAttachmentDetailsByKnowledgeID(item.ID)
	if err != nil {
		return err
	}
	b, err := json.Marshal(attachments)
	if err != nil {
		return err
	}
	item.Attachments = datatypes.JSON(b)
	return nil
}

func (h *AdminKnowledgeHandler) hydrateKnowledgeItemsAttachments(items []model.KnowledgeItem) error {
	if len(items) == 0 {
		return nil
	}
	for i := range items {
		if err := h.hydrateKnowledgeItemAttachments(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

func toAttachmentResp(d model.Document) attachmentResp {
	return attachmentResp{
		FileID:      d.ID,
		Title:       d.Title,
		URL:         "/uploads/" + d.FilePath,
		ContentType: d.ContentType,
		FileSize:    d.FileSize,
	}
}

func dedupeUint(ids []uint) []uint {
	seen := map[uint]struct{}{}
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func extractTextFromDocuments(uploadDir string, docs []model.Document) string {
	parts := make([]string, 0, len(docs))
	for _, doc := range docs {
		if text := ksvc.ExtractTextFromFile(filepath.Join(uploadDir, doc.FilePath)); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

func mergeContentText(oldText, additional string) string {
	oldText = strings.TrimSpace(oldText)
	additional = strings.TrimSpace(additional)
	if oldText == "" {
		return additional
	}
	if additional == "" {
		return oldText
	}
	if strings.Contains(oldText, additional) {
		return oldText
	}
	return oldText + " " + additional
}
