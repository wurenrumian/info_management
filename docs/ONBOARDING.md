# 新成员上手指南（Onboarding Guide）

本文档帮助新加入的开发者快速搭建开发环境、理解开发流程。

**开发规范详情请阅读：** [`docs/development-standard.md`](./development-standard.md)

---

## 目录

1. [项目概览](#1-项目概览)
2. [推荐：Docker 一键开发](#2-推荐docker-一键开发)
3. [金仓数据库统一配置](#3-金仓数据库统一配置)
4. [Superpowers 开发流程](#4-superpowers-开发流程)
5. [常用命令速查](#5-常用命令速查)
6. [FAQ](#6-faq)

---

## 1. 项目概览

| 项 | 值 |
|---|---|
| 语言 | Go 1.25.0 |
| Web 框架 | Gin |
| ORM | GORM |
| 生产数据库 | Kingbase（PostgreSQL 兼容） |
| 测试数据库 | SQLite in-memory |
| 模块名 | `manage` |

详细目录结构、代码归属、权限规范等见 [`docs/development-standard.md`](./development-standard.md)。

---

## 2. 推荐：Docker 一键开发

> **推荐使用 Docker 开发。** 配置完成后，数据库、后端服务、文件目录全部就绪，与团队其他成员环境完全一致，无需处理本地金仓安装、版本差异、端口冲突等问题。

### 2.1 前置条件

- Git
- Docker & Docker Compose

### 2.2 克隆与初始化

```bash
git clone <repo-url>
cd info_management
cp .env.example .env
```

### 2.3 金仓证书（必须）

将助教提供的 `.dat` 证书文件复制到项目根目录，命名为 `license_71193_0.dat`：

```bash
cp /path/to/助教给的证书.dat ./license_71193_0.dat
```

> 该文件会被 docker-compose.yml 挂载到金仓容器中作为许可证。**没有此文件容器将无法启动。**

### 2.4 配置 .env

```bash
# JWT 签名密钥（生产环境必须修改）
JWT_SECRET=dev-secret-change-in-production

# 微信小程序配置
WECHAT_APP_ID=wx8ca146a717a76c21
WECHAT_APP_SECRET=<你的 app_secret>

# Kingbase 端口映射
KINGBASE_PORT_BIND=54321:54321

# 服务端口
PORT=8080

# 统一文件上传目录
DOCUMENT_UPLOAD_DIR=/data/uploads/documents

# 开发模式（启用 dev 辅助接口）
APP_ENV=dev

# 微信订阅消息开关
WECHAT_SUBSCRIBE_MSG_ENABLED=true
```

> `DATABASE_DSN` 无需手动设置，docker-compose.yml 已内置默认值：`host=kingbase port=54321 user=system password=123456 dbname=kingbase sslmode=disable`

### 2.5 准备上传目录

Docker 容器以 UID 10001（`appuser`）运行，需要提前创建目录并授权：

```bash
./scripts/dev/prepare_upload_dirs.sh
```

### 2.6 启动

```bash
# 启动全部服务（kingbase + backend）
docker compose up -d

# 等待健康检查通过后验证
curl http://localhost:8080/healthz
```

### 2.7 仅启动金仓（本地 go run 开发）

如果你更习惯用 `go run` 调试代码，可以只启动数据库容器：

```bash
# 只启动数据库
docker compose up -d kingbase

# 加载直连环境变量
source ./scripts/dev/export.sh

# 本地运行
go run ./cmd/server
```

### 2.8 常用 Docker 命令

```bash
# 查看服务状态
docker compose ps

# 查看日志
docker compose logs -f backend
docker compose logs -f kingbase

# 停止服务（保留数据）
docker compose down

# 停止并清理数据卷（会丢失数据库数据）
docker compose down -v
```

---

## 3. 金仓数据库统一配置

### 3.1 团队统一参数

| 项 | 值 |
|---|---|
| 用户名 | `system` |
| 密码 | `123456` |
| 端口 | `54321` |
| 数据库 | `kingbase` |

> 以上凭据仅限团队本地开发/联调，生产环境必须使用独立凭据。

### 3.2 DSN 配置场景

**Docker Compose 方式（推荐）：**

```bash
DATABASE_DSN=host=kingbase port=54321 user=system password=123456 dbname=kingbase sslmode=disable
```

> `host=kingbase` 是 docker-compose 内部服务名，容器间通过 DNS 解析。docker-compose.yml 已内置此值。

**本地直连金仓（不推荐，仅特殊场景）：**

```bash
DATABASE_DSN=host=127.0.0.1 port=54321 user=system password=123456 dbname=kingbase sslmode=disable
```

快速加载：`source ./scripts/dev/export.sh`

**启用证书验证：**

```bash
DATABASE_DSN='host=127.0.0.1 port=54321 user=system password=123456 dbname=kingbase sslmode=verify-ca sslrootcert=/path/to/root.dat'
```

**WSL 连接 Windows 本机金仓：**

```bash
DATABASE_DSN=host=host.docker.internal port=54321 user=system password=123456 dbname=kingbase sslmode=disable
```

### 3.3 金仓联合测试

```bash
KINGBASE_DSN='host=127.0.0.1 port=54321 user=system password=123456 dbname=kingbase sslmode=disable' \
go test ./internal/repo -tags=integration -run TestKnowledge -count=1
```

建议联调阶段至少执行一次，尽早暴露环境差异。

---

## 4. Superpowers 开发流程

本项目使用 Superpowers 技能体系，核心原则：**先规划，后开发**。

### 4.1 三步走流程

```
Brainstorming → Writing Plans → Implement (TDD)
   探索设计        拆解步骤        编码实现
```

### 4.2 各阶段说明

**Brainstorming（头脑风暴）**
- 接到新需求时使用
- 探索项目上下文 → 提问澄清需求 → 提出方案对比 → 逐步确认设计
- 产出：`docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md`

**Writing Plans（编写计划）**
- 设计方案确认后使用
- 基于 Spec 拆解为独立可验证的步骤
- 产出：`docs/superpowers/plans/YYYY-MM-DD-<topic>-plan.md`

**Implement（编码实现）**
- 计划确认后使用
- TDD 驱动：先写失败测试 → 最小实现 → 重构
- 保持所有测试通过

### 4.3 其他常用技能

| 技能 | 何时使用 |
|------|----------|
| `systematic-debugging` | 遇到 bug、测试失败、异常行为 |
| `requesting-code-review` | 完成任务、合并前验证 |
| `receiving-code-review` | 收到 review 反馈后 |
| `dispatching-parallel-agents` | 2+ 个独立任务可并行 |

---

## 5. 常用命令速查

### 开发与测试

```bash
go run ./cmd/server              # 启动服务（本地直连）
go test ./... -count=1           # 全部测试
go test ./internal/config/ -v    # 指定包测试
go vet ./...                     # 代码检查
go fmt ./...                     # 格式化
```

### 联调脚本

```bash
source ./scripts/dev/export.sh                     # 加载环境变量
./scripts/dev/prepare_upload_dirs.sh               # 准备上传目录
./scripts/dev/dev_login_curl.sh                    # 获取 dev token
eval $(./scripts/dev/dev_login_export.sh)          # 导出 token
./scripts/dev/make_demo_knowledge_files.sh         # 生成知识库测试文件
./scripts/dev/knowledge_api_curl.sh                # 知识库全链路联调
./scripts/dev/upload_api_curl.sh                   # 文件上传全链路联调
```

### 合并前检查

- [ ] `go test ./... -count=1` 通过
- [ ] `go vet ./...` 通过
- [ ] 新增功能有测试
- [ ] API 文档已同步（`docs/api/`）
- [ ] 设计与计划已写入 `docs/superpowers/`

---

## 6. FAQ

**Q: 我不知道该把代码放哪里？**

A: 见 [`docs/development-standard.md`](./development-standard.md) 第 2-3 节。新模块在 `internal/service/` 下建子目录，model/repo/handler 按约定位置放置。

**Q: 我的模块需要文件上传能力？**

A: 调用 `POST /api/v1/files/upload` API，获得 `file_id` 后以 jsonb 存储引用。禁止各模块自行实现文件保存逻辑。

**Q: 测试怎么写？**

A: 使用 SQLite in-memory：`gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})`。每个模块至少 1 个 handler 测试 + 1 个 repo/service 测试 + 1 条 403 用例 + 1 条跨 scope 用例。

**Q: Docker 启动后连不上数据库？**

A: 检查 `docker compose ps` 确认金仓健康状态，确认 `license_71193_0.dat` 在项目根目录。

**Q: 权限流程怎么走？**

A: JWT 中间件 → `auth.GetActor(c)` → `authz.Authorize()` → `authz.BuildScope()` → repo 查询应用 scope。详见 development-standard.md 第 5 节。

**Q: 开发规范全文在哪？**

A: [`docs/development-standard.md`](./development-standard.md)，包含目录约定、禁止项、接口规范、权限规范、测试规范、代码风格、分支提交规范等完整内容。
