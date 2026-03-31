# Phase 2 并行开发总路线图（v0）

**日期**：2026-03-31  
**定位**：团队并行开发对齐文档（先边界、后细节）  
**前置条件**：Phase 1 基础底座已完成（用户/班级/权限/scope）

## 1. Phase 2 目标

在不引入高复杂度依赖的前提下，完成 4 个低耦合业务模块并行开发，形成“学生可用 + 管理可用 + 可审计 + 可扩展”的业务中层。

## 2. 并行模块清单（All Phases）

### Phase 2-A：党团流程（PartyFlow）

- 学生查看个人党团阶段、历史节点、下一步提示
- 管理员按范围查询/更新阶段，支持基础导入
- 事件审计：每次阶段变更留痕

当前状态：待模块负责人产出 v0 spec 与 implementation plan（按统一规范自行落盘）

### Phase 2-B：知识库问答（Knowledge Base）

- 结构化 FAQ + 关键词检索
- 学生端搜索与标准答复
- 管理端维护问答条目与附件链接

建议边界：
- 不接入通用大模型作为核心路径
- 优先“可解释检索结果 + 官方链接”

### Phase 2-C：审批流程（Approvals）

- 学生发起请假/盖章等申请
- 管理员处理通过/驳回，支持撤回
- 保存一学期数据，保留审批历史

建议边界：
- v0 先做线性流程
- 复杂流程编排后置

### Phase 2-D：信息发布与精准推送（Announcements）

- 管理端发布通知并按条件筛选目标群体
- 学生端查看通知与附件
- 推送先走小程序订阅消息能力占位

建议边界：
- 不做外部自动抓取
- 不做短信/邮件通道

## 3. 并行顺序与依赖

并行原则：
- 4 个模块都复用 Phase 1 权限/scope，不互相等待业务实现
- 先统一接口风格、错误码、审计字段，再并行写代码

建议节奏：
1. 第 0 周：统一开发规范与目录约定（本文档 + development-standard）
2. 第 1-2 周：四模块并行开发核心 API（各自最小闭环）
3. 第 3 周：合同测试与联调
4. 第 4 周：小程序/网站前端对接

## 4. 模块负责人建议（可替换）

- A 党团流程：Owner A（后端主开发）+ Tester A
- B 知识库：Owner B + Tester B
- C 审批流程：Owner C + Tester C
- D 信息发布：Owner D + Tester D

统一角色：
- Integrator（建议 1 人）：负责主分支集成、冲突处理、跨模块契约审查

## 5. 统一契约（必须对齐）

- API 前缀统一 `/api/v1`
- 统一响应格式：`{"data": ...}` / `{"error": "..."}`
- 权限统一走 `Authorize` + `BuildScope`
- 所有写操作记录 `admin_logs`
- 每个模块必须提供：
- 最少 1 个 handler 测试文件
- 最少 1 个 repo/service 测试文件
- 1 份模块 API 文档（`docs/api/`）

## 6. 代码放置约定（并行开发核心规则）

每个模块只能在既定目录扩展，不新增新的技术层级目录：

- `internal/model`：新增模块表结构（如 `party_progress.go`）
- `internal/repo`：新增模块 repo（`*_repo.go` + `*_repo_test.go`）
- `internal/service/<module>`：模块业务规则（如 `service/partyflow`）
- `internal/http/handler`：模块 handler（`*_handler.go`）
- `internal/http/router/router.go`：统一注册路由（避免多入口）
- `docs/api`：模块接口文档

禁止项：
- 不在模块内自建第二套路由入口
- 不绕过 `authz` 直接查全量数据
- 不在 handler 中写复杂业务规则（规则应在 service 层）

## 7. 验收门槛（Phase 2 统一）

1. `go test ./... -count=1` 全通过  
2. 角色权限边界有自动化测试覆盖  
3. 跨班/跨年级越权访问被拦截  
4. 模块文档完整（API + 规则 + 测试运行方式）  
5. 合并前至少 1 次跨模块联调记录

## 8. 下一步

1. 先确认本路线图与开发规范  
2. 各模块负责人独立补 v0 spec + implementation plan（A/B/C/D），命名与位置遵循开发规范
3. 进入 subagent-driven 或人工并行实施
