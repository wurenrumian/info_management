package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"manage/internal/model"
)

// WechatClientInterface defines the contract for sending WeChat subscribe messages.
type WechatClientInterface interface {
	SendSubscribeMessage(openid, templateID, page string, data map[string]interface{}) error
}

// RepoInterface defines the contract for notification data access.
type RepoInterface interface {
	GetTemplateByCode(code string) (*model.NotificationTemplate, error)
	CreateTemplate(t *model.NotificationTemplate) error
	CreateLog(log *model.NotificationLog) error
	IsUserSubscribed(userID uint, templateCode string) (bool, error)
	ConsumeSubscription(userID uint, templateCode string) error
	CountUnreadByUser(userID uint) (int64, error)
	ListLogs(filter LogFilter) ([]model.NotificationLog, int64, error)
}

// UserRepo defines the contract for retrieving user OpenID.
type UserRepo interface {
	GetUserOpenID(userID uint) (string, error)
}

// Service orchestrates notification sending, template management, and log querying.
type Service struct {
	wechatClient WechatClientInterface
	repo         RepoInterface
	userRepo     UserRepo
}

// NewService creates a new Service with the given dependencies.
func NewService(wechatClient WechatClientInterface, repo RepoInterface, userRepo UserRepo) *Service {
	return &Service{
		wechatClient: wechatClient,
		repo:         repo,
		userRepo:     userRepo,
	}
}

// SendRequest holds the parameters for sending a single notification.
type SendRequest struct {
	UserID       uint
	TemplateCode string
	Page         string
	TemplateData map[string]interface{}
}

// SendResult holds the per-user result of a batch send operation.
type SendResult struct {
	UserID uint
	Err    error
}

// Send sends a single notification to the specified user.
func (s *Service) Send(ctx context.Context, req SendRequest) error {
	if !s.isEnabled() {
		return nil
	}

	tmpl, err := s.repo.GetTemplateByCode(req.TemplateCode)
	if err != nil {
		return fmt.Errorf("get template %s: %w", req.TemplateCode, err)
	}

	subscribed, err := s.repo.IsUserSubscribed(req.UserID, req.TemplateCode)
	if err != nil {
		s.recordLog(req, "failed", fmt.Sprintf("check subscription: %v", err))
		return fmt.Errorf("check user subscription: %w", err)
	}
	if !subscribed {
		s.recordLog(req, "failed", "user not subscribed")
		return fmt.Errorf("user %d is not subscribed to template %s", req.UserID, req.TemplateCode)
	}

	openID, err := s.userRepo.GetUserOpenID(req.UserID)
	if err != nil {
		s.recordLog(req, "failed", fmt.Sprintf("get openid: %v", err))
		return fmt.Errorf("get user openid: %w", err)
	}

	if openID == "" {
		s.recordLog(req, "failed", "user has no openid")
		return fmt.Errorf("user %d has no openid", req.UserID)
	}

	err = s.wechatClient.SendSubscribeMessage(openID, tmpl.WechatTemplateID, req.Page, req.TemplateData)

	if err != nil {
		s.recordLog(req, "failed", err.Error())
		return fmt.Errorf("send subscribe message: %w", err)
	}
	if err := s.repo.ConsumeSubscription(req.UserID, req.TemplateCode); err != nil {
		s.recordLog(req, "failed", fmt.Sprintf("consume subscription: %v", err))
		return fmt.Errorf("consume subscription: %w", err)
	}

	s.recordLog(req, "sent", "")
	return nil
}

// SendBatch sends notifications to multiple users using the same template.
func (s *Service) SendBatch(ctx context.Context, templateCode string, users []uint, dataFn func(userID uint) map[string]interface{}) []SendResult {
	results := make([]SendResult, 0, len(users))
	for _, uid := range users {
		req := SendRequest{
			UserID:       uid,
			TemplateCode: templateCode,
			Page:         "",
			TemplateData: dataFn(uid),
		}
		err := s.Send(ctx, req)
		results = append(results, SendResult{UserID: uid, Err: err})
	}
	return results
}

// GetTemplate retrieves a notification template by its code.
func (s *Service) GetTemplate(code string) (*model.NotificationTemplate, error) {
	return s.repo.GetTemplateByCode(code)
}

// CreateTemplate persists a new notification template.
func (s *Service) CreateTemplate(t *model.NotificationTemplate) error {
	return s.repo.CreateTemplate(t)
}

// GetLogs retrieves notification send logs matching the filter.
func (s *Service) GetLogs(filter LogFilter) ([]model.NotificationLog, int64, error) {
	return s.repo.ListLogs(filter)
}

// GetUnreadCount returns the unread notification count for a user.
func (s *Service) GetUnreadCount(userID uint) (int64, error) {
	return s.repo.CountUnreadByUser(userID)
}

func (s *Service) recordLog(req SendRequest, status string, errMsg string) {
	dataBytes, _ := json.Marshal(req.TemplateData)
	log := &model.NotificationLog{
		UserID:       req.UserID,
		TemplateCode: req.TemplateCode,
		TemplateData: string(dataBytes),
		Status:       status,
		ErrorMsg:     errMsg,
		CreatedAt:    time.Now(),
	}
	if status == "sent" {
		now := time.Now()
		log.SentAt = &now
	}
	s.repo.CreateLog(log)
}

func (s *Service) isEnabled() bool {
	v := os.Getenv("WECHAT_SUBSCRIBE_MSG_ENABLED")
	return v != "false"
}
