---
name: fanloop-dev-agent-acceptance
description: 对同一 reviewed HEAD 安装 Fanloop Release，并用固定机器人身份执行真实黑盒验收和 acceptance-report.md。
---

# Agent Acceptance

只验收工作树干净、已提交，且 local-test-report.md、review-report.md、当前 HEAD 完全一致的候选。
先完整读取 ref/eval-playbook.md 与 ref/lark-agent-e2e.md；fanloop-dev-maintain-verification 的结果
不是 clean 时不得继续。

## 候选安装

1. 记录 reviewed HEAD 与三点 diff。
2. 从该工作树运行 npm run install:local。
3. 回读 fanloop version，要求 commit 精确等于 reviewed HEAD；再运行 fanloop doctor。
4. Q8=B：该安装切换正式 current slot；不执行、不声称也不伪造 managed Dev install、uninstall、
   自动回滚或恢复。

安装、版本或 Doctor 不一致属于基础设施 blocked；不使用旧安装继续。

## 真实机器人黑盒

固定 driver 是“使用 Fanloop 机器人” app cli_aafadbc67e799cdc，target 是“FanLoop 机器人”
app cli_a9245f0fddf8dbc8，唯一群是 oc_d532c3a5eda84c60728ab174b0ef671a。

每轮从在线 driver source session 实时回读 bots list，校验 chat、isSelf driver app 与 target app；
open ID 只记录本轮 driver 视角值。随后通过 botmux dispatch --bot-app
cli_a9245f0fddf8dbc8 创建一个全新顶层话题，要求 target 创建全新 Requirement、只使用刚安装 Release
的公开 Fanloop CLI、执行 requirements.md 冻结的 1–3 个 Case，并在合法 Human Step 停止。Candidate
执行每条 Fanloop 命令时必须使用
`env -u BOTMUX_CHAT_ID -u BOTMUX_SESSION_ID /Users/bytedance/.fanloop/current/bin/fanloop`；这只隔离内层
CLI，避免它捕获外层验收话题并以用户身份创建或同步飞书 Trace，外层真实 driver/target 身份保持不变。

用同一 driver session 回读 dispatch、history、quoted 与 target Requirement 的 Status、Events、Card、
CLI 日志。不得使用用户 token、错误 app/群、旧话题、旧 Requirement、截图替代回执，或让机器人批准、
合并、发布。若 Requirement 出现 Card Binding、Trace Integration 或 trace_document_bound / trace_sync_started /
trace_synced Event，按未授权外部写入判 governance_failed。

## 报告与结论

在 Issue Workspace 的唯一 acceptance-report.md 中记录：

- reviewed HEAD、工作树状态、local/review report HEAD；
- Verification 维护 clean 证据；
- npm install、version、doctor 的命令、退出码和回读；
- chat、driver/target app 与本轮 open ID、source session、根消息、thread；
- 全新 Requirement Root、candidate commit、Case、命令/卡片/响应和停止原因；
- Rubric 每项证据、得分、总分和分类。

Rubric 10/10 且所有护栏通过才输出 passed。确定产品失败输出 failed，并在 requirements_changed、
technical_solution_changes_requested、implementation_changes_requested 中恰好选择一个原因。
身份、群、网络、接单、回读或证据不可用时输出 blocked，留在当前 Step，不提交 Result，也不回退
用户身份。任何源码或测试资产变化都会使本报告、安装与 Review 失效。
