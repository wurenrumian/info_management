package profile

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"manage/internal/auth"
	"manage/internal/model"
	"manage/internal/testutil"
)

func TestPatchMeMergesProfileAttrs(t *testing.T) {
	db := testutil.NewTestDB(t)
	svc := NewService(db)

	require.NoError(t, db.Create(&model.User{
		ID:           100,
		StudentID:    "S100",
		Name:         "张三",
		Role:         model.RoleStudent,
		ClassID:      1,
		Grade:        "2023",
		ProfileAttrs: []byte(`{"theme":"blue","bio":"old"}`),
	}).Error)

	avatarURL := "https://example.com/a.png"
	bio := "new bio"
	require.NoError(t, svc.PatchMe(100, &avatarURL, &bio))

	var user model.User
	require.NoError(t, db.First(&user, 100).Error)

	var attrs map[string]any
	require.NoError(t, json.Unmarshal([]byte(user.ProfileAttrs), &attrs))
	require.Equal(t, "blue", attrs["theme"])
	require.Equal(t, "https://example.com/a.png", attrs["avatar_url"])
	require.Equal(t, "new bio", attrs["bio"])
}

func TestGetHomeReturnsKnowledgeCountAndWechatBoundFlag(t *testing.T) {
	db := testutil.NewTestDB(t)
	svc := NewService(db)

	openID := "openid-1"
	require.NoError(t, db.Create(&model.User{
		ID:        100,
		StudentID: "S100",
		Name:      "张三",
		Role:      model.RoleStudent,
		ClassID:   1,
		Grade:     "2023",
		OpenID:    &openID,
	}).Error)
	require.NoError(t, db.Create(&model.KnowledgeItem{
		Question:  "Q1",
		Answer:    "A1",
		CreatedBy: 100,
		UpdatedBy: 100,
	}).Error)

	data, err := svc.GetHome(auth.Actor{UserID: 100, Role: model.RoleStudent, ClassID: 1, Grade: "2023"})
	require.NoError(t, err)
	require.Equal(t, uint(100), data.Basic.ID)
	require.Equal(t, int64(1), data.QuickEntry.KnowledgeCount)
	require.True(t, data.Account.WechatBound)
}
