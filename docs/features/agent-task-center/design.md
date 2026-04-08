# Technical Design: Agent Task Center

## Overview

基于 PRD 的三层结构（Project → Feature → Task），构建集中化任务管理服务。

**技术栈**

| 层 | 技术 |
|----|------|
| 后端 | Go + Gin + GORM |
| 前端 | Vite + Vue3 + Pinia + Element Plus |
| CLI | Go（Monorepo） |
| 存储 | SQLite（开发）/ PostgreSQL（生产）|

## 项目结构

```
agent-task-center/
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── api/
│   │   │   ├── agent/        # /api/agent/ 路由 handlers
│   │   │   └── web/          # /api/ 路由 handlers
│   │   ├── model/            # GORM 数据模型
│   │   ├── service/          # 业务逻辑
│   │   └── store/            # Repository 模式
│   ├── pkg/config/
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── api/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── stores/
│   │   └── router/
│   └── package.json
├── cli/
│   ├── cmd/
│   ├── internal/
│   │   ├── local/
│   │   └── remote/
│   └── go.mod
└── docs/
```

## 数据模型

### ID + Key 并存原则

所有实体同时拥有：
- `ID int64`：自增主键，数据库外键关联，`json:"-"` 不暴露给 API
- `Key string`：语义化业务标识符，用于所有 API 路径和 CLI 参数

| 实体 | Key 格式 | 示例 |
|------|---------|------|
| Project | 用户指定 slug | `agent-task-center` |
| Feature | `feat-{N}-{slug}` | `feat-1-project-management` |
| Task | `{feature_key}/{task_id}` | `feat-1-project-management/1.1` |
| FeatureAttachment | `{feature_key}/attachments/{filename}` | `feat-1-project-management/attachments/design.md` |
| ExecutionRecord | `{task_key}/records/{timestamp}` | `feat-1-project-management/1.1/records/20260408T103000` |

### GORM 模型定义

