# 开发规范 (Development Standard)

本规范适用于所有小组成员，确保代码风格统一、模块可独立开发测试。

<strong><h1 style="color: red;">注意，本文档未充分考虑后续模块的开发</h1></strong>

---

## 1. 技术栈约定

| 层级            | 技术选型                                                                |
| --------------- | ----------------------------------------------------------------------- |
| 语言            | Go 1.22+                                                                |
| Web 框架        | Gin                                                                     |
| ORM             | GORM                                                                    |
| 开发/测试数据库 | SQLite（`gorm.io/driver/sqlite`）                                       |
| 生产数据库      | PostgreSQL / 人大金仓（Kingbase）                                       |
| 路由注册        | Gin Router                                                              |
| 认证方式        | Header 注入（`X-User-Id` 等），由 `middleware.IdentityFromHeaders` 解析 |

> **注意**：禁止在代码中硬编码数据库连接信息，所有 DSN 通过环境变量 `DATABASE_DSN` 传入。

---

## 2. 项目结构

```
manage/
├── cmd/
│   └── server/              # 应用入口，一个 cmd 对应一种部署形态
├── internal/
│   ├── app/                 # 应用启动逻辑
│   ├── auth/                # 身份标识（Actor 定义）
│   ├── http/
│   │   ├── handler/         # 请求处理（一个文件一个资源，如 admin_user_handler.go）
│   │   ├── middleware/      # 中间件（如身份解析）
│   │   ├── response/        # 统一响应封装
│   │   └── router/          # 路由注册
│   ├── model/               # 数据模型（对应数据库表结构）
│   ├── repo/                # 数据访问层（CRUD 操作）
│   │   └── *_test.go        # repo 层测试文件放同包
│   └── service/
│       └── authz/           # 权限校验逻辑
├── docs/                    # 项目文档（architecture/, api/, specs/）
├── tests/                   # 集成测试或 E2E 测试
└── go.mod
```

**新增模块时的目录结构示例**（以"知识库"为例）：

```
internal/
├── model/
│   └── knowledge.go         # KnowledgeArticle 模型
├── repo/
│   └── knowledge_repo.go    # + knowledge_repo_test.go
├── service/
│   └── knowledge/           # 业务逻辑封装
│       └── search.go
└── http/
    └── handler/
        └── knowledge_handler.go
```

---

## 3. 模块开发规范

### 3.1 Model 层

- 文件名：`model/资源名.go`（单数形式）
- 结构体名：PascalCase，如 `KnowledgeArticle`
- GORM tag 使用小写蛇形：`gorm:"size:20;uniqueIndex;not null"`
- JSON 字段使用 camelCase：`json:"studentId"`

### 3.2 Repo 层

- 文件名：`repo/资源名_repo.go`
- 构造函数：`NewXxxRepo(db *gorm.DB) *XxxRepo`
- 所有 `repo` 包的测试必须使用 SQLite in-memory 数据库：
  ```go
  db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
  require.NoError(t, db.AutoMigrate(&model.Xxx{}))
  ```
- 禁止在 repo 测试中使用真实的 PostgreSQL DSN

### 3.3 Service 层

- 按业务领域组织，如 `service/authz/`、`service/knowledge/`
- 纯业务逻辑，无 HTTP 依赖
- 包含可测试的业务规则

### 3.4 Handler 层

- 文件名：`handler/资源名_handler.go`
- 响应格式统一使用 `response.OK(c, data)` 或 `response.Error(c, status, msg)`
- HTTP 状态码约定：
  - `200` 成功
  - `400` 请求参数错误
  - `401` 未认证
  - `403` 无权限
  - `404` 资源不存在
  - `500` 服务器内部错误
- 禁止在 handler 中直接返回业务错误（如"用户已存在"用 `400` 而非 `200` 带 error 字段）

### 3.5 Router 层

- 所有路由在 `router/router.go` 的 `New(db *gorm.DB) *gin.Engine` 中注册
- 路由分组：`/api/v1` 提供 RESTful API，`/admin` 前缀的路由需校验权限
- 新增路由格式：
  - `GET    /resource`     → List
  - `GET    /resource/:id` → Get
  - `POST   /resource`    → Create
  - `PATCH  /resource/:id` → Update
  - `DELETE /resource/:id` → Delete（按需提供）

---

## 4. 身份与权限

### 4.1 身份标识（Header 注入）

| Header            | 说明                                           |
| ----------------- | ---------------------------------------------- |
| `X-User-Id`       | 用户 ID                                        |
| `X-User-Role`     | 角色（1=学生, 2=班干部, 3=老师, 4=超级管理员） |
| `X-User-Class-Id` | 班级 ID（班干部/老师范围控制用）               |
| `X-User-Grade`    | 年级（老师按年级查看用）                       |

### 4.2 权限校验

- 权限定义在 `service/authz/actions.go`
- 授权检查：`authz.Authorize(actor.Role, action)`
- Scope 控制：调用 `authz.BuildScope(actor)` 生成数据过滤条件

