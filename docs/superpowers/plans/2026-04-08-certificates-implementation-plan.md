# Certificates Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现审批流程中的双阶段 PDF 能力：审批前/审批中生成申请材料 PDF，审批通过后生成带编号、核验码和内部章的审批结果凭证 PDF。

**Architecture:** 证书模块作为 approvals 的配套服务。`certificate_templates` 保存服务端受控 Typst 模板元数据，`certificate_records` 保存每次生成结果；PDF 文件统一保存到文件服务。审批模块在申请提交后调用 `GenerateApplicationPDF`，在审批通过后调用 `GenerateApprovalCertificate`。

**Tech Stack:** Go 1.25, Gin, GORM, SQLite (testing), PostgreSQL/Kingbase (production), Typst CLI or Typst renderer wrapper

---

## File Structure

| 操作 | 文件路径 | 说明 |
|------|----------|------|
| 创建 | `internal/model/certificate_template.go` | PDF 模板模型 |
| 创建 | `internal/model/certificate_record.go` | PDF 生成记录 |
| 创建 | `internal/repo/certificate_template_repo.go` | 模板数据访问 |
| 创建 | `internal/repo/certificate_record_repo.go` | 记录数据访问 |
| 创建 | `internal/repo/certificate_record_repo_test.go` | repo 测试 |
| 创建 | `internal/service/certificates/numbering.go` | 编号、核验码与摘要生成 |
| 创建 | `internal/service/certificates/renderer.go` | Typst 渲染器封装 |
| 创建 | `internal/service/certificates/service.go` | PDF 生成、重试、作废逻辑 |
| 创建 | `internal/service/certificates/service_test.go` | service 测试 |
| 创建 | `internal/http/handler/certificate_handler.go` | 证书记录、核验和管理接口 |
| 创建 | `internal/http/handler/certificate_handler_test.go` | handler 测试 |
| 修改 | `internal/service/approvals/service.go` | 提交/通过后触发 PDF 生成 |
| 修改 | `internal/service/authz/actions.go` | 新增 certificates 动作 |
| 修改 | `internal/service/authz/authorize.go` | 新增权限 |
| 修改 | `internal/store/db.go` | 加入证书模型 |
| 修改 | `internal/http/router/router.go` | 注册证书路由 |
| 创建 | `templates/certificates/leave_application.typ` | 请假申请材料模板 |
| 创建 | `templates/certificates/leave_approval_certificate.typ` | 请假审批结果凭证模板 |
| 创建 | `templates/certificates/budget_application.typ` | 预算申请材料模板 |
| 创建 | `templates/certificates/budget_approval_certificate.typ` | 预算审批结果凭证模板 |
| 修改 | `docs/api/phase2-certificates-api.md` | 证书模块 API 文档 |

---

## Task 1: 模型、权限与迁移

**Files:**
- Create: `internal/model/certificate_template.go`
- Create: `internal/model/certificate_record.go`
- Modify: `internal/service/authz/actions.go`
- Modify: `internal/service/authz/authorize.go`
- Modify: `internal/store/db.go`

- [ ] **Step 1: 创建模板模型**

```go
// internal/model/certificate_template.go
package model

import (
	"time"

	"gorm.io/datatypes"
)

type CertificateTemplate struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Code            string         `gorm:"type:varchar(60);uniqueIndex;not null" json:"code"`
	Name            string         `gorm:"type:varchar(100);not null" json:"name"`
	ApprovalType    string         `gorm:"type:varchar(20);index;not null" json:"approval_type"`
	DocumentStage   string         `gorm:"type:varchar(30);index;not null" json:"document_stage"`
	Status          string         `gorm:"type:varchar(20);index;not null" json:"status"`
	Renderer        string         `gorm:"type:varchar(20);not null" json:"renderer"`
	TemplatePath    string         `gorm:"type:varchar(255);not null" json:"template_path"`
	TemplateVersion string         `gorm:"type:varchar(40);not null" json:"template_version"`
	FieldMapping    datatypes.JSON `gorm:"type:jsonb" json:"field_mapping"`
	Disclaimer      string         `gorm:"type:varchar(500)" json:"disclaimer"`
	CreatedBy       uint           `json:"created_by"`
	UpdatedBy       uint           `json:"updated_by"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}
