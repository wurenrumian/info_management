package model

import "time"

type Class struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ClassName     string    `gorm:"size:50;not null;index" json:"class_name"`
	Grade         string    `gorm:"size:10;index" json:"grade"`
	Major         string    `gorm:"size:100;index" json:"major"`
	CounselorID   *uint     `json:"counselor_id"`
	HeadStudentID *uint     `json:"head_student_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
