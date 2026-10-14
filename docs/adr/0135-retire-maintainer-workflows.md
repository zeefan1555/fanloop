---
status: accepted
date: 2026-10-14
---

# 退役 fanloop-maintainer 与 remnote-direct-maintainer

## 背景

second-brain-skills 的 playbook-flow 新增 `code-change` playbook（second-brain-skills #24），覆盖 fanloop、remnote-direct 等仓库的代码变更与 PR 交付。`fanloop-maintainer` 与 `remnote-direct-maintainer` 两套 Workflow 不再使用。

## 决定

- second-brain-skills 删除 `fanloop-workflow/fanloop-maintainer/` 与 `fanloop-workflow/remnote-direct-maintainer/` 两个包。
- 入口 `routes.yaml` 的 `fanloop-maintenance` 场景改指 `playbook-flow`。
- Trace Registry 去掉 `fanloop-maintainer` 的单独 profile；production 默认已是 `provider: none`，旧 ID 仍解析为 none。
- fanloop-remnote 插件删除 `remnote-direct-maintainer` 及 define/build/verify/deliver 四个子 Skill。
- 文档中的入口改指 `code-change` playbook；历史 ADR 保持原样。

## 兼容

Runtime 没有硬编码这两个 Workflow ID。状态解码与 Trace 解析测试仍用 `fanloop-maintainer` 作为夹具，保证旧 Requirement 状态可以读出。已存在的旧 Requirement 若需继续推进，可固定到退役前的配置快照（`FANLOOP_CONFIG_SOURCE` 指向旧提交）。
