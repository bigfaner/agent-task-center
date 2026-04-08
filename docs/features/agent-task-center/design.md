# Technical Design: Agent Task Center

## Overview

基于 PRD 的三层结构（Project → Feature → Task），构建集中化任务管理服务。

**技术栈**

| 层 | 技术 |
|----|------|
| 后端 | Go + Gin + GORM |
| 前端 | Vite + Vue3 + Pinia + Element Plus |
| CLI | Go（纳入 Monorepo） |
| 存储 | SQLite（开发）/ PostgreSQL（生产）|

## 项目结构

```
agent-task-center/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── api/          # Gin handlers + routes
│   │   ├── model/        # GORM 数据模型
│   │   ├── service/      # 业务逻辑
│   │   └── store/        # DB 访问层（Repository 模式）
│   ├── pkg/
│   │   └── config/       # 配置（env vars）
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── api/          # HTTP 客户端
│   │   ├── components/
│   │   ├── pages/
│   │   ├── stores/       # Pinia stores
│   │   └── router/
│   ├── index.html
│   └── package.json
├── cli/
│   ├── cmd/              # cobra 命令定义
│   ├── internal/
│   │   ├── local/        # 本地文件模式
│   │   └── remote/       # 远程 HTTP 模式
│   └── go.mod
└── docs/
```

## 后端架构

### 分层设计

```
HTTP Request
    ↓
api/ (Gin Handler)     — 参数绑定、响应格式化
    ↓
service/               — 业务逻辑、并发控制
    ↓
store/ (Repository)    — GORM 查询封装
    ↓
DB (SQLite / PostgreSQL)
```

### 数据模型

