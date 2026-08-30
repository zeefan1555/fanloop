---
name: technical-solution-writing
description: 将八个已确认片段和附录组装为只有九个正文结构的正式技术方案。用于 technical-solution-design 的方案成文 Step；不得重新推导或静默修改上游结论。
---

# 组装正式技术方案

读取已确认的 `.technical-solution/sections/01-background.md` 至 `08-delivery.md`、架构图及引用证据。
先把数据口径、容量测算、对比明细、接口明细、补充时序和开放问题整理到
`.technical-solution/sections/09-appendix.md`；附录片段同样不得包含 Markdown 标题。

组装 `technical-solution.md`，结构必须精确为：

```markdown
# <能表达方案核心结论的项目标题>
## 1. 需求背景
## 2. 核心问题
## 3. 设计目标
## 4. 方案调研
## 5. 总体方案
## 6. 难点解法
## 7. 方案收益
## 8. 落地规划
## 9. 附录
```

除项目标题外只允许这九个 Markdown 二级标题，禁止 `###`、`####` 和 `1.1` 式可见子标题。章节
内部用加粗结论、短段落、表格、列表、图片、Mermaid 和引用组织细节；不得为了消除子标题而删掉
信息。每一屏内容都应有明确结论、事实与推导，避免大幅留白。

标题要短、具体、有吸引点，使用“描述性定语 + 通俗后台术语”，避免“可用性”“性能”一类空词。
关键数据标明来源和口径；已测、估算、目标与待确认必须区分。总体架构图嵌入第五节并解释边界、
上下游、组件、依赖、数据流和箭头含义。

本 Step 只允许机械组装、去重和呈现修正。若发现片段间存在语义冲突，不自行选择答案，按最早受
影响层上报 `background_changed` 至 `delivery_changed` 中的一项，并在 Evidence 写明冲突、保留
内容、失效产物和回流 Step。

写入后回读并验证：恰好一个项目标题、九个规定标题且顺序正确、没有更深标题或 `1.1`、九节均
非空、图文链接有效。成功时上报 `technical_solution_written` 和 `technical-solution.md` 路径。
