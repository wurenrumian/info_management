# 电子证明与审批 PDF 模块设计（v1）

**日期**：2026-04-08
**更新日期**：2026-04-27
**阶段**：Phase 2-E

## 1. 目标

实现审批流程中的 PDF 生成能力，而不是独立的“任意证明生成器”：
- 学生提交请假/预算申请时，可生成申请材料 PDF，供本人预览和审批人查看。
- 审批通过后，系统生成审批结果凭证 PDF，作为学院内部流转和留痕材料。
- PDF 由服务端基于固定 Typst 模板和结构化数据生成，保证格式稳定、内容可追溯。
- 审批结果凭证带编号、核验码、内部审批章/系统生成章和效力说明。

## 2. 范围

### In Scope

- 固定审批 PDF 模板：`leave`、`budget`
- 两阶段 PDF：申请材料 PDF、审批结果凭证 PDF
- Typst 服务端模板渲染
- 生成结果存入统一文件服务，并写入生成记录
- 审批结果凭证编号、核验码、校验接口
- 内部审批章/系统生成章水印
- PDF 生成失败留痕，允许重试

### Out of Scope

- 学校正式电子签章/CA 签名
- 学校正式公章文件
- 开放式模板设计器
- 用户上传或编辑 Typst 源码
- 任意证明类型自助生成
- 奖学金、休学复学、宿舍调整等正式学校流程证明
- 大模型生成证明正文

说明：
- v1 的电子证明只服务已纳入审批流的 `leave` 和 `budget`。
- 奖学金助学金细则、休学复学细则、宿舍调整细则、校历节假日、请假条模板、预算模板进入文档库/知识库，不进入电子证明生成范围。
- 电子章在 v1 中定位为“内部审批章/系统生成章”，不等同于学校正式公章。

## 3. 核心取舍

### 3.1 为什么拆成两阶段 PDF

审批前的 PDF 是申请材料，代表“学生提交了什么”；审批后的 PDF 是审批结果凭证，代表“学院内部审批结果是什么”。两者不能混用。

如果审批前就生成带章证明，学生可能拿到看似已经批准的文件，带来误用风险。因此：
- 审批前允许生成申请材料 PDF，但不盖章，不写“已审批通过”。
- 审批通过后才生成带编号、核验码和内部章的审批结果凭证。

### 3.2 为什么不用任意模板生成器

当前可确定的业务入口只有请假和预算审批。任意模板生成器会引入模板权限、字段安全、格式审核、证明效力边界等复杂问题，v1 暂不做。

## 4. PDF 类型

| 类型 | 模板编码 | 审批类型 | 生成时机 | 是否盖章 | 用途 |
|------|----------|----------|----------|----------|------|
| 申请材料 PDF | `leave_application_pdf` | `leave` | 提交后或审批中 | 否 | 学生确认、审批人查看 |
| 审批结果凭证 | `leave_approval_certificate` | `leave` | 审批通过后 | 是，内部章 | 请假审批通过留痕 |
| 申请材料 PDF | `budget_application_pdf` | `budget` | 提交后或审批中 | 否 | 学生确认、审批人查看 |
| 审批结果凭证 | `budget_approval_certificate` | `budget` | 审批通过后 | 是，内部章 | 预算审批通过留痕 |

## 5. 数据模型

### 5.1 `certificate_templates`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 主键 |
| code | varchar(60) unique | 模板编码 |
| name | varchar(100) | 模板名称 |
| approval_type | varchar(20) | `leave` / `budget` |
| document_stage | varchar(30) | `application` / `approval_certificate` |
| status | varchar(20) | `active` / `inactive` |
| renderer | varchar(20) | 固定为 `typst` |
| template_path | varchar(255) | 服务端 Typst 模板路径或模板 key |
| template_version | varchar(40) | 模板版本 |
| field_mapping | jsonb | 字段映射与格式化规则 |
| disclaimer | varchar(500) | PDF 效力说明 |
| created_by | bigint FK | 创建人 |
| updated_by | bigint FK | 更新人 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

