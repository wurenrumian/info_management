# Announcements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现公共消息与定向通知的发布、学生查看、附件外链与订阅消息触发最小闭环。

**Architecture:** 使用单表 `announcements` 承载消息内容、范围与附件信息；学生端按 actor 计算命中结果；发布动作可选择调用现有 `notification.Service` 发送订阅消息。

**Tech Stack:** Go 1.25, Gin, GORM, SQLite (testing), PostgreSQL/Kingbase (production)

---

## File Structure

| 操作 | 文件路径 | 说明 |
|------|----------|------|
| 创建 | `internal/model/announcement.go` | 通知主表 |
| 创建 | `internal/repo/announcement_repo.go` | 公告数据访问 |
| 创建 | `internal/repo/announcement_repo_test.go` | repo 测试 |
| 创建 | `internal/service/announcements/service.go` | 发布与范围过滤逻辑 |
| 创建 | `internal/service/announcements/service_test.go` | service 测试 |
| 创建 | `internal/http/handler/announcement_handler.go` | 公告 API |
| 创建 | `internal/http/handler/announcement_handler_test.go` | handler 测试 |
| 修改 | `internal/service/authz/actions.go` | 新增 announcements 动作 |
| 修改 | `internal/service/authz/authorize.go` | 新增权限 |
| 修改 | `internal/store/db.go` | 加入公告模型 |
| 修改 | `internal/http/router/router.go` | 注册公告路由 |
| 修改 | `docs/api/phase2-announcements-api.md` | 升级为正式文档 |

### Task 1: 模型、权限与迁移

**Files:**
- Create: `internal/model/announcement.go`
- Modify: `internal/service/authz/actions.go`
- Modify: `internal/service/authz/authorize.go`
- Modify: `internal/store/db.go`

- [ ] **Step 1: 创建模型**

```go
// internal/model/announcement.go
package model

import (
	"time"

	"gorm.io/datatypes"
)

type Announcement struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	Title             string         `gorm:"type:varchar(200);not null" json:"title"`
	Content           string         `gorm:"type:text;not null" json:"content"`
	Status            string         `gorm:"type:varchar(20);index;not null" json:"status"`
	AudienceType      string         `gorm:"type:varchar(20);not null" json:"audience_type"`
	TargetScope       datatypes.JSON `gorm:"type:jsonb" json:"target_scope"`
	Tags              datatypes.JSON `gorm:"type:jsonb" json:"tags"`
	AttachmentFileIDs datatypes.JSON `gorm:"type:jsonb" json:"attachment_file_ids"`
	ExternalLinks     datatypes.JSON `gorm:"type:jsonb" json:"external_links"`
	AuthorID          uint           `gorm:"index;not null" json:"author_id"`
	PublishedAt       *time.Time     `json:"published_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}
```

- [ ] **Step 2: 新增 authz 动作**

```go
	ActionAnnouncementsList      = "announcements:list"
	ActionAnnouncementsGet       = "announcements:get"
	ActionAnnouncementsAdminList = "announcements:admin:list"
	ActionAnnouncementsAdminGet  = "announcements:admin:get"
	ActionAnnouncementsCreate    = "announcements:create"
	ActionAnnouncementsPatch     = "announcements:patch"
	ActionAnnouncementsPublish   = "announcements:publish"
	ActionAnnouncementsArchive   = "announcements:archive"
```

- [ ] **Step 3: 写权限规则**

```go
case model.RoleStudent:
	return action == ActionGetMe ||
		action == ActionMePatch ||
		action == ActionProfileHomeGet ||
		action == ActionKnowledgeSearch ||
		action == ActionNotifUnreadGet ||
		action == ActionFilesUpload ||
		action == ActionFilesGet ||
		action == ActionFilesList ||
		action == ActionAnnouncementsList ||
		action == ActionAnnouncementsGet
```

- [ ] **Step 4: 加入迁移**

```go
&model.Announcement{},
```

- [ ] **Step 5: 运行测试**

Run: `go test ./internal/service/authz ./internal/model ./internal/store -count=1`  
Expected: PASS

### Task 2: Repo 与 Service

**Files:**
- Create: `internal/repo/announcement_repo.go`
- Create: `internal/repo/announcement_repo_test.go`
- Create: `internal/service/announcements/service.go`
- Create: `internal/service/announcements/service_test.go`

- [ ] **Step 1: 写 repo**

```go
type AnnouncementRepo struct{ db *gorm.DB }

