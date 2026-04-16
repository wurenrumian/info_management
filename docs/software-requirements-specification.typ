#set document(
  title: "学院学生事务一站式系统软件需求规格说明书",
  author: "Codex",
)

#import "@preview/mmdr:0.2.1": mermaid

#set page(
  paper: "a4",
  margin: (top: 2.4cm, bottom: 2.4cm, left: 2.3cm, right: 2.3cm),
  numbering: "1",
  number-align: center,
  header: context {
    if counter(page).at(here()).at(0) > 1 {
      set text(size: 8.8pt, fill: rgb("#5E6B7A"))
      grid(
        columns: (1fr, 1fr),
        align(left)[学院学生事务一站式系统], align(right)[软件需求规格说明书],
      )
      v(-0.6em)
      line(length: 100%, stroke: 0.5pt + rgb("#5E6B7A"))
    }
  },
  footer: context {
    if counter(page).at(here()).at(0) > 1 {
      set text(size: 9pt, fill: rgb("#5E6B7A"))
      line(length: 100%, stroke: 0.5pt + rgb("#5E6B7A"))
      v(-0.4em)
      grid(
        columns: (1fr, 1fr),
        align(left)[版本：v1.0.0], align(right)[第 #counter(page).display() 页],
      )
    }
  },
)

#set text(
  font: ("Libertinus Serif", "SimSun", "SimHei", "Microsoft YaHei"),
  lang: "zh",
  size: 10.5pt,
  fallback: true,
)

#let brand = rgb("#365A7C")
#let accent = rgb("#EEF3F8")
#let border = rgb("#A7B8CA")

#set par(
  first-line-indent: 2em,
  justify: true,
  leading: 0.7em,
)

#show heading: it => block(above: 1.4em, below: 1em)[
  #let size = if it.level == 1 { 16pt } else if it.level == 2 { 13pt } else { 11pt }
  #set text(size: size, weight: "bold", fill: if it.level == 1 { brand } else { black })
  #if it.level == 1 {
    v(0.5em)
    it
    v(0.2em)
    line(length: 100%, stroke: 1.5pt + brand)
  } else {
    it
  }
]

#let cover-line = line(length: 100%, stroke: 0.8pt + brand)

#let card(body) = block(
  inset: 10pt,
  stroke: 0.7pt + border,
  radius: 6pt,
  fill: white,
)[#body]

#let section-note(title, body) = block(
  inset: 11pt,
  stroke: 0.7pt + border,
  radius: 6pt,
  fill: accent,
  above: 0.5em,
  below: 0.8em,
)[
  #set text(weight: "bold", fill: brand)
  #title
  #v(4pt)
  #set text(weight: "regular", fill: black)
  #body
]

#let small-title(body) = text(weight: "bold", fill: brand, body)

#let matrix-table(headers, rows) = table(
  columns: (22%, 78%),
  stroke: 0.5pt + border,
  inset: 8pt,
  fill: (x, y) => if y == 0 { brand.lighten(90%) } else { white },
  table.header(..headers.map(h => text(weight: "bold", fill: brand, h))),
  ..rows.flatten(),
)

#let tri-table(headers, rows) = table(
  columns: (24%, 18%, 58%),
  stroke: 0.5pt + border,
  inset: 8pt,
  fill: (x, y) => if y == 0 { brand.lighten(90%) } else { white },
  table.header(..headers.map(h => text(weight: "bold", fill: brand, h))),
  ..rows.flatten(),
)

#let quad-table(headers, rows) = table(
  columns: (17%, 21%, 21%, 41%),
  stroke: 0.5pt + border,
  inset: 8pt,
  fill: (x, y) => if y == 0 { brand.lighten(90%) } else { white },
  table.header(..headers.map(h => text(weight: "bold", fill: brand, h))),
  ..rows.flatten(),
)

#let flow-box(title, items) = block(
  width: 100%,
  inset: 10pt,
  stroke: 0.7pt + border,
  radius: 6pt,
  fill: white,
)[
  #set text(weight: "bold", fill: brand)
  #title
  #v(5pt)
  #set text(weight: "regular", fill: black)
  #items
]

#align(center)[
  #v(2.6cm)
  #set text(size: 22pt, weight: "bold", fill: brand)
  学院学生事务一站式系统
  #v(8pt)
  软件需求规格说明书

  #v(1.2cm)
  #cover-line
  #v(0.6cm)

  #set text(size: 11.5pt)
  面向老师评审的正式文档版本

  #v(4.8cm)
  #card[
    #set align(center)
    #set text(size: 11pt)
    项目名称：学院学生事务一站式系统 \
    文档类型：软件需求规格说明书（SRS） \
    适用对象：评审教师、开发人员、测试人员 \
    生成日期：2026-04-16
  ]
]

#pagebreak()

#outline(title: [目录])

#pagebreak()

= 1. 引言

== 1.1 编写目标

本文档用于对“学院学生事务一站式系统”进行系统级软件需求说明，面向老师评审、项目指导、开发实现与测试验收等场景，作为需求沟通、范围控制和后续设计实现的重要依据。

本文档编写遵循以下原则：

- 图文并茂，采用自然语言描述与需求模型结合的方式表达需求。
- 完整表述，覆盖功能性需求与非功能性需求。
- 共同参与，吸收原始需求、项目技术方案、现有 API 文档与设计文档中的共识内容。
- 语言简练，便于评审、阅读和理解。
- 前后一致，对同一需求使用统一术语、统一边界和统一约束。

