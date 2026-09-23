---
name: fanloop-config-check
description: 检查配置有效性，并对行为变更派发全新 Sub Agent 实际试用、核验原始证据；仅在 Maintainer Certify 的配置 Route 使用，不执行代码验收。
---

# 检查配置

逐项对照 `configuration_scope` 与实际 diff，检查目标是否满足、引用和命令是否正确、代码或未授权文件是否混入。发现范围变化回 Define；配置缺陷用 `configuration_needs_fix` 回 Build。

Markdown/指令改动只检查相关引用、命令帮助和 diff。YAML 或绑定改动再用已有兼容 CLI 在隔离配置副本上加载/Doctor，并核对受影响 Route。不为修改指引而创建真实业务数据。

涉及路由或绑定时，检查 Define 分类可被后续步骤读取，配置 Route 只绑定配置业务 Skills，代码 Route 的原业务 Skills 与条件保持有效；两类互斥，全部 Route 保留所需全景图 Condition。

## 独立试用

1. 复核 Define 的 `trial_plan` 与最终 diff 一致。执行规则、命令、路由、绑定、判断或完成标准有变化时必须试用；计划漏项先补齐用例。纯错别字/排版且行为未变才可免测，记录 `trial.outcome=not_applicable`、原因及对应 diff，不冒充 passed。
2. 基础检查通过后固定待测文件及摘要，准备已有兼容 CLI、隔离配置/临时仓库、专用数据与全新 Requirement Root。无需构建或安装候选 binary。目标环境不可达、授权不足或缺少可安全使用的数据时保持 blocked，不以文本审查代替试用。
3. 派发一名不继承本轮对话或修改上下文的全新 Sub Agent（支持时使用 `fork_turns=none`）。只给自然语言任务、验收标准、候选 Workflow/Skill 的实际读取入口、必要工具/测试数据，以及允许操作与停止边界。允许读取待使用的 Skill/配置；不提供实现 diff、修改说明、旧报告或引导它选择正确路线的答案。
4. 子 Agent 必须真正执行场景，保存所用 Skill 路径、命令 argv/stdout/stderr/退出码、实际文件变化及关键状态回读；不能只审阅文字或复述计划。可修改测试副本，不能改待测候选、真实笔记或全局 live，不能 push、创建/合并真实 PR。自迭代流程的试用按场景约定停止，避免递归启动无界的验收；到达尚未授权的后续步骤应报告停止位置，不伪造完成。
5. 回读原始产物逐项判断成功标准，不能仅接受子 Agent 的“通过”结论；确认候选摘要未变化，且没有越过允许副作用。全部目标满足才记 `trial.outcome=passed`。行为错误用 `configuration_needs_fix` 回 Build，修复后重新派发全新试用；环境/权限阻塞保持 Certify blocked。不得复用候选变更前的回执。

## 完成

基础检查和适用的独立试用都满足后提交配置 diff。`configuration_checked` 记录精确 HEAD/tree、配置文件摘要、检查命令与结果、待同步 live 基线，以及 `trial` 的 outcome、理由、场景覆盖、Sub Agent 标识和原始回执/证据路径。免测时不填写虚构 Agent 或回执；failed/blocked 不能提交成功 Condition。

不执行 Go 全量测试、Golden 更新、代码 CR/黑盒、构建、Release 或代码验证 manifest。配置行为试用只覆盖本次变化；证据保留在 Requirement Root 或仓库外，临时测试对象按授权范围清理后仍保留回执。
