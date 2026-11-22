---
status: accepted
date: 2026-11-23
supersedes_in_part: ADR-0110, ADR-0117, ADR-0118, ADR-0120, ADR-0124, ADR-0128, ADR-0131, ADR-0135
---

# 生产 Workflow 迁移为 Codex 插件

用户要求把日常 Fanloop Workflow 改成直接安装和执行的 Codex 插件。当前结果是：

- `portfolio-daily-brief` 与 `tradingview-backtest` 新建为 Codex 插件。
- `daily-journal` 与 `remnote` 已存在同产品插件，不再保留 Workflow。
- `ai-daily-brief`、`code-change`、`manage-fanloop-workflow`、`pstack-native`、`xmind-mindmap` 不再保留生产包。

## 决定

- `entrypoints/fanloop-workflow/routes.yaml` 保留 schema 2，但 `scenarios` 为空。新 `flow init` 没有生产场景。
- `entrypoints/fanloop-workflow/SKILL.md` 只指导已有 Requirement 使用本身 pinned controller；不得创建新 Root 或猜 Workflow。
- 删除 `.agents/skills/create-fanloop-workflow`、`.agents/skills/edit-fanloop-skills` 和 `plugins/fanloop-remnote`。
- 本地 Release 与 npm 分发包只保留 `fanloop-workflow` 兼容入口，不再包含 Workflow 作者入口。
- 删除 `second-brain-skills/fanloop-workflow/` 下全部生产包；产品 Skill 只在 `second-brain-agent/plugins/` 维护。

## Runtime 边界

通用 YAML loader、State/Event、pinned snapshot 与合成 Route Matrix 继续存在，用于已有 Requirement 和 Runtime 回归。它们不构成新产品入口。历史 ADR、旧 Requirement State 和已构建 Release 保持原样，不迁移、不重写。

## 验证

源码验证覆盖空场景 selector、Release 只含一个兼容 Skill、Runtime 单测和合成 fixture。Codex 插件在自己的父仓库测试包里验证；不要求连接真实账户、TradingView 或 Notion 才能完成离线打包检查。