== 1.2 读者对象

- 项目评审教师与课程答辩评委。
- 需求分析与系统设计人员。
- 后端、前端与测试开发人员。
- 后续参与项目维护的团队成员。

== 1.3 文档概述

本文档首先说明系统背景、目标用户、建设范围与约束条件，再对系统功能性需求、非功能性需求、界面需求、接口定义、交付与验收要求进行完整描述。其中，功能需求部分结合用例模型、流程模型与分析模型进行说明。

#section-note(
  [文档使用说明],
  [
    本文档中的“应”表示必须满足的需求，“可”表示在资源允许或后续迭代中可扩展的能力；凡涉及角色、班级、年级等访问边界时，均默认受统一权限与 Scope 机制约束。
  ],
)

== 1.4 术语定义

#matrix-table(
  ("术语", "定义"),
  (
    ([学生事务一站式系统], [面向学院学生事务管理与服务的一体化信息系统]),
    ([学生端], [面向学生用户的微信小程序端]),
    ([管理端], [面向老师、助教、团干部等管理角色的后台端]),
    ([JWT], [用于接口认证的 JSON Web Token]),
    ([RBAC], [基于角色的访问控制]),
    ([Scope], [基于班级、年级等组织边界的数据访问范围控制]),
    ([知识库], [存放政策问答、办事指引、模板附件等内容的业务模块]),
    ([党团流程], [入党、入团等线性事务流程及其阶段追踪]),
    ([审批流程], [请假申请、盖章申请、证明申请等需要审核的流程]),
    ([订阅通知], [基于微信小程序订阅消息的通知能力]),
    ([Kingbase], [人大金仓数据库，兼容 PostgreSQL]),
  ),
)

== 1.5 参考文献

- README.md
- docs/development-standard.md
- docs/ONBOARDING.md
- original_request/初始需求.md
- original_request/技术方案.md
- docs/api/phase1-foundation-api.md
- docs/api/phase2-files-api.md
- docs/api/phase2-knowledge-api.md
- docs/api/phase2-wechat-api.md
- docs/api/notification-api.md
- docs/superpowers/specs/2026-04-08-partyflow-design.md

= 2. 软件系统概述

== 2.1 软件产品概述

学院学生事务一站式系统面向学院日常学生管理与服务场景，旨在解决通知分散、流程不清、资料难找、事务办理依赖人工转发等问题。系统通过统一身份体系、统一权限控制、统一文件能力和统一事务入口，将学生高频事务整合到同一平台中。

系统总体定位如下：

- 面向学生提供统一查询、办理和接收通知的入口。
- 面向管理员提供内容维护、流程管理、通知发布和审批处理能力。
- 面向课程项目评审提供具有实际业务场景和可实现边界的系统方案。

#section-note(
  [系统分层图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C', 'secondaryColor': '#FFFFFF', 'tertiaryColor': '#FFFFFF' }}}%%
    flowchart TD
    subgraph PresentationLayer[表现层]
        P1[学生端微信小程序]
        P2[管理端后台]
    end
    subgraph BusinessLayer[业务层]
        B1[用户权限]
        B2[知识库]
        B3[文件服务]
        B4[党团流程]
        B5[通知服务]
        B6[审批与证明]
    end
    subgraph SupportLayer[支撑层]
        S1[JWT 认证]
        S2[RBAC]
        S3[Scope]
        S4[审计日志]
        S5[文件抽取]
        S6[消息发送]
    end
    subgraph DataLayer[数据层]
        D1[(Kingbase / PostgreSQL)]
        D2[(文件存储)]
        D3[(模板资源)]
    end
    P1 --> B1
    P2 --> B1
    B1 --> S1
    B2 --> S2
    B3 --> S3
    B4 --> S4
    B5 --> S5
    B6 --> S6
    S1 --> D1
    S2 --> D1
    S3 --> D1
    S4 --> D1
    S5 --> D2
    S6 --> D3",
    )
  ],
)

#section-note(
  [系统上下文图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart LR
    Student[学生] --> MiniApp[学生端微信小程序]
    Admin[老师/助教/团干部] --> AdminWeb[管理端后台]
    MiniApp --> System[学生事务一站式系统]
    AdminWeb --> System
    System --> WeChat[微信平台]
    System --> DB[(Kingbase/PostgreSQL)]
    System --> FileStore[(文件存储)]
    System --> Docs[政策文件/模板资料]",
    )
  ],
)

== 2.2 用户特征

1. 学生用户。主要使用移动端完成身份登录、知识检索、党团进度查看、通知接收、证明申请和审批进度查询等操作。该类用户数量较大，但单个用户操作相对简单，强调易用性和反馈及时性。

2. 团干部或班级学生干部。在本班或本组织范围内承担部分事务发布、流程维护、模板化通知发送等管理职责。该类用户需要受权限范围约束。

3. 教师、班主任、辅导员、助教。负责学生信息维护、班级管理、审核处理、知识库维护与通知管理等工作。此类用户对数据准确性、审计留痕和范围控制要求较高。

4. 超级管理员。负责系统基础配置、全局数据管理、重要内容维护和异常处理，拥有全局范围权限。

