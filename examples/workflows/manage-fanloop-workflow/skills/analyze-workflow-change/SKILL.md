---
name: analyze-workflow-change
description: 在 manage-fanloop-workflow 的分析节点中读取个人 Fornax 基线，并把用户目标建模为 Step、Condition、Route、Output 与 Skill 方案。
---

# 分析 Workflow 改动

只处理 Fanloop Workflow 配置。不要克隆或修改 `fanloop` 代码仓库，不要运行 Git 构建、PR 或代码维护流程。

读取用户材料，整理输入、输出、执行者、人工决定、失败处理、工具权限和完成标准。区分已确认事实、建议与
开放问题。只有缺口会改变范围、责任、Step 拓扑或验收边界时才返回
`workflow_analysis_incomplete` 并提问。

只使用个人 Fornax CN workspace `7543941556223377409` 和 key 前缀 `fanloop.workflow.`。
不得查询或写入团队空间 `7590123861772503042`。使用当前用户已登录的 `fornax-cli`，不传 AK/SK、JWT、签名或下载 token。

- 创建时分页查询 `fanloop.workflow.*`，确认 exact key 和包内 Workflow ID 均未占用。
- 更新时从 exact key 的最新 Online/prod released 版本下载基线，不使用未发布 commit、本地旧副本或同名猜测。

读取叶子 Help 后使用与 Fanloop 相同的只读命令。workspace 固定为 `7543941556223377409`，region 为 CN：

```bash
fornax-cli skill list --keyword fanloop.workflow.<suffix> --page-no 1 --page-size 100 --custom-region CN --workspace-id 7543941556223377409 --format json --output <dir>
fornax-cli skill list-releases --skill-id <SKILL_ID> --env Online --feature prod --custom-region CN --workspace-id 7543941556223377409 --format json --output <dir>
fornax-cli skill install --custom-region CN --workspace-id 7543941556223377409 --skill-query-json '[{"skill_key":"fanloop.workflow.<suffix>","version":"<version>"}]' --dir <dir>
```

`<suffix>` 是 Workflow ID 中 `-` 换成 `_` 后的结果。exact key 是 `fanloop.workflow.<suffix>`。

先为每条正常、终止和回流 Route 建模必须成立的业务事实，再设计 Prompt 和 Skill。一个 Condition 只表达
一个可由真实文档、平台回执或人工回复验证的事实，使用最窄的 Output 类型。Evidence 只用于审计。
人工决定使用 `enum_value`、`exclusive_group` 和绑定本次提问与回复的回执。

Panorama 与其他能力一样由 Workflow 自己选择。需要时声明普通 Condition、Output、Prompt/Skill 绑定和精确 Route operand；不需要时不添加。不要依赖 Runtime action、保留 Condition 或 acknowledge 命令。

分析中必须列出：

- 创建或更新身份、当前与目标流程。
- Step `id`、`name`、顺序和 `executor` 的保留、新增、调整与删除。
- 更新已有 Workflow 时列出 Step 身份映射表：

  ```text
  旧 Step ID | 新 Step ID | 是否同一业务节点 | 处理方式
  ```

  默认保持原 Step ID。Fanloop 没有 Step `replaces` 字段，也没有 `WORKFLOW_RECONCILIATION_REQUIRED`。
  同一业务节点必须继续使用原 ID。确需换 ID 时，把它写成删除旧 Step 并新增 Step，不能写成身份延续。
  无法从材料和基线确认映射时返回 `workflow_analysis_incomplete`，不猜测、不生成候选。删除节点必须单独评估对当前 Step、历史 Event 和 Output producer 的影响。
- Human Gate、正常 Route、终止 Route、回流目标和失败时停留位置。
- “问题或不变量 -> Condition ID -> Output key/type -> 证据来源 -> Route AND 组 -> 回流”矩阵。
- 可见 Output 的稳定 key 与业务 `description`。
- 绑定 Skill、所需工具、权限、非目标和静态验收边界。
- 明确说明不执行普通 Route 试用、Card/Panorama 渲染或行为验收。不规划 reconciliation 迁移验证。

从已有 Output 取得 `workflow_document_url`。所有审核材料写入该文档，不把本地 Markdown 当作交付产物。
临时文件只用于组织写入内容，成功更新文档后即可清理。

把分析整理成审核 HTML 所需的结构化数据：

- 用户原始要求和不丢语义的一句话摘要。
- 目标 Workflow、创建/更新类型、Fornax key/Skill ID、真实基线版本与候选身份占位。
- 推荐方案、需要人决定的问题、开放问题、主要风险和明确不变边界。
- 真实改前拓扑，以及计划中的改后拓扑；未生成候选的内容必须标为“计划”，不能伪装成已完成。

- Step 设计：Step、Human Gate、回流和身份映射。
- Condition 与 Route：业务事实、Condition、Output、证据、Route AND 组和回流。
- Output 设计：Output key、类型、description、生产者和使用位置。
- Skill 计划：绑定 Skill、optional、工具权限、输入输出、失败处理和非目标。

飞书正文不重复铺开这些长表，只保留极短索引。把上述数据交给 `explain-workflow-design`，创建或更新唯一主
审核 HTML Block。每个 ConditionResult 至少包含 `document_url`、同一 `block_id`、`revision_id`、数据分组
和本次摘要。数据及 HTML 全部形成并回读后，一次返回
`step_model_ready`、`condition_route_matrix_ready`、`output_contract_plan_ready`、`skill_plan_ready` 和
`workflow_design_visual_ready`。缺少任一部分时不得返回完整组合，也不得进入构建。