```

- [ ] **Step 2: 创建记录模型**

```go
// internal/model/certificate_record.go
package model

import (
	"time"

	"gorm.io/datatypes"
)

type CertificateRecord struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	ApprovalID       uint           `gorm:"index;not null" json:"approval_id"`
	ApplicantID      uint           `gorm:"index;not null" json:"applicant_id"`
	TemplateID       uint           `gorm:"index;not null" json:"template_id"`
	DocumentStage    string         `gorm:"type:varchar(30);index;not null" json:"document_stage"`
	CertificateNo    string         `gorm:"type:varchar(80);uniqueIndex" json:"certificate_no"`
	VerificationCode string         `gorm:"type:varchar(80);uniqueIndex" json:"verification_code"`
	VerificationHash string         `gorm:"type:varchar(128)" json:"verification_hash"`
	RenderedPayload  datatypes.JSON `gorm:"type:jsonb" json:"rendered_payload"`
	DocumentID       uint           `gorm:"index" json:"document_id"`
	SealStatus       string         `gorm:"type:varchar(30);index;not null" json:"seal_status"`
	SealAppliedBy    uint           `json:"seal_applied_by"`
	SealAppliedAt    *time.Time     `json:"seal_applied_at"`
	Status           string         `gorm:"type:varchar(20);index;not null" json:"status"`
	ErrorMessage     string         `gorm:"type:varchar(500)" json:"error_message"`
	GeneratedAt      *time.Time     `json:"generated_at"`
	RevokedAt        *time.Time     `json:"revoked_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}
```

- [ ] **Step 3: 新增 authz 动作**

```go
ActionCertificatesMyList                  = "certificates:my:list"
ActionCertificatesGet                     = "certificates:get"
ActionCertificatesTemplateAdminList       = "certificates:template:admin:list"
ActionCertificatesTemplateToggle          = "certificates:template:toggle"
ActionCertificatesApplicationRegenerate   = "certificates:application:regenerate"
ActionCertificatesCertificateRegenerate   = "certificates:approval_certificate:regenerate"
ActionCertificatesRevoke                  = "certificates:revoke"
```

- [ ] **Step 4: 配置权限**

学生允许：
- `ActionCertificatesMyList`
- `ActionCertificatesGet`

教师/超管允许：
- 学生权限
- `ActionCertificatesTemplateAdminList`
- `ActionCertificatesTemplateToggle`
- `ActionCertificatesApplicationRegenerate`
- `ActionCertificatesCertificateRegenerate`
- `ActionCertificatesRevoke`

团干部允许：
- scope 内查看审批材料
- 不允许重试生成审批结果凭证，不允许作废凭证

- [ ] **Step 5: 加入 AutoMigrate**

```go
&model.CertificateTemplate{},
&model.CertificateRecord{},
```

- [ ] **Step 6: 运行测试**

Run: `go test ./internal/service/authz ./internal/model ./internal/store -count=1`
Expected: PASS

---

## Task 2: Repo、编号与 Typst 渲染

**Files:**
- Create: `internal/repo/certificate_template_repo.go`
- Create: `internal/repo/certificate_record_repo.go`
- Create: `internal/repo/certificate_record_repo_test.go`
- Create: `internal/service/certificates/numbering.go`
- Create: `internal/service/certificates/renderer.go`
- Create: `templates/certificates/*.typ`

- [ ] **Step 1: 写模板 repo**

```go
type CertificateTemplateRepo struct{ db *gorm.DB }

func (r *CertificateTemplateRepo) GetActiveByApprovalTypeAndStage(approvalType, stage string) (*model.CertificateTemplate, error) {
	var item model.CertificateTemplate
	if err := r.db.Where("approval_type = ? AND document_stage = ? AND status = ?", approvalType, stage, "active").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
```

