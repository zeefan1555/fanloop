---
status: accepted
date: 2026-08-29
amends: ADR-0006, ADR-0052, ADR-0075, ADR-0079
---

# 运行时不认识生产 Workflow 事实

Fanloop 是解释固定 YAML Bundle 的通用 Loop 引擎。`cmd/`、`internal/` 与 `scripts/` 的
生产代码不得硬编码任何已发布 Workflow、Step、Condition、Output 或原子 Skill ID；新增
Workflow 时只允许增加 `workflows/<workflow-id>/` 的五份 YAML、同名
`skills/<workflow-id>/` Skill 组，以及 `entrypoints/fanloop-workflow/routes.yaml` 的显式
场景映射，不修改 Go、Node 或 Shell 实现。

五份 Workflow YAML 继续是 Stage、Job、Step、原子 Condition、正常/回流 Route、Prompt 与
SkillBinding 的唯一真值。Runtime 只做严格加载、图不变量校验、状态转换、Output 失效与投影，
不解释自然语言，不加载脚本钩子，不提供继承、插件或任意本地 Workflow 热加载。场景选择继续
没有默认值；缺少显式场景或映射时失败关闭。

Trace Registry 的部署差异进入 `internal/traceconfig/registry.yaml`。该文件随二进制嵌入并严格
校验，按 profile 提供默认策略和可选 Workflow 覆盖，包括 Registry 地址、是否要求 CLI 日志
文档、远端字段名及 Workflow Output 到远端字段的映射。Trace、State、Flow 自动 provision 与
Card Projection 只消费同一个 `traceconfig.Resolve(profile, workflowID)` 接口，不再比较具体
Workflow ID。维护者 Registry 的需求澄清、技术方案、MR 与 CLI 日志字段继续由配置映射保留。

Card 对 URL Output 的展示名称直接使用 `condition.yaml` 已有 `output.description`，缺失时回退到
Output key；Trace 头部只显示 Requirement、来源、绑定 Workflow、Release、更新时间和 Loop 次数，
全部业务 Output 统一进入当前 Output 表。发布后冒烟从 `release.json` 枚举 Workflow，并验证每个
Workflow 都能初始化到一个非空当前 Step，不再指定某个业务 Workflow 或 Step。

源码开发态与安装 Release 都按绑定 Workflow ID 解析
`skills/<workflow-id>/<skill-id>/SKILL.md`；Manifest 中 Skill ID 仍保持全 Release 唯一。
这保留 ADR-0057/0079 的 Release-bound path 与一一对应目录约束，并修复源码开发态仍按旧扁平
Skill 目录查找的问题。

本决策不修改五份生产 Workflow YAML 的任意字节，不修改 Step `id`、`name`、顺序或 `executor`，
不修改 Condition、Route、Prompt/SkillBinding、`idl/*.thrift`、生成物、State/Event 或公开 CLI
Request/Response。它保留 ADR-0038 的原子 Condition 与显式 Route、ADR-0052 的 Thrift-first YAML
authoring、ADR-0062 的当前 Bundle、ADR-0075 的完整 CLI 日志风险边界，以及 ADR-0079 的目录一一
对应和无默认 Workflow；只把遗留的业务分支迁移到配置或删除专用投影。
