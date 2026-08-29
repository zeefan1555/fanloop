---
name: commonloop-dev-domain-modeling
description: 澄清 Commonloop CLI 领域语言。用于反馈中的术语含混、与 CONTEXT.md 冲突，或需要区分 Issue Workspace、Requirement、Workflow、Step、Rule 等概念时。
---

# 领域建模

先读仓库 `CONTEXT.md`。主动把模糊词映射到已有领域词，并用具体场景检查边界；代码事实与描述冲突时明确记录冲突。

需求确认前，把稳定术语和架构选择分别写成 Issue Workspace `requirements.md` 的 CONTEXT/ADR 候选，不修改生产仓库。人确认后，只在新术语稳定且不含实现细节时更新 `CONTEXT.md`；仅当决策难以逆转、缺少上下文会令人意外且存在真实权衡时才新增 ADR。不要把 problem、spec 或实现细节塞进领域词典。
