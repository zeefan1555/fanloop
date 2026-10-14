---
name: remnote-flashcards
description: "Create, update, or review RemNote flashcards on today's daily note under the finest HOME Concept tag. Use when the user asks for flashcards, an original import, an original excerpt, or a check of card placement. Ordinary cards are atomic term cards or STAR combos. Do not add Card Cluster or write the Wiki unless the user explicitly asks. Follow the checklist in this skill with remnote-cli; do not call fanloop flow, and do not use this skill to change RemNote Direct source."
---

# RemNote 笔记与闪卡

本文件是 Cursor 插件 `fanloop-remnote` 的制卡入口。不要调用 `fanloop flow`，也不要启动 Fanloop Requirement。下面的清单就是原来的四个 Step、正常 Route 和回流 Route。内容规则、失败条件和 preflight 字段仍以 [学习卡契约](references/learning-card-contract.md) 为准，本清单不另写一套，也不缩短门禁。

材料、Rem ID、标题和卡片正文只来自用户给出的材料和当次 `remnote-cli` 回读。参考里的干跑示例和历史 Rem ID 不是本次知识库的数据，不得抄进计划。

每个步骤开始时发一条用户可见状态：步骤 id、即将做的事、还没通过的门禁。没有用户可见通道就停。不要写 `artifacts/panorama-*`，不要把保存文件当成已展示。进入 `retrospect_run` 时的那条状态不是完成消息。

绑定技能：定位用 [remnote-structure](../remnote-structure/SKILL.md)，命令用 [remnote-cli](../remnote-cli/SKILL.md)，审卡和写后核对用 [remnote-verify](../remnote-verify/SKILL.md)。[remnote-wiki](../remnote-wiki/SKILL.md) 只有用户本批明确要求维护 Wiki 时才读。搜索、读取、别名、标签和写卡都用 `remnote-cli`。桌面只在 CLI 确实没有对应命令时才用 [control-remnote](../control-remnote/SKILL.md)，例如重载中文搜索缓存的按钮。

## 不变量

- 计划形状见 [FlashcardsPlan](../remnote-cli/references/flashcards-plan.md)。`newCards` 最多 7 张，计划最大 100 KiB。不截断正文，不抬高上限。
- 每次 shell 只跑一个 `remnote-cli` 命令。不得并行，包括同一轮里的多个 shell。并行会报 `CLI port occupied`。
- `flashcards apply` 默认预览。不加 `--apply` 的输出只有 `validated` 和 `planHash`，没有卡片正文。`--apply` 之前读完计划文件本身，对照已审草稿；学习预检在这次阅读之后仍全部为 pass，才另一次调用加上 `--apply`。
- 含中日韩文字的 `search` 返回空，不是「没有这张卡」或「没有这个 Tag」。先做已知术语对照搜索。已知术语对照搜索返回空时，先点桌面按钮「重新加载搜索缓存」，再重搜。Also search the English form of the term when it has one. 查重和写入仍回到 CLI。不得据此新建术语卡。
- 今日笔记先只读查找。`sdk date get-todays-doc` 是写入，笔记不存在时会创建。只读阶段不得调用。
- 同一计划重试、跨日或换会话都用原来的 `--idempotency-key`。批次之间的键不冲突。
- 结果是 `partial` 或 `unknown`，或超时、断连：停止写入，留下原计划、键、已知 ID 和私有恢复记录，只读核对。不自动重放，不换键，不删除重建，不把空搜索当成没写过。
- 写前 preflight 的门禁 id 全部要在，且都是 pass：`term_search`、`atomics_first`、`short_titles`、`aliases_after_create`、`rem_ref_title_ids`、`star_labels_exact`、`arrow_scan`、`theme_literal`、`theme_para`、`tag_path`、`tag_approval`、`control_search`、`no_spec_unless_asked`、`card_cluster`、`named_methods`、`redo_reuse`、`combo_placement`、`alias_method`、`factual_review`。`arrow_hits` 为 0。契约里的失败条件一条成立就不得通过，也不得 `--apply`。example numbers state their condition, such as when idle; do not use a platform term as an everyday word, such as 小程序 on a WeChat card; a binary question's answer line states the choice. 这是 `factual_review`，不是 `kid_clarity` 的第六个键。theme_para maps 00-NOTE to note, 01-PROJECTS to project, 02-AREAS to area, 03-RESOURCES to resource, and 04-ARCHIVE to archive.
- 用户没要求时不为 `spec` 或策略规格说明书建卡。没要求时不写 Wiki，不用「未要求 Wiki」充当通过。
- combo rem-refs use flashcards apply TextSegments {"remId":"..."} or object-set-text RichText {"i":"q","_id":"..."}; do not use create --content-file with [[id:]].

## 步骤清单

同一时刻只记一种步骤结果。回流使后面的结论失效。同一问题连续三轮失败，或结果不明，就 blocked，并说明已完成和剩余范围。准备阶段不得把已经写好的卡再写一遍。

### 1. `resolve_home_tag`

复用对话里已有的理解和材料，不重新访谈。材料是 spec 或策略规格说明书，且用户没有明确要求为它制卡：blocked，不记 `home_tag_ready`，不进入术语检索。

按 [remnote-structure](../remnote-structure/SKILL.md) 定位唯一 HOME 和最细 Concept Tag。持续课程用教程名，不用作者名。叶子必须是 Concept。中文名称的 `search` 为空时，先做已知术语对照搜索。已知术语对照搜索返回空时，先点桌面按钮「重新加载搜索缓存」，再重搜。Also search the English form of the term when it has one. 不要把 Tag 判成缺失。单篇或单章记下逐字 `source_h1`。来源类型按 `theme_para` 对齐 Tag 所在 PARA 区。每个术语先 `search --cards-only`，够用则记下标题卡 Rem ID；Wiki 页和空 `cards` 不是术语闪卡。用户说该来源旧卡要重做时，那些卡不进入复用。复习证据不可用就标 unavailable，不推断掌握程度。

