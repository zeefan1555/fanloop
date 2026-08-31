---
name: fanloop-dev-agent-acceptance
description: 从冻结 candidate_head 做隔离安装，再由恰好一个无实现上下文的全新 Sub-agent 使用公开 CLI 完成真实黑盒验收。
---

# Agent 自动化验收

只验收工作树干净且 `HEAD == candidate_head` 的已 Review 候选。任何源码、测试或验证资产变化都使本 Step 失效。

## 隔离候选

1. 记录全局 `$HOME/.fanloop/current` 的真实目标、版本与 commit；运行 `./tests/run-unit`、`./tests/run-e2e`，并证明测试前后源码状态不变。
2. 创建临时目录，把 `FANLOOP_DATA_HOME`、`FANLOOP_CODEX_SKILLS_ROOT`、`FANLOOP_AGENT_SKILLS_ROOT`、`FANLOOP_TRAE_SKILLS_ROOT`、`FANLOOP_CLAUDE_SKILLS_ROOT` 全部指向其中的独立路径；清除 `BOTMUX_CHAT_ID`、`BOTMUX_SESSION_ID` 后，从 candidate_head 执行 `npm run install:local`。
3. 只使用隔离 `current/bin/fanloop` 回读 release 目标、version commit 和 Doctor。commit 必须精确等于 candidate_head，Doctor 必须 healthy。禁止修改或切换全局 current。

## 单个 Sub-agent 黑盒

派发恰好一个全新 Sub-agent。它只获得：隔离 CLI 的绝对路径、隔离环境变量、requirements.md 中已批准的 **1 至 3** 个场景、各自独立预期与停止边界；不获得实现上下文。

Sub-agent 必须为每个场景创建全新 Requirement Root，只能先读相关叶子 `--help`，再使用公开 CLI 驱动场景并记录 argv、stdout、stderr、退出码、前后 Status/Event/文件证据。不得读取源码、内部 Go 包、私有 helper、历史 Requirement 或其他实现材料；不得修改候选、push、建 PR、合并或更新全局 CLI；不得使用机器人、Botmux、用户凭据、Card/Trace 远端集成。

## 结论与产物

协调 Agent 复核 Sub-agent 原始证据，清理隔离安装和所有测试 Root，再证明全局 current 未变。把 candidate_head、测试、隔离安装、version/Doctor、场景证据、cleanup 和结论写入 `acceptance-report.md`。

用包含 Requirement 身份的稳定标题查找文档：零命中创建，唯一命中更新，多命中 blocked。发布唯一飞书验收交付报告，并语义回读正文非空、candidate_head、场景与结论一致。

全部场景通过才上报 `agent_acceptance_passed`、`acceptance_report_written`、`acceptance_document_published`。确定产品失败时上报 `agent_acceptance_failed`、两份报告和恰好一个最早责任回流；基础设施失败保持 blocked，不伪造产品失败或通过。
