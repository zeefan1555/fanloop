---
name: update-fanloop-workflow
description: 修改个人 Fornax 空间里已发布的 Fanloop Workflow，发布后用 fanloop update config 同步本地。只写 read_write 源，不改只读源。
---

# 更新 Fanloop Workflow

从已发布的 Fornax 包开始改，再发布回同一个 Skill，然后拉取本地。Git 仓库不参与这条路径。`tools/workflow-package build` 只用于恢复，不用于日常更新。

## 读取基线

1. 确认目标 Workflow ID 和要改的内容。读 `fanloop update config --help` 和 `fanloop doctor`。
2. 读取 `${FANLOOP_DATA_HOME:-$HOME/.fanloop}/fornax.json`。只使用 `access` 为 `read_write` 的那一项的 `workspace_id` 和 `key_prefix`。没有唯一的 `read_write` 源就停止。不要把 workspace id 写进仓库、日志或证据。
3. 使用用户已经登录的 `fornax-cli`。输出里不出现 AK/SK、JWT、签名或下载 token。
4. 用 `skill list --keyword <key_prefix>` 找到 key。key 的后缀不能代替 Workflow ID，用包内 `workflow.yaml` 的 `id` 确认目标。用 `skill list-releases --skill-id <ID> --env Online --feature prod` 取最新 `released` 版本。
5. 下载到一个新的临时目录，不安装到个人 Skills 目录。

```bash
fornax-cli skill install \
  --workspace-id <READ_WRITE_WORKSPACE_ID> \
  --skill-query-json '[{"skill_key":"<EXACT_KEY>","version":"<RELEASED_VERSION>"}]' \
  --dir <EMPTY_TEMP_DIRECTORY>
```

包根是 `fanloop-workflow-<id>/` 或 `loop-workflow-<id>/`，里面有包装 `SKILL.md` 和唯一 `<workflow-id>/`。目录名必须等于 `workflow.yaml` 的 `id`。

## 修改

在副本里改五份 YAML、`skills/**` 或说明。保留原始副本做 diff。改到可执行脚本的行为、IDL、Schema 或 CLI 时停止，这条路径只改配置。

展示完整 diff。YAML 的 Step `id`、`name`、顺序或 `executor` 有变化时单独标出。不要声明 Runtime 保留的 `panorama_presented` 或 `panorama_card_published`。

## 发布并同步

1. 再读一次最新 Online/prod released。基线变了就重新取基线再应用改动。
2. 用原来的包装 `SKILL.md` 和唯一 Workflow 目录打包。`skill commit` 和 `skill release` 都带上 `read_write` 源的 workspace。结果不明就先回读，不重复写入。
3. 确认目标版本已 `released`，再下载该版本并比较已审核文件的字节。
4. 运行 `fanloop update config`。它会替换整个已发布集合。需要留下的本地编辑先抄到临时目录。
5. 回读 `config/current/fanloop-workflow/<workflow-id>/`，比较字节并运行 `fanloop doctor`。

发布成功但本地同步失败时，说明远端已发布、本地仍是旧快照。修好之后只重试 `fanloop update config`。

只读源里的包不能用这个 Skill 发布。要写回 Git 仓库时，用 `scripts/mirror-fornax-to-git <snapshot-dir> <checkout-dir>`。
