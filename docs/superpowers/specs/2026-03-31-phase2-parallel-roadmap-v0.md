# Phase 2 并行开发总路线图（v2）

**日期**：2026-03-31
**更新日期**：2026-04-27
**定位**：团队并行开发对齐文档（结合当前仓库完成度、参考材料和最新模块取舍）
**前置条件**：Phase 1 基础底座已完成（用户/班级/权限/scope）

## 1. Phase 2 目标

在不引入高复杂度依赖的前提下，完成低耦合业务模块与必要支撑能力，形成“学生可用 + 管理可用 + 可审计 + 可扩展”的业务中层。

当前阶段的核心原则：
- 先完成结构化、可审计、可测试的业务闭环，再补复杂自动化能力。
- 优先复用已完成的认证、scope、通知、文件基础设施。
- 不做公众号/网站自动抓取，不做邮件/短信，不做理论自测。
- 学校正式流程、制度细则和模板材料优先进入文档库/知识库，只有明确适合学院内部流转的事项才进入审批流。

## 2. 当前完成度总览

### 已完成或基本完成的支撑能力

- Phase 1 基础底座：用户、班级、RBAC、scope、admin_logs、基础认证链路。
- 微信身份能力：`/api/v1/wechat/login`、`/api/v1/wechat/bind`、`/api/v1/auth/public-register`。
- 文档库 / 文件基础设施：文件上传、列表、详情、下载、删除、搜索接口已存在，`upload` service 与 `documents` 模型已落地。
- 共享通知能力：通知模板、发送日志、未读数、订阅状态上报、微信回调处理已存在。
- 知识库主链路：学生搜索/详情、管理员 CRUD、附件绑定、批量导入、AI 问答草稿预览与批量提交已存在。

### 已完成设计对齐、待实现的业务模块

- PartyFlow：spec / plan / API 文档已更新到 v1，待进入实现。
- Approvals：spec / plan / API 文档已更新到 v1，待进入实现。
- Certificates：spec / plan / API 文档已更新到 v1，定位为审批流 PDF 生成能力，待进入实现。

### 仍需正式设计的业务模块

- Announcements：已有 API 占位文档，尚未完成正式 spec / plan / 代码。

### 文档状态分层

| 模块 | API 文档 | Spec | Plan | 代码状态 |
|------|----------|------|------|----------|
| Foundation / WeChat | 已完成 | 已完成 | 已完成 | 已落地 |
| Document Library | 已完成 | 已完成 | 已完成 | 已落地 |
| Knowledge Base | 已完成 | 已完成 | 已完成 | 已落地并有增强项 |
| Notification Infra | 已完成 | 已完成 | 已完成 | 已落地 |
| PartyFlow | 已更新 v1 | 已更新 v1 | 已更新 v1 | 待实现 |
| Approvals | 已更新 v1 | 已更新 v1 | 已更新 v1 | 待实现 |
| Certificates | 已更新 v1 | 已更新 v1 | 已更新 v1 | 待实现 |
| Announcements | 占位稿 | 未完成 | 未完成 | 未启动 |

## 3. Phase 2 模块清单（按当前状态）

### Phase 2-0：文档库 / 文件基础设施（Document Library）

目标：
- 通用文件上传/下载 API（30MB 限制，本地存储）。
- `documents` 表与元数据管理。
- 共享 `upload` service。
- PDF/DOCX/XLSX 文本提取器复用。

当前状态：
- spec 已完成：`docs/superpowers/specs/2026-04-01-document-library-design.md`
- plan 已完成：`docs/superpowers/plans/2026-04-01-document-library-plan.md`
- API 文档已完成：`docs/api/phase2-files-api.md`
- 代码已存在：`internal/model/document.go`、`internal/service/upload/service.go`、`internal/http/handler/file_handler.go`

结论：
- 该模块已落地为共享基础设施。
- 参考材料、申请细则、模板类文件优先进入文档库/知识库，而不是为每类材料单独做业务流程。

### Phase 2-A：党团流程（PartyFlow）

目标：
- 学生查看本人当前党团状态、主阶段、历史动作和下一步提示。
- 管理员按 scope 查询、创建、更新、导入学生党团状态。
- 管理员记录谈话、公示、审批、备案、归档等流程动作。
- 系统按启用的提醒规则定时生成提醒，并复用现有通知能力发送。
- 所有状态变化、里程碑和提醒发送均可审计。

