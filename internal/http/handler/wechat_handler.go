package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/config"
	"manage/internal/http/response"
	"manage/internal/model"
	"manage/internal/repo"
	jwtauth "manage/internal/service/auth"
	gradesvc "manage/internal/service/grade"
	"manage/internal/service/notification"
	"manage/internal/service/wechat"
)

const (
	defaultPublicClassID   = config.DefaultPublicClassID
	defaultPublicClassName = config.DefaultPublicClassName
)

type WechatHandler struct {
	wechatSvc *wechat.Service
	userRepo  *repo.UserRepo
	devLogin  *jwtauth.DevLoginService
	notifSvc  *notification.Service
	gradeSvc  *gradesvc.Service
	db        *gorm.DB
	jwtSecret string
}

func NewWechatHandler(db *gorm.DB, appID, appSecret, jwtSecret string, notifSvc *notification.Service) *WechatHandler {
	return &WechatHandler{
		wechatSvc: wechat.NewService(appID, appSecret),
		userRepo:  repo.NewUserRepo(db),
		devLogin:  jwtauth.NewDevLoginService(db, jwtSecret),
		notifSvc:  notifSvc,
		gradeSvc:  gradesvc.NewService(db),
		db:        db,
		jwtSecret: jwtSecret,
	}
}

type wechatLoginReq struct {
	Code string `json:"code"`
}

type wechatBindReq struct {
	Code      string `json:"code"`
	StudentID string `json:"student_id"`
	Password  string `json:"password"`
}

type devLoginReq struct {
	StudentID string `json:"student_id"`
	Role      *int   `json:"role"`
}

type devLoginSubscribeCheckReq struct {
	StudentID        string                 `json:"student_id"`
	Role             *int                   `json:"role"`
	TemplateCode     string                 `json:"template_code"`
	WechatTemplateID string                 `json:"wechat_template_id"`
	Status           string                 `json:"status"`
	OpenID           string                 `json:"open_id"`
	Page             string                 `json:"page"`
	TemplateData     map[string]interface{} `json:"template_data"`
}

type publicRegisterReq struct {
	StudentID string `json:"student_id"`
	Name      string `json:"name"`
	Password  string `json:"password"`
	Code      string `json:"code"`
}

type publicLoginReq struct {
	StudentID string `json:"student_id"`
	Password  string `json:"password"`
	Code      string `json:"code"`
}

func (h *WechatHandler) Login(c *gin.Context) {
	var req wechatLoginReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		response.Error(c, 400, "missing code")
		return
	}

	openID, err := h.wechatSvc.CodeToOpenID(req.Code)
	if err != nil {
		response.Error(c, 400, "invalid authorization code")
		return
	}

	user, err := h.userRepo.GetByOpenID(openID)
	if err != nil {
		response.Error(c, 404, "account not bound, please bind first")
		return
	}

	effectiveGrade, err := h.gradeSvc.ResolveEffectiveGrade(user)
	if err != nil {
		response.Error(c, 500, "resolve effective grade failed")
		return
	}
	token, err := jwtauth.GenerateToken(user.ID, user.Role, user.ClassID, effectiveGrade, h.jwtSecret)
	if err != nil {
		response.Error(c, 500, "generate token failed")
		return
	}
	user.Grade = effectiveGrade

	response.OK(c, gin.H{
		"token": token,
		"user":  buildUserResponse(user, effectiveGrade),
	})
}

func (h *WechatHandler) Bind(c *gin.Context) {
	var req wechatBindReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		response.Error(c, 400, "missing code")
		return
	}

	openID, err := h.wechatSvc.CodeToOpenID(req.Code)
	if err != nil {
		response.Error(c, 400, "invalid authorization code")
		return
	}

	existing, _ := h.userRepo.GetByOpenID(openID)

	var userID uint

	actor, ok := auth.GetActor(c)
	if ok {
		userID = actor.UserID
	} else if req.StudentID != "" && req.Password != "" {
		user, err := h.userRepo.GetByStudentID(req.StudentID)
		if err != nil {
			response.Error(c, 401, "incorrect student id or password")
			return
		}
		if user.PasswordHash == nil {
			response.Error(c, 401, "incorrect student id or password")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
			response.Error(c, 401, "incorrect student id or password")
			return
		}
		userID = user.ID
	} else {
		response.Error(c, 401, "please login first or provide student id and password")
		return
	}

	if existing != nil && existing.ID != userID {
		response.Error(c, 409, "this wechat account is already bound to another user")
		return
	}

	if err := h.userRepo.UpdateByID(userID, map[string]any{"open_id": openID}); err != nil {
		response.Error(c, 500, "bind failed")
		return
	}

	if existing == nil && req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err == nil {
			hashStr := string(hash)
			_ = h.userRepo.UpdatePasswordHash(userID, hashStr)
		}
	}

	response.OK(c, gin.H{"ok": true, "message": "bind success"})
}

