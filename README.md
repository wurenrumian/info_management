# Manage Backend

第二阶段（Knowledge Base）已完成，当前除 Foundation RBAC 外，已支持知识库检索与管理接口、微信 OpenID 绑定与登录。

## Phase 1 Delivered

- Core models: `users` / `classes` / `admin_logs`
- 4-level RBAC actions + scope filtering
- JWT-based identity middleware
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

## Phase 2 Delivered

- Knowledge base search and management
- WeChat OpenID binding and login:
- `POST /api/v1/wechat/login` — 微信登录（code → openid → JWT）
- `POST /api/v1/wechat/bind` — 绑定 openid

## Run Server

```bash
# copy .env.example to .env and fill in your values
cp .env.example .env

# optional: DATABASE_DSN for postgres/kingbase-compatible db
# if DATABASE_DSN is empty, app still starts, but db-backed APIs are unavailable
go run ./cmd/server
```

## Authentication (JWT)

All `/api/v1/*` routes (except `/wechat/login` and `/wechat/bind`) require a JWT token:

```
Authorization: Bearer <token>
```

Token claims: `sub` (user ID), `role`, `class_id`, `grade`.

WeChat endpoints:
- `POST /api/v1/wechat/login` — no auth required, returns JWT token
- `POST /api/v1/wechat/bind` — optional auth (with token binds to current user, without token requires `student_id` + `password`)

## Environment Variables

| Variable | Description |
|----------|-------------|
| `JWT_SECRET` | JWT signing key (required in production) |
| `WECHAT_APP_ID` | WeChat Mini Program AppID |
| `WECHAT_APP_SECRET` | WeChat Mini Program AppSecret |
| `DATABASE_DSN` | Database connection string |
| `PORT` | Server port (default: 8080) |
| `DOCUMENT_UPLOAD_DIR` | Unified document upload directory |
| `KNOWLEDGE_UPLOAD_DIR` | Legacy knowledge upload directory (compat only) |
| `WECHAT_SUBSCRIBE_MSG_ENABLED` | Enable WeChat subscribe message sending (default: true) |

## Run Tests

```bash
go test ./... -count=1
```

## API Documentation

- `docs/api/phase1-foundation-api.md`
- `docs/api/phase2-files-api.md`
- `docs/api/phase2-knowledge-api.md`
- `docs/api/phase2-wechat-api.md`
- `docs/api/phase2-partyflow-api.md` (v0 placeholder)
- `docs/api/phase2-approvals-api.md` (v0 placeholder)
- `docs/api/phase2-announcements-api.md` (v0 placeholder)

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

## Team Workflow (Superpowers Optional)

- Superpowers is recommended for planning/execution consistency, but not required to run this project.
- Runtime/development baseline remains standard Go commands (`go run`, `go test`) with no superpowers runtime dependency.
- Team-shared artifacts can be committed under `docs/superpowers/` (designs/plans) when they improve collaboration.
- Personal local config (for example user-home codex/superpowers settings) must not be committed.
