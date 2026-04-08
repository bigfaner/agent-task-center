# PRD: Agent Task Center

## Background

当前 zcode 的 task-cli 是纯本地文件系统工具，任务数据（index.json）存在于单个项目目录中。这带来以下限制：

1. **无法集中管理** — 无法在一个地方查看多个项目的任务状态
2. **无可视化** — 缺少任务依赖图和执行进度的可视化
3. **无共享** — 多 agent 各自操作本地文件，无法共享任务状态
4. **无追踪** — 缺少执行历史和 agent 活动追踪

### 目标用户

| 用户 | 交互方式 | 核心需求 |
|------|---------|---------|
| 开发者/管理者 | Web UI | 查看多项目任务看板、监控执行进度 |
| AI Agent（Claude Code、Codex 等） | task-cli 远程模式 | 领取/管理任务，与现有工作流无缝衔接 |

## Goals

1. **集中化任务管理** — 提供统一的任务中心，管理多个项目的任务生命周期
2. **Agent 友好** — AI agent 通过 task-cli 远程模式无缝接入，零学习成本
3. **可视化看板** — 通过 Web UI 直观展示任务状态、依赖关系和执行进度
4. **执行追踪** — 记录每个任务的执行历史，支持回顾和分析

### 成功指标

- task-cli 切换到远程模式后，所有现有命令（claim、record、status 等）正常工作
- Web UI 能实时反映任务状态变更
- 支持至少 3 个 agent 同时领取不同任务而不冲突

## Scope

### In Scope

- Go 后端 REST API 服务（双路由：`/api/agent/` 面向 AI，`/api/` 面向 Web UI）
- Vue3 Web UI（任务看板、任务树、执行记录）
- task-cli 远程模式扩展（环境变量切换本地/远程模式）
- 多项目管理
- SQLite（本地开发）+ PostgreSQL（生产）双存储适配
- 扩展数据模型（Project → Feature → Task 三层结构）
- index.json 导入接口
- 任务内容（TaskContent）和 Feature 附件（FeatureAttachment）以 BLOB 存储，单文件限制 50MB
- `task get-content` 子命令

### Out of Scope（V1 不做）

- 认证/授权（API Key、用户系统等，后续迭代统一规划）
- 实时通知（WebSocket）
- CI/CD 集成
- 任务自动调度算法
- MCP Server 模式

## Non-Functional Requirements

| 类别 | 要求 |
|------|------|
| 并发安全 | 任务认领使用乐观锁（version 字段），同一任务并发认领时只有一个成功，其余返回 409 |
| API 响应时间 | 列表接口 P99 < 500ms，单任务操作 P99 < 200ms（本地 SQLite 环境） |
| 文件大小限制 | 单个附件/任务内容上传不超过 50MB，超出返回 413 |
| 存储兼容性 | 同一套代码通过配置切换 SQLite（开发）和 PostgreSQL（生产），无需修改业务逻辑 |
| 可观测性 | 所有 API 请求记录结构化日志（method、path、status、latency） |
| AI 友好错误信息 | 所有错误响应包含机器可读的 `code` 字段和人类可读的 `message` 字段，`code` 使用语义化字符串（如 `task.already_claimed`、`feature.not_found`），不使用数字错误码 |

**错误响应格式示例：**

```json
{
  "code": "task.already_claimed",
  "message": "Task 'feat-1-project-management/1.1' is already claimed by agent 'claude-agent-01'. Use /api/agent/tasks/feat-1-project-management%2F1.1/status to update its status.",
  "task_key": "feat-1-project-management/1.1",
  "claimed_by": "claude-agent-01"
}
```

错误 `message` 应包含足够上下文，让 agent 无需查阅文档即可理解原因并采取下一步行动。

---

## 本地目录结构

每个 Feature 在本地对应一个独立目录，通过 `task feature push/pull` 与服务端同步：

```
docs/features/<project-slug>/
├── feat-1-project-management/
│   ├── design.md                   # 技术方案（可 push 到服务端）
│   ├── ui/                         # UI 设计稿
│   │   └── mockup.pen
│   ├── tasks/
│   │   └── index.json
│   └── records/
│
├── feat-2-task-execution/
│   ├── design.md
│   ├── tasks/
│   └── records/
│
└── feat-N-.../
```

