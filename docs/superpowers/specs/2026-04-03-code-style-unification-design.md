# 代码风格统一与冗余消除设计文档

**日期**: 2026-04-03
**状态**: 待审核
**范围**: 全项目代码风格统一性修复 + 冗余代码消除

---

## 概述

本项目在代码审查中发现 10+ 处冗余代码和多处风格不一致问题。本设计采用**渐进式重构**策略，按优先级分 4 个阶段逐步修复，每次改动小且独立可测试。

---

## 阶段一：Bug 修复（P0）

### 1.1 修复 `ErrRecordNotFound` 比较

**问题**: `subscribe_handler.go:65` 使用 `err == gorm.ErrRecordNotFound` 直接比较，无法处理 wrapped errors。

**修复**:
```go
// Before
if err == gorm.ErrRecordNotFound {

// After
if errors.Is(err, gorm.ErrRecordNotFound) {
```

### 1.2 定义哨兵错误替换字符串匹配

**问题**: `me_handler.go:94` 使用 `err.Error() == "empty patch"` 字符串比较，脆弱且不可靠。

**修复**:
- 在 `internal/service/profile/service.go` 中定义: `var ErrEmptyPatch = errors.New("empty patch")`
- 在 `profile/service.go:86` 返回该哨兵错误
- 在 `me_handler.go:94` 使用 `errors.Is(err, profile.ErrEmptyPatch)` 替换字符串比较

### 1.3 修复 `wechat_handler.go` 错误处理

**问题**: `writeDevLoginError` (line 429-436) 使用 `err.Error()` 字符串匹配。

**修复**: 定义哨兵错误（如 `ErrDevLoginFailed`），使用 `errors.Is()` 替换。

---

## 阶段二：提取通用层（P1）

### 2.1 通用 `UpdateByID` 方法

**问题**: `user_repo.go:58-67`、`class_repo.go:70-79`、`knowledge_repo.go:173-182` 存在完全相同的 `Updates` + `RowsAffected` 逻辑。

**设计**: 新建 `internal/repo/base_repo.go`:
```go
package repo

import "gorm.io/gorm"

// UpdateByID updates a record by ID and returns ErrRecordNotFound if no rows affected.
func UpdateByID(db *gorm.DB, table string, id uint64, updates map[string]any) error {
    result := db.Table(table).Where("id = ?", id).Updates(updates)
    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return gorm.ErrRecordNotFound
    }
    return nil
}
```

**迁移**: 修改 3 个 repo 的 `UpdateByID` 方法调用通用函数。

### 2.2 用户响应 Helper

**问题**: `wechat_handler.go` 中 3 处（line 105-116, 198-209, 282-293）构建完全相同的 `gin.H` 用户响应。

**设计**: 在 `wechat_handler.go` 中提取:
```go
func buildUserResponse(u *model.User) gin.H {
    return gin.H{
        "id":       u.ID,
        "username": u.Username,
        "role":     u.Role,
        "class_id": u.ClassID,
    }
}
```

### 2.3 审计日志 Helper

**问题**: 6+ 处重复的 `_ = h.logRepo.Create(&model.AdminLog{...})` 模式。

**设计**: 新建 `internal/service/audit/logger.go`:
```go
package audit

import (
    "github.com/gin-gonic/gin"
    "manage/internal/auth"
    "manage/internal/model"
    "manage/internal/repo"
)

type Logger struct {
    logRepo *repo.AdminLogRepo
}

func NewLogger(logRepo *repo.AdminLogRepo) *Logger {
    return &Logger{logRepo: logRepo}
}

func (l *Logger) Log(c *gin.Context, actor *auth.Actor, action, targetType string, targetID uint64, details string) {
    _ = l.logRepo.Create(&model.AdminLog{
        ActorID:    actor.UserID,
        Action:     action,
        TargetType: targetType,
        TargetID:   targetID,
        Details:    details,
    })
}
```

**迁移**: 替换所有 handler 中的 `_ = h.logRepo.Create(...)` 调用。

### 2.4 统一列表响应

**问题**: 部分列表接口使用 `response.List()` 返回 total，部分使用 `response.OK()` 不返回 total。

**修复**:
- `admin_class_handler.go:39`、`admin_user_handler.go:41`、`admin_log_handler.go:35` 改为返回 total
- 全部使用 `response.List(c, items, total)`
- 分页参数统一使用 `normalizePage` 逻辑（参考 `knowledge/service.go:82-93`）

### 2.5 Scope 查询构建器

