package repo_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"manage/internal/model"
	"manage/internal/repo"
)

func setupAnnouncementRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Announcement{}))
	return db
}

func TestAnnouncementRepoCreateAndGetByID(t *testing.T) {
	db := setupAnnouncementRepoTestDB(t)
	r := repo.NewAnnouncementRepo(db)

	item := &model.Announcement{
		Title:        "公告标题",
		Content:      "公告正文",
		Status:       "draft",
		AudienceType: "all",
		TargetScope:  datatypes.JSON(`{}`),
		AuthorID:     999,
	}
	require.NoError(t, r.Create(item))

	got, err := r.GetByID(item.ID)
	require.NoError(t, err)
	require.Equal(t, item.Title, got.Title)
	require.Equal(t, item.Status, got.Status)
}

func TestAnnouncementRepoListWithStatusFilter(t *testing.T) {
	db := setupAnnouncementRepoTestDB(t)
	r := repo.NewAnnouncementRepo(db)

	items := []model.Announcement{
		{Title: "已发布1", Content: "x", Status: "published", AudienceType: "all", TargetScope: datatypes.JSON(`{}`), AuthorID: 1},
		{Title: "草稿1", Content: "x", Status: "draft", AudienceType: "all", TargetScope: datatypes.JSON(`{}`), AuthorID: 1},
		{Title: "已发布2", Content: "x", Status: "published", AudienceType: "all", TargetScope: datatypes.JSON(`{}`), AuthorID: 1},
	}
	require.NoError(t, db.Create(&items).Error)

	list, total, err := r.ListWithTotal("published", 20, 0)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, list, 2)
	require.Equal(t, "已发布2", list[0].Title)
	require.Equal(t, "已发布1", list[1].Title)
}

func TestAnnouncementRepoPublishSetsPublishedAt(t *testing.T) {
	db := setupAnnouncementRepoTestDB(t)
	r := repo.NewAnnouncementRepo(db)

	item := &model.Announcement{
		Title:        "待发布",
		Content:      "x",
		Status:       "draft",
		AudienceType: "all",
		TargetScope:  datatypes.JSON(`{}`),
		AuthorID:     1,
	}
	require.NoError(t, r.Create(item))

	now := time.Now().Round(time.Second)
	require.NoError(t, r.Publish(item.ID, now))

	got, err := r.GetByID(item.ID)
	require.NoError(t, err)
	require.Equal(t, "published", got.Status)
	require.NotNil(t, got.PublishedAt)
	require.WithinDuration(t, now, *got.PublishedAt, time.Second)
}

func TestAnnouncementRepoArchiveUpdatesStatus(t *testing.T) {
	db := setupAnnouncementRepoTestDB(t)
	r := repo.NewAnnouncementRepo(db)

	item := &model.Announcement{
		Title:        "待归档",
		Content:      "x",
		Status:       "published",
		AudienceType: "all",
		TargetScope:  datatypes.JSON(`{}`),
		AuthorID:     1,
	}
	require.NoError(t, r.Create(item))

	require.NoError(t, r.Archive(item.ID))

	got, err := r.GetByID(item.ID)
	require.NoError(t, err)
	require.Equal(t, "archived", got.Status)
}
