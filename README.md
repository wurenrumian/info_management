# Manage Backend

第一阶段（Foundation RBAC）已完成，当前实现了用户/班级基础模型、四级权限、数据范围控制和基础管理接口。

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

## Team Workflow (Superpowers Optional)

- Superpowers is recommended for planning/execution consistency, but not required to run this project.
- Runtime/development baseline remains standard Go commands (`go run`, `go test`) with no superpowers runtime dependency.
- Team-shared artifacts can be committed under `docs/superpowers/` (designs/plans) when they improve collaboration.
- Personal local config (for example user-home codex/superpowers settings) must not be committed.
