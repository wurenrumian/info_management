package userimport

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"manage/internal/model"
	"manage/internal/repo"
	gradesvc "manage/internal/service/grade"
)

var (
	ErrInvalidJSONBody    = errors.New("invalid json body")
	ErrUnsupportedFile    = errors.New("unsupported file type")
	ErrInvalidCSV         = errors.New("invalid csv file")
	ErrInvalidXLSX        = errors.New("invalid xlsx file")
	ErrMissingHeader      = errors.New("missing required header")
	ErrMissingRequired    = errors.New("missing required fields")
	ErrInvalidClassID     = errors.New("invalid class_id")
	ErrInvalidRole        = errors.New("invalid role")
	ErrInvalidEnrollYear  = errors.New("invalid enrollment_year")
	ErrDuplicateStudentID = errors.New("duplicate student_id")
	ErrClassNotFound      = errors.New("class not found")
)

type ImportRow struct {
	Index          int
	StudentID      string
	Name           string
	ClassID        uint
	Role           int
	Major          string
	College        string
	EnrollmentYear int
}

type ErrorItem struct {
	Row       int    `json:"row"`
	StudentID string `json:"student_id,omitempty"`
	Error     string `json:"error"`
}

type Result struct {
	Imported int         `json:"imported"`
	Failed   int         `json:"failed"`
	Errors   []ErrorItem `json:"errors"`
}

type Service struct {
	userRepo  *repo.UserRepo
	classRepo *repo.ClassRepo
	gradeSvc  *gradesvc.Service
}

func NewService(db *gorm.DB) *Service {
	return &Service{
		userRepo:  repo.NewUserRepo(db),
		classRepo: repo.NewClassRepo(db),
		gradeSvc:  gradesvc.NewService(db),
	}
}

type jsonBody struct {
	Users []map[string]any `json:"users"`
}

func (s *Service) ParseJSONPayload(body []byte) ([]ImportRow, error) {
	var req jsonBody
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, ErrInvalidJSONBody
	}
	return parseRows(req.Users, 1)
}

func (s *Service) ParseCSVPayload(body []byte) ([]ImportRow, error) {
	reader := csv.NewReader(bytes.NewReader(body))
	reader.TrimLeadingSpace = true
	rows, err := reader.ReadAll()
	if err != nil || len(rows) == 0 {
		return nil, ErrInvalidCSV
	}

	header := normalizeHeader(rows[0])
	required := []string{"student_id", "name", "class_id"}
	for _, key := range required {
		if _, ok := header[key]; !ok {
			return nil, ErrMissingHeader
		}
	}

	records := make([]map[string]any, 0, len(rows)-1)
	for i := 1; i < len(rows); i++ {
		if isEmptyRow(rows[i]) {
			continue
		}
		record := map[string]any{}
		for key, idx := range header {
			if idx >= len(rows[i]) {
				record[key] = ""
				continue
			}
			record[key] = strings.TrimSpace(rows[i][idx])
		}
		record["_row_index"] = i + 1
		records = append(records, record)
	}
	return parseRows(records, 2)
}

func (s *Service) ParseXLSXPayload(body []byte) ([]ImportRow, error) {
	readerAt := bytes.NewReader(body)
	zr, err := zip.NewReader(readerAt, int64(len(body)))
	if err != nil {
		return nil, ErrInvalidXLSX
	}

	shared, _ := readSharedStrings(zr.File)
	sheetXML, err := readFirstWorksheet(zr.File)
	if err != nil || len(sheetXML) == 0 {
		return nil, ErrInvalidXLSX
	}

	table, err := parseWorksheet(sheetXML, shared)
	if err != nil || len(table) == 0 {
		return nil, ErrInvalidXLSX
	}

	header := normalizeHeader(table[0])
	required := []string{"student_id", "name", "class_id"}
	for _, key := range required {
		if _, ok := header[key]; !ok {
			return nil, ErrMissingHeader
		}
	}

	records := make([]map[string]any, 0, len(table)-1)
	for i := 1; i < len(table); i++ {
		if isEmptyRow(table[i]) {
			continue
		}
		record := map[string]any{}
		for key, idx := range header {
			if idx >= len(table[i]) {
				record[key] = ""
				continue
			}
			record[key] = strings.TrimSpace(table[i][idx])
		}
		record["_row_index"] = i + 1
		records = append(records, record)
	}
	return parseRows(records, 2)
}

