# Phase 2 党团流程模块设计（v0，模糊版）

**日期**：2026-03-31  
**状态**：草案（供团队并行开发对齐边界）  
**目标**：在不锁死细节的前提下，提供可并行开发的最小契约与边界。

## 1. 背景与定位

Phase 1 已完成用户/班级/权限底座，Phase 2 进入业务模块并行开发。  
党团流程模块作为 Phase 2 标杆模块，先给出“可落地但不过度细化”的设计，便于团队成员并行实现。

本设计强调：
- 先形成最小闭环（查询 + 维护 + 导入 + 提醒占位）
- 复用 Phase 1 权限和数据范围
- 保留后续细化空间（状态机规则、提醒策略、模板配置）

## 2. 范围（v0）

### 2.1 包含

- 支持两条流程类型：
- `party`（入党）
- `league`（入团）

- 学生端：
- 查看本人当前阶段
- 查看历史节点
- 查看下一节点提示（文本级提示）

- 管理端：
- 查询流程记录（按班级/年级范围）
- 更新当前阶段
- 补录历史事件
- 批量导入基础进度（Excel）

- 提醒能力：
- 先落“提醒触发点与数据结构”
- 消息发送先走占位能力（站内/小程序订阅接口位），不在 v0 做复杂编排

### 2.2 不包含（v0 非目标）

- 不实现复杂工作流引擎（如任意流程编排、可视化流程设计器）
- 不实现自动抓取外部公众号/网站数据
- 不实现跨系统实名绑定流程
- 不实现邮件/短信多通道通知

## 3. 角色与权限（复用 Phase 1）

- 学生：仅可查看本人流程
- 团干部（ROLE_CADRE）：可查看/维护本班流程
- 教师（ROLE_TEACHER）：可查看本班与本年级，维护先限定本班
- 超级管理员（ROLE_SUPER_ADMIN）：全量维护 + 配置能力

策略要求：
- 继续使用 `Authorize(action)` + `BuildScope(actor)` 两层控制
- 所有查询与更新操作必须应用数据范围约束

## 4. 数据模型（v0 粗粒度）

## 4.1 `party_progress`

记录用户在某流程类型上的当前状态。

建议字段：
- `id` bigint PK
- `user_id` bigint FK
- `type` varchar(10) (`party`/`league`)
- `stage` varchar(32)
- `stage_updated_at` timestamp
- `extra_info` jsonb
- `created_at` timestamp
- `updated_at` timestamp

说明：
- `extra_info` 用于容纳不稳定扩展字段（例如备注、材料状态、临时标签）

## 4.2 `party_progress_events`

记录每次节点变更/补录历史。

建议字段：
- `id` bigint PK
- `progress_id` bigint FK
- `from_stage` varchar(32) NULL
- `to_stage` varchar(32)
- `event_time` timestamp
- `operator_id` bigint FK
- `remark` text NULL
- `attachments` jsonb NULL
- `created_at` timestamp

## 4.3 `party_stage_defs`（可选先建表）

流程阶段定义与顺序配置（给后续细化留位）。

建议字段：
- `id` bigint PK
- `type` varchar(10)
- `stage_code` varchar(32)
- `stage_name` varchar(64)
- `order_no` int
- `rule_json` jsonb NULL
- `enabled` bool

## 5. API 契约（v0）

学生端：
- `GET /api/v1/party-progress/me`
- 返回本人 `party` + `league` 当前状态与历史事件摘要

管理端：
- `GET /api/v1/admin/party-progress`
- 支持按 `type / class_id / grade / stage` 过滤

- `PATCH /api/v1/admin/party-progress/:id/stage`
- 更新当前阶段，并写入事件表

- `POST /api/v1/admin/party-progress/import`
- 上传并导入 Excel（v0 可先支持基础字段）

说明：
- 具体请求/响应字段允许在实现计划中细化
- 但路径语义与权限边界应保持稳定，避免并行开发期间频繁改协议

## 6. 并行开发边界

建议切分为三个可并行子任务：

1. **流程核心与数据层**
- 表结构、repo、service、状态更新与事件记录

2. **管理端接口**
- 列表查询、阶段更新、导入接口与基础校验

3. **学生端查询与提醒占位**
- `/me` 查询聚合、下一节点提示、提醒任务占位结构

统一依赖：
- Phase 1 权限与范围模块
- 统一响应结构与中间件

## 7. 风险与留白

- 阶段名称/顺序/规则目前未完全固定，需允许配置化扩展
- 导入模板可能因团队成员实现选择不同，需统一最小列集合
- 提醒策略（何时触发、触发几次）先做“可插拔”而不做业务硬编码

## 8. 验收标准（v0）

1. 学生可稳定查询本人党团流程状态与历史摘要
2. 管理员在权限范围内可查询并更新阶段
3. 阶段更新会写入事件记录，具备可审计性
4. 导入接口可完成基础批量初始化
5. 至少有一组覆盖角色边界与范围过滤的接口测试通过

## 9. 下一步

本设计通过后，进入 `writing-plans`：
- 输出 Phase 2 党团流程实施计划（任务拆分 + 文件清单 + 测试路径）
- 再将同样模板复用于其他并行模块（知识库、审批、信息发布）