## 数据模型

层级结构：**Project → Feature → Task**

### Key 设计原则

所有实体同时拥有**自增整型 ID**（数据库主键，用于外键关联）和**语义化 key**（业务标识符，用于 API 和 CLI）：

| 实体 | Key 格式 | 示例 |
|------|---------|------|
| Project | 用户指定的 slug | `agent-task-center` |
| Feature | `feat-{N}-{slug}` | `feat-1-project-management` |
| Task | `{feature_key}/{task_id}` | `feat-1-project-management/1.1` |
| FeatureAttachment | `{feature_key}/attachments/{filename}` | `feat-1-project-management/attachments/design.md` |
| ExecutionRecord | `{task_key}/records/{timestamp}` | `feat-1-project-management/1.1/records/20260408T103000` |

- **ID**：自增整型，数据库内部外键关联使用，不暴露给外部 API
- **Key**：全局唯一语义字符串，用于所有 API 路径和 CLI 参数，agent 可直接构造无需先查询

```go
type Project struct {
    ID          int64     `json:"-"`            // 数据库主键，不暴露
    Key         string    `json:"key"`          // slug，如 "agent-task-center"
    Name        string    `json:"name"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type Feature struct {
    ID         int64     `json:"-"`            // 数据库主键，不暴露
    Key        string    `json:"key"`          // 如 "feat-1-project-management"
    ProjectID  int64     `json:"-"`            // 外键，不暴露
    ProjectKey string    `json:"project_key"`
    Name       string    `json:"name"`
    Description string   `json:"description"`
    Status     string    `json:"status"`       // pending/in_progress/completed
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}

