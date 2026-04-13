---
created: 2026-04-13
related: design/tech-design.md
---

# API Handbook: Agent Task Center (V1)

## API Overview

双路由前缀设计：

| 前缀 | 面向 | 寻址方式 |
|------|------|---------|
| `/api/` | Web UI + 文件上传 | 按数字 ID |
| `/api/agent/` | task-cli (Agent) | 按语义 key（project/feature slug） |

所有请求/响应均为 JSON（除文件上传为 multipart）。V1 无认证。

---

## Web UI Endpoints

### List Projects

**Method**: `GET`
**Path**: `/api/projects`

#### Query Parameters

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| search | string | No | 项目名称模糊搜索 |
| page | int | No | 页码，默认 1 |
| pageSize | int | No | 每页条数，默认 20 |

#### Response (200)

```json
{
  "items": [
    {
      "id": 1,
      "name": "agent-task-center",
      "featureCount": 3,
      "taskTotal": 24,
      "completionRate": 62.5,
      "updatedAt": "2026-04-12T14:30:00Z"
    }
  ],
  "total": 10,
  "page": 1,
  "pageSize": 20
}
```

#### Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 500 | `internal_error` | 数据库查询失败 |

---

### Get Project Detail

**Method**: `GET`
**Path**: `/api/projects/{id}`

#### Response (200)

```json
{
  "id": 1,
  "name": "agent-task-center",
  "proposals": [
    {
      "id": 1,
      "slug": "agent-task-center",
      "title": "Proposal: Agent Task Center",
      "createdAt": "2026-04-12T10:00:00Z",
      "featureCount": 1
    }
  ],
  "features": [
    {
      "id": 1,
      "slug": "agent-task-center",
      "name": "Agent Task Center",
      "status": "in-progress",
      "completionRate": 62.5,
      "updatedAt": "2026-04-12T14:30:00Z"
    }
  ]
}
```

#### Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 404 | `not_found` | 项目不存在 |

---

### Get Feature Tasks (Kanban)

**Method**: `GET`
**Path**: `/api/features/{id}/tasks`

支持服务端过滤，筛选参数同时序列化到 URL 供分享。

#### Query Parameters

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| priority | string | No | 逗号分隔，如 `P0,P1` |
| tag | string | No | 逗号分隔，如 `core,api` |
| status | string | No | 逗号分隔，如 `pending,in_progress` |

#### Response (200)

```json
{
  "featureId": 1,
  "featureName": "Agent Task Center",
  "tasks": [
    {
      "id": 101,
      "taskId": "1.1",
      "title": "初始化项目结构",
      "status": "completed",
      "priority": "P0",
      "tags": ["core", "setup"],
      "claimedBy": "agent-01",
      "dependencies": []
    }
  ]
}
```

#### Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 404 | `not_found` | Feature 不存在 |

---

### Get Task Detail

**Method**: `GET`
**Path**: `/api/tasks/{id}`

#### Response (200)

```json
{
  "id": 101,
  "taskId": "1.1",
  "title": "初始化项目结构",
  "description": "## 任务描述\n...",
  "status": "completed",
  "priority": "P0",
  "tags": ["core", "setup"],
  "claimedBy": "agent-01",
  "dependencies": ["1.0"],
  "createdAt": "2026-04-12T10:00:00Z",
  "updatedAt": "2026-04-12T14:30:00Z"
}
```

#### Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 404 | `not_found` | 任务不存在 |

---

### List Task Execution Records

**Method**: `GET`
**Path**: `/api/tasks/{id}/records`

按时间倒序，支持懒加载。

#### Query Parameters

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| page | int | No | 页码，默认 1 |
| pageSize | int | No | 每页条数，默认 10 |

#### Response (200)

```json
{
  "items": [
    {
      "id": 1,
      "agentId": "agent-01",
      "summary": "实现了项目初始化脚手架",
      "filesCreated": ["server/go.mod", "web/package.json"],
      "filesModified": [],
      "keyDecisions": ["使用 chi 作为 HTTP router"],
      "testsPassed": 12,
      "testsFailed": 0,
      "coverage": 85.6,
      "acceptanceCriteria": [
        { "criterion": "项目可以编译运行", "met": true }
      ],
      "createdAt": "2026-04-12T14:30:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "pageSize": 10
}
```