```go
// backend/internal/model/

type Project struct {
    ID          int64     `gorm:"primaryKey;autoIncrement" json:"-"`
    Key         string    `gorm:"uniqueIndex;not null"     json:"key"`
    Name        string    `gorm:"not null"                 json:"name"`
    Description string    `                                json:"description"`
    CreatedAt   time.Time `                                json:"created_at"`
    UpdatedAt   time.Time `                                json:"updated_at"`
}

type Feature struct {
    ID          int64     `gorm:"primaryKey;autoIncrement"      json:"-"`
    Key         string    `gorm:"uniqueIndex;not null"          json:"key"`
    ProjectID   int64     `gorm:"not null;index"                json:"-"`
    ProjectKey  string    `gorm:"not null"                      json:"project_key"`
    Name        string    `gorm:"not null"                      json:"name"`
    Description string    `                                     json:"description"`
    Status      string    `gorm:"default:pending"               json:"status"`
    CreatedAt   time.Time `                                     json:"created_at"`
    UpdatedAt   time.Time `                                     json:"updated_at"`
}

// FeatureAttachment: Data 以 BLOB 存储，单文件限制 50MB
type FeatureAttachment struct {
    ID          int64     `gorm:"primaryKey;autoIncrement"      json:"-"`
    Key         string    `gorm:"uniqueIndex;not null"          json:"key"`
    FeatureID   int64     `gorm:"not null;index"                json:"-"`
    FeatureKey  string    `gorm:"not null"                      json:"feature_key"`
    Name        string    `gorm:"not null"                      json:"name"`
    ContentType string    `                                     json:"content_type"`
    Data        []byte    `gorm:"type:blob"                     json:"-"`
    FileSize    int64     `                                     json:"file_size"`
    CreatedAt   time.Time `                                     json:"created_at"`
    UpdatedAt   time.Time `                                     json:"updated_at"`
}

type Task struct {
    ID            int64      `gorm:"primaryKey;autoIncrement"     json:"-"`
    Key           string     `gorm:"uniqueIndex;not null"         json:"key"`
    TaskID        string     `gorm:"not null"                     json:"task_id"`   // 如 "1.1"，Feature 内唯一
    ProjectID     int64      `gorm:"not null;index"               json:"-"`
    ProjectKey    string     `gorm:"not null"                     json:"project_key"`
    FeatureID     int64      `gorm:"not null;index"               json:"-"`
    FeatureKey    string     `gorm:"not null"                     json:"feature_key"`
    Title         string     `gorm:"not null"                     json:"title"`
    Priority      string     `gorm:"default:P2"                   json:"priority"`
    Status        string     `gorm:"default:pending"              json:"status"`
    Dependencies  string     `                                    json:"dependencies"` // JSON 数组，存 task key 列表
    AgentID       *string    `                                    json:"agent_id,omitempty"`
    Tags          string     `                                    json:"tags,omitempty"` // JSON 数组
    EstimatedTime string     `                                    json:"estimated_time,omitempty"`
    Version       int        `gorm:"default:0"                    json:"-"` // 乐观锁版本号
    CreatedAt     time.Time  `                                    json:"created_at"`
    UpdatedAt     time.Time  `                                    json:"updated_at"`
    StartedAt     *time.Time `                                    json:"started_at,omitempty"`
    CompletedAt   *time.Time `                                    json:"completed_at,omitempty"`
}

// TaskContent: Data 以 BLOB 存储，1:1 关联 Task
type TaskContent struct {
    ID          int64     `gorm:"primaryKey;autoIncrement" json:"-"`
    TaskID      int64     `gorm:"uniqueIndex;not null"     json:"-"`
    TaskKey     string    `gorm:"not null"                 json:"task_key"`
    ContentType string    `                                json:"content_type"`
    Data        []byte    `gorm:"type:blob"                json:"-"`
    FileSize    int64     `                                json:"file_size"`
    CreatedAt   time.Time `                                json:"created_at"`
    UpdatedAt   time.Time `                                json:"updated_at"`
}

type ExecutionRecord struct {
    ID                 int64     `gorm:"primaryKey;autoIncrement"     json:"-"`
    Key                string    `gorm:"uniqueIndex;not null"         json:"key"`
    TaskID             int64     `gorm:"not null;index"               json:"-"`
    TaskKey            string    `gorm:"not null"                     json:"task_key"`
    AgentID            string    `                                    json:"agent_id"`
    Status             string    `                                    json:"status"`
    Summary            string    `                                    json:"summary"`
    FilesCreated       string    `                                    json:"files_created,omitempty"`   // JSON
    FilesModified      string    `                                    json:"files_modified,omitempty"`  // JSON
    KeyDecisions       string    `                                    json:"key_decisions,omitempty"`   // JSON
    TestsPassed        int       `                                    json:"tests_passed"`
    TestsFailed        int       `                                    json:"tests_failed"`
    Coverage           float64   `                                    json:"coverage"`
    AcceptanceCriteria string    `                                    json:"acceptance_criteria,omitempty"` // JSON
    CreatedAt          time.Time `                                    json:"created_at"`
}
```

## 后端架构

### 分层设计

```
HTTP Request
    ↓
api/agent/ 或 api/web/  — 参数绑定、响应格式化
    ↓
service/                — 业务逻辑、并发控制
    ↓
store/ (Repository)     — GORM 查询封装
    ↓
DB (SQLite / PostgreSQL)
```

### Store 接口（Repository 层）

```go
// internal/store/task.go
type TaskStore interface {
    FindByKey(key string) (*model.Task, error)
    FindByFeature(featureKey string, status ...string) ([]model.Task, error)
    Create(task *model.Task) error
    UpdateStatus(key, status string, version int) (rowsAffected int64, err error)
    UpdateClaim(key, agentID string, version int) (rowsAffected int64, err error)
}

// internal/store/feature.go
type FeatureStore interface {
    FindByKey(key string) (*model.Feature, error)
    FindByProject(projectKey string) ([]model.Feature, error)
    Create(feature *model.Feature) error
    Update(feature *model.Feature) error
    Delete(key string) error
}
```

