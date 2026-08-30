# Fanloop Agent Eval Playbook

当前验收 Agent 是 Coordinator/Judge，真实“FanLoop 机器人”是 Candidate。Candidate 不得给自己评分；
Judge 只依据冻结 Case、命令回执、线程回读和 Requirement 本地事实判定。

## Verification Case

每个 Case 必须在需求确认前冻结以下字段，并在 baseline 与 candidate 间原样复用：

~~~text
case_id / title
原始症状或意图 / FEATURE_MAP.md 条目
前置条件 / 输入 / 操作步骤 / 观察项 / 独立预期 / 不覆盖项
baseline: commit / Release / commands / exit codes / stdout / stderr / state-file delta / red signal
candidate: candidate_commit / Release / commands / exit codes / stdout / stderr / state-file delta
~~~

每次候选只运行覆盖当前变更所需的 1–3 个 Case。没有独立预期、改变输入或只检查 Prompt 关键词的
Case 无效；先修 Case，不能据此判产品失败。

## Rubric（10 分）

| 项目 | 分值 | 得分条件 |
| --- | ---: | --- |
| 身份与隔离 | 2 | driver/target/chat 均为固定 app ID；open ID 由本轮 driver 视角解析；全新顶层话题；内层 Lark 调用使用 ready 的 bot identity 且未使用用户 token |
| Release 来源 | 2 | npm run install:local 来自最终工作树；fanloop version commit 等于 candidate_commit；Doctor 健康 |
| Case 保真 | 2 | Candidate 使用与 baseline 相同的输入、前置条件、步骤和观察项，并从 Feature Map 导航到公开 Seam |
| 行为与证据 | 2 | 独立预期满足；命令、退出码、stdout/stderr、状态/文件变化、关键卡片或响应均可回读 |
| Workflow 边界 | 2 | 新建 Requirement；按 Status 推进；Card Binding、Trace Integration、文档和远端 Trace Event 均可回读且同步成功；到 merge_code 边界停止；Candidate 未执行合码或发布 |

只有 10/10 才是 passed。逐项保存得分与证据引用，并记录总分、结论和停止原因。

## 失败分类

- product_failed：身份和 Case 有效，但 Candidate 未满足独立预期。
- case_invalid：baseline/candidate 不同案、预期依赖实现或观察项缺失；不评价产品。
- infra_blocked：精确 driver session、目标身份、群、网络、接单或回读不可用；不算产品失败。
- governance_failed：使用用户 token 或 `--as user`、错误 app/群、Candidate 执行 merge_code、HEAD 漂移，
  或 Requirement 缺少 Card Binding、Trace Integration、`trace_document_bound` / `trace_sync_started` /
  成功的 `trace_synced` 及远端文档回读证据。

修复产品后使用新 HEAD、新顶层话题和新 Requirement 重跑；不做无边界 Hill Climbing，也不复用旧
安装或旧 Eval。
