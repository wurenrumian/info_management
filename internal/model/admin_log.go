package model

import (
	"time"

	"gorm.io/datatypes"
)

type AdminLog struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	AdminID    uint           `gorm:"index;not null" json:"admin_id"`
	Action     string         `gorm:"size:50;index;not null" json:"action"`
	TargetType string         `gorm:"size:30;index;not null" json:"target_type"`
	TargetID   uint           `gorm:"index;not null" json:"target_id"`
	Detail     datatypes.JSON `gorm:"type:jsonb" json:"detail"`
	IPAddress  string         `gorm:"size:50" json:"ip_address"`
	CreatedAt  time.Time      `json:"created_at"`
}
