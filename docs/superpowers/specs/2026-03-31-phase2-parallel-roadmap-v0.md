# Phase 2 并行开发总路线图（v1）

**日期**：2026-03-31  
**更新日期**：2026-04-08  
**定位**：团队并行开发对齐文档（结合当前仓库完成度的阶段路线图）  
**前置条件**：Phase 1 基础底座已完成（用户/班级/权限/scope）

## 1. Phase 2 目标

在不引入高复杂度依赖的前提下，完成 5 个低耦合业务模块与必要支撑能力，形成“学生可用 + 管理可用 + 可审计 + 可扩展”的业务中层。

当前阶段的核心原则：
- 先完成结构化、可审计、可测试的业务闭环，再补复杂自动化能力
- 优先复用已完成的认证、scope、通知、文件基础设施
- 不做公众号/网站自动抓取，不做邮件/短信，不做理论自测

## 2. 当前完成度总览

### 已完成或基本完成的支撑能力

- Phase 1 基础底座：用户、班级、RBAC、scope、admin_logs、基础认证链路
- 微信身份能力：`/api/v1/wechat/login`、`/api/v1/wechat/bind`、`/api/v1/auth/public-register`
- 文档库 / 文件基础设施：文件上传、列表、详情、下载、删除、搜索接口已存在，`upload` service 与 `documents` 模型已落地
- 共享通知能力：通知模板、发送日志、未读数、订阅状态上报、微信回调处理已存在
- 知识库主链路：学生搜索/详情、管理员 CRUD、附件绑定、批量导入、AI 问答草稿预览与批量提交已存在

### 仍处于规划或待实现的业务模块

- PartyFlow：仅有 API 占位文档，尚无 spec / plan / 代码
- Approvals：仅有 API 占位文档，尚无 spec / plan / 代码
- Announcements：仅有 API 占位文档，尚无 spec / plan / 代码
- Certificates：已进入路线图，但尚无 API / spec / plan / 代码

### 文档状态分层

| 模块 | API 文档 | Spec | Plan | 代码状态 |
|------|----------|------|------|----------|
| Foundation / WeChat | 已完成 | 已完成 | 已完成 | 已落地 |
| Document Library | 已完成 | 已完成 | 已完成 | 已落地 |
| Knowledge Base | 已完成 | 已完成 | 已完成 | 已落地并有增强项 |
| Notification Infra | 已完成 | 已完成 | 已完成 | 已落地 |
| PartyFlow | 占位稿 | 未完成 | 未完成 | 未启动 |
| Approvals | 占位稿 | 未完成 | 未完成 | 未启动 |
| Announcements | 占位稿 | 未完成 | 未完成 | 未启动 |
| Certificates | 无 | 未完成 | 未完成 | 未启动 |

## 3. Phase 2 模块清单（按当前状态）

### Phase 2-0：文档库 / 文件基础设施（Document Library）

目标：
- 通用文件上传/下载 API（30MB 限制，本地存储）
- `documents` 表与元数据管理
- 共享 `upload` service
- PDF/DOCX/XLSX 文本提取器复用

当前状态：
- spec 已完成：`docs/superpowers/specs/2026-04-01-document-library-design.md`
- plan 已完成：`docs/superpowers/plans/2026-04-01-document-library-plan.md`
- API 文档已完成：`docs/api/phase2-files-api.md`
- 代码已存在：`internal/model/document.go`、`internal/service/upload/service.go`、`internal/http/handler/file_handler.go`

结论：
- 该模块已不再是“待规划”，而是已落地的共享基础设施
- 后续只需按业务模块需要补字段或增强搜索能力，不建议重做边界

### Phase 2-A：党团流程（PartyFlow）

目标：
- 学生查看个人党团阶段、历史节点、下一步提示
- 管理员按范围查询/更新阶段，支持基础导入
- 每次阶段变更留痕
- 按明确规则触发关键节点提醒

建议边界：
- v0 先覆盖“入团 + 入党”两条稳定主线，共用一套阶段流转模型
- 理论自测暂不纳入当前阶段
- 提醒规则只覆盖固定节点，不做复杂工作流编排
- 通知发送复用现有 notification service，不单独造发送模块

当前状态：
- API 文档：`docs/api/phase2-partyflow-api.md`，仍是 placeholder
- spec / plan / 代码：均未开始

### Phase 2-B：知识库问答（Knowledge Base）

目标：
- 结构化 FAQ + 关键词检索
- 学生端搜索与标准答复
- 管理端维护问答条目与附件链接

当前状态：
- 初始 spec 已完成：`docs/superpowers/specs/2026-03-31-phase2-knowledge-design.md`
- 初始 plan 已完成：`docs/superpowers/plans/2026-03-31-phase2-knowledge.md`
- API 文档已完成：`docs/api/phase2-knowledge-api.md`
- 增强设计已补充：
  - `docs/superpowers/specs/2026-04-01-knowledge-enhancements.md`
  - `docs/superpowers/specs/2026-04-05-knowledge-attachment-decoupling-design.md`
  - `docs/superpowers/specs/2026-04-05-knowledge-ai-qa-design.md`
- 增强 plan 已补充：
  - `docs/superpowers/plans/2026-04-01-enhance-knowledge-pdf.md`
  - `docs/superpowers/plans/2026-04-05-knowledge-attachment-decoupling-plan.md`
  - `docs/superpowers/plans/2026-04-05-knowledge-ai-qa-implementation-plan.md`
