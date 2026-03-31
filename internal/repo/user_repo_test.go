package repo_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/authz"
)

func TestUserRepoListByScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Class{}, &model.User{}))

	require.NoError(t, db.Create(&model.User{ID: 1, StudentID: "S1", Name: "u1", Role: 1, ClassID: 10, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, StudentID: "S2", Name: "u2", Role: 1, ClassID: 11, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 3, StudentID: "S3", Name: "u3", Role: 1, ClassID: 12, Grade: "2022"}).Error)

	r := repo.NewUserRepo(db)
	selfOnly, err := r.ListByScope(authz.Scope{SelfUserID: 1}, 20, 0)
	require.NoError(t, err)
	require.Len(t, selfOnly, 1)
	require.Equal(t, uint(1), selfOnly[0].ID)

	classOnly, err := r.ListByScope(authz.Scope{ClassID: 11}, 20, 0)
	require.NoError(t, err)
	require.Len(t, classOnly, 1)
	require.Equal(t, uint(2), classOnly[0].ID)

	classOrGrade, err := r.ListByScope(authz.Scope{ClassID: 12, Grade: "2023"}, 20, 0)
	require.NoError(t, err)
	require.Len(t, classOrGrade, 3)

	all, err := r.ListByScope(authz.Scope{AllowAll: true}, 20, 0)
	require.NoError(t, err)
	require.Len(t, all, 3)

	none, err := r.ListByScope(authz.Scope{}, 20, 0)
	require.NoError(t, err)
	require.Len(t, none, 0)
}