---

### Get Document Content

**Method**: `GET`
**Path**: `/api/proposals/{id}/content`
**Path**: `/api/features/{id}/content`

#### Response (200)

```json
{
  "title": "Proposal: Agent Task Center",
  "content": "# Proposal: Agent Task Center\n\n...",
  "relatedFeatures": [
    { "id": 1, "name": "Agent Task Center", "slug": "agent-task-center" }
  ],
  "relatedTasks": []
}
```

---

### Upload File

**Method**: `POST`
**Path**: `/api/upload`
**Content-Type**: `multipart/form-data`

#### Request

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| project | string (query) | Yes | 项目名称，不存在则自动创建 |
| feature | string (query) | No | Feature slug，上传 index.json 时必填 |
| file | file | Yes | .json 或 .md 文件，≤5MB |

#### Response (200)

```json
{
  "filename": "index.json",
  "created": 5,
  "updated": 3,
  "skipped": 0,
  "message": "新增 5 个任务，更新 3 个任务"
}
```

#### Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 400 | `invalid_file` | 文件格式不支持或内容无效 |
| 400 | `missing_field` | task_id 或 title 缺失 |
| 413 | `file_too_large` | 文件超过 5MB |

---

## Agent Endpoints

### Claim Next Task

**Method**: `POST`
**Path**: `/api/agent/claim`

认领当前 feature 下优先级最高、依赖已满足的 pending 任务。使用 version CAS 乐观锁，内部最多重试 3 次。

#### Request Body

