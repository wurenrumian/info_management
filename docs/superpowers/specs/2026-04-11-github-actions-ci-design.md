# GitHub Actions CI Design

## 背景

当前项目已经具备较明确的团队开发规范，包括：

- 合并前必须通过 `go test ./... -count=1`
- 建议执行 `go vet ./...`
- 统一使用 `go fmt ./...`
- 多人并行开发，接口与代码变更通过 PR 集成

仓库当前技术栈简单，后端以 Go 为主，且已经存在稳定的测试资产。第一阶段引入 CI 的目标应聚焦于“质量闸门”而非“复杂交付流水线”，优先降低协作风险、减少低级错误进入主分支。

## 目标

第一版 CI 需要满足以下目标：

1. 在开发者 `push` 和发起 `pull_request` 时自动执行质量检查。
2. 将现有开发规范中的基础检查自动化，减少口头约定和人工漏检。
3. 保持实现简单、可维护、易理解，避免第一版就引入高维护成本。
4. 为后续增强项预留空间，例如 Docker 构建、Kingbase 集成测试、文档一致性检查。

## 非目标

第一版 CI 明确不包含以下内容：

- Docker 镜像构建校验
- Kingbase / PostgreSQL 外部依赖集成测试
- 自动部署流程
- 基于变更路径的接口文档一致性检查
- 额外引入重型 lint 工具（如 `golangci-lint`）

这些能力可以在后续阶段按需追加，但不应成为第一版上线的前置条件。

## 方案概览

采用 GitHub Actions 作为 CI 平台，新增单个 workflow 文件，例如：

- `.github/workflows/ci.yml`

该 workflow 由以下事件触发：

- `push`
- `pull_request`

workflow 内只保留一个主 job：

- `quality`

该 job 串行执行以下检查：

1. 检出代码
2. 安装指定版本 Go
3. 执行 `go fmt ./...` 并校验工作区无额外变更
4. 执行 `go vet ./...`
5. 执行 `go test ./... -count=1`

## 为什么采用单 job

第一版使用单个 `quality` job，而不是拆分为多个并行 job，原因如下：

- 对当前项目规模而言，单 job 足够快，收益高于复杂化带来的成本。
- 失败定位直观，开发者无需在多个 job 间来回切换。
- 更符合当前团队阶段：先建立稳定、可依赖的质量闸门，再考虑优化执行时间。

后续如果测试规模显著增长，再拆分为 `fmt`、`vet`、`test` 多个 job 会更合适。

## 检查项设计

### 1. 格式检查

命令：

```bash
go fmt ./...
git diff --exit-code
```

设计要点：

- CI 不负责自动提交格式化结果。
- 如果 `go fmt` 产生了修改，则 job 失败，并提示开发者先在本地格式化再提交。
- 这种方式与团队当前的开发习惯最一致，也最容易理解。

### 2. 静态检查

命令：

```bash
go vet ./...
```

设计要点：

- 直接使用 Go 官方工具链提供的基础静态检查。
- 保持规则简单，降低误报和维护负担。
- 与现有开发规范对齐，不额外引入团队尚未统一的 lint 风格偏好。

### 3. 测试检查

命令：

```bash
go test ./... -count=1
```

设计要点：

- 这是第一版 CI 中最关键的硬门槛。
- `-count=1` 保持与现有规范一致，避免测试缓存掩盖问题。
- 当前仓库已有较完整测试覆盖，这项检查具备实际拦截价值。

## GitHub 分支保护建议

为了让 CI 真正成为协作规则的一部分，需要同时配置 GitHub Branch Protection。

建议针对主分支（例如 `main`）开启以下选项：

1. Require a pull request before merging
2. Require status checks to pass before merging
3. 将 `quality` 设为 required status check
4. Require branches to be up to date before merging
5. Do not allow bypassing the above settings

这部分配置与 CI workflow 同等重要：

- CI 负责“发现问题”
- 分支保护负责“阻止问题合入主线”

如果只配置 workflow 而不启用保护规则，CI 只能起到提醒作用，无法形成真实的团队约束。

## 与现有规范的映射关系

本方案与 `docs/development-standard.md` 的主要对应关系如下：

- “合并前必须通过 `go test ./... -count=1`” -> CI 硬门槛
- “建议 `go vet ./...`” -> CI 标准检查项
- “格式化：`go fmt ./...`” -> CI 标准检查项
- “多人并行开发，减少冲突，提高可集成性” -> `push + pull_request` 触发
- “接口与代码通过 PR 集成” -> 分支保护要求通过 PR 合并

因此该方案不是额外引入一套新规则，而是将已有规则自动执行化。

## 推荐的 workflow 结构

建议 workflow 采用如下结构：

```yaml
name: ci

on:
  push:
  pull_request:

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - checkout
      - setup-go
      - fmt check
      - vet
      - test
```

实现细节建议：

- Go 版本优先读取仓库当前声明版本，避免 CI 与本地环境漂移。
- 使用 Go module cache 提升执行速度。
- job 名称保持简洁稳定，便于设置为 required status check。

## 后续演进路线

第一版稳定后，可按以下顺序增强：

1. 增加手动触发或特定分支触发的 Kingbase 集成测试
2. 增加 Docker 构建检查，确保部署产物稳定
3. 增加路径感知的规范检查，例如接口变更时提示同步修改 `docs/api`
4. 增加发布流程与部署流程，形成完整 CD

这些增强项应建立在第一版 CI 已被团队稳定使用的前提上，而不是一开始全部加入。

## 风险与取舍

### 优点

- 简单直接，易于团队接受
- 与现有规范高度一致
- 维护成本低
- 能快速提升主分支质量稳定性

### 代价

- 不能覆盖镜像构建层面的错误
- 不能覆盖依赖外部数据库的联调问题
- 不能自动检查文档变更一致性

该取舍是有意为之：第一版优先解决最常见、最值得自动化的问题。

## 落地建议

建议按以下顺序推进：

1. 新增 `.github/workflows/ci.yml`
2. 在仓库设置中开启主分支保护
3. 将 `quality` 配置为 required status check
4. 通知团队统一本地执行：

```bash
go fmt ./...
go vet ./...
go test ./... -count=1
```

5. 观察 1 到 2 周，确认是否存在明显误报、耗时过长或团队摩擦
6. 再决定是否进入第二阶段增强

## 结论

对于当前这个 Go 后端项目，第一版 CI 最合适的方案是：

- 平台：GitHub Actions
- 触发：`push + pull_request`
- 范围：基础质量闸门
- 检查项：`go fmt`、`go vet`、`go test ./... -count=1`
- 协作配套：GitHub Branch Protection

这套方案实现成本低、团队理解成本低、与现有规范一致性高，是当前阶段最稳妥的落地路径。
