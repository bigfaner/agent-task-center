---
created: 2026-04-13
prd: prd/prd-spec.md
status: Draft
---

# Technical Design: Agent Task Center (V1)

## Overview

Agent Task Center 是一个集中化的任务可视化与协作服务，由三个部分组成：

1. **Go Server** — REST API 服务，负责数据存储、文件解析、Web API 和 Agent API
2. **React Web UI** — 只读看板 + 文件上传，开发者通过浏览器查看四层实体状态
3. **task-cli 远程模式** — 通过 `TASK_SERVER_URL` 环境变量切换本地/远程模式

整体采用 Monorepo 结构，server/ 和 web/ 共存于同一仓库。Server 编译为裸二进制，支持 SQLite（本地/个人）和 PostgreSQL（服务端部署）双数据库，通过 `DB_DRIVER` 环境变量切换。

## Architecture

### Layer Placement

```
┌─────────────────────────────────────────────────────┐
│                   Web UI (React + Vite)              │
│  ProjectList | ProjectDetail | FeatureKanban         │
│  TaskDetail  | DocViewer     | FileUpload            │
└──────────────────────┬──────────────────────────────┘
                       │ REST API
┌──────────────────────┴──────────────────────────────┐
│                   Go Server                          │
│  handler/ → service/ → db/                          │
│  parser/ (index.json / proposal.md / manifest.md)   │
└──────┬────────────────────────┬─────────────────────┘
       │                        │
  SQLite (本地)          PostgreSQL (服务端)
```

### Component Diagram

```
agent-task-center/
├── server/
│   ├── cmd/server/main.go       # 入口，读取配置，启动 HTTP server
│   ├── internal/
│   │   ├── config/              # 环境变量配置
│   │   ├── db/                  # 数据库抽象层 + migrations
│   │   ├── handler/             # HTTP handlers (web + agent + upload)
│   │   ├── model/               # 数据模型定义
│   │   ├── parser/              # 文件解析 (index.json / .md)
│   │   └── service/             # 业务逻辑 (upsert / claim / record)
│   └── go.mod
├── web/
│   ├── src/
│   │   ├── pages/               # 页面组件
│   │   ├── components/          # 通用组件
│   │   └── api/                 # API client
│   ├── package.json
│   └── vite.config.ts
└── docs/
```

### Key Dependencies

**Server (Go):**
| 依赖 | 用途 |
|------|------|
| `github.com/go-chi/chi/v5` | HTTP router |
| `github.com/jmoiron/sqlx` | SQL 扩展（支持 struct scan） |
| `modernc.org/sqlite` | SQLite 驱动（纯 Go，无 CGO） |
| `github.com/lib/pq` | PostgreSQL 驱动 |
| `github.com/golang-migrate/migrate/v4` | 数据库迁移 |

**Web (React):**
| 依赖 | 用途 |
|------|------|
| `react` + `react-router-dom` | 路由 |
| `@tanstack/react-query` | 数据请求 + 缓存 |
| `react-markdown` + `remark-gfm` | Markdown 渲染 |
| `shadcn/ui` + `tailwindcss` | UI 组件库 |

## Data Models

### Go Structs

