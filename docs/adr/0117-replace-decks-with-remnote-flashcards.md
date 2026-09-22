---
status: accepted
date: 2026-09-22
amends_inventory: ADR-0110, ADR-0115
amends_scope: ADR-0104
---

# 用 RemNote 四步流程替换 Decks 制卡

用户先在对话中理解概念，再启动制卡。以 `remnote-flashcards` 完全替换
`material-flashcards`：删除旧 Workflow、Decks 专属 Skills 与入口，不保留别名、兼容或迁移分支。
其余三套生产 Workflow 不变，库存仍为四套；ADR-0110/0115 的旧制卡 ID 库存声明由本 ADR 取代。

新流程只有一个 Stage、一个 Job，四个 Agent Step 按下列顺序执行：

| ID | 名称 | executor |
| --- | --- | --- |
| resolve_home_tag | 定位分类 | agent |
| write_flashcards | 写入闪卡 | agent |
| update_wiki_links | 更新 Wiki 与引用 | agent |
| verify_result | 最终校验 | agent |

旧九步全部删除：frame_review_goal（确认复习目标，agent）、understand_source（理解原始材料，agent）、
select_knowledge（筛选高价值知识，agent）、plan_card_set（选择类型并原子拆卡，agent）、
draft_cards（生成或修订卡片草稿，agent）、review_card_quality（独立质量审核，agent）、
confirm_card_preview（预览并确认卡片，human）、persist_approved_cards（写入已批准卡片，agent）、
validate_persisted_cards（落盘后校验，agent）。固定 Human Step 与旧 Panorama Skill 随旧流程删除；
ADR-0104 对其他 Workflow 的 Panorama 投递规则不变。

HOME 分类先查找最具体的 Concept Tag，没有则当步创建并回读。写卡复用既有 RemNote Skill，按独立
记忆目标查重：重复跳过、实质补充更新原卡、新目标新建，允许 0..N。新卡属于实际创建日期的每日笔记，
单卡或普通 text 多卡主题挂 HOME Tag，多卡使用内置 Card Cluster。卡片先独立可答；第三步更新 Wiki
并用真实页面 ID 补原生引用；第四步回读卡片和相关 Wiki，写 Wiki 时确认 WIKI 实体子树零卡片。
失败回最早受影响步骤，重试从历史 Evidence 恢复真实 ID 并原位修复，保留复习记录。

六个已有 Skills 与必要 references 放在新 Workflow 同一 live 根。Skill 负责内容规则与实际操作，
五 YAML 负责四条正常/终止 Route、十条回流与输出。Runtime 只校验现有类型和 Route，不认证业务事实；
`result_verified` 是 RemNote 内容检查，不是 Fanloop `verify.evidence.outcome`。不改 Runtime、IDL、
存储或 YAML Schema，不在本地保存 RemNote 内容副本。只读评审不进入此四步入口；原文模式保留原规则。

ADR-0108 的 live 真值与安装不覆盖规则保留。新安装从新库存播种；本次既有安装在合码后单独定向部署
新目录并移除旧目录，先验证隔离副本，仅另同步 maintainer 中 workflow-products.md 的本次产品说明，其余 live 字节保持不变；既有全局实际三套，不能额外播种 pstack。旧 ID 的 Requirement 不迁移。
本次公开 CLI 验收使用明确标记的合成 ID 回执，只证明编排；不调用用户真实 RemNote 库、不声称业务写入验证。

审核来源：用户确认上述四步并要求按五 YAML 实现；主 Agent 在 host turn
`01a0c758-587c-7033-a315-406d625058ea` 完整审核批准 Q1–Q8、旧新 Step diff、六 Skill 边界、三个公开
CLI 验收场景及定向 live 部署。receipt：
`decision-0d1365b20069008489ccdb3b7c1546e32c38398d126fd703065a6dc084d0cf84`。

用户要求先刷新本地自迭代流程后，新 Root remnote-native-workflow-v2-20260922 以四步 maintainer 初始化。
主 Agent 在 host turn `01a0c77f-3a09-7693-ae89-f3115ce147c3` 再次批准相同产品契约及上述精确 live 部署白名单；
Define receipt 为 `decision-e0148ea99b33fbcbefcd051c012a84c3e0230c07d90ca6c073ef5e9249c1b38a`。