- 代码与测试已存在：`internal/http/handler/knowledge_handler.go`、`internal/http/handler/admin_knowledge_handler.go`、`internal/repo/knowledge_repo.go`

结论：
- 知识库已从“待开发模块”转为“已实现并持续增强模块”
- `roadmap` 后续应以维护增强项和联调为主，不再作为待补 spec 的对象

### Phase 2-C：审批流程（Approvals）

目标：
- 学生发起请假/盖章等申请
- 管理员处理通过/驳回，支持撤回
- 保存一学期数据，保留审批历史

建议边界：
- v0 先做线性流程
- 复杂流程编排后置
- 优先支持少量固定申请类型，不做通用表单引擎
- 审批超时提醒若纳入 v0，应复用现有通知能力

当前状态：
- API 文档：`docs/api/phase2-approvals-api.md`，仍是 placeholder
- spec / plan / 代码：均未开始

### Phase 2-D：信息发布与精准推送（Announcements）

目标：
- 管理端发布通知并按条件筛选目标群体
- 支持发布面向全体学生的公共消息
- 学生端查看通知与附件
- 支持管理员手动附带公众号文章链接、学校官网链接等官方外链

建议边界：
- 不做外部自动抓取
- 不做短信/邮件通道
- 公共消息与精准投放共用同一发布模型
- 推送链路复用现有 notification service，小程序订阅消息先做能力接线

当前状态：
- API 文档：`docs/api/phase2-announcements-api.md`，仍是 placeholder
- spec / plan / 代码：均未开始

说明：
- 虽然“信息发布”业务模块未开始，但其底层通知能力已经存在
- 模块实施时应优先复用 `internal/service/notification` 与订阅状态能力

### Phase 2-E：电子证明生成（Certificates）

目标：
- 学生选择标准证明类型并生成预览
- 系统基于学生结构化信息填充模板并导出 PDF
- 管理端维护少量标准证明模板与字段映射

建议边界：
- v0 只做少量固定模板，不做开放式模板设计器
- 优先实现生成与下载，不把复杂审批流程绑定为前置条件
- 不依赖大模型生成证明正文

当前状态：
- 尚未创建专属 API 文档
- 尚未创建 spec / plan
- 尚无代码

## 4. 依赖关系与实际并行策略

当前真实依赖关系：
- PartyFlow / Approvals / Announcements / Certificates 都依赖 Phase 1 的认证、RBAC、scope
- PartyFlow / Approvals / Announcements / Knowledge / Certificates 都可复用 Phase 2-0 文件服务
- PartyFlow / Approvals / Announcements 都可复用现有 notification service

因此当前推荐并行方式不是“从零同时开 5 个模块”，而是：
1. 直接复用已完成的基础设施与支撑模块
2. 把剩余未启动业务模块拆成独立 spec / plan
3. 先完成最小闭环 API，再做通知、导入、前端联调

## 5. 统一契约（继续有效）

- API 前缀统一 `/api/v1`
- 统一响应格式：`{"data": ...}` / `{"error": "..."}`
- 权限统一走 `Authorize` + `BuildScope`
- 所有写操作记录 `admin_logs`
- 每个模块必须提供：
- 最少 1 个 handler 测试文件
- 最少 1 个 repo/service 测试文件
- 1 份模块 API 文档（`docs/api/`）

## 6. 代码放置约定（继续有效）

每个模块只能在既定目录扩展，不新增新的技术层级目录：

- `internal/model`：新增模块表结构
- `internal/repo`：新增模块 repo（`*_repo.go` + `*_repo_test.go`）
- `internal/service/<module>`：模块业务规则
- `internal/http/handler`：模块 handler
- `internal/http/router/router.go`：统一注册路由
- `docs/api`：模块接口文档

禁止项：
- 不在模块内自建第二套路由入口
- 不绕过 `authz` 直接查全量数据
- 不在 handler 中写复杂业务规则（规则应在 service 层）

## 7. v1 阶段验收定义

### 已达成

- Foundation / WeChat / Files / Knowledge / Notification 已有文档、代码与测试基础
- 统一响应格式、RBAC、scope、admin_logs 已形成可复用约束

### 待达成

1. `PartyFlow` 补齐 spec + plan + API 文档并进入实现
2. `Approvals` 补齐 spec + plan + API 文档并进入实现
3. `Announcements` 补齐 spec + plan + API 文档并进入实现
4. `Certificates` 完成路线图层面的独立建模并补齐首版 spec
5. 三个未启动模块至少完成 1 次与文件服务、通知服务的契约联调设计

## 8. 下一步（按优先级）

1. 补齐 `PartyFlow` 的正式 spec 与 plan，并将 `phase2-partyflow-api.md` 从 placeholder 升级为正式 API 文档
2. 补齐 `Approvals` 的正式 spec 与 plan，并明确申请类型、状态流转、撤回规则
3. 补齐 `Announcements` 的正式 spec 与 plan，并明确公共消息、精准范围、外链与附件结构
4. 为 `Certificates` 新建独立 spec，明确模板来源、PDF 生成位置与下载链路
5. 等四个模块文档边界稳定后，再进入并行实施
