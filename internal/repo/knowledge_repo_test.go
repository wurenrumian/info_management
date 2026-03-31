package repo_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/model"
	"manage/internal/repo"
)

func TestKnowledgeRepoSearchAndUpdate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.KnowledgeItem{}))

	r := repo.NewKnowledgeRepo(db)

	item := &model.KnowledgeItem{
		Question:  "休学申请怎么办理",
		Answer:    "先联系辅导员并提交表格",
		Keywords:  datatypes.JSON(`["休学","申请"]`),
		CreatedBy: 999,
		UpdatedBy: 999,
	}
	require.NoError(t, r.Create(item))

	hit, total, err := r.SearchWithTotal("休学", 20, 0)
	require.NoError(t, err)
	require.Len(t, hit, 1)
	require.Equal(t, int64(1), total)
	require.Equal(t, item.ID, hit[0].ID)

	miss, totalMiss, err := r.SearchWithTotal("不存在关键词", 20, 0)
	require.NoError(t, err)
	require.Len(t, miss, 0)
	require.Equal(t, int64(0), totalMiss)

	require.NoError(t, r.UpdateByID(item.ID, map[string]any{"answer": "按学院流程提交审批"}))
	all, totalAll, err := r.ListWithTotal("", 20, 0)
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.Equal(t, int64(1), totalAll)
	require.Equal(t, "按学院流程提交审批", all[0].Answer)

	got, err := r.GetByID(item.ID)
	require.NoError(t, err)
	require.Equal(t, item.ID, got.ID)

	require.NoError(t, r.DeleteByID(item.ID))
	_, err = r.GetByID(item.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestKnowledgeRepoUpdateDeleteNotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.KnowledgeItem{}))

	r := repo.NewKnowledgeRepo(db)

	require.ErrorIs(t, r.UpdateByID(99999, map[string]any{"answer": "x"}), gorm.ErrRecordNotFound)
	require.ErrorIs(t, r.DeleteByID(99999), gorm.ErrRecordNotFound)
}
