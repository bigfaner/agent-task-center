---
feature: "agent-task-center"
created: 2026-04-13
prd: prd/prd-ui-functions.md
status: Draft
---

# UI Design: Agent Task Center

> Design layer: defines HOW the UI looks and behaves. See `prd/prd-ui-functions.md` for WHAT it must do.

## Design Principles

- **只读优先**：Web UI 是观察工具，主操作路径是浏览，不是编辑
- **信息密度适中**：看板页信息密集，详情页信息完整，列表页信息精简
- **状态可感知**：每个页面的加载/空/错误状态都有明确视觉反馈
- **URL 即状态**：筛选条件序列化到 URL，支持分享和刷新恢复

## Tech Stack

- React + Vite + TypeScript
- shadcn/ui + Tailwind CSS
- @tanstack/react-query（数据请求）
- react-router-dom v6（路由）
- react-markdown + remark-gfm（Markdown 渲染）

---

## Page 1: 项目列表页 `/`

### Layout

```
┌─────────────────────────────────────────────────────────┐
│  [Logo] Agent Task Center          [Upload 上传]         │
├─────────────────────────────────────────────────────────┤
│  Projects                                                │
│  ┌─────────────────────────────────────────────────┐    │
│  │ 🔍 搜索项目...                                   │    │
│  └─────────────────────────────────────────────────┘    │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │ 项目名称        Features  任务数  完成率   更新时间 │   │
│  ├──────────────────────────────────────────────────┤   │
│  │ [link] my-app       3      24    75.0%   2h ago  │   │
│  │ [link] backend      1       8    100%    1d ago  │   │
│  │ ...                                              │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### Component Hierarchy

```
ProjectListPage
├── AppHeader
│   └── UploadButton → opens UploadDialog
├── SearchInput (debounce 300ms)
└── ProjectTable
    ├── ProjectTableRow (×N)
    │   ├── ProjectNameLink → /projects/:id
    │   ├── FeatureCountBadge
    │   ├── TaskCountText
    │   ├── CompletionRateBar  (progress bar + percentage)
    │   └── RelativeTime
    └── TableSkeleton (loading state)
```

### States

| State | Component | Visual |
|-------|-----------|--------|
| Loading | TableSkeleton | 5行骨架行，灰色动画 |
| Populated | ProjectTable | 完整数据表格 |
| Empty | EmptyState | 插图 + "暂无项目" + "点击上传开始" 按钮 |
| Error | ErrorState | 红色提示 + "重试" 按钮 |
| Filtered Empty | EmptyState | "未找到匹配项目" |

### Interactions

| Trigger | Action |
|---------|--------|
| 输入搜索框 | 300ms 防抖后过滤列表（前端过滤） |
| 点击项目名称 | 跳转 `/projects/:id` |
| 点击「上传」 | 打开 UploadDialog |
| 清空搜索框 | 恢复完整列表 |

### Data Binding

| UI Element | API Field | Format |
|------------|-----------|--------|
| 项目名称 | `name` | 链接文字 |
| Features 数 | `featureCount` | 数字 |
| 任务数 | `taskTotal` | 数字 |
| 完成率 | `completionRate` | `75.0%` + 进度条 |
| 更新时间 | `updatedAt` | 相对时间（`2h ago`） |

---

## Page 2: 项目详情页 `/projects/:id`

### Layout

```
┌─────────────────────────────────────────────────────────┐
│  ← 返回   my-app                      [Upload 上传]      │
├─────────────────────────────────────────────────────────┤
│  [Features] [Proposals]                                  │
├─────────────────────────────────────────────────────────┤
│  Features Tab:                                           │
│  ┌──────────────────────────────────────────────────┐   │
│  │ Feature 名称    状态 badge   完成率      更新时间  │   │
│  ├──────────────────────────────────────────────────┤   │
│  │ [link] auth     [in-progress]  60.0%   3h ago   │   │
│  │ [link] dashboard [done]       100%    1d ago    │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
│  Proposals Tab:                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │ 提案标题              关联 Features  创建时间     │   │
│  ├──────────────────────────────────────────────────┤   │
│  │ [link] Auth Proposal      2       2026-01-10    │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### Component Hierarchy

