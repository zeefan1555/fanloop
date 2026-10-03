---
name: remnote-verify
description: 检查 RemNote 闪卡内容质量，以及闪卡和 Wiki 创建、修改或修复后的实际状态。由制卡与 Wiki 工作流在交付前调用；也用于用户要求检查位置、HOME Tag 或 WIKI 混入闪卡时。
---

# RemNote 写后验证

## 草稿立即审

prepare_cards_and_wiki 每完成一批草稿即派独立只读助手审查，不等待 Wiki 分支。
共用来源、模式和学习目标；审查普通卡的来源一致、单一记忆目标、独立理解、术语解释及下述内容规则。
原文模式逐段比对指定原文，不改写；全部重复时仍检查复用决定与实际答案补充。
回执包含 material_digest、draft_digest、来源引用、覆盖 card_keys 及每张 PASS/FAIL 和原因。
助手不写卡、不写 Wiki、不代替作者改稿。作者只修失败项，助手只复审改变的卡；新增术语返回 Wiki 清单。
未覆盖、未结束、旧稿证据或失败项非空不满足 draft_quality_passed。

使用 `remnote-cli` 读取真实状态。输入本批次内容 Rem ID、预期卡片 ID、HOME Tag ID、Wiki 页面 ID，以及新增／更新／迁移动作；缺少 ID 时先定向查找。仅请求检查时保持只读；创建或修复任务中的失败项在已授权范围内修复并重新验证。操作前遵循 `remnote-cli` 的 status 和命令帮助规则。

## 集中验证

从任务回执的 `ids` 和本批次操作记录确定检查集合；用 `context` 集中读取最终卡片、答案和受影响
Wiki 内容，需要正文的每个 ID 显式列入 `remIds`。用 `verify scope` 合并父级、日期对应的日记 ID、
类型、Tag 反查、集群、引用及实际卡片集合预期。原文模式逐段核对源文，普通卡逐项对应已审草稿，
0 张新卡时仍检查实际更新或复用的对象；没有动作的检查记录不适用理由。

下面各项是必须证明的事实，不是要求把每条示例命令逐一重复执行。已有任务或范围回执完整覆盖
事实且符合 remnote-cli 的新鲜度规则时直接引用；后续补引用、元数据更新、修复或外部变更影响的
部分重新读取和检查。不完整、截断、错误 KB 或已失效回执一律不算通过；不降低必需范围换取成功。
回执哈希和数量不能代替正文质量、顺序和原文保真的判断。
最终正文未变且覆盖证据新鲜时复用独立草稿审查；正文改变必须将对应卡交独立助手重审。
引用注入保留可见文本时仍检查实际目标；可见正文改变不能只因哈希更新就放行。

## 闪卡