最细 Concept 沿 HOME 树确认不存在时，顺序固定：

1. 向用户提出完整祖先路径和要新建的 Concept 名称，等用户明确同意。空的中文 `search` 不是同意，也不是缺失证明。
2. 用户同意之后，才做这次明确批准的写入：`remnote-cli create --title "<Concept 名称>" --parent-id <父级 ID>`，然后 `remnote-cli sdk rem object-set-type --target-id <新 Rem ID> --args-json '[1]'`。每次 shell 一个命令。
3. `read` 回读 `remType=concept` 和父级。把这个真实 Rem ID 记入 `tag_id` 和 `created_tag_ids`。
4. 后面的预览和 `--apply` 用这个 `tagRemId`。不要猜 ID，不要在用户同意前创建，也不要靠 `flashcards apply` 顺手建 Tag。

没得到同意就停在本步，不预览。

离开本步还要 `tag_approved=passed`。叶子已存在时 approval 为 not_needed，`created_tag_ids` 为空。

通过 `home_tag_ready` 必须同时有：`home_id`、`tag_id`、`ancestor_ids`、`tag_ancestor_path`、`created_tag_ids`、`source_refs`、`material_digest`、`mode`、`goal_keys`、`existing_card_ids`、`term_keys`、`term_scope`、`source_h1`（不是单篇时为 `n/a`）。ID 来自当前连接。

失败：

- 缺来源、权限、范围歧义或工具故障：blocked。
- 分类要改：`home_tag_needs_fix`，留在本步。维护 CLI 另走 playbook-flow 的 `code-change` playbook，不在制卡里升级工具。

### 2. `prepare_cards_and_wiki`

本步只起草，不写卡，不调用 `wiki apply`。默认不读 Wiki 技能。用户要求了 Wiki 但写入没完成：blocked，不要用术语回执掩盖。Wiki 写完也不等于 `terms_ready`。

顺序：列出术语；每个术语 `remnote-cli search "<query>" --cards-only`；够用则复用标题卡 Rem ID；没有才起草短标题原子卡，长问句只作为别名计划；原子卡都有归宿之后才起草组合卡。组合卡引用只用 `term_key` 占位或已有标题卡 Rem ID。

每完成一批草稿，交给独立只读审卡。审卡按 [remnote-verify](../remnote-verify/SKILL.md)。审卡者只读材料和草稿，不改卡、不改 Wiki。只修失败卡，再复审改动部分。作者自证不能通过。

能派子 Agent 时，派一个不继承起草上下文的只读审卡。这个宿主派不出子 Agent 时，请父级或宿主跑这次只读审卡，并把 `reviewer_receipt` 交回来。没有这样的父级时，请用户按同一份检查审这份草稿，回执记 `reviewer=user`。起草者不得自己填回执并称作独立审卡。子 Agent、父级和用户都没有完成审卡时，`draft_quality_passed` 失败，不得写卡。只复审改变的卡，对照重新计算的 draft_digest。回执记录 `plugin_commit`，即本插件的 git commit。

离开本步必须五项同时成立，缺一不可。第五项是 `control_search_passed=passed`：已知术语对照搜索已经做过；对照为空时已经点过「重新加载搜索缓存」并重搜。原文导入或摘录且 `terms_ready` 为 `skip_original` 时，可以带 `skip_original` 通过。

- `preparation_recorded`：`material_digest`、`draft_digest`、`card_keys`、`term_pipeline`、`source_h1`、`theme_title`、`preflight`、`reviewer_receipt`（覆盖 `card_keys`、失败项、来源、摘要、`kid_clarity`、`star_headings`）、`term_keys`、`skipped_count`。不写 `wiki_receipt`。
- `draft_quality_passed`：审卡覆盖的 `card_keys` 与 `preparation_receipt.card_keys` 逐项相同，失败项为空。每张普通卡有 `kid_clarity` 五问 `one_goal`、`one_pass`、`concrete`、`plain_words`、`no_arrow`，值只能是 yes 或 no，且全部 yes。每张组合卡的 `star_headings` 逐字是 `S — Situation：背景`、`T — Task：任务／目标`、`A — Action：行动`、`R — Result：结果`。combo plain_words rejects unexplained trading slang: 梭哈, 全仓, 一把梭, 优秀个股; rewrite with plain kid words such as 一次花光 or 分开买.
- `terms_ready=passed`：`term_pipeline` 覆盖本批每一个术语。每条有 `query`、`cards_only: true`，`outcome` 为 `reuse` 或 `create`。`reuse` 带标题卡 Rem ID。`create` 带同一记忆目标的 miss 和 `term_key`。组合卡 `rem_refs` 只含这些标题卡 ID 或本批 `term_key`。空管道、别名 Rem、Wiki 页、`pageKey`、字面括号、编造 ID 都不得通过。全部重复允许零新卡，仍要有搜索证据和标题卡 ID。原文导入或原文摘录只有 `mode` 为 `original_import` 或 `original_excerpt`、`term_pipeline.action` 为 `skip_original`、术语创建为 0、Wiki 调用为 0 时可以通过。
- `learning_preflight_passed=passed`：`preflight.draft_digest` 等于当前 `draft_digest`，上面列出的每个门禁都是 pass，`arrow_hits` 为 0。任一 fail、缺对象、摘要不一致或结果不明都不得通过。
- `control_search_passed=passed`：已知术语对照搜索已完成。已知术语对照搜索返回空时，先点桌面按钮「重新加载搜索缓存」，再重搜。Also search the English form of the term when it has one.

`source_h1` 与 `theme_title` 对不上，不要把 `theme_literal` 写成 pass。

失败：