```
ProjectDetailPage
├── PageHeader
│   ├── BackLink → /
│   ├── ProjectTitle
│   └── UploadButton → opens UploadDialog
├── TabGroup
│   ├── Tab: Features (default active)
│   └── Tab: Proposals
├── FeatureTable (Features tab)
│   ├── FeatureTableRow (×N)
│   │   ├── FeatureNameLink → /features/:id/tasks
│   │   ├── StatusBadge
│   │   ├── CompletionRateBar
│   │   └── RelativeTime
│   └── TableSkeleton
└── ProposalTable (Proposals tab)
    ├── ProposalTableRow (×N)
    │   ├── ProposalTitleLink → /proposals/:id
    │   ├── FeatureCountText
    │   └── DateText
    └── TableSkeleton
```

### Status Badge Colors

| Status | Color | Label |
|--------|-------|-------|
| `prd` | blue | PRD |
| `design` | purple | Design |
| `tasks` | yellow | Tasks |
| `in-progress` | orange | In Progress |
| `done` | green | Done |

### States

| State | Scope | Visual |
|-------|-------|--------|
| Loading | 整页 | 骨架屏（Tab + 表格行） |
| Populated | 整页 | 完整 Tab + 表格 |
| Empty Tab | 当前 Tab | "此项目暂无 [Features/Proposals]" |

---

## Page 3: 任务看板页 `/features/:id/tasks`

### Layout

```
┌─────────────────────────────────────────────────────────┐
│  ← 返回   auth — Tasks                                   │
├─────────────────────────────────────────────────────────┤
│  Priority: [All▼]  Tags: [All▼]  Status: [All▼]  [清除] │
├──────────┬──────────┬──────────┬────────────────────────┤
│ Pending  │In Progress│Completed│ Blocked                │
│  (5)     │  (2)     │  (8)    │  (1)                   │
├──────────┼──────────┼──────────┼────────────────────────┤
│ ┌──────┐ │ ┌──────┐ │ ┌──────┐│ ┌──────┐              │
│ │1.1   │ │ │2.1   │ │ │1.2  ││ │3.1   │              │
│ │Setup │ │ │Auth  │ │ │Login││ │Deploy│              │
│ │[P0]  │ │ │[P1]  │ │ │[P1] ││ │[P2]  │              │
│ │core  │ │ │agent1│ │ │     ││ │      │              │
│ └──────┘ │ └──────┘ │ └──────┘│ └──────┘              │
│ ┌──────┐ │          │         │                        │
│ │1.3   │ │          │         │                        │
│ │...   │ │          │         │                        │
│ └──────┘ │          │         │                        │
└──────────┴──────────┴──────────┴────────────────────────┘
```

### Component Hierarchy

```
FeatureKanbanPage
├── PageHeader
│   ├── BackLink → /projects/:projectId
│   └── FeatureTitle
├── FilterBar
│   ├── PrioritySelect (multi, options: P0/P1/P2)
│   ├── TagSelect (multi, dynamic from task tags)
│   ├── StatusSelect (multi, options: pending/in_progress/completed/blocked)
│   └── ClearFiltersButton (visible when any filter active)
└── KanbanBoard
    ├── KanbanColumn: Pending
    ├── KanbanColumn: In Progress
    ├── KanbanColumn: Completed
    └── KanbanColumn: Blocked
        └── TaskCard (×N per column)
            ├── TaskIdText
            ├── TaskTitleText
            ├── PriorityBadge
            ├── TagList
            └── ClaimedByText (if claimed)
```

### Task Card Design

```
┌─────────────────────────────┐
│ 1.1  [P0 red badge]         │
│ Setup project scaffold      │
│ [core] [api]                │
│ agent-01                    │
└─────────────────────────────┘
```

### Priority Badge Colors

| Priority | Color |
|----------|-------|
| P0 | red (`bg-red-100 text-red-700`) |
| P1 | orange (`bg-orange-100 text-orange-700`) |
| P2 | blue (`bg-blue-100 text-blue-700`) |

### States

| State | Scope | Visual |
|-------|-------|--------|
| Loading | 整页 | 四列骨架卡片 |
| Populated | 整页 | 四列 Kanban + 筛选器 |
| Filtered | 卡片层 | 仅显示匹配卡片，列标题更新计数 |
| Empty Column | 单列 | 列内显示 "—" 占位 |

### URL State Sync

筛选条件序列化到 URL query string：
```
/features/42/tasks?priority=P0&tag=core&status=pending
```
- 页面加载时从 URL 读取并初始化筛选器
- 筛选变更时用 `replaceState` 更新 URL（不产生历史记录）
- 筛选参数作为 query string 传给 `GET /api/features/:id/tasks`

