package authz_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"manage/internal/model"
	"manage/internal/service/authz"
)

func TestAuthorizeMatrix(t *testing.T) {
	cases := []struct {
		role   int
		action string
		allow  bool
	}{
		{model.RoleStudent, authz.ActionGetMe, true},
		{model.RoleStudent, authz.ActionKnowledgeSearch, true},
		{model.RoleStudent, authz.ActionKnowledgeList, false},
		{model.RoleStudent, authz.ActionUsersList, false},
		{model.RoleCadre, authz.ActionUsersList, true},
		{model.RoleCadre, authz.ActionKnowledgeCreate, true},
		{model.RoleCadre, authz.ActionUsersPatch, false},
		{model.RoleTeacher, authz.ActionClassesGet, true},
		{model.RoleTeacher, authz.ActionKnowledgePatch, true},
		{model.RoleTeacher, authz.ActionClassesCreate, false},
		{model.RoleSuperAdmin, authz.ActionAdminLogsList, true},
		{999, authz.ActionGetMe, false},
	}

	for _, tc := range cases {
		require.Equal(t, tc.allow, authz.Authorize(tc.role, tc.action))
	}
}
