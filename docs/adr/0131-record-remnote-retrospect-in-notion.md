---
status: accepted
date: 2026-10-07
amends: ADR-0127-review-drafts-and-prepare-wiki-concurrently
---

# 复盘经验写入已有的 Notion 表

ADR-0127 把 `retrospect_run` 的零或一条发现放进「共享维护清单」。那一句不再使用。可复用的复盘经验写到已经存在的 Notion 多维表「remnote workflow 经验表」。

- Database ID：`80b2f42e-0e34-4d21-ab84-245bd4404f13`
- Data source ID：`ac9ebbdb-47b5-4f87-b821-df8a3eba5509`
- URL：https://example.invalid/redacted

写入工具是官方 Notion CLI `ntn`。操作者用 `ntn login`。没有钥匙串的机器设置 `NOTION_KEYRING=0`。Integration API key 不是主路径。不增加 Node SDK，不新建维护平台。

`FANLOOP_REMN_EXPERIENCE_DATA_SOURCE_ID` 与 `FANLOOP_REMN_EXPERIENCE_DATABASE_ID` 覆盖上述 ID。默认值写在 `skills/remnote-flashcards/skills/remnote-flashcards/references/notion-experience.md`。五份 Workflow YAML 的 schema 不变，不把这两个 ID 塞进 `workflow.yaml`。

`findings` 仍是 0..1，形状仍是证据、机制、建议改动、验证方法。有发现才 POST `v1/pages`，一行对应一条。零发现不调用 `ntn`。调用失败时 `notion_sync.status=failed` 并写下原因，仍然可以提交 `retrospective_recorded` 并走成功的终止 Route。progress Evidence 照旧追加摘要。复盘仍不修改 Workflow、验收条件或 CLI。

不改 Step `id`、名称、顺序或 `executor`。强制点是 retrospect prompt、学习卡契约和上述参考里的同步句子，以及 `scripts/validate-configuration.py --check-notion-experience-sync`。缺句子或默认 ID 时检查失败。
