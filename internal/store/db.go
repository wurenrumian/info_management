package store

import (
	"manage/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func OpenAndMigrate(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.User{}, &model.Class{}, &model.AdminLog{}, &model.KnowledgeItem{}, &model.KnowledgeAttachment{}, &model.Document{}, &model.NotificationTemplate{}, &model.NotificationLog{}, &model.UserSubscribe{}, &model.Announcement{}, &model.Approval{}, &model.ApprovalAction{}, &model.PartyflowStatus{}, &model.PartyflowEvent{}, &model.PartyflowReminderRule{}); err != nil {
		return nil, err
	}
	return db, nil
}
