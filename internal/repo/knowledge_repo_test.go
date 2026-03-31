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

	hit, err := r.Search("休学", 20, 0)
	require.NoError(t, err)
	require.Len(t, hit, 1)
	require.Equal(t, item.ID, hit[0].ID)

	miss, err := r.Search("不存在关键词", 20, 0)
	require.NoError(t, err)
	require.Len(t, miss, 0)

	require.NoError(t, r.UpdateByID(item.ID, map[string]any{"answer": "按学院流程提交审批"}))
	all, err := r.List("", 20, 0)
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.Equal(t, "按学院流程提交审批", all[0].Answer)
}

