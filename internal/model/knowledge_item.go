package model

import (
	"time"

	"gorm.io/datatypes"
)

// KnowledgeItem is a structured FAQ entry for keyword-based knowledge search.
type KnowledgeItem struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Question    string         `gorm:"type:text;not null" json:"question"`
	Answer      string         `gorm:"type:text;not null" json:"answer"`
	ContentText string         `gorm:"type:text" json:"content_text"`
	Keywords    datatypes.JSON `gorm:"type:jsonb" json:"keywords"`
	Attachments datatypes.JSON `gorm:"type:jsonb" json:"attachments"`
	CreatedBy   uint           `gorm:"index;not null" json:"created_by"`
	UpdatedBy   uint           `gorm:"index;not null" json:"updated_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
