package handler

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/model"
	"manage/internal/repo"
	jwtauth "manage/internal/service/auth"
	"manage/internal/service/notification"
	"manage/internal/service/wechat"
)

type WechatHandler struct {
	wechatSvc *wechat.Service
	userRepo  *repo.UserRepo
	devLogin  *jwtauth.DevLoginService
	notifSvc  *notification.Service
	db        *gorm.DB
	jwtSecret string
}

func NewWechatHandler(db *gorm.DB, appID, appSecret, jwtSecret string, notifSvc *notification.Service) *WechatHandler {
	return &WechatHandler{
		wechatSvc: wechat.NewService(appID, appSecret),
		userRepo:  repo.NewUserRepo(db),
		devLogin:  jwtauth.NewDevLoginService(db, jwtSecret),
		notifSvc:  notifSvc,
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

	token, err := jwtauth.GenerateToken(user.ID, user.Role, user.ClassID, user.Grade, h.jwtSecret)
	if err != nil {
		response.Error(c, 500, "generate token failed")
		return
	}

	response.OK(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":         user.ID,
			"student_id": user.StudentID,
			"name":       user.Name,
			"role":       user.Role,
			"class_id":   user.ClassID,
			"grade":      user.Grade,
			"major":      user.Major,
		},
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
	if strings.TrimSpace(os.Getenv("APP_ENV")) != "dev" {
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
		"user": gin.H{
			"id":         user.ID,
			"student_id": user.StudentID,
			"name":       user.Name,
			"role":       user.Role,
			"class_id":   user.ClassID,
			"grade":      user.Grade,
			"major":      user.Major,
		},
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
	if studentID == "" || name == "" {
		response.Error(c, 400, "missing student_id or name")
		return
	}

	user, err := h.userRepo.GetByStudentID(studentID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 500, "public register failed")
			return
		}

		user = &model.User{
			StudentID: studentID,
			Name:      name,
			Role:      model.RoleStudent,
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

	token, err := jwtauth.GenerateToken(user.ID, user.Role, user.ClassID, user.Grade, h.jwtSecret)
	if err != nil {
		response.Error(c, 500, "generate token failed")
		return
	}

	response.OK(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":         user.ID,
			"student_id": user.StudentID,
			"name":       user.Name,
			"role":       user.Role,
			"class_id":   user.ClassID,
			"grade":      user.Grade,
			"major":      user.Major,
		},
	})
}

// DevLoginAndSendSubscribeCheck handles a dev-only probe flow:
// register/login -> upsert subscribe state -> send one subscribe message.
func (h *WechatHandler) DevLoginAndSendSubscribeCheck(c *gin.Context) {
	if strings.TrimSpace(os.Getenv("APP_ENV")) != "dev" {
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
			Fields:           `{"thing1":"验证消息","time2":"时间"}`,
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
		sub.SubscribedAt = now
		if err := h.db.Create(&sub).Error; err != nil {
			response.Error(c, 500, "failed to record subscription")
			return
		}
	case err != nil:
		response.Error(c, 500, "failed to query subscription")
		return
	default:
		if err := h.db.Model(&model.UserSubscribe{}).Where("id = ?", existing.ID).Updates(sub).Error; err != nil {
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
		templateData := req.TemplateData
		if templateData == nil {
			templateData = map[string]interface{}{
				"thing1": map[string]string{"value": "Dev登录订阅验证"},
				"time2":  map[string]string{"value": now.Format("2006-01-02 15:04:05")},
			}
		}
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

	response.OK(c, gin.H{
		"token":               token,
		"user":                user,
		"template_code":       templateCode,
		"subscription_status": subStatus,
		"send_ok":             sendOK,
		"send_error":          sendErr,
	})
}

func (h *WechatHandler) writeDevLoginError(c *gin.Context, err error, fallback string) {
	switch err.Error() {
	case "missing student_id":
		response.Error(c, 400, "missing student_id")
	case "invalid role":
		response.Error(c, 400, "invalid role")
	default:
		response.Error(c, 500, fallback)
	}
}
