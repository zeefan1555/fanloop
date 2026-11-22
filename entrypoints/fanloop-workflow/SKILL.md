---
name: fanloop-workflow
description: 继续已初始化的 Fanloop Requirement 或检查 pinned Runtime 状态。新品流程已迁移到 second-brain-agent 的 Codex 插件；不要再创建新生产 Workflow。
---

# Fanloop 旧 Requirement 兼容入口

Fanloop 继续保留通用 Runtime，但当前生产产品工作流已迁移为 `second-brain-agent/plugins/` 下的 Codex 插件。本 Release 的 `routes.yaml` 没有新场景。

## 选择入口

- 用户要制卡、记日记、做账户早报或 TradingView 回测：读取 `second-brain-agent/plugins/` 下对应插件，不创建 Requirement。
- 用户要修改代码、设计或编辑文档：直接按任务执行，不把普通任务改造成 Workflow。
- 用户明确要继续一个已经存在的 Fanloop Requirement：使用该 Root 的固定控制器读取最新 Status，按 pinned 快照继续；不切换到新配置，也不升级成插件运行。

## 旧 Requirement

解析绝对 Requirement Root 后，先检查 `<ROOT>/bound-release-home/current/bin/fanloop`。存在时设置对应的 `FANLOOP_DATA_HOME` 和四个 Skill Root，清除 `FANLOOP_CONFIG_SOURCE`，只用该控制器执行 `flow status`。失败就报告阻塞，不回退到全局 current、其他 Release 或改写 State。

没有固定控制器的历史 Root 只读检查；不得伪造控制器或为旧 State 提供迁移。

## 新 Root 边界

`routes.yaml` 的 `scenarios` 为空。不要猜 Workflow ID、调用 `flow init`、恢复已退役包或把 Codex 插件包装成 Workflow。需要新产品流程时，在父仓库新建或维护插件 Skill。

已有 Requirement 的 Status、Prompt、Condition、Route 和 pinned Skill 是它的历史事实；只有用户明确要求继续该旧 Run 时才恢复。
