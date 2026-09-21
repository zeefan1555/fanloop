---
name: flashcard-quality-review
description: 冷读并独立审查闪卡草稿的语义、SuperMemo v1 学习质量、来源忠实度、隐私和 Decks 合规性。用于 material-flashcards 的独立质量审核 Step；不修改输入。
---

# 独立审核卡片质量

把自己视为没有聊天上下文的首次复习者。读取经 SHA 绑定的目标、来源理解、选择、计划和精确草稿；不得修改任何输入，也不得在报告中补写缺失事实。

## 逐卡门禁

逐卡检查并给出证据：

- 内容已经理解，且建立在必要基础上；
- 值得长期记忆，具有现实适用性和合理优先级；
- 一卡一个最小检索目标，没有机械集合或长枚举；
- 问题脱离原文可独立、唯一作答，必要上下文足够但不过载；
- 答案忠于来源，来源事实、归因判断、用户理解与 unknown 没有混写；
- 必要来源、日期或版本保留；
- 个性化例子、经历、情绪、动机、诊断和因果都有来源或用户确认；
- 相似卡不造成概念干扰，受控冗余确有不同检索价值；
- 四项 v1 non-goal 没有被伪装实现；
- 绑定 `flashcard` Skill 自身的 Decks 文件级校验全部通过。

Decks 格式、标签、解析和概念卡实现只由绑定 `flashcard` Skill 判断；本 Skill 不复制其规则。

## Twenty Rules 审查

报告必须列出规则 1–20，每条状态只能是 `pass / fail / not-applicable`，并与固定的 8 adopted / 8 partial / 4 non-goal 映射一致。`not-applicable` 必须有事实理由；适用但未满足时只能是 `fail`。

## 报告与路由

报告记录 `card_draft_path`、草稿原始字节 SHA-256、逐卡结果、20 rules 结果、Decks 校验和完整 findings。完整报告只写 Requirement Root 内权限受限、通用命名的文件；Result Evidence 只给同一相对路径和报告 SHA，可附固定分类与最小非敏感摘要。

零阻塞项时同时上报 `quality_review_written` 与 `card_quality_passed`。有阻塞项时上报 `quality_review_written`，并且只选择最早受影响层：

1. 目标问题 → `review_goal_changed`；
2. 来源或证据问题 → `source_understanding_changed`；
3. 价值或选材问题 → `knowledge_selection_changed`；
4. 类型、原子性、集合/枚举或干扰问题 → `card_plan_changed`；
5. 措辞、上下文或 Decks 实现问题 → `card_draft_changed`。

不得用多个回流 Condition 代替最早层判断，也不得把卡片正文、个人细节或完整 findings 放入 Summary、Evidence、Event 或 Trace。