- [ ] **Step 2: 写记录 repo**

```go
func (r *CertificateRecordRepo) LatestByApprovalAndStage(approvalID uint, stage string) (*model.CertificateRecord, error) {}
func (r *CertificateRecordRepo) ListByApplicant(applicantID uint, limit, offset int) ([]model.CertificateRecord, int64, error) {}
func (r *CertificateRecordRepo) GetByVerificationCode(code string) (*model.CertificateRecord, error) {}
```

- [ ] **Step 3: 写编号与核验码生成**

```go
func NewCertificateNo(approvalType string, now time.Time, seq uint) string {}
func NewVerificationCode() (string, error) {}
func VerificationHash(record *model.CertificateRecord, payload []byte) string {}
```

- [ ] **Step 4: 写 Typst 渲染器接口**

```go
type Renderer interface {
	Render(ctx context.Context, templatePath string, payload map[string]any) ([]byte, error)
}
```

实现要求：
- Typst 只读取服务端模板路径。
- 输入数据写入临时 JSON 文件或通过受控参数传入。
- 渲染失败返回明确错误，不吞错。
- 测试中使用 fake renderer，不依赖本机 Typst。

- [ ] **Step 5: 创建四个固定模板**

- `leave_application.typ`
- `leave_approval_certificate.typ`
- `budget_application.typ`
- `budget_approval_certificate.typ`

模板必须包含效力说明位置；申请材料模板不显示内部章，审批结果凭证模板显示内部章/水印。

- [ ] **Step 6: 运行测试**

Run: `go test ./internal/repo ./internal/service/certificates -count=1`
Expected: PASS

---

## Task 3: Service 与 Approvals 集成

**Files:**
- Create: `internal/service/certificates/service.go`
- Create: `internal/service/certificates/service_test.go`
- Modify: `internal/service/approvals/service.go`

- [ ] **Step 1: 定义服务接口**

```go
type ApprovalPDFService interface {
	GenerateApplicationPDF(ctx context.Context, approvalID uint) (*model.CertificateRecord, error)
	GenerateApprovalCertificate(ctx context.Context, approvalID uint) (*model.CertificateRecord, error)
}
```

- [ ] **Step 2: 实现申请材料 PDF 生成**

规则：
- 允许 `pending`、`approved`、`rejected`、`withdrawn`、`expired` 状态生成申请材料 PDF。
- `seal_status` 固定为 `none`。
- 不生成 `certificate_no` 和 `verification_code`。

- [ ] **Step 3: 实现审批结果凭证生成**

规则：
- 仅 `approved` 状态允许生成。
- 自动生成 `certificate_no`、`verification_code`、`verification_hash`。
- `seal_status` 固定为 `internal_seal_applied`。
- PDF 中必须包含效力说明。

- [ ] **Step 4: 接入审批提交与通过动作**

建议集成点：
- `CreateApproval` 成功创建审批后，异步或同步生成申请材料 PDF。
- `ApproveApproval` 成功写入审批通过结果后，生成审批结果凭证 PDF。

失败策略：
- 申请材料 PDF 失败时，审批申请仍可创建，但记录失败状态，允许重试。
- 审批结果凭证 PDF 失败时，不回滚审批通过结果，允许管理员重试。

- [ ] **Step 5: 写 service 测试**

覆盖：
- 提交后生成申请材料 PDF，不带章、不带核验码。
- 审批通过后生成审批结果凭证，带编号、核验码和内部章。
- 未通过审批不能生成审批结果凭证。
- renderer 失败时写入 failed 记录。

- [ ] **Step 6: 运行测试**

Run: `go test ./internal/service/certificates ./internal/service/approvals -count=1`
Expected: PASS

---

## Task 4: Handler、路由与 API 文档

