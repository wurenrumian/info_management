package authz_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"manage/internal/auth"
	"manage/internal/model"
	"manage/internal/service/authz"
)

func TestBuildScope(t *testing.T) {
	student := authz.BuildScope(auth.Actor{Role: model.RoleStudent, UserID: 11})
	require.Equal(t, uint(11), student.SelfUserID)
	require.False(t, student.AllowAll)

	cadre := authz.BuildScope(auth.Actor{Role: model.RoleCadre, ClassID: 3})
	require.Equal(t, uint(3), cadre.ClassID)
	require.Empty(t, cadre.Grade)

	teacher := authz.BuildScope(auth.Actor{Role: model.RoleTeacher, ClassID: 5, Grade: "2023"})
	require.Equal(t, uint(5), teacher.ClassID)
	require.Equal(t, "2023", teacher.Grade)

	superAdmin := authz.BuildScope(auth.Actor{Role: model.RoleSuperAdmin})
	require.True(t, superAdmin.AllowAll)

	unknown := authz.BuildScope(auth.Actor{Role: 999})
	require.False(t, unknown.AllowAll)
	require.Zero(t, unknown.SelfUserID)
	require.Zero(t, unknown.ClassID)
	require.Empty(t, unknown.Grade)
}
