package model

import (
	"time"

	"gorm.io/datatypes"
)

type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	StudentID    string         `gorm:"size:20;uniqueIndex;not null" json:"student_id"`
	Name         string         `gorm:"size:50;not null" json:"name"`
	OpenID       *string        `gorm:"size:100" json:"open_id"`
	PasswordHash *string        `gorm:"size:255" json:"-"`
	Role         int            `gorm:"not null;index" json:"role"`
	ClassID      uint           `gorm:"index" json:"class_id"`
	Grade        string         `gorm:"size:10;index" json:"grade"`
	Major        string         `gorm:"size:100" json:"major"`
	ExtraAttrs   datatypes.JSON `gorm:"type:jsonb" json:"extra_attrs"`
	ProfileAttrs datatypes.JSON `gorm:"type:jsonb" json:"profile_attrs"`
	Class        Class          `gorm:"foreignKey:ClassID" json:"class"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}
