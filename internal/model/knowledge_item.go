package model

import (
	"time"

	"gorm.io/datatypes"
)

// KnowledgeItem is a structured FAQ entry for keyword-based knowledge search.
type KnowledgeItem struct {
	ID          uint           `gorm:"primaryKey"`
	Question    string         `gorm:"type:text;not null"`
	Answer      string         `gorm:"type:text;not null"`
	Keywords    datatypes.JSON `gorm:"type:jsonb"`
	Attachments datatypes.JSON `gorm:"type:jsonb"`
	CreatedBy   uint           `gorm:"index;not null"`
	UpdatedBy   uint           `gorm:"index;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