func (h *WechatHandler) DevRegisterOrLogin(c *gin.Context) {
	if !config.IsDevEnv() {
		response.Error(c, 403, "dev register-or-login is disabled")
		return
	}

	var req devLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}

	token, user, err := h.devLogin.RegisterOrLogin(req.StudentID, req.Role)
	if err != nil {
		h.writeDevLoginError(c, err, "dev register-or-login failed")
		return
	}

	response.OK(c, gin.H{
		"token": token,
		"user":  buildUserResponse(user, user.Grade),
	})
}

// PublicRegister handles production/public registration using student_id + name,
// with optional WeChat code binding.
func (h *WechatHandler) PublicRegister(c *gin.Context) {
	var req publicRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}

	studentID := strings.TrimSpace(req.StudentID)
	name := strings.TrimSpace(req.Name)
	password := strings.TrimSpace(req.Password)
	if studentID == "" || name == "" {
		response.Error(c, 400, "missing student_id or name")
		return
	}
	if password == "" {
		password = name
	}

	user, err := h.userRepo.GetByStudentID(studentID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 500, "public register failed")
			return
		}
		if err := h.ensureDefaultPublicClass(); err != nil {
			response.Error(c, 500, "public register failed")
			return
		}

		hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			response.Error(c, 500, "public register failed")
			return
		}
		hashStr := string(hash)
		user = &model.User{
			StudentID:    studentID,
			Name:         name,
			Role:         model.RoleStudent,
			ClassID:      defaultPublicClassID,
			PasswordHash: &hashStr,
		}
		if err := h.userRepo.Create(user); err != nil {
			response.Error(c, 500, "public register failed")
			return
		}
	} else {
		if strings.TrimSpace(user.Name) != name {
			response.Error(c, 401, "student id and name do not match")
			return
		}
		if err := verifyPasswordAndMaybeBackfill(h.userRepo, user, password); err != nil {
			response.Error(c, 401, "incorrect student id or password")
			return
		}
	}

	if code := strings.TrimSpace(req.Code); code != "" {
		openID, err := h.wechatSvc.CodeToOpenID(code)
		if err != nil {
			response.Error(c, 400, "invalid authorization code")
			return
		}

		existing, _ := h.userRepo.GetByOpenID(openID)
		if existing != nil && existing.ID != user.ID {
			response.Error(c, 409, "this wechat account is already bound to another user")
			return
		}

		if err := h.userRepo.UpdateByID(user.ID, map[string]any{"open_id": openID}); err != nil {
			response.Error(c, 500, "bind failed")
			return
		}
		user.OpenID = &openID
	}

	effectiveGrade, err := h.gradeSvc.ResolveEffectiveGrade(user)
	if err != nil {
		response.Error(c, 500, "resolve effective grade failed")
		return
	}
	token, err := jwtauth.GenerateToken(user.ID, user.Role, user.ClassID, effectiveGrade, h.jwtSecret)
	if err != nil {
		response.Error(c, 500, "generate token failed")
		return
	}
	user.Grade = effectiveGrade

	response.OK(c, gin.H{
		"token": token,
		"user":  buildUserResponse(user, effectiveGrade),
	})
}