```go
// Project — 顶层实体，首次 push 时自动创建
type Project struct {
    ID        int64     `db:"id"`
    Name      string    `db:"name"`       // unique，来自 push 时的 project 参数
    CreatedAt time.Time `db:"created_at"`
    UpdatedAt time.Time `db:"updated_at"`
}

// Proposal — 来自 proposal.md
type Proposal struct {
    ID        int64     `db:"id"`
    ProjectID int64     `db:"project_id"`
    Slug      string    `db:"slug"`       // 文件路径中的目录名，unique per project
    Title     string    `db:"title"`      // 解析自 Markdown 第一个 H1
    Content   string    `db:"content"`    // 原始 Markdown 内容
    CreatedAt time.Time `db:"created_at"`
    UpdatedAt time.Time `db:"updated_at"`
}

// Feature — 来自 manifest.md
type Feature struct {
    ID        int64     `db:"id"`
    ProjectID int64     `db:"project_id"`
    Slug      string    `db:"slug"`       // manifest.md frontmatter 中的 feature 字段，unique per project
    Name      string    `db:"name"`       // 显示名称
    Status    string    `db:"status"`     // prd/design/tasks/in-progress/done
    Content   string    `db:"content"`    // 原始 manifest.md 内容
    CreatedAt time.Time `db:"created_at"`
    UpdatedAt time.Time `db:"updated_at"`
}

// Task — 来自 index.json
type Task struct {
    ID           int64     `db:"id"`
    FeatureID    int64     `db:"feature_id"`
    TaskID       string    `db:"task_id"`       // 如 "1.1"，unique per feature
    Title        string    `db:"title"`
    Description  string    `db:"description"`   // Markdown 内容
    Status       string    `db:"status"`        // pending/in_progress/completed/blocked
    Priority     string    `db:"priority"`      // P0/P1/P2
    Tags         string    `db:"tags"`          // JSON array，如 ["core","api"]
    Dependencies string    `db:"dependencies"`  // JSON array of task_id strings
    ClaimedBy    string    `db:"claimed_by"`    // agent_id，未认领为空
    Version      int64     `db:"version"`       // 乐观锁版本号，初始为 0
    CreatedAt    time.Time `db:"created_at"`
    UpdatedAt    time.Time `db:"updated_at"`
}

// ExecutionRecord — 来自 task record 提交
type ExecutionRecord struct {
    ID                 int64     `db:"id"`
    TaskID             int64     `db:"task_id"`
    AgentID            string    `db:"agent_id"`
    Summary            string    `db:"summary"`
    FilesCreated       string    `db:"files_created"`       // JSON array
    FilesModified      string    `db:"files_modified"`      // JSON array
    KeyDecisions       string    `db:"key_decisions"`       // JSON array
    TestsPassed        int       `db:"tests_passed"`
    TestsFailed        int       `db:"tests_failed"`
    Coverage           float64   `db:"coverage"`
    AcceptanceCriteria string    `db:"acceptance_criteria"` // JSON array of {criterion, met}
    CreatedAt          time.Time `db:"created_at"`
}
```

### Input / Summary / Detail Structs

```go
// --- Parser 输入类型 ---

type TaskInput struct {
    TaskID       string   `json:"task_id"`
    Title        string   `json:"title"`
    Description  string   `json:"description"`
    Priority     string   `json:"priority"`     // P0/P1/P2，默认 P1
    Tags         []string `json:"tags"`
    Dependencies []string `json:"dependencies"` // task_id 列表
}

type ProposalInput struct {
    Slug    string // 来自文件路径目录名
    Title   string // 解析自 Markdown 第一个 H1
    Content string // 原始 Markdown 内容
}

type FeatureInput struct {
    Slug    string // manifest.md frontmatter feature 字段
    Name    string // 显示名称（同 slug 或 frontmatter name 字段）
    Status  string // frontmatter status 字段
    Content string // 原始 manifest.md 内容
}

// --- Service 聚合返回类型 ---

type ProjectSummary struct {
    ID             int64     `json:"id"`
    Name           string    `json:"name"`
    FeatureCount   int       `json:"featureCount"`
    TaskTotal      int       `json:"taskTotal"`
    CompletionRate float64   `json:"completionRate"` // 0-100
    UpdatedAt      time.Time `json:"updatedAt"`
}

type ProjectDetail struct {
    ID        int64            `json:"id"`
    Name      string           `json:"name"`
    Proposals []ProposalSummary `json:"proposals"`
    Features  []FeatureSummary  `json:"features"`
}

type ProposalSummary struct {
    ID           int64     `json:"id"`
    Slug         string    `json:"slug"`
    Title        string    `json:"title"`
    CreatedAt    time.Time `json:"createdAt"`
    FeatureCount int       `json:"featureCount"`
}

type FeatureSummary struct {
    ID             int64     `json:"id"`
    Slug           string    `json:"slug"`
    Name           string    `json:"name"`
    Status         string    `json:"status"`
    CompletionRate float64   `json:"completionRate"`
    UpdatedAt      time.Time `json:"updatedAt"`
}

type TaskDetail struct {
    ID           int64     `json:"id"`
    TaskID       string    `json:"taskId"`
    Title        string    `json:"title"`
    Description  string    `json:"description"`
    Status       string    `json:"status"`
    Priority     string    `json:"priority"`
    Tags         []string  `json:"tags"`
    ClaimedBy    string    `json:"claimedBy"`
    Dependencies []string  `json:"dependencies"`
    CreatedAt    time.Time `json:"createdAt"`
    UpdatedAt    time.Time `json:"updatedAt"`
}

type UpsertSummary struct {
    Filename string `json:"filename"`
    Created  int    `json:"created"`
    Updated  int    `json:"updated"`
    Skipped  int    `json:"skipped"`
    Message  string `json:"message"`
}
```

