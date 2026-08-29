---
name: commonloop-dev-grill-with-docs
description: 编排 Commonloop CLI 需求澄清。用于在 Issue Workspace 组合 commonloop-dev-grilling 与 commonloop-dev-domain-modeling，把每轮人工决策立即沉淀到 requirements.md。
---

# Grill with Docs

同时使用 `commonloop-dev-grilling` 和 `commonloop-dev-domain-modeling`。前者维护 design tree 和当前
frontier，后者挑战含混领域词并整理 CONTEXT/ADR 候选。Workflow 必须显式绑定这两个原子 Skill；
不要依赖宿主递归解析本文件中的引用。

每轮人的答案到达后立即更新 Issue Workspace 根目录的 `requirements.md`，再计算下一轮 frontier。
文件至少记录问题 ID、候选与推荐、人的选择与理由、消息 ID、约束、非目标、验收条件、实现
必要性、Open Questions 和 CONTEXT/ADR 候选。还必须包含可直接展示的“完整改造计划”，依次覆盖
目标、现状问题、逐项改造、影响文件/契约、保持不变与非目标、验证计划、交付边界；不得仅引用
历史对话或外部文档。确认前只写候选，不修改仓库 `CONTEXT.md` 或 ADR。

验证计划必须列出拟确认的公开 Test Seams。每个 Seam 写明公开入口、覆盖行为、基线失败信号、
选择理由、不覆盖项和 TDD 适用性；优先复用能覆盖需求的最高层现有 Seam，数量尽可能少。
没有可靠 Seam 或独立预期时明确写 `TDD: not applicable`，不得为满足流程制造测试。Test Seams 随完整改造
计划进入现有 `confirm_requirements` 门禁，不增加第二个人工审批 Step。

frontier 为空且人确认共享理解后才完成。可查的代码、仓库、日志和环境事实由当前 Agent 自查；
不得把事实调查转给用户，也不得要求子代理。

## 飞书需求文档产物

frontier 为空且 `requirements.md` 已包含完整改造计划后，使用当前宿主的 `lark-doc`
能力以 Markdown 发布需求文档：

1. 使用包含 Requirement 标题的稳定文档标题。同一 Requirement 始终更新同一份飞书文档，不为回流或新一轮澄清重复创建。
2. 首次发布前按稳定标题查找：唯一命中则更新，零命中才创建，多命中则停止。创建结果不确定时必须先查找，不得直接重试创建。
3. 创建或更新后使用返回 URL 回读文档，确认回读正文非空且与最终 `requirements.md` 对应。
4. 只有发布和回读都成功时，才向当前 Step 返回：

```text
condition_id=requirements_grilled
requirements_path=requirements.md
condition_id=requirements_document_published
requirement_document_url=<已回读验证的飞书文档 URL>
```

飞书能力不可用、标题多命中、创建/更新失败或回读正文为空时返回
`progress_status=blocked` 和真实原因；不返回任何成功 Condition。
