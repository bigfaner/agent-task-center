# Proposal: Agent Task Center

> Status: draft
> Created: 2026-04-12

## Problem

zcode 的 task-cli 是纯本地文件系统工具，任务数据（index.json）存在于单个项目目录中：

1. **不可见** — 缺少任务依赖图和执行进度的可视化
2. **不集中** — 无法在一个地方查看多个项目的任务状态
3. **不共享** — 多 agent 各自操作本地文件，无法共享任务状态和执行历史
4. **不可追溯** — 缺少执行记录和 agent 活动追踪

## Solution

构建 Agent Task Center —— 一个集中化的任务可视化与协作服务：

- **Web UI 只读看板** — 开发者通过浏览器查看 Project、Proposal、Feature、Task 的状态和进度
- **CLI 数据推送** — 通过 task-cli 将本地 index.json 和文档上传到 Server
- **Agent 远程协作** — Agent 通过 task-cli 远程模式认领任务、更新状态、提交执行记录

### 核心理念

**Agent 优先，人类监督。**

- Agent 是执行引擎，通过 CLI 与 Server 交互
- 人类通过 Web UI 观察全局进度、审查执行结果
- 数据流向：本地文档 → Server 存储 → Web UI 可视化

## Data Model

四层独立实体：

```
Project → Proposal → Feature → Task
```

| 实体 | 说明 | 来源 |
|------|------|------|
| Project | 项目（如 agent-task-center） | 手动创建或首次上传时自动创建 |
| Proposal | 提案文档 | 上传 proposal.md |
| Feature | 功能模块 | 上传 manifest.md 或通过 index.json 关联 |
| Task | 具体任务 | 上传 index.json 批量导入 |

### Task 状态流转

```
pending → in_progress → completed
   ↓          ↓
blocked ←─────┘
```

| 状态 | 触发方 | 说明 |
|------|--------|------|
| pending | 上传 index.json | 任务等待认领 |
| in_progress | Agent `task claim` | Agent 认领执行 |
| completed | Agent `task record` | Agent 提交执行记录 |
| blocked | Agent `task status` | 任务被阻塞 |

## Architecture Overview

```
┌─────────────────────────────────┐
│           Web UI (只读)          │
│  项目列表 | 提案 | 功能 | 任务看板  │
│  执行记录 | 依赖图 | Agent 活动    │
└──────────────┬──────────────────┘
               │ REST API (/api/)
┌──────────────┴──────────────────┐
│           Server                │
│  数据存储 | 文件解析 | Agent API   │
└──────┬───────────────┬──────────┘
       │               │
┌──────┴──────┐  ┌─────┴──────────┐
│  task-cli   │  │  task-cli      │
│  (推送数据)  │  │  (Agent 远程)   │
│  push/上传   │  │  claim/record  │
└─────────────┘  └────────────────┘
```

### 双路由设计

| 路由前缀 | 面向 | 说明 |
|----------|------|------|
| `/api/` | Web UI | 按 ID 寻址，返回聚合数据 |
| `/api/agent/` | task-cli (Agent) | 按 Key 寻址，语义化操作 |

## Key Features

### 1. 项目全景

- 项目列表：名称、Feature 数、任务完成率、更新时间
- 进入项目后查看该项目的 Proposal、Feature、Task

### 2. 任务看板

- Kanban 视图：按状态分列（pending / in_progress / completed / blocked）
- 任务卡片：task_id、标题、优先级 badge、认领的 agent_id
- 筛选条件：优先级、标签、状态（序列化到 URL，支持分享）
- 拖拽排序（仅视觉，不修改状态 — 状态变更通过 Agent API）

### 3. 任务依赖图

- Feature 内的任务依赖关系 DAG 可视化
- 节点颜色反映状态
- 点击节点跳转任务详情

### 4. 执行记录时间线

- 任务详情页内按时间倒序展示执行记录
- 每条记录：时间戳、agent_id、summary
- 展开后：修改文件列表、关键决策、测试通过/失败数、覆盖率、验收标准

