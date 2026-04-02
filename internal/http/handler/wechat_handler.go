package handler

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/repo"
	jwtauth "manage/internal/service/auth"
	"manage/internal/service/wechat"
)

type WechatHandler struct {
	wechatSvc *wechat.Service
	userRepo  *repo.UserRepo
	jwtSecret string
}

func NewWechatHandler(db *gorm.DB, appID, appSecret, jwtSecret string) *WechatHandler {
	return &WechatHandler{
		wechatSvc: wechat.NewService(appID, appSecret),
		userRepo:  repo.NewUserRepo(db),
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

	if err := h.userRepo.UpdateByID(userID, map[string]any{"openid": openID}); err != nil {
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