- 分类错：`home_tag_needs_fix`，回到 `resolve_home_tag`。
- 草稿正文、STAR、箭头或儿童口吻：`flashcards_need_fix`，留在本步。
- 标题卡引用、别名 Rem 或漏掉的术语搜索：`term_refs_need_fix`，留在本步。不在写入步发明 ID。

### 3. `persist_and_verify`

先核对上一步五项仍有效，且 `material_digest` 与 `draft_digest` 对得上当前材料和最新草稿。缺回执、摘要不一致、`term_pipeline` 没覆盖本批术语、审卡覆盖不等于 `card_keys`、失败项非空、preflight 缺失或不一致、任一门禁不是 pass：不得写卡，回到准备步骤。`source_h1` 与 `theme_title` 按契约重算，不一致就阻塞。

默认不注入 Wiki 页面 ID，不等待 Wiki，不调用 `wiki apply`。

计划没有字段可以指定已经存在的主题。`title` 会新建一个主题，只挂本计划的 `newCards`。组合卡计划如果再写 `title`，会再造一个主题。`reusedCardRemIds` 和 `answerPatches` 也不改父级。

术语卡计划写 `title`，用已经对上的 `theme_title`。已经有 Rem ID 的标题卡可以写进组合卡答案，段是 `{"remId":"<标题卡 Rem ID>"}`。本批新建的标题卡等这次术语卡 apply 回读 ID 之后，组合卡再单独一次 `flashcards apply`，计划不写 `title`。不要把 `term_key` 或 `newCards[].key` 写成 `{"remId"}`，也不要用 `create --content-file` 的 `[[id:]]`。

组合卡必须放在主题 Rem 里面，排在术语卡之后，是最后一个子 Rem。组合卡自己不挂 Tag。主题上的 HOME Tag 索引这些卡。组合卡那次 `flashcards apply` 结束后，父级还是每日笔记，并且自己带上了 Tag。不要把组合卡留在每日笔记上。

接着每次 shell 一个命令。先干跑：

`remnote-cli move-note <COMBO_ID> --new-parent-rem-id <THEME_ID> --position last --expected-old-parent-rem-id <DAILY_ID>`

对照输出后，同一条加上 `--apply`。然后去掉组合卡自己的 Tag。`update-tags --remove-tag-ids` 会失败，它要求 allowDestructive。用：

`remnote-cli sdk rem object-remove-tag --target-id <COMBO_ID> --args-json '["<TAG_ID>", false]' --allow-destructive`

回读：主题子 Rem 的顺序是术语卡在前、这张组合卡为最后一个；组合卡没有 tags；`review-stats --root-id <THEME_ID>` 列出本批全部卡片；`search-by-tag` 只返回主题。

别名在标题卡已经存在之后再加。每张标题卡一条命令：`remnote-cli sdk rem object-get-or-create-alias-with-text --target-id <ID> --args-json '[["<alias>"]]'`。返回的别名 Rem 只供搜索。`remnote-cli update --add-aliases` 是否可用取决于 SDK 版本。SDK 0.0.46 上它失败，错误是 `Native transactions are unavailable: SDK 0.0.46 cannot acknowledge abort`。该版本用上面的 SDK 方法，不要改走桌面，也不要把长问句写成卡片标题。

先预览。预览输出只有 `validated` 和 `planHash`，没有卡片正文。读完计划文件，与已审草稿对照，预检仍全部 pass，才 `--apply`。术语卡计划和组合卡计划各预览一次，键不冲突。已经复用的标题卡不新建。不得引用别名 Rem。`--apply` 前再扫普通学习卡的正面和答案行，`arrow_hits` 必须为 0。今日笔记若还没有只读确认，这时才可以调用 `sdk date get-todays-doc`，并标明它是写入。`tagRemId` 用第 1 步回读到的 Concept ID。

主题按真实创建日期放进每日笔记。只有一张卡、没有主题时，这张卡直接属于今日笔记并挂最细 Concept Tag。有主题时，术语卡是主题的子 Rem，组合卡是最后一个子 Rem，只给主题挂该 Tag。主题标题用已经对上的 `theme_title`。更新保留原位置、Rem ID、`cardId` 和复习记录。

`cardCluster` 只按契约。写后对每个多卡主题运行 `object-has-powerup ["cc"]`。意外的 `true` 是 remnote-direct 回归：`verification_retry`，不要记成成功，也不要调用 `object-add-powerup` 去补或去抹。

写后用回读重新判定 `learning_preflight_passed`。只复述准备阶段的 pass 不得报 passed。没有 Wiki 修改时，不跑 WIKI 零卡片扫描，也不把缺少 Wiki 引用判失败。

离开本步必须六项同时成立。另外两项是 `combo_placed=passed` 与 `aliases_written=passed`。没有组合卡时记 `no_combo`。没有新别名时记 `no_new_aliases`。

- `flashcards_written`：`created_rem_ids`、`updated_rem_ids`、`reused_rem_ids`、`card_ids`、`topic_ids`、`daily_rem_ids`、`tag_id`、`source_refs`、`skipped_count`、`reference_pairs`、`batch_ids`、`review_receipt`、`verification_receipts`、`cluster_powerup`。`reference_pairs` 的目标是标题卡 Rem ID。`cluster_powerup` 为 `true`、`false`，或没有多卡主题时的 `not_applicable`。
- `terms_referenced=passed`：组合卡答案里的引用 Rem ID 等于本批标题卡 Rem ID，回读标题是短名，点击目标带背面。没有组合卡时核对没有引用到别名 Rem 或 Wiki 页，并记下 `no_combo_refs`。
- `result_verified=passed`：内容、结构、预检和标题卡引用都通过。零新卡仍核对复用对象和分类。
- `learning_preflight_passed=passed`：来自写后回读，不是准备阶段的复述。
- `combo_placed=passed`：组合卡是主题的最后一个子 Rem，且没有自己的 Tag。没有组合卡时记 `no_combo`。
- `aliases_written=passed`：新别名用 `sdk rem object-get-or-create-alias-with-text --target-id <ID> --args-json '[["<alias>"]]'`。没有新别名时记 `no_new_aliases`。

