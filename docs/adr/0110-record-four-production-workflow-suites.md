---
status: accepted
date: 2026-09-22
amends_inventory: ADR-0062, ADR-0108
amended_by: ADR-0117, ADR-0118
---

库存中的旧制卡 ID 已由 [ADR-0117](0117-replace-decks-with-remnote-flashcards.md) 替换为 `remnote-flashcards`，其余本决策不变。

# 记录四套生产 Workflow 清单（仅库存）

生产 live 配置现为四套 Workflow，路径均为 `workflows/<id>/`：

- `technical-solution-design`
- `fanloop-maintainer`
- `remnote-flashcards`
- `pstack-native`

历史 ADR（尤其 ADR-0062「当前三套 Bundle」、ADR-0108「三套生产 Workflow 的十五份 YAML」）中的套数 /「十五份」表述仅作当时库存快照；**对本 ADR 之后的套数盘点以本 ADR 为准**。本 ADR **不**改动既有三套 Workflow 的 Step `id`/`name`/顺序/`executor`。制卡套的 id 由 ADR-0117 改为 `remnote-flashcards`。第四套引入时的 id 是 `pstack-delivery`；ADR-0118 把它改为 `pstack-native`，并替换其 Step 拓扑。上表是修订后的库存。

五文件 Bundle 形状、live 根加载与 Dashboard 边界仍遵循 ADR-0062 / ADR-0108 的未冲突条款。ADR-0049 等处的「三套 Skill 根目录」指客户端 Skill Root，与 Workflow 套数无关，不在本 ADR 修订范围。
