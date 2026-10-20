---
status: accepted
date: 2026-09-26
supersedes: ADR-0121
supersedes_in_part: ADR-0049, ADR-0065, ADR-0082, ADR-0084, ADR-0085, ADR-0088
amends: ADR-0057, ADR-0095, ADR-0120
amended_by: ADR-0144-route-skill-maintenance-through-runtime
---

# 使用显式注册中心更新与本地源码分发

Fanloop 只从 GitHub 源码本地构建和安装，不发布 npmjs.org、GitHub Packages、私有 package registry、跨平台归档或其他安装制品。`npm run build` 与 `npm run install:local` 仅是现有 Shell 脚本的快捷入口。GitHub 继续承载源码、PR 与 CI；合并后的唯一安装来源是经过验证的精确 merge commit。

`./scripts/install-local.sh` 只安装 CLI Release，必须离线可用并保留已有 `current/workflows`。全新安装尚无配置时创建真实的空 `current/workflows`，先让 `version` 和 `fanloop update` 可用；重复安装继续保留该 bootstrap 状态。生产 Workflow 与 Skills 继续以工作流注册中心的已发布通道为唯一真值，但只由新的公开 Thrift 命令 `fanloop update` 联网同步。`flow`、`trace`、`card`、`doctor`、`dashboard`、`version` 和本地安装不隐式查询工作流注册中心。

`fanloop update` 读取 `workflow-selection.json` 中明确选择的完整 Workflow 集合，在同文件系统私有 staging 中查询精确已发布版本、下载并检查 ZIP 路径/类型/大小/重复项，复用通用 loader 校验五 YAML、同根 Skills 与完整树。全部成功后才在共享配置锁下原子替换 `current/workflows`，因此远端删除的资源也同步删除。查询、下载、解压、身份或 Bundle 校验任一失败都保留先前可用树，不做部分合并，也不回退源码或其他配置源。

Dashboard 默认查看 live 根且只读。只有显式绝对 `FANLOOP_CONFIG_SOURCE` 指向非 live、非符号链接的隔离候选根时才提供 whole-file 保存。保存要求请求路径已在当前 allowlist、观察 SHA-256 未变化、目标路径无符号链接，并在原子替换前验证整个候选 Bundle。候选页面必须标注非生产；保存不调用工作流注册中心，不能冒充 commit、release、update 或线上回读。

作者工具条款由 [ADR-0144](0144-route-skill-maintenance-through-runtime.md) 修订：仓库的 `.agents/skills/create-fanloop-workflow/` 与 `.agents/skills/edit-fanloop-skills/` 提供对应 Runtime 当前步骤调用的方法。已绑定步骤沿用当前 Root 和作者 workspace，不再启动嵌套维护 Run；只有未绑定步骤且明确要求启动维护 Run 的公共调用才选择对应 Runtime 并按 Status 初始化。create 作者 Skill 与 `fanloop-workflow` 一起进入本地 Release，并由安装器在 Codex、Agent、Trae 与 Claude Skill Root 暴露。

`examples/workflows/example/` 和 `examples/workflows/manage-fanloop-workflow/` 仅作为通用测试夹具，不作为作者模板或生产维护、发布入口，也不进入生产库存、Release、fallback 或默认选择。作者方法使用当前步骤批准的材料和配置源中的完整包结构，不依赖未打包示例。通用 Runtime/Contract 测试保留测试自有 fixture 的独立覆盖；退休的 technical-solution 测试资产不作为生产配置副本保留。

ADR 文件名四位数字只用于排序，不是唯一身份；同号、不同完整文件名的 ADR 可以共存。工具、Skill 与评审不得只因数字重复修改文件名、顺延编号或使候选失效，只处理真实的同路径内容冲突和失效引用。

本决策保留 ADR-0120 的工作流注册中心发布真值、完整包校验与配置/代码分离，修订其“配置消费命令隐式同步”和“Dashboard 永久只读”条款；修订 ADR-0057 的单一全局入口，将独立的 Workflow 作者 Skill 作为第二个公开入口；保留 ADR-0095 的不可变本地 Release、Doctor、原子 current 切换和精确提交身份。它取代 ADR-0121 及更早 ADR 中的 npm/registry/launcher 发布和安装条款，不改变其中仍有效的产品身份、Workflow Runtime、状态、路由与审计边界。