#tri-table(
  ("用户类别", "主要终端", "典型关注点"),
  (
    ([学生], [微信小程序], [使用门槛低、办事入口集中、检索结果准确、状态反馈及时]),
    ([团干部 / 学生干部], [管理端后台], [班级范围内事务处理、通知发布、流程维护、操作便捷]),
    ([教师 / 助教], [管理端后台], [数据准确、审计留痕、权限边界清晰、批量维护效率]),
    ([超级管理员], [管理端后台], [系统可控、配置统一、全局审计、异常可追踪]),
  ),
)

== 2.3 设计和实现约束

- 后端技术栈使用 Go、Gin、GORM。
- 数据库使用 Kingbase 或 PostgreSQL 兼容数据库。
- 测试环境使用 SQLite in-memory。
- 认证方式采用 JWT，并结合微信登录与绑定能力。
- 所有业务接口统一使用 `/api/v1` 前缀。
- 成功响应统一为 `{"data": ...}`，失败响应统一为 `{"error": "..."}`。
- 文件上传、下载与管理由统一文件服务提供，业务模块不得自行实现文件落盘逻辑。
- 业务规则不得写入 HTTP handler，必须下沉至 service 层。
- 管理类数据查询必须经过权限校验与 Scope 过滤。
- 系统主要依赖已有学院资料和管理员录入，不依赖校级平台直接开放接口。

== 2.4 假设与依赖

- 学号、姓名、班级、年级等基础身份数据可以由管理员预置或通过公开注册逐步补齐。
- 学生侧主要通过微信小程序访问系统。
- 管理员可通过后台界面进行维护，必要时可批量导入结构化数据。
- 学院常用政策文件、办事模板和事务规则可由管理员整理后录入系统。
- 微信平台提供登录与订阅通知相关能力。
- 本期建设不纳入学业情况分析与预警功能。

#matrix-table(
  ("范围边界", "说明"),
  (
    ([纳入范围], [统一身份认证、知识库与模板、文件服务、通知公告、党团流程、审批与电子证明等学生事务核心能力]),
    ([不纳入范围], [学业情况分析与预警、校级平台深度实时互通、复杂工作流引擎、多渠道消息平台等高复杂或高依赖能力]),
  ),
)

= 3. 功能性需求描述

== 3.1 软件功能概述

系统围绕“统一身份、统一内容、统一流程、统一通知”展开，核心功能包括：

- 用户与权限管理
- 微信登录与账号绑定
- 学生信息与班级信息管理
- 知识库与智能问答
- 文件与模板管理
- 党团事务流程管理
- 通知公告与精准推送
- 电子证明与审批流程

#quad-table(
  ("功能域", "服务对象", "核心目标", "主要产出"),
  (
    ([用户与权限], [学生、管理员], [建立可信身份与访问边界], [登录态、角色权限、范围控制结果]),
    ([知识与模板], [学生、管理员], [实现政策咨询与材料获取], [问答结果、附件、模板下载]),
    ([流程与审批], [学生、管理员], [实现事务状态可见与办理可跟踪], [流程进度、审批状态、证明结果]),
    ([通知与公告], [学生、管理员], [提高信息触达效率], [公告内容、订阅消息、发送记录]),
  ),
)

#section-note(
  [功能分解图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#365A7C', 'primaryTextColor': '#FFFFFF', 'lineColor': '#365A7C' }}}%%
    mindmap
      root((学生事务一站式系统))
        用户与权限
          微信登录
          JWT认证
          RBAC
          Scope控制
        信息管理
          学生信息
          班级信息
          审计日志
        知识服务
          政策问答
          文档检索
          模板下载
        事务流程
          党团流程
          电子审批
          证明申请
        通知服务
          公告发布
          标签分发
          订阅消息
        文件服务
          上传
          下载
          附件引用",
    )
  ],
)

== 3.2 软件需求的用例模型

系统总体用例关系可归纳如下：

#tri-table(
  ("角色", "场景类型", "主要用例"),
  (
    (
      [学生],
      [使用],
      [微信登录 / 注册绑定；查看和维护个人信息；搜索知识库与下载模板；查看党团进度；提交审批或证明申请；接收通知与查看未读数],
    ),
    ([团干部 / 学生干部], [管理], [维护知识库内容；发布通知公告；维护党团流程；处理授权范围内的审批事项]),
    ([教师 / 助教], [管理], [维护班级与学生信息；维护知识库内容；发布通知公告；维护党团流程；审批申请事项]),
    ([超级管理员], [全局管理], [系统配置与审计；全局内容维护；高权限管理操作]),
  ),
)

#section-note(
  [角色权限关系图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart TD
    R1[普通学生]
    R2[团干部/学生干部]
    R3[教师/助教]
    R4[超级管理员]
    P1[个人信息查看与维护]
    P2[知识检索与模板下载]
    P3[本人流程与申请查看]
    P4[班级范围内事务管理]
    P5[年级/范围内内容维护]
    P6[审批处理]
    P7[系统全局管理]
    R1 --> P1
    R1 --> P2
    R1 --> P3
    R2 --> P4
    R3 --> P5
    R3 --> P6
    R4 --> P7",
    )
  ],
)

#section-note(
  [核心业务闭环图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart LR
    A[身份建立] --> B[内容获取]
    B --> C[事务办理]
    C --> D[通知反馈]
    D --> A",
    )
  ],
)

== 3.3 软件需求的分析模型

=== 3.3.1 领域分析模型

