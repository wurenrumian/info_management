package model

import "time"

// UserSubscribe tracks a user's subscription status for a WeChat subscribe message template.
type UserSubscribe struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"index;not null" json:"user_id"`
	TemplateCode     string    `gorm:"size:64;index;not null" json:"template_code"`
	WechatTemplateID string    `gorm:"size:100;not null" json:"wechat_template_id"`
	Status           string    `gorm:"size:20;not null;default:subscribed" json:"status"`
	SubscribedAt     time.Time `json:"subscribed_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TableName returns the database table name for UserSubscribe.
func (UserSubscribe) TableName() string {
	return "user_subscribes"
}
