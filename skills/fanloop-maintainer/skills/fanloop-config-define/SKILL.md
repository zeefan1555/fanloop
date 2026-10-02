---
name: fanloop-config-define
description: 为纯 Workflow YAML、Skill 指令和参考材料改造确认文件范围、现有能力与 live 交付目标；仅在 Maintainer Define 的配置 Route 使用。
---

# 配置范围

输入是用户目标和待修改的 Workflow/Skill。要求已明确就直接使用，不重复访谈。仅在命令能力不确定时读取对应叶子帮助或实现；普通文字修改不做无关能力调研。

确认只涉及已有 Schema 下的 YAML、Skill 指令/参考材料和必要说明。新增可执行脚本、CLI 能力、IDL/Schema、Runtime 或测试基础设施不属于本 Skill，记录范围变化并回 Define。

在唯一阅读文档 `delivery-record.md` 简记目标、验收点、初始文件、仓库绝对路径、Workflow ID、Git 基线、源码工作区、实际本地配置源、修改前摘要、开始时间和授权来源；新文件摘要记为不存在。`configuration_scope_recorded` 对象引用此记录并保留范围事实。原始工具日志另存或引用 CLI transcript，不复制正文。四步共享此文档，不创建阶段报告、不发布飞书。完整包仍是 skills/<workflow-id>/ 下根 SKILL.md、五份 YAML、子 Skills 和资源。

记录 `trial_plan.method`：普通模板、说明或 Prompt 生成要求为 `self_check`；仅错字排版且行为不变为 `not_applicable`；路由、绑定、权限、安全、审批或外部写入规则变化为 `independent`。按实际语义判断，不按后缀免测。需要试用时简记最小代表性场景、成功标准、数据、允许副作用和停止位置，覆盖关键失败边界；不扩成全产品回归。

当前 Agent 连续执行普通配置四步，不派发实施或验收 Agent。`independent` 才派发全新试用；无真实业务写入授权时使用隔离配置和专用测试数据。免测写具体理由，不能冒充 passed。

范围清楚就结束 Define，不生成 Spec、Tickets 或代码 Verification Contract。同一目标所需关联说明可补入实际文件清单及修改前摘要，不回 Define；只有目标、验收、权限、副作用或类别扩大才携原类别回 Define。后续沿用 `maintenance_kind=configuration`。
