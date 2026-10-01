---
name: fanloop-config-deliver
description: 交付已检查的本地 Workflow Skill 包并核对公开 CLI 读取结果；仅在 Maintainer Deliver 的配置 Route 使用。
---

# 交付本地配置

只交付 `configuration_check` 对应的精确文件摘要。复核目标仓库与 Define 的 Git 基线；有并发变更或候选变化时保持 blocked 或用 configuration_needs_fix 回 Build，重新检查。

1. 核对完整包位于目标仓库 `skills/<workflow-id>/`，根 SKILL.md、五份 YAML、子 Skills 和资源齐全。提交已检查的精确变更并记录 commit SHA；不修改无关文件，不覆盖用户修改。
2. 使用目标绑定仓库或显式绝对 `FANLOOP_CONFIG_SOURCE` 执行公开 Fanloop CLI，运行 Doctor 并从 Status 回读准确的 Skill 路径。逐文件核对摘要；配置下一次调用即生效，无下载、复制、同步或重建 CLI。
3. 把仓库绝对路径、基线/交付 commit、完整包摘要、公开 CLI argv/stdout/stderr/退出码写入 delivery-record.md。`live_configuration_synced` 返回本地源交付与回读事实对象，`delivery_record_written` 返回记录路径；附加 configuration_change 与 panorama_presented，选择当前真实 terminal route。

纯配置交付不要求代码验证 manifest、Go 全量测试、构建或 CLI 安装，不提交未发生的 code_merged、merge_tree_verified 等代码通道事实。需要共享仓库 PR 时按用户明确授权执行；本地提交不冒充远端合并。代码变更回 Define 重新分类。
