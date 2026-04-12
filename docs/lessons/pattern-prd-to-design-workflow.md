# PRD 完成后的正确工作流顺序

## Problem

eval-prd 通过后，eval 报告和 skill 模板中推荐下一步执行 `/breakdown-tasks`，跳过了技术设计和 UI 设计阶段。这会导致任务拆解缺少架构指导，任务粒度和接口定义不够具体。

## Root Cause

eval-prd 模板中的 Recommendation 部分默认推荐 `/breakdown-tasks` 作为下一步，没有区分"PRD 是否需要设计阶段"。实际上 breakdown-tasks 依赖 design-doc 的输出来确定任务结构。

## Solution

正确的 feature 生命周期流转顺序：

```
write-prd → eval-prd → design-tech / ui-design → breakdown-tasks → execute
```

PRD 通过评估后，应先执行：
- `/design-tech` — 产出技术设计文档（架构、接口、数据模型）
- `/ui-design` — 产出 UI 设计规格（可选，视 prd-ui-functions.md 是否存在）

设计文档定稿后，再执行 `/breakdown-tasks` 将设计拆解为可执行任务。

## Key Takeaway

eval-prd 的下一步不是 breakdown-tasks，而是 design-tech 或 ui-design。设计文档是任务拆解的必要输入。