service 层依赖接口而非具体实现，便于单元测试时 mock。

### 双套 API 路由

**Agent API** — key 寻址，面向 task-cli 和 AI agent

```
GET    /api/agent/projects/:key
GET    /api/agent/projects/:key/features
GET    /api/agent/features/:key
GET    /api/agent/features/:key/attachments
GET    /api/agent/features/:key/attachments/:name
PUT    /api/agent/features/:key/attachments/:name    # task feature push

GET    /api/agent/features/:key/tasks                # 任务列表（含依赖状态）
GET    /api/agent/features/:key/tasks/next           # 下一个可认领任务
POST   /api/agent/features/:key/import               # 导入 index.json
GET    /api/agent/tasks/:key                         # key 含 "/" 需 URL encode
GET    /api/agent/tasks/:key/content
POST   /api/agent/tasks/:key/claim
PUT    /api/agent/tasks/:key/status
POST   /api/agent/tasks/:key/records
```

> Task key 含 `/`（如 `feat-1/1.1`），路由参数使用 `*key` 通配符捕获，或要求调用方 URL encode（`feat-1%2F1.1`）。推荐使用通配符方案，对 agent 更友好。

**Web UI API** — ID 寻址，面向 Vue3 前端

```
GET    /api/projects
POST   /api/projects
GET    /api/projects/:id
PUT    /api/projects/:id
DELETE /api/projects/:id

GET    /api/projects/:id/features
POST   /api/projects/:id/features
GET    /api/features/:id
PUT    /api/features/:id
DELETE /api/features/:id

GET    /api/features/:id/attachments
POST   /api/features/:id/attachments        # multipart/form-data
GET    /api/features/:id/attachments/:aid
PUT    /api/features/:id/attachments/:aid
DELETE /api/features/:id/attachments/:aid

GET    /api/features/:id/tasks
POST   /api/features/:id/tasks
GET    /api/tasks/:id
PUT    /api/tasks/:id
DELETE /api/tasks/:id
GET    /api/tasks/:id/content
PUT    /api/tasks/:id/content
PUT    /api/tasks/:id/status
GET    /api/tasks/:id/records

GET    /api/agents
```

### 并发认领（claim）

使用乐观锁（`version` 字段）防止多 agent 同时认领同一任务：

```go
// service/task.go
func (s *TaskService) Claim(taskKey, agentID string) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        var task model.Task
        if err := tx.Where("key = ? AND status = ? AND agent_id IS NULL", taskKey, "pending").
            First(&task).Error; err != nil {
            return ErrTaskNotAvailable
        }
        now := time.Now()
        result := tx.Model(&task).
            Where("version = ?", task.Version).
            Updates(map[string]any{
                "status":     "in_progress",
                "agent_id":   agentID,
                "started_at": now,
                "version":    task.Version + 1,
            })
        if result.RowsAffected == 0 {
            return ErrTaskAlreadyClaimed // 乐观锁冲突
        }
        return result.Error
    })
}
```

### 获取下一个可认领任务

```go
// GET /api/agent/features/:key/tasks/next
// 返回第一个 status=pending 且所有 dependencies 均已 completed 的任务
func (s *TaskService) NextClaimable(featureKey string) (*model.Task, error) {
    var tasks []model.Task
    s.db.Where("feature_key = ? AND status = ?", featureKey, "pending").Find(&tasks)
    for _, t := range tasks {
        deps := parseDeps(t.Dependencies) // []string of task keys
        if s.allCompleted(deps) {
            return &t, nil
        }
    }
    return nil, ErrNoClaimableTask
}
```

### 存储切换

```go
// pkg/config/config.go
if strings.HasPrefix(cfg.DatabaseURL, "postgres://") {
    db, err = gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
} else {
    db, err = gorm.Open(sqlite.Open(cfg.DatabaseURL), &gorm.Config{})
}
```