func (s *Service) Import(rows []ImportRow) (Result, error) {
	result := Result{Errors: make([]ErrorItem, 0)}
	classCache := map[uint]bool{}

	for _, row := range rows {
		if err := validateRow(row); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, ErrorItem{
				Row:       row.Index,
				StudentID: row.StudentID,
				Error:     err.Error(),
			})
			continue
		}

		if _, ok := classCache[row.ClassID]; !ok {
			if _, err := s.classRepo.GetByID(row.ClassID); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					classCache[row.ClassID] = false
				} else {
					return result, err
				}
			} else {
				classCache[row.ClassID] = true
			}
		}
		if !classCache[row.ClassID] {
			result.Failed++
			result.Errors = append(result.Errors, ErrorItem{
				Row:       row.Index,
				StudentID: row.StudentID,
				Error:     ErrClassNotFound.Error(),
			})
			continue
		}

		if _, err := s.userRepo.GetByStudentID(row.StudentID); err == nil {
			result.Failed++
			result.Errors = append(result.Errors, ErrorItem{
				Row:       row.Index,
				StudentID: row.StudentID,
				Error:     ErrDuplicateStudentID.Error(),
			})
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return result, err
		}

		user := &model.User{
			StudentID:      row.StudentID,
			Name:           row.Name,
			Role:           row.Role,
			ClassID:        row.ClassID,
			Major:          row.Major,
			College:        row.College,
			EnrollmentYear: row.EnrollmentYear,
		}
		if err := s.userRepo.Create(user); err != nil {
			if isUniqueStudentIDErr(err) {
				result.Failed++
				result.Errors = append(result.Errors, ErrorItem{
					Row:       row.Index,
					StudentID: row.StudentID,
					Error:     ErrDuplicateStudentID.Error(),
				})
				continue
			}
			return result, err
		}

		if err := s.gradeSvc.SyncUserGradeByClassID(user.ID, row.ClassID); err != nil {
			return result, err
		}
		result.Imported++
	}

	return result, nil
}

func parseRows(records []map[string]any, startIndex int) ([]ImportRow, error) {
	out := make([]ImportRow, 0, len(records))
	for i, record := range records {
		row := ImportRow{Role: model.RoleStudent, Index: startIndex + i}

		if raw, ok := record["_row_index"]; ok {
			if v, ok := raw.(int); ok {
				row.Index = v
			}
		}

		row.StudentID = strings.TrimSpace(stringValue(record["student_id"]))
		row.Name = strings.TrimSpace(stringValue(record["name"]))
		row.Major = strings.TrimSpace(stringValue(record["major"]))
		row.College = strings.TrimSpace(stringValue(record["college"]))

		classID, err := uintValue(record["class_id"])
		if err != nil {
			row.ClassID = 0
		} else {
			row.ClassID = classID
		}

		if roleRaw, ok := record["role"]; ok && strings.TrimSpace(stringValue(roleRaw)) != "" {
			role, err := intValue(roleRaw)
			if err != nil {
				return nil, ErrInvalidRole
			}
			row.Role = role
		}

		if enrollRaw, ok := record["enrollment_year"]; ok && strings.TrimSpace(stringValue(enrollRaw)) != "" {
			enrollmentYear, err := intValue(enrollRaw)
			if err != nil {
				return nil, ErrInvalidEnrollYear
			}
			row.EnrollmentYear = enrollmentYear
		}

		out = append(out, row)
	}
	return out, nil
}

func validateRow(row ImportRow) error {
	if row.StudentID == "" || row.Name == "" || row.ClassID == 0 {
		return ErrMissingRequired
	}
	switch row.Role {
	case model.RoleStudent, model.RoleCadre, model.RoleTeacher, model.RoleSuperAdmin:
	default:
		return ErrInvalidRole
	}
	return nil
}

func normalizeHeader(header []string) map[string]int {
	out := make(map[string]int, len(header))
	for i, col := range header {
		key := strings.ToLower(strings.TrimSpace(col))
		if key != "" {
			out[key] = i
		}
	}
	return out
}

