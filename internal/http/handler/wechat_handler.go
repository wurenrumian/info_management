package handler

import (
	"net/http"

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
		response.Error(c, 400, "微信授权码无效")
		return
	}

	user, err := h.userRepo.GetByOpenID(openID)
	if err != nil {
		response.Error(c, 404, "未绑定账号，请先绑定")
		return
	}

	token, err := jwtauth.GenerateToken(user.ID, user.Role, user.ClassID, user.Grade, h.jwtSecret)
	if err != nil {
		response.Error(c, 500, "生成 token 失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
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
		response.Error(c, 400, "微信授权码无效")
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
			response.Error(c, 401, "学号或密码不正确")
			return
		}
		if user.PasswordHash == nil {
			response.Error(c, 401, "学号或密码不正确")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
			response.Error(c, 401, "学号或密码不正确")
			return
		}
		userID = user.ID
	} else {
		response.Error(c, 401, "请登录后绑定或提供学号和密码")
		return
	}

	if existing != nil && existing.ID != userID {
		response.Error(c, 409, "该微信已绑定其他账号")
		return
	}

	if err := h.userRepo.UpdateByID(userID, map[string]any{"openid": openID}); err != nil {
		response.Error(c, 500, "绑定失败")
		return
	}

	if existing == nil && req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err == nil {
			hashStr := string(hash)
			_ = h.userRepo.UpdatePasswordHash(userID, hashStr)
		}
	}

	response.OK(c, gin.H{"ok": true, "message": "绑定成功"})
}
