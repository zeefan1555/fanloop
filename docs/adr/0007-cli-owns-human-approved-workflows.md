---
status: superseded by ADR-0029
---

# CLI 持有人工确认的固定 Fanloop Workflow

Fanloop Workflow 的合法状态、动作、人工门禁、反馈与失败分类、产物 Schema、Card 规则和回流关系由人评审后写入单个版本化 JSON，通过 Go 强类型校验并在发布时嵌入 CLI。CLI 根据当前状态和执行结果选择合法 Route，Workflow Skill 只能执行 CLI 返回的动作，调用方不能直接提交目标状态。通用性表示同一运行时可以注册多套固定 Fanloop Workflow，不表示运行时解释自然语言流程、继承或引用其他配置、加载脚本钩子或运行插件；只有真实维护成本出现后才拆分文件。
