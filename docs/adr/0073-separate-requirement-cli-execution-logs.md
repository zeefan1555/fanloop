---
status: accepted
date: 2026-08-24
amends: ADR-0039, ADR-0051
---

# 单独保存 Requirement 范围的 CLI 执行日志

Fanloop 在四个业务事实与投影目录之外增加第五个职责目录 `.fanloop/log`，其中
`cli.jsonl` 以一行一个 `CLIExecutionLogEntry` 保存 CLI 进程级诊断事实。它不是 Flow
State、RegisteredOutput、Event、Trace 或 Card，不参与恢复、路由、渲染、远端投影和
Doctor 健康判断。由此扩展 ADR-0039 的目录边界，同时保留各原有目录职责。

所有具有有效绝对 Requirement `--root` 的真实叶子调用在结束后记录一次，包括成功、公开
失败、Cobra 参数失败、部分非零结果、只读和 `--dry-run`。Help 不记录；无 Root 的
`version`、npm launcher `update` 和隐藏 `__install` 不建立全局日志。统一 `cmd.Execute`
是唯一写入接缝，叶子实现不重复记录。

日志只保存 schema version、invocation ID、开始时间、耗时、command ID、CLI/release/commit
版本、dry-run、exit code 和可选稳定公开 error code。不得保存 raw argv、flag 值、输入输出、
Request/Response、URL、Evidence、环境变量、密钥或 token。`idl/storage.thrift` 新增
`CLI_EXECUTION_LOG_SCHEMA_VERSION = 1` 与 `CLIExecutionLogEntry`，按 ADR-0051 生成 Go
类型并使用自然 JSONL；不增加公开命令或 Service。

写入是独立的 best-effort 诊断副作用：失败不改变原命令 stdout、stderr、exit code，也不
回滚或污染 Flow/Output/Event/Trace/Card。目录和文件固定为 `0700/0600`，打开目录与文件时
拒绝符号链接，对数据文件持有排他锁后一次追加完整行。首版只使用单文件，不增加配置、日志
级别、全局日志、轮转、压缩、过期清理、迁移或历史回填。

本决策保留 ADR-0011 的公开结果信封、ADR-0015 的 durable-effect-only Event、ADR-0020 的
只读 Doctor 业务边界和 ADR-0065 的 launcher-owned update。只读与 dry-run 的业务语义不变，
Help 明确披露统一诊断追加；Workflow YAML、Step 拓扑和推进语义不变。