#section-note(
  [领域对象关系图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    classDiagram
    class User {
      +id
      +student_id
      +name
      +role
      +class_id
      +grade
      +major
    }
    class Class {
      +id
      +class_name
      +grade
      +major
    }
    class KnowledgeItem {
      +id
      +question
      +answer
      +keywords
    }
    class DocumentFile {
      +id
      +title
      +file_path
      +content_type
    }
    class PartyProgress {
      +id
      +flow_type
      +current_stage
      +stage_started_at
    }
    class Approval {
      +id
      +type
      +status
      +current_approver_id
    }
    class Announcement {
      +id
      +title
      +tags
      +published_at
    }
    User --> Class : belongs to
    KnowledgeItem --> DocumentFile : attaches
    Announcement --> DocumentFile : attaches
    PartyProgress --> User : tracks
    Approval --> User : applicant
    Announcement --> User : author",
    )
  ],
)

=== 3.3.2 当前项目映射说明

#matrix-table(
  ("需求模块", "当前仓库状态"),
  (
    ([用户、权限、班级、个人主页], [已有实现与 API 文档]),
    ([微信登录、绑定、公开注册], [已有实现与 API 文档]),
    ([文件服务], [已有实现与 API 文档]),
    ([知识库检索与管理], [已有实现与 API 文档]),
    ([通知订阅与日志], [已有实现与 API 文档]),
    ([党团流程], [已有设计文档，属于已规划范围]),
    ([审批流程], [已有 API 规划文档，属于已规划范围]),
    ([公告与精准推送], [已有 API 规划文档，属于已规划范围]),
  ),
)

以下小节对各功能需求进行自然语言与模型结合的详细描述。

== 3.4 用户与权限管理需求

=== 3.4.1 功能目标

系统应建立统一的用户身份、角色权限与数据范围控制机制，保证学生与管理员在各自职责边界内访问和操作数据。

=== 3.4.2 需求描述

- 系统应支持学生、团干部、教师、超级管理员四类基本角色。
- 系统应支持 JWT 认证，并在登录后为用户签发包含 `sub`、`role`、`class_id`、`grade` 等信息的身份令牌。
- 系统应支持基于角色的权限控制。
- 系统应支持基于班级、年级等组织边界的数据范围控制。
- 系统应支持学生查看和维护本人的非敏感个人资料。
- 系统应支持管理员按权限维护学生信息、班级信息和管理日志。
- 系统应记录关键管理操作的审计日志，便于追踪责任。

#quad-table(
  ("需求要素", "输入", "处理", "输出"),
  (
    ([身份认证], [微信 code、JWT、学号等身份信息], [校验身份、签发或解析 token], [用户身份上下文]),
    ([权限控制], [角色、班级、年级、动作类型], [RBAC + Scope 计算], [允许或拒绝访问]),
    ([资料维护], [个人资料变更请求], [字段校验与权限判断], [更新后的用户资料]),
  ),
)

=== 3.4.3 业务规则

- 学生只能查看和修改允许编辑的本人资料。
- 管理员不得绕过 Scope 直接查看不属于自身管理范围的数据。
- `classes.grade` 为年级事实来源，用户年级快照由系统维护，不由普通业务接口直接修改。

=== 3.4.4 主要用例

1. 学生登录后查看个人信息并修改昵称、头像、简介等资料。
2. 教师查询本班或本年级学生列表并查看详细信息。
3. 超级管理员维护班级信息并查询审计日志。

== 3.5 微信登录与账号绑定需求

=== 3.5.1 功能目标

系统应提供面向微信小程序场景的登录、注册和绑定机制，降低学生首次使用门槛，并形成稳定身份映射。

=== 3.5.2 需求描述

- 系统应支持学生通过微信授权码进行登录。
- 系统应支持未绑定用户进行公开注册或激活。
- 系统应支持已登录用户绑定微信账号。
- 系统应支持必要场景下通过学号与密码完成绑定校验。
- 系统应在首次注册时自动创建学生账号，并将其挂接到默认班级，便于后续数据补全。

=== 3.5.3 登录绑定流程

#section-note(
  [登录绑定流程图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart TD\nA[学生发起微信授权] --> B[微信返回 code]\nB --> C[系统调用 code2Session]\nC --> D[获取 openid]\nD --> E{是否已绑定系统账号}\nE -- 是 --> F[返回 token 与用户信息]\nE -- 否 --> G[提示公开注册或绑定]\nG --> H[完成绑定后进入系统]",
    )
  ],
)

#section-note(
  [学生注册 / 登录业务流程图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    sequenceDiagram
    participant U as 学生
    participant W as 微信平台
    participant S as 系统
    participant D as 用户数据库
    U->>W: 发起授权登录
    W-->>U: 返回 code
    U->>S: 提交 code
    S->>W: code2Session
    W-->>S: openid
    S->>D: 查询 openid 绑定关系
    alt 已绑定
        S-->>U: 返回 token 与用户信息
    else 未绑定
        S-->>U: 提示先绑定或公开注册
    end",
    )
  ],
)

=== 3.5.4 业务规则

- 学生学号和姓名不匹配时，系统应拒绝公开注册。
- 一个微信账号只能绑定一个系统用户。
- 认证失败时系统应返回明确错误信息，不得生成伪造身份。

#matrix-table(
  ("场景", "系统行为"),
  (
    ([首次使用且未绑定], [允许公开注册或绑定后进入系统，并建立默认身份映射]),
    ([已绑定用户登录], [直接返回 token 与用户信息]),
    ([绑定冲突], [拒绝绑定并返回明确错误信息]),
  ),
)

