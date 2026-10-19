---
name: build-workflow-candidate
description: 在 manage-fanloop-workflow 的构建节点中依据已确认分析生成五份 YAML 与绑定 Skills，并完成静态 Bundle 校验和审核包。
---

# 构建 Workflow 候选

从 `workflow_document_url` 读取 Step 设计、Condition 与 Route、Output 设计和 Skill 计划四个章节。任何章节
缺失、互相矛盾，或目标、责任、Step 拓扑、验收边界已经变化时，只返回
`workflow_analysis_changed`，不要用生成配置替代重新分析。

在临时目录生成候选。目录名必须等于 `workflow.yaml.id`，包含根 `SKILL.md`、五份 YAML 与所有绑定
`skills/<skill-id>/SKILL.md`。使用当前已安装 CLI 接受的 Schema。不要修改 Runtime、IDL、仓库 seed，
不要加入兼容字段或 fallback。

按以下边界检查候选：

1. 五份 YAML 能被当前已安装的 fanloop 严格加载，引用闭合。把候选放到隔离配置源的 `fanloop-workflow/<workflow-id>/`，用独立 `FANLOOP_DATA_HOME` 与绝对 `FANLOOP_CONFIG_SOURCE` 运行 `fanloop doctor`，不得出现 unhealthy。
2. Step 职责与 `executor` 明确；正常、终止和回流 Route 目标闭合；每个 Condition 被至少一条 Route 使用。
3. 每个 SkillBinding 指向同 Bundle 的真实非空 Skill，并显式声明 `optional`。
4. Human Gate 保留；Agent 不替人批准；影响 Route 的事实不能只写入 Evidence 或 Skill 禁止语句。
5. Output key 稳定且使用最窄类型；用户可见名称写在 `description`，不翻译 key。
6. 按分析结论决定是否加入 Panorama。需要时把普通 Condition、`panorama_snapshot_path:path`、包内投递 Skill 和选定 Route operand 一起写入候选；不需要时完全省略。不得依赖任何 Runtime action 或 acknowledge 接口。

## Step ID

更新已有 Workflow 时逐项回读分析中的 Step 身份映射表：

- 未改变身份的 Step 保留原 ID。
- Fanloop 不接受 Step `replaces`，也不返回 `WORKFLOW_RECONCILIATION_REQUIRED`。不要在 YAML 中写入该字段，也不要为改 ID 初始化 Requirement 或做迁移试用。
- 同一业务节点必须保留原 ID。删除、拆分、合并、executor/责任变化或语义变化都按真实增删记录，不得伪装成改名。
- 映射缺失、重复或有歧义时停止构建并返回 `workflow_analysis_changed`，不得猜测或发布。

默认只做静态配置校验，禁止普通 Route 正负例、Card 渲染、Panorama 渲染或格式验证脚本。静态校验通过
不等于行为验收，审核包必须明确这一点。不执行 Requirement 初始化。

候选仍可写到临时 `.manage-fanloop-workflow/candidate/<workflow-id>/` 供打包，但它不是交付产物。把完整五份 YAML、
全部绑定 Skill、Step 表、Condition/Route 矩阵、完整 diff、静态校验 argv/退出码/结果、未验证范围和候选身份
交给 `explain-workflow-design`，更新唯一飞书文档内的主审核 HTML。第一屏必须显示用户要求、真实基线、
精确候选身份、真实改前/改后拓扑和等待人的决定。随后展示 Step、Condition/Route、Output、Skill 的变化，
明确“会影响”和“不会影响”的边界、风险、未验证内容和一个真实失败反例。完整 YAML、Diff 与证据放在可
展开区域，不在飞书正文重复粘贴。

候选 HTML 写入后记录并锁定候选身份、渲染数据摘要和 HTML 内容摘要。人工审核前后不得无声改写；任何候选
变化都必须重新生成 HTML，并使旧批准失效。

返回的 `workflow_candidate_ready`、`workflow_review_package_ready` 和 `workflow_review_visual_ready` 都是对象，
包含同一 `document_url`、主审核 HTML 的 `block_id`、`revision_id` 和候选身份；不得返回本地路径作为用户产物。

候选完成时一次返回：`workflow_candidate_ready`、`bundle_schema_validated`、`route_model_validated`、
`skill_bindings_validated`、`output_contract_validated`、`workflow_review_package_ready` 和
`workflow_review_visual_ready`。
