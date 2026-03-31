# Phase 2-B 知识库问答设计（v0）

**日期**：2026-03-31  
**阶段**：Phase 2-B Knowledge Base

## 1. 目标

在不引入大模型与外部搜索引擎的前提下，实现“结构化 FAQ + 数据库分词检索”的知识库问答能力，提供学生可用检索与管理员维护闭环。

## 2. 范围

包含：
- 学生端关键词检索 FAQ，返回标准答案与附件链接
- 管理端 FAQ 列表/新增/修改
- 维护操作写入 `admin_logs`
- 复用 Phase 1 的 `Authorize + BuildScope`

不包含：
- 通用大模型问答
- 外部信息抓取
- ES 集群引入

## 3. 权限

- 学生（role=1）：可搜索 `GET /api/v1/knowledge/search`
- 团干部/教师/超管（role>=2）：可维护知识库（列表/新增/修改）

## 4. 数据模型

新增 `knowledge_items`：
- `id` bigint PK
- `question` text
- `answer` text
- `keywords` jsonb（字符串数组）
- `attachments` jsonb（对象数组，包含 `title,url`）
- `created_by` bigint
- `updated_by` bigint
- `created_at` timestamp
- `updated_at` timestamp

## 5. 检索策略

- PostgreSQL/Kingbase：`to_tsvector` + `plainto_tsquery` + `ts_rank`
- SQLite 测试：降级为 `LIKE` 匹配（question/answer/keywords）

## 6. API 约定

- `GET /api/v1/knowledge/search?q=...&limit=20&offset=0`
- `GET /api/v1/admin/knowledge?query=...&limit=20&offset=0`
- `POST /api/v1/admin/knowledge`
- `PATCH /api/v1/admin/knowledge/:id`

响应保持统一：
- 成功：`{"data": ...}`
- 失败：`{"error": "..."}`

## 7. 测试要求

- handler 测试：至少覆盖参数错误、权限拒绝、成功路径
- repo 测试：覆盖关键词命中与无命中
- authz 测试：覆盖 `role>=2` 管理权限矩阵
- 全量通过：`go test ./... -count=1`
