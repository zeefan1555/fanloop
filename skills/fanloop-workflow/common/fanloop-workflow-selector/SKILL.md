---
name: fanloop-workflow-selector
description: 在新 Fanloop Requirement 初始化前，根据显式选择或默认规则选择正式 Workflow。用于尚无 .fanloop State 且即将执行 flow init 的场景；已有 Requirement 不得重新选择。
---

# 选择初始 Workflow

只在 `flow status` 返回 `NOT_INITIALIZED` 时选择。已有 State 时立即返回其中固定的
WorkflowRef，不刷新规则、不根据当前人员或仓库重选。

仅在没有 State 时，严格按以下优先级选择一个 Workflow ID：

1. 调用方明确提供的 Workflow；
2. `default`。

## 版本化路由规则

以下内联块是本 Skill 的唯一路由规则真值；键和值均按字面量精确匹配：

<!-- fanloop-selector-routes:start -->
```yaml
schema_version: 1
default: promotion-design
repositories: {}
departments: {}
```
<!-- fanloop-selector-routes:end -->

选择后确认该 Workflow 存在于当前配套 Release，再把 ID 原样传给：

```bash
fanloop flow init --root <ABSOLUTE_ROOT> --workflow <WORKFLOW_ID> ...
```

未命中规则时使用 `default`。显式值或已命中规则的目标不存在时停止并报告配置错误，
不得静默回退。输出所选 Workflow ID、命中层级和使用的非敏感证据；不写 State。