### Interactions

| Trigger | Action |
|---------|--------|
| 选择筛选器 | 更新 URL + 重新请求 API |
| 点击「清除」 | 清空所有筛选，更新 URL |
| 点击任务卡片 | 跳转 `/tasks/:id` |

---

## Page 4: 任务详情页 `/tasks/:id`

### Layout

```
┌─────────────────────────────────────────────────────────┐
│  ← 返回看板                                              │
├─────────────────────────────────────────────────────────┤
│  1.1 Setup project scaffold                              │
│  [in-progress badge]  [P1 badge]  agent-01              │
│  Tags: [core] [api]                                      │
│  Dependencies: [1.0 →]                                   │
├─────────────────────────────────────────────────────────┤
│  Description                                             │
│  ┌──────────────────────────────────────────────────┐   │
│  │ (Markdown rendered content)                      │   │
│  └──────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────┤
│  Execution Records (2)                                   │
│  ┌──────────────────────────────────────────────────┐   │
│  │ ● 2026-04-13 10:30  agent-01                     │   │
│  │   Implemented auth middleware                    │   │
│  │   [展开 ▼]                                       │   │
│  ├──────────────────────────────────────────────────┤   │
│  │ ● 2026-04-12 15:00  agent-01                     │   │
│  │   Initial scaffold                               │   │
│  │   [展开 ▼]                                       │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### Component Hierarchy

```
TaskDetailPage
├── PageHeader
│   └── BackLink → /features/:featureId/tasks
├── TaskHeader
│   ├── TaskIdAndTitle
│   ├── StatusBadge
│   ├── PriorityBadge
│   ├── ClaimedByText
│   ├── TagList
│   └── DependencyLinks (→ /tasks/:depId)
├── TaskDescription
│   └── MarkdownRenderer
└── ExecutionRecordTimeline
    ├── RecordCount
    └── RecordItem (×N, newest first)
        ├── TimelineConnector (vertical line)
        ├── RecordHeader
        │   ├── Timestamp
        │   ├── AgentIdText
        │   ├── SummaryText
        │   └── ExpandToggle
        └── RecordDetail (collapsed by default)
            ├── FilesCreatedList
            ├── FilesModifiedList
            ├── KeyDecisionsList
            ├── TestResultsRow (passed/failed/coverage)
            └── AcceptanceCriteriaList
                └── CriterionRow (✓/✗ + text)
```

### Record Detail (Expanded)

```
┌──────────────────────────────────────────────────────┐
│ ● 2026-04-13 10:30  agent-01                         │
│   Implemented auth middleware                        │
│   [收起 ▲]                                           │
│                                                      │
│   Files Created                                      │
│   • src/middleware/auth.go                           │
│                                                      │
│   Files Modified                                     │
│   • server/main.go                                   │
│                                                      │
│   Key Decisions                                      │
│   • 使用 JWT 而非 session cookie                     │
│                                                      │
│   Tests  ✓ 12  ✗ 0  Coverage 85.6%                  │
│                                                      │
│   Acceptance Criteria                                │
│   ✓ 未认证请求返回 401                               │
│   ✓ 有效 token 通过验证                              │
└──────────────────────────────────────────────────────┘
```

### States

| State | Visual |
|-------|--------|
| Loading | 骨架屏（header + description + timeline） |
| Populated | 完整详情 + 时间线 |
| No Records | 时间线区域显示 "暂无执行记录" |
| Record Collapsed | 仅显示时间戳 + agent + summary |
| Record Expanded | 显示所有字段 |

---

## Page 5: 文档查看页 `/proposals/:id`

### Layout

```
┌─────────────────────────────────────────────────────────┐
│  ← 返回                                                  │
├─────────────────────────────────────────────────────────┤
│  Auth System Proposal                                    │
├──────────────────────────────────┬──────────────────────┤
│  (Markdown rendered content)     │  目录 (TOC)           │
│                                  │  • Overview          │
│  ## Overview                     │  • Goals             │
│  ...                             │  • Design            │
│                                  │                      │
│  ## Goals                        │                      │
│  ...                             │                      │
│                                  │                      │
├──────────────────────────────────┴──────────────────────┤
│  Related                                                 │
│  Features: [auth →]  [dashboard →]                      │
└─────────────────────────────────────────────────────────┘
```

### Component Hierarchy

```
DocViewerPage
├── PageHeader
│   └── BackLink
├── DocTitle
├── DocLayout (two-column on desktop, single on mobile)
│   ├── MarkdownContent
│   │   └── MarkdownRenderer (react-markdown + remark-gfm)
│   └── TableOfContents (desktop sidebar, sticky)
│       └── TocItem (×N, from H2/H3 headings)
└── RelatedSection
    ├── RelatedFeatureLinks (→ /features/:id/tasks)
    └── RelatedTaskLinks (→ /tasks/:id)