```json
{
  "projectName": "agent-task-center",
  "featureSlug": "agent-task-center",
  "agentId": "agent-01"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| projectName | string | Yes | 项目名称 |
| featureSlug | string | Yes | Feature slug |
| agentId | string | Yes | Agent 标识符 |

#### Response (200)

```json
{
  "id": 102,
  "taskId": "1.2",
  "title": "实现数据库 schema",
  "description": "## 任务描述\n...",
  "priority": "P0",
  "tags": ["core", "db"],
  "dependencies": ["1.1"]
}
```

#### Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 404 | `no_available_task` | 无可用任务（全部已认领或依赖未满足） |
| 409 | `version_conflict` | 乐观锁重试 3 次后仍冲突（极少见） |

---

### Update Task Status

**Method**: `PATCH`
**Path**: `/api/agent/tasks/{taskId}/status`

`taskId` 为数字 ID（claim 返回的 `id` 字段）。

#### Request Body

```json
{
  "agentId": "agent-01",
  "status": "blocked"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| agentId | string | Yes | 必须与 claimed_by 一致 |
| status | string | Yes | `in_progress` / `blocked` / `pending` |

#### Response (200)

```json
{ "ok": true }
```

#### Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 403 | `unauthorized_agent` | agentId 与 claimed_by 不匹配 |
| 404 | `not_found` | 任务不存在 |
| 400 | `invalid_status` | 非法状态值 |

---

### Submit Execution Record

**Method**: `POST`
**Path**: `/api/agent/tasks/{taskId}/records`

提交执行记录，同时将任务状态更新为 `completed`。

#### Request Body

```json
{
  "agentId": "agent-01",
  "summary": "实现了数据库 schema 和 migration",
  "filesCreated": ["server/internal/db/schema.sql"],
  "filesModified": ["server/go.mod"],
  "keyDecisions": ["使用 golang-migrate 管理迁移"],
  "testsPassed": 8,
  "testsFailed": 0,
  "coverage": 78.5,
  "acceptanceCriteria": [
    { "criterion": "所有表创建成功", "met": true },
    { "criterion": "迁移可回滚", "met": true }
  ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| agentId | string | Yes | 必须与 claimed_by 一致 |
| summary | string | Yes | 执行摘要 |
| filesCreated | string[] | No | 新建文件列表 |
| filesModified | string[] | No | 修改文件列表 |
| keyDecisions | string[] | No | 关键决策 |
| testsPassed | int | No | 通过测试数 |
| testsFailed | int | No | 失败测试数 |
| coverage | float | No | 覆盖率（0-100） |
| acceptanceCriteria | array | No | `{criterion: string, met: bool}` 数组 |

#### Response (200)

```json
{ "recordId": 42, "taskStatus": "completed" }
```

#### Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 403 | `unauthorized_agent` | agentId 与 claimed_by 不匹配 |
| 404 | `not_found` | 任务不存在 |

---

### Get Task Content

**Method**: `GET`
**Path**: `/api/agent/tasks/{taskId}/content`

`taskId` 为字符串格式（如 `1.2`），需配合 project + feature 参数定位。

#### Query Parameters

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| project | string | Yes | 项目名称 |
| feature | string | Yes | Feature slug |

#### Response (200)

```json
{
  "taskId": "1.2",
  "title": "实现数据库 schema",
  "description": "## 任务描述\n\n实现所有数据库表的 DDL...",
  "priority": "P0",
  "tags": ["core", "db"],
  "dependencies": ["1.1"],
  "status": "in_progress",
  "claimedBy": "agent-01"
}
```

---

### Push Docs Directory

**Method**: `POST`
**Path**: `/api/push`
**Content-Type**: `multipart/form-data`

批量上传 docs/ 目录下的所有文件，等同于批量执行 `/api/upload`。

#### Request

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| project | string (query) | Yes | 项目名称 |
| files | file[] | Yes | 多个文件，文件名需包含相对路径信息 |

Server 根据文件名（`proposal.md` / `manifest.md` / `index.json`）路由到对应解析器。`index.json` 需要从路径中提取 feature slug（`docs/features/<slug>/tasks/index.json`）。

#### Response (200)

```json
{
  "results": [
    { "filename": "proposal.md", "created": 1, "updated": 0 },
    { "filename": "manifest.md", "created": 0, "updated": 1 },
    { "filename": "index.json", "created": 8, "updated": 2 }
  ],
  "totalCreated": 9,
  "totalUpdated": 3
}
```

---

## Data Contracts

### TaskSummary（看板卡片）

```json
{
  "id": 101,
  "taskId": "1.1",
  "title": "string",
  "status": "pending | in_progress | completed | blocked",
  "priority": "P0 | P1 | P2",
  "tags": ["string"],
  "claimedBy": "string | null",
  "dependencies": ["string"]
}
```

### UpsertSummary（上传结果）

```json
{
  "filename": "string",
  "created": 0,
  "updated": 0,
  "skipped": 0,
  "message": "string"
}
```

---

## Error Codes

错误响应格式（同时面向人类和 AI Agent）：

```json
{
  "error": "error_code",
  "message": "简洁描述",
  "hint": "修复建议或下一步操作（可选）"
}
```

| Code | HTTP Status | message | hint |
|------|-------------|---------|------|
| `not_found` | 404 | `"Resource not found"` | `"Check the ID is correct"` |
| `no_available_task` | 404 | `"No tasks available to claim"` | `"All pending tasks are either claimed or have unmet dependencies"` |
| `version_conflict` | 409 | `"Task was claimed by another agent"` | `"Retry claim to get the next available task"` |
| `invalid_file` | 400 | `"Invalid file format"` | `"Only .json and .md files are accepted. index.json must contain task_id and title fields"` |
| `missing_field` | 400 | `"Required field missing: {field}"` | `"Provide the missing field and retry"` |
| `file_too_large` | 413 | `"File exceeds 5MB limit"` | `"Split the file or reduce its size"` |
| `unauthorized_agent` | 403 | `"Task is claimed by a different agent"` | `"Only the agent that claimed this task can update it"` |
| `invalid_status` | 400 | `"Invalid status: {value}"` | `"Valid values: pending, in_progress, blocked"` |
| `internal_error` | 500 | `"Internal server error"` | `"Check server logs for details"` |
