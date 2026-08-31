---
name: fanloop-dev-grill-with-docs
description: 编排 Fanloop CLI 需求澄清，把已确认决策、公开 CLI 验收场景和完整改造计划沉淀到 requirements.md 与唯一飞书需求文档。
---

# Grill with Docs

同时使用 Workflow 绑定的 `fanloop-dev-grilling` 与 `fanloop-dev-domain-modeling`：前者维护 design tree 和当前 frontier，后者挑战含混术语并整理 CONTEXT/ADR 候选。可查的代码、仓库、日志和环境事实由当前 Agent 自查，不反问人。

每轮答案到达后立即更新 Issue Workspace 根目录的 `requirements.md`。文件至少记录：问题 ID、选择与理由、决策、约束、非目标、验收条件、实现必要性、公开 Test Seam、完整改造计划、CONTEXT/ADR 候选和 Open Questions。确认前只写候选，不修改源码、`CONTEXT.md` 或 ADR。

## 验收场景

为本次变化选取 **1 至 3** 个端到端公开 CLI 场景。每个场景必须写明：

- 用户意图、前置条件与全新 Requirement Root；
- 可从叶子 `--help` 发现的公开 CLI 命令与输入；
- 独立预期、退出码、stdout/stderr、State/Event/文件等可观察证据；
- 隔离边界、cleanup、停止位置和明确不覆盖项。

独立预期来自已确认需求，不得从实现算法反推。场景只供后续全新 Sub-agent 验收，不包含源码路径、私有 helper 或实现提示。Test Seam 数量保持最少；没有可靠独立预期时写明 `TDD: not applicable`，不得造测试。

## 飞书需求文档产物

Open Questions 为空且 `requirements.md` 完整后，使用当前宿主的 `lark-doc` 能力发布 Markdown：

1. 标题包含 Requirement 身份并保持稳定；同一 Requirement 始终更新同一份唯一飞书需求文档。
2. 先按稳定标题查找：零命中才创建，唯一命中更新，多命中立即 blocked。创建结果不确定时先重查，不重复创建。
3. 创建或更新后按返回 URL 语义回读，正文必须非空，且决策、验收场景、完整改造计划与最终 `requirements.md` 一致。
4. 成功后返回 `requirements_grilled=requirements.md` 与 `requirements_document_published=<URL>`。

飞书不可用、标题多命中、写入失败或语义回读失败时保持 blocked，不返回成功 Condition。
