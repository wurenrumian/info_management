# Certificates Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现标准证明模板管理、学生一键生成 PDF 证明、结果入文档库并可下载的最小闭环。

**Architecture:** 使用 `certificate_templates` 管理模板结构与字段映射，`certificate_records` 管理生成记录；服务端渲染 PDF 后保存为 `Document`，下载统一复用现有文件接口。

**Tech Stack:** Go 1.25, Gin, GORM, SQLite (testing), PostgreSQL/Kingbase (production)

---

## File Structure

| 操作 | 文件路径 | 说明 |
|------|----------|------|
| 创建 | `internal/model/certificate_template.go` | 证明模板模型 |
| 创建 | `internal/model/certificate_record.go` | 证明生成记录 |
| 创建 | `internal/repo/certificate_template_repo.go` | 模板数据访问 |
| 创建 | `internal/repo/certificate_record_repo.go` | 记录数据访问 |
| 创建 | `internal/repo/certificate_template_repo_test.go` | repo 测试 |
| 创建 | `internal/service/certificates/renderer.go` | PDF 渲染器 |
| 创建 | `internal/service/certificates/service.go` | 模板管理与生成逻辑 |
| 创建 | `internal/service/certificates/service_test.go` | service 测试 |
| 创建 | `internal/http/handler/certificate_handler.go` | 证明确认接口 |
| 创建 | `internal/http/handler/certificate_handler_test.go` | handler 测试 |
| 修改 | `internal/service/authz/actions.go` | 新增 certificates 动作 |
| 修改 | `internal/service/authz/authorize.go` | 新增权限 |
| 修改 | `internal/store/db.go` | 加入证书模型 |
| 修改 | `internal/http/router/router.go` | 注册证书路由 |
| 创建 | `docs/api/phase2-certificates-api.md` | 证书模块 API 文档 |

### Task 1: 模型、权限与迁移

**Files:**
- Create: `internal/model/certificate_template.go`
- Create: `internal/model/certificate_record.go`
- Modify: `internal/service/authz/actions.go`
- Modify: `internal/service/authz/authorize.go`
- Modify: `internal/store/db.go`

- [ ] **Step 1: 创建模型**

```go
// internal/model/certificate_template.go
package model

import (
	"time"

	"gorm.io/datatypes"
)

type CertificateTemplate struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Code           string         `gorm:"type:varchar(40);uniqueIndex;not null" json:"code"`
	Name           string         `gorm:"type:varchar(100);not null" json:"name"`
	Status         string         `gorm:"type:varchar(20);index;not null" json:"status"`
	TemplateSchema datatypes.JSON `gorm:"type:jsonb" json:"template_schema"`
	FieldMapping   datatypes.JSON `gorm:"type:jsonb" json:"field_mapping"`
	CreatedBy      uint           `json:"created_by"`
	UpdatedBy      uint           `json:"updated_by"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}
```

```go
// internal/model/certificate_record.go
package model

import (
	"time"

	"gorm.io/datatypes"
)

type CertificateRecord struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	UserID          uint           `gorm:"index;not null" json:"user_id"`
	TemplateID      uint           `gorm:"index;not null" json:"template_id"`
	RenderedPayload datatypes.JSON `gorm:"type:jsonb" json:"rendered_payload"`
	DocumentID      uint           `gorm:"index" json:"document_id"`
	Status          string         `gorm:"type:varchar(20);index;not null" json:"status"`
	CreatedAt       time.Time      `json:"created_at"`
}
```

- [ ] **Step 2: 新增 authz 动作**

```go
	ActionCertificatesTemplateList   = "certificates:template:list"
	ActionCertificatesGenerate       = "certificates:generate"
	ActionCertificatesMyList         = "certificates:my:list"
	ActionCertificatesAdminList      = "certificates:admin:list"
	ActionCertificatesTemplateCreate = "certificates:template:create"
	ActionCertificatesTemplatePatch  = "certificates:template:patch"
	ActionCertificatesTemplateToggle = "certificates:template:toggle"
```

- [ ] **Step 3: 权限映射**

```go
case model.RoleStudent:
	return action == ActionGetMe ||
		action == ActionMePatch ||
		action == ActionProfileHomeGet ||
		action == ActionKnowledgeSearch ||
		action == ActionNotifUnreadGet ||
		action == ActionFilesUpload ||
		action == ActionFilesGet ||
		action == ActionFilesList ||
		action == ActionCertificatesTemplateList ||
		action == ActionCertificatesGenerate ||
		action == ActionCertificatesMyList
