# User Stories: Agent Task Center

## Story 1: 跨项目任务监控

**As a** 开发者/管理者
**I want to** 在一个看板中查看所有项目的任务状态
**So that** 无需在多个本地目录间切换即可监控整体进度

**Acceptance Criteria:**
- Given 系统中有 ≥2 个项目，每个项目下有若干 Feature 和 Task
- When 用户打开 Web UI 首页
- Then 页面显示所有项目的列表，包含 Feature 数和任务完成率

---

## Story 2: 任务筛选与分享

**As a** 开发者/管理者
**I want to** 按项目、Feature、优先级、标签、状态筛选任务
**So that** 快速定位当前最需要关注的任务

**Acceptance Criteria:**
- Given Feature 下有多个不同优先级和状态的任务
- When 用户在 FilterBar 中选择筛选条件（如 priority=P0, status=in_progress）
- Then 看板仅显示符合条件的任务，筛选条件同步到 URL query params
- When 用户复制当前 URL 分享给他人
- Then 他人打开链接后看到相同的筛选结果

---

## Story 3: 执行历史审计

**As a** 开发者/管理者
**I want to** 查看任务的执行历史（summary、修改文件、测试结果、关键决策）
**So that** 审计 Agent 的工作并了解任务完成情况

**Acceptance Criteria:**
- Given 任务已由 Agent 完成并提交了执行记录
- When 用户在任务详情页查看 RecordTimeline
- Then 时间线按时间倒序显示所有执行记录
- When 用户展开某条记录
- Then 显示 summary、files_created、files_modified、key_decisions、tests_passed、tests_failed、coverage 和 acceptance_criteria 字段

---

## Story 4: 创建项目与 Feature

**As a** 开发者/管理者
**I want to** 通过 Web UI 创建项目和组织 Feature
**So that** 在将任务交给 Agent 之前先搭建好项目结构

**Acceptance Criteria:**
- Given 用户在项目列表页
- When 用户点击"新建项目"并填写 Key、名称、描述
- Then 项目创建成功，返回项目列表并显示新项目
- Given 用户进入某项目详情页
- When 用户点击"新建 Feature"并填写 Key、名称、描述
- Then Feature 创建成功，显示在 Feature 列表中

---

## Story 5: Feature 附件管理

**As a** 开发者/管理者
**I want to** 上传设计文档和附件到 Feature
**So that** Agent 可以获取最新的技术方案而无需手动共享文件

**Acceptance Criteria:**
- Given 用户在 Feature 详情页
- When 用户上传一个 ≤50MB 的文件
- Then 附件保存成功，出现在附件列表中
- When Agent 执行 `task feature pull`
- Then 附件内容下载到本地对应目录

---

## Story 6: Agent 一键认领任务

**As a** AI Agent
**I want to** 通过单个 CLI 命令认领下一个可用任务
**So that** 无需学习新工具即可开始工作

**Acceptance Criteria:**
- Given Feature 下有 pending 任务且其所有依赖任务已完成
- When Agent 执行 `task claim`（已设置 `TASK_REMOTE_URL`）
- Then 返回下一个可认领任务的详情，任务状态变为 `in_progress`，`agent_id` 被设置
- Given 两个 Agent 同时认领同一任务
- Then 只有一个成功（200），另一个收到 409 错误（`task.already_claimed`）

---

## Story 7: Agent 提交执行记录

**As a** AI Agent
**I want to** 完成任务后提交执行记录
**So that** 任务中心反映我的工作进度，其他 Agent 知道该任务已完成

**Acceptance Criteria:**
- Given Agent 已认领任务（状态为 `in_progress`）
- When Agent 执行 `task record` 并提供 record.json
- Then 执行记录保存到服务端，任务状态更新为 `completed`，记录 `completed_at`
- Given 其他 Agent 查询该 Feature 的可认领任务
- Then 该任务不再出现在可认领列表中

---

## Story 8: Agent 获取任务详情

**As a** AI Agent
**I want to** 通过 CLI 获取任务的详细内容（Markdown）
**So that** 在开始工作前了解完整的任务规格

**Acceptance Criteria:**
- Given 任务存在且有关联的 TaskContent
- When Agent 执行 `task get-content <task-key>`
- Then 返回 Markdown 格式的任务详细内容
- Given 任务不存在
- Then 返回 404 错误（`task.not_found`），message 中包含可用的查询建议
