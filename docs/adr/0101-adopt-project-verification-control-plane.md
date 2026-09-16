---
status: accepted
date: 2026-09-16
amends: ADR-0055, ADR-0099, ADR-0100
supersedes_in_part: ADR-0094
---

# 建立项目级验证控制面、验证技能与功能地图

Fanloop 采用项目级 Verification Skill、行为级 Feature Map 和维护 Skill，使 Agent 能从当前候选启动
隔离环境、驱动真实公开 CLI、观察持久化副作用、保留证据并完成清理。验证能力是开发基础设施，不是
`run-unit` 的别名，也不替代五份 Workflow YAML、Thrift、Contract 或现有测试入口。

## 验证控制面

公开控制面目标为 `fanloop verify` 命令组：

- `verify doctor` 检查候选版本、live 配置、隔离目录与必要工具；
- `verify smoke` 驱动一条完整本地 Requirement 闭环并生成证据；
- `verify snapshot --root` 捕获 Status、State、Output、Event、Trace、Card 与 CLI transcript；
- `verify cleanup --run-id --dry-run` 只清理该运行拥有的临时资源，永远保留证据。

控制面使用运行中的 `os.Executable()` 再次调用公开 CLI，不调用内部 Runtime 充当用户路径证据。每次
运行使用独立 Requirement Root、数据目录和 run ID，清除 Botmux 环境，默认不访问用户身份或真实远端。
证据保存在受限权限目录，包含候选身份、命令、退出码、stdout/stderr、前后快照、副作用、清理状态和
`passed|failed|blocked` 结论。

该公开命令需要新增 Thrift Service、Request/Response 和生成物。具体 field ID、可选性、枚举、错误目录
和叶子 Help 必须在编辑 `idl/` 前按仓库门禁另行人工审核；本 ADR 不构成对未展示 IDL diff 的授权。

## Live Skills 与 Feature Map

验证入口位于 `skills/fanloop-maintainer/fanloop-dev-verify/`，按 Launch、Doctor、Drive、Evidence、
Cleanup 组织。Feature Map 位于其 `references/features/`，以 README 为索引，每个用户能力文件固定使用：

1. `Sub-features`
2. `How to get to it (user POV)`
3. `Driving it with fanloop`
4. `Gotchas`

Feature Map 是便于 Agent 检索的行为记忆，不是第二套契约。稳定依据仍是叶子 `--help`、五份生产 YAML
和真实可观察行为。根目录不恢复 `FEATURE_MAP.md`，`tests/capabilities.json` 继续只承担历史测试迁移归属。

维护入口位于 `skills/fanloop-maintainer/fanloop-dev-maintain-verification/`。维护以 Feature 为覆盖单位：
并行完成只读源码审计，由一个协调者逐项真实驱动，区分 doc drift、harness gap 与 product gap，结果只能
是 `clean`、`changed` 或 `blocked`。维护只修验证资产；需要修改公共 CLI、IDL、Workflow 或产品行为的
gap 返回正常研发流程，不用文档修订掩盖产品问题。

## Maintainer 集成

保留 ADR-0100 的 3 Stage / 3 Job / 9 Step，不恢复 ADR-0091 的验证维护、Feature Map、Eval 或 CI 独立
Step。后续只在现有 `implement_code`、`review_code` 与 `execute_agent_acceptance` 的 Prompt/SkillBinding
中接入维护和验证 Skill，不新增或改变 Step、Route、Condition、顺序或 executor。任何 `prompt.yaml`
修改仍需先展示真实生产 YAML 前后片段并获得单独人工批准。

ADR-0055 的两个仓库测试入口保持不变。`run-unit` 校验验证资产结构与契约；`run-e2e` 后续在现有入口内
增加 verify smoke shard，而不是增加第三个 `tests/run-*`。公共命令和 IDL 落地时按 e2e 档验证。

用户在 2026-09-16 提供完整改造目标，明确要求按 pstack 文章建立验证控制面、Verification Skill、
Feature Map、维护循环、开发流程接入以及后续自动维护与隔离并发能力。
