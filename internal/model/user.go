package model

import (
	"time"

	"gorm.io/datatypes"
)

type User struct {
	ID           uint           `gorm:"primaryKey"`
	StudentID    string         `gorm:"size:20;uniqueIndex;not null"`
	Name         string         `gorm:"size:50;not null"`
	OpenID       *string        `gorm:"size:100"`
	Role         int            `gorm:"not null;index"`
	ClassID      uint           `gorm:"index"`
	Grade        string         `gorm:"size:10;index"`
	Major        string         `gorm:"size:100"`
	ExtraAttrs   datatypes.JSON `gorm:"type:jsonb"`
	ProfileAttrs datatypes.JSON `gorm:"type:jsonb"`
	Class        Class          `gorm:"foreignKey:ClassID"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
