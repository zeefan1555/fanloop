---
name: fanloop-dev-verify
description: 按 targeted 或 e2e 风险档复验 Fanloop CLI 本地源码并写 local-test-report.md。不安装候选 Release，不运行机器人验收。
---

# 按风险验证 Fanloop CLI

1. 读取确认需求、仓库根 FEATURE_MAP.md、测试计划和已确认的 Verification Case；记录当前分支、
   HEAD、git status --short 与执行前 diff 摘要。纯文档、自迭代 Skill、未改变
   Step/Route/Condition/executor 与 CLI/IDL/Storage/发布/测试入口的 Prompt/SkillBinding，或具备可靠
   聚焦 Seam 的局部行为可以选择 targeted；推进语义、代码/IDL、durable state、发布/安装/打包、
   测试入口或影响不确定时必须选择 e2e。
2. 在 candidate 上原样复用 Case 的 baseline 输入、前置条件和观察项，执行全部聚焦命令；记录
   candidate commit、退出码、stdout/stderr 与状态/文件变化。targeted 全部零退出后上报
   targeted_validation_passed，并与 local_test_report_written 在同一次 Result 提交。
3. e2e 在聚焦命令通过后，从仓库根运行 ./tests/run-unit 与 ./tests/run-e2e；两者零退出后上报
   e2e_entrypoint_passed，并与 local_test_report_written 在同一次 Result 提交。
4. 把 profile、baseline/candidate 对照、命令、时间、退出码、HEAD、执行前后源码状态和可选 E2E
   报告路径写入同一个 local_test_report_path；确认全部本地验证覆盖同一 HEAD。
5. 任一必需命令失败时保留错误摘要并上报 local_validation_failed 回到代码实现；不降档、不伪造。
   源码或测试资产改变后，旧本地验证立即失效。

候选 Release 安装、逐 Feature 维护与真实机器人黑盒属于 Review 后的
fanloop-dev-maintain-verification 和 fanloop-dev-agent-acceptance。本 Skill 不要求先有 MR，
不读取或等待远端 checks，不批准 MR、不合并、不发布。
