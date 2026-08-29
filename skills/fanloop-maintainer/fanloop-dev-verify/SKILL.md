---
name: fanloop-dev-verify
description: 按 targeted 或 e2e 风险档验证当前 Fanloop CLI 本地源码，记录 HEAD、命令、退出码和源码状态。不读取或等待远端 MR checks。
---

# 按风险验证 Fanloop CLI

1. 读取确认需求和测试计划，记录当前分支、HEAD、`git status --short` 与执行前 diff 摘要。纯文档、
   self-iteration Skill、未改变 Step/Route/Condition/executor 与 CLI/IDL/Storage/发布/测试入口的
   Prompt/SkillBinding，或具备可靠聚焦 seam 的局部行为可以选择 `targeted`；推进语义、代码/IDL、
   durable state、发布/安装/打包、测试入口或影响不确定时必须选择 `e2e`。
2. 执行计划中的全部聚焦命令。`targeted` 全部零退出后上报 `targeted_validation_passed`。
3. `e2e` 在聚焦命令通过后，从仓库根运行 `./tests/run-unit` 与 `./tests/run-e2e`；两者零退出后
   上报 `e2e_entrypoint_passed`。
4. 记录 profile、命令、时间、退出码、HEAD、执行前后源码状态和可选 E2E 报告路径到
   `local_test_report_path`；确认测试未改变源码状态后上报 `local_test_report_written`。
5. 任一必需命令失败时保留错误摘要并上报 `local_validation_failed` 回到代码实现；不降档、不伪造。

本 Skill 不要求先有 MR，不读取或等待远端 checks，不批准 MR、不合并、不发布。
