package repo_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/model"
	"manage/internal/repo"
)

func TestDocumentRepoCRUD(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Document{}))

	r := repo.NewDocumentRepo(db)

	doc := &model.Document{
		Title:       "report.pdf",
		FilePath:    "2026/04/123_report.pdf",
		FileSize:    1024,
		ContentType: "application/pdf",
		UploaderID:  100,
	}
	require.NoError(t, r.Create(doc))
	require.Greater(t, doc.ID, uint(0))

	got, err := r.GetByID(doc.ID)
	require.NoError(t, err)
	require.Equal(t, doc.Title, got.Title)
	require.Equal(t, doc.FilePath, got.FilePath)

	docs, total, err := r.ListWithTotal(20, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, docs, 1)

	byIDs, err := r.ListByIDs([]uint{doc.ID, 9999})
	require.NoError(t, err)
	require.Len(t, byIDs, 1)
	require.Equal(t, doc.ID, byIDs[0].ID)

	require.NoError(t, r.DeleteByID(doc.ID))
	_, err = r.GetByID(doc.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestDocumentRepoSearchWithTotal(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Document{}))

	r := repo.NewDocumentRepo(db)
	require.NoError(t, r.Create(&model.Document{
		Title:       "奖学金指南.docx",
		FilePath:    "documents/2026/04/a.docx",
		FileSize:    1024,
		ContentType: "application/msword",
		ContentText: "奖学金申请需要提交综测排名证明和成绩单",
		UploaderID:  100,
	}))
	require.NoError(t, r.Create(&model.Document{
		Title:       "请假流程.pdf",
		FilePath:    "documents/2026/04/b.pdf",
		FileSize:    1024,
		ContentType: "application/pdf",
		ContentText: "请假流程需要先联系辅导员",
		UploaderID:  100,
	}))

	hits, total, err := r.SearchWithTotal("综测排名证明", 20, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, hits, 1)
	require.Equal(t, "奖学金指南.docx", hits[0].Title)
}

func TestDocumentRepoDeleteNotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Document{}))

	r := repo.NewDocumentRepo(db)
	require.ErrorIs(t, r.DeleteByID(99999), gorm.ErrRecordNotFound)
}
