package model

import (
	"time"

	"gorm.io/datatypes"
)

// Announcement 表示公告/通知内容。
type Announcement struct {
	ID uint `gorm:"primaryKey" json:"id"`

	Title   string `gorm:"type:varchar(200);not null" json:"title"`
	Content string `gorm:"type:text;not null" json:"content"`

	// draft / published / archived
	Status string `gorm:"type:varchar(20);index;not null" json:"status"`

	// all / targeted
	AudienceType string         `gorm:"type:varchar(20);not null" json:"audience_type"`
	TargetScope  datatypes.JSON `gorm:"type:jsonb" json:"target_scope"`

	Tags              datatypes.JSON `gorm:"type:jsonb" json:"tags"`
	AttachmentFileIDs datatypes.JSON `gorm:"type:jsonb" json:"attachment_file_ids"`
	ExternalLinks     datatypes.JSON `gorm:"type:jsonb" json:"external_links"`

	AuthorID    uint       `gorm:"index;not null" json:"author_id"`
	PublishedAt *time.Time `json:"published_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
