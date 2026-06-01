package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/http/response"
	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/authz"
	"manage/internal/service/profile"
)

type MeHandler struct {
	svc      *profile.Service
	userRepo *repo.UserRepo
}

type updateProfileReq struct {
	Nickname       *string `json:"nickname"`
	Major          *string `json:"major"`
	College        *string `json:"college"`
	EnrollmentYear *int    `json:"enrollment_year"`
	AvatarURL      *string `json:"avatar_url"`
	Bio            *string `json:"bio"`
}

type changePasswordReq struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func NewMeHandler(db *gorm.DB) *MeHandler {
	return &MeHandler{svc: profile.NewService(db), userRepo: repo.NewUserRepo(db)}
}

func (h *MeHandler) GetMe(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.ErrorWithCode(c, 401, 40100, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionGetMe) {
		response.ErrorWithCode(c, 403, 40300, "forbidden")
		return
	}

	user, err := h.svc.GetMe(actor)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "user not found")
			return
		}
		response.ErrorWithCode(c, 500, 50000, "query me failed")
		return
	}
	response.OK(c, user)
}

func (h *MeHandler) GetProfileHome(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.ErrorWithCode(c, 401, 40100, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionProfileHomeGet) {
		response.ErrorWithCode(c, 403, 40300, "forbidden")
		return
	}

	data, err := h.svc.GetHome(actor)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "user not found")
			return
		}
		response.ErrorWithCode(c, 500, 50000, "query profile home failed")
		return
	}

	response.OK(c, data)
}

func (h *MeHandler) PatchPassword(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.ErrorWithCode(c, 401, 40100, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionMePatch) {
		response.ErrorWithCode(c, 403, 40300, "forbidden")
		return
	}

	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithCode(c, 400, 40001, "invalid body")
		return
	}
	currentPassword := strings.TrimSpace(req.CurrentPassword)
	newPassword := strings.TrimSpace(req.NewPassword)
	if currentPassword == "" || newPassword == "" {
		response.ErrorWithCode(c, 400, 40001, "missing current_password or new_password")
		return
	}

	user, err := h.userRepo.GetByIDInScope(authz.Scope{SelfUserID: actor.UserID}, actor.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "user not found")
			return
		}
		response.ErrorWithCode(c, 500, 50000, "query me failed")
		return
	}

	if err := verifyPasswordAndMaybeBackfill(h.userRepo, user, currentPassword); err != nil {
		response.ErrorWithCode(c, 401, 40101, "incorrect current password")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		response.ErrorWithCode(c, 500, 50000, "update password failed")
		return
	}
	if err := h.userRepo.UpdatePasswordHash(user.ID, string(hash)); err != nil {
		response.ErrorWithCode(c, 500, 50000, "update password failed")
		return
	}

	response.OK(c, gin.H{"ok": true})
}

func (h *MeHandler) PatchMe(c *gin.Context) {
	actor, ok := auth.GetActor(c)
	if !ok {
		response.ErrorWithCode(c, 401, 40100, "unauthorized")
		return
	}
	if !authz.Authorize(actor.Role, authz.ActionMePatch) {
		response.ErrorWithCode(c, 403, 40300, "forbidden")
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.ErrorWithCode(c, 400, 40001, "invalid body")
		return
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		response.ErrorWithCode(c, 400, 40001, "invalid body")
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		response.ErrorWithCode(c, 400, 40001, "invalid body")
		return
	}

	for _, k := range []string{"id", "student_id", "real_name", "role"} {
		if _, ok := raw[k]; ok {
			response.ErrorWithCode(c, 400, 40002, k+" is read-only")
			return
		}
	}

	var req updateProfileReq
	if err := json.Unmarshal(body, &req); err != nil {
		response.ErrorWithCode(c, 400, 40001, "invalid body")
		return
	}

	if req.Nickname != nil {
		trimmed := strings.TrimSpace(*req.Nickname)
		l := utf8Len(trimmed)
		if l < 1 || l > 20 {
			response.ErrorWithCode(c, 400, 40001, "nickname length must be between 1 and 20")
			return
		}
		req.Nickname = &trimmed
	}
	if req.Major != nil {
		trimmed := strings.TrimSpace(*req.Major)
		l := utf8Len(trimmed)
		if l < 1 || l > 50 {
			response.ErrorWithCode(c, 400, 40001, "major length must be between 1 and 50")
			return
		}
		req.Major = &trimmed
	}
	if req.College != nil {
		trimmed := strings.TrimSpace(*req.College)
		l := utf8Len(trimmed)
		if l < 1 || l > 50 {
			response.ErrorWithCode(c, 400, 40001, "college length must be between 1 and 50")
			return
		}
		req.College = &trimmed
	}
	if req.Bio != nil {
		trimmed := strings.TrimSpace(*req.Bio)
		l := utf8Len(trimmed)
		if l > 100 {
			response.ErrorWithCode(c, 400, 40001, "bio length must be between 0 and 100")
			return
		}
		req.Bio = &trimmed
	}
	if req.AvatarURL != nil {
		trimmed := strings.TrimSpace(*req.AvatarURL)
		if !isValidAvatarURL(trimmed) {
			response.ErrorWithCode(c, 400, 40001, "avatar_url must be a valid http/https URL")
			return
		}
		req.AvatarURL = &trimmed
	}
	if req.EnrollmentYear != nil {
		maxYear := time.Now().Year() + 1
		if *req.EnrollmentYear < 2000 || *req.EnrollmentYear > maxYear {
			response.ErrorWithCode(c, 400, 40003, "enrollment_year out of range")
			return
		}
	}

	out, err := h.svc.PatchMe(actor.UserID, profile.PatchMeInput{
		Nickname:       req.Nickname,
		Major:          req.Major,
		College:        req.College,
		EnrollmentYear: req.EnrollmentYear,
		AvatarURL:      req.AvatarURL,
		Bio:            req.Bio,
	})
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			response.Error(c, 404, "user not found")
		case errors.Is(err, profile.ErrEmptyPatch):
			response.ErrorWithCode(c, 400, 40001, "empty patch")
		default:
			response.ErrorWithCode(c, 500, 50000, "update me failed")
		}
		return
	}

	response.OK(c, out)
}

func verifyPasswordAndMaybeBackfill(userRepo *repo.UserRepo, user *model.User, plain string) error {
	if user.PasswordHash == nil || strings.TrimSpace(*user.PasswordHash) == "" {
		if plain != strings.TrimSpace(user.Name) {
			return errors.New("incorrect password")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		hashStr := string(hash)
		if err := userRepo.UpdatePasswordHash(user.ID, hashStr); err != nil {
			return err
		}
		user.PasswordHash = &hashStr
		return nil
	}
	return bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(plain))
}

func utf8Len(s string) int {
	return len([]rune(s))
}

func isValidAvatarURL(raw string) bool {
	if strings.HasPrefix(raw, "/uploads/") {
		return true
	}
	parsed, parseErr := url.Parse(raw)
	return parseErr == nil &&
		parsed != nil &&
		(parsed.Scheme == "http" || parsed.Scheme == "https") &&
		parsed.Host != ""
}
