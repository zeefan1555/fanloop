---
name: fanloop-dev-code-review
description: 审查已完成本地验证的 Fanloop CLI 本地 Diff。用于以记录的 main 基线核对 requirements/Spec/Tickets、仓库规则与 ADR；不要求已存在 PR。
---

# 自查修复

固定比较点为 bootstrap 记录的 `main` Commit，记录当前 HEAD，读取完整三点 diff、commit 列表、仓库规则、
Issue Workspace `requirements.md`、Spec/Tickets、本地验证报告和相关 ADR。不读取或要求 PR。分别检查：

- Standards：违反仓库明确规则、根因修复原则、安全要求或明显的重复、散弹修改、投机抽象。
- Design/Spec：技术方案、验收条件与实现逐项对应；识别遗漏、部分实现、超范围行为和
  看似实现但语义错误。
- Test Seam / Tracer Bullet：测试只能落在 requirements、Spec 与 Tickets 已确认 Test Seams；阻断
  私有实现或内部调用次数断言、用实现的同算法重算预期、先批量写测试再批量实现、无 Seam 却制造
  测试，以及缺少独立 Demo / verification path 的水平切片。
- Architecture：复核 IDL、durable storage、五份 Workflow YAML、Runtime 与 Release 边界；
  任意 `.thrift` diff 必须能定位到编码前人工审核记录，实际范围不得超过获批片段。
- Verification / Agent Eval：本地报告中的 baseline/candidate 必须是同一已确认 Case；涉及用户可观察
  CLI/Agent 行为、`skills/**` 或相关 Prompt 时，必须有真实 Agent Eval 且 Rubric 通过。报告的
  candidate commit、安装 Release commit、Agent Requirement commit 与当前 reviewed HEAD 必须完全一致；
  纯说明文档的 `N/A` 必须有可核对理由。缺失、失败、身份错误或 HEAD 漂移均为阻塞项。

工具能自动发现的格式问题直接运行工具，不写成评审意见。把结论写入
`review_report_path`，明确写入 reviewed HEAD；发现有效阻塞项时记录具体文件和行为并回流实现，
没有阻塞项时同时产出 `review_result=passed` 与报告路径。修复后旧验证和 Review 都失效，必须
重新验证、再 Review。不要因“可以更漂亮”扩大范围。
