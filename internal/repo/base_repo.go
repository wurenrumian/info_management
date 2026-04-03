package repo

import "gorm.io/gorm"

// UpdateByID updates a single record by id and returns ErrRecordNotFound
// when no rows are affected.
func UpdateByID(query *gorm.DB, id uint, updates map[string]any) error {
	tx := query.Where("id = ?", id).Updates(updates)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
