package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserBeforeCreateSetsDefaultGrade(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}))

	u := User{
		StudentID: "S-DEFAULT-GRADE",
		Name:      "Default Grade User",
		Role:      RoleStudent,
		ClassID:   1,
	}
	require.NoError(t, db.Create(&u).Error)

	var got User
	require.NoError(t, db.First(&got, u.ID).Error)
	require.Equal(t, DefaultUserGrade, got.Grade)
}
