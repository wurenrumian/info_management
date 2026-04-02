package model

import "time"

// NotificationLog records the result of a WeChat subscribe message send attempt.
type NotificationLog struct {
	ID           uint   `gorm:"primaryKey"`
	UserID       uint   `gorm:"index;not null"`
	TemplateCode string `gorm:"size:64;index;not null"`
	TemplateData string `gorm:"type:jsonb"`
	Status       string `gorm:"size:16;not null;default:pending"`
	ErrorMsg     string `gorm:"size:500"`
	SentAt       *time.Time
	CreatedAt    time.Time
}

// TableName returns the database table name for NotificationLog.
func (NotificationLog) TableName() string {
	return "notification_logs"
}
