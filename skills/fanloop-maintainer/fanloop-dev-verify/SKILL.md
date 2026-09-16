---
name: fanloop-dev-verify
description: 从当前 Fanloop 候选建立隔离环境，按 Feature Map 驱动真实公开 CLI，并保留可复核证据。
---

# Verify Fanloop

先读 [Feature Map](references/features/README.md)，只验证与当前变化相关的 Feature；广泛回归按索引顺序
执行并以 `multi-surface-journeys.md` 收尾。公开 `fanloop verify` 控制面落地后优先使用它；在此之前仍按
本 Skill 的相同步骤直接调用公开 Fanloop CLI，不以内部 Go 调用代替用户路径。

## Launch

记录候选 HEAD 与工作树。创建独立 session、evidence、`FANLOOP_DATA_HOME` 和四类 Skill Root，清除
`BOTMUX_CHAT_ID`、`BOTMUX_SESSION_ID`，从当前候选执行 `./scripts/install-local.sh`。只使用隔离
`current/bin/fanloop`；每个 Feature 使用全新 Requirement Root。

完成标准：隔离 current 的 `version.commit_sha` 等于候选 HEAD，全局 current 未改变。

## Doctor

每个新 session 首次 Drive 前运行 `fanloop version` 与 `fanloop doctor`。任何意外结果后重新 Doctor；
Doctor 不健康时先保存输出并停止驱动，不在错误实例上继续尝试。

完成标准：候选身份明确，Doctor healthy，Workflow 与 live Skill 配置来自本次隔离环境。

## Drive

先读目标叶子 `--help`，再按对应 Feature 页执行用户可见入口。每次 Workflow 动作前读取最新
`flow status`；写操作先执行 `--dry-run` 并观察其确实没有改变 durable files，再执行真实调用和只读回读。
使用稳定 command ID、Step ID、Condition ID、Output key、错误码和文件路径，不从内部实现推断成功。

外部 Trace/Lark 写入只有在需求明确授权对应身份、目标和时机时才执行；否则记录具体不可达前置条件。

## Evidence

保存候选 commit、二进制摘要、Feature/variant、argv、stdin、stdout、stderr、退出码、前后 Status、
State、Output、Event、Trace、Card、CLI transcript 和副作用回读。证明必须同时包含触发动作和稳定终态；
只保存最终输出、内部调用结果或测试通过记录都不足以证明真实用户路径。

证据位于 session 外的独立 evidence 目录，权限限制为当前用户。完整 transcript 可能含敏感信息；验证
默认只使用合成数据，不把历史 Requirement 或用户凭据复制进证据。

## Cleanup

清理本次创建的隔离安装、Requirement Root 和临时 Skill Root，只处理本次记录的精确路径。清理前复制
所有需要保留的事实，清理后回读 evidence 仍存在，并再次证明全局 current 未改变。没有可靠安全清理
入口时保留 session 并报告路径，不猜测或扩大删除范围。

完成标准：运行产生的临时资源不再存活，证据完整可读，结论明确为 passed、failed 或 blocked。