失败：

- 分类错：`home_tag_needs_fix`，回到 `resolve_home_tag`。
- 草稿正文要改：`flashcards_need_fix`，回到 `prepare_cards_and_wiki`。
- 标题卡引用错：`term_refs_need_fix`，回到 `prepare_cards_and_wiki`。
- 写后 `cc`、父级或回读不一致，或临时读取失败：`verification_retry`，留在本步。先恢复真实 ID，只补已经查明缺失的动作。

### 4. `retrospect_run`

确认 `result_verified=passed` 和最终回执仍有效。不进行新的知识写入，不补 Wiki 日志。

只读已有运行记录。统计内容准备、独立审卡、工具读写、完整扫描、返工和等待。保留总 wall time，重叠时间不相加，分不开就写 `unknown`。

`retrospective_recorded` 包含 `timing`、`full_scan_count`、`rework_count`、`this_run_violations`、`findings`（0 或 1）、`notion_sync`。`this_run_violations` 是本轮曾经失败的门禁 id，没有则为空列表。发现最多一条，固定四段：证据、重复机制、建议改动、验证方法。允许零发现。

用户未要求集群而 `cluster_powerup` 为 true：这条就是本轮唯一发现。证据是主题 Rem ID 和 `object-has-powerup ["cc"]` 的结果。机制是 `flashcards apply` 在省略或 `false` 时不得添加 `cc`。建议改动指向 remnote-direct。验证是同一查询回到 false。这条发现表示内容证据已失效，回到 `persist_and_verify`，不能在本步把 true 收成成功。

有 finding 时按 [notion-experience.md](references/notion-experience.md) 用 `ntn` 写一行。零发现不调用 `ntn`。`ntn` 失败只记 `notion_sync`，仍然可以记 `retrospective_recorded`。复盘不修改 Workflow、验收条件或 CLI。

FANLOOP_REMN_EXPERIENCE_DATA_SOURCE_ID overrides the data source id; FANLOOP_REMN_EXPERIENCE_DATABASE_ID overrides the database id. Defaults live in references/notion-experience.md.

Auth is ntn login; headless hosts set NOTION_KEYRING=0. Do not require an integration API key.

Map 经验 to the one-line lesson, 日期 to the ISO run date, 主题 to the RemNote theme title, 类别 to 文案清晰度|工具链路|流程门禁|其他, 详情 to evidence + mechanism + suggested change + verification, and 编码状态 to 待编码 unless the finding already points at a merged fix (已编码) or is a non-code lesson (不适用).

When findings is non-empty, POST one page per finding with ntn api v1/pages to data source 「remnote workflow 经验表」; zero findings skip the write; a failed ntn call records notion_sync and still allows retrospective_recorded.

失败时回到最早一层：`home_tag_needs_fix` 回 `resolve_home_tag`；`flashcards_need_fix` 或 `term_refs_need_fix` 回 `prepare_cards_and_wiki`；`verification_retry` 回 `persist_and_verify`。实际内容证据已失效时不能用本步结束。

通过之后另发一条完成消息，不要停在进入本步时的状态。完成消息写真实日期、创建、更新、复用和跳过的数量、分类、标题卡与组合卡 ID、验证范围，以及没实测的项。默认不列 Wiki 页面。不用固定分钟数承诺提速。

## 先选模式

- **只读评审**：读取目标卡片并给出问题与修改建议，不要求写权限，不执行写入、发布或界面操作。
- **普通闪卡**：按增量学习、费曼法和 SuperMemo 规则创建或修改卡片。先区分原子术语卡和组合卡。轻量学习默认不写 Wiki；只有用户明确要求 Wiki 时才查询或创建 Wiki 页面。
- **原文导入／原文摘录**：保留用户指定原文，只读取对应模式规则；跳过费曼改写、STAR、Wiki 补充和 ELI5。
- **普通笔记／文章**：仅在用户明确要求写入 RemNote 普通内容时处理，并读取文章格式；不执行闪卡评审流程。

先确定模式，再读取该模式引用的参考文件。共享的 ID、分类、去重和写后回读规则仍适用。

## 极简分工

- HOME：顶层 `HOME` Folder 下保存 `00-NOTE`、`01-PROJECTS`、`02-AREAS`、`03-RESOURCES` 和 `04-ARCHIVE` 五个一级 Folder；`00-NOTE` 保存尚未归入正式知识结构的随手笔记和凭证记录。更深节点的 Folder／Concept Tag 查找、动态判型和创建由同 Workflow 的 [remnote-structure](../remnote-structure/SKILL.md) 统一管理。
- WIKI：顶层 `WIKI` Folder，与 HOME 同级，按原 `llm-wiki` 组织直接保存 `SCHEMA.md`、`index.md`、`log.md`、`raw`、`entities`、`concepts`、`comparisons`、`queries` 和 `_archive`。目录节点使用 Folder，三份 `.md` 使用 Document，具体页面保持普通 `text`，整个 WIKI 子树中的非叶子 Rem 默认折叠；页面身份、别名、来源和引用由同 Workflow 的 [remnote-wiki](../remnote-wiki/SKILL.md) 统一管理，不使用概念 Tag。
- 每日笔记：保存闪卡、普通笔记和文章的真实实体。新闪卡写入今日笔记并挂 HOME 分类 Tag；Wiki 页面直接属于 WIKI 的对应分区。

使用普通文档和 Rem 维护这套结构。HOME 中的 Folder 组织分类树，Concept Tag 通过标签关系归类每日笔记中的内容；WIKI 页面直接保存定义、例子、来源和互联引用，不生成闪卡。不创建 `wiki-*` Tag，不使用查询 Portal，也不把 Wiki 页面复制到 HOME。

