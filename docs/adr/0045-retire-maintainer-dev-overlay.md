---
status: accepted
date: 2026-08-17
amends: ADR-0037
---

# 退役维护者 DEV Overlay

Fanloop 只保留正式发布使用的单一运行时、Workflow Bundle 和配套 Skills。
删除 `fanloop_dev` build tag、`fanloop-dev` Workflow、维护者 DEV Skills、
特殊 Doctor/Version 行为和 `scripts/dev.sh` 本地安装器；源码验证直接使用现有
`go build`、Go 测试和仓库 E2E 入口。

正式 `fanloop@9.0.0` 五文件 Bundle 与发布清单不变。旧
`fanloop-dev@3.0.0` Requirement 不迁移，当前源码不提供 alias、兼容层、
fallback 或替代 maintenance Workflow。仓库外既有 `~/.fanloop` Release、
Requirement 数据和 Skill 链接不由本次仓库变更删除。

本决策仅取代 ADR-0037 中继续携带维护者专用 `fanloop-dev` Bundle 的部分；
其余 Thrift-first、Workflow 绑定和直接切换边界不变。完整人工审核记录、
变更前后 YAML 边界与验收条件见
`docs/specs/retire-fanloop-dev.md`。
