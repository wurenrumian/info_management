package model

import "time"

// Class represents an academic class.
type Class struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ClassName     string    `gorm:"type:varchar(50);not null;index" json:"class_name"`
	Grade         string    `gorm:"type:varchar(10);index" json:"grade"`
	Major         string    `gorm:"type:varchar(100);index" json:"major"`
	CounselorID   *uint     `json:"counselor_id"`
	HeadStudentID *uint     `json:"head_student_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
