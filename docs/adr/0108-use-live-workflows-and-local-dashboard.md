---
status: accepted
date: 2026-09-21
amends: ADR-0019, ADR-0024, ADR-0026, ADR-0032, ADR-0041, ADR-0054, ADR-0058, ADR-0062, ADR-0079, ADR-0092, ADR-0095, ADR-0099, ADR-0105, ADR-0107
---
**Status / Supersession:** suite-count /「三套」/「十五份 YAML」inventory claims are superseded by [ADR-0110](./0110-record-four-production-workflow-suites.md) for inventory only; Step topology of pre-existing workflows is unchanged by that ADR.


# 使用 Live Workflow 与本地 Dashboard

Workflow YAML 与其绑定 Skills 统一放在 `$FANLOOP_DATA_HOME/current/workflows/<workflow-id>/`。每个目录
包含五份 YAML 和 `skills/<skill-id>/SKILL.md`；Fanloop 每次 CLI 调用只按持久 Workflow ID 从同一个
live 根重新加载两者。Workflow 与绑定 Skills 不再具有 Release 版本或 digest，不进入 Release Manifest，
也没有 packaged fallback、overlay、旧 Schema reader 或迁移路径。

Requirement State、Event、Output Registry 与 Card Projection 继续保存创建它们的 CLI Release version 和
Workflow ID，但删除 Workflow digest。直接切换到 State/Event 14、Output Registry 4、Card Projection 7
与 Release Manifest 4；旧 Storage Schema 由现有 unsupported-schema 错误拒绝。

`$FANLOOP_DATA_HOME/current` 从指向整个 Release 的符号链接改为稳定目录：

```text
current/
├── release -> ../releases/<release-version>
├── bin -> release/bin
├── entrypoints -> release/entrypoints
├── release.json -> release/release.json
└── workflows/
    └── <workflow-id>/{五份 YAML,skills/...}
```

首次安装先校验配置源的完整 Workflow 树，再通过同级临时目录播种 `current/workflows`；后续安装只验证
现有 live tree 并原子替换 `current/release`，不得覆盖、合并或删除 live 字节。识别到旧式
`current -> releases/<version>` 时允许一次性原子切换到新布局，失败必须恢复旧链接；这不是 Runtime
兼容读取。不可变 Release 只包含 CLI、统一 `fanloop-workflow` 入口和 Manifest。

新增 `fanloop dashboard`。它在 `127.0.0.1` 的系统分配端口前台运行，打印 URL 并提供嵌入式静态页面。
服务只列出和读写 live 根中已有的五份 YAML 与 `SKILL.md`：请求路径必须在白名单内且不能经过符号链接；
保存携带上次读取的 SHA-256，stale 写返回 HTTP 409；YAML 在落盘前以完整候选 Bundle 校验，Skill 必须
非空，最终使用同目录临时文件和原子替换。页面每两秒轮询，干净编辑器自动刷新，脏编辑器只标记 stale。
本次不提供远程访问、认证、后台 Daemon、历史回滚、创建、删除、重命名、重排或可视化图编辑。

本决策对历史 ADR 的影响如下：

- 取代 ADR-0019、ADR-0105 中 Workflow digest 绑定、跨 digest 继续与 provenance 部分；保留持久
  Workflow ID、历史 Event 不重判以及当前事实必须与最新 Workflow 兼容的校验。
- 取代 ADR-0024、ADR-0026、ADR-0032、ADR-0041、ADR-0054、ADR-0092、ADR-0095、ADR-0107 中
  Workflow 打包、Manifest Workflow artifact 和整体 `current` 切换部分；保留不可变 CLI Release、
  二进制/入口摘要、staged Doctor、失败回滚与精确 CLI commit 身份。
- 取代 ADR-0058、ADR-0062、ADR-0079、ADR-0099 中 Workflow/Skill 分离目录、Release-bound Skill 路径、
  `config/current` 和 Workflow digest 部分；保留五份 YAML 真值、通用 Runtime、严格目录/绑定校验、
  场景显式选择与每次调用重载 Skill。
- 保留 ADR-0057、ADR-0061 的唯一全局 `fanloop-workflow` 入口及四类 Agent Skill Root 暴露。该入口仍是
  具有 Release version 和摘要的 `SkillArtifact`；它不属于任何 Workflow 的绑定 Skills。

三套生产 Workflow 的十五份 YAML 字节完全不变。共 34 个 Step 的 ID、名称、顺序和 executor 均无新增、
删除、改名、重排或变化。Go 代码只提供 live 根解析、通用加载、验证、执行与 Dashboard 文件边界，不承载
任何生产 Workflow 专用语义。

人工审核记录：主 Agent依据用户对本 Requirement 的完整授权，在 host turn
`01a0c298-5af7-7163-b89b-b9bc3695fb93` 批准了完整需求、未变化的生产 YAML/Step 对比、列举的 Thrift
直接切换与 Dashboard `INVALID_INPUT_JSON` 注解补充；receipt 为
`decision-f3b346ca790472604989e12c343e71c1a29950fe64672dee88c476db4921256a`。

本决策的生产配置存储/发布、Dashboard编辑或代码验证地图位置条款由 [ADR-0120](0120-host-production-workflows-in-fornax.md) 按其明确范围修订；其余门禁继续有效。
