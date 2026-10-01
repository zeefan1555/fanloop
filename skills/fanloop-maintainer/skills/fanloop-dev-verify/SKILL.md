---
name: fanloop-dev-verify
description: 验证 Fanloop 代码候选时读取源码工作树内唯一验证入口与Feature Map，保留真实公开CLI证据。
---

# 验证入口

实现 Agent读取本次源码工作树的 `.cursor/skills/verify-fanloop/SKILL.md`。无源码上下文的黑盒 Verifier
读取执行 Agent从冻结提交复制并校验摘要的只读副本。两种情况都严格执行该入口的 Launch、Doctor、Drive、
Evidence 和 Cleanup；Feature Map 在同目录 `features/README.md`。Workflow Skill 包不保存代码验证地图副本。

使用精确候选安装与公开CLI，按当前Requirement的 Acceptance Set 和 Feature Impact Set验证。`verify.evidence.outcome` 只取公开CLI写出的schema-1 manifest；禁止手写manifest，OUTPUT_INVALID时按唯一入口重新驱动。不得用文本审阅、旧安装或其他HEAD代替动态证据。入口缺失时保持blocked，不自行推断命令。
