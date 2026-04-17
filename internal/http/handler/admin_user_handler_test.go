package handler_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
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

func setupTestRouter(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Class{}, &model.User{}, &model.AdminLog{}))
	require.NoError(t, db.Create(&model.Class{ID: 1, ClassName: "C1", Grade: "2023", Major: "CS"}).Error)
	require.NoError(t, db.Create(&model.Class{ID: 2, ClassName: "C2", Grade: "2023", Major: "CS"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 100, StudentID: "S100", Name: "u100", Role: model.RoleStudent, ClassID: 1, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 101, StudentID: "S101", Name: "u101", Role: model.RoleStudent, ClassID: 2, Grade: "2023"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 999, StudentID: "A999", Name: "admin", Role: model.RoleSuperAdmin, ClassID: 1, Grade: "2023"}).Error)
	_ = router.New(db)
	return db
}

func TestGetMeOnlyReturnsSelf(t *testing.T) {
	db := setupTestRouter(t)
	r := router.New(db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	token := testutil.GenerateTestToken(100, 1, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "S100")
	require.NotContains(t, w.Body.String(), "S101")
}

func TestAdminUsersListRespectsScope(t *testing.T) {
	db := setupTestRouter(t)
	r := router.New(db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	token := testutil.GenerateTestToken(200, 2, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "S100")
	require.NotContains(t, w.Body.String(), "S101")

	var resp struct {
		Total int64 `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, int64(2), resp.Total)
}

func TestPatchUserWritesAdminLog(t *testing.T) {
	db := setupTestRouter(t)
	r := router.New(db)

	body := []byte(`{"name":"new-name"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/100", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	token := testutil.GenerateTestToken(999, 4, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var logs []model.AdminLog
	require.NoError(t, db.Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, uint(999), logs[0].AdminID)
	require.Equal(t, uint(100), logs[0].TargetID)
}

func TestPatchUserRejectsGradeField(t *testing.T) {
	db := setupTestRouter(t)
	r := router.New(db)

	body := []byte(`{"grade":"2099"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/100", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	token := testutil.GenerateTestToken(999, model.RoleSuperAdmin, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "grade is system-managed")
}

func TestPatchUserClassIDSyncsGrade(t *testing.T) {
	db := setupTestRouter(t)
	r := router.New(db)

	require.NoError(t, db.Create(&model.Class{ID: 3, ClassName: "C3", Grade: "2025", Major: "AI"}).Error)

	body := []byte(`{"class_id":3}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/100", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	token := testutil.GenerateTestToken(999, model.RoleSuperAdmin, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var user model.User
	require.NoError(t, db.First(&user, 100).Error)
	require.Equal(t, uint(3), user.ClassID)
	require.Equal(t, "2025", user.Grade)
}

func TestImportUsersOnlySuperAdmin(t *testing.T) {
	db := setupTestRouter(t)
	r := router.New(db)

	body := []byte(`{"users":[{"student_id":"S200","name":"u200","class_id":1}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	token := testutil.GenerateTestToken(200, model.RoleTeacher, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestImportUsersJSONSupportsPartialFailure(t *testing.T) {
	db := setupTestRouter(t)
	r := router.New(db)

	body := []byte(`{
		"users":[
			{"student_id":"S200","name":"u200","class_id":1,"role":1},
			{"student_id":"S100","name":"dup","class_id":1,"role":1},
			{"student_id":"S201","name":"bad-class","class_id":99999}
		]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	token := testutil.GenerateTestToken(999, model.RoleSuperAdmin, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"imported":1`)
	require.Contains(t, w.Body.String(), `"failed":2`)
	require.Contains(t, w.Body.String(), "duplicate student_id")
	require.Contains(t, w.Body.String(), "class not found")

	var user model.User
	require.NoError(t, db.Where("student_id = ?", "S200").First(&user).Error)
	require.Equal(t, "2023", user.Grade)
}

func TestImportUsersCSV(t *testing.T) {
	db := setupTestRouter(t)
	r := router.New(db)

	csv := "student_id,name,class_id,role,major,college,enrollment_year\nS210,u210,1,1,CS,Info,2023\n"
	req := buildImportFileRequest(t, "/api/v1/admin/users/import", "users.csv", []byte(csv))
	token := testutil.GenerateTestToken(999, model.RoleSuperAdmin, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"imported":1`)
	require.Contains(t, w.Body.String(), `"failed":0`)
}

func TestImportUsersXLSX(t *testing.T) {
	db := setupTestRouter(t)
	r := router.New(db)

	xlsx := buildUsersXLSX(t, [][]string{
		{"student_id", "name", "class_id", "role", "major", "college", "enrollment_year"},
		{"S220", "u220", "1", "1", "CS", "Info", "2023"},
	})
	req := buildImportFileRequest(t, "/api/v1/admin/users/import", "users.xlsx", xlsx)
	token := testutil.GenerateTestToken(999, model.RoleSuperAdmin, 1, "2023")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"imported":1`)
	require.Contains(t, w.Body.String(), `"failed":0`)
}

func buildImportFileRequest(t *testing.T, path, filename string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = fileWriter.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func buildUsersXLSX(t *testing.T, rows [][]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	addFile := func(name, content string) {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}

	addFile("[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
</Types>`)
	addFile("_rels/.rels", `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`)
	addFile("xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets>
</workbook>`)
	addFile("xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`)

	var sheet bytes.Buffer
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sheet.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for rIdx, row := range rows {
		sheet.WriteString(fmt.Sprintf(`<row r="%d">`, rIdx+1))
		for cIdx, cell := range row {
			col := string(rune('A' + cIdx))
			sheet.WriteString(fmt.Sprintf(`<c r="%s%d" t="inlineStr"><is><t>%s</t></is></c>`, col, rIdx+1, cell))
		}
		sheet.WriteString(`</row>`)
	}
	sheet.WriteString(`</sheetData></worksheet>`)
	addFile("xl/worksheets/sheet1.xml", sheet.String())

	require.NoError(t, zw.Close())
	return buf.Bytes()
}
