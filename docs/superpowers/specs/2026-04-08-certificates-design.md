# 电子证明模块设计（v0）

**日期**：2026-04-08  
**阶段**：Phase 2-E

## 1. 目标

实现电子证明生成的最小闭环：
- 学生选择标准证明模板
- 系统使用结构化信息填充模板
- 生成 PDF 并提供下载
- 管理端维护少量标准模板与字段映射

## 2. 范围

### In Scope

- 固定模板管理
- 学生生成本人证明
- 生成结果存档并可下载
- 生成记录与模板变更留痕

### Out of Scope

- 开放式模板设计器
- 任意 HTML 编辑
- 复杂审批前置
- 大模型生成正文

## 3. 数据模型

### 3.1 `certificate_templates`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 主键 |
| code | varchar(40) unique | 模板编码 |
| name | varchar(100) | 模板名称 |
| status | varchar(20) | `active` / `inactive` |
| template_schema | jsonb | 模板结构 |
| field_mapping | jsonb | 字段映射 |
| created_by | bigint FK | 创建人 |
| updated_by | bigint FK | 更新人 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

### 3.2 `certificate_records`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 主键 |
| user_id | bigint FK | 生成人 |
| template_id | bigint FK | 使用模板 |
| rendered_payload | jsonb | 实际填充值快照 |
| document_id | bigint FK | 对应生成的 PDF 文件 |
| status | varchar(20) | `generated` / `failed` |
| created_at | timestamp | 生成时间 |

## 4. 模板设计

v0 使用 JSON 模板结构，不直接存储可执行脚本。

`template_schema` 示例：

```json
{
  "title": "在读证明",
  "body_lines": [
    "兹证明 {student_name}，学号 {student_id}，系 {grade} 级 {major} 学生。",
    "特此证明。"
  ],
  "footer": "信息学院"
}
```

`field_mapping` 示例：

```json
{
  "student_name": "user.name",
  "student_id": "user.student_id",
  "grade": "user.grade",
  "major": "class.major"
}
```

## 5. 生成链路

1. 学生选择模板编码
2. 服务端按模板字段映射收集本人数据
3. 渲染为 PDF
4. 将 PDF 保存为文档库文件，得到 `document_id`
5. 写入 `certificate_records`
6. 返回下载信息

说明：
- v0 直接复用 Phase 2-0 文档库，不单独建设文件存储
- PDF 生成位置固定在服务端，保证格式稳定

## 6. API 设计

### 6.1 学生端

- `GET /api/v1/certificates/templates`
- `POST /api/v1/certificates/generate`
- `GET /api/v1/certificates/me?limit=20&offset=0`

生成请求体：

```json
{
  "template_code": "student_status"
}
```

成功响应：

```json
{
  "data": {
    "record_id": 1,
    "template_code": "student_status",
    "document_id": 88,
    "download_url": "/api/v1/files/88/download",
    "created_at": "2026-04-08T12:00:00Z"
  }
}
```

### 6.2 管理端

- `GET /api/v1/admin/certificates/templates`
- `POST /api/v1/admin/certificates/templates`
- `PATCH /api/v1/admin/certificates/templates/:id`
- `POST /api/v1/admin/certificates/templates/:id/activate`
- `POST /api/v1/admin/certificates/templates/:id/deactivate`

## 7. 权限与 Scope

- 学生：查看模板列表、生成本人证明、查看本人生成记录
- 管理员：维护模板
- 模板写操作记录 `admin_logs`

新增 authz 动作建议：

```go
ActionCertificatesTemplateList   = "certificates:template:list"
ActionCertificatesGenerate       = "certificates:generate"
ActionCertificatesMyList         = "certificates:my:list"
ActionCertificatesAdminList      = "certificates:admin:list"
ActionCertificatesTemplateCreate = "certificates:template:create"
ActionCertificatesTemplatePatch  = "certificates:template:patch"
ActionCertificatesTemplateToggle = "certificates:template:toggle"
```

## 8. 代码结构

```text
internal/
├── model/
│   ├── certificate_template.go
│   └── certificate_record.go
├── repo/
│   ├── certificate_template_repo.go
│   ├── certificate_template_repo_test.go
│   ├── certificate_record_repo.go
│   └── certificate_record_repo_test.go
├── service/
│   └── certificates/
│       ├── renderer.go
│       ├── service.go
│       └── service_test.go
└── http/
    └── handler/
        ├── certificate_handler.go
        └── certificate_handler_test.go
```

## 9. 测试策略

- handler 测试：
  - 学生生成成功
  - 模板不存在返回 404
  - 停用模板不可生成
  - 管理员模板管理成功
- repo / service 测试：
  - 字段映射解析正确
  - PDF 生成失败时记录失败状态
  - 生成成功时会创建 `Document` 与 `CertificateRecord`

## 10. 验收标准

- 至少 1 个标准证明模板可生成 PDF
- 学生可通过现有文件下载链路获取证明
- 模板可启停并保留生成记录
- 不依赖审批流即可完成最小闭环

