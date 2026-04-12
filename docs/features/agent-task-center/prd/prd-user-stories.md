---
feature: "Agent Task Center"
---

# User Stories: Agent Task Center

## Story 1: 项目全景监控

**As a** 开发者
**I want to** 在一个页面查看所有项目的任务完成概览（项目名称、Feature 数、完成率、更新时间）
**So that** 不用在多个本地目录间切换即可了解全局进度

**Acceptance Criteria:**
- Given Server 中有 3 个项目的数据
- When 开发者打开项目列表页
- Then 页面展示所有 3 个项目的名称、Feature 数、任务总数、完成率、最近更新时间
- And 列表默认按最近更新时间倒序排列
- And 支持按项目名称模糊搜索

---

## Story 2: 任务看板筛选

**As a** 开发者
**I want to** 在 Feature 级别的看板页面按优先级、标签、状态筛选任务
**So that** 快速定位当前最需要关注的任务

**Acceptance Criteria:**
- Given 一个 Feature 下有 pending、in_progress、completed、blocked 四种状态的任务
- When 开发者打开该 Feature 的任务看板
- Then 任务按状态分为四列展示（pending / in_progress / completed / blocked）
- And 每张卡片显示 task_id、标题、优先级 badge、认领者、标签
- When 开发者选择筛选条件（如 P0 优先级）
- Then 看板仅显示 P0 任务，筛选参数序列化到 URL
- And 复制 URL 在新标签页打开后保持相同的筛选状态

---

## Story 3: 执行历史审计

**As a** 开发者
**I want to** 查看任务的执行历史（时间戳、agent_id、修改文件、测试结果、关键决策、验收标准）
**So that** 审计 Agent 的工作并了解任务完成质量

**Acceptance Criteria:**
- Given 一个任务有 2 条执行记录
- When 开发者打开该任务详情页
- Then 页面显示任务基本信息（task_id、标题、状态、优先级、认领者、标签、依赖、描述）
- And 下方显示执行记录时间线，按时间倒序排列
- And 每条记录显示时间戳、agent_id、summary
- When 开发者展开某条记录
- Then 显示完整详情：filesCreated、filesModified、keyDecisions、testsPassed、testsFailed、coverage、acceptanceCriteria

---

## Story 4: CLI 数据推送

**As a** 开发者
**I want to** 通过 task-cli 将本地任务数据推送到 Server
**So that** Web UI 自动更新展示最新状态

**Acceptance Criteria:**
- Given 本地 docs/ 目录包含 proposal.md、manifest.md 和 index.json
- When 开发者执行 task push 命令
- Then Server 接收并解析所有文件
- And 自动创建/更新 Project、Proposal、Feature、Task 实体
- And 已存在的 Task 按 task_id 进行 Upsert（更新内容，不丢失执行记录）
- And CLI 输出推送结果摘要

---

## Story 5: Agent 认领任务

**As an** AI Agent
**I want to** 通过 task-cli 认领下一个可用任务
**So that** 无缝接入现有工作流开始执行

**Acceptance Criteria:**
- Given 一个 Feature 下有 3 个 pending 任务，优先级分别为 P0、P1、P2，其中 P0 任务依赖另一个已完成的任务
- When Agent 执行 task claim（携带 agent_id）
- Then Server 返回 P0 任务（优先级最高且依赖已满足）
- And 该任务状态变为 in_progress
- And 该任务的 claimed_by 字段记录 agent_id
- When 另一个 Agent 同时尝试 claim 同一任务
- Then 乐观锁阻止冲突，第二个 Agent 获得 P1 任务

---

## Story 6: Agent 提交执行记录

**As an** AI Agent
**I want to** 完成任务后提交执行记录（summary、files、tests、decisions、acceptance criteria）
**So that** 任务中心反映我的工作进度

**Acceptance Criteria:**
- Given Agent 已认领一个任务（状态为 in_progress）
- When Agent 执行 task record 并提交 record.json（包含 summary、filesCreated、filesModified、keyDecisions、testsPassed、testsFailed、coverage、acceptanceCriteria）
- Then Server 存储该执行记录，关联到对应任务
- And 任务状态更新为 completed
- When 开发者在 Web UI 查看该任务详情
- Then 能看到新提交的执行记录出现在时间线顶部
