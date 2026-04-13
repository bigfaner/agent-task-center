# Task Files Must Include YAML Frontmatter

## Problem

`/breakdown-tasks` 生成的任务文件（如 `1.1-init-monorepo.md`）缺少 YAML frontmatter，只有 Markdown 正文内容。

## Root Cause

执行 `/breakdown-tasks` 时未参考 `templates/task.md` 模板，直接按自由格式写任务文档，遗漏了 frontmatter 元数据块。

## Solution

每个任务文件必须以 YAML frontmatter 开头，严格按照模板格式：

```markdown
---
id: "1.1"
title: "Initialize Monorepo Structure"
priority: "P0"
estimated_time: "1h"
dependencies: ""
status: pending
---

# 1.1: Initialize Monorepo Structure

## Description
...
```

模板位置：`zcode/plugins/zcode/skills/breakdown-tasks/templates/task.md`

## Key Takeaway

执行任何 skill 前，先读取该 skill 目录下的 `templates/` 文件，确认输出格式要求。`task-cli` 等工具依赖 frontmatter 字段（`id`、`status`、`dependencies`）做机器解析，缺失会导致 `task validate` 失败或 `task claim` 无法正确识别任务。
