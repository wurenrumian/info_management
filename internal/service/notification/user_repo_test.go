package notification

import (
	"testing"

	"manage/internal/model"
	"manage/internal/testutil"

	"github.com/stretchr/testify/require"
)

func TestGormUserRepoGetUserOpenID(t *testing.T) {
	db := testutil.NewTestDB(t)
	openID := "wx-openid-2"
	require.NoError(t, db.Create(&model.User{
		ID:        20,
		StudentID: "S20",
		Name:      "u20",
		Role:      model.RoleStudent,
		ClassID:   1,
		Grade:     "2023",
		OpenID:    &openID,
	}).Error)

	repo := NewGormUserRepo(db)
	got, err := repo.GetUserOpenID(20)
	require.NoError(t, err)
	require.Equal(t, openID, got)
}
