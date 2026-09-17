---
status: accepted
date: 2026-09-17
supersedes: ADR-0106, ADR-0107
supersedes_in_part: ADR-0100, ADR-0101
amends: ADR-0103, ADR-0104, ADR-0105
---

# 将 Maintainer 收敛为四步自主验证闭环

`fanloop-maintainer` 从 3 Stage / 3 Job / 9 Step 改为 4 Stage / 4 Job / 4 Step：

```text
Define:  define_verification_contract
Build:   build_until_verified
Certify: certify_candidate
Deliver: merge_and_update_local
```

四步不是效率指标，而是把需求契约、持续实现验证、独立认证和交付四类不同责任分别放入一个深模块。
Fanloop 已有公开 `fanloop verify`、Verification Skill 与 Feature Map，本决策只收敛编排，不增加新的
Verify Service、Scenario DSL、Workflow Runtime、IDL 或持久化 Schema。

## Step 契约变化

| 原 Step | 新 Step | 变化 |
| --- | --- | --- |
| `bootstrap_techdesign`、`clarify_requirements` | `define_verification_contract` | 合并工作区准备、需求澄清、Acceptance Set、Feature Impact Set 与主 Agent批准。 |
| `design_technical_solution`、`confirm_technical_solution`、`implement_code` | `build_until_verified` | 方案按复杂度生成；实现、测试、验证资产维护、真实 Feature 验证和自修复成为同一循环。 |
| `review_code`、`execute_agent_acceptance`、`confirm_main_agent_acceptance` | `certify_candidate` | 冻结身份，并行执行独立 Review 与无源码上下文黑盒验证，再记录主 Agent最终决定。 |
| `merge_and_update_local` | `merge_and_update_local` | 保留 ID；只交付已认证候选，main 前进时不修改代码而是回 Build。 |

删除八个旧 Step，新增三个 Step，保留一个 Step；四个新 Step 均为 `agent`。Stage 和 Job 各从三个改为
四个。旧 Step ID、人工跳转 Route 和旧 Condition 不保留兼容别名；当前 Schema Requirement 按
ADR-0105 使用当前 Bundle，若当前 Step 已不存在则明确返回 `WORKFLOW_MISMATCH`。

## 验证闭环

Define 将一至三个用户端到端场景作为 Acceptance Set，同时记录覆盖全部受影响 Feature、入口和关键状态的
Feature Impact Set。Build 反复执行最小修改、聚焦测试、`./tests/run-unit` 和真实公开 CLI 验证；只有新的
可证伪假设才继续重试，环境不可达或无新证据时 blocked。用户表面变化必须同步维护 Verification Skill
与 Feature Map，验证资产与源码、测试一样属于候选，变化后旧证据全部失效。

Certify 冻结 base、head、Git tree、候选 binary SHA-256 和验证资产摘要。独立 Reviewer 可读源码，独立
Verifier 不读源码且只使用隔离候选、叶子 Help、Feature Map 和 Acceptance Set。两者均通过后主 Agent
决定是否接受候选。报告只引用原始 run、manifest、snapshot 和回执，不复制另一份证据正文。

`verify smoke` 的现有实现证明隔离安装、初始化、第一次 Route 推进、取证和清理，不代表完整 Feature
验收或全生命周期；真实 Feature 由普通公开 CLI 与 Feature Map 驱动，完整生命周期和 Route Matrix 继续
由 CI `requirement-e2e` 覆盖。

## 交付

Deliver 首次 fetch 时若 `origin/main` 已前进，只记录新 base 并回 Build；它不得合入 main 或产生未认证
代码。Build 集成后产生新候选并完整重走 Certify，因此不再保留单独的 main 集成确认。冲突若迫使需求、
Workflow、IDL、Schema 或公开语义变化，则回 Define 并重新取得相应人工批准。

main 未变化时，Deliver 幂等创建或更新唯一 PR，等待精确 certification head 的 required checks，随后用
`--auto --squash --match-head-commit` 合并。合并后要求 merge commit tree 等于已认证 head tree，再只把
本 Requirement 的源码 worktree 切到该 merge commit，从该提交更新全局 current，并运行 Doctor 与
post-merge smoke。PR 已合并后的本地失败只重试同一 merge commit。

## 产物与边界

固定产物收敛为 `requirements.md`、`verification-report.md`、`certification-report.md` 和
`delivery-record.md`。飞书文档只是可选阅读投影，不再是 Route 必需事实。Panorama 仍在进入每个 Step
时展示一次。Spec 与 Tickets 只在跨模块协调、方案存在明显分支或需要独立并行任务时生成，不是 Workflow
门禁。周期性 Verification maintenance 保持在 Workflow 外，稳定后以 `clean|changed|blocked` 自动任务运行。

人工审核记录：用户提供并更新《Fanloop 自迭代自主验证闭环改造计划》，随后明确要求按照该计划从最新
main 创建开发分支，并补充“不使用仓库 Maintainer 流程，直接修改代码验证”。该授权覆盖本 ADR 列出的
四步 Step 集合、名称、顺序、executor、删除人工跳转 Route 和上述交付回流语义。
