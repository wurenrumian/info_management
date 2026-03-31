# 第一阶段基础底座设计（用户/班级/权限）

**日期**：2026-03-31  
**阶段目标**：实现后续模块可并行开发所依赖的最小稳定底座，包括用户类型、班级关系、四级权限与数据范围控制。

## 1. 背景与范围

当前项目需要先落地可复用的基础能力，避免各业务模块重复定义用户模型、权限规则和班级关系。第一阶段只交付“底座”，不进入高耦合业务模块。

第一阶段包含：
- 后端基础工程：Go + Gin + GORM + Kingbase（按 PostgreSQL 兼容）
- 基础数据模型：`users`、`classes`、`admin_logs`
- 权限模型：四级角色 + 数据范围
- 学生接口：仅 `GET /api/v1/me`（只读本人）
- 管理接口：用户与班级的查询/维护 + 操作日志查询

第一阶段不包含：
- 微信登录绑定流程（`openid` 字段保留但不启用绑定）
- 审批、党团流程、知识库、通知推送、文档库等业务模块

## 2. 设计目标与约束

目标：
- 为后续低重合度模块并行开发提供统一权限与数据边界。
- 保证“学生可查看本人信息，管理员按职责看对应范围数据”。
- 保留后续接入 JWT/小程序登录能力，不破坏当前接口协议。

约束：
- 严格保持四级权限模型。
- 助教归入教师层，第一阶段无审批权限。
- 超级管理员负责学生信息维护并可查看操作日志。

## 3. 整体架构

分层结构（第一阶段）：
- `internal/model`：数据实体与表映射
- `internal/repo`：数据库访问与范围查询
- `internal/service`：角色权限判断、数据范围策略
- `internal/http`：路由、Handler、中间件
- `internal/auth`：开发期身份注入（请求头模拟用户），后续替换为 JWT

核心策略：
1. 先做功能权限判断（能否调用接口）。
2. 再做数据范围判断（能访问哪些记录）。

## 4. 角色与权限模型

角色定义：
- `ROLE_STUDENT = 1`：普通学生
- `ROLE_CADRE = 2`：团支书/团干部
- `ROLE_TEACHER = 3`：班主任/教师/助教（无审批权限）
- `ROLE_SUPER_ADMIN = 4`：超级管理员

功能权限（第一阶段）：
- 学生：`GET /api/v1/me`
- 团干部：查看本班用户与班级数据
- 教师：查看本班与本年级用户/班级数据（无审批）
- 超管：用户与班级全量管理、日志查询

## 5. 数据范围策略（Scope）

范围规则：
- 学生：仅本人（`user.id == actor.id`）
- 团干部：本班（`class_id == actor.class_id`）
- 教师：本班或本年级（`class_id == actor.class_id OR grade == actor.grade`）
- 超管：全量（无附加过滤）

执行方式：
- 通过 `Authorize(action)` 判断功能权限。
- 通过 `ScopeFilter(actor)` 生成查询条件并注入 repository 层。
- 更新/查询详情时均应用相同 scope，避免“列表有过滤、详情越权”的漏洞。

## 6. 数据模型（第一阶段）

### 6.1 users

- `id` bigint PK
- `student_id` varchar(20) UNIQUE
- `name` varchar(50)
- `openid` varchar(100) NULL
- `role` int
- `class_id` bigint FK
- `grade` varchar(10)
- `major` varchar(100)
- `extra_attrs` jsonb
- `created_at` timestamp
- `updated_at` timestamp

说明：
- `extra_attrs` 用于承载休学/复学/特殊身份等变化字段，避免频繁改表。

### 6.2 classes

- `id` bigint PK
- `class_name` varchar(50)
- `grade` varchar(10)
- `major` varchar(100)
- `counselor_id` bigint FK NULL
- `head_student_id` bigint FK NULL
- `created_at` timestamp
- `updated_at` timestamp

### 6.3 admin_logs

- `id` bigint PK
- `admin_id` bigint FK
- `action` varchar(50)
- `target_type` varchar(30)
- `target_id` bigint
- `detail` jsonb
- `ip_address` varchar(50)
- `created_at` timestamp

说明：
- 管理员对用户/班级执行写操作时必须记录日志。

## 7. API 清单（第一阶段）

学生接口：
- `GET /api/v1/me`：获取本人基础信息与班级信息（只读）

管理接口：
- `GET /api/v1/admin/users`：分页查询用户（姓名/学号/年级/班级过滤）
- `GET /api/v1/admin/users/:id`：查看用户详情
- `PATCH /api/v1/admin/users/:id`：更新用户信息（含 `extra_attrs`）
- `GET /api/v1/admin/classes`：分页查询班级
- `GET /api/v1/admin/classes/:id`：查看班级详情
- `POST /api/v1/admin/classes`：创建班级（超管）
- `PATCH /api/v1/admin/classes/:id`：更新班级（超管）
- `GET /api/v1/admin/logs`：查询管理员操作日志（超管）

## 8. 非目标与后续接口兼容性

非目标：
- 不实现微信登录绑定、审批流、知识库、信息发布、通知、文档上传

兼容性要求：
- 保留 `openid` 字段，后续可直接加入绑定流程，不破坏已定义用户结构。
- 所有后续模块调用权限系统时只需声明 `action` 并复用 scope。

## 9. 第一阶段验收标准

1. 四个角色访问同一资源时，功能权限结果符合矩阵。
2. 跨班/跨年级访问被 scope 正确拦截。
3. `GET /api/v1/me` 仅返回本人数据，不能越权读取他人信息。
4. 用户/班级写操作会生成 `admin_logs` 记录。
5. 至少覆盖权限判定与范围过滤关键路径的自动化测试可通过。

## 10. 并行开发衔接点

第一阶段完成后，后续模块（党团流程、知识库、通知、审批、文档库）可并行推进，统一依赖：
- `users/classes` 作为主数据
- `Authorize(action)` 作为功能权限网关
- `ScopeFilter(actor)` 作为数据边界网关

由此保证并行开发时不重复定义身份体系，也不互相覆盖权限逻辑。