== 3.6 知识库与智能问答需求

=== 3.6.1 功能目标

系统应为学生提供高频政策咨询、流程指引和模板资料查询能力，降低重复答疑成本，提高信息获取效率。

=== 3.6.2 需求描述

- 系统应支持学生按关键词检索知识库内容。
- 系统应支持查看问答详情、附件与相关模板。
- 系统应支持管理员维护问答内容、关键词与关联附件。
- 系统应支持基于文档正文抽取结果进行检索。
- 系统应优先基于标准答案和官方附件进行回复，降低生成式回答带来的不确定性。
- 系统可支持 AI 辅助生成问答草稿，但草稿必须经管理员确认后才能入库。

#quad-table(
  ("需求要素", "输入", "处理", "输出"),
  (
    ([学生检索], [关键词、分页参数], [全文检索与回退检索], [问答列表、总数、附件摘要]),
    ([详情查看], [问答 ID], [加载正文、关键词与附件], [标准答案与相关资料]),
    ([管理维护], [问答内容、关键词、附件], [校验、保存、更新索引], [可发布知识条目]),
    ([草稿生成], [文件、目标条数范围], [抽取文本并生成草稿], [待审核问答草稿]),
  ),
)

=== 3.6.3 检索逻辑模型

#section-note(
  [知识检索逻辑图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart TD\nA[学生输入关键词] --> B[执行全文检索]\nB --> C{是否命中}\nC -- 是 --> D[返回问答列表]\nC -- 否 --> E[执行分词与 LIKE 回退检索]\nE --> F{是否命中}\nF -- 是 --> D\nF -- 否 --> G[提示换词或查看相关分类]\nD --> H[展示标准答案、附件与模板]",
    )
  ],
)

#section-note(
  [管理员维护知识库流程图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart LR
    A[管理员上传政策文件或模板] --> B[系统抽取文本并保存文件元数据]
    B --> C[管理员录入或生成问答草稿]
    C --> D[审核确认后入库发布]",
    )
  ],
)

=== 3.6.4 业务规则

- 敏感信息不得由系统进行不受控生成，应优先展示标准答案、办事说明和官方附件。
- 学生侧只允许检索已发布或可访问的知识内容。
- 删除类知识库操作应由较高权限角色执行。

=== 3.6.5 主要用例

1. 学生搜索“休学申请怎么办理”，系统返回标准答案和申请表附件。
2. 管理员录入“奖学金申请材料”问答，并绑定相关 PDF、Word 模板。
3. 高权限管理员上传文件并生成问答草稿预览，审核后批量入库。

#section-note(
  [知识库数据流图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart LR
    A[政策文件/模板] --> B[管理员维护知识项]
    B --> C[学生检索并查看结果]",
    )
  ],
)

== 3.7 文件与模板管理需求

=== 3.7.1 功能目标

系统应提供统一文件上传、下载、检索和附件引用能力，为知识库、公告、证明与审批等模块提供基础支撑。

=== 3.7.2 需求描述

- 系统应支持登录用户上传规定类型的文件。
- 系统应支持文件元数据查询、文件下载与标题检索。
- 系统应支持基于文档正文抽取文本进行检索。
- 系统应支持按业务场景分类存储文件。
- 各业务模块应通过 `file_id` 引用统一文件，不得重复实现文件管理。

#quad-table(
  ("文件场景", "输入", "处理", "输出"),
  (
    ([文件上传], [文件二进制、scene 参数], [类型校验、大小校验、存储、抽取文本], [file_id 与文件元数据]),
    ([文件检索], [关键词], [按标题与正文检索], [文件列表与摘要片段]),
    ([文件引用], [业务表单中的 file_id], [建立关联关系], [模块可用附件引用]),
  ),
)

=== 3.7.3 文件服务关系模型

#section-note(
  [文件服务关系图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart LR
    Upload[统一文件服务]
    Know[知识库模块]
    Ann[公告模块]
    Approval[审批模块]
    Cert[证明模块]
    Know --> Upload
    Ann --> Upload
    Approval --> Upload
    Cert --> Upload",
    )
  ],
)

#section-note(
  [统一文件处理流程图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart LR
    A[用户上传文件] --> B[系统校验类型与大小]
    B --> C[按场景存储并生成 file_id]
    C --> D[业务模块引用或用户下载]",
    )
  ],
)

=== 3.7.4 业务规则

- 允许上传的文件类型应受限制。
- 单个文件大小应受限制。
- 删除文件应仅允许高权限角色执行。
- 文件删除后应保留管理操作日志。

== 3.8 党团事务流程管理需求

=== 3.8.1 功能目标

系统应支持对入团、入党等线性事务流程进行可视化展示、阶段追踪、历史留痕和关键节点提醒。

=== 3.8.2 需求描述

- 系统应支持学生查看本人在入团或入党流程中的当前阶段、阶段起始时间与下一步提示。
- 系统应支持管理员在权限范围内查询、创建、更新和批量导入学生流程进度。
- 系统应对每次阶段变化记录事件日志。
- 系统应对明确规则的关键节点生成提醒任务，并结合通知能力发送提醒。