func NewAnnouncementRepo(db *gorm.DB) *AnnouncementRepo { return &AnnouncementRepo{db: db} }

func (r *AnnouncementRepo) Create(item *model.Announcement) error {
	return r.db.Create(item).Error
}
```

- [ ] **Step 2: 写命中判断**

```go
func MatchAudience(actor auth.Actor, scope AudienceScope, classMajor string) bool {
	if scope.All {
		return true
	}
	// check grades / class_ids / roles / majors
	return false
}
```

- [ ] **Step 3: 写 service 测试**

```go
func TestListForStudentFiltersDraftAndOutOfScope(t *testing.T) {
	// seed one draft, one published in scope, one published out of scope
	// expect only one item returned
}
```

- [ ] **Step 4: 运行测试**

Run: `go test ./internal/repo ./internal/service/announcements -count=1`  
Expected: PASS

### Task 3: Handler、路由与 API 文档

**Files:**
- Create: `internal/http/handler/announcement_handler.go`
- Create: `internal/http/handler/announcement_handler_test.go`
- Modify: `internal/http/router/router.go`
- Modify: `docs/api/phase2-announcements-api.md`

- [ ] **Step 1: 写 handler**

```go
type AnnouncementHandler struct {
	svc *announcements.Service
}

func NewAnnouncementHandler(db *gorm.DB, notifSvc *notification.Service) *AnnouncementHandler {
	return &AnnouncementHandler{
		svc: announcements.NewService(
			repo.NewAnnouncementRepo(db),
			repo.NewUserRepo(db),
			repo.NewClassRepo(db),
			notifSvc,
		),
	}
}
```

```go
func (h *AnnouncementHandler) ListStudent(c *gin.Context) {}
func (h *AnnouncementHandler) GetStudent(c *gin.Context) {}
func (h *AnnouncementHandler) ListAdmin(c *gin.Context) {}
func (h *AnnouncementHandler) GetAdmin(c *gin.Context) {}
func (h *AnnouncementHandler) Create(c *gin.Context) {}
func (h *AnnouncementHandler) Patch(c *gin.Context) {}
func (h *AnnouncementHandler) Publish(c *gin.Context) {}
func (h *AnnouncementHandler) Archive(c *gin.Context) {}
```

- [ ] **Step 2: 注册路由**

```go
announcementHandler := handler.NewAnnouncementHandler(db, notifSvc)

api.GET("/announcements", announcementHandler.ListStudent)
api.GET("/announcements/:id", announcementHandler.GetStudent)

admin.GET("/announcements", announcementHandler.ListAdmin)
admin.GET("/announcements/:id", announcementHandler.GetAdmin)
admin.POST("/announcements", announcementHandler.Create)
admin.PATCH("/announcements/:id", announcementHandler.Patch)
admin.POST("/announcements/:id/publish", announcementHandler.Publish)
admin.POST("/announcements/:id/archive", announcementHandler.Archive)
```

- [ ] **Step 3: 升级 API 文档**

```md
# Phase2 Announcements API

## Student Endpoints

- `GET /api/v1/announcements`
- `GET /api/v1/announcements/:id`

## Admin Endpoints

- `GET /api/v1/admin/announcements`
- `GET /api/v1/admin/announcements/:id`
- `POST /api/v1/admin/announcements`
- `PATCH /api/v1/admin/announcements/:id`
- `POST /api/v1/admin/announcements/:id/publish`
- `POST /api/v1/admin/announcements/:id/archive`
```

- [ ] **Step 4: 运行测试**

Run: `go test ./internal/http/handler ./internal/http/router -count=1`  
Expected: PASS

### Task 4: 发布触达与全量验证

**Files:**
- Modify: `internal/service/announcements/service.go`
- Modify: `internal/service/announcements/service_test.go`

- [ ] **Step 1: 接入通知服务**

```go
if req.SendNotification {
	if err := s.notifSvc.SendBatch(ctx, notification.BatchSendRequest{
		UserIDs:       userIDs,
		TemplateCode:  req.TemplateCode,
		Page:          "/pages/announcement/detail?id=" + strconv.Itoa(int(item.ID)),
		TemplateData:  buildTemplateData(item),
	}); err != nil {
		return nil, err
	}
}
```

- [ ] **Step 2: 跑全量测试**

Run: `go test ./... -count=1`  
Expected: PASS