func (h *WechatHandler) PublicLogin(c *gin.Context) {
	var req publicLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}

	studentID := strings.TrimSpace(req.StudentID)
	password := strings.TrimSpace(req.Password)
	if studentID == "" || password == "" {
		response.Error(c, 400, "missing student_id or password")
		return
	}

	user, err := h.userRepo.GetByStudentID(studentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 401, "incorrect student id or password")
			return
		}
		response.Error(c, 500, "login failed")
		return
	}
	if err := verifyPasswordAndMaybeBackfill(h.userRepo, user, password); err != nil {
		response.Error(c, 401, "incorrect student id or password")
		return
	}

	if code := strings.TrimSpace(req.Code); code != "" {
		openID, err := h.wechatSvc.CodeToOpenID(code)
		if err != nil {
			response.Error(c, 400, "invalid authorization code")
			return
		}
		existing, _ := h.userRepo.GetByOpenID(openID)
		if existing != nil && existing.ID != user.ID {
			response.Error(c, 409, "this wechat account is already bound to another user")
			return
		}
		if err := h.userRepo.UpdateByID(user.ID, map[string]any{"open_id": openID}); err != nil {
			response.Error(c, 500, "bind failed")
			return
		}
		user.OpenID = &openID
	}

	effectiveGrade, err := h.gradeSvc.ResolveEffectiveGrade(user)
	if err != nil {
		response.Error(c, 500, "resolve effective grade failed")
		return
	}
	token, err := jwtauth.GenerateToken(user.ID, user.Role, user.ClassID, effectiveGrade, h.jwtSecret)
	if err != nil {
		response.Error(c, 500, "generate token failed")
		return
	}
	user.Grade = effectiveGrade

	response.OK(c, gin.H{
		"token": token,
		"user":  buildUserResponse(user, effectiveGrade),
	})
}

// DevLoginAndSendSubscribeCheck handles a dev-only probe flow:
// register/login -> upsert subscribe state -> send one subscribe message.
func (h *WechatHandler) DevLoginAndSendSubscribeCheck(c *gin.Context) {
	if !config.IsDevEnv() {
		response.Error(c, 403, "dev login-and-send-subscribe-check is disabled")
		return
	}
	if h.notifSvc == nil {
		response.Error(c, 500, "notification service unavailable")
		return
	}

	var req devLoginSubscribeCheckReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "invalid body")
		return
	}

	status := req.Status
	if status == "" {
		status = "accept"
	}
	if status != "accept" && status != "reject" {
		response.Error(c, 400, "status must be accept or reject")
		return
	}

	templateCode := strings.TrimSpace(req.TemplateCode)
	if templateCode == "" {
		templateCode = "dev_login_check"
	}

	wechatTemplateID := strings.TrimSpace(req.WechatTemplateID)
	if wechatTemplateID == "" {
		wechatTemplateID = "tmpl_dev_login_check"
	}

	token, user, err := h.devLogin.RegisterOrLogin(req.StudentID, req.Role)
	if err != nil {
		h.writeDevLoginError(c, err, "dev login-and-send-subscribe-check failed")
		return
	}

	openID := strings.TrimSpace(req.OpenID)
	if openID == "" && user.OpenID == nil {
		openID = "dev-openid-" + strings.ToLower(user.StudentID)
	}
	if openID != "" {
		if err := h.userRepo.UpdateByID(user.ID, map[string]any{"open_id": openID}); err != nil {
			response.Error(c, 500, "update openid failed")
			return
		}
		user.OpenID = &openID
	}

	if _, err := h.notifSvc.GetTemplate(templateCode); err != nil {
		if createErr := h.notifSvc.CreateTemplate(&model.NotificationTemplate{
			Code:             templateCode,
			WechatTemplateID: wechatTemplateID,
			Name:             "Dev Login Check",
		}); createErr != nil {
			response.Error(c, 500, "create dev notification template failed")
			return
		}
	}

	subStatus := "subscribed"
	if status == "reject" {
		subStatus = "unsubscribed"
	}

	now := time.Now()
	var existing model.UserSubscribe
	err = h.db.Where("user_id = ? AND template_code = ?", user.ID, templateCode).First(&existing).Error
	sub := model.UserSubscribe{
		UserID:           user.ID,
		TemplateCode:     templateCode,
		WechatTemplateID: wechatTemplateID,
		Status:           subStatus,
		UpdatedAt:        now,
	}

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		if subStatus == "subscribed" {
			sub.GrantedCount = 1
		}
		sub.SubscribedAt = now
		if err := h.db.Create(&sub).Error; err != nil {
			response.Error(c, 500, "failed to record subscription")
			return
		}
	case err != nil:
		response.Error(c, 500, "failed to query subscription")
		return
	default:
		updates := map[string]any{
			"wechat_template_id": wechatTemplateID,
			"status":             subStatus,
			"updated_at":         now,
		}
		if err := h.db.Model(&model.UserSubscribe{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
			response.Error(c, 500, "failed to update subscription")
			return
		}
	}

	sendOK := false
	sendErr := ""
	if subStatus == "subscribed" {
		page := strings.TrimSpace(req.Page)
		if page == "" {
			page = "/pages/index/index"
		}
		templateData := buildDevSubscribeTemplateData(templateCode, req.TemplateData, user, c.ClientIP(), now)
		if err := h.notifSvc.Send(context.Background(), notification.SendRequest{
			UserID:       user.ID,
			TemplateCode: templateCode,
			Page:         page,
			TemplateData: templateData,
		}); err != nil {
			sendErr = err.Error()
		} else {
			sendOK = true
		}
	} else {
		sendErr = "subscription status is reject"
	}

	currentSub, err := h.subscribeHandlerState(user.ID, templateCode)
	if err != nil {
		response.Error(c, 500, "failed to query subscription")
		return
	}

	response.OK(c, gin.H{
		"token":               token,
		"user":                user,
		"template_code":       templateCode,
		"subscription_status": subStatus,
		"granted_count":       currentSub.GrantedCount,
		"consumed_count":      currentSub.ConsumedCount,
		"remaining_count":     remainingSubscribeCount(currentSub),
		"send_ok":             sendOK,
		"send_error":          sendErr,
	})
}

