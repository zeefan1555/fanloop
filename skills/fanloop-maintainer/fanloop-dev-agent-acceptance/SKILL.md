---
name: fanloop-dev-agent-acceptance
description: 用固定机器人身份并行运行两个真实黑盒 Case，并验证内层 Fanloop 与用户身份、Card 和远端 Trace 完全隔离。
---

# 机器人端到端验收

只验收 CI 已在 candidate_head 全绿、工作树只读且 PR head 未漂移的候选。先完整读取
ref/lark-agent-e2e.md；Eval 阶段的冻结 Case、评分和 acceptance-report.md 必须属于同一 SHA。

## 候选与身份

1. 从候选工作树安装 Release，回读 fanloop version 的 commit 等于 candidate_head，fanloop doctor 健康。
2. driver 固定为“使用 Fanloop 机器人” app cli_aafadbc67e799cdc，target 固定为“FanLoop 机器人”
   app cli_a9245f0fddf8dbc8，群固定为 oc_d532c3a5eda84c60728ab174b0ef671a。
3. open ID 每轮从 driver 在线 source session 实时解析；不得使用用户 token、用户身份、旧话题或旧 Requirement。

## 两个并行 Case

通过外层 botmux dispatch 建立两个全新顶层话题并并行派发。每个 target 在独立目录创建全新
Requirement，只使用刚安装 Release 的公开 CLI，并在 merge_code 前停止。内层 CLI 命令必须清除
BOTMUX_CHAT_ID 与 BOTMUX_SESSION_ID，且不得携带用户凭证。

外层 Botmux 只负责机器人通信，不得泄漏到内层 Requirement。以下任一事实出现都判
governance_failed：Card Binding、Trace Integration、trace_document_bound、trace_sync_started、远端
trace_synced、用户 Trace 文档、用户 CLI 日志文档、--as user 或任何用户 token。不得为了通过而回退
用户身份或手工补资源。

## 证据与结论

用同一 driver session 回读 dispatch、history、quoted、两个 target 回复以及 Requirement 的 Status、
Events、Card、CLI 输出和文件树。acceptance-report.md 记录 candidate_head、安装/Doctor、两个话题、
身份、Case、命令、前后状态、隔离断言、得分和停止原因。

两个 Case 均满足独立预期且治理断言全部通过才输出 passed。确定失败输出 failed，并在需求、方案、
实现、验证技能、功能地图中恰好选择一个最早责任层。身份、网络、接单或回读不可用时保持 blocked，
不提交 Result。任何候选变化都会使本验收失效。
