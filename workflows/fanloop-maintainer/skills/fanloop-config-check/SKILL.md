---
name: fanloop-config-check
description: 检查已修改 Workflow/Skill 的配置有效性并记录候选提交；仅在 Maintainer Certify 的配置 Route 使用，不执行代码验收。
---

# 检查配置

逐项对照 `configuration_scope` 与实际 diff，检查目标是否满足、引用和命令是否正确、代码或未授权文件是否混入。发现范围变化回 Define；配置缺陷用 `configuration_needs_fix` 回 Build。

Markdown/指令改动只检查相关引用、命令帮助和 diff。YAML 或绑定改动再用已有兼容 CLI 在隔离配置副本上加载/Doctor，并核对受影响 Route；只在必要时用最小场景检查行为。不为修改指引而创建真实业务数据。

涉及路由或绑定时，检查 Define 分类可被后续步骤读取，配置 Route 只绑定配置业务 Skills，代码 Route 的原业务 Skills 与条件保持有效；两类互斥，全部 Route 保留所需全景图 Condition。

复核通过后提交配置 diff。`configuration_checked` 记录精确 HEAD/tree、检查命令与真实结果、待同步文件的 live 基线摘要。无需 Go 全量测试、Golden 更新、代码 CR/黑盒、构建、Release 或代码验证 manifest；也不把未运行项写成通过。
