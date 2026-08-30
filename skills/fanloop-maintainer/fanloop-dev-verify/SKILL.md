---
name: fanloop-dev-verify
description: 按 targeted 或 e2e 风险档复验 Fanloop CLI 本地源码，并在需要时用固定机器人身份执行真实 Agent E2E。不读取或等待远端 MR checks。
---

# 按风险验证 Fanloop CLI

1. 读取确认需求、仓库根 `FEATURE_MAP.md`、测试计划和其中已确认的 Verification Case；记录当前分支、
   HEAD、`git status --short` 与执行前 diff 摘要。纯文档、
   self-iteration Skill、未改变 Step/Route/Condition/executor 与 CLI/IDL/Storage/发布/测试入口的
   Prompt/SkillBinding，或具备可靠聚焦 seam 的局部行为可以选择 `targeted`；推进语义、代码/IDL、
   durable state、发布/安装/打包、测试入口或影响不确定时必须选择 `e2e`。
2. 在 `candidate` 上原样复用 Case 的 baseline 输入、前置条件和观察项，执行全部聚焦命令；同时记录
   candidate commit、退出码、stdout/stderr 与状态/文件变化。`targeted` 全部零退出后上报
   `targeted_validation_passed`；必须与最终 `local_test_report_written` 在同一次结果中提交。
3. `e2e` 在聚焦命令通过后，从仓库根运行 `./tests/run-unit` 与 `./tests/run-e2e`；两者零退出后
   上报 `e2e_entrypoint_passed`；必须与最终 `local_test_report_written` 在同一次结果中提交。
4. 当变更涉及用户可观察 CLI/Agent 行为、`skills/**` 或相关 Prompt 时，先从同一最终工作树运行
   `npm run install:local`，再回读 `fanloop version` 并确认其中 commit 等于 HEAD，随后运行
   `fanloop doctor`。读取 `ref/eval-playbook.md` 与 `ref/lark-agent-e2e.md`，只发送一次全新 candidate
   话题并由当前验证 Agent 独立评分。纯说明文档可以记录 Agent E2E `N/A` 及理由。
5. 机器人身份、精确群、在线 source session、网络或接单证据不可用时，记录基础设施 `blocked`，
   不上报通过或产品失败；绝不回退用户 token。目标 Agent 到达 Human Step 时记录并停止，不得代批。
6. 把 profile、baseline/candidate 对照、命令、时间、退出码、HEAD、安装 Release/commit、执行前后源码
   状态、机器人根消息/thread/Requirement/Eval 证据和可选 E2E 报告路径写入同一个
   `local_test_report_path`；确认本地验证、Agent Eval 与源码状态都覆盖同一 HEAD 后上报
   `local_test_report_written`。
7. 任一必需命令或 Rubric 失败时保留错误摘要并上报 `local_validation_failed` 回到代码实现；不降档、
   不伪造。源码或测试资产改变后，旧本地验证和旧 Agent Eval 立即失效。

本 Skill 不要求先有 MR，不读取或等待远端 checks，不批准 MR、不合并、不发布。