一份内容只有一个实体。单张卡直接放在今日笔记并挂 HOME 中最具体的 Concept Tag；同一主题生成多张卡时，先在今日笔记建立一个主题 Rem 并给主题打该 Tag，再把卡片作为它的直接子 Rem。卡片可以引用 WIKI 页面，但不得放入 WIKI。普通文章仍写入每日笔记，并使用主题标签和“文章”类型标签。

## 分类

先搜索与当前内容相关的分类和祖先路径，再选择最具体的类别；只有结构审计或迁移才读取 HOME 全部 Tag：

- 01-PROJECTS：正在推进、会结束的具体结果。
- 02-AREAS：长期负责或持续维持的领域。
- 03-RESOURCES：学习或参考主题。
- 04-ARCHIVE：已结束且需要保留的内容。
- 00-NOTE：尚未归入正式知识结构的随手笔记和凭证记录。

按 `remnote-structure` 查找或创建分类结构。学习批次打领域分组下最细的那一个 Concept Tag，使用精确 Rem ID 建立标签关系。例如路径 `HOME / 03-RESOURCES / 金融知识 / 估值`：`03-RESOURCES` 与 `金融知识` 是分组，打在批次上的是 `估值`。已有更细 Concept 时，不要停在一级 Folder 或宽泛领域上；没有该 Concept 时先创建并回读类型和父级，再打 Tag。

持续课程或教程系列的这个最细 Tag 用教程或课程名，不用作者名。例：`HOME / 03-RESOURCES / 金融知识 / 人人都是量化交易员`。作者可以记在别的元数据里。`刑无刀` 不是这门课的 Tag。单篇概念批次仍用该概念的 Tag，例如 `估值`。

单张卡给卡片自身打这个 Tag。多张卡放在主题 Rem 下时，只给主题打 Tag。组合卡自己不挂 Tag，并且是主题的最后一个子 Rem。普通文章再复用 `HOME / 03-RESOURCES / 文章` Concept 作为第二个类型标签。名称只用于查找和展示。

## 顶层内容命名

新建主题 Rem、普通笔记和文章使用“日期 + 类型或来源 + 中文主题”的可搜索标题；RemNote 标题不添加 `.md` 后缀。例如：`2026-06-12 X 长期投资框架`、`2026-06-12 meeting 项目复盘`。

- 日期提供时间线索；类型可用 `meeting`、`project`、`book`、`resource`、`idea`，外部内容可直接使用来源名。
- 单篇文章或单章的闪卡主题按[学习卡契约](references/learning-card-contract.md)：`YYYY-MM-DD <来源类型> <文章原 H1>`。来源类型对齐 Tag 所在 PARA 区。theme_para maps 00-NOTE to note, 01-PROJECTS to project, 02-AREAS to area, 03-RESOURCES to resource, and 04-ARCHIVE to archive. 教程 Tag 在 `03-RESOURCES` 下时，H1「前言」写成 `2026-10-06 resource 前言`。历史事故字符串 `2026-10-06 book 前言` 只保留为记录。不要改写成「术语与方法」或「量化地图」。
- 外部内容保持中文可读主题；项目材料优先保留项目语义，不为模板损害辨识度。不是单篇文章时，才用概念名或用户指定的主题，例如 `2026-10-06 resource 市盈率 PE`。
- 只对新内容和用户明确指定的高价值旧内容应用；不批量重命名历史内容。
- 该规则只作用于顶层内容单元。单张卡的正面仍是可直接复习的问题，原文导入和原文摘录仍保留原文标题，WIKI 页面仍使用干净规范名。

## 共用概念

普通主题 Tag 可以容纳多篇笔记，不按每篇笔记新建分类。轻量学习路径默认跳过 WIKI：普通金融或概念批次不查询、不创建、不引用 Wiki 页面，也不为了术语建页。术语的最小定义写在卡片自己的背面。

默认路径不查、不建、不引用 Wiki。术语先 `search --cards-only`：够用则复用标题卡 Rem ID，不够才新建短标题原子卡，长问句只做这张卡的别名。组合卡只用 `{"remId":"<标题卡 Rem ID>"}` 引用标题卡，或用 `object-set-text` 的 `{"i":"q","_id":"<标题卡 Rem ID>"}`。界面显示短名，例如 `[[PE]]`。引用别名 Rem 会打开空页，不得写入。

combo rem-refs use flashcards apply TextSegments {"remId":"..."} or object-set-text RichText {"i":"q","_id":"..."}; do not use create --content-file with [[id:]].

只有用户本批明确要求维护 Wiki 时，才读取 [remnote-wiki](../remnote-wiki/SKILL.md)。该工作可选，不满足 `terms_ready`，也不代替标题卡引用。原文模式不建术语卡，也不补 Wiki。

## 增量学习

每次制卡前先把材料拆成独立记忆目标。创建任何术语卡之前，按短名、别名和关键问法运行 `remnote-cli search "<query>" --cards-only --include-review-stats --view full`，找同一记忆目标的术语闪卡，并按最具体的主题 Tag 补充检索。命中且这张卡够用时复用它：组合卡引用该标题卡的 Rem ID，不新建。只有没有术语闪卡时才创建。

Wiki 概念页不是术语闪卡。搜索命中 Wiki 页面，或该 Rem 的 `cards` 为空，都不算已经有这张卡。默认不查询 Wiki；即使用户要求了 Wiki，页面是否存在也不代表学习者已经掌握，也不代替上面的 `--cards-only` 搜索。

用户明确说某一来源的旧卡质量差，或要求重做时：删除或替换那些卡，不要复用该来源上的低质量闪卡。用户没有要求重做时，仍按下面的增量复用。

对每个记忆目标只做一种处理：

