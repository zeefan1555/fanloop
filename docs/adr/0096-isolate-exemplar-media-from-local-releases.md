---
status: accepted
date: 2026-09-16
amends: ADR-0095
---

# 将范文图片与本地 CLI Release 隔离

技术方案范文的 81 张图片和画板预览约占 38 MiB，而本机 CLI 二进制约 6 MiB。图片仅用于维护范文和按需视觉对照，不参与 Runtime、Workflow 推进、状态解释或普通文本审校。

范文图片迁移到源码仓库顶层 `exemplars/technical-solution/<主题>/images/`，继续受 Git 版本控制并保留原字节。`skills/` 保留范文正文、来源记录、导读、图片引用和说明；`./scripts/build-local.sh` 继续按原样复制 `skills/`，但不复制顶层 `exemplars/`。因此本地 Build 与安装 Release 不再携带范文图片。

普通写作与审校依据范文正文、导读和图片说明完成，不因 Release 中没有范文图片而阻塞。只有任务明确要求视觉对照时，才从源码仓库查看对应图片；图片或看图能力不可用时，只报告该项未覆盖范围。当前方案自身引用的图片和主架构图要求不变。

本决策修订 ADR-0095 的“完整范文原始资料随 Skill 复制”和“完整范文配套交付”：完整性边界收敛为 CLI、Workflow、Skill、范文文本及其摘要校验。它不修改五份 Workflow YAML、Step、Route、Condition、Prompt/SkillBinding、Thrift IDL、Runtime、State、Storage 或公开 CLI 契约。

本变更选择 `e2e` 验证档：聚焦验证源码图片仍完整、Build 和安装目录不含范文图片、Manifest/Doctor 接受文本 Skill，并从同一工作树运行 `./tests/run-unit` 与 `./tests/run-e2e`。