### Database Schema

```sql
CREATE TABLE projects (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE proposals (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL REFERENCES projects(id),
    slug       TEXT NOT NULL,
    title      TEXT NOT NULL,
    content    TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, slug)
);

CREATE TABLE features (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL REFERENCES projects(id),
    slug       TEXT NOT NULL,
    name       TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'prd',
    content    TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, slug)
);

CREATE TABLE tasks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    feature_id   INTEGER NOT NULL REFERENCES features(id),
    task_id      TEXT NOT NULL,
    title        TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'pending',
    priority     TEXT NOT NULL DEFAULT 'P1',
    tags         TEXT NOT NULL DEFAULT '[]',
    dependencies TEXT NOT NULL DEFAULT '[]',
    claimed_by   TEXT NOT NULL DEFAULT '',
    version      INTEGER NOT NULL DEFAULT 0,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(feature_id, task_id)
);

CREATE TABLE execution_records (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id             INTEGER NOT NULL REFERENCES tasks(id),
    agent_id            TEXT NOT NULL,
    summary             TEXT NOT NULL,
    files_created       TEXT NOT NULL DEFAULT '[]',
    files_modified      TEXT NOT NULL DEFAULT '[]',
    key_decisions       TEXT NOT NULL DEFAULT '[]',
    tests_passed        INTEGER NOT NULL DEFAULT 0,
    tests_failed        INTEGER NOT NULL DEFAULT 0,
    coverage            REAL NOT NULL DEFAULT 0,
    acceptance_criteria TEXT NOT NULL DEFAULT '[]',
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

> PostgreSQL 兼容性：将 `INTEGER PRIMARY KEY AUTOINCREMENT` 改为 `BIGSERIAL PRIMARY KEY`，`DATETIME` 改为 `TIMESTAMPTZ`，其余 SQL 兼容。通过 `DB_DRIVER=postgres` 切换。

## Interfaces

### Service Layer

```go
// ProjectService
type ProjectService interface {
    List(ctx context.Context, search string, page, pageSize int) ([]ProjectSummary, int, error)
    Get(ctx context.Context, id int64) (*ProjectDetail, error)
    Upsert(ctx context.Context, name string) (*Project, error)
}

// FeatureService
type FeatureService interface {
    ListByProject(ctx context.Context, projectID int64) ([]FeatureSummary, error)
    GetTasks(ctx context.Context, featureID int64) ([]Task, error)
}

