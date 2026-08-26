---
status: accepted
date: 2026-08-19
amends: ADR-0026, ADR-0037
amended_by: ADR-0058, ADR-0061
---

# 只全局暴露 Workflow Skill，并返回 Release-bound 原子 Skill 路径

Fanloop 的完整 Release 继续包含 Manifest 声明的全部 Skills，但每个已配置的 Codex、
Agent、Trae Skill Root 只全局暴露 `fanloop-workflow`。安装、切换和 Doctor 不再把
“Release 内完整交付的 Skills”与“全局发现入口”视为同一集合。`fanloop-workflow`
仍按 ADR-0026 通过 `current` 软链随整包原子切换；用户拥有的普通路径继续拒绝覆盖。

Flow 公共 `Skill` 视图新增 required `path`。Runtime 使用受校验的 Release Manifest
Skill path，把每个 YAML Skill ID 投影为当前运行二进制所在不可变 Release 中对应
Manifest Artifact 的绝对 `SKILL.md` 路径。Artifact 可以位于
`skills/fanloop-workflow/{common,<workflow-id>}/<id>` 或
`skills/self-iteration/<id>`。路径不进入 Workflow YAML 或
durable State/Event，不经过全局 Skill 搜索，也不回退到 `current` 或外部 Skill。
缺失匹配的 packaged Skill 是当前 Release 不完整，应返回内部错误。

升级直接退役旧版创建的原子 Skill 全局软链，但只删除仍精确指向 Fanloop
当前 Manifest 记录的 `current/<artifact-path>` 的软链；同时兼容清理旧扁平版本的
`current/skills/<name>` 软链。文件、目录、已改变的链接、外部链接及链接目标均不修改。
Doctor 继续摘要校验全部 packaged Skills，只对唯一全局入口执行 link check。Version
继续报告完整 packaged Skill inventory。

本决策修订 ADR-0026 中“为全部官方 Skills 创建三端软链”的范围，并修订 ADR-0037 的
Status Skill 视图。它保留 ADR-0024 的完整 Release 与只读 Doctor、ADR-0032 的默认数据
目录、ADR-0041 的事务修复和用户路径保护、ADR-0040/0042 的 Thrift-first 生成边界、
ADR-0054 的 Release Manifest 真值以及 ADR-0055 的双测试入口。五份生产 Workflow YAML、
Step 契约、Condition、Route、Prompt Skill 绑定和 release manifest schema 均不改变。
具体 Thrift 人审记录、影响和验收条件见
`docs/specs/single-workflow-skill-path.md`。