```

### States

| State | Visual |
|-------|--------|
| Loading | 加载 spinner（居中） |
| Populated | 双栏布局（内容 + TOC） |
| Error | "文档不存在或加载失败" + 返回按钮 |

---

## Dialog: 文件上传

### Layout

```
┌─────────────────────────────────────────────────────────┐
│  上传文件                                          [×]   │
├─────────────────────────────────────────────────────────┤
│  Project: [my-app                              ▼]        │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │                                                  │   │
│  │         拖拽文件到此处，或 点击选择               │   │
│  │         支持 .json 和 .md，最大 5MB              │   │
│  │                                                  │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
│                              [取消]  [上传]              │
└─────────────────────────────────────────────────────────┘
```

### Component Hierarchy

```
UploadDialog
├── DialogHeader ("上传文件")
├── ProjectSelector (dropdown, required)
├── DropZone
│   ├── DragOverlay (active when dragging)
│   ├── FileInput (hidden, accept=".json,.md")
│   └── SelectedFilePreview (filename + size)
├── ValidationMessage (error text, if invalid)
├── UploadProgress (progress bar, during upload)
├── UploadResult (success summary, after upload)
│   └── UpsertSummaryText ("新增 5 个任务，更新 2 个任务")
└── DialogFooter
    ├── CancelButton
    └── UploadButton (disabled until file selected + project chosen)
```

### State Machine

```
Idle → (file selected) → Validating
Validating → (valid) → Ready
Validating → (invalid) → Invalid
Ready → (click Upload) → Uploading
Uploading → (success) → Success
Uploading → (error) → Error
Invalid → (re-select file) → Validating
Error → (click Retry) → Uploading
Success → (click Close) → [dialog closes, parent list refreshes]
```

### States

| State | Visual |
|-------|--------|
| Idle | 空 DropZone + 提示文字 |
| Validating | DropZone 显示文件名 + spinner |
| Invalid | 红色错误提示（格式/大小/内容） |
| Ready | 文件名 + 大小 + 绿色校验通过 |
| Uploading | 进度条（indeterminate） |
| Success | 绿色 ✓ + 解析摘要文字 |
| Error | 红色错误信息 + "重试" 按钮 |

### Validation Messages

| Rule | Message |
|------|---------|
| 非 .json/.md | "仅支持 .json 和 .md 文件" |
| 超过 5MB | "文件大小不能超过 5MB" |
| index.json 格式错误 | "index.json 格式无效，需包含 task_id 和 title 字段" |

---

## Routing

```
/                          → ProjectListPage
/projects/:id              → ProjectDetailPage
/features/:id/tasks        → FeatureKanbanPage
/tasks/:id                 → TaskDetailPage
/proposals/:id             → DocViewerPage
```

## Shared Components

| Component | Usage |
|-----------|-------|
| `AppHeader` | 顶部导航栏，含 Logo + Upload 按钮 |
| `StatusBadge` | Feature/Task 状态 badge，颜色映射 |
| `PriorityBadge` | P0/P1/P2 badge |
| `CompletionRateBar` | 进度条 + 百分比文字 |
| `RelativeTime` | 相对时间显示（`2h ago`） |
| `TableSkeleton` | 表格骨架屏 |
| `EmptyState` | 空状态插图 + 提示文字 |
| `ErrorState` | 错误提示 + 重试按钮 |
| `MarkdownRenderer` | react-markdown 封装，含 GFM 支持 |
| `UploadDialog` | 文件上传弹窗 |

## Responsive Behavior

| Breakpoint | Behavior |
|------------|----------|
| Desktop (≥1024px) | 文档查看页双栏（内容 + TOC）；看板四列并排 |
| Tablet (768-1023px) | 看板四列横向滚动；文档查看页单栏 |
| Mobile (<768px) | 看板列切换为垂直堆叠；表格列精简 |
