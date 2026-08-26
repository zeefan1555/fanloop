---
status: superseded by ADR-0029
---

# Workflow 使用显式 Route

Fanloop Workflow 用 `completion.to` 明确无环主链，用失败和反馈 Route 明确回流目标、证据来源与失效产物；Phase 只负责展示顺序，CLI 不再通过阶段位置推算 Route 或产物失效范围。每个 Node 统一由 Actions 与完成规则组成，产物使用强类型 Schema，MR Check 豁免也由 Node 以固定 `name_contains` 规则声明。发布前验证全图可达、唯一 Terminal、Route 只能回到当前或祖先。
