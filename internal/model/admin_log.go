package model

import (
	"time"

	"gorm.io/datatypes"
)

type AdminLog struct {
	ID         uint           `gorm:"primaryKey"`
	AdminID    uint           `gorm:"index;not null"`
	Action     string         `gorm:"size:50;index;not null"`
	TargetType string         `gorm:"size:30;index;not null"`
	TargetID   uint           `gorm:"index;not null"`
	Detail     datatypes.JSON `gorm:"type:jsonb"`
	IPAddress  string         `gorm:"size:50"`
	CreatedAt  time.Time
}
