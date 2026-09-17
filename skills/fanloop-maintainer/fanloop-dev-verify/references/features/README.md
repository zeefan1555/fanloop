# Fanloop Feature Map

Fanloop 用户可见行为的验证索引。契约真值仍是叶子 `--help`、五份生产 Workflow YAML 与真实可观察结果；
本地图只把这些行为压缩成 Agent 可搜索、可驱动的共享记忆。

## Baseline

- 使用当前候选的隔离安装、独立 `FANLOOP_DATA_HOME` 和全新 Requirement Root。
- 每次 Drive 前运行 Doctor，写操作先 dry-run，再比较 durable files。
- 默认清除 Botmux，不使用用户身份或真实远端；不可达路径记录具体前置条件。
- 证据保存在 session 外，Cleanup 后仍须存在。

## Features

- [Installation and health](installation-health.md)
- [Workflow selection](workflow-selection.md)
- [Cross-release continuation](cross-release-continuation.md)
- [Requirement lifecycle](requirement-lifecycle.md)
- [Progress and result](progress-and-result.md)
- [Routing and recovery](routing-and-recovery.md)
- [Output and state](output-and-state.md)
- [Trace](trace.md)
- [Card and panorama](card-and-panorama.md)
- [Live Skill configuration](live-skill-config.md)
- [Workflow products](workflow-products.md)
- [Multi-surface journeys](multi-surface-journeys.md)

## Proof and skip rules

证明必须包含入口、原始命令结果、持久化或外部副作用回读，以及候选身份。路径不可达时记录授权、身份、
网络、OS 或外部状态前置条件；mock 只可位于生产边界，不得跳过待验证的 CLI、Runtime 或持久化路径。
