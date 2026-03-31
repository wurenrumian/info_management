package handler

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/authz"
	knowledgeSvc "manage/internal/service/knowledge"
)

// AdminKnowledgeHandler handles knowledge management APIs.
type AdminKnowledgeHandler struct {
	svc       *knowledgeSvc.Service
	logRepo   *repo.AdminLogRepo
	uploadDir string
}

// NewAdminKnowledgeHandler creates an admin knowledge handler.
func NewAdminKnowledgeHandler(db *gorm.DB) *AdminKnowledgeHandler {
	uploadDir := os.Getenv("KNOWLEDGE_UPLOAD_DIR")
	if strings.TrimSpace(uploadDir) == "" {
		uploadDir = "./data/uploads/knowledge"
	}
	return &AdminKnowledgeHandler{
		svc:       knowledgeSvc.NewService(db),
		logRepo:   repo.NewAdminLogRepo(db),
		uploadDir: uploadDir,
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
	items, err := h.svc.List(c.Query("query"), limit, offset)
	if err != nil {
		response.Error(c, 500, "list knowledge failed")
		return
	}
	response.OK(c, items)
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
	attachments, err := json.Marshal(req.Attachments)
	if err != nil {
		response.Error(c, 400, "invalid attachments")
		return
	}

	item := model.KnowledgeItem{
		Question:    strings.TrimSpace(req.Question),
		Answer:      strings.TrimSpace(req.Answer),
		Keywords:    datatypes.JSON(keywords),
		Attachments: datatypes.JSON(attachments),
		CreatedBy:   actor.UserID,
		UpdatedBy:   actor.UserID,
	}

	if err := h.svc.Create(&item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	_ = h.logRepo.Create(model.AdminLog{AdminID: actor.UserID, Action: "knowledge.create", TargetType: "knowledge", TargetID: item.ID, IPAddress: c.ClientIP()})
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
	if req.Attachments != nil {
		attachments, err := json.Marshal(*req.Attachments)
		if err != nil {
			response.Error(c, 400, "invalid attachments")
			return
		}
		updates["attachments"] = datatypes.JSON(attachments)
	}
	if len(updates) == 0 {
		response.Error(c, 400, "empty patch")
		return
	}
	updates["updated_by"] = actor.UserID

	if err := h.svc.Patch(uint(id64), updates); err != nil {
		response.Error(c, 500, "patch knowledge failed")
		return
	}

	_ = h.logRepo.Create(model.AdminLog{AdminID: actor.UserID, Action: "knowledge.patch", TargetType: "knowledge", TargetID: uint(id64), IPAddress: c.ClientIP()})
	response.OK(c, gin.H{"updated": true})
}

// ImportKnowledge imports one knowledge item with uploaded attachment files.
func (h *AdminKnowledgeHandler) ImportKnowledge(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.Error(c, 401, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionKnowledgeCreate) {
		response.Error(c, 403, "forbidden")
		return
	}

	question := strings.TrimSpace(c.PostForm("question"))
	answer := strings.TrimSpace(c.PostForm("answer"))
	if question == "" || answer == "" {
		response.Error(c, 400, "missing fields")
		return
	}

	attachments, contentText, err := h.saveUploadedFiles(c)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	keywords := splitKeywords(c.PostForm("keywords"))
	keywordsJSON, err := json.Marshal(keywords)
	if err != nil {
		response.Error(c, 400, "invalid keywords")
		return
	}
	attachmentsJSON, err := json.Marshal(attachments)
	if err != nil {
		response.Error(c, 400, "invalid attachments")
		return
	}

	item := model.KnowledgeItem{
		Question:    question,
		Answer:      answer,
		ContentText: contentText,
		Keywords:    datatypes.JSON(keywordsJSON),
		Attachments: datatypes.JSON(attachmentsJSON),
		CreatedBy:   actor.UserID,
		UpdatedBy:   actor.UserID,
	}
	if err := h.svc.Create(&item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	_ = h.logRepo.Create(model.AdminLog{
		AdminID:    actor.UserID,
		Action:     "knowledge.import",
		TargetType: "knowledge",
		TargetID:   item.ID,
		IPAddress:  c.ClientIP(),
	})
	response.OK(c, item)
}

func splitKeywords(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		s := strings.TrimSpace(part)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (h *AdminKnowledgeHandler) saveUploadedFiles(c *gin.Context) ([]map[string]string, string, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, "", fmt.Errorf("invalid multipart form")
	}
	files := form.File["files"]
	if len(files) == 0 {
		return nil, "", fmt.Errorf("missing files")
	}

	if err := os.MkdirAll(h.uploadDir, 0o755); err != nil {
		return nil, "", fmt.Errorf("prepare upload dir failed")
	}

	attachments := make([]map[string]string, 0, len(files))
	textParts := make([]string, 0, len(files))
	for _, file := range files {
		if !allowedAttachment(file) {
			return nil, "", fmt.Errorf("unsupported file type")
		}
		savedName := uniqueFileName(file.Filename)
		savedPath := filepath.Join(h.uploadDir, savedName)
		if err := c.SaveUploadedFile(file, savedPath); err != nil {
			return nil, "", fmt.Errorf("save file failed")
		}
		attachments = append(attachments, map[string]string{
			"title": file.Filename,
			"url":   "/uploads/knowledge/" + savedName,
		})
		if text := knowledgeSvc.ExtractTextFromFile(savedPath); text != "" {
			textParts = append(textParts, text)
		}
	}
	return attachments, strings.Join(textParts, " "), nil
}

func allowedAttachment(file *multipart.FileHeader) bool {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx":
		return true
	default:
		return false
	}
}

func uniqueFileName(origin string) string {
	base := filepath.Base(origin)
	base = strings.ReplaceAll(base, " ", "_")
	return fmt.Sprintf("%d_%s", time.Now().UnixNano(), base)
}
