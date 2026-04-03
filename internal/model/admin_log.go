package model

import (
	"time"

	"gorm.io/datatypes"
)

// AdminLog records one administrative operation for auditing.
type AdminLog struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	AdminID    uint           `gorm:"index;not null" json:"admin_id"`
	Action     string         `gorm:"type:varchar(50);index;not null" json:"action"`
	TargetType string         `gorm:"type:varchar(30);index;not null" json:"target_type"`
	TargetID   uint           `gorm:"index;not null" json:"target_id"`
	Detail     datatypes.JSON `gorm:"type:jsonb" json:"detail"`
	IPAddress  string         `gorm:"type:varchar(50)" json:"ip_address"`
	CreatedAt  time.Time      `json:"created_at"`
}