- 已有卡片表达同一记忆目标，且新材料没有增加事实、条件、边界或纠正：跳过，不重复创建。
- 已有卡片表达同一记忆目标，但新材料提供了实质补充或纠正：编辑原卡对应的答案 Rem，保留卡片 Rem ID、位置和复习记录；修改后重新回读并评审。
- 新材料包含独立的新记忆目标：创建新卡；不能为了覆盖材料而改写成与旧卡近似的问题。

“是否学会”不使用二元猜测。优先读取 RemNote 返回的卡片复习事实，包括复习历史、最近复习时间、下次到期时间、连续答错次数和原生掌握等级，并区分未复习、正在巩固、有失败信号和稳定掌握。稳定掌握的旧卡遇到重复材料时直接跳过；正在巩固或有失败信号的旧卡也不复制，只在内容本身存在歧义、干扰或知识缺口时修复原卡，否则继续交给 RemNote 的调度器。

搜索结果已带 `cards` 时直接使用其中的原生复习事实；需要检查整个主题、标签或今日笔记时，分别使用 `remnote-cli review-stats --root-id <rem-id>`、`--tag-id <tag-rem-id>` 或 `--today`。命令返回一个 Rem 的空 `cards` 数组时，只能说明该 Rem 没有生成卡片；`repetitionHistory` 为空时表示尚无复习记录。

如果当前 CLI 没有暴露上述复习数据，必须把状态记为“复习证据不可用”，本次只能执行内容查重和原卡更新；不得根据创建日期、Wiki 是否存在、搜索命中或主观印象声称用户已经学会，也不得伪造复习记录。用户在对话中的纠正用于更新内容或规则，不替代 RemNote 的真实复习历史。

## 卡片写作

创建、修改或修复普通闪卡时，先读[学习卡契约](references/learning-card-contract.md)。卡型、短标题与别名、STAR 原文、箭头、主题、Concept Tag、spec、Card Cluster、写入前 `preflight` 和审卡五问都只以契约为准。再读 [费曼学习法制卡](references/feynman-card-method.md) 和 [SuperMemo 20 条知识表述原则](references/supermemo-20-rules.md)。先用费曼流程证明理解，再用 SuperMemo 原则拆成可复习的原子卡。原文导入和原文摘录不套这两份改写规则，也不套契约里的费曼和 STAR。

费曼流程必须在排版前完成。对每个候选记忆目标先做到：

1. 不照抄原文，用零基础读者能懂的话讲清它是什么、为什么成立、怎样在具体例子中发生，以及最容易失效的边界。
2. 找出未解释术语、缩写、循环定义和条件到结论之间的跳步；回到来源补齐后重新讲一遍。用户没有要求 Wiki 时，不要靠 Wiki 页面补这个缺口。
3. 能换一个例子得到同样结论，并能用一句话指出本卡唯一要提取的答案，再进入制卡。

做不到任一项时，继续学习或放弃该卡，不能用更长答案、更多链接或固定标题掩盖理解缺口。

用户明确要求 ELI5、图解或可分享网页时，读取[复杂概念的 ELI5 网页](references/complex-concept-eli5.md)，调用宿主已有的 `eli5` 技能生成并按用户要求发布，再把链接写回 RemNote。本插件不自带 `eli5`。宿主没有该技能时停下来说明缺口，不编造网页或链接。未要求独立网页时，复杂概念直接在卡片和对话中解释。

按契约生成 0..N 张卡片。一篇文章或一个主题优先 3–7 张；目标更少时不凑数，独立目标明显多于 7 个时按主题拆开。对照稿见 [市盈率干跑示例](references/example-pe-valuation.md) 和 [前言文章干跑示例](references/example-book-preface.md)。前者是概念批次，主题用 `resource`。后者是单章，主题用文章原 H1。决策记录见 [ADR-0129](references/adr-0129-lightweight-remnote-learning-cards.md)。

## 两种写入格式

写入前先判断是闪卡还是普通笔记。闪卡写入今日笔记，并挂最细的 HOME Concept Tag。普通笔记和文章可写入每日笔记。只有用户明确要求 Wiki 时，内容才引用 Wiki 页面。

创建、修改或修复普通文章时，必须读取[文章格式](references/article-format.md)。

### 闪卡

主题放在每日笔记。只有一张卡、没有主题时，卡片直接属于今日笔记，并挂最细 Concept Tag。同一批次有多张卡时，在今日笔记下建一个普通 `text` 主题 Rem。术语卡是它的子 Rem。组合卡必须是主题的最后一个子 Rem，自己不挂 Tag；主题的 HOME Tag 索引这些卡。不要把组合卡留在每日笔记上。主题保持 `text`，不转成 Document 或 Folder。

Card Cluster 只按学习卡契约：用户没要求时计划省略 `cardCluster` 或写 `false`，两者都不添加 `cc`；用户明确要求，且有主题标题、新卡至少 2 张时才写 `cardCluster: true`。不要用 `object-add-powerup` 补 `cc`。写后按 [卡片集群](references/card-clusters.md) 用 `object-has-powerup ["cc"]` 回读，结果记入 `cluster_powerup`。意外的 `true` 是 remnote-direct 回归。

阅读用的 Markdown 缩进稿（`问题 ===` 加子 bullet）只帮助看背面形状。它不是 `flashcards apply` 的计划。计划是 `question` 字符串加 `answer` 的 `Segment[][]`，见 [FlashcardsPlan](../remnote-cli/references/flashcards-plan.md) 和两份干跑示例里的计划节。别名和标题卡引用见契约。段间空 Rem 含一个 `U+200B`；Markdown 导入会忽略真正的空 bullet。

其他已有语法见[格式参考](references/remnote-import-format.md)。

#### 原文导入闪卡

用户要求“直接使用原本笔记内容”时进入原文导入模式，不套用 STAR 改写：

