package profile

import (
	"encoding/json"
	"errors"
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
	me, err := svc.PatchMe(100, PatchMeInput{
		AvatarURL: &avatarURL,
		Bio:       &bio,
	})
	require.NoError(t, err)
	require.Equal(t, uint(100), me.ID)

	var user model.User
	require.NoError(t, db.First(&user, 100).Error)

	var attrs map[string]any
	require.NoError(t, json.Unmarshal([]byte(user.ProfileAttrs), &attrs))
	require.Equal(t, "blue", attrs["theme"])
	require.Equal(t, "https://example.com/a.png", attrs["avatar_url"])
	require.Equal(t, "new bio", attrs["bio"])
}

func TestPatchMeReturnsErrEmptyPatchSentinel(t *testing.T) {
	db := testutil.NewTestDB(t)
	svc := NewService(db)

	require.NoError(t, db.Create(&model.User{
		ID:        100,
		StudentID: "S100",
		Name:      "张三",
		Role:      model.RoleStudent,
		ClassID:   1,
		Grade:     "2023",
	}).Error)

	_, err := svc.PatchMe(100, PatchMeInput{})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrEmptyPatch))
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

func TestGetHomeBasicMatchesGetMeSemantics(t *testing.T) {
	db := testutil.NewTestDB(t)
	svc := NewService(db)

	require.NoError(t, db.Create(&model.User{
		ID:        100,
		StudentID: "S100",
		Name:      "张三",
		Role:      model.RoleStudent,
		ClassID:   1,
		Grade:     "2023",
		Major:     "人工智能",
		ProfileAttrs: []byte(`{
			"nickname":"阿三",
			"bio":"今天也在认真生活",
			"avatar_url":"https://example.com/avatar.png"
		}`),
	}).Error)

	actor := auth.Actor{UserID: 100, Role: model.RoleStudent, ClassID: 1, Grade: "2023"}
	me, err := svc.GetMe(actor)
	require.NoError(t, err)
	home, err := svc.GetHome(actor)
	require.NoError(t, err)

	require.Equal(t, me.ID, home.Basic.ID)
	require.Equal(t, me.StudentID, home.Basic.StudentID)
	require.Equal(t, me.RealName, home.Basic.RealName)
	require.Equal(t, me.Nickname, home.Basic.Nickname)
	require.Equal(t, me.Major, home.Basic.Major)
	require.Equal(t, me.Bio, home.Basic.Bio)
	require.Equal(t, me.AvatarURL, home.Basic.AvatarURL)
}