已确认边界：
- v1 覆盖“入团 + 入党”两条主线，但不做强工作流引擎。
- 使用独立学生党团状态表、事件表、提醒规则表。
- 规则由后端 seed 默认生成，业务人员通过管理接口启停和调整，不要求手写规则文件。
- 入党积极分子每满 90 天提交一次思想汇报，使用周期提醒规则 `party_activist_report_every_90d`。
- 临时调整某项规则日期先放在 `metadata.reminder_overrides`，批量调整需求明确后再拆表。
- 定时扫描使用 Go 标准库 `time.Timer` / `time.Ticker`，不引入 cron/队列/工作流引擎。

当前状态：
- spec：`docs/superpowers/specs/2026-04-08-partyflow-design.md`
- plan：`docs/superpowers/plans/2026-04-08-partyflow-implementation-plan.md`
- API 文档：`docs/api/phase2-partyflow-api.md`
- 代码：待实现

### Phase 2-B：知识库问答（Knowledge Base）

目标：
- 结构化 FAQ + 关键词检索。
- 学生端搜索与标准答复。
- 管理端维护问答条目与附件链接。
- 承接不适合进入审批流的制度细则、校历、模板文件和参考资料。

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
- 知识库已从“待开发模块”转为“已实现并持续增强模块”。
- 新增参考资料 `docs/reference/` 中，奖学金助学金申请细则、休学复学细则、宿舍调整细则、校历节假日、请假条模板、预算申请模板默认进入文档库/知识库。

### Phase 2-C：审批流程（Approvals）

目标：
- 学生发起请假/离校申请或活动预算申请。
- 学生查看本人申请状态、详情和审批历史。
- 学生可在待审批时撤回。
- 管理员按 scope 查看、处理、转交和提醒。
- 审批动作与管理员操作均留痕。

已确认边界：
- v1 只做固定申请类型：`leave`、`budget`。
- 奖学金/助学金、休学/复学、宿舍调整、校历/节假日、请假条模板、预算模板均进入文档库/知识库，不进入审批流。
- 不做通用表单引擎，不接管学校/学生处/研究生院/宿管/财务等正式系统流程。
- 采用线性单当前审批人模型。
- 团干部可查看和提醒，但默认不能最终审批。
- 教师/超管可审批、驳回、转交、标记过期。
- 超时只提醒，不自动通过或驳回。
- 定时扫描使用 Go 标准库 `time.Timer` / `time.Ticker`。

当前状态：
- spec：`docs/superpowers/specs/2026-04-08-approvals-design.md`
- plan：`docs/superpowers/plans/2026-04-08-approvals-implementation-plan.md`
- API 文档：`docs/api/phase2-approvals-api.md`
- 代码：待实现

### Phase 2-D：信息发布与精准推送（Announcements）

目标：
- 管理端发布通知并按条件筛选目标群体。
- 支持发布面向全体学生的公共消息。
- 学生端查看通知与附件。
- 支持管理员手动附带公众号文章链接、学校官网链接等官方外链。

建议边界：
- 不做外部自动抓取。
- 不做短信/邮件通道。
- 公共消息与精准投放共用同一发布模型。
- 推送链路复用现有 notification service，小程序订阅消息先做能力接线。

当前状态：
- API 文档：`docs/api/phase2-announcements-api.md`，仍是 placeholder。
- spec / plan / 代码：均未开始。

说明：
- 虽然“信息发布”业务模块未开始，但其底层通知能力已经存在。
- 模块实施时应优先复用 `internal/service/notification` 与订阅状态能力。

### Phase 2-E：电子证明与审批 PDF（Certificates）

目标：
- 为审批流程提供 PDF 生成能力，而不是独立的任意证明生成器。
- 学生提交请假/预算申请时，可生成申请材料 PDF，供本人预览和审批人查看。
- 审批通过后，系统生成审批结果凭证 PDF，作为学院内部流转和留痕材料。
- PDF 由服务端基于固定 Typst 模板和结构化数据生成。

已确认边界：
- v1 只服务 `leave` 和 `budget` 两类审批。
- PDF 分两阶段：申请材料 PDF、审批结果凭证 PDF。
- 审批前 PDF 不盖章、不带核验码；审批通过后才生成带编号、核验码和内部章的结果凭证。
- Typst 模板由服务端受控发布，不允许用户上传或编辑 Typst 源码。
- 电子章定位为“内部审批章/系统生成章”，不等同于学校正式公章。
- 不接入学校正式电子签章/CA，不做任意证明类型自助生成。

