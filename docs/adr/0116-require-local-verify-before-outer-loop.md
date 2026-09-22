---
status: accepted
date: 2026-09-22
amends: ADR-0112
---

# 外层循环先要本地 verify 证据

本决策规定 ADR-0112 之后的外层循环。

## 前提

Cut 1 到 Cut 4 已经在 main 上。ADR-0112 是 verify manifest 门禁。ADR-0113 把同一 source 加到 `post_merge_smoke_passed`。ADR-0114 只留一条验证路径。ADR-0115 把 Agent 入口指到 `.cursor/skills/verify-fanloop/SKILL.md`。

## 规则

同一变更在本地隔离候选上留下 schema-1 `manifest.json`，且 `outcome` 为 `passed` 之前，不要为该变更启动云端 Agent，也不要并行启动第二个 Agent。

证据只按 `.cursor/skills/verify-fanloop/SKILL.md` 取得。命令是 `fanloop verify smoke` 或 `fanloop verify snapshot`。步骤在 `docs/outer-loop.md`。

CI 全绿仍然不是验收。

## Line cook 与 dishwasher

产品变更由一个执行者按已有 Skill 完成。这个执行者是 line cook。维护验证地图和删掉多余路径的人是 dishwasher。产品变更的执行者多于 dishwasher。dishwasher 的定时步骤已经写在 `workflows/fanloop-maintainer/skills/fanloop-dev-maintain-verification/SKILL.md` 的 Scheduled maintenance。一次维护运行，不按产品变更的数量扩成 Agent 农场。

两种工作都先要上面的本地 manifest。有了 `passed` 之后，一个变更仍然只有一个负责人。并行 live drive 只出现在该维护 Skill 已经写明的隔离条件里，每个环境有独立的 `FANLOOP_DATA_HOME`。

## 以后的叫醒

Sentry 错误和 Slack 消息以后可以叫醒一个 Agent。本决策不添加这两种连接器。叫醒之后仍然先跑 `docs/outer-loop.md`，不能直接并行。

## 收口

Cut 1 到 Cut 5 的厨房工作到此结束。之后的工作是这条路径上的产品行为，不是再加厨房流程。

## 未做

不实现 Slack 或 Sentry webhook。不部署并行 Agent 农场。不改移植的 pstack `SKILL.md` 正文。不改 Go、Thrift 和五份生产 YAML。不在 `.cursor/skills/` 下新增入口。