// TaskService
type TaskService interface {
    Get(ctx context.Context, id int64) (*TaskDetail, error)
    ListRecords(ctx context.Context, taskID int64, page, pageSize int) ([]ExecutionRecord, int, error)
    // 乐观锁认领：返回认领成功的任务，若无可用任务返回 ErrNoAvailableTask
    Claim(ctx context.Context, featureSlug, projectName, agentID string) (*Task, error)
    UpdateStatus(ctx context.Context, taskID int64, agentID, status string) error
    SubmitRecord(ctx context.Context, taskID int64, record *ExecutionRecord) error
}

// UploadService
type UploadService interface {
    ParseAndUpsert(ctx context.Context, projectName string, filename string, content []byte) (*UpsertSummary, error)
}
```

### Parser Interface

```go
type Parser interface {
    // 返回解析后的实体，由 UploadService 负责 Upsert
    ParseIndexJSON(data []byte) ([]TaskInput, error)
    ParseProposalMD(slug string, data []byte) (*ProposalInput, error)
    ParseManifestMD(data []byte) (*FeatureInput, error)
}
```

### Config

```go
type Config struct {
    DBDriver   string // "sqlite" | "postgres"，默认 "sqlite"
    DBPath     string // SQLite 文件路径，默认 "./data/tasks.db"
    DBConnStr  string // PostgreSQL DSN
    Port       int    // HTTP 端口，默认 8080
    StaticDir  string // Web UI 静态文件目录（空则不 serve）
}
```

## Optimistic Locking — Claim 流程

```
1. 查询当前 feature 下所有 pending 且依赖已满足的任务，按 priority 排序（P0 > P1 > P2）
2. 取第一个候选任务，记录其 (id, version)
3. 执行 UPDATE:
   UPDATE tasks
   SET status='in_progress', claimed_by=?, version=version+1, updated_at=?
   WHERE id=? AND version=? AND status='pending'
4. 若 rows_affected=0（被其他 Agent 抢先），取下一个候选任务重试
5. 最多重试 3 次，全部失败返回 ErrNoAvailableTask
```

依赖满足条件：`dependencies` 中所有 task_id 对应的任务 status = 'completed'。

## task-cli 远程模式集成

task-cli 检测 `TASK_SERVER_URL` 环境变量：

| 环境变量 | 行为 |
|---------|------|
| 未设置 | 本地文件系统模式（现有行为） |
| 已设置（如 `http://localhost:8080`） | 远程模式，所有操作通过 HTTP 请求 Server |

远程模式下的命令映射：

| task-cli 命令 | HTTP 请求 |
|--------------|-----------|
| `task push` | `POST /api/push` multipart |
| `task claim` | `POST /api/agent/claim` |
| `task status <id> <status>` | `PATCH /api/agent/tasks/{taskId}/status` |
| `task record --data record.json` | `POST /api/agent/tasks/{taskId}/records` |
| `task get-content <key>` | `GET /api/agent/tasks/{key}/content` |

### agent.yaml — Agent 身份配置

agent_id 从项目根目录下的 `agent.yaml` 读取，不使用环境变量。

```yaml
# agent.yaml（放在项目根目录，不提交到 git）
agent_id: "agent-01"        # 必填，唯一标识此 Agent 实例
description: "主力开发 Agent" # 可选，描述信息
```

读取优先级：
1. 项目根目录 `./agent.yaml`
2. 若文件不存在或 `agent_id` 为空，task-cli 报错并提示用户创建配置文件

```
Error: agent_id not configured.
Please create agent.yaml in your project root:

  agent_id: "your-agent-name"

This identifies you when claiming and submitting tasks.
```

`Config` struct 更新：

```go
type Config struct {
    DBDriver   string // "sqlite" | "postgres"，默认 "sqlite"
    DBPath     string // SQLite 文件路径，默认 "./data/tasks.db"
    DBConnStr  string // PostgreSQL DSN
    Port       int    // HTTP 端口，默认 8080
    StaticDir  string // Web UI 静态文件目录（空则不 serve）
}

// AgentConfig — task-cli 侧读取，不属于 Server 配置
type AgentConfig struct {
    AgentID     string `yaml:"agent_id"`
    Description string `yaml:"description"`
}

## File Upload & Push 解析逻辑

```
POST /api/upload?project=<name>  或  POST /api/push?project=<name>

