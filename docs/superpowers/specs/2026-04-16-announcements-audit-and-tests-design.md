# Announcement 审计日志与测试补齐设计

**日期**：2026-04-16
**范围**：Announcement 模块增量修补

## 1. 目标

补齐 Announcement 模块在验收标准中缺失的两部分能力：

- 管理端创建、编辑、发布、归档操作写入 `admin_logs`
- handler / service / repo 三层测试覆盖补齐，确保公告发布链路与学生可见性规则可回归验证

## 2. 设计决策

### 2.1 审计日志

- 沿用现有 `audit.Logger` 的 best-effort 模式
- 在 `AnnouncementHandler` 中新增 `auditLogger`
- 仅在业务动作成功后写日志，失败请求不记日志
- 动作名固定为：
  - `announcements.create`
  - `announcements.patch`
  - `announcements.publish`
  - `announcements.archive`
- `target_type` 固定为 `announcement`
- `target_id` 使用公告主键 ID

### 2.2 测试策略

- handler 测试同时覆盖 HTTP 返回和审计日志落库
- service 测试聚焦 audience 命中规则、发布时间写入、状态机约束
- repo 测试聚焦 CRUD 与状态更新行为，不重复验证 service 层范围逻辑

## 3. 影响文件

- 修改：`internal/http/handler/announcement_handler.go`
- 修改：`internal/http/handler/announcement_handler_test.go`
- 修改：`internal/service/announcements/service_test.go`
- 新增：`internal/repo/announcement_repo_test.go`

## 4. 验收映射

- `Create/Patch/Publish/Archive` 成功后均写入 `admin_logs`
- 学生端列表只返回已发布且命中范围的公告
- 学生端详情访问未发布或不命中公告返回 `404`
- `Publish` 写入 `published_at`
- `Archive` 后再次 `Publish` 返回状态错误
- 全量通过：`go test ./... -count=1`
