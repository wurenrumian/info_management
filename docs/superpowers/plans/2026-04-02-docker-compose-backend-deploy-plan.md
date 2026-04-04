# Docker Compose Backend Deploy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a minimal, reproducible Docker-based backend deployment setup (China mirror friendly) that connects to an externally prepared Kingbase instance for local and server-side integration testing.

**Architecture:** Use a multi-stage Docker build for the Go backend binary, then run the backend with docker-compose as a single service. The compose service mounts persistent upload storage and Kingbase certificate, and receives DSN/env config from `.env`.

**Tech Stack:** Go 1.25, Docker multi-stage build, Docker Compose v2, Alpine runtime, Kingbase/PostgreSQL-compatible DSN.

---

### Task 1: Add Docker Build Context Guardrail

**Files:**
- Create: `.dockerignore`
- Verify: `go.mod`

- [ ] **Step 1: Create `.dockerignore` with minimal but safe excludes**

```gitignore
# VCS and editor
.git
.gitignore
.idea
.vscode

# Environment and secrets
.env
.env.local
.env.*.local

# Build/test/cache artifacts
bin
build
dist
coverage
*.log
*.tmp
*.out

# Runtime data
data/uploads

# OS junk
.DS_Store
Thumbs.db

# Not needed in image context
original_request
docs
```

- [ ] **Step 2: Verify Docker context remains buildable**

Run: `test -f go.mod && echo OK`
Expected: prints `OK`

---

### Task 2: Create Multi-Stage Dockerfile with China Mirrors

**Files:**
- Create: `Dockerfile`
- Verify: `cmd/server/main.go`

- [ ] **Step 1: Write builder + runtime Dockerfile**

```dockerfile
# syntax=docker/dockerfile:1

FROM registry.cn-hangzhou.aliyuncs.com/library/golang:1.25-alpine AS builder

WORKDIR /app

# China-friendly module fetch
ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM registry.cn-hangzhou.aliyuncs.com/library/alpine:3.20 AS runtime

WORKDIR /app

# China-friendly apk mirror + CA certs for external services/DB TLS
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 appuser

COPY --from=builder /out/server /app/server

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/server"]
```

- [ ] **Step 2: Build image to verify Dockerfile syntax and dependency resolution**

Run: `docker build -t manage-backend:dev .`
Expected: build completes successfully and outputs final image id.

---

### Task 3: Add docker-compose Runtime Definition

**Files:**
- Create: `docker-compose.yml`
- Verify: `.env.example`, `internal/http/router/router.go`

- [ ] **Step 1: Create compose file for single backend service**

```yaml
services:
  backend:
    container_name: manage-backend
    build:
      context: .
      dockerfile: Dockerfile
    image: manage-backend:dev
    restart: unless-stopped
    ports:
      - "8080:8080"
    env_file:
      - .env
    environment:
      PORT: "8080"
      DOCUMENT_UPLOAD_DIR: /data/uploads/documents
    volumes:
      - ./data/uploads/documents:/data/uploads/documents
      - ./license_71193_0.dat:/certs/kingbase.dat:ro
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1"]
      interval: 10s
      timeout: 3s
      retries: 5
      start_period: 15s
```

- [ ] **Step 2: Validate compose syntax before runtime**

Run: `docker compose config >/tmp/manage-compose.rendered.yml && echo OK`
Expected: prints `OK` and generated config file contains `manage-backend` service.

---

### Task 4: Document Required `.env` Runtime Keys for Compose

**Files:**
- Modify: `.env.example`
- Modify: `README.md`

- [ ] **Step 1: Ensure `.env.example` contains compose-ready DSN and upload notes**

```dotenv
# 服务端口
PORT=8080

# JWT 签名密钥（生产环境必须修改）
JWT_SECRET=dev-secret-change-in-production

# 微信小程序配置
WECHAT_APP_ID=your_app_id
WECHAT_APP_SECRET=your_app_secret

# 统一文件上传目录（容器内固定路径）
DOCUMENT_UPLOAD_DIR=/data/uploads/documents

# 数据库连接（Kingbase/PostgreSQL compatible）
# 证书模式示例（推荐）:
# DATABASE_DSN=host=host.docker.internal port=54321 user=system password=123456 dbname=kingbase sslmode=verify-ca sslrootcert=/certs/kingbase.dat
# 非证书模式示例（仅联调排障）:
# DATABASE_DSN=host=host.docker.internal port=54321 user=system password=123456 dbname=kingbase sslmode=disable
DATABASE_DSN=
```

- [ ] **Step 2: Add README deployment quickstart section for compose**

```markdown
## Docker Compose (Backend Only)

1. Prepare `.env` from `.env.example` and fill `DATABASE_DSN`.
2. Ensure Kingbase is reachable from Docker network (for Linux: `host.docker.internal` mapping via Docker engine).
3. Start backend:

```bash
docker compose up -d --build
```

4. Check health:

```bash
curl -f http://127.0.0.1:8080/healthz
```
```

- [ ] **Step 3: Verify docs updates are present**

Run: `rg -n "Docker Compose \(Backend Only\)|DOCUMENT_UPLOAD_DIR=/data/uploads/documents|DATABASE_DSN" README.md .env.example`
Expected: output contains all three patterns.

---

### Task 5: End-to-End Verification for Local Integration Backend

**Files:**
- Verify: `docker-compose.yml`, `Dockerfile`, `.env`

- [ ] **Step 1: Render and start the service**

Run: `docker compose up -d --build`
Expected: backend container starts without crash-loop.

- [ ] **Step 2: Validate health endpoint**

Run: `curl -i http://127.0.0.1:8080/healthz`
Expected: HTTP 200 and body contains `{"status":"ok"}`.

- [ ] **Step 3: Validate logs for DB connection/migration errors**

Run: `docker compose logs backend --tail=100`
Expected: no fatal startup errors.

- [ ] **Step 4: Stop services cleanly (optional after verification)**

Run: `docker compose down`
Expected: backend container removed and network cleaned.

---

### Task 6: Manual Commit Handoff (User-Owned)

**Files:**
- Verify: `git status`

- [ ] **Step 1: Inspect changed files**

Run: `git status --short`
Expected: only Docker/deployment related files and intended docs changes are listed.

- [ ] **Step 2: User performs commit manually**

```bash
git add Dockerfile docker-compose.yml .dockerignore .env.example README.md
git commit -m "chore: add docker compose deployment for backend"
```

Expected: commit is created by the user (assistant does not commit).