- README 只负责目录和 Tag 映射；README 中每个单篇文章链接对应一张多行闪卡。
- 正面使用原文章标题，背面按原顺序保留完整正文，不拆卡、不摘要、不改写观点，也不补 Wiki 引用。
- 仅删除 YAML frontmatter、Decks 锚点等 Obsidian 元数据，并把 HTML 标题、列表、链接和代码块转换为等价的 RemNote 结构；可见内容不变。
- 卡片只使用 README 中最具体的二级分类 Tag，不使用“文章”类型 Tag。
- 写入前按原文章标题和来源路径查重；写入后逐段对比源文，确认无缺段、乱序或意外生成的子卡片。

```markdown
- 原文章标题 ===
    - 原文第一段
    - 原文第二段
    - 原文小标题
    - 原文后续内容
```

#### 原文摘录闪卡

用户要求“原文标题是问题，原文内容是答案”或只希望保留材料中的指定段落时，进入原文摘录模式。该模式不套用 STAR 改写，但与“原文导入闪卡”不同：它只保存用户指定的摘录，不要求导入整篇文章。

- 正面使用原文标题，作为一个完整、可直接复习的问题；不得替换成摘要标题或自行改写的问题。
- 背面使用与该标题对应的原文内容；保留原文措辞、顺序和必要段落结构，不总结、不扩写、不补充 Wiki 定义。
- 可以删除不属于摘录范围的前后文，以及 YAML frontmatter、Decks 锚点等元数据；不得删除摘录内部的可见内容。
- HTML 标题、列表、链接和代码块转换为等价的 RemNote 结构，但不改变可见含义。
- 默认沿用普通闪卡的最具体主题 Tag；如果材料明确给出来源 Tag，按来源 Tag 映射，不额外添加“文章”类型 Tag。
- 写入前按“原文标题 + 来源路径或来源标识”查重；写入后逐段对比摘录，确认问题与答案对应、内容完整且没有意外生成子卡片。

```markdown
- 原文标题 ===
    - 原文摘录第一段
    - 原文摘录第二段
    - 原文摘录中的列表项
```

### 普通笔记

普通笔记按[文章格式](references/article-format.md)创建和核验。

## 工作流

通过条件和回流以文首步骤清单为准。不要调用 `fanloop flow`。本 Skill 在 prepare_cards_and_wiki 只起草，在 persist_and_verify 才写卡；使用 resolve_home_tag 已回读的分类 ID、来源及查重结果，不重新访谈或重复收集材料。分类不正确时回第一步修复。

1. 选普通或原文模式；只读评审和普通笔记/文章不属于本 Workflow 入口。写前运行 `remnote-cli status`，核对连接、兼容版本和写入权限。
2. 按增量学习查重。创建术语卡前先 `search --cards-only`。同一记忆目标的术语闪卡够用就复用标题卡 Rem ID；Wiki 页不算。中日韩查询返回空不是 miss，先沿 Tag 树、`search-by-tag` 或 `context` 交叉核对；仍不能确定时，只让桌面点重载中文搜索缓存的按钮，然后回到 CLI 重查，不要新建。用户要求重做时删除或替换该来源的旧卡，不复用。否则重复跳过、实质补充原位更新、新记忆目标才创建。保留实际 cardId、位置和复习记录。允许 0 张新卡，仍记录复用 ID 并前进。
3. 普通卡先搜术语，再起草原子卡，再起草引用标题卡的组合卡。完成费曼与 SuperMemo 自检后，每批草稿立即交独立只读助手按 remnote-verify 审卡。派不出子 Agent 时，请父级或宿主审，或请用户审并记 `reviewer=user`。起草者自填回执不算通过。默认不派 Wiki 助手。审卡回执包含来源及草稿摘要、覆盖卡号、每张普通卡的 kid_clarity 五问、每张组合卡的逐字 star_headings、失败项；只修失败卡再复审。
4. 离开准备步骤必须同时有 draft_quality_passed、terms_ready 和 learning_preflight_passed。terms_ready 来自 term_pipeline：每个术语都有 `--cards-only` 搜索，复用带标题卡 Rem ID，新建带 miss。学习预检的全部门禁为 pass，且与当前 draft_digest 一致。不得用 Wiki 不适用回执前进。今日笔记先只读查找；恢复使用原回执日期。只有已经授权写入、而且只读查找没有确认今日笔记时，才调用 `sdk date get-todays-doc`，并标明这是写入，可能创建每日笔记。按[remnote-cli](../remnote-cli/SKILL.md) 和文首写入顺序：术语卡计划用 `title` 新建主题。计划没有字段可以指定已经存在的主题，组合卡计划不写 `title`。组合卡 apply 之后用 `move-note` 移成主题的最后一个子 Rem，再用 `object-remove-tag` 去掉组合卡自己的 Tag。别名命令是 `remnote-cli sdk rem object-get-or-create-alias-with-text --target-id <ID> --args-json '[["<alias>"]]'`，每张标题卡一条。`update --add-aliases` 取决于 SDK 版本；SDK 0.0.46 报 `Native transactions are unavailable: SDK 0.0.46 cannot acknowledge abort`。先预览。预览输出只有 `validated` 和 `planHash`。读完计划文件并与已审草稿对照后，才显式 `--apply`。稳定键不冲突。没有实际对象时不调用空计划。`remnote-cli` 不得并行。
5. 普通卡按契约进入计划。任务入口负责最细 Concept Tag，以及可选的普通 `text` 主题。`cardCluster` 只在契约允许时为 true。原文模式先核对计划能表达完整正文、顺序和层级；超过限制不能截短或改写，应选完整支持该结构的现有入口，否则报告能力缺口。写后使用回执 ID 集中核对父级、类型、最终答案、标题卡引用和原生卡片集合；部分成功按 remnote-cli 的未知写入规则保存并核对，不整批重建。
6. persist_and_verify 调用 [remnote-verify](../remnote-verify/SKILL.md)，将真实正文与已审草稿逐项对应，改变正文的卡独立重审；未改变且证据新鲜的审查直接复用。写后重新判定 learning_preflight_passed。内容、结构、标题卡引用全部通过才报 flashcards_written、terms_referenced 与 result_verified，随后轻量复盘交付，不在复盘新增知识写入。缺少 Wiki 引用不算失败。
7. retrospect_run 只记录一条或零条发现。有发现时按 [notion-experience.md](references/notion-experience.md) 用 `ntn` 写入「remnote workflow 经验表」，不另建维护平台，也不在复盘里改 Workflow。

