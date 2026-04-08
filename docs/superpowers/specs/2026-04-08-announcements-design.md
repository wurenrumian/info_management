# 信息发布模块设计（v0）

**日期**：2026-04-08  
**阶段**：Phase 2-D

## 1. 目标

实现学院通知发布的最小闭环：
- 管理员发布公共消息或定向通知
- 学生查看命中自己的通知
- 通知可附带附件与官方外链
- 发布后可复用现有通知能力触发订阅消息

## 2. 范围

### In Scope

- 公共消息发布
- 按年级 / 专业 / 班级 / 角色定向发布
- 外链与文件附件
- 学生端列表与详情
- 管理端创建、更新、发布、下线

### Out of Scope

- 外部公众号 / 网站自动抓取
- 邮件 / 短信发送
- 富文本编辑器复杂能力
- 阅读回执与已读统计细化

## 3. 数据模型

### 3.1 `announcements`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | 主键 |
| title | varchar(200) | 标题 |
| content | text | 正文 |
| status | varchar(20) | `draft` / `published` / `archived` |
| audience_type | varchar(20) | `all` / `targeted` |
| target_scope | jsonb | 定向条件 |
| tags | jsonb | 标签数组 |
| attachment_file_ids | jsonb | 附件文件 ID 列表 |
| external_links | jsonb | 外链列表 |
| author_id | bigint FK | 发布人 |
| published_at | timestamp | 发布时间 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

### 3.2 `external_links` 结构

```json
[
  {
    "title": "学院官网通知",
    "url": "https://example.edu.cn/news/123",
    "source": "school_site"
  }
]
```

### 3.3 `target_scope` 结构

```json
{
  "grades": ["2023", "2024"],
  "majors": ["信息管理"],
  "class_ids": [11, 12],
  "roles": [1]
}
```

## 4. 业务规则

- `audience_type=all` 时，`target_scope` 为空对象
- `draft` 状态学生不可见
- `published` 状态按目标范围过滤可见性
- `archived` 状态默认不在学生列表展示
- 发布时可选择是否触发订阅消息发送

## 5. API 设计

### 5.1 学生端

- `GET /api/v1/announcements?limit=20&offset=0`
- `GET /api/v1/announcements/:id`

### 5.2 管理端

- `GET /api/v1/admin/announcements?status=draft&limit=20&offset=0`
- `GET /api/v1/admin/announcements/:id`
- `POST /api/v1/admin/announcements`
- `PATCH /api/v1/admin/announcements/:id`
- `POST /api/v1/admin/announcements/:id/publish`
- `POST /api/v1/admin/announcements/:id/archive`

创建请求体：

```json
{
  "title": "五一假期安全提醒",
  "content": "请同学们离校前做好登记。",
  "audience_type": "targeted",
  "target_scope": {
    "grades": ["2023"],
    "majors": ["信息管理"]
  },
  "tags": ["假期", "安全"],
  "attachment_file_ids": [21],
  "external_links": [
    {
      "title": "学院官网原文",
      "url": "https://example.edu.cn/news/holiday",
      "source": "school_site"
    }
  ]
}
```

发布请求体：

```json
{
  "send_notification": true,
  "template_code": "announcement_publish"
}
```

## 6. 权限与 Scope

- 学生：只能查看命中自己范围的已发布通知
- 管理员：只能管理自身 scope 可覆盖的通知
- 超管：可管理所有通知
- 发布、归档、编辑操作记录 `admin_logs`

新增 authz 动作建议：

```go
ActionAnnouncementsList    = "announcements:list"
ActionAnnouncementsGet     = "announcements:get"
ActionAnnouncementsAdminList = "announcements:admin:list"
ActionAnnouncementsAdminGet  = "announcements:admin:get"
ActionAnnouncementsCreate  = "announcements:create"
ActionAnnouncementsPatch   = "announcements:patch"
ActionAnnouncementsPublish = "announcements:publish"
ActionAnnouncementsArchive = "announcements:archive"
```

## 7. 代码结构

```text
internal/
├── model/
│   └── announcement.go
├── repo/
│   ├── announcement_repo.go
│   └── announcement_repo_test.go
├── service/
│   └── announcements/
│       ├── service.go
│       └── service_test.go
└── http/
    └── handler/
        ├── announcement_handler.go
        └── announcement_handler_test.go
```

## 8. 测试策略

- handler 测试：
  - 学生只能看到命中范围的 `published` 通知
  - draft 通知学生不可见
  - 管理员创建、发布、归档成功
  - 非 scope 内管理员操作 403
- repo / service 测试：
  - 范围过滤正确
  - `audience_type=all` 能命中所有学生
  - 发布动作会写 `published_at`
  - 开启 `send_notification` 时调用通知服务

## 9. 验收标准

- 公共消息与定向通知均可发布
- 学生可稳定查看命中的通知列表和详情
- 附件与外链可同时存在
- 管理员操作有审计记录

