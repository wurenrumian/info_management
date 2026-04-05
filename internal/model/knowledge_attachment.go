package model

import "time"

// KnowledgeAttachment links one knowledge item with one uploaded document.
type KnowledgeAttachment struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	KnowledgeID uint      `gorm:"index:idx_knowledge_file,unique;not null" json:"knowledge_id"`
	FileID      uint      `gorm:"index:idx_knowledge_file,unique;not null" json:"file_id"`
	CreatedBy   uint      `gorm:"index;not null" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}
