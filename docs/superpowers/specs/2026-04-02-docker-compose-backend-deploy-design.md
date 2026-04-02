# Backend Docker Compose 部署设计（简化版）

**日期**：2026-04-02  
**目标**：在本地与服务器使用同一套最小容器化方案，快速提供可联调的远程后端。  
**约束**：低并发业务优先简化；不引入额外网关服务；金仓数据库镜像由环境预先准备。

## 1. 设计结论

采用单后端服务容器方案：
- 使用多阶段 `Dockerfile` 构建 Go 可执行文件
- 使用 `docker-compose.yml` 启动后端容器
- 数据库使用外部金仓实例（通过 `DATABASE_DSN` 连接）
- 文件上传目录使用宿主机目录挂载持久化
- 金仓证书通过只读 volume 挂载到容器

该方案满足：
- 足够简单，维护成本低
- 环境一致（本地/服务器）
- 支持并行开发期间稳定远程联调

## 2. 文件与职责

### 2.1 `Dockerfile`

职责：构建可运行后端镜像。

设计要点：
- builder 阶段使用 Go 官方镜像的国内镜像源
- 配置 `GOPROXY=https://goproxy.cn,direct`
- runtime 阶段使用轻量 Alpine，并切换国内 apk 源
- 仅拷贝可执行文件进入 runtime 镜像
- 暴露 `8080`

### 2.2 `docker-compose.yml`

职责：定义后端运行参数与挂载。

设计要点：
- 单服务：`backend`
- `ports: 8080:8080`
- 使用 `.env` 注入运行变量（`DATABASE_DSN`、`JWT_SECRET` 等）
- 挂载上传目录：`./data/uploads/documents -> /data/uploads/documents`
- 挂载证书：`./license_71193_0.dat -> /certs/kingbase.dat:ro`
- 健康检查：调用 `/healthz`
- 重启策略：`unless-stopped`

### 2.3 `.dockerignore`

职责：减小构建上下文，加速构建。

设计要点：
- 排除 `.git`、`data/`、日志、临时文件、编辑器目录
- 保留构建必须的源码和 `go.mod/go.sum`

## 3. 配置约定

- 统一上传目录变量：`DOCUMENT_UPLOAD_DIR=/data/uploads/documents`
- `DATABASE_DSN` 由 `.env` 注入
- 推荐 DSN（证书模式）示例：
  - `sslmode=verify-ca`
  - `sslrootcert=/certs/kingbase.dat`

说明：如本地联调阶段先排查网络连通性，可临时改 `sslmode=disable`，跑通后再切回证书模式。

## 4. 运行流程

本地/服务器统一流程：
1. 准备 `.env`（含 `DATABASE_DSN`）
2. `docker compose up -d --build`
3. 检查 `GET /healthz`
4. 前端联调

## 5. 风险与边界

- 当前不引入网关（Nginx/Caddy），HTTPS/域名转发后置
- compose 不包含金仓容器定义（由环境预置）
- 如服务器无法直连外网镜像仓库，需提前配置 Docker 镜像加速

## 6. 验收标准

- 镜像可成功构建
- 容器可启动且 `healthz` 返回 200
- 后端可通过金仓 DSN 成功连库并自动迁移
- 上传目录可写入且重启后数据保留