#quad-table(
  ("需求要素", "输入", "处理", "输出"),
  (
    ([进度查询], [学生身份或管理员筛选条件], [按 flow_type 与 Scope 查询], [当前阶段、历史节点、下一步提示]),
    ([阶段变更], [学生标识、目标阶段、备注], [校验权限、写入当前状态与事件], [更新后的流程进度]),
    ([批量导入], [结构化导入数据], [解析、校验、去重、更新], [导入结果与错误反馈]),
    ([规则提醒], [阶段起始时间、规则编码], [定时扫描、去重判断、消息发送], [提醒记录与事件日志]),
  ),
)

=== 3.8.3 流程模型

#section-note(
  [入党流程阶段图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    stateDiagram-v2\n    [*] --> 申请人\n    申请人 --> 积极分子\n    积极分子 --> 发展对象\n    发展对象 --> 预备党员\n    预备党员 --> 正式党员",
    )
  ],
)

说明：

- 入党流程采用固定阶段编码。
- 入团流程为较简化的线性流程。
- 本期不支持复杂自定义状态机配置。

=== 3.8.4 提醒逻辑

#section-note(
  [提醒逻辑图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart TD
    A[定时任务扫描流程数据] --> B{是否满足提醒规则}
    B -- 否 --> C[结束]
    B -- 是 --> D{是否已发送同规则提醒}
    D -- 是 --> C
    D -- 否 --> E[生成提醒记录]
    E --> F[调用通知服务发送订阅消息]
    F --> G[写入流程事件日志]",
    )
  ],
)

#section-note(
  [党团流程维护流程图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart LR
    A[管理员录入或导入学生阶段] --> B[系统校验 Scope 与阶段合法性]
    B --> C[更新当前进度并记录事件]
    C --> D[学生查看阶段与下一步提示]",
    )
  ],
)

=== 3.8.5 业务规则

- 学生只能查看本人的党团流程。
- 管理员只能在 Scope 范围内维护流程数据。
- 阶段变化应自动留痕。
- 本期不纳入理论自测与复杂提醒编排。

#section-note(
  [党团流程职责分栏图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart LR
    subgraph Student[学生]
        S1[查询本人流程]
    end
    subgraph Admin[管理员]
        A1[录入或更新流程节点]
    end
    subgraph System[系统定时任务]
        T1[到达规则触发时间]
    end
    S1 -->|读取当前状态与历史事件| Student
    A1 -->|校验权限、保存变化并留痕| Admin
    T1 -->|扫描提醒条件、发送通知| System",
    )
  ],
)

== 3.9 通知公告与精准推送需求

=== 3.9.1 功能目标

系统应支持管理员发布公告、维护通知模板，并按用户标签或范围进行信息推送，提高事务通知的触达效率。

=== 3.9.2 需求描述

- 系统应支持管理员创建通知模板。
- 系统应支持查询发送记录、未读数量和订阅状态。
- 系统应支持学生上报订阅授权结果。
- 系统应支持管理员发布带标签、带附件的公告。
- 系统应支持按年级、专业、班级或角色范围进行目标分发。

#quad-table(
  ("需求要素", "输入", "处理", "输出"),
  (
    ([公告发布], [标题、正文、标签、目标范围], [校验内容并保存公告], [可浏览的公告内容]),
    ([模板管理], [模板编码、微信模板 ID、名称], [保存模板定义], [可复用通知模板]),
    ([订阅上报], [用户授权结果], [更新订阅状态与可用次数], [发送资格状态]),
    ([消息发送], [目标用户集、模板数据], [筛选用户、调用微信接口、记录日志], [发送结果与未读统计]),
  ),
)

=== 3.9.3 通知逻辑模型

#section-note(
  [通知分发逻辑图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart TD\nA[管理员创建公告或通知模板] --> B[配置标签与目标范围]\nB --> C[系统筛选目标用户]\nC --> D{用户是否具备可用订阅次数}\nD -- 是 --> E[发送微信订阅消息]\nD -- 否 --> F[保留站内可读记录]\nE --> G[记录发送日志]\nF --> G",
    )
  ],
)

#section-note(
  [公告发布与通知发送流程图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart LR
    A[管理员创建公告或模板] --> B[配置标签与目标范围]
    B --> C[系统筛选目标用户并判断订阅资格]
    C --> D[发送消息并记录日志]",
    )
  ],
)

=== 3.9.4 业务规则

- 通知模板维护仅允许管理角色执行。
- 发送记录应可追踪发送状态、错误信息和发送时间。
- 未读数可基于消息状态进行统计。
- 本期不强制实现短信、邮件等多渠道通知。

== 3.10 电子证明与审批流程需求

=== 3.10.1 功能目标

系统应支持学生在线发起事务申请，支持管理员在线审批，并为电子证明生成与状态跟踪提供基础能力。

=== 3.10.2 需求描述

- 系统应支持学生提交标准化申请，如请假申请、盖章申请、证明申请等。
- 系统应支持管理员查看待办事项并执行通过、驳回等审批操作。
- 系统应支持记录审批历史、当前审批人和最终状态。
- 系统应支持调用学生基础信息填充证明模板，并生成 PDF 预览或结果文件。
- 系统应支持学生查看本人申请状态和处理结果。

