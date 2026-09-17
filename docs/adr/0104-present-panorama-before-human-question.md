---
status: accepted
date: 2026-09-17
amends: ADR-0086, ADR-0102
---

# Panorama 先于人工问题展示

Fanloop 对齐 Treeloop `47c0d47336df00a9ea9545f9328195fbd7ee6f0b` 的 Panorama 投递时序：进入新
Step 后，Panorama 必须作为第一条用户可见消息单独展示，不得先发送进度前缀、澄清问题或确认问题。
`local_agent` 使用宿主可继续执行的用户可见消息通道原样发送本次 render 的 `data.content`；展示成功后
同一轮继续执行 Prompt，真正需要 human 输入的问题随后发送。

最终普通回复不再重新 render 或重复 Panorama。`panorama_presented` 与 `panorama_card_published` 仍只在
本次 Render、Deliver 与 Verify 全部成功后成立，并继续返回当前 Fanloop 契约要求的精确
`snapshot_path`。Botmux、AIME、Aiden 的唯一通道、禁止 fallback/双发/旧快照扫描以及 Workflow Route
语义保持不变。

本决策同步适用于 `fanloop-maintainer`、`technical-solution-design` 与 `material-flashcards` 三个 Panorama
Skill，以及通用 `fanloop-workflow` 入口；不修改 Workflow YAML、Step、Route、Condition、Thrift、State
或 Runtime。

用户于 2026-09-17 根据实际界面截图明确要求“全景图在上，需要澄清的问题在下”，并指定复用 Treeloop
全景渲染 Skill 的现有规则。
