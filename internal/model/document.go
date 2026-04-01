package model

import "time"

// Document represents an uploaded file in the document library.
type Document struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"type:varchar(200);not null" json:"title"`
	FilePath    string    `gorm:"type:varchar(500);not null" json:"file_path"`
	FileSize    int64     `gorm:"not null" json:"file_size"`
	ContentType string    `gorm:"type:varchar(100);not null" json:"content_type"`
	UploaderID  uint      `gorm:"index;not null" json:"uploader_id"`
	CreatedAt   time.Time `json:"created_at"`
}
