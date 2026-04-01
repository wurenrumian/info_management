# Manage Backend

第二阶段（Knowledge Base）已完成，当前除 Foundation RBAC 外，已支持知识库检索与管理接口。

## Phase 1 Delivered

- Core models: `users` / `classes` / `admin_logs`
- 4-level RBAC actions + scope filtering
- Header-based identity middleware
- Student endpoint: `GET /api/v1/me`
- Admin endpoints:
- `GET /api/v1/admin/users`
- `GET /api/v1/admin/users/:id`
- `PATCH /api/v1/admin/users/:id`
- `GET /api/v1/admin/classes`
- `GET /api/v1/admin/classes/:id`
- `POST /api/v1/admin/classes`
- `PATCH /api/v1/admin/classes/:id`
- `GET /api/v1/admin/logs`

## Run Server

```bash
# optional: DATABASE_DSN for postgres/kingbase-compatible db
# if DATABASE_DSN is empty, app still starts, but db-backed APIs are unavailable
go run ./cmd/server
```

## Identity Headers (Phase 1)

All `/api/v1/*` routes use header identity injection:

- `X-User-Id`
- `X-User-Role`
- `X-User-Class-Id` (for cadre/teacher scope)
- `X-User-Grade` (for teacher grade scope)

## Run Tests

```bash
go test ./... -count=1
```

## API Documentation

- `docs/api/phase1-foundation-api.md`
- `docs/api/phase2-knowledge-api.md`

## Knowledge Base Demo (Phase 2)

```bash
# 1) 启动服务（需先设置 DATABASE_DSN）
go run ./cmd/server

# 2) 安装系统依赖（PDF提取需要）
sudo apt-get install -y poppler-utils

# 3) 生成联调附件（含 docx/xlsx/pdf）
./scripts/dev/make_demo_knowledge_files.sh /tmp

# 4) 调用导入/检索/管理接口（含PDF正文搜索验证）
./scripts/dev/knowledge_api_curl.sh
```

## Kingbase Local Test (Docker)

```bash
# 1) 启动本地金仓容器（个人脚本）
./scripts/dev/kingbase_docker_up.sh

# 2) 设置连接串
export DATABASE_DSN='host=127.0.0.1 port=54321 user=system password=123456 dbname=test sslmode=disable'

# 3) 启动服务并执行接口联调
go run ./cmd/server
./scripts/dev/make_demo_knowledge_files.sh /tmp
./scripts/dev/knowledge_api_curl.sh
```

## Kingbase Local Test (Docker)

```bash
# 1) 启动本地金仓容器（个人脚本）
./scripts/dev/kingbase_docker_up.sh

# 2) 设置连接串
export DATABASE_DSN='host=127.0.0.1 port=54321 user=system password=123456 dbname=test sslmode=disable'

# 3) 启动服务并执行接口联调
go run ./cmd/server
./scripts/dev/make_demo_knowledge_files.sh /tmp
./scripts/dev/knowledge_api_curl.sh
```

## Team Workflow (Superpowers Optional)

- Superpowers is recommended for planning/execution consistency, but not required to run this project.
- Runtime/development baseline remains standard Go commands (`go run`, `go test`) with no superpowers runtime dependency.
- Team-shared artifacts can be committed under `docs/superpowers/` (designs/plans) when they improve collaboration.
- Personal local config (for example user-home codex/superpowers settings) must not be committed.
