---
status: accepted
date: 2026-09-23
amends: ADR-0104, ADR-0115, ADR-0116, ADR-0117, ADR-0118
---

# 每个 Workflow 必须声明并使用全景图 Condition

用户明确要求每个 Workflow 都有全景图 Condition，补齐 RemNote，并将限制写入 AGENTS.md。
检查四套生产定义后，RemNote 与 pstack-native 均缺少该 Condition，统一补齐。

两套新增 `panorama_presented`，输出 `panorama_snapshot_path:path`，通过 Condition Prompt 绑定
各自 live 根中的必需 `workflow-panorama` Skill。沿用现有 maintainer 的 renderer 与宿主投递协议，
每次进入 Step 或回流重入先展示，全量展示成功才提交精确快照路径；失败或结果不明保持 blocked。
正常、终止和回流 Route 的每个 AND 组均加入该 Condition，例如：

```yaml
# before
when: {any_of: [[home_tag_ready]]}
# after
when: {any_of: [[home_tag_ready, panorama_presented]]}
```

ADR-0117 删除旧 Decks Panorama Skill 的决定保留；RemNote 从现在起使用本流程的展示 Skill。
ADR-0104 的展示顺序重新适用于 RemNote，并适用于 pstack-native。ADR-0118 的 47 个原生 pstack
Skill 正文保留；新增的 Fanloop 展示 Skill 不属于原生 catalog。其余两套已有 Panorama 契约保持原样。

所有 `workflow.yaml` 保持字节不变：Step id、name、顺序和 executor 无新增、删除、改名、重排或变化。
不改 Runtime、Schema、Thrift、持久化或公开 CLI。Runtime 仍只验证回执类型与路由，不证明宿主已展示。
安装保留现有 live 内容的规则继续成立；源码修改不等于已更新用户的 live Workflow。

## 配置与代码验证边界

用户进一步明确：Workflow 和 Skill 指令是配置，配置变更与代码变更分开处理。
本决策修订 ADR-0115/0116 中将所有变更都送入代码验证与外层循环的适用范围。
纯配置在已有 Schema 和 Runtime 下检查 Bundle、引用、必需 Skill 与路由完整性，按需用已有
兼容 CLI 做隔离验证；不要求修改代码测试或 Golden、构建 Release、安装 CLI 或代码验证 manifest。
可执行 Skill 脚本、Schema、加载器、Runtime 和验证基础设施的修改仍属于代码变更。

本次撤回为全景图配置配套增加的 Go 测试及 Contract case/Golden 修改，只保留配置及治理文档。
当前代码测试直接断言生产配置，CI 也无条件执行代码验证；这项既有耦合尚未重构，不能宣称代码
测试已独立于生产配置。本次规则变更不是 CI 实现变更，后续框架解耦应作为单独代码改动处理。

2026-09-23 补充：用户明确轻量化仍须经过真实步骤并完成合码与 live 交付。保留原四个 Step 的 id、名称、
顺序和 executor，每步增加 `configuration_change` 路由；原代码事实组合额外带互斥的 `code_change`，
原代码 Prompt、必需 Skills 和验证保留。首条 Route 的 Prompt 分流：代码读取 available_routes 中的原 Prompt/Skills，
配置执行范围记录、修改、轻量检查、PR 合码和 live 同步。每个 AND 组均要求 Panorama；不改 Runtime 或 IDL。
配置交付不要求代码 manifest、构建或 CLI 安装，但必须回读 MERGED、merge tree 和 live 文件，不能停在工作区。
本补充取代本 ADR 与 AGENTS 中“纯配置不初始化 Maintainer”的表述；只有明确只改本机 live 的请求直接编辑。
RemNote Card Cluster 使用已有 SDK 的 `cc` 添加/查询，以 CLI 回读 true 为准，取消 UI 添加与截图要求。

2026-09-23 Skill 职责拆分：四个 Step 与既有 Condition 组合不变；Define 记录 maintenance_kind，后续
按该输出选 Route，范围变化先回 Define。配置范围/修改/检查/交付分别绑定 fanloop-config-define、
fanloop-config-edit、fanloop-config-check、fanloop-config-deliver，原代码 Prompt/Skills 保留。
公共 current.prompt 仅绑定路由 Skill；配置业务 Skill 绑定在各 Step 对应 Condition Prompt，按所选
Route 加载，避免首条配置 Route 的 Skill 被代码任务无条件执行。CLI 仍列出两类候选，未新增引擎动态过滤能力。
本次按用户明确要求新增四份配置 Skill，取代先前将两套业务步骤写入公共 Skill 的做法；不配套修改
冻结 Skill 数量的代码测试，既有测试与生产配置耦合仍是独立代码问题。
