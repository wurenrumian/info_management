# 知识库问答-附件解耦设计（Issue #27）

## 背景与目标

当前 `POST /api/v1/admin/knowledge/import` 同时承担上传文件、创建问答和隐式关联附件，职责耦合。  
本次改造目标是把知识库拆分为三层能力：

1. 文件管理：沿用 `/api/v1/files/*`
2. 问答管理：沿用 `/api/v1/admin/knowledge*` CRUD
3. 问答-附件关联管理：新增显式关联/解绑/查询接口

并满足以下验收：

- 支持“仅导入文件，不要求 question/answer”
- 支持问答创建后再绑定多个已上传文件
- 支持解绑误关联附件
- 搜索/详情返回的附件来自显式关联

## 方案对比

### 方案 A：继续使用 `knowledge_items.attachments jsonb`，仅新增关联 API

- 优点：改动小，短期实现快
- 缺点：难做唯一约束与审计，后续迁移成本高

### 方案 B：落地关系表 `knowledge_attachments`，并保留 `attachments jsonb` 作为兼容输出（推荐）

- 优点：有唯一约束，关系可审计，支持精确增删改查；兼容老响应字段
- 缺点：需要迁移与双写/回填策略

### 方案 C：立即移除 `attachments jsonb`，全量切换响应结构

- 优点：模型最干净
- 缺点：前后端兼容风险高，改动面大

结论：采用方案 B。

## 数据模型设计

新增表：`knowledge_attachments`

- `id`：主键
- `knowledge_id`：外键（指向 `knowledge_items.id`）
- `file_id`：外键（指向 `documents.id`）
- `created_by`：关联操作人
- `created_at`：关联时间
- 唯一约束：`unique(knowledge_id, file_id)`
- 索引：`knowledge_id`、`file_id`

说明：

- `knowledge_items.attachments` 暂时保留，用于兼容当前响应结构；
- 业务真值来源切换为关系表，`attachments` 由关系表+文档表组装（或同步写入）。

## API 设计

### 1) 新增：批量关联附件

- `POST /api/v1/admin/knowledge/:id/attachments`
- body:

```json
{
  "file_ids": [1, 2, 3]
}
```

- 行为：
  - 校验 knowledge 存在
  - 校验 file 存在
  - 幂等去重（已存在关系不重复插入）
  - 返回 `added_count`、`already_count`、当前关联列表

### 2) 新增：解绑单个附件

- `DELETE /api/v1/admin/knowledge/:id/attachments/:file_id`
- 行为：
  - 若关系不存在，返回 404 或 `deleted:false`（本次采用 404，和现有 not found 风格一致）
  - 删除关系后返回 `deleted:true`

### 3) 新增：查询当前附件关联

- `GET /api/v1/admin/knowledge/:id/attachments`
- 返回按文档信息组装后的附件数组（含 `file_id`、`title`、`url`、`content_type` 等）

### 4) 既有导入接口兼容改造

- `POST /api/v1/admin/knowledge/import` 标记 deprecated
- 支持两种调用：
  1. 仅传 `files`：只导入文件并返回 `file_ids`（不创建问答、不绑定）
  2. 传 `question/answer + files`：兼容旧行为，但内部改为“创建问答 + 关系表绑定”

## 查询与响应策略

管理端和学生端问答查询保持现有 JSON 结构，`attachments` 字段继续返回数组；  
但其内容改为来自 `knowledge_attachments + documents` 的显式关联结果，避免隐式误绑定。

`content_text` 的检索来源改为：

- 优先来自问答本身字段（若已有）
- 绑定文件时可按文件抽取文本聚合更新（后续可异步化；本期先同步拼接）

## 迁移策略

1. 新增 `knowledge_attachments` 模型并 AutoMigrate
2. 启动期执行一次回填：读取 `knowledge_items.attachments` 中可识别的 `file_id`，写入关系表（去重）
3. 新写入统一走关系表
4. 保留 `attachments jsonb` 字段一个过渡周期，后续再评估删除

## 权限与审计

权限沿用 `role >= 2`（`knowledge:create/patch/delete` 维度）。  
新增审计动作：

- `knowledge.attach`
- `knowledge.detach`

## 测试策略

新增/扩展 handler 测试：

1. `POST /admin/knowledge/:id/attachments` 成功、幂等、file 不存在、knowledge 不存在
2. `DELETE /admin/knowledge/:id/attachments/:file_id` 成功、关系不存在
3. `GET /admin/knowledge/:id/attachments` 成功与空结果
4. `import` 仅文件模式成功
5. 兼容模式（问答+文件）成功并可在关联列表中查到
6. 学生搜索返回附件来自显式关联

## 风险与回滚

风险：

- 兼容逻辑双路径可能导致行为分叉
- 历史 `attachments` 数据格式不统一导致回填遗漏

缓解：

- 回填仅处理带 `file_id` 的附件对象，不可解析项记录日志
- 兼容期保留旧字段输出，避免前端一次性改造

回滚：

- 仅回滚新接口路由与关系写入逻辑，旧 `attachments jsonb` 数据仍可读取

