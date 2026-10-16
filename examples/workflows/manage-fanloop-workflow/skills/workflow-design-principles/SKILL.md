---
name: workflow-design-principles
description: 从可验证的 Output 与 Condition 反推 Fanloop Workflow 的 Route、Step、Prompt 和 Skill，并保持失败路径与人机边界显式。
---

# 从结果反推 Workflow

设计和复核 Fanloop Workflow 时使用以下顺序。不要先画 Step，再为已有 Step 补理由。

1. 写清最终必须留下的业务结果，以及谁会使用它。
2. 为结果定义稳定 Output key、最窄类型、业务 description、唯一生产 Step 和使用位置。
3. 把“结果为什么可信”拆成原子 Condition。每个 Condition 只表达一个可由文档、平台回执或人工回复验证的事实。
4. 先闭合正常、终止和回流 Route 的 `when.any_of`。外层是 OR，内层是 AND；所有前进所需事实都必须在同一 AND 组中。
5. 同时定义失败后停留位置、回流目标和恢复条件。不要只设计成功路径，也不要用 Prompt 禁止语代替 Route。
6. 只在执行者、Human Gate、持久 Output 或独立失败边界不同时拆 Step。其余工作留在同一 Step 的 Skill 中。
7. 最后才写 Prompt 和绑定 Skill，让它们生产既定 Output 并满足既定 Condition，不得反向发明新的流程语义。

## 判定原则

- **证据贴近事实**：Condition 的 Output 保存可复核结果；Evidence 只用于审计，不能参与路由或代替 Condition。
- **Route 完整闭合**：每个 Condition 都被实际 Route 使用；每条 Route 都有明确目标；零匹配、多匹配和悬空事实都属于设计错误。
- **契约保持最小**：不为展示、措辞或中间推理新增持久 Output、Condition、Step 或配置项。
- **可见产物可打开**：每个阶段需要给人查看的产物使用 `url` 或 `url_list`，并写清业务 `description`。结构化对象可保留审计细节，但不能代替可点击链接。
- **失败是一等路径**：预期失败必须有可见的停留、回流或终止语义；不可恢复失败保持 blocked，不猜测成功。
- **人的决定不可代理**：人工决定使用独立 Condition、互斥结果和绑定本次问题与回复的回执；Agent 不替人批准。
- **身份稳定**：更新已有 Workflow 时同一业务节点保留原 Step ID。Fanloop 没有 Step `replaces`；确需换 ID 时记为删除旧 Step 并新增 Step。删除、拆分、合并、executor 或职责变化按真实增删记录，不得伪装成改名。

## 交付前检查

- 从每条 Route 反向读，能够指出它依赖的 Output、证据来源和生产 Step。
- 从每个 Output 正向读，能够指出它被哪个 Condition 或用户表面使用。
- 从每个阶段正向读，能够指出全景图中可点击的主产物；只存在 `object` 不算可见交付。全景图本身由 Runtime `pending_actions` 投递，不设计 `panorama_presented` 或 `panorama_card_published`。
- 移除任一 Step 后若 Output、Human Gate 或失败边界都不受影响，该 Step 不应存在。
- 只执行当前 Workflow 要求的静态校验。不增加 Requirement 初始化、迁移、普通 Route、Card、Panorama 或最终 Agent 行为试用。
