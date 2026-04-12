---
feature: "agent-task-center-v2"
---

# Agent Task Center V2 — UI Functions

> Requirements layer: defines WHAT the UI must do. Not HOW it looks (that's ui-design.md).

## UI Scope

| # | UI Surface | Type | Priority |
|---|-----------|------|----------|
| 1 | 任务依赖图（DAG） | 图形页 | P0 |
| 2 | 看板依赖视图 + 拖拽 | 列表页（看板新模式） | P0 |
| 3 | 全局 Agent 活动面板 | 列表页 | P0 |
| 4 | Agent 执行历史轨迹 | 详情页 | P1 |
| 5 | 任务级 Agent 信息增强 | 详情页（agent-task-center 增强） | P1 |

---

## UI Function 1: 任务依赖图（DAG）

### Description

Feature 级别的任务依赖关系 DAG 可视化，以图形拓扑展示任务间的依赖链路。

### User Interaction Flow

1. 用户从看板页切换到「依赖图」视图 → 加载 DAG 图形
2. 浏览图形 → 通过节点颜色快速识别任务状态
3. 悬停节点 → Tooltip 显示任务摘要（task_id、标题、状态、认领者）
4. 点击节点 → 跳转对应任务详情页
5. 缩放/平移 → 浏览大型依赖图

### Data Requirements

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| 节点 task_id | string | Task.taskId | 节点标签 |
| 节点标题 | string | Task.title | 节点标签 |
| 节点状态颜色 | enum | Task.status | pending=灰/in_progress=蓝/completed=绿/blocked=红 |
| 节点优先级 | string | Task.priority | 边框粗细或标签 |
| 有向边 | array | Task.dependencies | A→B 表示 B 依赖 A |
| Tooltip 摘要 | object | 聚合 | task_id、标题、状态、认领者 |

### States

| State | Display | Trigger |
|-------|---------|---------|
| Loading | 加载动画 | 进入页面 |
| Populated | DAG 图形渲染 | 数据加载完成 |
| Empty | 空状态提示 | 该 Feature 无任务 |
| No Dependencies | 节点无连线 | 所有任务独立无依赖 |

---

## UI Function 2: 看板依赖视图 + 拖拽

### Description

看板页新增的依赖视图模式，从上到下按依赖关系和优先级排列任务，支持拖拽改变依赖关系。

### User Interaction Flow

1. 用户在看板页点击视图切换 → 从状态视图切换到依赖视图
2. 浏览垂直排列的任务列表 → 理解任务执行顺序
3. 拖拽 Task A 到 Task B 上方 → B 依赖 A
4. 拖拽 Task A 到 Task B 下方 → A 依赖 B
5. 系统实时检测循环依赖 → 阻止无效操作
6. 保存成功 → 视图自动刷新

### Data Requirements

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| task_id | string | Task.taskId | 任务编号 |
| 标题 | string | Task.title | |
| 优先级 | string | Task.priority | P0(红)/P1(橙)/P2(蓝) |
| 状态 | string | Task.status | badge 颜色 |
| 认领者 | string | Task.claimedBy | |
| 依赖关系 | array | Task.dependencies | 控制排列顺序 |

### States

| State | Display | Trigger |
|-------|---------|---------|
| Loading | 骨架屏 | 切换到依赖视图 |
| Populated | 垂直任务列表 | 数据加载完成 |
| Dragging | 插入位置蓝色虚线指示 | 拖拽任务卡片 |
| Saving | 操作确认动画 | 保存依赖变更 |
| Cycle Error | Toast 提示循环依赖 | 拖拽导致循环 |

### Validation Rules

- 拖拽放置时实时检测循环依赖，≤ 100ms
- 不允许自依赖（拖拽到自身位置忽略操作）
- 依赖变更保存后同步更新涉及任务的 dependencies 字段

---

## UI Function 3: 全局 Agent 活动面板

### Description

展示所有 Agent 的活跃状态和工作负载，支持按 Agent ID 搜索。

### User Interaction Flow

1. 用户点击导航栏「Agent」入口 → 加载 Agent 列表
2. 浏览 Agent 列表 → 了解各 Agent 活跃状态、工作负载
3. 点击 Agent ID → 跳转 Agent 执行历史轨迹页
4. 使用搜索框 → 按 Agent ID 模糊过滤

### Data Requirements

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| Agent ID | string | Record.agentId 聚合 | 可点击链接 |
| 活跃任务数 | number | 聚合计算 | in_progress 状态计数 |
| 已完成任务数 | number | 聚合计算 | completed 状态计数 |
| 最近活动时间 | datetime | Record.createdAt 聚合 | 最后活动时间 |
| 当前认领任务 | string | Task 聚合 | task_id + 标题 |

### States

| State | Display | Trigger |
|-------|---------|---------|
| Loading | 骨架屏 | 进入页面 |
| Populated | Agent 列表 | 数据加载完成 |
| Empty | 空状态提示 | 无任何 Agent 活动 |
| No Active | 全部 Agent 活跃数为 0 | 无 Agent 正在执行 |

### Validation Rules

- 搜索框输入防抖 300ms
- 列表默认按最近活动时间倒序

---

## UI Function 4: Agent 执行历史轨迹

### Description

展示单个 Agent 的所有执行记录，按时间倒序排列。

### User Interaction Flow

1. 从 Agent 面板点击 Agent ID → 加载执行历史
2. 浏览执行记录列表 → 了解该 Agent 的完整工作轨迹
3. 点击任务 task_id → 跳转对应任务详情页
4. 使用翻页 → 加载更多历史记录
5. 使用时间范围选择器 → 筛选指定时间段的执行记录
6. 使用操作类型下拉 → 过滤 claim / record / status 类型

### Data Requirements

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| 时间戳 | datetime | Record.createdAt | |
| 任务 task_id | string | Record.taskId | 可点击链接 |
| 任务标题 | string | Task.title | |
| 操作类型 | string | Record 操作推断 | claim / record / status |
| 摘要 | string | Record.summary | |

### States

| State | Display | Trigger |
|-------|---------|---------|
| Loading | 骨架屏 | 进入页面 |
| Populated | 执行记录列表 | 数据加载完成 |
| Empty | "该 Agent 暂无执行记录" | Agent 无活动历史 |

### Validation Rules

- 时间范围选择器开始日期不得晚于结束日期
- 操作类型下拉默认为"全部类型"（不过滤）

---

## UI Function 5: 任务级 Agent 信息增强

### Description

增强 agent-task-center 任务详情页的认领者信息展示，新增可点击链接和统计数据。

### User Interaction Flow

1. 用户打开任务详情页 → 基本信息区域显示增强后的认领者信息
2. 查看 Agent ID（可点击）、认领时间、Agent 总完成数
3. 点击 Agent ID 链接 → 跳转 Agent 活动面板

### Data Requirements

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| 认领者 Agent ID | string | Task.claimedBy | 可点击链接 |
| 认领时间 | datetime | Record.createdAt | 该 Agent claim 操作的时间 |
| Agent 总完成数 | number | 聚合计算 | 该 Agent 已完成的任务总数 |

### States

| State | Display | Trigger |
|-------|---------|---------|
| Claimed | Agent ID 链接 + 认领时间 + 总完成数 | 任务已被认领 |
| Unclaimed | "暂未认领" | 任务无人认领 |
