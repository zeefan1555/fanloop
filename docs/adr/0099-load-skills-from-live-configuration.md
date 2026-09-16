---
status: accepted
date: 2026-09-16
supersedes_in_part: ADR-0024, ADR-0057, ADR-0058, ADR-0080, ADR-0095, ADR-0096, ADR-0097
---

# 从独立 live 配置读取原子 Skills

`skills/` 与 `exemplars/` 是源码仓库中的 live 配置，不再复制进本地 Release，也不再进入 Release
Manifest 或 CLI dirty 版本摘要。Release 只携带二进制、五文件 Workflow Bundle 和统一入口
`entrypoints/fanloop-workflow`；因此只修改 Skill、范文或图片不需要重建或重装 CLI。

`./scripts/install-local.sh` 在完成 Release 安装时原子维护
`~/.fanloop/config/current -> <源码仓库>`。默认使用执行安装脚本的仓库根；隔离安装可通过绝对路径
`FANLOOP_CONFIG_SOURCE` 指定另一份配置仓库。运行时默认从
`$FANLOOP_DATA_HOME/config/current` 读取配置，测试或隔离调用可用绝对路径
`FANLOOP_CONFIG_ROOT` 覆盖。

每次生成 `flow init/status/progress/result` 的当前视图时，Runtime 都重新解析
`<config-root>/skills/<workflow-id>/<skill-id>/SKILL.md`，并返回解析后的真实绝对路径。Agent 必须在
每次最新 Status 后读取该路径，不缓存旧 Skill。Skill 内容变化可以影响正在运行的 Requirement；
Workflow ID、Step、Condition、Route、Prompt/SkillBinding、Output 和状态失效语义仍由初始化时绑定的
Workflow digest 决定。

构建和安装使用同一个配置校验器，要求 Workflow 与 Skill 分组一一对应、Skill ID 全局唯一、每个
直接子目录包含普通 `SKILL.md`，且所有绑定都存在于所属 Workflow 组。Doctor 增加独立
`skill_config` 检查；配置缺失、损坏或与内置 Workflow 不匹配时失败关闭，不回退到任何 Release 副本。
统一入口仍由 Manifest 摘要校验并通过四个客户端 Skill Root 暴露。

旧 Release 继续按其既有 packaged Skill 语义运行；本变更不提供新 CLI 对旧 Manifest 的迁移或
fallback。新 Release 的 Manifest schema 仍为 3，`skills` 列表只包含统一入口，因此无需修改
Thrift IDL。五份生产 Workflow YAML、State/Event/Output Schema 和公开 CLI Request/Response 均不变。

本变更使用 `e2e` 验证档：证明构建目录不含 `skills/`/`exemplars/`，仅修改 live Skill 不改变 CLI
版本，安装后修改 Skill 可被同一 CLI 的下一次 Status 立即读取，配置损坏会被 Doctor 阻断，并运行
`./tests/run-unit`、`./tests/run-e2e` 与安装后公开 CLI 验证。

人工确认记录：用户于 2026-09-16 明确要求“skill 文件夹只作为配置，cli 每次拉这里的内容，而不是
每次修改 skill 也要重新构建 cli 版本”。