```go
// backend/internal/model/

type Project struct {
    ID          string    `gorm:"primaryKey" json:"id"`
    Name        string    `gorm:"not null"   json:"name"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type Feature struct {
    ID          string    `gorm:"primaryKey" json:"id"`
    ProjectID   string    `gorm:"not null;index" json:"project_id"`
    Name        string    `gorm:"not null"   json:"name"`
    Description string    `json:"description"`
    Status      string    `gorm:"default:pending" json:"status"` // pending/in_progress/completed
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// FeatureAttachment 统一存储 Feature 关联文件（design.md、.pen、图片等）
// Data 字段以 BLOB 存储，SQLite: BLOB，PostgreSQL: BYTEA，单文件限制 50MB
type FeatureAttachment struct {
    ID          string    `gorm:"primaryKey" json:"id"`
    FeatureID   string    `gorm:"not null;index" json:"feature_id"`
    Name        string    `gorm:"not null"   json:"name"`         // 文件名
    ContentType string    `json:"content_type"`                   // MIME 类型
    Data        []byte    `gorm:"type:blob"  json:"-"`            // 二进制内容
    FileSize    int64     `json:"file_size"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type Task struct {
    ID            string     `gorm:"primaryKey" json:"id"`
    ProjectID     string     `gorm:"not null;index" json:"project_id"`
    FeatureID     string     `gorm:"not null;index" json:"feature_id"`
    Title         string     `gorm:"not null"   json:"title"`
    Priority      string     `gorm:"default:P2" json:"priority"`  // P0/P1/P2
    Status        string     `gorm:"default:pending" json:"status"`
    Dependencies  string     `json:"dependencies"` // JSON 数组序列化存储
    AgentID       *string    `json:"agent_id,omitempty"`
    Tags          string     `json:"tags,omitempty"` // JSON 数组序列化存储
    EstimatedTime string     `json:"estimated_time,omitempty"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
    StartedAt     *time.Time `json:"started_at,omitempty"`
    CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// TaskContent 以 BLOB 存储任务详细内容，SQLite: BLOB，PostgreSQL: BYTEA
type TaskContent struct {
    ID          string    `gorm:"primaryKey" json:"id"`
    TaskID      string    `gorm:"uniqueIndex" json:"task_id"` // 1:1
    ContentType string    `json:"content_type"`               // 通常 text/markdown
    Data        []byte    `gorm:"type:blob"  json:"-"`
    FileSize    int64     `json:"file_size"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type ExecutionRecord struct {
    ID                 string               `gorm:"primaryKey" json:"id"`
    TaskID             string               `gorm:"not null;index" json:"task_id"`
    AgentID            string               `json:"agent_id"`
    Status             string               `json:"status"`
    Summary            string               `json:"summary"`
    FilesCreated       string               `json:"files_created,omitempty"`  // JSON
    FilesModified      string               `json:"files_modified,omitempty"` // JSON
    KeyDecisions       string               `json:"key_decisions,omitempty"`  // JSON
    TestsPassed        int                  `json:"tests_passed"`
    TestsFailed        int                  `json:"tests_failed"`
    Coverage           float64              `json:"coverage"`
    AcceptanceCriteria string               `json:"acceptance_criteria,omitempty"` // JSON
    CreatedAt          time.Time            `json:"created_at"`
}
```

### API 路由

```
POST   /api/projects
GET    /api/projects
GET    /api/projects/:id
DELETE /api/projects/:id

POST   /api/projects/:id/features
GET    /api/projects/:id/features
GET    /api/features/:id
PUT    /api/features/:id
DELETE /api/features/:id

GET    /api/features/:id/attachments
POST   /api/features/:id/attachments        # multipart/form-data
GET    /api/features/:id/attachments/:aid   # 返回原始字节 + Content-Type
PUT    /api/features/:id/attachments/:aid   # multipart/form-data
DELETE /api/features/:id/attachments/:aid

GET    /api/features/:id/tasks
POST   /api/features/:id/tasks
POST   /api/features/:id/import             # 导入 index.json

GET    /api/tasks/:id
PUT    /api/tasks/:id
DELETE /api/tasks/:id
GET    /api/tasks/:id/content               # 返回原始字节 + Content-Type
PUT    /api/tasks/:id/content               # multipart/form-data
POST   /api/tasks/:id/claim
PUT    /api/tasks/:id/status
GET    /api/tasks/:id/records
POST   /api/tasks/:id/records

GET    /api/agents
```

### 并发认领（claim）

使用数据库行级锁防止多 agent 同时认领同一任务：

```go
// service/task.go
func (s *TaskService) Claim(taskID, agentID string) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        var task model.Task
        // SELECT ... FOR UPDATE（PostgreSQL）/ 事务内更新（SQLite）
        if err := tx.Set("gorm:query_option", "FOR UPDATE").
            Where("id = ? AND status = ? AND agent_id IS NULL", taskID, "pending").
            First(&task).Error; err != nil {
            return ErrTaskNotAvailable
        }
        now := time.Now()
        return tx.Model(&task).Updates(map[string]any{
            "status":     "in_progress",
            "agent_id":   agentID,
            "started_at": now,
        }).Error
    })
}
```

> SQLite 不支持 `SELECT FOR UPDATE`，改用事务 + 乐观锁（检查 updated_at）。

### 存储切换

通过环境变量 `DATABASE_URL` 区分：

```go
// pkg/config/config.go
if strings.HasPrefix(cfg.DatabaseURL, "postgres://") {
    db, err = gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
} else {
    db, err = gorm.Open(sqlite.Open(cfg.DatabaseURL), &gorm.Config{})
}
```

## 前端架构

### 页面结构

```
/                    → 重定向到 /projects
/projects            → 项目列表
/projects/:id        → 项目详情（Feature 列表）
/features/:id        → Feature 详情（任务看板 + 任务树 + 附件）
/features/:id/board  → 任务看板（Kanban）
/features/:id/tree   → 任务树
/tasks/:id           → 任务详情 + 执行记录
```

### Pinia Store 结构

```
stores/
├── project.ts    # 项目列表、当前项目
├── feature.ts    # Feature 列表、当前 Feature
├── task.ts       # 任务列表、任务详情、看板操作
└── agent.ts      # Agent 活动状态
```

### 任务看板拖拽

使用 `@vueuse/core` + 原生 HTML5 Drag & Drop，或引入 `vue-draggable-plus`（基于 SortableJS）。拖拽结束后调用 `PUT /api/tasks/:id/status`。

## CLI 设计

### 模式切换

```go
// cli/internal/client.go
type TaskClient interface {
    Claim(taskID, agentID string) error
    UpdateStatus(taskID, status string) error
    GetContent(taskID string) ([]byte, string, error)
    CreateRecord(taskID string, record RecordInput) error
    FeaturePush(featureID, filePath string) error
    FeaturePull(featureID, name, destPath string) error
}

func NewClient() TaskClient {
    if url := os.Getenv("TASK_REMOTE_URL"); url != "" {
        return &RemoteClient{BaseURL: url, ProjectID: os.Getenv("TASK_PROJECT_ID")}
    }
    return &LocalClient{RootDir: os.Getenv("TASK_ROOT_DIR")}
}
```

### 新增命令

```
task get-content <task-id>              # 获取任务内容，输出到 stdout
task feature push <feature-id> <file>  # 上传本地文件到服务端
task feature pull <feature-id> <name>  # 下载附件到本地（默认当前目录）
```

### 环境变量

| 变量 | 说明 |
|------|------|
| `TASK_REMOTE_URL` | 远程服务地址，设置后切换远程模式 |
| `TASK_PROJECT_ID` | 当前项目 ID |
| `TASK_FEATURE_ID` | 当前 Feature ID（可选） |
| `TASK_ROOT_DIR` | 本地模式根目录（默认当前目录） |

## 数据库迁移

使用 GORM AutoMigrate 在启动时自动建表，生产环境可替换为 `golang-migrate`。

## 错误处理

统一响应格式：

```json
{ "error": "task not available", "code": "TASK_NOT_AVAILABLE" }
```

HTTP 状态码：
- `400` 参数错误
- `404` 资源不存在
- `409` 并发冲突（任务已被认领）
- `413` 文件超过 50MB 限制
- `500` 服务内部错误

## 测试策略

| 层 | 策略 |
|----|------|
| service/ | 单元测试，mock store 接口 |
| api/ | 集成测试，使用 httptest + SQLite 内存库 |
| CLI | 单元测试，mock TaskClient 接口 |
| 前端 | Vitest 单元测试核心 store 逻辑 |
