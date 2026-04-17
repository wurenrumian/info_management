package model

import (
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const DefaultUserGrade = "2024"

// User represents one user account in the system.
type User struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	StudentID      string         `gorm:"type:varchar(20);uniqueIndex;not null" json:"student_id"`
	Name           string         `gorm:"type:varchar(50);not null" json:"name"`
	OpenID         *string        `gorm:"type:varchar(100)" json:"open_id"`
	PasswordHash   *string        `gorm:"type:varchar(255)" json:"-"`
	Role           int            `gorm:"not null;index" json:"role"`
	ClassID        uint           `gorm:"index" json:"class_id"`
	Grade          string         `gorm:"type:varchar(10);not null;default:'2024';index" json:"grade"`
	Major          string         `gorm:"type:varchar(100)" json:"major"`
	College        string         `gorm:"type:varchar(100)" json:"college"`
	EnrollmentYear int            `gorm:"index" json:"enrollment_year"`
	ExtraAttrs     datatypes.JSON `gorm:"type:jsonb" json:"extra_attrs"`
	ProfileAttrs   datatypes.JSON `gorm:"type:jsonb" json:"profile_attrs"`
	Class          Class          `gorm:"foreignKey:ClassID" json:"class"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// BeforeCreate keeps grade required with a stable default for new users.
func (u *User) BeforeCreate(_ *gorm.DB) error {
	if strings.TrimSpace(u.Grade) == "" {
		u.Grade = DefaultUserGrade
	}
	return nil
}
