package notification

import (
	"manage/internal/model"

	"gorm.io/gorm"
)

// Repo provides database operations for notification templates and logs.
type Repo struct {
	db *gorm.DB
}

// NewRepo creates a new Repo instance with the given database connection.
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// GetTemplateByCode retrieves a notification template by its unique code.
func (r *Repo) GetTemplateByCode(code string) (*model.NotificationTemplate, error) {
	var t model.NotificationTemplate
	if err := r.db.Where("code = ?", code).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// CreateTemplate persists a new notification template to the database.
func (r *Repo) CreateTemplate(t *model.NotificationTemplate) error {
	return r.db.Create(t).Error
}

// CreateLog records a notification send attempt result.
func (r *Repo) CreateLog(log *model.NotificationLog) error {
	return r.db.Create(log).Error
}

// IsUserSubscribed returns true when the user has an active subscription for the template code.
func (r *Repo) IsUserSubscribed(userID uint, templateCode string) (bool, error) {
	var total int64
	err := r.db.Model(&model.UserSubscribe{}).
		Where("user_id = ? AND template_code = ? AND status = ?", userID, templateCode, "subscribed").
		Count(&total).Error
	if err != nil {
		return false, err
	}
	return total > 0, nil
}

// CountUnreadByUser returns the number of unread notifications for a user.
func (r *Repo) CountUnreadByUser(userID uint) (int64, error) {
	var total int64
	err := r.db.Model(&model.NotificationLog{}).
		Where("user_id = ? AND status = ?", userID, "pending").
		Count(&total).Error
	return total, err
}

// LogFilter defines query parameters for listing notification logs.
type LogFilter struct {
	UserID       *uint
	TemplateCode *string
	Status       *string
	Offset       int
	Limit        int
}

// ListLogs retrieves notification logs matching the filter with pagination.
func (r *Repo) ListLogs(filter LogFilter) ([]model.NotificationLog, int64, error) {
	q := r.db.Model(&model.NotificationLog{})
	if filter.UserID != nil {
		q = q.Where("user_id = ?", *filter.UserID)
	}
	if filter.TemplateCode != nil {
		q = q.Where("template_code = ?", *filter.TemplateCode)
	}
	if filter.Status != nil {
		q = q.Where("status = ?", *filter.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	var logs []model.NotificationLog
	if err := q.Order("created_at desc").Offset(filter.Offset).Limit(filter.Limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
