package tests

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"manage/internal/http/router"
	"manage/internal/model"
	"manage/internal/testutil"
)

func setupContractRouter(t *testing.T) http.Handler {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Class{}, &model.User{}, &model.AdminLog{}))

	require.NoError(t, db.Create(&model.Class{ID: 1, ClassName: "C1", Grade: "2023", Major: "CS"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 100, StudentID: "S100", Name: "student", Role: model.RoleStudent, ClassID: 1, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 200, StudentID: "C200", Name: "cadre", Role: model.RoleCadre, ClassID: 1, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 300, StudentID: "T300", Name: "teacher", Role: model.RoleTeacher, ClassID: 1, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 999, StudentID: "A999", Name: "admin", Role: model.RoleSuperAdmin, ClassID: 1, Grade: "2023"}).Error)

	return router.New(db)
}

func TestPhase1Contract_RoleMatrix(t *testing.T) {
	r := setupContractRouter(t)

	type testCase struct {
		name    string
		method  string
		path    string
		body    []byte
		token   string
		content bool
		want    int
	}

	cases := []testCase{
		{
			name:   "student can get me",
			method: http.MethodGet, path: "/api/v1/me", want: http.StatusOK,
			token: testutil.GenerateTestToken(100, 1, 1, "2023"),
		},
		{
			name:   "student cannot list users",
			method: http.MethodGet, path: "/api/v1/admin/users", want: http.StatusForbidden,
			token: testutil.GenerateTestToken(100, 1, 1, "2023"),
		},
		{
			name:   "cadre can list users",
			method: http.MethodGet, path: "/api/v1/admin/users", want: http.StatusOK,
			token: testutil.GenerateTestToken(200, 2, 1, "2023"),
		},
		{
			name:   "cadre cannot patch users",
			method: http.MethodPatch, path: "/api/v1/admin/users/100", body: []byte(`{"name":"x"}`), want: http.StatusForbidden,
			token:   testutil.GenerateTestToken(200, 2, 1, "2023"),
			content: true,
		},
		{
			name:   "teacher can list classes",
			method: http.MethodGet, path: "/api/v1/admin/classes", want: http.StatusOK,
			token: testutil.GenerateTestToken(300, 3, 1, "2023"),
		},
		{
			name:   "teacher cannot create classes",
			method: http.MethodPost, path: "/api/v1/admin/classes", body: []byte(`{"class_name":"N1","grade":"2024","major":"CS"}`), want: http.StatusForbidden,
			token:   testutil.GenerateTestToken(300, 3, 1, "2023"),
			content: true,
		},
		{
			name:   "superadmin can patch users",
			method: http.MethodPatch, path: "/api/v1/admin/users/100", body: []byte(`{"name":"updated"}`), want: http.StatusOK,
			token:   testutil.GenerateTestToken(999, 4, 1, "2023"),
			content: true,
		},
		{
			name:   "superadmin can list logs",
			method: http.MethodGet, path: "/api/v1/admin/logs", want: http.StatusOK,
			token: testutil.GenerateTestToken(999, 4, 1, "2023"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+tc.token)
			if tc.content {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code)
		})
	}
}
