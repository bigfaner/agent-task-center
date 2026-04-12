---
feature: "agent-task-center-v2"
---

# User Stories: Agent Task Center V2

## Story 1: 任务依赖可视化

**As a** 开发者
**I want to** 在 DAG 图形中直观查看 Feature 内所有任务的依赖关系拓扑
**So that** 不用翻阅 JSON 文件就能理解任务执行顺序和阻塞关系

**Acceptance Criteria:**
- Given 一个 Feature 下有 8 个任务，其中 3 个任务有依赖关系
- When 开发者打开该 Feature 的依赖图页面
- Then 页面以 DAG 图形展示所有 8 个任务节点
- And 有依赖关系的任务之间显示有向边（A → B 表示 B 依赖 A）
- And 节点颜色反映任务状态（completed=绿、in_progress=蓝、pending=灰、blocked=红）
- When 开发者点击某个任务节点
- Then 跳转到该任务的详情页

---

## Story 2: 拖拽调整依赖关系

**As a** 开发者
**I want to** 在依赖视图中通过拖拽直观地调整任务间的依赖关系
**So that** 不用修改 JSON 文件重新推送就能快速调整执行计划

**Acceptance Criteria:**
- Given 一个 Feature 的依赖视图中有 5 个任务
- When 开发者将 Task A 拖拽到 Task B 的上方
- Then B 依赖 A（A 成为 B 的前置任务），视图自动刷新排列
- When 开发者将 Task A 拖拽到 Task B 的下方
- Then A 依赖 B（B 成为 A 的前置任务），视图自动刷新排列
- When 拖拽操作会导致循环依赖
- Then 操作被拒绝，显示"无法建立依赖：会形成循环依赖"提示，任务恢复原位
- And 依赖变更通过 API 保存到 Server，后续 push 的 index.json 不覆盖手动调整的依赖

---

## Story 3: Agent 活动全局监控

**As a** 开发者
**I want to** 在一个面板查看所有 Agent 的活跃状态、工作负载和执行历史
**So that** 了解团队 Agent 的整体工作情况，及时发现异常

**Acceptance Criteria:**
- Given 有 3 个 Agent 曾在系统中有过活动
- When 开发者打开 Agent 活动面板
- Then 页面展示所有 3 个 Agent 的 ID、活跃任务数、已完成任务数、最近活动时间、当前认领任务
- And 列表默认按最近活动时间倒序排列
- When 开发者点击某个 Agent ID
- Then 跳转到该 Agent 的执行历史轨迹页面
- And 历史轨迹按时间倒序展示该 Agent 的所有执行记录

---

## Story 4: 任务级 Agent 信息查看

**As a** 开发者
**I want to** 在任务详情页快速了解执行该任务的 Agent 的概况
**So that** 评估该任务执行者的经验和可靠性

**Acceptance Criteria:**
- Given 一个任务已被 Agent claude-opus-001 认领并完成
- When 开发者打开该任务详情页
- Then 基本信息区域显示：认领者 Agent ID（可点击链接）、认领时间、该 Agent 总完成数
- When 开发者点击认领者 Agent ID 链接
- Then 跳转到该 Agent 的活动面板页面
