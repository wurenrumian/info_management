package model

import "time"

type Class struct {
	ID            uint      `gorm:"primaryKey"`
	ClassName     string    `gorm:"size:50;not null;index"`
	Grade         string    `gorm:"size:10;index"`
	Major         string    `gorm:"size:100;index"`
	CounselorID   *uint
	HeadStudentID *uint
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
