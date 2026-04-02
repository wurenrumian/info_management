package model

import "time"

// NotificationLog records the result of a WeChat subscribe message send attempt.
type NotificationLog struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       uint       `gorm:"index;not null" json:"user_id"`
	TemplateCode string     `gorm:"size:64;index;not null" json:"template_code"`
	TemplateData string     `gorm:"type:jsonb" json:"template_data"`
	Status       string     `gorm:"size:16;not null;default:pending" json:"status"`
	ErrorMsg     string     `gorm:"size:500" json:"error_msg"`
	SentAt       *time.Time `json:"sent_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// TableName returns the database table name for NotificationLog.
func (NotificationLog) TableName() string {
	return "notification_logs"
}
