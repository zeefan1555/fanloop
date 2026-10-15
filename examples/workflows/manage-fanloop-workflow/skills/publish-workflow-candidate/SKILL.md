---
name: publish-workflow-candidate
description: 在 manage-fanloop-workflow 的发布节点中幂等发布已批准候选到个人 Fornax 空间，并同步和回读本地 live 配置。
---

# 发布 Workflow 候选

只发布人工明确批准的精确候选。所有 Fornax 请求使用 CN region 和个人 workspace
`7543941556223377409`。key 前缀只使用 `fanloop.workflow.`。不得读取或写入团队空间
`7590123861772503042`。只使用当前用户已登录的 `fornax-cli`。AK/SK、JWT、签名和下载 token 不进入文件、命令参数、日志、聊天或报告，也不写入 `fornax.json`。

发布前重新查询 exact key、Skill ID、全部 commits 和最新 Online/prod released 记录。基线与审核时不同，
返回 `publish_baseline_changed`，不得覆盖、另建 key 或复用旧批准。

创建时使用 `fanloop.workflow.<suffix>` 新 key；`<suffix>` 是目标 `workflow.yaml.id` 中 `-` 换成 `_`。
更新时沿用原 Skill ID。本 Workflow 自身的 key 是 `fanloop.workflow.manage_fanloop_workflow`。
上传 zip 根只包含根 `SKILL.md` 和唯一 Workflow 目录。读取当前 `fornax-cli skill create|commit|release` 叶子 Help 后执行写入。版本在该 Skill
自己的最新 commit 上递增；Agent Buddy 和 Fornax 版本互不绑定。

只读回读使用：

```bash
fornax-cli skill list --keyword fanloop.workflow.<suffix> --page-no 1 --page-size 100 --custom-region CN --workspace-id 7543941556223377409 --format json --output <dir>
fornax-cli skill list-releases --skill-id <SKILL_ID> --env Online --feature prod --custom-region CN --workspace-id 7543941556223377409 --format json --output <dir>
fornax-cli skill install --custom-region CN --workspace-id 7543941556223377409 --skill-query-json '[{"skill_key":"fanloop.workflow.<suffix>","version":"<version>"}]' --dir <dir>
```

create、commit 或 release 结果不明时先按 key、Skill ID、版本和 release 记录回读，不盲目重试。只有同一
Skill ID、目标版本、Online/prod 状态为 released 才返回 `fornax_release_published`。

若 `${FANLOOP_DATA_HOME:-$HOME/.fanloop}/fornax.json` 的 `read_write` 源不是 workspace `7543941556223377409` 加前缀 `fanloop.workflow.`，停止发布并报告该差异。不要改写到其他空间，也不要写入凭证。

发布成功后使用本次被创建或修改的目标 Workflow 的真实 workspace、目标 Skill ID 和目标 Workflow ID 生成
配置中心链接：

```text
https://fornax.bytedance.net/space/7543941556223377409/components/skill/<SKILL_ID>/file/<WORKFLOW_ID>/workflow.yaml
```

`<SKILL_ID>` 必须来自目标 Workflow 本次 Fornax 发布回执，`<WORKFLOW_ID>` 必须来自目标候选的
`workflow.yaml.id`。不得误用 `manage-fanloop-workflow` 管理流程自身的 Skill ID 或 Workflow ID；只有本次目标
本身就是 `manage-fanloop-workflow` 时两者才相同。

不得只返回 Fornax 首页、Skill 列表页或猜测路径。把目标链接写入飞书文档的发布章节、最终 HTML 可视化和
用户可见回复，并返回 `fornax_config_center_link_ready`。没有目标 Workflow 的可点击链接时发布 Step 不得终止。

同时生成目标 Workflow 的版本 Diff 链接。先用审核基线版本和本次 released 版本取得完整文件 Diff，并写入
同一飞书文档的“候选与完整 Diff”及“Fornax 发布与本地同步”章节；最终 HTML 也必须展示版本号、改动文件和
完整 Diff。Fornax 当前版本对比是页面内弹窗，没有独立可分享路由时，`workflow_version_diff_url` 返回这份
可直接打开并看到完整 Diff 的飞书文档 URL；不得拼接或猜测不存在的 Fornax Diff 深链。若平台以后返回真实
可分享 Diff URL，则优先使用并同时写入飞书文档和 HTML。

最终用户可见回复必须分别给出“目标 Workflow 配置地址”和“版本 Diff 地址”。Diff 未写入文档、HTML 未展示
或链接打不开时，不得返回 `workflow_version_diff_link_ready`，发布 Step 不得终止。

下载该精确版本，剥离 Fornax wrapper 和根包装元数据后，与审核候选逐字节比较。相同才返回
`published_bundle_matched`。

`fanloop update config` 只从配置仓库同步 Workflow 包，不从 Fornax 下载，也不升级 CLI。确认用户需要保留的本地未发布编辑已另存后执行它。回读
`${FANLOOP_DATA_HOME:-$HOME/.fanloop}/config/current/fanloop-workflow/<workflow-id>/`。
只有该 live 目录与已发布候选逐字节一致时才返回 `local_config_synchronized`。候选尚未进入该配置快照时，不得把同步说成成功。
随后运行 `fanloop doctor`，没有 unhealthy 才返回 `local_doctor_healthy`。再用 `fanloop config drift` 对照个人 Fornax release；`in_sync` 才算镜像对齐。远端成功而本地失败时保留远端事实，只重试同步和回读。

把 Workflow ID、Fornax key/Skill ID/版本、release 记录、下载比对、本地同步、Doctor、静态验证边界和
用户更新提示全部追加到 `workflow_document_url` 指向的同一飞书文档，不新建第二份交付文档。

同时生成：

> 你是【角色】，使用 fanloop 中的【Workflow ID】Workflow。

机器人机器上的 `fanloop update config` 只验证本次发布，不会更新其他人的电脑。`fanloop update` 只切换 CLI Release，不会安装 Workflow。每次创建或更新成功后，必须
在最终回复中单独展示以下提示，不能只放在证据或审核包中：

> Workflow 已发布。请在每台实际使用该 Workflow 的机器上执行 `fanloop update config`，更新完成后再启动或继续使用。

返回 `user_local_update_instruction_ready` 时，Output 必须包含上述完整提示。没有明确展示该提示时，发布
Step 不得终止。

发布前回读人工审核回执与主审核 HTML 的锁定摘要。最终 HTML 必须保留原用户要求、改前/改后、影响边界、
风险和已批准候选内容，只追加分析/构建/审核/发布四节点最终状态、Fornax 版本、配置地址、版本 Diff 地址、
Doctor 结果和用户下一步。任何已批准内容变化都使批准失效并阻止发布。

交给 `manage-workflow-document` 更新同一 HTML5 Block。回读文档确认 block 存在且已批准内容摘要未变化后才返回
`delivery_visualization_ready`，对象包含 `document_url`、`block_id`、`revision_id` 和内容摘要。

全部完成后返回 `publish_baseline_revalidated`、`fornax_release_published`、`published_bundle_matched`、
`fornax_config_center_link_ready`、`workflow_version_diff_link_ready`、`local_config_synchronized`、
`local_doctor_healthy`、`usage_prompt_ready`、`user_local_update_instruction_ready` 和
`delivery_visualization_ready`。