### 5. Proposal / Feature 文档查看

- Markdown 渲染展示 proposal.md、manifest.md 等文档内容
- 关联的 Feature 和 Task 列表

### 6. Agent 活动面板

- Agent 列表：活跃任务数、已完成任务数、最近活动时间
- Agent 执行历史轨迹

### 7. 数据上传

**CLI 推送：**
- `task push` — 将本地 docs/ 目录推送到 Server
- 解析 index.json 批量创建 Task
- 解析 proposal.md / manifest.md 创建 Proposal 和 Feature

**Web UI 上传：**
- 上传 index.json 文件批量导入 Task
- 上传 proposal.md 等文档

### 8. Agent 远程操作

Agent 通过 task-cli 远程模式与 Server 交互：

- `task claim` — 认领下一个可执行任务（乐观锁防并发冲突）
- `task status <id>` — 更新任务状态
- `task record` — 提交执行记录（summary、files、tests、decisions）
- `task get-content <key>` — 获取任务详细内容

## Data Flow

```
[本地 zcode 工作流]                 [Server]              [Web UI]

开发者 + Agent 本地协作          数据存储 + 解析         只读可视化
        │                            │                      │
  生成 proposal.md ──push──→ 存储为 Proposal ──API──→ 提案列表页
  拆分 feature/tasks ──push──→ 创建 Feature + Task ──API──→ 看板视图
        │                            │                      │
  Agent 执行 ──claim──→ 任务状态: in_progress ──API──→ 看板实时更新
  Agent 完成 ──record──→ 执行记录 + completed ──API──→ 时间线展示
```

## Scope

### In Scope

- 四层数据模型（Project → Proposal → Feature → Task）
- Web UI 只读看板（项目/提案/功能/任务四层导航）
- 任务看板（Kanban 视图、筛选、依赖图）
- 执行记录时间线
- Agent 活动面板
- Proposal / Feature 文档查看
- CLI 推送（push index.json 和文档）
- Web UI 上传 index.json
- Agent 远程操作（claim / status / record / get-content）
- 并发认领安全（乐观锁）

### Out of Scope

- 认证 / 授权（V1 不做）
- Web UI 创建 / 编辑实体
- AI 自动规划 / 任务拆解
- Feature 文件同步（push/pull 设计文档附件）
- 实时通知（WebSocket）
- CI/CD 集成
- MCP Server 模式
- 任务自动调度算法

## User Stories

### US1: 项目全景监控

**As a** 开发者
**I want to** 在一个页面查看所有项目的任务完成概览
**So that** 不用在多个本地目录间切换即可了解全局进度

### US2: 任务看板筛选

**As a** 开发者
**I want to** 按优先级、标签、状态筛选任务看板
**So that** 快速定位当前最需要关注的任务

### US3: 执行历史审计

**As a** 开发者
**I want to** 查看任务的执行历史（修改文件、测试结果、关键决策）
**So that** 审计 Agent 的工作并了解任务完成质量

### US4: CLI 数据推送

**As a** 开发者
**I want to** 通过 task-cli 将本地任务数据推送到 Server
**So that** Web UI 自动更新展示最新状态

### US5: Agent 认领任务

**As an** AI Agent
**I want to** 通过 task-cli 认领下一个可用任务
**So that** 无缝接入现有工作流开始执行

### US6: Agent 提交执行记录

**As an** AI Agent
**I want to** 完成任务后提交执行记录
**So that** 任务中心反映我的工作进度

## Success Metrics

| 指标 | 目标 | 说明 |
|------|------|------|
| 多项目可视 | ≥3 个项目同时展示 | 消除项目间切换 |
| Agent 零学习成本 | 现有 task-cli 命令远程模式正常工作 | 通过环境变量切换 |
| 看板实时性 | 上传/状态变更后刷新即见 | 无需手动刷新 |
| 并发安全 | ≥3 个 Agent 同时领取不同任务不冲突 | 乐观锁保证 |