---

## 5. 测试要求

### 5.1 必须测试的场景

| 层次          | 测试内容                         |
| ------------- | -------------------------------- |
| Repo          | CRUD 操作 + scope 过滤逻辑       |
| Service/Authz | 权限边界条件                     |
| Handler       | 请求参数校验、权限拒绝、正常流程 |

### 5.2 测试命令

```bash
go test ./... -count=1 -v
```

### 5.3 测试覆盖率

- **Repo 层**：必须覆盖主要查询路径（List/GET/Patch/Delete）
- **Authz**：覆盖所有角色 × 动作组合

---

## 6. API 设计原则

1. **稳定性**：API 一旦定稿，除非需求变更，不做破坏性修改
2. **版本化**：通过 URL 前缀 `/api/v1/` 区分版本
3. **响应格式**：
   ```json
   // 成功
   {"data": {...}}
   // 失败
   {"error": "错误描述"}
   ```
4. **分页**（List 接口）：使用 `?limit=&offset=` 参数，返回格式：
   ```json
   {"data": [...], "total": 100}
   ```
5. **更新请求**（PATCH）：使用 JSON body，仅传递需修改的字段（部分更新）

---

## 7. 模块 README 要求

每个模块（尤其是新增的功能模块）必须在 `docs/` 下对应的子目录中包含 `README.md`，内容至少包括：

1. **模块概述**：该模块负责什么功能
2. **数据模型**：核心数据结构及字段说明
3. **API 列表**：提供的接口及简要说明
4. **业务规则**：关键业务逻辑说明（如权限、scope 过滤）
5. **测试说明**：如何运行本模块的测试

---

## 8. 代码风格

- **格式化**：`go fmt ./...`
- **导入顺序**：标准库 → 第三方库 → 内部包（用 `goimports` 自动处理）
- **命名**：
  - 包名：简短小写（`repo`, `authz`）
  - 函数/变量：PascalCase 或 camelCase
  - 常量：全大写下划线分隔或 PascalCase（按场景选择，保持一致）
- **错误处理**：禁止忽略 `err`（使用 `_` 显式忽略需加注释说明）
- **注释**：exported 函数/类型必须添加 GoDoc 注释

---

## 9. 数据库迁移

- 开发阶段使用 `db.AutoMigrate(...)`
- 生产环境变更需编写迁移文件或 SQL 脚本，放在 `docs/migrations/` 目录
- **禁止删除已有的 `AutoMigrate` 调用**，仅允许追加新模型

---

## 10. Git 提交规范（推荐）

```
<type>: <简短描述>

type: feat | fix | docs | refactor | test | chore
```

示例：
```
feat: 添加知识库模块基础CRUD
fix: 修复用户scope过滤遗漏班级ID的问题
docs: 更新knowledge模块README
test: 为UserRepo添加Update测试用例
```

---

## 11. 环境变量

| 变量           | 说明                       | 默认值           |
| -------------- | -------------------------- | ---------------- |
| `DATABASE_DSN` | PostgreSQL/Kingbase 连接串 | 空（仅内存模式） |
| `PORT`         | HTTP 监听端口              | `8080`           |

---

## 12. 规范执行

- **PR 必须通过**：`go test ./...` 全部通过
- **Lint 检查**（如有 CI）：
  ```bash
  go vet ./...
  go fmt ./...
  ```
- 代码审查时重点关注：是否破坏既有接口、是否添加测试、是否更新文档

---

## 13. Phase 2 并行开发约定（新增）

### 13.1 并行模块范围

当前约定并行开发 4 个业务模块：

- `partyflow`（党团流程）
- `knowledge`（知识库问答）
- `approvals`（审批流程）
- `announcements`（信息发布与精准推送）

### 13.2 模块目录放置规则（强制）

每个模块必须放在以下既有层级中，不新增新的技术层目录：

- `internal/model`：模块表结构，文件示例：`party_progress.go`
- `internal/repo`：模块数据访问，文件示例：`party_progress_repo.go`
- `internal/service/<module>`：模块业务逻辑，目录示例：`service/partyflow`
- `internal/http/handler`：模块 HTTP 入口，文件示例：`party_progress_handler.go`
- `internal/http/router/router.go`：统一注册模块路由
- `docs/api`：模块 API 文档

### 13.3 并行协作边界

- 禁止模块自行创建独立 router 入口文件。
- 禁止绕过 `Authorize + BuildScope` 做跨范围数据访问。
- 禁止在 handler 中堆积复杂业务规则（必须下沉到 service）。
- 每个模块至少包含：
- 1 个 handler 测试文件
- 1 个 repo 或 service 测试文件
- 1 份 API 文档

### 13.4 合并与验收

- 合并前必须通过：`go test ./... -count=1`
- 每个并行模块至少提供一条权限边界测试（403 场景）
- 每个并行模块至少提供一条 scope 过滤测试（跨班/跨年级场景）
- 由统一集成人员负责跨模块路由冲突与契约冲突处理