FANLOOP_REMN_EXPERIENCE_DATA_SOURCE_ID overrides the data source id; FANLOOP_REMN_EXPERIENCE_DATABASE_ID overrides the database id. Defaults live in references/notion-experience.md.

Auth is ntn login; headless hosts set NOTION_KEYRING=0. Do not require an integration API key.

Map 经验 to the one-line lesson, 日期 to the ISO run date, 主题 to the RemNote theme title, 类别 to 文案清晰度|工具链路|流程门禁|其他, 详情 to evidence + mechanism + suggested change + verification, and 编码状态 to 待编码 unless the finding already points at a merged fix (已编码) or is a non-code lesson (不适用).

When findings is non-empty, POST one page per finding with ntn api v1/pages to data source 「remnote workflow 经验表」; zero findings skip the write; a failed ntn call records notion_sync and still allows retrospective_recorded.

## 修改与核验

新卡片写入今日笔记并挂 HOME Tag，普通笔记和文章可写入今日笔记；补充已有卡片、文章或 Wiki 页面时更新原 Rem，保留原位置和 ID，不因日期变化创建副本。只维护本次相关内容；日常写入不顺带整理全库、重命名无关分类或清理 Portal。

只修改答案正文时，优先使用任务计划的 `answerPatches`，提供精确卡片/答案 ID 与刚读取的 `expectedTextHash`。计划不支持的原位结构操作先核对公开 SDK 能力及授权；需要背面字段时查 `object-set-back-text`，不可把卡片正面 ID 当作答案子 Rem ID。多行字符串不等于多个 Rem。必须保留卡片 Rem ID、cardId、位置和复习记录；能力不足或结果不明就 blocked，不用已拒绝的原子 update 命令、替代卡或删除重建绕过。

回读折叠内容时若有 `IsCollapsed`，按具体 Rem ID 继续读取，不能把省略当丢失。

完成前核对：

- `remnote-verify` 已证明：新单卡直接属于今日笔记并挂最细 HOME Concept Tag。多卡使用主题时，主题是普通 `text`、直接属于今日笔记、只由主题挂该 Tag。术语卡是主题的子 Rem，组合卡是最后一个子 Rem，组合卡没有 tags。`search-by-tag` 只返回主题。`review-stats --root-id` 在主题上列出本批全部卡片。`cluster_powerup` 与契约的 opt-in 一致；用户未要求时 `object-has-powerup ["cc"]` 为 false。意外的 true 判失败，不要补加，也不要改调 `object-add-powerup`。主题未被转换为 Document/Folder，标题没有残留 `/cluster`。
- 新建顶层主题、普通笔记和文章符合“日期 + 类型或来源 + 中文主题”；卡片正面、原文标题和 WIKI 规范名没有被该模板污染。单篇或单章的 `theme_title` 与 `source_h1` 按契约一致。
- 本次新增的 PARA 分类 Tag 按 `remnote-structure` 回读为 Concept，并且是领域分组下最细的那个 Tag。祖先路径证明在 `preflight.tag_path`。作者名、粗 Folder 和 Wiki 页都不能当这个 Tag。文章内容本身仍按文章规则保持 `text`。只有用户要求 Wiki 时，Wiki 页面才按 `remnote-wiki` 直接属于对应分区，并且不创建概念 Tag。
- 普通学习卡的 `preflight` 全部门禁为 pass。新建术语卡都有 `search --cards-only` 的 miss。复用的是术语闪卡标题 Rem。审卡回执含每张卡的五问 `kid_clarity`，且全部为 yes；组合卡列有逐字 `star_headings`。
- 原文导入闪卡一篇源文章对应一张卡，标题、正文顺序和可见内容与源文一致，Obsidian 元数据已移除，正文没有意外生成子卡片。
- 原文摘录闪卡的正面等于原文标题，背面只包含指定原文摘录；摘录内部顺序、可见内容和结构保持一致，元数据已移除，正文没有意外生成子卡片。
- 普通文章符合[文章格式](references/article-format.md)中的完成核验条件。
- 本次新增的 HOME 节点按 `remnote-structure` 区分 Folder 与 Concept；闪卡实体位于每日笔记并通过 Tag 归类，不创建查询 Portal。
- 组合卡引用的是标题卡 Rem ID，不是别名 Rem。默认批次没有新建或更新 Wiki 页面。用户明确要求 Wiki 时，页面才按 `remnote-wiki` 直接属于对应分区；那次写入不代替 terms_ready。
- 普通学习卡满足学习卡契约的失败条件。背面形状、短标题、别名、标题卡引用、STAR 原文、箭头、口诀和儿童口吻不在本清单再写一遍。
- 卡片保持确认后的排列与答案内容，不能只检查工具返回成功或数量。
- 每批草稿已经独立审查，最终回读正文逐项对应已审草稿；正文改变及失败卡已独立复审，未变部分复用新鲜证据。
- 删除概念前检查所有引用已迁移；删除物移入回收站，交付时告知可恢复。
- 用户要求 Wiki 时，WIKI 中没有生成卡片。普通笔记和文章仍保存真实实体，不创建指向同一实体的链接占位或迁移说明。
- 不为日常写卡添加额外索引、日志、同步系统或全库清理任务。

交付时简述写入哪个日期、多少张卡／篇笔记、使用哪个 HOME 分类标签和核验结果。