#quad-table(
  ("需求要素", "输入", "处理", "输出"),
  (
    ([申请提交], [审批类型、表单数据、附件], [表单校验、生成审批单、确定当前审批人], [待处理审批单]),
    ([审批处理], [动作、意见、操作者], [状态流转、记录动作历史], [审批结论与更新时间]),
    ([申请撤回], [申请人身份、申请状态], [校验是否可撤回并更新状态], [withdrawn 状态结果]),
    ([证明生成], [模板、学生基础信息、审批结果], [填充模板并输出 PDF], [可下载或预览的证明文件]),
  ),
)

=== 3.10.3 审批流程模型

#section-note(
  [审批流程图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    sequenceDiagram\n    participant U as 学生\n    participant S as 系统\n    participant A as 审批人\n    U->>S: 提交审批申请与附件\n    S->>S: 校验表单并生成审批单\n    S-->>A: 分配待办事项\n    A->>S: 审批通过或驳回\n    S-->>U: 更新审批状态与处理结果\n    S->>S: 记录审批历史",
    )
  ],
)

#section-note(
  [审批办理流程图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart TD
    subgraph Student[学生]
        S1[填写表单并上传附件]
        S2[提交申请]
        S3[查看申请状态]
        S4[获取最终结果或撤回结果]
    end
    subgraph System[系统]
        SY1[校验表单并创建审批单]
        SY2[确定当前审批人并写入 pending]
        SY3[更新状态并生成证明文件]
    end
    subgraph Approver[审批人/管理员]
        A1[在待办列表中接收申请]
        A2[执行通过或驳回并填写意见]
        A3[完成流程处理]
    end
    S1 --> S2
    S2 --> SY1
    SY1 --> SY2
    SY2 --> A1
    A1 --> A2
    A2 --> SY3
    SY3 --> S3
    S3 --> S4
    SY3 --> A3",
    )
  ],
)

=== 3.10.4 业务规则

- 审批流程至少应支持 `pending`、`approved`、`rejected`、`withdrawn` 等状态。
- 审批动作应有操作者、时间和意见留痕。
- 长时间未处理的申请可作为后续提醒场景，但本期不强制要求复杂催办机制。
- 电子证明生成依赖标准模板与学生基础信息，不依赖外部教务系统数据。

== 3.11 需求范围边界

为保证系统需求清晰，本期明确不纳入以下内容：

- 学业情况分析与预警。
- 与校级平台的深度实时数据互通。
- 自由配置型复杂工作流引擎。
- 全渠道消息触达平台。
- 高风险、不可解释的自动化决策功能。

= 4. 非功能性需求

== 4.1 安全性需求

- 系统应使用认证机制保护受限接口。
- 系统应基于角色和 Scope 控制数据访问。
- 系统应避免敏感个人信息在前端或日志中明文暴露。
- 系统应对关键管理操作留存审计日志。
- 系统应通过环境变量注入数据库连接、JWT 密钥等敏感配置，禁止硬编码。

#tri-table(
  ("非功能属性", "目标方向", "说明"),
  (
    ([安全性], [保护数据和接口], [通过认证、权限、范围控制和审计能力保障业务安全]),
    ([性能], [满足低至中并发场景], [保证常用接口、检索接口和文件服务的可接受响应时间]),
    ([可靠性], [保证状态一致与可追踪], [关键业务需具备状态记录、异常追踪和必要的重试能力]),
    ([可维护性], [支持多人协作开发], [通过统一目录规范、模块边界和共享服务提高可维护性]),
    ([可测试性], [保证变更可验证], [要求覆盖参数、权限、范围和核心业务路径测试]),
    ([可用性], [便于学生和管理员快速使用], [降低学习成本，提供明确错误提示与状态反馈]),
  ),
)

== 4.2 性能需求

- 系统应满足学院场景下的中低并发访问需求。
- 常见查询接口应在合理时间内返回结果。
- 文件上传与知识检索应在可接受时间内完成。
- 对于较大文档的文本抽取和问答草稿生成，可采用异步或流式方式改善体验。

#matrix-table(
  ("性能关注点", "需求说明"),
  (
    ([普通查询接口], [应支持日常学生与管理员访问场景下的稳定响应，不因单次普通查询造成明显阻塞]),
    ([知识检索接口], [应优先优化高频关键词检索体验，并允许在无命中时采用回退策略以保证结果可用性]),
    ([文件处理场景], [上传、抽取文本与生成草稿可采用分阶段处理，以避免长时间阻塞用户操作]),
  ),
)

== 4.3 可靠性需求

- 系统应保证核心业务数据持久化存储。
- 系统出现局部模块异常时，不应影响其他基础能力的可用性。
- 对于审批、流程、通知发送等关键业务，应记录状态和历史，便于重试与追踪。

== 4.4 可维护性需求

- 系统应采用模块化分层架构。
- 业务规则应集中在 service 层，接口层保持轻量。
- 文件、通知、权限等共性能力应作为共享服务复用。
- 新模块应遵循统一目录、接口与响应规范，便于团队协作扩展。

== 4.5 可测试性需求

- 每个模块应具备至少基本的 handler 测试与 repo 或 service 测试。
- 权限、参数校验和 Scope 行为应纳入测试覆盖。
- 合并前应通过完整自动化测试。

== 4.6 可用性需求

- 学生端操作流程应简洁，减少复杂配置和学习成本。
- 错误提示应明确，便于用户理解下一步动作。
- 知识检索和党团进度等高频功能应尽量减少操作层级。

= 5. 界面需求

== 5.1 学生端界面需求

学生端以微信小程序为主要载体，界面应满足以下要求：

