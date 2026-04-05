package repo_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/model"
	"manage/internal/repo"
)

func TestKnowledgeAttachmentRepoCRUD(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.KnowledgeAttachment{}))

	r := repo.NewKnowledgeAttachmentRepo(db)

	require.NoError(t, r.CreateBatch([]model.KnowledgeAttachment{
		{KnowledgeID: 10, FileID: 100, CreatedBy: 1},
		{KnowledgeID: 10, FileID: 101, CreatedBy: 1},
	}))

	// duplicate should be ignored
	require.NoError(t, r.CreateBatch([]model.KnowledgeAttachment{
		{KnowledgeID: 10, FileID: 101, CreatedBy: 2},
	}))

	rows, err := r.ListByKnowledgeID(10)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	all, err := r.ListByKnowledgeIDs([]uint{10, 11})
	require.NoError(t, err)
	require.Len(t, all, 2)

	require.NoError(t, r.DeleteByKnowledgeAndFileID(10, 100))

	rows, err = r.ListByKnowledgeID(10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, uint(101), rows[0].FileID)
}

func TestKnowledgeAttachmentRepoDeleteNotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.KnowledgeAttachment{}))

	r := repo.NewKnowledgeAttachmentRepo(db)
	require.ErrorIs(t, r.DeleteByKnowledgeAndFileID(1, 2), gorm.ErrRecordNotFound)
}
