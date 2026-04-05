# Knowledge 文档自动问答（AI 生成 + 超管审核）设计

## 1. 背景与目标

当前知识库问答需人工逐条录入，效率低且易遗漏。目标是让系统基于已上传文档正文（`documents.content_text`）自动生成问答草稿，由超级管理员审核后再批量入库。

本期目标：先打通最小闭环，优先稳定上线。

## 2. 范围

### In Scope

- 新增 AI 预览生成接口（同步）
- 新增知识库批量创建接口（事务全成功）
- 仅 `role = 4`（super admin）可调用
- 前端人工审核与编辑后再提交
- 审计日志记录 AI 生成与批量入库行为

### Out of Scope

- 自动触发（上传后后台自动跑）
- 复杂去重与相似度聚类
- 异步任务队列与重试中心

## 3. 角色与流程

1. 普通用户上传文档（沿用 `/api/v1/files/upload`）。
2. 超管前端选择文档，调用 AI 预览生成接口。
3. 后端读取 `documents.content_text`，注入 AI，返回问答草稿。
4. 超管在前端审核、编辑、删减草稿。
5. 超管调用批量提交接口，后端事务写入 `knowledge_items`（全成功才提交）。

## 4. API 设计

## 4.1 `POST /api/v1/admin/knowledge/qa-generate-preview`

权限：`role = 4`

请求体：

```json
{
  "file_ids": [12, 18],
  "qa_count_range": { "min": 5, "max": 12 }
}
```

约束：

- `file_ids` 非空，且文件必须存在
- `1 <= min <= max <= 30`
- 预览返回条数需落在 `[min, max]`；若模型输出不足 `min`，返回 `500 generate preview failed`

响应（草稿，不含知识库真实 `id`）：

```json
{
  "data": [
    {
      "question": "奖学金申请需要什么材料？",
      "answer": "需要成绩单、综测证明、申请表。",
      "keywords": ["奖学金", "申请材料"],
      "attachment_file_ids": [12]
    }
  ],
  "total": 9
}
```

说明：

- 预览阶段尚未入库，不返回 `id/created_at/updated_at/created_by/updated_by`。
- 批量提交成功后，后端返回完整知识项结构（与现有接口一致）。

错误：

- `400 invalid body`
- `400 missing file_ids`
- `400 invalid qa_count_range`
- `400 file not found`
- `401 unauthorized`
- `403 forbidden`
- `500 generate preview failed`

## 4.2 `POST /api/v1/admin/knowledge/batch`

权限：`role = 4`

请求体：

```json
{
  "items": [
    {
      "question": "奖学金申请需要什么材料？",
      "answer": "需要成绩单、综测证明、申请表。",
      "keywords": ["奖学金", "申请材料"],
      "attachment_file_ids": [12]
    }
  ]
}
```

事务语义：

- 任一 item 失败则整体回滚
- 全部成功才返回成功

成功响应（保持与现有知识详情结构一致）：

```json
{
  "data": [
    {
      "id": 101,
      "question": "奖学金申请需要什么材料？",
      "answer": "需要成绩单、综测证明、申请表。",
      "keywords": ["奖学金", "申请材料"],
      "attachments": [
        {
          "file_id": 12,
          "title": "scholarship.docx",
          "url": "/uploads/knowledge/2026/04/123_scholarship.docx",
          "content_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
          "file_size": 20480
        }
      ],
      "created_by": 1,
      "updated_by": 1,
      "created_at": "2026-04-05T00:00:00Z",
      "updated_at": "2026-04-05T00:00:00Z"
    }
  ],
  "total": 1
}
```

错误：

- `400 invalid body`
- `400 empty items`
- `400 invalid item`
- `400 file not found`
- `401 unauthorized`
- `403 forbidden`
- `500 batch create knowledge failed`

## 5. AI 接入策略

- 输入来源：后端读取 `documents.content_text`，并附带 `file_id/title/file_path/url` 元信息。
- 输出形式：要求 AI 严格输出 JSON，结构固定为 `items[]`，字段仅包含：
  - `question`
  - `answer`
  - `keywords`
  - `attachment_file_ids`
- 后端校验：
  - JSON 反序列化成功
  - 字段完整且非空
  - `attachment_file_ids` 必须属于请求 `file_ids` 且文件存在
- 失败处理：格式错误或校验失败直接报错，不进入批量入库。

## 6. 数据与实现要点

- 复用现有 `knowledge_items` 与 `knowledge_attachments`。
- `batch` 接口内部循环调用现有创建/绑定逻辑，但放在单事务内执行。
- 生成预览不写库，仅返回草稿。

## 7. 审计日志

新增动作：

- `knowledge.ai_generate_preview`
- `knowledge.batch_create`

## 8. 测试范围

- Handler：权限、参数校验、错误码
- Service：批量事务回滚验证
- AI 解析：结构化输出成功/失败路径
- 回归：不影响现有 `admin/knowledge` CRUD 与附件绑定接口

## 9. 验收标准

- 超管可基于文档一键生成问答草稿
- 前端可编辑后一次性提交
- 批量提交全成功才落库
- 响应结构与现有知识库返回风格保持一致
