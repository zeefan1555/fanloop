---
name: fanloop-workflow-selector
description: 在新 Fanloop Requirement 初始化前，根据用户显式选择的场景映射正式 Workflow。用于尚无 .fanloop State 且即将执行 flow init 的场景；已有 Requirement 不得重新选择。
---

# 选择初始 Workflow

只在 `flow status` 返回 `NOT_INITIALIZED` 时选择。已有 State 时立即返回其中固定的
WorkflowRef，不刷新规则、不根据当前人员或仓库重选。

仅在没有 State 时读取用户显式选择的场景。没有选择时展示下方可用场景及说明并等待，
不得猜测、默认选择或执行 `flow init`。场景按 key 字面量精确匹配，再映射为 Workflow ID。

## 版本化路由规则

以下内联块是本 Skill 的唯一路由规则真值；键和值均按字面量精确匹配：

<!-- fanloop-selector-routes:start -->
```yaml
schema_version: 2
scenarios:
  technical-solution:
    workflow: technical-solution-design
    description: 编写、评审并人工确认技术方案
  fanloop-maintenance:
    workflow: fanloop-maintainer
    description: 维护 Fanloop 自身代码、Workflow 与 Release
```
<!-- fanloop-selector-routes:end -->

选择后确认该 Workflow 存在于当前配套 Release，再把 ID 原样传给：

```bash
fanloop flow init --root <ABSOLUTE_ROOT> --workflow <WORKFLOW_ID> ...
```

场景不存在或映射目标不在当前配套 Release 时停止并报告配置错误，不得静默回退。
输出所选场景、Workflow ID 和使用的非敏感证据；本 Skill 不写 State。
