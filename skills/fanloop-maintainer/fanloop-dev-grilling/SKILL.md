---
name: fanloop-dev-grilling
description: 对 Fanloop CLI 维护需求执行 design-tree/frontier 式批量拷问。用于把所有依赖已满足的决策分轮问完，直到与人形成共享理解。
---

# Grilling

把需求建模为 design tree：每个决策分支连接依赖它的后续决策。每轮重新计算 frontier，即所有
前置决策已经确定、现在无需猜测即可回答的问题；一次问完整个 frontier，每题编号并给推荐答案。
依赖本轮其他未决答案的问题留到下一轮。

每题格式：

```text
❓ Q1 - <标题>：<问题与选项>
➡️ <推荐答案与理由>
```

人的回答会改变 design tree；更新已决分支后再计算下一轮。事实调查由当前 Agent 使用仓库和工具
完成，不问人，也不要求子代理。frontier 为空后总结全部决策、约束、非目标与验收条件，并让人
确认共享理解；确认前不得实施。
