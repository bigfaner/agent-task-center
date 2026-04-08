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

- Go 后端 REST API 服务
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

```go
type Project struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type Feature struct {
    ID          string    `json:"id"`
    ProjectID   string    `json:"project_id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Status      string    `json:"status"`      // pending/in_progress/completed
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// FeatureAttachment 统一管理 Feature 关联的所有文件，内容以 BLOB 存储（SQLite: BLOB，PostgreSQL: BYTEA）
type FeatureAttachment struct {
    ID          string    `json:"id"`
    FeatureID   string    `json:"feature_id"`
    Name        string    `json:"name"`         // 文件名，如 "design.md", "mockup.pen", "wireframe.png"
    ContentType string    `json:"content_type"` // MIME 类型
    Data        []byte    `json:"-"`            // 二进制内容，不序列化到 JSON
    FileSize    int64     `json:"file_size"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type Task struct {
    // 复用现有字段
    ID           string   `json:"id"`
    Title        string   `json:"title"`
    Priority     string   `json:"priority"`    // P0/P1/P2
    Status       string   `json:"status"`      // pending/in_progress/completed/blocked
    Dependencies []string `json:"dependencies"`
    File         string   `json:"file"`
    Record       string   `json:"record"`

    // 扩展字段
    ProjectID     string     `json:"project_id"`
    FeatureID     string     `json:"feature_id"`
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
    ID          string    `json:"id"`
    TaskID      string    `json:"task_id"`
    ContentType string    `json:"content_type"` // 通常为 text/markdown
    Data        []byte    `json:"-"`            // 二进制内容，不序列化到 JSON
    FileSize    int64     `json:"file_size"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type ExecutionRecord struct {
    ID                 string                `json:"id"`
    TaskID             string                `json:"task_id"`
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

## Features

### Feature 1: 项目管理

管理 Project 和 Feature 的生命周期，为任务提供组织结构。

**API**

| API | Method | Description |
|-----|--------|-------------|
| `/api/projects` | GET | 项目列表 |
| `/api/projects` | POST | 创建项目 |
| `/api/projects/{id}` | GET | 项目详情 |
| `/api/projects/{id}` | DELETE | 删除项目 |
| `/api/projects/{id}/features` | GET | Feature 列表 |
| `/api/projects/{id}/features` | POST | 创建 Feature |
| `/api/features/{id}` | GET | Feature 详情 |
| `/api/features/{id}` | PUT | 更新 Feature |
| `/api/features/{id}` | DELETE | 删除 Feature |
| `/api/features/{id}/attachments` | GET | 附件列表 |
| `/api/features/{id}/attachments` | POST | 上传附件（multipart/form-data，支持任意文件类型） |
| `/api/features/{id}/attachments/{aid}` | GET | 下载附件 |
| `/api/features/{id}/attachments/{aid}` | PUT | 替换附件 |
| `/api/features/{id}/attachments/{aid}` | DELETE | 删除附件 |

**Acceptance Criteria**

- [ ] 创建项目，返回项目 ID
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

### Feature 2: 任务执行

Agent 领取和执行任务的核心流程，包括任务 CRUD、状态流转、执行记录。

**API**

| API | Method | Description |
|-----|--------|-------------|
| `/api/features/{id}/tasks` | GET | 任务列表 |
| `/api/features/{id}/tasks` | POST | 创建任务 |
| `/api/features/{id}/import` | POST | 导入 index.json 批量创建任务 |
| `/api/tasks/{id}` | GET | 任务详情 |
| `/api/tasks/{id}` | PUT | 更新任务 |
| `/api/tasks/{id}` | DELETE | 删除任务 |
| `/api/tasks/{id}/content` | GET | 获取任务内容 |
| `/api/tasks/{id}/content` | PUT | 更新任务内容 |
| `/api/tasks/{id}/claim` | POST | 认领任务 |
| `/api/tasks/{id}/status` | PUT | 更新任务状态 |
| `/api/tasks/{id}/records` | GET | 执行记录列表 |
| `/api/tasks/{id}/records` | POST | 创建执行记录 |
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

### Feature 3: 可视化看板

Web UI，直观展示任务状态、依赖关系和执行进度。

**功能描述**

**任务看板**
- 按状态分列显示：待处理 → 进行中 → 已完成 → 已阻塞
- 支持拖拽变更状态
- 按项目/Feature/优先级/标签筛选

**任务树**
- 树状结构展示 Project → Feature → Task 层级和依赖关系
- 点击任务节点显示详情

**执行记录**
- 时间线展示任务的执行历史
- 显示执行摘要、修改文件、关键决策、测试结果

**Acceptance Criteria**

- [ ] 任务看板：按状态分列显示，支持拖拽变更状态
- [ ] 支持按 Project / Feature / 优先级 / 标签筛选
- [ ] 任务树：树状展示 Project → Feature → Task 层级和依赖关系
- [ ] 执行记录：时间线展示执行历史

---

### Feature 4: task-cli 远程模式

扩展 task-cli，通过环境变量切换本地/远程模式，Agent 零学习成本接入。

**环境变量**

| 环境变量 | 示例值 | 说明 |
|---------|--------|------|
| `TASK_REMOTE_URL` | `http://localhost:8080` | 远程服务地址 |
| `TASK_PROJECT_ID` | `my-project` | 当前项目 ID |
| `TASK_FEATURE_ID` | `feat-001` | 当前 Feature ID |

当 `TASK_REMOTE_URL` 设置时，task-cli 切换到远程模式，所有命令通过 HTTP 调用而非本地文件。未设置时保持现有本地文件行为。

新增子命令：
- `task get-content <id>`: 获取任务详细内容（本地模式读取本地文件，远程模式调用 API）
- `task feature push [feature-id]`: 将本地 `design.md` 推送到服务端
- `task feature pull [feature-id]`: 从服务端拉取文档内容到本地 `design.md`

**Acceptance Criteria**

- [ ] 设置 `TASK_REMOTE_URL` 后，claim/record/status/query 等命令走远程 API
- [ ] 未设置时保持现有本地文件行为
- [ ] 新增 `task get-content` 子命令
- [ ] 支持通过 `TASK_FEATURE_ID` 指定当前 Feature 上下文
- [ ] `task feature push` 将本地 design.md 上传到服务端
- [ ] `task feature pull` 将服务端内容写入本地 design.md