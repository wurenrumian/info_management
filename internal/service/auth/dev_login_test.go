package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"manage/internal/model"
)

func TestRegisterOrLoginCreatesDefaultClassBeforeUser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Class{}, &model.User{}))

	svc := NewDevLoginService(db, "test-secret")

	token, user, err := svc.RegisterOrLogin("2024201514", nil)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, user)
	require.Equal(t, uint(defaultDevClassID), user.ClassID)

	var class model.Class
	require.NoError(t, db.First(&class, defaultDevClassID).Error)
	require.Equal(t, defaultDevGrade, class.Grade)
	require.Equal(t, defaultDevMajor, class.Major)
}

func TestRegisterOrLoginUsesEffectiveClassGradeForToken(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Class{}, &model.User{}))

	require.NoError(t, db.Create(&model.Class{
		ID:        20,
		ClassName: "C20",
		Grade:     "2027",
		Major:     "CS",
	}).Error)
	require.NoError(t, db.Create(&model.User{
		ID:        200,
		StudentID: "S200",
		Name:      "u200",
		Role:      model.RoleStudent,
		ClassID:   20,
		Grade:     "2020",
	}).Error)

	svc := NewDevLoginService(db, "test-secret")
	token, user, err := svc.RegisterOrLogin("S200", nil)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, "2027", user.Grade)

	claims, err := ParseToken(token, "test-secret")
	require.NoError(t, err)
	require.Equal(t, "2027", claims.Grade)
}
