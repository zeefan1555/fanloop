---
status: accepted
date: 2026-09-22
amends: ADR-0114
---

# 把 Agent 入口指到同一条验证路径

ADR-0114 留下一条验证路径。本决策把这条路径写进 Agent 已经会打开的文件。

## 摩擦

只读 `AGENTS.md` 时，验证门禁只要求运行 `fanloop verify doctor` 和相关 Feature 场景。Skill、Feature Map 和四套生产 Workflow 的位置都不在这份文件里。

`docs/ownership.md` 已经列出四套目录。Quality 一行当时写的是 E2E gates。本地测试入口是 `./tests/run-unit`。完整 Route Matrix 在 CI 的 `requirement-e2e`。`cmd/verify.go` 和 `internal/verify` 实现公开 CLI。搜索 verify 的 Agent 会先落到这两个包。

改之前，`README.md` 的使用一节另有一份 doctor、smoke、snapshot、cleanup 命令清单。按那份清单执行会跳过 Feature Map 和 `verify.evidence.outcome`。`docs/technical-design.md` 与 `CONTEXT.md` 当时只写了 Verification Skill 和 Feature Map 这两个名称。

## 入口

安装、`fanloop verify doctor`、Feature Map 和 `verify.evidence.outcome` 从 `.cursor/skills/verify-fanloop/SKILL.md` 开始。`features/` 仍是指向 `workflows/fanloop-maintainer/skills/fanloop-dev-verify/references/features/` 的符号链接。

四套生产 Workflow 仍是 `docs/ownership.md` 里的 `technical-solution-design`、`fanloop-maintainer`、`material-flashcards`、`pstack-delivery`。`AGENTS.md` 指向该文件和 `.cursor/skills/verify-fanloop/SKILL.md`。`README.md` 删掉第二份命令清单。Quality 边界改为 `./tests/run-unit` 与 CI `requirement-e2e`。

`cmd/verify.go` 与 `internal/verify` 留在原包。`docs/ownership.md` 写明它们实现 CLI。Agent 入口是 `.cursor/skills/verify-fanloop/SKILL.md`。

## 未动

Go 包、Thrift、五份生产 YAML 和移植的 pstack `SKILL.md` 正文保持原样。`workflows/pstack-delivery/skills/create-verification-skill/references/feature-map-example/` 仍在。该 Skill 正文链接这份示例地图。删除该目录会打断正文里的链接。本次不改移植正文，所以示例地图保留。Fanloop 的 Feature Map 仍是 `.cursor/skills/verify-fanloop/features/`。
