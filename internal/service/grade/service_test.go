package grade

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"manage/internal/model"
)

func TestResolveEffectiveGrade_PreferClassGradeThenFallbackUserGrade(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Class{}))
	svc := NewService(db)

	require.NoError(t, db.Create(&model.Class{
		ID:        10,
		ClassName: "C10",
		Grade:     "2026",
		Major:     "CS",
	}).Error)
	require.NoError(t, db.Create(&model.User{
		ID:        100,
		StudentID: "S100",
		Name:      "u100",
		Role:      model.RoleStudent,
		ClassID:   10,
		Grade:     "2023",
	}).Error)
	require.NoError(t, db.Create(&model.User{
		ID:        101,
		StudentID: "S101",
		Name:      "u101",
		Role:      model.RoleStudent,
		ClassID:   999,
		Grade:     "2022",
	}).Error)

	var userWithClass model.User
	require.NoError(t, db.First(&userWithClass, 100).Error)
	got, err := svc.ResolveEffectiveGrade(&userWithClass)
	require.NoError(t, err)
	require.Equal(t, "2026", got)

	var userMissingClass model.User
	require.NoError(t, db.First(&userMissingClass, 101).Error)
	got, err = svc.ResolveEffectiveGrade(&userMissingClass)
	require.NoError(t, err)
	require.Equal(t, "2022", got)
}