**Files:**
- Create: `internal/http/handler/certificate_handler.go`
- Create: `internal/http/handler/certificate_handler_test.go`
- Modify: `internal/http/router/router.go`
- Modify: `docs/api/phase2-certificates-api.md`

- [ ] **Step 1: 写 handler**

```go
func (h *CertificateHandler) ListMine(c *gin.Context) {}
func (h *CertificateHandler) Get(c *gin.Context) {}
func (h *CertificateHandler) Verify(c *gin.Context) {}
func (h *CertificateHandler) ListAdminTemplates(c *gin.Context) {}
func (h *CertificateHandler) ToggleTemplate(c *gin.Context) {}
func (h *CertificateHandler) RegenerateApplicationPDF(c *gin.Context) {}
func (h *CertificateHandler) RegenerateApprovalCertificate(c *gin.Context) {}
func (h *CertificateHandler) Revoke(c *gin.Context) {}
```

- [ ] **Step 2: 注册路由**

```go
api.GET("/certificates/verify", certificateHandler.Verify) // public route, register before api.Use(JWTAuth(...))

api.GET("/certificates/me", certificateHandler.ListMine)
api.GET("/certificates/:id", certificateHandler.Get)

admin.GET("/certificates/templates", certificateHandler.ListAdminTemplates)
admin.POST("/certificates/templates/:id/activate", certificateHandler.ToggleTemplate)
admin.POST("/certificates/templates/:id/deactivate", certificateHandler.ToggleTemplate)
admin.POST("/approvals/:id/application-pdf/regenerate", certificateHandler.RegenerateApplicationPDF)
admin.POST("/approvals/:id/certificate/regenerate", certificateHandler.RegenerateApprovalCertificate)
admin.POST("/certificates/:id/revoke", certificateHandler.Revoke)
```

注意：
- `GET /certificates/verify` 必须注册在 `api.Use(middleware.JWTAuth(...))` 之前，保持公开访问。
- `GET /certificates/verify` 必须注册在 `/certificates/:id` 前，避免路由冲突。

- [ ] **Step 3: 审批详情响应补充 PDF 列表**

`GET /api/v1/approvals/:id` 与 `GET /api/v1/admin/approvals/:id` 增加：

```json
{
  "certificate_records": [
    {
      "id": 1,
      "document_stage": "application",
      "document_id": 88,
      "download_url": "/api/v1/files/88/download",
      "status": "generated"
    }
  ]
}
```

约定：
- 详情接口统一返回 `certificate_records` 字段。
- 尚无可用记录时返回空数组。

- [ ] **Step 4: 写 handler 测试**

覆盖：
- 学生只能查看本人证书记录。
- 教师可重试生成 scope 内审批 PDF。
- 团干部不能生成审批结果凭证。
- 核验接口不返回敏感字段。
- 核验接口无需登录即可访问。

- [ ] **Step 5: 运行测试**

Run: `go test ./internal/http/handler ./internal/http/router -count=1`
Expected: PASS

---

## Task 5: 全量验证

**Files:**
- Modify: `internal/service/certificates/service_test.go`
- Modify: `internal/http/handler/certificate_handler_test.go`
- Modify: `docs/api/phase2-certificates-api.md`

- [ ] **Step 1: 补失败和边界测试**

- renderer 失败，不回滚审批。
- revoked 凭证核验时显示已作废。
- inactive 模板不能生成新 PDF。
- `expired` 审批仍可重建申请材料 PDF。
- 跨班/跨年级 scope 访问被拒绝。

- [ ] **Step 2: 全量运行**

Run: `go test ./... -count=1`
Expected: PASS

---

## Notes

- v1 不接入学校正式电子签章和 CA 体系。
- 内部章文案建议固定为“学院内部审批章”或“系统生成章”。
- PDF 上必须展示效力说明，避免被误认为学校正式公章文件。
- 如果后续学校提供正式模板或签章能力，只替换模板和签章适配层，不改变审批与记录主流程。
