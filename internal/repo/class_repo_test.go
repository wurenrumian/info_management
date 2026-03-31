package repo_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/authz"
)

func TestClassRepoListByScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Class{}))

	require.NoError(t, db.Create(&model.Class{ID: 10, ClassName: "A", Grade: "2023", Major: "CS"}).Error)
	require.NoError(t, db.Create(&model.Class{ID: 11, ClassName: "B", Grade: "2023", Major: "CS"}).Error)
	require.NoError(t, db.Create(&model.Class{ID: 12, ClassName: "C", Grade: "2022", Major: "SE"}).Error)

	r := repo.NewClassRepo(db)

	oneClass, err := r.ListByScope(authz.Scope{ClassID: 12}, 20, 0)
	require.NoError(t, err)
	require.Len(t, oneClass, 1)
	require.Equal(t, uint(12), oneClass[0].ID)

	classOrGrade, err := r.ListByScope(authz.Scope{ClassID: 12, Grade: "2023"}, 20, 0)
	require.NoError(t, err)
	require.Len(t, classOrGrade, 3)

	gradeOnly, err := r.ListByScope(authz.Scope{Grade: "2023"}, 20, 0)
	require.NoError(t, err)
	require.Len(t, gradeOnly, 2)
	ids := []uint{gradeOnly[0].ID, gradeOnly[1].ID}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	require.Equal(t, []uint{10, 11}, ids)

	all, err := r.ListByScope(authz.Scope{AllowAll: true}, 20, 0)
	require.NoError(t, err)
	require.Len(t, all, 3)

	none, err := r.ListByScope(authz.Scope{}, 20, 0)
	require.NoError(t, err)
	require.Len(t, none, 0)
}
