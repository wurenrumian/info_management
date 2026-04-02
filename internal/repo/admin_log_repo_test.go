package repo_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/model"
	"manage/internal/repo"
)

func setupAdminLogTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.AdminLog{}))
	return db
}

func TestAdminLogRepoCreate(t *testing.T) {
	db := setupAdminLogTestDB(t)
	r := repo.NewAdminLogRepo(db)

	log := model.AdminLog{
		AdminID:    100,
		Action:     "knowledge.create",
		TargetType: "knowledge",
		TargetID:   1,
		IPAddress:  "127.0.0.1",
	}
	require.NoError(t, r.Create(log))

	var found model.AdminLog
	require.NoError(t, db.Where("admin_id = ? AND action = ?", 100, "knowledge.create").First(&found).Error)
	require.NotZero(t, found.ID)
	require.Equal(t, uint(100), found.AdminID)
	require.Equal(t, "knowledge.create", found.Action)
	require.Equal(t, "knowledge", found.TargetType)
	require.Equal(t, uint(1), found.TargetID)
	require.Equal(t, "127.0.0.1", found.IPAddress)
}

func TestAdminLogRepoList(t *testing.T) {
	db := setupAdminLogTestDB(t)
	r := repo.NewAdminLogRepo(db)

	logs := []model.AdminLog{
		{AdminID: 1, Action: "user.patch", TargetType: "user", TargetID: 10, IPAddress: "10.0.0.1"},
		{AdminID: 2, Action: "knowledge.create", TargetType: "knowledge", TargetID: 5, IPAddress: "10.0.0.2"},
		{AdminID: 1, Action: "class.create", TargetType: "class", TargetID: 3, IPAddress: "10.0.0.1"},
	}
	for _, log := range logs {
		require.NoError(t, r.Create(log))
	}

	result, err := r.List(20, 0)
	require.NoError(t, err)
	require.Len(t, result, 3)
	require.Equal(t, "class.create", result[0].Action)
	require.Equal(t, "knowledge.create", result[1].Action)
	require.Equal(t, "user.patch", result[2].Action)
}

func TestAdminLogRepoListWithPagination(t *testing.T) {
	db := setupAdminLogTestDB(t)
	r := repo.NewAdminLogRepo(db)

	for i := 0; i < 5; i++ {
		require.NoError(t, r.Create(model.AdminLog{
			AdminID:    1,
			Action:     "test.action",
			TargetType: "test",
			TargetID:   uint(i),
			IPAddress:  "127.0.0.1",
		}))
	}

	result, err := r.List(2, 0)
	require.NoError(t, err)
	require.Len(t, result, 2)

	result, err = r.List(2, 2)
	require.NoError(t, err)
	require.Len(t, result, 2)

	result, err = r.List(2, 4)
	require.NoError(t, err)
	require.Len(t, result, 1)
}

func TestAdminLogRepoListEmpty(t *testing.T) {
	db := setupAdminLogTestDB(t)
	r := repo.NewAdminLogRepo(db)

	result, err := r.List(10, 0)
	require.NoError(t, err)
	require.Empty(t, result)
}
