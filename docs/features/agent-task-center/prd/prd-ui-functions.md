---
feature: "Agent Task Center"
---

# Agent Task Center — UI Functions

> Requirements layer: defines WHAT the UI must do. Not HOW it looks (that's ui-design.md).

## UI Scope

| # | UI Surface | Type | Priority |
|---|-----------|------|----------|
| 1 | 项目列表页 | 仪表盘 | P0 |
| 2 | 项目详情页（Proposal/Feature Tab） | 列表页 | P0 |
| 3 | 任务看板页（Feature 级别） | 列表页（Kanban） | P0 |
| 4 | 任务详情页 | 详情页 | P0 |
| 5 | 文档查看页 | 详情页 | P1 |
| 6 | 文件上传 | 弹窗/对话框 | P1 |

---

## UI Function 1: 项目列表页

### Description

全局仪表盘，展示所有项目的概览统计。是用户进入 Task Center 的首页。

### User Interaction Flow

1. 用户打开 Task Center → 自动加载项目列表
2. 浏览项目卡片/行 → 查看各项目完成率
3. 点击项目名称 → 跳转项目详情页
4. 使用搜索框 → 实时过滤项目列表

### Data Requirements

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| 项目名称 | string | Project.name | 可点击链接 |
| Feature 数 | number | 聚合计算 | Feature 计数 |
| 任务总数 | number | 聚合计算 | 所有关联 Task |
| 完成率 | number | 聚合计算 | completed / total × 100% |
| 最近更新 | datetime | Project.updatedAt | 最后数据变更时间 |

### States

| State | Display | Trigger |
|-------|---------|---------|
| Loading | 骨架屏 / 加载动画 | 首次进入 |
| Populated | 项目列表 + 统计数据 | 数据加载完成 |
| Empty | 空状态提示 + 上传引导 | 无任何项目数据 |
| Error | 错误提示 + 重试按钮 | API 请求失败 |

### Validation Rules

- 搜索框输入防抖 300ms
- 完成率显示为百分比，保留 1 位小数

---

## UI Function 2: 项目详情页

### Description

展示单个项目下的 Proposal 列表和 Feature 列表，以 Tab 切换。

### User Interaction Flow

1. 从项目列表点击进入 → 默认显示 Feature Tab
2. 切换到 Proposal Tab → 查看 Proposal 列表
3. 点击 Feature 名称 → 跳转任务看板页
4. 点击 Proposal 标题 → 跳转文档查看页
5. 点击「上传」按钮 → 打开文件上传弹窗

### Data Requirements

**Proposal Tab:**

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| 提案标题 | string | Proposal.title | 可点击链接 |
| 创建时间 | datetime | Proposal.createdAt | |
| 关联 Feature 数 | number | 聚合计算 | |

**Feature Tab:**

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| Feature 名称 | string | Feature.name | 可点击链接 |
| 任务完成率 | number | 聚合计算 | |
| 状态 badge | string | Feature.status | prd/design/tasks/in-progress/done |
| 最近更新 | datetime | Feature.updatedAt | |

### States

| State | Display | Trigger |
|-------|---------|---------|
| Loading | 骨架屏 | 进入页面 |
| Populated | 双 Tab 列表 | 数据加载完成 |
| Empty Tab | 空状态提示 | 当前 Tab 无数据 |

### Validation Rules

- N/A — 只读列表页，无用户输入校验

---

## UI Function 3: 任务看板页

### Description

Feature 级别的 Kanban 看板，按状态分列展示任务。支持筛选。

### User Interaction Flow

1. 从 Feature 列表点击进入 → 加载四列 Kanban
2. 使用筛选器（优先级/标签/状态）→ 动态过滤卡片
3. 点击任务卡片 → 跳转任务详情页
4. 复制当前 URL → 分享带筛选条件的看板视图

### Data Requirements

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| task_id | string | Task.taskId | 任务编号 |
| 标题 | string | Task.title | |
| 优先级 | string | Task.priority | P0(红)/P1(橙)/P2(蓝) |
| 认领者 | string | Task.claimedBy | agent_id 或空 |
| 标签 | string[] | Task.tags | |

### States

| State | Display | Trigger |
|-------|---------|---------|
| Loading | 四列骨架屏 | 进入页面 |
| Populated | 四列 Kanban + 筛选器 | 数据加载完成 |
| Filtered | 仅显示匹配卡片 | 应用筛选条件 |
| Empty Column | 列内空状态 | 该状态无任务 |

### Validation Rules

- 筛选参数序列化到 URL query string（priority=P0&tag=core&status=pending）
- 页面加载时从 URL 恢复筛选状态
- 标签选项从当前 Feature 的所有任务标签动态聚合

---

## UI Function 4: 任务详情页

### Description

展示单个任务的完整信息和执行记录时间线。

### User Interaction Flow

1. 从看板/列表点击任务 → 加载任务详情
2. 查看基本信息区域 → 了解任务状态、描述
3. 滚动到时间线区域 → 浏览执行记录
4. 点击展开按钮 → 查看某条记录的完整详情
5. 点击依赖任务链接 → 跳转到依赖任务详情

### Data Requirements

**基本信息：**

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| task_id | string | Task.taskId | |
| 标题 | string | Task.title | |
| 状态 | string | Task.status | 带 badge 颜色 |
| 优先级 | string | Task.priority | |
| 认领者 | string | Task.claimedBy | |
| 标签 | string[] | Task.tags | |
| 依赖任务 | string[] | Task.dependencies | 可点击链接 |
| 描述 | markdown | Task.description | Markdown 渲染 |

**执行记录（每条）：**

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| 时间戳 | datetime | Record.createdAt | |
| agent_id | string | Record.agentId | |
| summary | string | Record.summary | |
| filesCreated | string[] | Record.filesCreated | 展开后 |
| filesModified | string[] | Record.filesModified | 展开后 |
| keyDecisions | string[] | Record.keyDecisions | 展开后 |
| testsPassed | number | Record.testsPassed | 展开后 |
| testsFailed | number | Record.testsFailed | 展开后 |
| coverage | number | Record.coverage | 展开后 |
| acceptanceCriteria | array | Record.acceptanceCriteria | 展开后，{criterion, met} |

### States

| State | Display | Trigger |
|-------|---------|---------|
| Loading | 骨架屏 | 进入页面 |
| Populated | 基本信息 + 时间线 | 数据加载完成 |
| No Records | 基本信息 + "暂无执行记录" | 任务未被认领/执行 |
| Expanded Record | 记录展开显示完整字段 | 用户点击展开 |

---

## UI Function 5: 文档查看页

### Description

Markdown 渲染展示 Proposal 或 Feature 的文档内容。

### User Interaction Flow

1. 从 Proposal 列表点击标题 → 加载文档内容
2. 浏览 Markdown 渲染 → 使用标题锚点导航
3. 查看页面底部关联列表 → 了解关联 Feature / Task

### Data Requirements

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| 文档标题 | string | 文档内容解析 | |
| 文档内容 | markdown | Proposal/Feature.content | Markdown 渲染 |
| 关联 Feature | array | 关联查询 | 文档底部展示 |
| 关联 Task | array | 关联查询 | 文档底部展示 |

### States

| State | Display | Trigger |
|-------|---------|---------|
| Loading | 加载动画 | 进入页面 |
| Populated | 渲染后的 Markdown + 关联列表 | 数据加载完成 |
| Error | 错误提示 | 文档不存在或解析失败 |

### Validation Rules

- N/A — 只读展示页，无用户输入校验

---

## UI Function 6: 文件上传

### Description

弹窗式文件上传，支持 index.json、proposal.md、manifest.md。

### User Interaction Flow

1. 用户点击「上传」按钮 → 弹出上传对话框
2. 选择或拖拽文件到上传区域
3. 系统校验文件格式 → 显示校验结果
4. 用户确认上传 → 显示上传进度
5. 上传完成 → 显示解析结果摘要（新增/更新 N 个任务等）
6. 用户关闭弹窗 → 列表页自动刷新

### Data Requirements

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| 上传文件 | file | 用户选择 | .json 或 .md |
| 解析摘要 | object | Server 响应 | 新增/更新计数 |

### States

| State | Display | Trigger |
|-------|---------|---------|
| Idle | 上传区域 + 拖拽提示 | 弹窗打开 |
| Validating | 校验中 | 文件选择后 |
| Invalid | 错误提示（格式/内容） | 校验失败 |
| Uploading | 进度条 | 上传中 |
| Success | 解析结果摘要 | 上传并解析成功 |
| Error | 错误信息 + 重试 | 上传或解析失败 |

### Validation Rules

- 仅接受 .json 和 .md 后缀文件
- index.json 必须是合法 JSON，包含 task_id 和 title 字段
- 文件大小上限 5MB
