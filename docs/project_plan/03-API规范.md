# 03 API 规范

## 1. 总体规范（全阶段通用）

1. Base URL：`/api/v1`
2. 鉴权：`Authorization: Bearer <access_token>`
3. 数据格式：`application/json`
4. 文件上传：`multipart/form-data`
5. 响应格式：

```json
{
  "code": 0,
  "message": "ok",
  "data": {},
  "request_id": "req_xxx"
}
```

错误码：
1. `0` 成功
2. `1001` 参数错误
3. `1002` 未认证
4. `1003` 无权限
5. `1004` 资源不存在
6. `1005` 业务冲突
7. `1006` 状态不允许
8. `2001` 文件上传失败
9. `3001` 系统错误

---

## 2. Phase 1 详细 API（开发主契约）

详细结构化定义以 `03-openapi-phase1.yaml` 为准，这里给分组清单：

1. Auth
- `POST /auth/wechat-login`
- `POST /auth/bind-student`
- `POST /auth/refresh`
- `POST /auth/logout`

2. Me
- `GET /me`
- `PATCH /me`

3. User Admin
- `GET /users`
- `GET /users/{id}`
- `POST /users`
- `PATCH /users/{id}`

4. Class Admin
- `GET /classes`
- `POST /classes`
- `PATCH /classes/{id}`
- `POST /classes/{id}/members:batch-bind`

5. Admin Admin
- `GET /admins`
- `POST /admins`
- `PATCH /admins/{id}`

6. Import / Export
- `POST /imports/users`
- `POST /imports/classes`
- `GET /imports/{job_id}`
- `GET /exports/users`

7. Admin Logs
- `GET /admin-logs`

### 2.1 Phase 1 通用口径

1. 除 `POST /auth/wechat-login` 与 `POST /auth/refresh` 外，其余接口默认需要 Access Token。
2. 学生仅允许访问本人信息和本人有权限看到的数据。
3. 管理员接口默认要求 L2 及以上角色。
4. 数据范围优先按“可管理班级”判断，其次按“可管理年级”判断，L4 不受范围限制。
5. 列表接口默认分页；若未显式传入分页参数，使用文档默认值。

### 2.2 Auth 语义

1. `POST /auth/wechat-login`
- 入参为微信登录 `code`。
- 若系统中已存在对应 `openid` 且已绑定学号，则返回 Access Token、Refresh Token 与用户信息。
- 若仅拿到微信身份但未完成学号绑定，返回未绑定态所需信息，由前端继续调用绑定接口。

2. `POST /auth/bind-student`
- 绑定字段最小集为 `student_id + name`。
- 若学号与姓名校验失败，返回 `1005`。
- 若该学号已绑定其他微信身份，返回 `1005`。

3. `POST /auth/refresh`
- 使用 Refresh Token 换取新的 Access Token。
- 失效或非法 Refresh Token 返回 `1002`。

4. `POST /auth/logout`
- 服务端使当前会话失效或标记 Refresh Token 不再可用。

### 2.3 Me 语义

1. `GET /me`
- 返回当前登录用户基础信息、角色、班级、年级、专业、扩展属性。

2. `PATCH /me`
- 仅允许修改个人资料字段，如 `phone`、允许自维护的 `extra_attrs`。
- 不允许通过该接口修改角色、学号、班级归属。

### 2.4 Users / Classes / Admins 语义

1. `GET /users`
- 支持按关键字、角色、班级、年级过滤。
- 结果集受调用者数据范围限制。

2. `POST /users` / `PATCH /users/{id}`
- 仅管理员可用。
- 创建和更新必须校验学号唯一性、班级引用合法性和角色合法性。

3. `GET /classes` / `POST /classes` / `PATCH /classes/{id}`
- 仅管理员可用。
- 班级名称、年级、专业构成班级核心识别信息。

4. `POST /classes/{id}/members:batch-bind`
- 仅管理员可用。
- 对 `user_ids` 批量绑定到指定班级。
- 已在其他有效班级中的用户，按业务冲突处理，返回 `1005` 或在导入场景中计入失败明细。

5. `GET /admins` / `POST /admins` / `PATCH /admins/{id}`
- 用于维护“用户 -> 管理员能力”的映射和范围。
- 管理员等级与用户基础角色口径必须保持一致。

### 2.5 Import / Export 语义

1. `POST /imports/users`
- 上传用户 Excel 文件，创建异步导入作业。
- 立即返回 `job_id`，不要求请求内完成全部解析。

2. `POST /imports/classes`
- 上传班级 Excel 文件，创建异步导入作业。

3. `GET /imports/{job_id}`
- 返回作业类型、状态、总数、成功数、失败数、错误报告路径与明细摘要。

4. `GET /exports/users`
- 按当前筛选条件导出用户数据。
- 导出结果可以是文件流或签名下载地址，但返回方式必须在实现中固定，不允许同环境混用。

### 2.6 Admin Logs 语义

1. `GET /admin-logs`
- 支持按管理员、动作、目标对象、时间区间筛选。
- 日志查询同样受调用者权限限制，普通管理员不能查询超出其范围的敏感日志。

### 2.7 Phase 1 最低错误场景

1. 参数缺失、格式错误：`1001`
2. 未登录、Token 失效：`1002`
3. 已登录但越权：`1003`
4. 查询对象不存在：`1004`
5. 绑定冲突、导入冲突、状态冲突：`1005`
6. 不允许的状态变更：`1006`

---

## 3. Phase 2 API（文字草案）

### 3.1 `/knowledge/*`

1. 学生端：
- 知识检索
- 知识详情
- 热门问题

2. 管理端：
- 新增知识条目
- 更新知识条目
- 上下线与分类维护

### 3.2 `/announcements/*`

1. 管理端：
- 创建通知
- 编辑通知
- 发布与撤回
- 目标范围预览
- 发送记录查询

2. 学生端：
- 通知列表
- 通知详情
- 已读状态回传

### 3.3 `/party-progress/*`

1. 学生端：
- 当前阶段查询
- 关键节点说明

2. 管理端：
- 阶段批量更新
- 单人纠正
- 提醒规则维护

### 3.4 `/approvals/*`

1. 学生端：
- 提交申请
- 查看状态
- 撤回申请

2. 管理端：
- 待办列表
- 审批通过
- 驳回
- 转交
- 历史查询

### 3.5 `/documents/*` + `/certificates/*`

1. 文档库：
- 文档上传
- 文档分类
- 文档查询
- 模板维护

2. 电子证明：
- 证明申请
- 预览
- 下载
- 生成记录查询

说明：Phase 2 各线先补充本节字段草案，再进入开发。

---

## 4. 维护规则

1. API 改动先更新本文件
2. 若影响 Phase 1 结构化定义，必须同步更新 `03-openapi-phase1.yaml`
3. 未更新文档的接口，不进入联调与合并
