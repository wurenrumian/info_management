package repo_test

import (
	"testing"
	"time"

	"manage/internal/model"
	"manage/internal/repo"
	"manage/internal/service/authz"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupApprovalRepoDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Approval{}, &model.ApprovalAction{}))
	require.NoError(t, db.Create(&model.User{ID: 100, StudentID: "S100", Name: "u1", Role: model.RoleStudent, ClassID: 1, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 101, StudentID: "S101", Name: "u2", Role: model.RoleStudent, ClassID: 2, Grade: "2022"}).Error)
	return db
}

func TestApprovalRepoGetByIDInScope(t *testing.T) {
	db := setupApprovalRepoDB(t)
	r := repo.NewApprovalRepo(db)

	now := time.Now()
	a1 := &model.Approval{ApplicantID: 100, ApprovalType: model.ApprovalTypeLeave, Status: model.ApprovalStatusPending, Title: "A1", Semester: "2026-1", SubmittedAt: now}
	a2 := &model.Approval{ApplicantID: 101, ApprovalType: model.ApprovalTypeLeave, Status: model.ApprovalStatusPending, Title: "A2", Semester: "2026-1", SubmittedAt: now}
	require.NoError(t, r.Create(a1))
	require.NoError(t, r.Create(a2))

	_, err := r.GetByIDInScope(authz.Scope{SelfUserID: 100}, a1.ID)
	require.NoError(t, err)

	_, err = r.GetByIDInScope(authz.Scope{SelfUserID: 100}, a2.ID)
	require.Error(t, err)
}

func TestApprovalRepoListOverduePending(t *testing.T) {
	db := setupApprovalRepoDB(t)
	r := repo.NewApprovalRepo(db)

	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	require.NoError(t, r.Create(&model.Approval{
		ApplicantID: 100, ApprovalType: model.ApprovalTypeLeave, Status: model.ApprovalStatusPending,
		Title: "overdue", Semester: "2026-1", SubmittedAt: time.Now(), DueAt: &past,
	}))
	require.NoError(t, r.Create(&model.Approval{
		ApplicantID: 100, ApprovalType: model.ApprovalTypeLeave, Status: model.ApprovalStatusApproved,
		Title: "approved", Semester: "2026-1", SubmittedAt: time.Now(), DueAt: &past,
	}))
	require.NoError(t, r.Create(&model.Approval{
		ApplicantID: 100, ApprovalType: model.ApprovalTypeLeave, Status: model.ApprovalStatusPending,
		Title: "future", Semester: "2026-1", SubmittedAt: time.Now(), DueAt: &future,
	}))

	list, err := r.ListOverduePending(time.Now(), 20)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "overdue", list[0].Title)
}
