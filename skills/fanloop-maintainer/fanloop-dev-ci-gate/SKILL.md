---
name: fanloop-dev-ci-gate
description: 校验 main Ruleset 与冻结 PR 的全部必需 CI checks，失败时归属最早责任层。
---

# CI 硬门禁

只读 GitHub 仓库、Ruleset、PR 和 checks。要求 main 使用 PR required、strict、squash、linear
history，禁止 force push、删除和 bypass，人工审批数为 0；记录 Ruleset ID。

必需 checks 必须在 candidate_head 上成功，并覆盖 Workflow 契约/Route Matrix、生成物与 IDL
新鲜度、./tests/run-unit、./tests/run-e2e、安装/Doctor、机器人身份隔离、本地材料禁入库和 Review
后候选只读。排队或平台异常保持 blocked；确定失败才分类为需求、方案、实现、验证技能或功能地图
之一，写入 acceptance-report.md。不得用旧 SHA、部分成功或人工口头结论代替。
