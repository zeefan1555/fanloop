---
status: accepted
date: 2026-08-24
amends: ADR-0073
---

# 完整记录 Requirement 范围的 CLI 双向 transcript

`.fanloop/log/cli.jsonl` 从低敏执行元数据改为完整、未脱敏且不截断的 Agent 与 CLI
双向 transcript。每条成功追加的记录保存传给 `cmd.Execute` 的有序 arguments、CLI 实际
消费的 stdin，以及 CLI 实际写出的 stdout 和 stderr；空流使用空字符串。`--input @file`
只把路径保留在 arguments，文件内容不作为 stdin 重复记录。

`idl/storage.thrift` 将 `CLI_EXECUTION_LOG_SCHEMA_VERSION` 直接提升到 2，并在
`CLIExecutionLogEntry` 保留 field ID 1-11 后新增 required field 12 `arguments`、13
`stdin`、14 `stdout` 和 15 `stderr`。历史 schema 1 行不迁移、不回填；当前产品没有日志
reader，因此不增加双读、兼容层或迁移工具。

统一 `cmd.Execute` 继续是唯一记录接缝。实现只使用标准库 tee/multi-writer，在调用方原有
stream 上旁路捕获数据，并继续复用 `internal/executionlog.Append` 的 `0700/0600` 权限、
符号链接与非普通文件拒绝、排他锁和单行完整追加。日志仍是 best-effort；写入失败不得改变
stdout、stderr、退出码或已提交业务事实。

记录范围保持 ADR-0073 不变：仅带有效绝对 `--root` 的真实公开叶子调用记录，包括只读、
失败和 `--dry-run`；Help、`version`、`update`、`__install`、未知命令和无有效 Root 的调用
不记录。Workflow YAML、公开 Request/Response、Event、Trace、Card 和 Doctor 语义不变。

完整 transcript 可能包含密码、token、URL、Evidence 或其他敏感值。CLI 不提供脱敏、截断、
配置开关、轮转、上传、清理或审计强保证；README、CONTEXT 和叶子 Help 必须明确披露风险，
调用方负责保护 Requirement 目录。
