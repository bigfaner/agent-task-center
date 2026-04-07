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
- 扩展 Task 数据模型（增加 project_id、agent_id、tags 等）
- index.json 导入接口
- 任务内容（TaskContent）管理
- `task get-content` 子命令

### Out of Scope（V1 不做）

- 认证/授权（API Key、用户系统等，后续迭代统一规划）
- 实时通知（WebSocket）
- CI/CD 集成
- 任务自动调度算法
- MCP Server 模式

## Requirements

### 4.1 后端 API

| API | Method | Description |
|-----|--------|-------------|
| `/api/projects` | GET | 项目列表 |
| `/api/projects` | POST | 创建项目 |
| `/api/projects/{id}` | DELETE | 删除项目 |
| `/api/projects/{id}/tasks` | GET | 任务列表 |
| `/api/projects/{id}/tasks` | POST | 创建任务 |
| `/api/projects/{id}/tasks/content/{taskId}` | GET | 获取任务内容 |
| `/api/projects/{id}/tasks/content/{taskId}` | PUT | 更新任务内容 |
| `/api/projects/{id}/import` | POST | 导入 index.json 批量创建任务 |
| `/api/tasks/{id}` | GET | 任务详情 |
| `/api/tasks/{id}` | PUT | 更新任务 |
| `/api/tasks/{id}` | DELETE | 删除任务 |
| `/api/tasks/{id}/claim` | POST | 认领任务 |
| `/api/tasks/{id}/status` | PUT | 更新任务状态 |
| `/api/tasks/{id}/records` | GET | 执行记录列表 |
| `/api/tasks/{id}/records` | POST | 创建执行记录 |
| `/api/agents` | GET | Agent 列表及活动状态 |

### 4.2 数据模型

```go
type Project struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
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
    AgentID       *string    `json:"agent_id,omitempty"`
    Tags          []string   `json:"tags,omitempty"`
    EstimatedTime string     `json:"estimated_time,omitempty"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
    StartedAt     *time.Time `json:"started_at,omitempty"`
    CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

type TaskContent struct {
    ID        string    `json:"id"`
    TaskID    string    `json:"task_id"`
    Content   string    `json:"content"`     // Markdown 内容
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
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

### 4.3 task-cli 远程模式

通过环境变量控制：

| 环境变量 | 示例值 | 说明 |
|---------|--------|------|
| `TASK_REMOTE_URL` | `http://localhost:8080` | 远程服务地址 |
| `TASK_PROJECT_ID` | `my-project` | 当前项目 ID |

当 `TASK_REMOTE_URL` 设置时，task-cli 切换到远程模式，所有命令通过 HTTP 调用而非本地文件。未设置时保持现有本地文件行为。

新增子命令：
- `task get-content <id>`: 获取任务详细内容（本地模式读取本地文件，远程模式调用 API）

### 4.4 Web UI 功能

**任务看板**
- 按状态分列显示：待处理 → 进行中 → 已完成 → 已阻塞
- 支持拖拽变更状态
- 按项目/优先级/标签筛选

**任务树**
- 树状结构展示任务层级和依赖关系
- 点击任务节点显示详情

**执行记录**
- 时间线展示任务的执行历史
- 显示执行摘要、修改文件、关键决策、测试结果

## Acceptance Criteria

### AC-1: 项目管理
- [ ] 创建项目，返回项目 ID
- [ ] 查看项目列表
- [ ] 删除项目

### AC-2: 任务 CRUD
- [ ] 在项目下创建任务
- [ ] 查看项目下的任务列表
- [ ] 更新任务内容（标题、描述、优先级、标签等）
- [ ] 删除任务
- [ ] 导入 index.json 文件批量创建任务

### AC-3: 任务内容
- [ ] 创建/更新任务的详细内容（Markdown）
- [ ] 通过 API 和 task-cli `get-content` 获取任务内容

### AC-4: 任务状态流转
- [ ] 认领任务（设置 agent_id，状态变为 in_progress）
- [ ] 更新任务状态（pending/in_progress/completed/blocked）
- [ ] 并发认领不冲突（同一任务只能被一个 agent 认领）

### AC-5: 执行记录
- [ ] 创建执行记录（包含 summary、files、tests、coverage 等）
- [ ] 查看任务的执行历史

### AC-6: task-cli 远程模式
- [ ] 设置 `TASK_REMOTE_URL` 后，claim/record/status/query 等命令走远程 API
- [ ] 未设置时保持现有本地文件行为
- [ ] 新增 `task get-content` 子命令

### AC-7: Web UI
- [ ] 任务看板：按状态分列显示，支持拖拽变更状态
- [ ] 任务树：树状展示依赖关系
- [ ] 执行记录：时间线展示执行历史
