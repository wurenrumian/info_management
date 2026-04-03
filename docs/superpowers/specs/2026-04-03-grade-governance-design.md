# Grade 治理设计（方案 2：Class 事实源 + User 快照）

## 1. 背景与目标

当前系统同时存在 `users.grade` 与 `classes.grade`，但缺少统一治理规则，容易出现年级不一致，进而影响教师范围权限（`class_id OR grade`）。

本设计目标：
- 明确 `grade` 在权限系统中的语义与来源
- 建立单一事实源（Single Source of Truth）
- 保留现有兼容性，避免一次性大改权限链路
- 建立可审计、可回填、可演进的治理机制

## 2. 语义定义

- `role`：身份（student/cadre/teacher/super_admin），决定授权策略
- `grade`：授权范围字段，不是个人资料
- `enrollment_year`：个人资料字段，不参与权限

结论：
- `grade` 与 `enrollment_year` 可同值，但语义不同，不能混用

## 3. 设计原则

1. `classes.grade` 是事实源
2. `users.grade` 仅作为兼容快照
3. 快照由系统维护，不允许业务接口自由写入
4. 鉴权与 JWT 读取使用“有效 grade”概念：优先事实源，快照兜底
5. 所有 grade 变更必须可追踪

## 4. 数据规则

### 4.1 有效 grade 解析

`effective_grade(user)` 规则：
1. 若 `user.class_id > 0` 且可查到 class，则 `effective_grade = class.grade`
2. 若 class 缺失或查询异常，则回退 `user.grade`

### 4.2 一致性约束

- 对于 `student/cadre/teacher`，若 `class_id` 有效，`users.grade` 必须同步为对应 `classes.grade`
- `super_admin` 可允许 `users.grade` 为空
- 禁止在管理端 `PATCH /admin/users/:id` 直接修改 `grade`

## 5. 写路径治理

### 5.1 用户创建

- 传入 `class_id` 时，创建前先读取 class
- 将 `users.grade` 设为 `classes.grade`

### 5.2 用户改班（class_id 变化）

- 在用户更新流程中检测 `class_id` 变化
- 自动重算并覆盖 `users.grade`

### 5.3 班级改年级（classes.grade 变化）

- 在班级更新流程中检测 `grade` 变化
- 批量同步该班用户的 `users.grade`

## 6. 读路径治理

### 6.1 权限作用域

- `authz.BuildScope` 逻辑保持不变（teacher: `class_id OR grade`）
- 构造 actor/JWT 时优先写入 `effective_grade`

### 6.2 JWT 签发

- 登录与开发登录签发 token 时，`grade` claim 取 `effective_grade`
- 保留 claim 字段，保证现有 middleware 与测试兼容

## 7. 接口行为调整

### 7.1 Admin 用户更新接口

`PATCH /api/v1/admin/users/:id`
- 从可更新字段中移除 `grade`
- 若请求体出现 `grade`，返回 `400`（message: `grade is system-managed`）

### 7.2 Admin 班级更新接口

`PATCH /api/v1/admin/classes/:id`
- 当 `grade` 改变时触发用户批量 grade 同步
- 写入审计日志：`classes.grade_sync`

## 8. 审计与可观测性

新增/补充以下审计行为：
- `users.class_change_sync_grade`
- `classes.patch_sync_users_grade`
- `auth.issue_token_with_effective_grade`（可选，低优先级）

最低要求：班级改年级触发批量同步必须有审计。

## 9. 迁移策略

### 9.1 一次性回填（发布前）

- 遍历 `users`，对存在 `class_id` 的用户，以 class.grade 覆盖 user.grade
- 记录回填数量与失败项

### 9.2 灰度与回滚

- 先上线“同步逻辑 + 兼容读取”
- 再上线“禁止直接改 users.grade”
- 回滚时仍可依赖 `users.grade` 快照运行，不影响基础权限

## 10. 测试策略

### 10.1 Service/Repo

- `effective_grade` 优先 class.grade，缺失时回退 user.grade
- class.grade 变更触发同班用户 grade 同步
- user.class_id 变更触发 user.grade 同步

### 10.2 Handler

- `PATCH /admin/users/:id` 传 `grade` 返回 400
- `PATCH /admin/users/:id` 改 `class_id` 后，返回用户 grade 已同步
- `PATCH /admin/classes/:id` 改 `grade` 后，同班用户 grade 已同步

### 10.3 鉴权/JWT

- token 中 `grade` 使用有效值
- teacher 范围查询对 `class_id OR grade` 回归通过

## 11. 非目标

- 本次不删除 `users.grade` 列
- 本次不重写 `authz` scope 模型
- 本次不做跨班级复杂组织树权限

## 12. 后续演进

当治理稳定后，可评估 Phase 2：
- 将权限查询从 `users.grade` 逐步切到 class 维度实时解析
- 最终评估是否移除 `users.grade` 快照列
