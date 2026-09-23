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