文件名路由：
  index.json    → ParseIndexJSON → Upsert Tasks（按 task_id，需要 feature slug）
  proposal.md   → ParseProposalMD → Upsert Proposal
  manifest.md   → ParseManifestMD → Upsert Feature

Upsert 语义：
  - Project: 按 name 查找，不存在则创建
  - Proposal: 按 (project_id, slug) 查找，存在则更新 title/content
  - Feature: 按 (project_id, slug) 查找，存在则更新 name/status/content
  - Task: 按 (feature_id, task_id) 查找，存在则更新 title/description/priority/tags/dependencies
          不更新 status/claimed_by/version（保留执行状态）
```

push 命令扫描目录结构：
```
docs/proposals/<slug>/proposal.md  → slug 取目录名
docs/features/<slug>/manifest.md   → slug 取目录名
docs/features/<slug>/tasks/index.json → feature_slug 取目录名
```

## Error Handling

### Error Types

```go
var (
    ErrNotFound         = errors.New("not found")
    ErrNoAvailableTask  = errors.New("no available task")
    ErrVersionConflict  = errors.New("version conflict")
    ErrInvalidFile      = errors.New("invalid file format")
    ErrUnauthorizedAgent = errors.New("task claimed by different agent")
)
```

### Error Propagation Strategy

```
db 层     → 返回原始 database/sql 错误（sql.ErrNoRows、驱动错误等）
service 层 → 将 sql.ErrNoRows 包装为 ErrNotFound；version CAS 失败包装为 ErrVersionConflict；其余错误透传
handler 层 → 通过 errors.Is 匹配 sentinel errors，映射到 HTTP 状态码；未匹配的返回 500
```

示例：
```go
// service 层
row, err := db.GetTask(ctx, id)
if errors.Is(err, sql.ErrNoRows) {
    return nil, ErrNotFound
}

