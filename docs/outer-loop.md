# 取得本地证据后再开外层循环

代码变更要开云端 Agent、并行 Agent，或请求合并之前，按本页取得本地 verify 证据。规则在 [ADR-0116](adr/0116-require-local-verify-before-outer-loop.md)。驱动步骤只使用 [`.cursor/skills/verify-fanloop/SKILL.md`](../.cursor/skills/verify-fanloop/SKILL.md)。命令和 Feature Map 都只在该 Skill 里。

纯 Workflow YAML、Skill 指令及参考材料变更按 `AGENTS.md` 的配置校验路径执行，不要求本页的构建、隔离 Release 或代码验证 manifest。

## 取得证据

按该 Skill 的 Launch、Doctor、Drive、Evidence 和 Cleanup 执行。冷证明驱动 `features/installation-health.md`。本次改动驱动对应的 Feature。固定旅程是该 Skill 里的 `fanloop verify smoke`。

证明文件是该 Skill 写明的 `manifest.json`。只从该文件读 `outcome`。`candidate.commit_sha` 必须等于 `git rev-parse HEAD`。

## 再开外层

`outcome` 为 `passed` 时，才可以打开 PR 或请求合并。在这之前不要为同一变更启动云端 Agent 或第二个 Agent。

`failed` 或 `blocked` 时停在本机。按该 Skill 的 Failure modes 重跑，或报告 blocked。不要手写 `manifest.json`。

Sentry 或 Slack 叫醒之后，仍然从本页的取得证据开始。

## 维护验证地图

定时维护使用已有 Skill 当前 Fornax maintainer 包内 `skills/fanloop-dev-maintain-verification/SKILL.md` 的 Scheduled maintenance。

`TestCursorVerifyEntryIsTheOnlyFeatureMap` 与 `TestAgentNavigationPointsAtThePavedVerifyPath` 已经挡住第二条 Cursor 入口和丢失的指针。