func isEmptyRow(row []string) bool {
	for _, col := range row {
		if strings.TrimSpace(col) != "" {
			return false
		}
	}
	return true
}

func stringValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	default:
		return ""
	}
}

func uintValue(v any) (uint, error) {
	s := strings.TrimSpace(stringValue(v))
	if s == "" {
		return 0, ErrInvalidClassID
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(n), nil
}

func intValue(v any) (int, error) {
	s := strings.TrimSpace(stringValue(v))
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func isUniqueStudentIDErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "users.student_id") || strings.Contains(msg, "duplicate key")
}

type sharedStrings struct {
	SI []struct {
		T string `xml:"t"`
		R []struct {
			T string `xml:"t"`
		} `xml:"r"`
	} `xml:"si"`
}

func readSharedStrings(files []*zip.File) ([]string, error) {
	data, err := readZipFile(files, "xl/sharedStrings.xml")
	if err != nil || len(data) == 0 {
		return []string{}, nil
	}
	var sst sharedStrings
	if err := xml.Unmarshal(data, &sst); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(sst.SI))
	for _, item := range sst.SI {
		if strings.TrimSpace(item.T) != "" {
			out = append(out, item.T)
			continue
		}
		var b strings.Builder
		for _, r := range item.R {
			b.WriteString(r.T)
		}
		out = append(out, b.String())
	}
	return out, nil
}

func readFirstWorksheet(files []*zip.File) ([]byte, error) {
	var first string
	for _, f := range files {
		if strings.HasPrefix(f.Name, "xl/worksheets/") && strings.HasSuffix(f.Name, ".xml") {
			if first == "" || f.Name < first {
				first = f.Name
			}
		}
	}
	if first == "" {
		return nil, ErrInvalidXLSX
	}
	return readZipFile(files, first)
}

type worksheet struct {
	Rows []sheetRow `xml:"sheetData>row"`
}

type sheetRow struct {
	R     int         `xml:"r,attr"`
	Cells []sheetCell `xml:"c"`
}

type sheetCell struct {
	R  string `xml:"r,attr"`
	T  string `xml:"t,attr"`
	V  string `xml:"v"`
	IS *struct {
		T string `xml:"t"`
		R []struct {
			T string `xml:"t"`
		} `xml:"r"`
	} `xml:"is"`
}

func parseWorksheet(data []byte, shared []string) ([][]string, error) {
	var ws worksheet
	if err := xml.Unmarshal(data, &ws); err != nil {
		return nil, err
	}
	table := make([][]string, 0, len(ws.Rows))
	for _, row := range ws.Rows {
		maxCol := 0
		values := map[int]string{}
		for _, cell := range row.Cells {
			col := columnIndex(cell.R)
			if col > maxCol {
				maxCol = col
			}
			values[col] = cellText(cell, shared)
		}

		out := make([]string, maxCol+1)
		for idx, value := range values {
			if idx >= 0 && idx < len(out) {
				out[idx] = strings.TrimSpace(value)
			}
		}
		table = append(table, out)
	}
	return table, nil
}

func cellText(cell sheetCell, shared []string) string {
	switch cell.T {
	case "s":
		i, err := strconv.Atoi(strings.TrimSpace(cell.V))
		if err != nil || i < 0 || i >= len(shared) {
			return ""
		}
		return shared[i]
	case "inlineStr":
		if cell.IS == nil {
			return ""
		}
		if strings.TrimSpace(cell.IS.T) != "" {
			return cell.IS.T
		}
		var b strings.Builder
		for _, run := range cell.IS.R {
			b.WriteString(run.T)
		}
		return b.String()
	default:
		return cell.V
	}
}

func columnIndex(ref string) int {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return 0
	}
	col := strings.TrimRightFunc(ref, func(r rune) bool { return r >= '0' && r <= '9' })
	col = strings.ToUpper(col)
	if col == "" {
		return 0
	}
	idx := 0
	for _, r := range col {
		if r < 'A' || r > 'Z' {
			return 0
		}
		idx = idx*26 + int(r-'A'+1)
	}
	return idx - 1
}

func readZipFile(files []*zip.File, name string) ([]byte, error) {
	for _, f := range files {
		if filepath.Clean(f.Name) != filepath.Clean(name) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	return nil, nil
}
