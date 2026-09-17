---
name: fanloop-dev-grill-with-docs
description: 生成 Fanloop 变更的 requirements.md、Acceptance Set、Feature Impact Set，并记录主 Agent需求决定。
---

# Define Verification Contract

执行子 Agent先从仓库、日志、当前五份 YAML 和 ADR 获取可观察事实，只向人询问无法从现状得到的产品决定。
使用 `fanloop-dev-grilling` 收敛 frontier，使用 `fanloop-dev-domain-modeling` 消除含混术语。

在 Issue Workspace 根目录写入并回读唯一 `requirements.md`，至少包含：目标、范围、非目标、风险边界、
行为契约、ADR impact、停止条件、Acceptance Set、Feature Impact Set 和 Open Questions。

- Acceptance Set：一至三个用户端到端场景，每个场景包含公开入口、独立预期、退出码、可观察状态、
  副作用、证据与停止位置。
- Feature Impact Set：本次受影响的全部 Feature、公开入口、关键成功状态和关键失败状态。它决定 Build
  的回归范围，Acceptance Set 不是覆盖上限。

Open Questions 为空后，执行子 Agent向主 Agent申请批准或拒绝，并使用 `fanloop-dev-decision-receipt`
记录明确回复。批准前不得修改受管代码。飞书文档可按需要作为阅读投影，但创建、发布或回读不是 Workflow
Route 必需事实。

批准时返回 `verification_contract_written`、`requirements_approved` 与
`requirements_decision_recorded`；拒绝时返回对应拒绝组合。
