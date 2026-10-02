---
name: fanloop-config-deliver
description: 交付已检查的本地 Workflow Skill 包并核对公开 CLI 读取结果；仅在 Maintainer Deliver 的配置 Route 使用。
---

# 交付本地配置

只交付 `configuration_check` 对应的精确文件摘要。复核目标仓库与 Define 的 Git 基线；有并发变更或候选变化时保持 blocked 或用 configuration_needs_fix 回 Build，重新检查。

1. 核对完整包位于目标仓库 `skills/<workflow-id>/`，根 SKILL.md、五份 YAML、子 Skills 和资源齐全。提交已检查的精确变更并记录 commit SHA；不修改无关文件，不覆盖用户修改。
2. 使用目标绑定仓库或显式绝对 `FANLOOP_CONFIG_SOURCE` 执行公开 Fanloop CLI，运行 Doctor 并从 Status 回读准确的 Skill 路径。逐文件核对摘要；配置下一次调用即生效，无下载、复制、同步或重建 CLI。
3. 有共享 PR 授权时使用 gh 幂等创建或更新唯一 PR，回读同一 HEAD 的轻量 changes 结果及适用 required checks。纯配置允许引擎任务按规则 skipped，不允许 changes 失败、未知或候选漂移；混合变更回 Define，不能自行忽略 CI。按授权合并后核对 MERGED、merge tree 等于已检查 tree，再更新本 Requirement 源码并重做步骤 2 生效回读；本地提交不能冒充合并。不绕过保护、不重建或安装 CLI。
4. 向同一 delivery-record.md 追加基线/提交、完整包摘要、公开 CLI 命令结果及原始日志引用、PR（适用时）和实际耗时；给用户此文件的绝对路径链接。`live_configuration_synced` 返回实际完成的本地源交付与回读事实，`delivery_record_written` 返回此记录路径；附加 configuration_change 与 panorama_presented，选择真实 terminal route。

纯配置交付不要求代码验证 manifest、Go 全量测试、构建或 CLI 安装，不提交 code_merged 等代码通道事实。普通修改不增加 Agent、阶段报告、重复检查或无关调研。平时只记录实际耗时，不运行配对基准；仅优化验收或排查延迟时对比相同检查和本地提交/生效回读终点，包含 Agent 阅读、编排、Panorama 展示及工具等待，外部共同等待单列。额外 30 秒为实测目标，未达标如实说明。
