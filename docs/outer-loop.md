# 取得本地证据后再开外层循环

代码变更要开云端 Agent、并行 Agent，或请求合并之前，按本页取得本地 verify 证据。规则在 [ADR-0116](adr/0116-require-local-verify-before-outer-loop.md) 和 [ADR-0123](adr/0123-use-one-freeze-self-iteration.md)。驱动步骤只使用 [`.cursor/skills/verify-fanloop/SKILL.md`](../.cursor/skills/verify-fanloop/SKILL.md)。命令和 Feature Map 都只在该 Skill 里。

纯 Workflow YAML、Skill 指令及参考材料变更按 `AGENTS.md` 的配置校验路径执行，不要求本页的构建、隔离 Release 或代码验证 manifest。

## 风险与小批 Build

Define 先记录 `low|medium|high` 风险、Acceptance Set、Feature Impact Set、外部依赖和停止位置；不确定时使用
`high`。Build 按可独立验证的小批纵向切片推进，每批先跑最小聚焦检查。完整 `./tests/run-unit` 与公开 CLI
验证只对准备冻结的稳定候选执行，避免每个中间状态重复支付全量成本。

## 取得证据

按该 Skill 的 Launch、Doctor、Drive、Evidence 和 Cleanup 执行。冷证明驱动 `features/installation-health.md`。本次改动驱动对应的 Feature。固定旅程是该 Skill 里的 `fanloop verify smoke`。

证明文件是该 Skill 写明的 `manifest.json`。只从该文件读 `outcome`。`candidate.commit_sha` 必须等于 `git rev-parse HEAD`。

## Pre-Certify

本地证据通过后仍先完成一次 Pre-Certify：回读完整 diff、ADR 与契约，确认生成物、Feature Map、外部依赖、
工作树 clean，并固定同一 candidate identity。任一项变化都留在 Build 并重做本地证据，不能带着漂移候选进入认证。

## 再开外层

`outcome` 为 `passed` 且 Pre-Certify 完成时，才可以启动外层 Agent。独立 Code Review 与黑盒验证可并行认证，
前提是二者读取同一冻结 candidate identity、写状态完全隔离，并等待全部结果；不得因一个先失败就中断另一个，
否则下一轮仍缺少完整反馈。

两项认证通过后再进行一次性交付：只发布一个已验证的生产配置版本、维护一个 PR，并在同一 head 上等待
required checks。交付完成后在 `delivery-record.md` 记录 lead time、Build/Certify 次数与返工来源。

`failed` 或 `blocked` 时停在本机。按该 Skill 的 Failure modes 重跑，或报告 blocked。不要手写 `manifest.json`。

Sentry 或 Slack 叫醒之后，仍然从本页的取得证据开始。

## 维护验证地图

定时维护使用second-brain 配置仓库 `skills/fanloop-workflow/fanloop-maintainer/` 包内 `skills/fanloop-dev-maintain-verification/SKILL.md` 的 Scheduled maintenance。

`TestCursorVerifyEntryIsTheOnlyFeatureMap` 与 `TestAgentNavigationPointsAtThePavedVerifyPath` 已经挡住第二条 Cursor 入口和丢失的指针。
