//go:build integration

package repo_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"manage/internal/model"
	"manage/internal/repo"
)

func TestKnowledgeRepoSearchWithKingbase(t *testing.T) {
	dsn := os.Getenv("KINGBASE_DSN")
	if dsn == "" {
		t.Skip("KINGBASE_DSN is empty; skip integration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetConnMaxLifetime(2 * time.Minute)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetMaxOpenConns(5)

	require.NoError(t, db.AutoMigrate(&model.KnowledgeItem{}))
	require.NoError(t, db.Exec("TRUNCATE TABLE knowledge_items RESTART IDENTITY").Error)

	r := repo.NewKnowledgeRepo(db)
	require.NoError(t, r.Create(&model.KnowledgeItem{
		Question:  "休学申请怎么办理",
		Answer:    "先联系辅导员并提交休学申请表",
		Keywords:  datatypes.JSON(`["休学","申请"]`),
		CreatedBy: 1,
		UpdatedBy: 1,
	}))
	require.NoError(t, r.Create(&model.KnowledgeItem{
		Question:  "奖学金评定流程",
		Answer:    "按学院通知提交材料",
		Keywords:  datatypes.JSON(`["奖学金","评定"]`),
		CreatedBy: 1,
		UpdatedBy: 1,
	}))
	require.NoError(t, r.Create(&model.KnowledgeItem{
		Question:  "奖学金申请材料有哪些",
		Answer:    "提交综测排名证明、成绩单和个人陈述",
		Keywords:  datatypes.JSON(`["奖学金","申请","材料"]`),
		CreatedBy: 1,
		UpdatedBy: 1,
	}))

	hits, err := r.Search("休学 申请", 20, 0)
	require.NoError(t, err)
	require.NotEmpty(t, hits)
	require.Equal(t, "休学申请怎么办理", hits[0].Question)

	miss, err := r.Search("完全不存在的关键词", 20, 0)
	require.NoError(t, err)
	require.Empty(t, miss)

	cnPhraseHits, err := r.Search("奖学金申请", 20, 0)
	require.NoError(t, err)
	require.NotEmpty(t, cnPhraseHits)
}