当前状态：
- spec：`docs/superpowers/specs/2026-04-08-certificates-design.md`
- plan：`docs/superpowers/plans/2026-04-08-certificates-implementation-plan.md`
- API 文档：`docs/api/phase2-certificates-api.md`
- 代码：待实现

## 4. 依赖关系与实际并行策略

当前真实依赖关系：
- PartyFlow / Approvals / Announcements / Certificates 都依赖 Phase 1 的认证、RBAC、scope。
- PartyFlow / Approvals / Announcements / Knowledge / Certificates 都可复用 Phase 2-0 文件服务。
- PartyFlow / Approvals / Announcements 都可复用现有 notification service。
- Certificates 依赖 Approvals 的审批数据与审批状态，但可以先实现模型、渲染器、记录 repo 和 fake service 测试。
- Approvals 可先不阻塞 Certificates 的基础渲染能力，但通过审批详情返回 PDF 列表时需要两边契约对齐。

推荐并行方式：
1. PartyFlow 和 Approvals 可以并行实现，二者业务表和 API 基本独立。
2. Certificates 可以先做模板、记录、渲染器、编号核验能力，再接入 Approvals 的提交和通过动作。
3. Announcements 暂缓到前面三块实现或进入联调后再正式设计，避免同时铺开过多业务面。
4. 文档库/知识库持续承接学校正式流程细则和模板文件，不为这些资料额外做审批流程。

## 5. 统一契约（继续有效）

- API 前缀统一 `/api/v1`。
- 统一响应格式：`{"data": ...}` / `{"error": "..."}`。
- 权限统一走 `Authorize` + `BuildScope`。
- 所有写操作记录 `admin_logs` 或模块内等价审计表。
- 涉及文件上传、下载、PDF 保存的模块统一复用文件服务。
- 每个待实现模块必须提供：
  - 最少 1 个 handler 测试文件。
  - 最少 1 个 repo/service 测试文件。
  - 至少 1 条 403 用例。
  - 至少 1 条跨班/跨年级 scope 用例。
  - 1 份模块 API 文档（`docs/api/`）。

## 6. 代码放置约定（继续有效）

每个模块只能在既定目录扩展，不新增新的技术层级目录：

- `internal/model`：新增模块表结构。
- `internal/repo`：新增模块 repo（`*_repo.go` + `*_repo_test.go`）。
- `internal/service/<module>`：模块业务规则。
- `internal/http/handler`：模块 handler。
- `internal/http/router/router.go`：统一注册路由。
- `docs/api`：模块接口文档。

禁止项：
- 不在模块内自建第二套路由入口。
- 不绕过 `authz` 直接查全量数据。
- 不在 handler 中写复杂业务规则，规则应在 service 层。
- 不绕过统一文件服务直接保存业务附件或生成 PDF。

## 7. v2 阶段验收定义

### 已达成

- Foundation / WeChat / Files / Knowledge / Notification 已有文档、代码与测试基础。
- 统一响应格式、RBAC、scope、admin_logs 已形成可复用约束。
- PartyFlow 的状态管理、提醒规则和技术方案已完成 spec / plan / API 对齐。
- Approvals 的申请类型、状态流转、权限边界和超时提醒已完成 spec / plan / API 对齐。
- Certificates 的双阶段 PDF、Typst 渲染、内部章和核验方案已完成 spec / plan / API 对齐。

### 待达成

1. `PartyFlow` 按 plan 实现模型、repo、service、handler、提醒扫描和测试。
2. `Approvals` 按 plan 实现模型、repo、service、handler、超时提醒和测试。
3. `Certificates` 按 plan 实现模板元数据、PDF 生成记录、Typst 渲染封装、核验接口，并接入审批流。
4. `Announcements` 补齐正式 spec / plan / API 文档并进入实现。
5. 三个待实现模块完成与文件服务、通知服务、审批详情响应的契约联调。

## 8. 下一步（按优先级）

1. 先实现 `PartyFlow`，因为它独立度最高，且能验证“状态表 + 事件表 + 规则表 + 定时提醒”的通用模式。
2. 并行或随后实现 `Approvals`，先打通 `leave` / `budget` 的提交、撤回、审批、转交、提醒闭环。
3. 实现 `Certificates` 的基础 PDF 渲染与记录能力，再接入 `Approvals` 的提交和审批通过动作。
4. 等前三块进入联调后，再为 `Announcements` 使用 brainstorming 单独做正式 spec / plan。
5. 将 `docs/reference/` 中不进入审批流的制度细则和模板材料整理进文档库/知识库导入清单。
