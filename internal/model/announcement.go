package model

import (
	"time"

	"gorm.io/datatypes"
)

type Announcement struct {
	ID uint `gorm:"primaryKey" json:"id"`

	Title   string `gorm:"type:varchar(200);not null" json:"title"`
	Content string `gorm:"type:text;not null" json:"content"`

	Status string `gorm:"type:varchar(20);index;not null" json:"status"`

	AudienceType string         `gorm:"type:varchar(20);not null" json:"audience_type"`
	TargetScope  datatypes.JSON `gorm:"type:jsonb" json:"target_scope"`

	PublishedAt *time.Time `json:"published_at"`

	CreatedBy uint `gorm:"index;not null" json:"created_by"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}