// FeatureAttachment 统一管理 Feature 关联的所有文件，内容以 BLOB 存储（SQLite: BLOB，PostgreSQL: BYTEA）
type FeatureAttachment struct {
    ID          int64     `json:"-"`            // 数据库主键，不暴露
    Key         string    `json:"key"`          // 如 "feat-1-project-management/attachments/design.md"
    FeatureID   int64     `json:"-"`            // 外键，不暴露
    FeatureKey  string    `json:"feature_key"`
    Name        string    `json:"name"`         // 文件名，如 "design.md", "mockup.pen"
    ContentType string    `json:"content_type"` // MIME 类型
    Data        []byte    `json:"-"`            // 二进制内容，不序列化到 JSON
    FileSize    int64     `json:"file_size"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type Task struct {
    ID            int64      `json:"-"`            // 数据库主键，不暴露
    Key           string     `json:"key"`          // 如 "feat-1-project-management/1.1"
    TaskID        string     `json:"task_id"`      // 短 ID，如 "1.1"，在 Feature 内唯一
    Title         string     `json:"title"`
    Priority      string     `json:"priority"`     // P0/P1/P2
    Status        string     `json:"status"`       // pending/in_progress/completed/blocked
    Dependencies  []string   `json:"dependencies"` // 依赖的 task key 列表
    ProjectID     int64      `json:"-"`            // 外键，不暴露
    ProjectKey    string     `json:"project_key"`
    FeatureID     int64      `json:"-"`            // 外键，不暴露
    FeatureKey    string     `json:"feature_key"`
    AgentID       *string    `json:"agent_id,omitempty"`
    Tags          []string   `json:"tags,omitempty"`
    EstimatedTime string     `json:"estimated_time,omitempty"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
    StartedAt     *time.Time `json:"started_at,omitempty"`
    CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// TaskContent 以 BLOB 存储任务详细内容（SQLite: BLOB，PostgreSQL: BYTEA）
type TaskContent struct {
    ID          int64     `json:"-"`            // 数据库主键，不暴露
    TaskID      int64     `json:"-"`            // 外键，不暴露
    TaskKey     string    `json:"task_key"`
    ContentType string    `json:"content_type"` // 通常为 text/markdown
    Data        []byte    `json:"-"`            // 二进制内容，不序列化到 JSON
    FileSize    int64     `json:"file_size"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type ExecutionRecord struct {
    ID                 int64                 `json:"-"`        // 数据库主键，不暴露
    Key                string                `json:"key"`      // 如 "feat-1-project-management/1.1/records/20260408T103000"
    TaskID             int64                 `json:"-"`        // 外键，不暴露
    TaskKey            string                `json:"task_key"`
    AgentID            string                `json:"agent_id"`
    Status             string                `json:"status"`
    Summary            string                `json:"summary"`
    FilesCreated       []string              `json:"files_created,omitempty"`
    FilesModified      []string              `json:"files_modified,omitempty"`
    KeyDecisions       []string              `json:"key_decisions,omitempty"`
    TestsPassed        int                   `json:"tests_passed"`
    TestsFailed        int                   `json:"tests_failed"`
    Coverage           float64               `json:"coverage"`
    AcceptanceCriteria []AcceptanceCriterion `json:"acceptance_criteria,omitempty"`
    CreatedAt          time.Time             `json:"created_at"`
}

type AcceptanceCriterion struct {
    Criterion string `json:"criterion"`
    Met       bool   `json:"met"`
}
```

## User Stories

### 开发者/管理者

**US-1**: As a developer, I want to view all tasks across multiple projects in one dashboard, so that I can monitor overall progress without switching between local directories.

**US-2**: As a developer, I want to filter tasks by project, feature, priority, and tag, so that I can focus on what's relevant to me at any given time.

**US-3**: As a developer, I want to view the execution history of a task (summary, files changed, test results), so that I can review what an agent did and audit its work.

**US-4**: As a developer, I want to create projects and features via the Web UI, so that I can organize tasks before handing them off to agents.

**US-5**: As a developer, I want to upload design documents and attachments to a feature, so that agents can access the latest specs without manual file sharing.

### AI Agent

**US-6**: As an AI agent, I want to claim the next available task with a single CLI command, so that I can start working without learning new tools or changing my existing workflow.

**US-7**: As an AI agent, I want to submit an execution record after completing a task, so that the task center reflects my work and other agents know the task is done.

**US-8**: As an AI agent, I want to fetch task content (Markdown) via CLI, so that I can read the full task specification before starting work.

---

## Features

### Feature 1: 项目管理 `P0`

管理 Project 和 Feature 的生命周期，为任务提供组织结构。

**Agent API** (`/api/agent/`) — key 寻址，供 task-cli 和 AI agent 调用

| API | Method | Description |
|-----|--------|-------------|
| `/api/agent/projects/{key}` | GET | 项目详情 |
| `/api/agent/projects/{key}/features` | GET | Feature 列表 |
| `/api/agent/features/{key}` | GET | Feature 详情 |
| `/api/agent/features/{key}/attachments` | GET | 附件列表 |
| `/api/agent/features/{key}/attachments/{name}` | GET | 下载附件内容 |
| `/api/agent/features/{key}/attachments/{name}` | PUT | 上传/替换附件（`task feature push`） |

**Web UI API** (`/api/`) — ID 寻址，供 Vue3 前端调用

| API | Method | Description |
|-----|--------|-------------|
| `/api/projects` | GET | 项目列表 |
| `/api/projects` | POST | 创建项目 |
| `/api/projects/{id}` | GET | 项目详情 |
| `/api/projects/{id}` | PUT | 更新项目 |
| `/api/projects/{id}` | DELETE | 删除项目（级联） |
| `/api/projects/{id}/features` | GET | Feature 列表 |
| `/api/projects/{id}/features` | POST | 创建 Feature |
| `/api/features/{id}` | GET | Feature 详情 |
| `/api/features/{id}` | PUT | 更新 Feature |
| `/api/features/{id}` | DELETE | 删除 Feature |
| `/api/features/{id}/attachments` | GET | 附件列表 |
| `/api/features/{id}/attachments` | POST | 上传附件（multipart/form-data） |
| `/api/features/{id}/attachments/{aid}` | GET | 下载附件 |
| `/api/features/{id}/attachments/{aid}` | PUT | 替换附件 |
| `/api/features/{id}/attachments/{aid}` | DELETE | 删除附件 |

**Acceptance Criteria**

- [ ] 创建项目，返回项目 key
- [ ] 查看项目列表
- [ ] 删除项目（级联删除其下 Feature 和 Task）
- [ ] 在项目下创建 Feature
- [ ] 查看项目下的 Feature 列表
- [ ] 更新/删除 Feature
- [ ] 通过 multipart 上传任意类型附件（.md、.html、.pen、图片等）
- [ ] 下载/替换/删除附件
- [ ] `task feature push` 将本地文件上传为附件
- [ ] `task feature pull` 将附件下载到本地对应目录

---

### Feature 2: 任务执行 `P0`

Agent 领取和执行任务的核心流程，包括任务 CRUD、状态流转、执行记录。

**Agent API** (`/api/agent/`) — key 寻址，供 task-cli 和 AI agent 调用

| API | Method | Description |
|-----|--------|-------------|
| `/api/agent/features/{key}/tasks` | GET | 任务列表（含依赖状态） |
| `/api/agent/features/{key}/tasks/next` | GET | 获取下一个可认领任务（pending 且依赖已全部完成） |
| `/api/agent/features/{key}/import` | POST | 导入 index.json 批量创建任务 |
| `/api/agent/tasks/{key}` | GET | 任务详情 |
| `/api/agent/tasks/{key}/content` | GET | 获取任务内容（Markdown） |
| `/api/agent/tasks/{key}/claim` | POST | 认领任务 |
| `/api/agent/tasks/{key}/status` | PUT | 更新任务状态 |
| `/api/agent/tasks/{key}/records` | POST | 提交执行记录 |

**Web UI API** (`/api/`) — ID 寻址，供 Vue3 前端调用

| API | Method | Description |
|-----|--------|-------------|
| `/api/features/{id}/tasks` | GET | 任务列表 |
| `/api/features/{id}/tasks` | POST | 创建任务 |
| `/api/tasks/{id}` | GET | 任务详情 |
| `/api/tasks/{id}` | PUT | 更新任务（标题、优先级、标签等） |
| `/api/tasks/{id}` | DELETE | 删除任务 |
| `/api/tasks/{id}/content` | GET | 获取任务内容 |
| `/api/tasks/{id}/content` | PUT | 更新任务内容 |
| `/api/tasks/{id}/status` | PUT | 更新任务状态（看板拖拽） |
| `/api/tasks/{id}/records` | GET | 执行记录列表 |
| `/api/agents` | GET | Agent 列表及活动状态 |

**Acceptance Criteria**

- [ ] 在 Feature 下创建任务
- [ ] 查看 Feature 下的任务列表
- [ ] 更新任务内容（标题、描述、优先级、标签等）
- [ ] 删除任务
- [ ] 导入 index.json 文件批量创建任务（自动关联 Feature）
- [ ] 创建/更新任务的详细内容（Markdown）
- [ ] 认领任务（设置 agent_id，状态变为 in_progress）
- [ ] 更新任务状态（pending/in_progress/completed/blocked）
- [ ] 并发认领不冲突（同一任务只能被一个 agent 认领）
- [ ] 创建执行记录（包含 summary、files、tests、coverage 等）
- [ ] 查看任务的执行历史

---

### Feature 3: 可视化看板 `P1`

Web UI，直观展示任务状态、依赖关系和执行进度。

**页面与组件清单**

| 页面/组件 | 路由/位置 | 职责 |
|-----------|-----------|------|
| `ProjectList.vue` | `/projects` | 项目列表入口 |
| `ProjectDetail.vue` | `/projects/:id` | Feature 列表 |
| `FeatureDetail.vue` | `/features/:id` | 看板 + 依赖图 + 附件 |
| `TaskDetail.vue` | `/tasks/:id` | 任务详情 + 执行记录 |
| `TaskBoard.vue` | FeatureDetail 子组件 | Kanban 看板，四列状态 |
| `TaskDependencyGraph.vue` | FeatureDetail 子组件 | D3.js 有向图 |
| `FilterBar.vue` | FeatureDetail 子组件 | 筛选条件栏 |
| `RecordTimeline.vue` | TaskDetail 子组件 | 执行记录时间线 |
| `TaskCard.vue` | TaskBoard 子组件 | 单个任务卡片 |

**功能描述**

**任务看板（TaskBoard）**
- 四列：待处理 / 进行中 / 已完成 / 已阻塞
- 使用 `vue-draggable-plus`（基于 SortableJS）支持拖拽变更状态
- 拖拽结束调用 `PUT /api/tasks/:id/status`
- 每张卡片显示：任务 ID、标题、优先级 badge、agent_id（如有）、依赖数量

**筛选栏（FilterBar）**
- 筛选维度：Project / Feature / 优先级（P0/P1/P2）/ 标签（多选）/ 状态
- 筛选条件序列化为 URL query params（如 `?priority=P0&tag=backend`），刷新后保留，支持分享链接
- Pinia `filterStore` 负责从 URL 初始化状态、同步写回 URL

**任务依赖图（TaskDependencyGraph）**
- 使用 D3.js 渲染 Feature 内任务的有向无环图（DAG）
- 节点：任务卡片（显示 task_id、title、status 颜色编码）
- 边：依赖关系箭头，方向为 "被依赖任务 → 依赖方任务"
- 节点颜色：pending=灰、in_progress=蓝、completed=绿、blocked=红
- 点击节点跳转到 `/tasks/:id`
- 图数据来源：`GET /api/features/:id/tasks` 返回的 `dependencies` 字段

**执行记录时间线（RecordTimeline）**
- 垂直时间线，每条记录显示：时间戳、agent_id、状态、summary
- 展开后显示：修改文件列表、关键决策、测试通过/失败数、覆盖率
- 数据来源：`GET /api/tasks/:id/records`

**Pinia Store 扩展**

| Store | 新增状态 | 说明 |
|-------|----------|------|
| `filterStore` | `project`, `feature`, `priority[]`, `tags[]`, `status[]` | 筛选条件，与 URL query params 双向同步 |
| `task.ts` | 现有 | 看板操作调用 `PUT /api/tasks/:id/status` |

**Acceptance Criteria**

- [ ] 任务看板：按状态分四列显示，支持拖拽变更状态，拖拽后实时更新
- [ ] 支持按 Project / Feature / 优先级 / 标签 / 状态筛选，筛选条件持久化到 URL query params
- [ ] 刷新页面后筛选条件保留；筛选链接可分享
- [ ] 任务依赖图：用 D3.js 渲染 Feature 内任务的 DAG，节点颜色反映任务状态
- [ ] 点击依赖图节点跳转到任务详情页
- [ ] 执行记录时间线：按时间倒序展示，支持展开查看详情（文件、决策、测试结果）
- [ ] TaskCard 显示优先级 badge 和 agent_id

---

### Feature 4: task-cli 远程模式 `P1`

扩展 task-cli，通过环境变量切换本地/远程模式，Agent 零学习成本接入。

**环境变量**

| 环境变量 | 示例值 | 说明 |
|---------|--------|------|
| `TASK_REMOTE_URL` | `http://localhost:8080` | 远程服务地址 |
| `TASK_PROJECT_KEY` | `agent-task-center` | 当前项目 key |
| `TASK_FEATURE_KEY` | `feat-1-project-management` | 当前 Feature key |

当 `TASK_REMOTE_URL` 设置时，task-cli 切换到远程模式，所有命令通过 HTTP 调用而非本地文件。未设置时保持现有本地文件行为。

新增子命令：
- `task get-content <task-key>`: 获取任务详细内容（本地模式读取本地文件，远程模式调用 Agent API）
- `task feature push [feature-key]`: 将本地 `design.md` 推送到服务端
- `task feature pull [feature-key]`: 从服务端拉取文档内容到本地 `design.md`

**Acceptance Criteria**

- [ ] 设置 `TASK_REMOTE_URL` 后，claim/record/status/query 等命令走 Agent API (`/api/agent/`)
- [ ] 未设置时保持现有本地文件行为
- [ ] 新增 `task get-content <task-key>` 子命令
- [ ] 支持通过 `TASK_FEATURE_KEY` 指定当前 Feature 上下文
- [ ] `task feature push` 将本地 design.md 上传到服务端
- [ ] `task feature pull` 将服务端内容写入本地 design.md