```

- [ ] **Step 4: 迁移新表**

```go
&model.CertificateTemplate{},
&model.CertificateRecord{},
```

- [ ] **Step 5: 运行测试**

Run: `go test ./internal/service/authz ./internal/model ./internal/store -count=1`  
Expected: PASS

### Task 2: Repo 与渲染服务

**Files:**
- Create: `internal/repo/certificate_template_repo.go`
- Create: `internal/repo/certificate_record_repo.go`
- Create: `internal/repo/certificate_template_repo_test.go`
- Create: `internal/service/certificates/renderer.go`
- Create: `internal/service/certificates/service.go`
- Create: `internal/service/certificates/service_test.go`

- [ ] **Step 1: 写模板 repo**

```go
type CertificateTemplateRepo struct{ db *gorm.DB }

func (r *CertificateTemplateRepo) GetActiveByCode(code string) (*model.CertificateTemplate, error) {
	var item model.CertificateTemplate
	if err := r.db.Where("code = ? AND status = ?", code, "active").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
```

- [ ] **Step 2: 写渲染器接口**

```go
type Renderer interface {
	Render(templateSchema []byte, payload map[string]string) ([]byte, error)
}
```

- [ ] **Step 3: 写生成测试**

```go
func TestGenerateCreatesDocumentAndRecord(t *testing.T) {
	// seed one active template
	// call service.Generate(userID, "student_status")
	// expect one Document and one CertificateRecord
}
```

- [ ] **Step 4: 运行测试**

Run: `go test ./internal/repo ./internal/service/certificates -count=1`  
Expected: PASS

### Task 3: Handler、路由与 API 文档

**Files:**
- Create: `internal/http/handler/certificate_handler.go`
- Create: `internal/http/handler/certificate_handler_test.go`
- Modify: `internal/http/router/router.go`
- Create: `docs/api/phase2-certificates-api.md`

- [ ] **Step 1: 写 handler**

```go
type CertificateHandler struct {
	svc *certificates.Service
}

func NewCertificateHandler(db *gorm.DB) *CertificateHandler {
	return &CertificateHandler{
		svc: certificates.NewService(
			repo.NewCertificateTemplateRepo(db),
			repo.NewCertificateRecordRepo(db),
			repo.NewDocumentRepo(db),
			repo.NewUserRepo(db),
			repo.NewClassRepo(db),
		),
	}
}
```

```go
func (h *CertificateHandler) ListTemplates(c *gin.Context) {}
func (h *CertificateHandler) Generate(c *gin.Context) {}
func (h *CertificateHandler) ListMine(c *gin.Context) {}
func (h *CertificateHandler) ListAdminTemplates(c *gin.Context) {}
func (h *CertificateHandler) CreateTemplate(c *gin.Context) {}
func (h *CertificateHandler) PatchTemplate(c *gin.Context) {}
func (h *CertificateHandler) ToggleTemplate(c *gin.Context) {}
```

- [ ] **Step 2: 注册路由**

```go
certificateHandler := handler.NewCertificateHandler(db)

api.GET("/certificates/templates", certificateHandler.ListTemplates)
api.POST("/certificates/generate", certificateHandler.Generate)
api.GET("/certificates/me", certificateHandler.ListMine)

admin.GET("/certificates/templates", certificateHandler.ListAdminTemplates)
admin.POST("/certificates/templates", certificateHandler.CreateTemplate)
admin.PATCH("/certificates/templates/:id", certificateHandler.PatchTemplate)
admin.POST("/certificates/templates/:id/activate", certificateHandler.ToggleTemplate)
admin.POST("/certificates/templates/:id/deactivate", certificateHandler.ToggleTemplate)
```

- [ ] **Step 3: 写 API 文档**

```md
# Phase2 Certificates API

## Student Endpoints

- `GET /api/v1/certificates/templates`
- `POST /api/v1/certificates/generate`
- `GET /api/v1/certificates/me`

## Admin Endpoints

- `GET /api/v1/admin/certificates/templates`
- `POST /api/v1/admin/certificates/templates`
- `PATCH /api/v1/admin/certificates/templates/:id`
- `POST /api/v1/admin/certificates/templates/:id/activate`
- `POST /api/v1/admin/certificates/templates/:id/deactivate`
```

- [ ] **Step 4: 运行测试**

Run: `go test ./internal/http/handler ./internal/http/router -count=1`  
Expected: PASS

### Task 4: 全量验证

**Files:**
- Modify: `internal/service/certificates/service_test.go`
- Modify: `docs/api/phase2-certificates-api.md`

- [ ] **Step 1: 补失败链路测试**

```go
func TestGenerateFailsWhenTemplateInactive(t *testing.T) {
	// inactive template
	// expect not found or forbidden style error
}
```

- [ ] **Step 2: 全量运行**

Run: `go test ./... -count=1`  
Expected: PASS

