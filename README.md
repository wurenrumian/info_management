# Manage Backend

## 项目简介

`Manage Backend` 是一个面向校园/组织管理场景的后端服务，提供统一的用户身份体系、权限控制、文件管理、知识库检索与微信相关能力。

项目目标：
- 提供稳定的 REST API 作为前后端协作契约
- 通过 RBAC + Scope 控制不同角色的数据访问边界
- 支持知识库与文件等高频业务能力的快速落地

## 核心能力

- 身份认证与权限控制
- JWT 身份体系（`sub/role/class_id/grade`）
- 4 级角色权限与 Scope 数据过滤

- 用户与班级管理
- 用户信息查询与更新
- 班级管理与管理员审计日志

- 文件管理
- 上传、列表、详情、下载、删除
- 统一文件目录配置（`DOCUMENT_UPLOAD_DIR`）

- 知识库
- 知识检索（学生侧）
- 知识增删改查与附件导入（管理侧）

- 微信相关
- 微信登录与绑定
- 公共注册、开发环境登录辅助
- 订阅上报与通知能力（按配置开关）

## 技术栈

- Go 1.25.0
- Gin
- GORM
- PostgreSQL / Kingbase（生产与联调）
- SQLite in-memory（测试）

## 快速开始

### 1) 配置环境变量

```bash
cp .env.example .env
```

关键变量：
- `DATABASE_DSN`
- `JWT_SECRET`
- `WECHAT_APP_ID` / `WECHAT_APP_SECRET`
- `DOCUMENT_UPLOAD_DIR`
- `APP_ENV`（`dev` 时启用开发辅助接口）

### 2) 启动服务

```bash
go run ./cmd/server
```

默认端口：`8080`（可通过 `PORT` 覆盖）。

## 认证说明

`/api/v1/*` 路由默认需要 JWT（少数 public/dev 接口除外）：

```text
Authorization: Bearer <token>
```

常用认证相关接口：
- `POST /api/v1/auth/public-register`
- `POST /api/v1/wechat/login`
- `POST /api/v1/wechat/bind`
- `POST /api/v1/dev/register-or-login`（仅 dev）
- `POST /api/v1/dev/login-and-send-subscribe-check`（仅 dev）

## 测试与质量检查

```bash
go test ./... -count=1
go vet ./...
```

## API 文档

详见 `docs/api/`：
- `phase1-foundation-api.md`
- `phase2-files-api.md`
- `phase2-knowledge-api.md`
- `phase2-wechat-api.md`
- `notification-api.md`

## 项目目录（简版）

```text
cmd/server                 # 进程入口
internal/http              # handler/middleware/router
internal/service           # 业务服务
internal/repo              # 数据访问层
internal/model             # 数据模型
scripts/dev                # 本地联调脚本
```

## 本地联调脚本（补充）

这些脚本是开发辅助，不是 README 主体。

推荐顺序：
```bash
source ./scripts/dev/export.sh
./scripts/dev/prepare_upload_dirs.sh
```

Windows PowerShell 可用：
```powershell
.\scripts\dev\prepare_upload_dirs.ps1
```

常用脚本：
- `scripts/dev/dev_login_curl.sh`：获取 dev token
- `scripts/dev/upload_api_curl.sh`：文件上传链路联调
- `scripts/dev/make_demo_knowledge_files.sh`：生成知识库导入样例文件
- `scripts/dev/knowledge_api_curl.sh`：知识库全链路联调

## 相关规范

- 开发规范：`docs/development-standard.md`
- API 契约：`docs/api/`
