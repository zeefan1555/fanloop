---
status: accepted
date: 2026-08-19
amends: ADR-0057
amended_by: ADR-0061
---

# 分组发布 Workflow 与自迭代 Skill

Fanloop Release 使用两个固定 Skill 根：

```text
skills/
├── fanloop-workflow/
│   ├── common/<skill-id>/
│   └── <workflow-id>/<skill-id>/
└── self-iteration/<skill-id>/
```

`fanloop-workflow/common` 保存多个 Workflow 共享的原子 Skill；
`fanloop-workflow/<workflow-id>` 保存仅该 Workflow 使用的业务 Skill。二者都进入
Release Manifest。目录分组不改变 Skill ID，也不引入同名覆盖；所有 Artifact 名称在
Release 内全局唯一。按照 ADR-0057，只有名为 `fanloop-workflow` 的 Artifact 链接到
Codex、Agent 和 Trae 的用户 Skill 根目录；其他原子 Skill 均通过 Flow 返回的
Release-bound `path` 使用。

`self-iteration` 保存 `fanloop_cli` 自维护专用 Skill。它们同样作为独立 Artifact 进入
Manifest、归档摘要、Release 校验和 Doctor 内容检查，但安装器不为它们创建用户全局链接。
维护 Workflow 与普通业务 Workflow 一样，从 Flow 返回的配套 Release 确定路径使用原子
Skill，普通用户不能把它们当作全局 Skill 调用。

每套 Workflow 的 `prompt.yaml` 继续以现有 Skill ID 声明逐 Step 依赖。发布门禁把 ID
解析到唯一 Manifest Artifact：普通 Workflow 只允许引用 `common` 或与自身 Workflow ID
同名目录，`fanloop-maintainer` 允许引用 `common` 与 `self-iteration`。因此新增部门可以
复用通用 Skill，也可以添加自己的目录，而 Workflow YAML 不需要知道 Release 绝对路径。

Manifest 继续使用 `SkillArtifact.path` 保存完整相对路径。全局暴露行为由固定
`fanloop-workflow` Skill 名称决定，不增加 install flag、Skill Group IDL、插件系统或
Runtime resolver。发布器递归发现固定
形状的 `SKILL.md`，拒绝非法深度、重复 Skill ID 和跨 Workflow 目录引用。

本决策保留 ADR-0006 的 Skill/Workflow/Runtime 职责分离、ADR-0009 和 ADR-0024 的
完整 Release 边界、ADR-0026 的三客户端链接、ADR-0028 的 Skill 内部布局自治以及
ADR-0052/0054 的 Thrift 真值，并修订 ADR-0057 中扁平 `skills/<id>` 的路径示例为上述分组
路径。它改变 Release 内 Skill 的物理组织，但不改变 Release Manifest 字段、Workflow
YAML Schema 或公开 CLI。
