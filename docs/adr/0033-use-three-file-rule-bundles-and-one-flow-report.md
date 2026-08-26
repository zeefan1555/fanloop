---
status: accepted
date: 2026-08-12
---

# 使用三文件 Rule Bundle 和统一 Flow Report

Fanloop 的一个 Workflow 版本由不可拆分的 `workflow.yaml`、`flow.yaml`、`loop.yaml` 组成。`workflow.yaml` 只描述有序 Stage/Step 与执行方式；`flow.yaml` 为每个 Step 定义唯一准出 Rule；`loop.yaml` 定义 Step failure 与 external feedback 回流 Rule。Flow/Loop Rule ID 在整个 Bundle 内全局唯一，Workflow digest 覆盖三份文件的规范化语义。

所有状态上报统一使用 `flow report`。`progress` 只更新当前 Step 的非终态事实；`decision` 只提交 `rule_id` 以及可选 artifact、Evidence 和 Checks。Rule 决定向前、完成或回流，调用方不提交 pass/fail、failure name、feedback category 或目标 Step。Flow Rule 中声明的 artifact 归属于其 Step；回流时根据目标与 owner 自动失效。

这次切换直接删除单文件 Workflow、`on_failure`、`feedback_rules`、独立 `loop.feedback`、旧 Report/Event 结构及 decoder，不提供 alias、双读写、迁移 adapter 或 Feature Flag。State Schema 5 和 Event Schema 5 反映发生字节变化的 durable 格式；独立 `card render` 继续与 Flow/Loop 解耦。Card 的最终落盘格式由 ADR-0034 约束。四份 Agent 手册由同一 Bundle 生成并在发布及 Doctor 中校验。

本决策取代 ADR-0029 中的 `on_failure` 部分、ADR-0031 中的独立 Loop/category 和旧 Report 契约；有序 Step、调用方不能提交目标、Card 与发送解耦等仍兼容的决策继续有效。
