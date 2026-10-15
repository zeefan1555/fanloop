---
status: accepted
date: 2026-09-26
amends: ADR-0101, ADR-0108, ADR-0120
---

# 自动校验 Workflow YAML，不设专属人工审批

Workflow YAML 是配置。修改五份 YAML、绑定 Skill 或其参考材料不再仅因文件类型要求独立人工审批，也不再要求为 Step `id`、名称、顺序或 executor 差异单独等待人确认。产品需求批准、业务 Workflow 自身声明的 Human Step、独立行为试用、工作流注册中心发布回读及 Thrift IDL 人审仍按各自契约执行。

Fanloop 继续只使用现有共享加载路径：`yaml.v3` 严格解码并拒绝未知字段，生成 `yamlidl` 类型执行字段和枚举约束，`ValidateBundle` 校验五文件 Schema、Step、Prompt、Condition、Route、可达性、方向、互斥、歧义和引用关系，随后校验同 Bundle 的必需 Skill。任何失败都在激活或执行前关闭；不增加第二套 validator、预审服务、兼容层或 Runtime 专用分支。

验证阶段用候选公开 CLI 驱动隔离配置。代表性的语法、结构、Step、Condition、Prompt 与 Route 错误必须在 `flow init` 返回结构化 `WORKFLOW_INVALID`、退出码 `5`，并且不得创建 Requirement State。有效配置仍需按影响面完成 Route Matrix 或独立行为试用；机器结构校验不冒充业务正确性。

本决策不修改 `idl/yaml.thrift`、公开 CLI、Workflow 拓扑、State 或持久化 Schema。它只取代 ADR-0101、ADR-0108、ADR-0120 及仓库治理文字中 YAML 专属人工审核部分，保留这些 ADR 的验证控制面、live reload、工作流注册中心真值、原子激活、Dashboard 隔离与 fail-closed 规则。ADR-0052 的 YAML IDL 和跨文件验证架构保持不变。
