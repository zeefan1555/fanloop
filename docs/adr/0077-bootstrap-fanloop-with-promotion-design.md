---
status: accepted
date: 2026-08-26
supersedes_in_part: ADR-0009, ADR-0027, ADR-0053, ADR-0059, ADR-0062, ADR-0065
---

# 从固定 Treeloop 快照直接建立 Fanloop

Fanloop 以源仓库 commit `a5c64d81151a86810d0303d37e535995db3d5ad6` 为代码基线，
采用 `fanloop` 命令、`github.com/zeefan1555/fanloop` Go module、`fanloop-cli` npm 包、
`~/.fanloop` 数据目录、`.fanloop` Requirement 状态和 `FANLOOP_*` 环境变量。切换不提供
旧命令、目录、环境变量、Skill 或 Workflow ID 的兼容别名、双读写或迁移逻辑。

当前 Release 只携带两套五文件 Workflow Bundle：

- `promotion-design` 是新 Requirement 的默认 Workflow，包含需求澄清、需求人工确认、方案
  写作、陌生评委审校和方案人工确认五个 Step；
- `fanloop-maintainer` 保留原维护流程的八个 Step、Condition 和 Route 拓扑，自迭代 Skill
  统一使用 `fanloop-dev-*` ID。

删除原默认研发 Workflow 与部门 Workflow 及其不再被引用的业务 Skill。Selector 只保留
“显式选择优先，否则 `promotion-design`”规则，不再包含仓库或部门路由。五文件 Bundle、
normalized digest、Release-bound Skill path 和运行中 Requirement 固定绑定等未冲突契约继续有效。
旧默认流程专属的 action-driver、Stage 状态机、CLI/Trace/Card 联动、Route 与跨域 Output Guard
回归组随之退役；它们不得映射到 `promotion-design` 的窄测试来伪装为保留能力。

`promotion-design` 复用同一个结构化 Skill 完成澄清、写作与审校，不为每个 Step 创建重复
Skill。`.promotion/brainstorm.md`、`evidence.md`、`require_points.md`、`方案.md` 和
`.promotion/review.md` 是领域产物；不得创建 `.promotion/state.json`。流程推进、人工门禁、
Output 修订和回流失效只由 `.fanloop/flow/state.json` 负责。

仓库当前只初始化本地 Git `main` 分支，不创建、推送或配置远端。npm 默认源改为公开 npm，
本地 Release 构建不依赖已发布包。代码仍为 `UNLICENSED`；创建公开 GitHub 仓库、选择许可证
以及公开前的内部资料审计是后续独立决策，本次不授权发布。

本决策取代 ADR-0059 的旧默认值、部门路由和已删除 Workflow 清单，取代 ADR-0027 与
ADR-0009 中 Codebase、字节 npm 和内部发布渠道的要求，并修订 ADR-0053/0065 的新 Requirement
初始化前强制在线更新：源码或本地安装直接使用当前配套 Release。ADR-0062 的每 ID 一套当前
Bundle、ADR-0006 的职责边界、ADR-0032 的单一数据目录、ADR-0036 的五文件架构真值、
ADR-0057/0058 的 Release-bound Skill 分组以及未冲突的本地验证门禁继续有效。