检查闪卡时先读取 [SuperMemo 20 条知识表述原则](../remnote-flashcards/references/supermemo-20-rules.md)，按最终回读内容逐卡检查理解、独立、原子、可判分、低干扰、可追溯和记忆价值。普通卡同时按[制卡 Skill 的 STAR 通用解释结构](../remnote-flashcards/SKILL.md#star-通用解释结构)检查内容与排版：S/T 区分背景问题与目标，A 直接作答并说明原理和适用的取舍，R 若保留须区分实际结果与预期或推演；合理省略 R 不判失败，不要求固定答案 Rem 数量，不接受空标题、套话或无依据的结果。原文导入／摘录保持 `remnote-flashcards` 的模式约定，不强制费曼改写或 STAR 排版。每张卡给出 `PASS` 或 `FAIL + 违反规则编号 + 原因`；失败项修复后重新回读评审，内容质量通过与下面的结构验证同时满足才算完成。

1. **日期与实体位置。** 用 `sdk date get-todays-doc` 获取今日笔记 ID，再 `read <ID> --view full` 确认是当天的 `dailyDocument`。新单卡的 `parentRemId` 必须等于该 ID；多卡的父级必须是同一个普通 `text` 主题，主题父级等于该 ID。原卡更新保留原日期；错放修复使用本次确定的目标日期。跨午夜新写入重新解析日期；已写卡按回执记录的实际创建日期与每日笔记 ID 核验，不因校验日期改变而重建。引用、Portal、标题中的日期均不能替代真实父子关系。
2. **分类身份。** `read <TAG_ID> --view full --ancestor-depth 8`，必要时继续读祖先，证明它是 HOME 子树下最具体的 Concept，且不是 Folder／Document。同名不等于同一标签。
3. **真实标签。** 回读单卡或主题的 `tags`，要求包含预期 Tag ID，且只有一个 HOME 主题分类；多卡子卡不重复打分类 Tag。再用 `search-by-tag --tag-id <TAG_ID> --result-mode tagged --view full` 反查，翻页直到找到该内容 ID 或结果耗尽。正文引用不算打 Tag。
4. **卡片完整性。** 对内容根运行 `review-stats --root-id <CONTENT_ID>`，核对实际卡片 Rem ID 集合与本批次预期完全一致，防止漏卡、答案生成子卡或主题误成卡。逐卡回读正文与直接子 Rem；折叠时按具体 ID 继续读取。移动前后的 cardId、复习记录和内容必须保持不变。多卡主题的 Card Cluster 与类型按 `remnote-flashcards` 的集群规则检查。
5. **范围复核。** 当天创建时用 `review-stats --today` 确认目标卡片均在今天；跨午夜检查回执中的实际创建日记；历史卡更新则检查原日记。修复 HOME 错放后对原父级复查，确认卡片已移出，分类 Tag 仍留在 HOME。普通写卡只检查本批次，不扫描无关历史内容。

## Wiki

1. 回读本批次页面、aliases、原生引用、相关 `index.md` 条目与 `log.md` 记录。页面在正确 WIKI 分区、保持普通 `text`，无 HOME 或 `wiki-*` 分类 Tag；索引中每个受影响页面恰好出现一次，目标引用可解析。根和目录为 Folder，元文档为 Document，非叶节点按 `remnote-wiki` 折叠。
2. 存在 Wiki 修改时，在最后一次相关修改后调用现有 `verify scope`，显式覆盖 WIKI 根及全部后代，以 `powerup: u`、`zeroCards` 和残留卡片结构检查证明。Wiki 批次的 `verificationScope: affected_objects` 不满足此门禁；只检查根、抽样或只看到禁用开关也不算。没有后续相关变更时完整扫描一次即可，后续 Wiki 修改使证明失效并重查；重连/跨会话按 remnote-cli 新鲜度规则处理。
3. 检查本批次页面及正文是否残留实际卡片结构；用户报告 WIKI 混卡时扩大到整个 WIKI。折叠的 `read` 不代表完整子树，使用 `sdk rem object-get-descendants` 枚举精确后代，再读取可疑节点。普通 Rem 的 SDK `getPracticeDirection` 也可能返回默认 `forward`，不能单凭它判定生成了卡片；结合 `backText`、卡片类型／语法、CLI `cardDirection` 和实际 `cards` 判断。Disable Cards 下实际卡片结构仍属于失败项。
4. 知识页面误开卡片时保留页面、正文、aliases 和引用，给实际卡片 Rem 设置 `object-set-practice-direction --args-json '["none"]'` 并回读；真正错放的学习卡按闪卡流程原位移至每日笔记并挂 HOME Tag。不能将知识页面整批搬走或删除重建。完成修复后重新运行整个 WIKI 的零卡片检查。

### 验证范围

完整 Wiki 检查使用实际根 ID 替换下列占位值，传给 `verify scope --expect <FILE|-> --summary`：

```json
{"schemaVersion":1,"inspect":{"schemaVersion":1,"subtreeRootIds":["<WIKI_ID>"]},"expectations":[{"kind":"zeroCards","remId":"<WIKI_ID>","noCardStructure":true},{"kind":"powerup","remId":"<WIKI_ID>","code":"u","value":true}]}
```

必须完整返回且所有预期通过；实际集合截断、超预算、失败或残留卡片结构均不算零卡片证明。

用户不从 WIKI 进入复习。WIKI 零卡片门禁以 CLI 对完整实体子树的实际卡片与残留卡片结构证明为准，
使用最后一次相关修改后的完整 `verify scope` 回执，不以局部任务回执代替。
不要求界面计数为零，也不默认打开目录或复习队列。没有新建或修改 Wiki 时只检查受影响页面与引用，
不把日常复用扩大为全库审计；接口预算不足时报告未覆盖范围，不能抽样放行。

RemNote 目录计数可能纳入引用关联的外部闪卡，不等于 WIKI 实体子树卡片数。这类计数不阻塞完成，不为清零而删除、停用、移动正常卡片或移除知识引用。只有用户明确要求排查原生队列或清除界面计数时才调用 `control-remnote`；进入队列只观察，不揭示答案、不评分，按实际卡片 Rem ID 核对父级后退出。

## 完成门禁

仅在所有适用项通过后报告完成，简述日期、内容数量、分类路径与 WIKI 卡片数；保留可追溯 Rem ID。部分失败或读取不完整时列明具体 ID、失败项和未验证范围，不能把命令成功、空搜索或工具超时当成通过。重试写入前先回读，避免重复创建。内容只在 RemNote 中保留，不保存本地内容副本。