说明：
- `template_path` 指向服务端受控模板，不接受用户提交 Typst 源码。
- v1 可由后端 seed 固定模板，管理端只做启停和查看，不做在线编辑。

### 5.2 `certificate_records`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 主键 |
| approval_id | bigint FK | 关联审批申请 |
| applicant_id | bigint FK | 申请人 |
| template_id | bigint FK | 使用模板 |
| document_stage | varchar(30) | `application` / `approval_certificate` |
| certificate_no | varchar(80) nullable | 审批结果凭证编号，申请材料可为空 |
| verification_code | varchar(80) nullable | 对外核验码，申请材料可为空 |
| verification_hash | varchar(128) nullable | 防篡改摘要 |
| rendered_payload | jsonb | 实际填充值快照 |
| document_id | bigint FK nullable | 生成的 PDF 文件 ID |
| seal_status | varchar(30) | `none` / `internal_seal_applied` |
| seal_applied_by | bigint FK nullable | 盖内部章操作者，系统任务可为 0 |
| seal_applied_at | timestamp nullable | 盖章时间 |
| status | varchar(20) | `generated` / `failed` / `revoked` |
| error_message | varchar(500) nullable | 失败原因 |
| generated_at | timestamp nullable | 生成时间 |
| revoked_at | timestamp nullable | 作废时间 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

索引建议：
- `(approval_id, document_stage)`：查询某申请的 PDF
- `certificate_no` unique where not null
- `verification_code` unique where not null
- `(applicant_id, created_at)`：学生查看本人记录

## 6. 模板与渲染

v1 使用 Typst，但模板来源必须受控：
- Typst 源文件随代码或部署包发布，例如 `templates/certificates/leave_application.typ`。
- 服务端只把结构化数据传入模板，不执行用户提交的模板代码。
- 模板版本写入 `certificate_records.rendered_payload`，便于之后追溯。

`field_mapping` 示例：

```json
{
  "student_name": "applicant.name",
  "student_id": "applicant.student_id",
  "approval_title": "approval.title",
  "leave_start_at": "approval.form_data.start_at",
  "leave_end_at": "approval.form_data.end_at",
  "approver_name": "approval.current_approver.name",
  "decided_at": "approval.decided_at"
}
```

审批结果凭证 PDF 必须包含：
- 申请人姓名、学号、班级/年级
- 申请类型和关键表单字段
- 审批结果、审批人、审批通过时间
- `certificate_no`
- `verification_code`
- 内部审批章/系统生成章水印
- 效力说明

建议效力说明：

```text
本文件由学院信息管理系统自动生成，仅用于学院内部审批留痕与流转，不等同于学校正式公章文件。
```

## 7. 生成链路

### 7.1 申请材料 PDF

1. 学生提交 `leave` 或 `budget` 申请。
2. 审批模块创建 `approvals` 记录。
3. 证书服务根据 `approval_id` 和 `document_stage=application` 收集申请人与表单数据。
4. Typst 渲染申请材料 PDF。
5. PDF 保存到统一文件服务，得到 `document_id`。
6. 写入 `certificate_records(status=generated, seal_status=none)`。
7. 审批详情返回申请材料 PDF 下载信息。

说明：
- 申请材料 PDF 可自动生成，也可由管理员手动触发重建。
- 申请撤回、驳回或标记过期后，申请材料 PDF 仍作为历史材料保留。

### 7.2 审批结果凭证 PDF

1. 教师/超管审批通过申请。
2. 审批模块写入 `approval_actions(action_type=approve)`。
3. 证书服务根据 `approval_id` 和 `document_stage=approval_certificate` 生成编号和核验码。
4. Typst 渲染审批结果凭证 PDF，并加入内部章/水印和效力说明。
5. PDF 保存到统一文件服务，得到 `document_id`。
6. 写入 `certificate_records(status=generated, seal_status=internal_seal_applied)`。
7. 审批详情返回审批结果凭证下载信息。

失败处理：
- PDF 生成失败不回滚已完成的审批结论。
- 写入 `certificate_records(status=failed, error_message=...)`。
- 管理员可在审批详情中手动重试生成。

