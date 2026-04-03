package model

import (
	"time"

	"gorm.io/datatypes"
)

// User represents one user account in the system.
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	StudentID    string         `gorm:"type:varchar(20);uniqueIndex;not null" json:"student_id"`
	Name         string         `gorm:"type:varchar(50);not null" json:"name"`
	OpenID       *string        `gorm:"type:varchar(100)" json:"open_id"`
	PasswordHash *string        `gorm:"type:varchar(255)" json:"-"`
	Role         int            `gorm:"not null;index" json:"role"`
	ClassID      uint           `gorm:"index" json:"class_id"`
	Grade        string         `gorm:"type:varchar(10);index" json:"grade"`
	Major        string         `gorm:"type:varchar(100)" json:"major"`
	ExtraAttrs   datatypes.JSON `gorm:"type:jsonb" json:"extra_attrs"`
	ProfileAttrs datatypes.JSON `gorm:"type:jsonb" json:"profile_attrs"`
	Class        Class          `gorm:"foreignKey:ClassID" json:"class"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}
