package announcements

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"manage/internal/auth"
	"manage/internal/model"
)

func setupAnnouncementService(t *testing.T) (*gorm.DB, *Service) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Class{}, &model.User{}, &model.Announcement{}))
	return db, NewService(db, nil)
}

func TestListForStudentFiltersPublishedAndCrossScope(t *testing.T) {
	db, svc := setupAnnouncementService(t)

	classA := model.Class{ID: 1, ClassName: "信管1班", Grade: "2023", Major: "信息管理"}
	classB := model.Class{ID: 2, ClassName: "计科2班", Grade: "2022", Major: "计算机"}
	require.NoError(t, db.Create(&classA).Error)
	require.NoError(t, db.Create(&classB).Error)

	student := model.User{
		ID:        101,
		StudentID: "S101",
		Name:      "Alice",
		Role:      model.RoleStudent,
		ClassID:   classA.ID,
		Grade:     "2023",
		Major:     "信息管理",
	}
	require.NoError(t, db.Create(&student).Error)

	scopeIn := datatypes.JSON(`{"grades":["2023"],"majors":["信息管理"],"class_ids":[1],"roles":[1]}`)
	scopeCrossGrade := datatypes.JSON(`{"grades":["2022"]}`)
	scopeCrossClass := datatypes.JSON(`{"class_ids":[2]}`)

	rows := []model.Announcement{
		{
			Title:        "全员已发布",
			Content:      "all published",
			Status:       StatusPublished,
			AudienceType: AudienceAll,
			TargetScope:  datatypes.JSON(`{}`),
			AuthorID:     900,
		},
		{
			Title:        "命中范围已发布",
			Content:      "in scope",
			Status:       StatusPublished,
			AudienceType: AudienceTargeted,
			TargetScope:  scopeIn,
			AuthorID:     900,
		},
		{
			Title:        "跨年级不命中",
			Content:      "cross grade",
			Status:       StatusPublished,
			AudienceType: AudienceTargeted,
			TargetScope:  scopeCrossGrade,
			AuthorID:     900,
		},
		{
			Title:        "跨班级不命中",
			Content:      "cross class",
			Status:       StatusPublished,
			AudienceType: AudienceTargeted,
			TargetScope:  scopeCrossClass,
			AuthorID:     900,
		},
		{
			Title:        "草稿不应返回",
			Content:      "draft",
			Status:       StatusDraft,
			AudienceType: AudienceAll,
			TargetScope:  datatypes.JSON(`{}`),
			AuthorID:     900,
		},
	}
	require.NoError(t, db.Create(&rows).Error)

	actor := auth.Actor{
		UserID:  student.ID,
		Role:    model.RoleStudent,
		ClassID: classA.ID,
		Grade:   "2023",
	}
	list, total, err := svc.ListForStudent(actor, 20, 0)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, list, 2)

	gotTitles := map[string]bool{}
	for _, item := range list {
		gotTitles[item.Title] = true
		require.Equal(t, StatusPublished, item.Status)
	}
	require.True(t, gotTitles["全员已发布"])
	require.True(t, gotTitles["命中范围已发布"])
	require.False(t, gotTitles["跨年级不命中"])
	require.False(t, gotTitles["跨班级不命中"])
	require.False(t, gotTitles["草稿不应返回"])
}

func TestMatchAudienceAll(t *testing.T) {
	ok, err := matchAnnouncementForActor(model.Announcement{
		AudienceType: AudienceAll,
	}, auth.Actor{
		UserID:  100,
		Role:    model.RoleStudent,
		ClassID: 1,
		Grade:   "2023",
	}, "信息管理")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestMatchAudienceTargetedGrades(t *testing.T) {
	ok, err := matchAnnouncementForActor(model.Announcement{
		AudienceType: AudienceTargeted,
		TargetScope:  datatypes.JSON(`{"grades":["2023"]}`),
	}, auth.Actor{
		UserID:  100,
		Role:    model.RoleStudent,
		ClassID: 1,
		Grade:   "2023",
	}, "信息管理")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestMatchAudienceTargetedMajors(t *testing.T) {
	ok, err := matchAnnouncementForActor(model.Announcement{
		AudienceType: AudienceTargeted,
		TargetScope:  datatypes.JSON(`{"majors":["信息管理"]}`),
	}, auth.Actor{
		UserID:  100,
		Role:    model.RoleStudent,
		ClassID: 1,
		Grade:   "2023",
	}, "信息管理")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestMatchAudienceTargetedClassIDs(t *testing.T) {
	ok, err := matchAnnouncementForActor(model.Announcement{
		AudienceType: AudienceTargeted,
		TargetScope:  datatypes.JSON(`{"class_ids":[1]}`),
	}, auth.Actor{
		UserID:  100,
		Role:    model.RoleStudent,
		ClassID: 1,
		Grade:   "2023",
	}, "信息管理")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestPublishSetsPublishedAt(t *testing.T) {
	db, svc := setupAnnouncementService(t)

	item := model.Announcement{
		Title:        "待发布公告",
		Content:      "正文",
		Status:       StatusDraft,
		AudienceType: AudienceAll,
		TargetScope:  datatypes.JSON(`{}`),
		AuthorID:     900,
	}
	require.NoError(t, db.Create(&item).Error)

	before := time.Now()
	got, err := svc.Publish(context.Background(), item.ID, PublishRequest{})
	require.NoError(t, err)
	require.Equal(t, StatusPublished, got.Status)
	require.NotNil(t, got.PublishedAt)
	require.False(t, got.PublishedAt.Before(before))

	var stored model.Announcement
	require.NoError(t, db.First(&stored, item.ID).Error)
	require.Equal(t, StatusPublished, stored.Status)
	require.NotNil(t, stored.PublishedAt)
}

func TestArchiveStatusCannotBePublished(t *testing.T) {
	db, svc := setupAnnouncementService(t)

	item := model.Announcement{
		Title:        "已归档公告",
		Content:      "正文",
		Status:       StatusArchived,
		AudienceType: AudienceAll,
		TargetScope:  datatypes.JSON(`{}`),
		AuthorID:     900,
	}
	require.NoError(t, db.Create(&item).Error)

	_, err := svc.Publish(context.Background(), item.ID, PublishRequest{})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrAnnouncementState))
}
