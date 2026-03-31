# 04 数据库设计 DDL（Phase 1）

> 本文为当前主版本，已从历史文档整合。

# 数据库设计 DDL 草案（Kingbase/PostgreSQL 兼容）

## 1. 说明

1. 以下为 Phase 1 核心表
2. 默认主键使用 `bigserial`
3. 时间字段统一 `timestamp`
4. 本文冻结的是 Phase 1 的表职责、核心字段和索引方向
5. 为兼容导入流程，部分外键先采用弱约束，延后在实现阶段补强

## 2. 枚举建议

- role_level: 1学生 / 2团支书团干部 / 3班主任教师 / 4超级管理员
- user_status: 在读/休学/复学/毕业
- import_job_status: pending / processing / success / partial_success / failed
- import_job_type: users / classes

## 3. 核心表 DDL

```sql
create table if not exists users (
  id bigserial primary key,
  student_id varchar(20) unique,
  name varchar(50) not null,
  openid varchar(100) unique,
  role int not null default 1,
  class_id bigint,
  grade varchar(10),
  major varchar(100),
  phone varchar(20),
  extra_attrs jsonb not null default '{}'::jsonb,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp
);

create table if not exists classes (
  id bigserial primary key,
  class_name varchar(50) not null,
  grade varchar(10) not null,
  major varchar(100) not null,
  counselor_user_id bigint,
  head_student_user_id bigint,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp
);

create table if not exists class_members (
  id bigserial primary key,
  class_id bigint not null,
  user_id bigint not null,
  joined_at timestamp not null default current_timestamp,
  left_at timestamp,
  is_active boolean not null default true,
  unique(class_id, user_id)
);

create table if not exists admins (
  id bigserial primary key,
  user_id bigint not null unique,
  admin_level int not null,
  managed_class_ids jsonb not null default '[]'::jsonb,
  managed_grade_scope jsonb not null default '[]'::jsonb,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  last_login_at timestamp
);

create table if not exists admin_logs (
  id bigserial primary key,
  admin_id bigint not null,
  action varchar(50) not null,
  target_type varchar(50) not null,
  target_id bigint,
  detail jsonb not null default '{}'::jsonb,
  ip_address varchar(50),
  created_at timestamp not null default current_timestamp
);

create table if not exists import_jobs (
  id bigserial primary key,
  job_id varchar(64) not null unique,
  job_type varchar(30) not null,
  status varchar(20) not null,
  total_rows int not null default 0,
  success_rows int not null default 0,
  failed_rows int not null default 0,
  error_report_path varchar(500),
  created_by bigint,
  detail jsonb not null default '{}'::jsonb,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp
);
```

## 4. 索引建议

```sql
create index if not exists idx_users_role on users(role);
create index if not exists idx_users_class_id on users(class_id);
create index if not exists idx_users_grade on users(grade);
create index if not exists idx_users_major on users(major);
create index if not exists idx_users_extra_attrs_gin on users using gin(extra_attrs);

create index if not exists idx_classes_grade_major on classes(grade, major);
create index if not exists idx_class_members_class_id on class_members(class_id);
create index if not exists idx_class_members_user_id on class_members(user_id);

create index if not exists idx_admin_logs_admin_id on admin_logs(admin_id);
create index if not exists idx_admin_logs_target on admin_logs(target_type, target_id);
create index if not exists idx_admin_logs_created_at on admin_logs(created_at);

create index if not exists idx_import_jobs_status on import_jobs(status);
create index if not exists idx_import_jobs_created_at on import_jobs(created_at);
```

## 5. 约束补充

1. `users.class_id` 与 `classes.id` 建议后续补 FK（导入阶段可先弱约束）
2. `class_members.class_id`、`class_members.user_id` 建议加 FK
3. 外键策略优先 `on update cascade on delete restrict`

## 6. 表职责说明

1. `users`
- 系统内用户主表。
- 存储学号、姓名、微信身份映射、角色、基础归属信息。
- `extra_attrs` 用于承载非高频固定字段，不替代核心字段。

2. `classes`
- 班级主表。
- 存储班级名称、年级、专业和班主任/学生负责人引用。

3. `class_members`
- 班级成员关系表。
- 用于记录用户与班级的有效关系、历史关系和绑定状态。

4. `admins`
- 管理能力映射表。
- 用于描述某个用户是否具备管理员能力、管理员等级和可管理范围。

5. `admin_logs`
- 管理操作审计表。
- 用于记录管理端关键写操作，不用于记录普通学生查询行为。

6. `import_jobs`
- 异步导入作业表。
- 用于记录导入任务状态、计数、错误报告位置和执行明细。

## 7. 字段口径补充

1. `users.role`
- 表示用户基础角色等级。
- 与 `admins.admin_level` 的口径必须保持一致，不允许互相矛盾。

2. `users.class_id`
- 表示当前生效班级。
- 若存在班级调整历史，以 `class_members` 中 `is_active=true` 的关系为准，`users.class_id` 作为当前归属冗余字段。

3. `admins.managed_class_ids`
- 当前使用 JSON 数组存储，表达管理员可管理的班级范围。
- 若后续范围管理复杂化，可拆为独立关系表，但 Phase 1 暂不扩展。

4. `admins.managed_grade_scope`
- 当前使用 JSON 数组存储，表达管理员可管理的年级范围。

5. `import_jobs.detail`
- 用于记录导入错误摘要、模板版本、执行批次等补充信息。
- 详细逐行错误可同时落在错误报告文件中。

## 8. 导入模板与数据策略

1. 用户导入至少包含：学号、姓名、角色、年级、专业、班级。
2. 班级导入至少包含：班级名称、年级、专业、班主任、负责人。
3. 导入采用“逐行校验 + 作业汇总”的方式，不要求整文件全成全败。
4. 可回滚的含义是：同一导入批次创建的数据可按作业维度撤销，不要求支持任意历史重算。
