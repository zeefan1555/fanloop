---
name: remnote-flashcards
description: Create and verify RemNote flashcards from understood concepts.
---

# RemNote Flashcards

This directory is one complete Workflow Skill package. The five YAML files beside
this entry define its execution graph; bound Skills and resources live in `skills/`.

Select `remnote-flashcards` through the public `fanloop-workflow` entry, initialize
with `fanloop flow init --workflow remnote-flashcards`, and follow the current
Status and its exact required Skill paths. Preserve all business Human Gates.

`retrospect_run` writes a durable finding into the existing Notion database
「remnote workflow 经验表」 with the official `ntn` CLI. Read
`skills/remnote-flashcards/references/notion-experience.md` before the POST.
Zero findings do not call `ntn`. A failed call records `notion_sync` and still
allows `retrospective_recorded`.

FANLOOP_REMN_EXPERIENCE_DATA_SOURCE_ID overrides the data source id; FANLOOP_REMN_EXPERIENCE_DATABASE_ID overrides the database id. Defaults live in references/notion-experience.md.

Auth is ntn login; headless hosts set NOTION_KEYRING=0. Do not require an integration API key.

Map 经验 to the one-line lesson, 日期 to the ISO run date, 主题 to the RemNote theme title, 类别 to 文案清晰度|工具链路|流程门禁|其他, 详情 to evidence + mechanism + suggested change + verification, and 编码状态 to 待编码 unless the finding already points at a merged fix (已编码) or is a non-code lesson (不适用).

When findings is non-empty, POST one page per finding with ntn api v1/pages to data source 「remnote workflow 经验表」; zero findings skip the write; a failed ntn call records notion_sync and still allows retrospective_recorded.