### 日志中间件

所有 API 请求记录结构化日志（满足 PRD NFR 可观测性要求）：

```go
// internal/middleware/logger.go
func Logger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        slog.Info("request",
            "method",  c.Request.Method,
            "path",    c.Request.URL.Path,
            "status",  c.Writer.Status(),
            "latency", time.Since(start).String(),
        )
    }
}

// cmd/server/main.go
r := gin.New()
r.Use(middleware.Logger())
```

使用 Go 标准库 `log/slog`，输出 JSON 格式，无额外依赖。

## 错误处理

### 统一错误响应格式

所有错误响应使用语义化 `code`（点分隔字符串），不使用数字错误码：

```go
// pkg/apierr/apierr.go
type APIError struct {
    Code    string         `json:"code"`    // 如 "task.already_claimed"
    Message string         `json:"message"` // 含上下文的可读描述
    Details map[string]any `json:"details,omitempty"`
}
```

**示例响应：**

```json
// 409 - 任务已被认领
{
  "code": "task.already_claimed",
  "message": "Task 'feat-1-project-management/1.1' is already claimed by agent 'claude-01'. Use POST /api/agent/tasks/feat-1-project-management%2F1.1/status to update its status.",
  "details": { "task_key": "feat-1-project-management/1.1", "claimed_by": "claude-01" }
}

// 404 - 资源不存在
{
  "code": "feature.not_found",
  "message": "Feature 'feat-99-unknown' does not exist. Use GET /api/agent/projects/my-project/features to list available features.",
  "details": { "feature_key": "feat-99-unknown" }
}

// 409 - 无可认领任务
{
  "code": "task.none_claimable",
  "message": "No claimable tasks in feature 'feat-1-project-management'. All pending tasks have unmet dependencies.",
  "details": { "feature_key": "feat-1-project-management", "pending_count": 3 }
}
```

**HTTP 状态码映射：**

| 状态码 | 场景 |
|--------|------|
| 400 | 参数错误（`request.invalid`） |
| 404 | 资源不存在（`*.not_found`） |
| 409 | 并发冲突（`task.already_claimed`、`task.none_claimable`） |
| 413 | 文件超过 50MB（`file.too_large`） |
| 500 | 服务内部错误（`server.internal_error`） |

## 前端架构

### 页面结构

```
/                    → 重定向到 /projects
/projects            → 项目列表
/projects/:id        → 项目详情（Feature 列表）
/features/:id        → Feature 详情（看板 + 依赖图 + 附件）
/tasks/:id           → 任务详情 + 执行记录
```

### 组件清单

```
pages/
├── ProjectList.vue          # /projects
├── ProjectDetail.vue        # /projects/:id
├── FeatureDetail.vue        # /features/:id（组合 TaskBoard + TaskDependencyGraph + FilterBar）
└── TaskDetail.vue           # /tasks/:id（组合 RecordTimeline）

components/
├── TaskBoard.vue            # Kanban 四列看板
├── TaskCard.vue             # 单个任务卡片（优先级 badge、agent_id、依赖数）
├── TaskDependencyGraph.vue  # D3.js DAG 有向图
├── FilterBar.vue            # 筛选条件栏（与 URL query params 双向同步）
└── RecordTimeline.vue       # 执行记录垂直时间线
```

### Pinia Store

```
stores/
├── project.ts    # 项目列表、当前项目
├── feature.ts    # Feature 列表、当前 Feature
├── task.ts       # 任务列表、看板操作（调用 PUT /api/tasks/:id/status）
├── filter.ts     # 筛选条件（project/feature/priority[]/tags[]/status[]），与 URL query params 双向同步
└── agent.ts      # Agent 活动状态
```

### 任务看板拖拽

使用 `vue-draggable-plus`（基于 SortableJS）。拖拽结束后调用 `PUT /api/tasks/:id/status`。