// handler 层
func writeError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, ErrNotFound):
        respondJSON(w, 404, errorBody("not_found", err))
    case errors.Is(err, ErrNoAvailableTask):
        respondJSON(w, 404, errorBody("no_available_task", err))
    case errors.Is(err, ErrVersionConflict):
        respondJSON(w, 409, errorBody("version_conflict", err))
    case errors.Is(err, ErrUnauthorizedAgent):
        respondJSON(w, 403, errorBody("unauthorized_agent", err))
    case errors.Is(err, ErrInvalidFile):
        respondJSON(w, 400, errorBody("invalid_file", err))
    default:
        respondJSON(w, 500, errorBody("internal_error", err))
    }
}
```

### HTTP Error Responses

错误响应同时面向人类开发者和 AI Agent，提供结构化 + 自然语言双重信息：

```json
{
  "error": "error_code",
  "message": "简洁的人类可读描述",
  "hint": "具体的修复建议或下一步操作（可选）"
}
```

| 场景 | HTTP Status | error_code | message | hint |
|------|-------------|------------|---------|------|
| 资源不存在 | 404 | `not_found` | `"Resource not found"` | `"Check the ID is correct"` |
| 无可用任务 | 404 | `no_available_task` | `"No tasks available to claim"` | `"All pending tasks are either claimed or have unmet dependencies"` |
| 乐观锁冲突 | 409 | `version_conflict` | `"Task was claimed by another agent"` | `"Retry claim to get the next available task"` |
| 文件格式无效 | 400 | `invalid_file` | `"Invalid file format"` | `"Only .json and .md files are accepted. index.json must contain task_id and title fields"` |
| 必填字段缺失 | 400 | `missing_field` | `"Required field missing: {field}"` | `"Provide the missing field and retry"` |
| Agent 无权操作 | 403 | `unauthorized_agent` | `"Task is claimed by a different agent"` | `"Only the agent that claimed this task can update it"` |
| 非法状态值 | 400 | `invalid_status` | `"Invalid status: {value}"` | `"Valid values: pending, in_progress, blocked"` |
| 服务器内部错误 | 500 | `internal_error` | `"Internal server error"` | `"Check server logs for details"` |

## Testing Strategy

### Unit Tests

- `parser/` — 各文件格式的解析逻辑，覆盖正常/边界/错误输入
  - 工具：`testify/assert`
- `service/` — Claim 乐观锁逻辑（mock DB interface），Upsert 语义验证
  - 工具：`testify/assert` + interface stub（手写 mock，无需 gomock）

### Integration Tests

- 使用 SQLite in-memory DB，测试完整 HTTP 请求链路
  - 工具：`net/http/httptest` + `testify/assert`
- 关键场景：
  - 3 个 Agent 并发 claim，验证无冲突
  - push → claim → record 完整流程
  - Upsert 幂等性（重复 push 不丢失执行记录）

### Web UI Tests

- 工具：Vitest + React Testing Library
- 范围：关键页面组件渲染（ProjectList、FeatureKanban、TaskDetail）
- API 调用使用 `msw`（Mock Service Worker）mock
- 覆盖率目标：核心组件 ≥ 60%

### Coverage Target

- `parser/` + `service/` ≥ 80%
- `handler/` ≥ 60%（集成测试覆盖）
- Web UI 核心组件 ≥ 60%

### Filtering Strategy Note

V1 筛选采用**服务端过滤**：前端将筛选参数（priority、tag、status）作为 query string 传给 `GET /api/features/{id}/tasks`，Server 在 SQL 层过滤后返回结果。筛选参数同时序列化到 URL，页面加载时从 URL 恢复并重新请求。

```
GET /api/features/{id}/tasks?priority=P0&tag=core&status=pending
```

`FeatureService.GetTasks` 签名更新为：

```go
type TaskFilter struct {
    Priorities []string // ["P0","P1"]，空表示不过滤
    Tags       []string // ["core","api"]，空表示不过滤
    Statuses   []string // ["pending","in_progress"]，空表示不过滤
}

// FeatureService
type FeatureService interface {
    ListByProject(ctx context.Context, projectID int64) ([]FeatureSummary, error)
    GetTasks(ctx context.Context, featureID int64, filter TaskFilter) ([]Task, error)
}

## Security Considerations

V1 无认证/授权，API 开放访问。

| 风险 | 缓解措施 |
|------|---------|
| 文件上传恶意内容 | 限制文件大小 ≤ 5MB，仅接受 .json/.md |
| SQL 注入 | 全部使用参数化查询（sqlx） |
| 路径遍历 | push 时对文件路径做白名单校验 |
| HTTPS | 部署时由反向代理（nginx/caddy）处理 TLS |

## Open Questions

- [x] 数据库选型：SQLite（本地）+ PostgreSQL（服务端），`DB_DRIVER` 切换
- [x] 乐观锁方案：version 字段 CAS
- [x] task-cli 集成：`TASK_SERVER_URL` 环境变量
- [x] 部署方式：裸二进制
- [x] Web UI 静态文件内嵌到 Go 二进制（`embed.FS`），单文件部署，无需额外静态文件服务

## Appendix

### Alternatives Considered

| 方案 | Pros | Cons | 未选原因 |
|------|------|------|---------|
| Node.js + TypeScript Server | 前后端同语言 | 运行时依赖 Node | 用户选择 Go |
| PostgreSQL only | 生产级并发 | 需要额外部署 | 个人使用场景 SQLite 足够 |
| SELECT FOR UPDATE 悲观锁 | 实现简单 | 需要事务，SQLite 并发差 | version CAS 更轻量 |
| Next.js（前后端合并） | 减少部署复杂度 | V1 不需要 SSR | 用户选择 React + Vite |
