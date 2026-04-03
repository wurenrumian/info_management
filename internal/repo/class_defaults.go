package repo

import (
	"errors"

	"gorm.io/gorm"

	"manage/internal/model"
)

// EnsureClass ensures a class with the given id exists.
func EnsureClass(db *gorm.DB, id uint, className, grade, major string) error {
	var existing model.Class
	err := db.Select("id").Where("id = ?", id).First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return db.Create(&model.Class{
		ID:        id,
		ClassName: className,
		Grade:     grade,
		Major:     major,
	}).Error
}
