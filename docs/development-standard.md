# 开发规范（Team Development Standard）

本规范用于多人并行开发，目标是：
- 减少冲突
- 提高可集成性
- 让每个模块可独立开发与验证

适用范围：本仓库所有后端开发、测试、文档与提交流程。

---

## 1. 技术基线

- 语言：Go 1.24.1+
- Web：Gin
- ORM：GORM
- 生产数据库：PostgreSQL / Kingbase（PostgreSQL 兼容）
- 测试数据库：SQLite（in-memory）

约束：
- 禁止硬编码数据库连接信息
- 连接串只允许通过 `DATABASE_DSN` 注入

团队开发环境（Kingbase 本地/测试容器）统一约定：
- 用户名：`system`
- 密码：`123456`
- 数据库：`test`
- 端口：`54321`
- 证书：使用金仓提供的根目录 `.dat` 根证书文件（本机路径）

推荐 `DATABASE_DSN` 写法（开发环境参考）：
```bash
export DATABASE_DSN='host=127.0.0.1 port=54321 user=system password=123456 dbname=test sslmode=verify-ca sslrootcert=/path/to/root.dat'
```

说明：
- 以上账号密码仅用于团队本地开发/联调，不得用于生产环境
- 生产环境必须使用独立凭据，并通过部署平台注入 `DATABASE_DSN`

---

## 2. 统一目录约定（强制）

```text
manage/
├── cmd/server/                     # 进程入口
├── internal/
│   ├── app/                        # 应用启动
│   ├── auth/                       # Actor / 身份上下文
│   ├── model/                      # 表结构定义
│   ├── repo/                       # 数据访问
│   ├── service/                    # 业务规则（按模块分目录）
│   │   ├── authz/
│   │   ├── partyflow/
│   │   ├── knowledge/
│   │   ├── approvals/
│   │   └── announcements/
│   └── http/
│       ├── handler/                # HTTP 入口
│       ├── middleware/
│       ├── response/
│       └── router/
├── tests/                          # 集成/契约测试
└── docs/
    ├── api/
    └── superpowers/
```

禁止项：
- 禁止创建第二套路由入口（仅 `internal/http/router/router.go` 可注册路由）
- 禁止把业务规则写进 handler（必须下沉到 service）
- 禁止模块绕过 `authz + scope` 直接全量查数据

---

## 3. Phase 2 并行模块与代码归属

当前并行模块：
- `partyflow`
- `knowledge`
- `approvals`
- `announcements`

每个模块代码必须放在以下位置：
- model：`internal/model/<module>*.go`
- repo：`internal/repo/<module>*_repo.go`
- service：`internal/service/<module>/...`
- handler：`internal/http/handler/<module>*_handler.go`
- API 文档：`docs/api/<module>-api.md`

示例（partyflow）：
- `internal/model/party_progress.go`
- `internal/repo/party_progress_repo.go`
- `internal/service/partyflow/service.go`
- `internal/http/handler/party_progress_handler.go`
- `docs/api/phase2-partyflow-api.md`

---

## 4. 接口与响应规范

- API 前缀：`/api/v1`
- 成功响应：`{"data": ...}`
- 失败响应：`{"error": "..."}`

HTTP 状态码：
- `200` 成功
- `400` 参数错误
- `401` 未认证
- `403` 无权限
- `404` 资源不存在
- `500` 服务器错误

列表接口建议：
- 入参：`limit`, `offset`
- 返回：`{"data": [...], "total": N}`（若有总数需求）

---

## 5. 身份与权限规范

身份来源（当前）：JWT token（`Authorization: Bearer <token>`）

Token 由 `POST /api/v1/wechat/login` 返回，包含 `sub`（user ID）、`role`、`class_id`、`grade`。

环境变量：
- `JWT_SECRET` — JWT 签名密钥（生产环境必须设置）
- `WECHAT_APP_ID` / `WECHAT_APP_SECRET` — 微信小程序配置

权限流程（必须）：
1. JWT 中间件解析 token 并注入 `auth.Actor`
2. `auth.GetActor(c)`
3. `authz.Authorize(actor.Role, action)`
4. `authz.BuildScope(actor)`
5. repo 查询/更新应用 scope

测试中生成 token 使用 `testutil.GenerateTestToken(userID, role, classID, grade)`。

---

## 6. 数据与迁移规范

- 开发阶段：`AutoMigrate` 允许追加模型
- 禁止删除既有 `AutoMigrate` 调用
- 生产迁移脚本（若需要）放 `docs/migrations/`