## 8. API 设计

### 8.1 学生端与公开核验接口

- `GET /api/v1/certificates/me?approval_type=leave&limit=20&offset=0`
- `GET /api/v1/certificates/:id`
- `GET /api/v1/certificates/verify?code=xxx`

学生主要通过审批详情获取 PDF：
- `GET /api/v1/approvals/:id` 与 `GET /api/v1/admin/approvals/:id` 返回统一详情结构，并包含 `certificate_records`。
- 当尚无可用 PDF 记录时，`certificate_records` 返回空数组。
- `GET /api/v1/certificates/verify?code=xxx` 为公开核验接口，不要求登录。

### 8.2 管理端

- `GET /api/v1/admin/certificates/templates`
- `POST /api/v1/admin/certificates/templates/:id/activate`
- `POST /api/v1/admin/certificates/templates/:id/deactivate`
- `POST /api/v1/admin/approvals/:id/application-pdf/regenerate`
- `POST /api/v1/admin/approvals/:id/certificate/regenerate`
- `POST /api/v1/admin/certificates/:id/revoke`

### 8.3 内部服务接口

审批模块调用证书服务，不一定暴露为 HTTP：

```go
type ApprovalPDFService interface {
	GenerateApplicationPDF(ctx context.Context, approvalID uint) (*CertificateRecord, error)
	GenerateApprovalCertificate(ctx context.Context, approvalID uint) (*CertificateRecord, error)
}
```

## 9. 权限与 Scope

- 学生：查看本人审批相关 PDF。
- 团干部：可在授权 scope 内查看审批材料和提醒，但不能生成带内部章的审批结果凭证。
- 教师/超管：可在授权 scope 内查看、重试生成、作废审批结果凭证。
- 公开核验接口不要求登录，但只返回最小必要信息，不暴露身份证号、联系方式、详细申请原因等敏感字段。

新增 authz 动作建议：

```go
ActionCertificatesMyList              = "certificates:my:list"
ActionCertificatesGet                 = "certificates:get"
ActionCertificatesTemplateAdminList   = "certificates:template:admin:list"
ActionCertificatesTemplateToggle      = "certificates:template:toggle"
ActionCertificatesApplicationRegenerate = "certificates:application:regenerate"
ActionCertificatesCertificateRegenerate = "certificates:approval_certificate:regenerate"
ActionCertificatesRevoke              = "certificates:revoke"
```

## 10. 代码结构

```text
internal/
├── model/
│   ├── certificate_template.go
│   └── certificate_record.go
├── repo/
│   ├── certificate_template_repo.go
│   ├── certificate_record_repo.go
│   └── certificate_record_repo_test.go
├── service/
│   └── certificates/
│       ├── numbering.go
│       ├── renderer.go
│       ├── service.go
│       └── service_test.go
└── http/
    └── handler/
        ├── certificate_handler.go
        └── certificate_handler_test.go
```

## 11. 测试策略

- handler 测试：
  - 学生只能查看本人 PDF 记录
  - 非本人访问返回 403 或 404
  - 管理员可重试生成失败 PDF
  - 团干部不能重试生成审批结果凭证
- service 测试：
  - 提交后生成申请材料 PDF，不带章和核验码
  - 审批通过后生成审批结果凭证，带编号、核验码和内部章
  - 审批未通过时不能生成审批结果凭证
  - PDF 生成失败时记录失败状态，不回滚审批结果
  - 核验接口不返回敏感字段
- repo 测试：
  - 按 `approval_id + document_stage` 查询正确
  - `certificate_no` / `verification_code` 唯一约束有效

## 12. 验收标准

- 请假申请通过审批详情可查看申请材料 PDF。
- 预算申请通过审批详情可查看申请材料 PDF。
- 请假审批通过后自动或手动生成审批结果凭证 PDF。
- 预算审批通过后自动或手动生成审批结果凭证 PDF。
- 审批结果凭证包含编号、核验码、内部章和效力说明。
- PDF 统一通过文件服务下载，不单独建设文件存储。
- 不依赖学校正式电子签章即可完成学院内部留痕闭环。
