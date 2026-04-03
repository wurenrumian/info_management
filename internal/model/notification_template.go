package model

import "time"

// NotificationTemplate represents a WeChat subscribe message template configuration.
type NotificationTemplate struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Code             string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"code"`
	WechatTemplateID string    `gorm:"type:varchar(100);not null" json:"wechat_template_id"`
	Name             string    `gorm:"type:varchar(100);not null" json:"name"`
	Fields           string    `gorm:"type:jsonb" json:"fields"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TableName returns the database table name for NotificationTemplate.
func (NotificationTemplate) TableName() string {
	return "notification_templates"
}