GORM 字段约定：
- 表字段语义稳定优先，变化字段优先放 `jsonb`
- 时间字段统一：`CreatedAt` / `UpdatedAt`（日志类至少 `CreatedAt`）

---

## 7. 测试规范（强制）

SQLite in-memory 统一写法：
```go
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
```

每个模块最少测试资产：
- 1 个 handler 测试文件（权限/参数/正常流程）
- 1 个 repo 或 service 测试文件（scope 或核心规则）
- 至少 1 条 403 用例
- 至少 1 条跨班/跨年级 scope 用例

金仓集成测试（长期建议）：
```bash
KINGBASE_DSN='host=127.0.0.1 port=54321 user=system password=123456 dbname=test sslmode=disable' \
go test ./internal/repo -tags=integration -run TestKnowledgeRepoSearchWithKingbase -count=1
```
说明：
- 该命令用于验证知识库在 Kingbase 的真实全文检索链路（`to_tsvector/plainto_tsquery/ts_rank`）
- 知识库检索范围应覆盖：`question` / `answer` / `keywords` / `content_text`（导入文档抽取文本）
- 若启用证书，请将 `sslmode` 与 `sslrootcert` 按实际环境调整
- 不强制固定 DSN 模板，允许根据 Docker/Windows/本地部署方式调整连接参数
- 无论是否改动 SQL，联调阶段都建议至少执行 1 次金仓联合测试，尽早暴露环境差异问题

金仓联合测试环境约定（Docker / Windows）：
- Docker 镜像场景：建议端口映射 `54321:54321`，DSN 使用 `host=127.0.0.1 port=54321`
- Windows 本机金仓场景：DSN 优先使用 `host=127.0.0.1`；若 WSL 无法直连，再改用 `host=host.docker.internal` 或实际 Windows IP
- 证书场景：统一使用金仓根目录 `.dat` 根证书文件，参考写法：
```bash
KINGBASE_DSN='host=127.0.0.1 port=54321 user=system password=123456 dbname=test sslmode=verify-ca sslrootcert=/path/to/root.dat'
```
- 集成测试通过标准：测试中需至少覆盖“建表/迁移 + 插入 + 检索命中 + 检索无命中”完整链路

合并前必须通过：
```bash
go test ./... -count=1
```

---

## 8. 代码风格规范

- 格式化：`go fmt ./...`
- 建议：`go vet ./...`
- 包名：简短小写
- 文件名：按资源命名（`*_repo.go`, `*_handler.go`）
- 导出符号必须有 GoDoc 注释
- 禁止静默吞错；显式忽略错误需有理由

---

## 9. 分支与提交规范

分支命名建议：
- `feat/<module>-<topic>`
- `fix/<module>-<topic>`
- `docs/<topic>`

提交信息：
- `<type>: <summary>`
- type: `feat|fix|docs|refactor|test|chore`

建议每个任务一个 commit，避免“超大混合提交”。

---

## 10. 并行协作流程

1. 先对齐模块 spec 与 plan（`docs/superpowers/...`）
2. 按模块 owner 开发，避免跨模块大范围改动
3. 路由冲突和契约冲突由集成人员统一处理
4. 合并前完成：代码 + 测试 + API 文档

接口变更流程：
- 先改 `docs/api/*.md`
- 再改代码
- 必须在 PR 中明确“兼容/不兼容”

---

## 11. 文档要求

每个模块至少提供：
- 模块说明（职责）
- 数据模型
- API 列表
- 权限规则
- 测试命令

推荐位置：
- API 文档：`docs/api/`
- 设计与计划：`docs/superpowers/specs/`、`docs/superpowers/plans/`

---

## 12. 联调脚本管理

团队联调脚本统一放在：`scripts/dev/`

约束：
- 可复用的联调脚本应提交到仓库，避免个人本地散落脚本
- 脚本必须通过环境变量接收可变参数，不得硬编码生产凭据
- 脚本只提交源码，不提交运行产物（例如上传目录内文件）

当前知识库联调脚本：
- `scripts/dev/make_demo_knowledge_files.sh`（生成联调用 docx/xlsx）
- `scripts/dev/knowledge_api_curl.sh`（调用知识库导入/搜索/详情/更新/删除接口，包含基础断言）
- `scripts/dev/knowledge_repo_kingbase_integration.sh`（运行知识库金仓集成测试）

---

## 13. CI / 合并门槛

最低门槛：
- `go test ./... -count=1` 通过
- 新增功能有测试
- 路由/API 文档已同步
- 不破坏现有权限边界

未满足以上任一项，不建议合并。