func (h *WechatHandler) subscribeHandlerState(userID uint, templateCode string) (*model.UserSubscribe, error) {
	var sub model.UserSubscribe
	if err := h.db.Where("user_id = ? AND template_code = ?", userID, templateCode).First(&sub).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func (h *WechatHandler) writeDevLoginError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, jwtauth.ErrMissingStudentID):
		response.Error(c, 400, "missing student_id")
	case errors.Is(err, jwtauth.ErrInvalidRole):
		response.Error(c, 400, "invalid role")
	default:
		response.Error(c, 500, fallback)
	}
}

func buildUserResponse(user *model.User, effectiveGrade string) gin.H {
	return gin.H{
		"id":         user.ID,
		"student_id": user.StudentID,
		"name":       user.Name,
		"role":       user.Role,
		"class_id":   user.ClassID,
		"grade":      effectiveGrade,
		"major":      user.Major,
	}
}

func buildDevSubscribeTemplateData(templateCode string, input map[string]interface{}, user *model.User, clientIP string, now time.Time) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range input {
		out[k] = v
	}

	setDefaultTemplateValue(out, "thing1", "Dev登录订阅验证")
	setDefaultTemplateValue(out, "time2", now.Format("2006-01-02 15:04:05"))

	if templateCode == "loging_notification" {
		ip := strings.TrimSpace(clientIP)
		if ip == "" {
			ip = "127.0.0.1"
		}
		setDefaultTemplateValue(out, "character_string3", ip)

		userLabel := strings.TrimSpace(user.Name)
		if userLabel == "" {
			userLabel = strings.TrimSpace(user.StudentID)
		} else if strings.TrimSpace(user.StudentID) != "" {
			userLabel = userLabel + " (" + strings.TrimSpace(user.StudentID) + ")"
		}
		setDefaultTemplateValue(out, "thing4", userLabel)
	}

	return out
}

func setDefaultTemplateValue(data map[string]interface{}, key, fallback string) {
	if field, ok := data[key].(map[string]interface{}); ok {
		if value, ok := field["value"].(string); ok && strings.TrimSpace(value) != "" {
			return
		}
		field["value"] = fallback
		data[key] = field
		return
	}
	if field, ok := data[key].(map[string]string); ok {
		if value, ok := field["value"]; ok && strings.TrimSpace(value) != "" {
			return
		}
		field["value"] = fallback
		data[key] = field
		return
	}
	data[key] = map[string]interface{}{"value": fallback}
}

func (h *WechatHandler) ensureDefaultPublicClass() error {
	return repo.EnsureClass(h.db, defaultPublicClassID, defaultPublicClassName, "", "")
}
