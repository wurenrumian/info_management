package repo

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"manage/internal/model"
)

func TestUpdateByIDUpdatesRecord(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, db.Create(&model.User{
		ID:        1,
		StudentID: "S001",
		Name:      "old",
		Role:      model.RoleStudent,
		ClassID:   1,
		Grade:     "2023",
	}).Error)

	require.NoError(t, UpdateByID(db.Model(&model.User{}), 1, map[string]any{
		"name": "new",
	}))

	var user model.User
	require.NoError(t, db.First(&user, 1).Error)
	require.Equal(t, "new", user.Name)
}

func TestUpdateByIDReturnsRecordNotFoundWhenNoRowsAffected(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))

	err = UpdateByID(db.Model(&model.User{}), 999, map[string]any{
		"name": "new",
	})
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