- 首页应集中展示个人常用入口、通知摘要和高频事项。
- 知识库页面应支持关键词搜索、结果列表和详情查看。
- 党团流程页面应直观显示当前阶段、历史节点和下一步提示。
- 审批与证明页面应支持表单填写、附件上传、状态查看。
- 通知页面应支持未读提示和内容查看。

#section-note(
  [学生端信息架构图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart LR
    subgraph StudentApp[学生端微信小程序]
        H[首页]
        K[知识库]
        P[党团流程]
        A[审批/证明]
        N[通知中心]
    end
    H --> K
    H --> P
    H --> A
    H --> N",
    )
  ],
)

== 5.2 管理端界面需求

管理端界面应满足以下要求：

- 支持学生、班级、知识库、通知、流程、审批等模块化导航。
- 支持分页查询、条件筛选和详情查看。
- 支持结构化表单录入与附件绑定。
- 支持对批量导入、审批处理和发送记录进行清晰反馈。

#section-note(
  [管理端导航结构图],
  [
    #mermaid(
      "%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#EEF3F8', 'primaryTextColor': '#365A7C', 'primaryBorderColor': '#A7B8CA', 'lineColor': '#365A7C' }}}%%
    flowchart LR
    subgraph AdminPortal[管理端后台]
        U[用户与班级]
        K[知识与文件]
        A[公告与通知]
        P[流程与审批]
    end",
    )
  ],
)

== 5.3 界面一致性要求

- 相同概念在不同页面使用统一术语。
- 相同状态使用统一颜色、文案和交互逻辑。
- 成功、失败、待处理等状态表达应前后一致。

= 6. 接口定义

== 6.1 外部接口

#tri-table(
  ("接口对象", "类型", "用途"),
  (
    ([微信登录接口], [外部平台接口], [获取 openid，完成登录与绑定]),
    ([微信订阅消息接口], [外部平台接口], [完成订阅通知发送]),
    ([文件系统或存储介质], [外部资源接口], [保存上传文件与生成文档]),
    ([数据库], [基础设施接口], [持久化业务数据]),
  ),
)

== 6.2 内部接口

系统内部以 REST API 作为前后端协作接口，主要包括：

- 认证与用户接口
- 微信登录与绑定接口
- 文件上传、查询、下载接口
- 知识库检索与管理接口
- 通知模板、发送记录和未读数接口
- 党团流程查询与维护接口
- 审批申请与处理接口

#quad-table(
  ("接口域", "调用方", "代表接口", "职责说明"),
  (
    ([认证与身份], [学生端、管理端], [`/api/v1/wechat/login`、`/api/v1/wechat/bind`], [完成登录、绑定和身份建立]),
    ([用户与班级], [管理端], [`/api/v1/admin/users`、`/api/v1/admin/classes`], [完成学生与班级资料维护]),
    (
      [知识与文件],
      [学生端、管理端],
      [`/api/v1/knowledge/search`、`/api/v1/files/upload`],
      [提供问答检索、文件上传与引用能力],
    ),
    (
      [通知与流程],
      [学生端、管理端],
      [`/api/v1/user/subscribe/report`、党团流程接口、审批接口],
      [支撑状态提醒、流程维护和事务审批],
    ),
  ),
)

== 6.3 接口规范要求

- API 前缀统一为 `/api/v1`。
- 所有受限接口应要求 JWT 认证。
- 成功和失败响应格式统一。
- 列表接口应支持分页参数。
- 业务错误应返回可读错误信息。

= 7. 进度要求

系统建设建议分阶段推进：

1. 第一阶段。完成统一身份、用户权限、班级管理、微信登录与基础文件服务。
2. 第二阶段。完成知识库、通知能力、公告管理、党团流程等核心业务模块。
3. 第三阶段。完成电子证明、审批流程、体验优化与联调验收。

进度安排应遵循“基础能力优先、共享服务优先、核心闭环优先”的原则。

= 8. 交付要求

项目交付内容应包括：

- 可运行的软件系统。
- 后端源代码与必要配置说明。
- 数据模型和接口文档。
- 软件需求规格说明书。
- 设计说明、测试说明与答辩材料。

= 9. 何种形式来交付

项目交付形式包括：

- 系统演示环境或可部署项目代码。
- 文档形式交付的需求说明、设计说明、API 文档和测试材料。
- 用于课程汇报和老师评审的演示文稿。

= 10. 验收要求

系统验收应围绕“功能正确、边界清晰、流程完整、文档齐全”进行。

== 10.1 功能验收

- 学生能够完成登录、查看个人信息、搜索知识、查看通知和查询本人流程。
- 管理员能够完成学生管理、知识维护、通知模板管理和范围内事务维护。
- 文件上传、下载、附件引用流程可正常执行。
- 党团流程与审批流程具备基本闭环。

== 10.2 非功能验收

- 系统具备基本权限控制与审计能力。
- 关键接口响应格式统一。
- 文档描述与系统范围前后一致。
- 代码结构与模块边界符合开发规范。

== 10.3 文档验收

- 需求文档内容完整、术语统一、图文结合。
- 需求说明与现有实现、现有规划不冲突。
- 文档能清晰说明系统目标、功能范围和实现边界。

== 10.4 评审结论标准

若系统能够在演示环境中完成核心业务场景展示，并且需求文档、接口文档、设计材料与实现结果保持一致，则可视为满足课程项目的阶段性验收要求。
