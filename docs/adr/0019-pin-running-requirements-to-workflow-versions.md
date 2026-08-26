---
status: accepted
amended_by: ADR-0029
---

# 运行中需求固定 Workflow 版本

State 绑定初始化时的 Workflow ID、版本和摘要；同一版本内容不可修改。发布包只携带当前受支持的 Workflow，不保留旧 Schema 或 `flow migrate`；旧存储需要在切换前结束，新需求使用发布清单指定的当前版本。
