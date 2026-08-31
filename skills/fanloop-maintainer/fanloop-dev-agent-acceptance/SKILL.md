---
name: fanloop-dev-agent-acceptance
description: 用固定机器人身份并行运行两个真实黑盒 Case，并验证内层 Fanloop 与用户身份、Card 和远端 Trace 完全隔离。
---

# 机器人端到端验收

只验收 CI 已在 candidate_head 全绿、工作树只读且 PR head 未漂移的候选。先完整读取
ref/lark-agent-e2e.md；Eval 阶段内容寻址的 Playbook、原始 brief/Rubric、评分和 acceptance-report.md
必须属于同一 SHA。

## 当前候选的真实本地安装

1. 在候选工作树确认工作树干净、`git rev-parse HEAD` 等于 `candidate_head`，PR head 未漂移。
2. 切换全局 current 前，从本次 Status 返回的当前 Skill 绝对路径取得 Skill 目录，并固定初始化该
   Requirement 的完整控制器 Release：

   ~~~bash
   controller_binary="$(
     "<CURRENT_SKILL_DIRECTORY>/scripts/pin-controller-release.sh" \
       "<ABSOLUTE_REQUIREMENT_ROOT>"
   )"
   controller_home="<ABSOLUTE_REQUIREMENT_ROOT>/bound-release-home"
   ~~~

   脚本必须先证明旧 current 能读取当前 State，再使用现有 `__install` 原子安装到
   `bound-release-home`，最后证明固定副本的 Status 与安装级 Doctor 成立。失败保持 `blocked`，不得
   安装候选、猜测 Release、扫描 Git 历史或修改 State。把固定控制器路径、`version`、Workflow digest
   和 Doctor 写入验收报告。
3. 只对候选安装子进程清除所有隔离覆盖，然后从候选工作树执行真实本地安装：

   ~~~bash
   env \
     -u FANLOOP_DATA_HOME \
     -u FANLOOP_CODEX_SKILLS_ROOT \
     -u FANLOOP_AGENT_SKILLS_ROOT \
     -u FANLOOP_TRAE_SKILLS_ROOT \
     -u FANLOOP_CLAUDE_SKILLS_ROOT \
     -u BOTMUX_CHAT_ID \
     -u BOTMUX_SESSION_ID \
     npm run install:local
   ~~~

4. 要求 `$HOME/.fanloop/current` 指向 `$HOME/.fanloop/releases/` 内的 Release；使用
   `$HOME/.fanloop/current/bin/fanloop` 回读 `version`，其中 commit 必须等于 `candidate_head`，随后运行
   `doctor` 并要求 `healthy`。把实际 `readlink`、二进制路径、版本和 Doctor 输出写入验收报告。
5. 候选切换后，外层维护 Requirement 的全部 Status、Progress、Result 与 Card 只使用
   `controller_binary`，并设置 `controller_home` 与其 `skill-roots/{codex,agent,trae,claude}`；两个机器人
   和其全新 Requirement 只使用默认 `$HOME/.fanloop/current/bin/fanloop`。禁止把两者互换。
6. 禁止用 `go build -o`、`mktemp` 下的候选二进制、私有 bin 或设置临时 `FANLOOP_DATA_HOME` 代替安装；
   安装成功后保留该候选为本机 `current`，不恢复旧版本。

固定控制器、安装、current、commit 或 Doctor 任一证据不成立时保持 `blocked`，不得派发机器人。

## 机器人身份

1. driver 固定为“使用 Fanloop 机器人” app cli_aafadbc67e799cdc，target 固定为“FanLoop 机器人”
   app cli_a9245f0fddf8dbc8，群固定为 oc_d532c3a5eda84c60728ab174b0ef671a。
2. open ID 每轮从 driver 在线 source session 实时解析；不得使用用户 token、用户身份、旧话题或旧 Requirement。

## 两个并行 Case

校验 `eval_playbook_path` 文件名摘要，以及其中恰好两个 brief_sha256 和 rubric_sha256。通过外层
botmux dispatch 建立两个全新顶层话题，并将这两个原始 brief 路径直接作为 `--brief-file` 并行派发。
“全新”只指话题、目录和 Requirement；禁止生成、复制、选择或改写题面与 Rubric。每个 target 只使用
刚安装 Release 的公开 CLI，并在 merge_code 前停止。内层 CLI 命令必须清除 BOTMUX_CHAT_ID 与
BOTMUX_SESSION_ID，且不得携带用户凭证。

外层 Botmux 只负责机器人通信，不得泄漏到内层 Requirement。以下任一事实出现都判
governance_failed：Playbook/brief/Rubric 摘要不一致、Card Binding、Trace Integration、trace_document_bound、trace_sync_started、远端
trace_synced、用户 Trace 文档、用户 CLI 日志文档、--as user 或任何用户 token。不得为了通过而回退
用户身份或手工补资源。

## 证据与结论

用同一 driver session 回读 dispatch、history、quoted、两个 target 回复以及 Requirement 的 Status、
Events、Card、CLI 输出和文件树。acceptance-report.md 记录 candidate_head、默认 current 的 readlink、
固定控制器的路径/版本/Workflow digest/Doctor、候选实际安装二进制与版本/Doctor、两个话题、
身份、Case、命令、前后状态、隔离断言、得分和停止原因。

两个 Case 均满足独立预期且治理断言全部通过才输出 passed。确定失败输出 failed，并在需求、方案、
实现、验证技能、功能地图中恰好选择一个最早责任层。身份、网络、接单或回读不可用时保持 blocked，
不提交 Result。任何候选变化都会使本验收失效。