**问题**: `user_repo.go:18-56` 和 `class_repo.go:18-56` 的 switch 构建 scope 过滤查询逻辑 90% 相同。

**设计**: 新建 `internal/repo/scope_query.go`:
```go
package repo

import (
    "gorm.io/gorm"
    "manage/internal/auth"
    "manage/internal/service/authz"
)

// ApplyScope applies scope filtering to a query based on actor scope.
func ApplyScope(query *gorm.DB, scope authz.Scope, modelType string) *gorm.DB {
    switch {
    case scope.IsAdmin:
        return query
    case scope.SelfUserID > 0:
        return query.Where("user_id = ?", scope.SelfUserID)
    case scope.ClassID > 0:
        return query.Where("class_id = ?", scope.ClassID)
    default:
        return query.Where("1 = 0")
    }
}
```

**迁移**: 修改 `user_repo.go` 和 `class_repo.go` 调用 `ApplyScope`。

---

## 阶段三：配置集中化（P2）

### 3.1 新建 `internal/config/config.go`

**问题**: 环境变量读取和硬编码常量分散在多个文件中。

**设计**:
```go
package config

import "os"

// Server configuration
var (
    JWTSecret     = getEnv("JWT_SECRET", "dev-secret-change-in-production")
    DatabaseDSN   = getEnv("DATABASE_DSN", "")
    WechatAppID   = getEnv("WECHAT_APP_ID", "")
    WechatAppSecret = getEnv("WECHAT_APP_SECRET", "")
)

// Upload directories
var (
    DocumentUploadDir  = getEnv("DOCUMENT_UPLOAD_DIR", "./data/uploads/documents")
    KnowledgeUploadDir = getEnv("KNOWLEDGE_UPLOAD_DIR", "./data/uploads/documents")
)

// Default class constants
const (
    DefaultDevClassID        = 10
    DefaultDevGrade          = "2020"
    DefaultDevMajor          = "信息管理"
    DefaultPublicClassID     = 9999
    DefaultPublicClassName   = "未绑定班级"
)

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
        }
    return fallback
}
```

### 3.2 迁移

- `router.go:21-24`、`file_handler.go:30-33`、`admin_knowledge_handler.go:35-41` → `config.DocumentUploadDir`
- `router.go:35` 和 `testutil/token.go:12` → `config.JWTSecret`
- `dev_login.go:14-16` 和 `wechat_handler.go:24-25` → `config.DefaultXxxClass`

### 3.3 默认班级创建合并

**问题**: `ensureDefaultDevClass` (dev_login.go:89-103) 和 `ensureDefaultPublicClass` (wechat_handler.go:487-501) 几乎相同。

**设计**: 合并为统一函数:
```go
func EnsureClass(db *gorm.DB, id uint64, name, grade, major string) error {
    var existing model.Class
    if err := db.Where("id = ?", id).First(&existing).Error; err == nil {
        return nil
    }
    return db.Create(&model.Class{
        ID:    id,
        Name:  name,
        Grade: grade,
        Major: major,
    }).Error
}
```

---

## 阶段四：风格统一（P3）

### 4.1 请求结构体统一

**问题**: 部分 handler 使用命名请求类型，部分使用匿名内联结构体。

**修复**: 全部改为命名类型:
- `notification_handler.go:37` → 定义 `sendNotificationReq`
- `me_handler.go:80` → 定义 `updateProfileReq`

### 4.2 Import 别名统一

**问题**: `jwtauth`（小写）vs `knowledgeSvc`（驼峰）。

**修复**: 全部使用小写字母:
- `knowledgeSvc` → `ksvc`

### 4.3 注释补充

为以下文件添加文档注释:
- `internal/model/user.go` — `User` 模型及字段说明
- `internal/model/class.go` — `Class` 模型及字段说明
- `internal/model/admin_log.go` — `AdminLog` 模型及字段说明
- `internal/service/auth/jwt.go` — JWT 服务包和方法说明
- `internal/service/wechat/service.go` — 微信服务包和方法说明

### 4.4 GORM Tag 统一

**问题**: `size:N` vs `type:varchar(N)` vs `type:text` 混用。

**修复**: 统一使用 `type:varchar(N)` 格式（更明确）。

---

## 验收标准

1. 所有测试通过 (`go test ./...`)
2. `go vet ./...` 无警告
3. 无重复的 `UpdateByID` 实现
4. 无重复的用户响应 JSON 构建
5. 所有列表接口返回 `total`
6. 所有 `ErrRecordNotFound` 比较使用 `errors.Is()`
7. 环境变量读取集中在 `config` 包
8. 所有公开类型有文档注释