### 任务依赖图（D3.js DAG）

数据来源：`GET /api/features/:id/tasks`，取每个 task 的 `dependencies` 字段构建有向图。

```typescript
// components/TaskDependencyGraph.vue
interface GraphNode {
  id: string        // task key
  taskId: string    // 短 ID，如 "1.1"
  title: string
  status: 'pending' | 'in_progress' | 'completed' | 'blocked'
}
interface GraphEdge {
  source: string    // 被依赖任务 key
  target: string    // 依赖方任务 key
}
```

节点颜色映射：`pending=#9ca3af`、`in_progress=#3b82f6`、`completed=#22c55e`、`blocked=#ef4444`

点击节点使用 `router.push('/tasks/:id')` 跳转。

### 筛选状态（filterStore + URL 同步）

```typescript
// stores/filter.ts
interface FilterState {
  projectId: string | null
  featureId: string | null
  priority: string[]   // ['P0', 'P1', 'P2'] 子集
  tags: string[]
  status: string[]
}

// 初始化：从 route.query 读取
// 写入：watch(filterState, () => router.replace({ query: serialize(filterState) }))
```

## CLI 设计

### 模式切换

```go
// cli/internal/client.go
type TaskClient interface {
    NextClaimable(featureKey string) (*Task, error)
    Claim(taskKey, agentID string) error
    UpdateStatus(taskKey, status string) error
    GetContent(taskKey string) ([]byte, string, error)
    CreateRecord(taskKey string, record RecordInput) error
    FeaturePush(featureKey, filePath string) error
    FeaturePull(featureKey, name, destPath string) error
}

func NewClient() TaskClient {
    if url := os.Getenv("TASK_REMOTE_URL"); url != "" {
        return &RemoteClient{
            BaseURL:    url,
            ProjectKey: os.Getenv("TASK_PROJECT_KEY"),
            FeatureKey: os.Getenv("TASK_FEATURE_KEY"),
        }
    }
    return &LocalClient{RootDir: os.Getenv("TASK_ROOT_DIR")}
}
```

### 环境变量

| 变量 | 说明 |
|------|------|
| `TASK_REMOTE_URL` | 远程服务地址，设置后切换远程模式 |
| `TASK_PROJECT_KEY` | 当前项目 key（如 `agent-task-center`） |
| `TASK_FEATURE_KEY` | 当前 Feature key（如 `feat-1-project-management`） |
| `TASK_ROOT_DIR` | 本地模式根目录（默认当前目录） |

### 新增命令

```
task get-content <task-key>              # 获取任务内容，输出到 stdout
task feature push [feature-key] <file>  # 上传本地文件到服务端
task feature pull [feature-key] <name>  # 下载附件到本地（默认当前目录）
```

`task claim` 远程模式实现：调用 `GET /api/agent/features/{key}/tasks/next` 获取任务，再调用 `POST /api/agent/tasks/{key}/claim` 认领。

## 数据库迁移

使用 GORM AutoMigrate 在启动时自动建表。生产环境可替换为 `golang-migrate`。

Key 字段加 `uniqueIndex`，确保语义唯一性约束在数据库层强制执行。

## 测试策略

| 层 | 类型 | 工具 | 覆盖率目标 | 重点 |
|----|------|------|------------|------|
| service/ | 单元测试 | Go testing + testify/mock | ≥ 80% | 并发 claim 乐观锁逻辑 |
| api/agent/ | 集成测试 | httptest + SQLite 内存库 | 覆盖所有错误路径 | key 路由、错误响应格式 |
| api/web/ | 集成测试 | httptest + SQLite 内存库 | 覆盖所有错误路径 | CRUD、状态流转 |
| CLI | 单元测试 | Go testing + testify/mock | ≥ 70% | mock TaskClient 接口 |
| 前端 | 单元测试 | Vitest | ≥ 70% | filterStore URL 同步、task store 状态流转 |
