---
status: accepted
date: 2026-10-06
amended: 2026-10-10
amends: ADR-0117, ADR-0119, ADR-0126-remnote-bounded-workflows-and-maintenance, ADR-0127-review-drafts-and-prepare-wiki-concurrently
---

# Lightweight RemNote Learning Cards

The user finalized the ordinary RemNote learning path. This amends the card
shape, default Card Cluster rule, and default Wiki scope in ADR-0117,
ADR-0126, and ADR-0127. It does not change the four step IDs, their order,
and their executors. The same-day amendment below replaces the `wiki_ready`
route gate.

Tag each learning batch with the most specific HOME Concept Tag under a domain
grouping, for example `HOME / 03-RESOURCES / 金融知识 / 估值`. Do not stop at a
broad folder when a finer Concept exists. Create that Concept through
`remnote-structure` only when it is missing. For an ongoing tutorial, that
Concept is the tutorial name, for example `人人都是量化交易员`, not the author
name.

Ordinary cards have two backs. An atomic term card remembers one term:
definition, formula, or single-point principle. Its front is `问题 ===`. Its
back is short Feynman text: one formal definition line, one plain-language
analogy, and an optional one-line example. The analogy is one sentence for a
curious child of about 8–10 with no background in the topic: a tangible
everyday scene, no undefined jargon, and a rewrite if the reader has to pause.
The formal definition line owns the terms. An optional example is arithmetic a
child can do, not the formula with numbers substituted. A combo question and
every STAR section must be clear, after one pass, to a curious five-year-old
who can read. If an adult has to pause and decode, rewrite. S is a concrete
scene with characters or objects, not abstract finance, and it names what
looks confusing. T is one sentence that starts with 「你要学会」 and names the
mistake to avoid. A is a worked mini-example with small numbers and worded
arithmetic. R is one memorable 口诀. Learning combos keep all four sections.
Do not use STAR on an atomic card. In one batch, every term gets its own atomic card before
any combo that uses it. A combo question bridges from those cards and asks one
decision. An atomic term card's title is the short name, such as `PE`. Longer search
text, such as `市盈率（PE）是什么？`, is an alias on that same title card.
`getOrCreateAliasWithText` may create that alias. Combo references use the
title card Rem id, not the alias Rem. A `flashcards apply` question or answer
segment uses `{"remId":"<titleCardRemId>"}`. A direct RichText update uses
`{"i":"q","_id":"<titleCardRemId>"}`. The interface shows the blue short
title, and hovering it opens a preview of the Feynman children. The visible
headings stay `S — Situation：背景`, `T — Task：任务／目标`, `A — Action：行动`,
and `R — Result：结果`. A fuller paragraph may sit under each letter.
Chinese-only labels are not a substitute. A combo card
remembers one goal that relates two or more terms: contrast, causal chain, or
decision rule. Its front is `问题 ===`. Its back uses that STAR shape, and the
R section is the required 口诀. Atomic and combo answer lines do not use
arrow symbols such as `→`, `←`, `⇒`, `⇐`, `>>`, or `<<`. RemNote treats those
in an answer line as another nested card. Write arithmetic as words, or with
`=` and `÷` only.
Keep one memory goal per card. Prefer 3–7 cards per article or theme, and
allow 0..N. This replaces the rule that every ordinary card uses STAR.

Entities stay in Daily Notes. A multi-card batch may sit under one ordinary
`text` theme Rem so the batch is readable. When the cards come from one
article or chapter, the theme title is `YYYY-MM-DD <source type> <article title>`,
using the article's own title. A book chapter uses `book` and the H1, for
example `2026-10-06 book 前言`. Do not invent a paraphrase. Before creating a
term card, search with `remnote-cli search` and `--cards-only` for an existing
term flashcard with the same memory target. Reuse that title Rem when it is
good enough. A Wiki concept page is not a term flashcard. Do not create a card
for `spec` or 策略规格说明书 unless the user asks. A methodology article also
gets short-title atomics for named methods, such as `做看疑` and `四个阶段`,
plus STAR combos that connect those methods to terms. If the user says prior
cards for a source are bad, or asks for a redo, delete or replace those cards
and do not reuse them. Otherwise incremental reuse still applies. Only that theme, or the single card
when there is no theme, receives the Concept Tag. Child cards are not
re-tagged. Reject an author name, a coarse Folder used as the only tag, and
a Wiki page used as the tag. The batch records ancestor-path proof.

Card Cluster (`cc`) is optional and is not the default for 2–7 cards. The
ordinary learning path is RemNote Direct: `remnote-cli` talks to the plugin
over the local WebSocket. Help documents `cardCluster`. Omit and `false`
never add built-in `cc`. Only `cardCluster: true` adds it, and only when the
plan also has a theme title and at least two new cards. A theme `title` is
not an opt-in. Do not add `cc` with `object-add-powerup` on this path.
There is no second path for the case where help omits the field. After apply, `object-has-powerup ["cc"]` remains the regression
check. Record it as `flashcards_written.cluster_powerup`. When the user did
not opt in, the result must be false. When the user opted in and the batch
met the title and new-card conditions, the result must be true. An
unexpected true is a remnote-direct regression.

Historical footnote, 2026-10-06: a plan that omitted `cardCluster` still
produced `cc=true` on theme `2026-10-06 book 前言` (`C4lWZBYMbeUIoL9Zd`).
The PE theme `hhA0zbVbEpZbrLoCu` stayed false. That incident is not the
current contract. ADR-0119 still says to add and read `cc` through the CLI
when that request exists; it does not require Card Cluster on every
multi-card theme. This replaces ADR-0117's sentence that multi-card themes
use the built-in Card Cluster.

Amended 2026-10-07. The learning-card rules live in one reference,
`skills/remnote-flashcards/skills/remnote-flashcards/references/learning-card-contract.md`.
Before `--apply`, preparation stores `source_h1`, `theme_title`, and a
`preflight` object. A single article or chapter must use
`YYYY-MM-DD <source type> <source_h1>` with the literal H1; a mismatch
blocks the write. The reviewer receipt lists five yes/no kid-clarity
answers per ordinary card and the four STAR headings verbatim per combo.
`draft_quality_passed` still checks that reviewer receipt. The route gate for
the checklist is `learning_preflight_passed`, restated from this contract.
The 2026-10-07 card-shape amendment does not rename the remnote-direct
scaffold placeholder.

Skip WIKI on the ordinary path unless the user explicitly asks for wiki.
An ordinary finance or concept batch does not create pages or inject wiki
references. Term cards and title-card rem-refs are the default, not a wiki
not-applicable receipt. When the user does ask, the wiki skill may run as an
optional side path. Its page IDs do not satisfy `terms_ready` and are not the
combo rem-ref target. A wiki mutation still needs ADR-0127's zero-card proof.
This replaces the default that every learning batch prepares wiki pages.

Amended later on 2026-10-07. The sentence that kept `wiki_ready` on the route,
and let it pass from a not-applicable receipt, is withdrawn. Prepare advances
only when `preparation_recorded`, `draft_quality_passed`, `terms_ready`, and
`learning_preflight_passed` are all present. `terms_ready` requires one
`search --cards-only` row per term, reuse by title Rem id or create on a miss,
and combo refs that point at those title cards. Persist writes new atomics
first, reads the title Rem ids, then injects `{"remId":"<titleCardRemId>"}`.
`terms_referenced` is that readback. `remnote-wiki` is optional on prepare and
is not executed on the default path. The repair value `wiki_links` is now
`term_refs`. The four step IDs stay: `resolve_home_tag`,
`prepare_cards_and_wiki`, `persist_and_verify`, `retrospect_run`. Display
names describe the term path. Original import and excerpt may pass
`terms_ready` only with `skip_original` and zero term creates.

Original import and original excerpt stay verbatim. Incremental dedupe, daily
note placement, multiline `===` cards, and Concept versus Folder are unchanged.
The dry-run examples are
`skills/remnote-flashcards/skills/remnote-flashcards/references/example-pe-valuation.md`
and
`skills/remnote-flashcards/skills/remnote-flashcards/references/example-book-preface.md`.
Both show a Direct plan that omits `cardCluster`. Neither is evidence of a
live RemNote write.

Amended 2026-10-07 after the remnote-flashcards run. Combo question text and
every STAR section stay on the five-year-old plain-words rule. Unexplained
adult trading slang fails `one_pass` and `plain_words`. The words from that
run are 梭哈, 全仓, 一把梭, and 优秀个股. The plain rewrites from that run are
一次花光 and 分开买. Combo rem-refs use
`{"remId":"<titleCardRemId>"}` on `flashcards apply` question and answer segments, or
`{"i":"q","_id":"<titleCardRemId>"}` via `object-set-text`. Markdown
`[[id:]]` through `create --content-file` is not a write path. That run's
`[[id:gTpsZTEYvBxB4S1PM]]` (分批加仓) was stored as `[[DCF]]0`. The four step
IDs are unchanged.

Amended 2026-10-09. The source-type token in `YYYY-MM-DD <source type> <source_h1>`
must match the Concept tag's PARA area. theme_para maps 00-NOTE to note,
01-PROJECTS to project, 02-AREAS to area, 03-RESOURCES to resource, and
04-ARCHIVE to archive. This replaces the sentence that a book chapter always
uses `book`. The forward preface example under `03-RESOURCES` is
`2026-10-06 resource 前言`. The 2026-10-06 incident string
`2026-10-06 book 前言` (`C4lWZBYMbeUIoL9Zd`) stays as history. A combo card is
the last child of the theme and has no tag of its own. After the untitled
combo apply, dry-run then `--apply`
`move-note <COMBO_ID> --new-parent-rem-id <THEME_ID> --position last --expected-old-parent-rem-id <DAILY_ID>`,
then `object-remove-tag --target-id <COMBO_ID> --args-json '["<TAG_ID>", false]' --allow-destructive`.
Do not leave the combo on the daily note. Aliases are
`sdk rem object-get-or-create-alias-with-text --target-id <ID> --args-json '[["<alias>"]]'`.
`update --add-aliases` fails on SDK 0.0.46 with
`Native transactions are unavailable: SDK 0.0.46 cannot acknowledge abort`.
Forward routes also require `tag_approved`, `control_search_passed`,
`combo_placed`, and `aliases_written`. An empty CJK search is not authoritative:
run a known-term control search first. 已知术语对照搜索返回空时，先点桌面按钮「重新加载搜索缓存」，再重搜。
Also search the English form of the term when it has one. Preview without
`--apply` shows only `validated` and `planHash`; compare the plan file to the
reviewed draft digest before `--apply`. Never run remnote-cli in parallel
(`CLI port occupied`). `sdk date get-todays-doc` is a write. Reviewer fallback:
parent or host when the runner cannot spawn a sub-agent, otherwise
`reviewer=user`. Re-review only changed cards against a recomputed digest.
Receipts record `plugin_commit`. example numbers state their condition, such as when idle; do not use a platform term as an everyday word, such as 小程序 on a WeChat card; a binary question's answer line states the choice.
Runtime still checks types and routing. The four step IDs are unchanged.

Amended 2026-10-10. `flashcards apply` writes combo fronts and answers from
`TextSegments`; a plain-string question containing `[[` or `]]` is invalid.
Before apply and after readback, a structured scan records
`no_literal_brackets=true` only when the question, answers, `backText`, and
aliases contain no literal bracket reference. Each combo's expected reference IDs
come from `preparation_receipt.combo_rem_refs`, with new `term_key` values resolved
through `flashcards_written.ids`. That expected set must be a non-empty subset of
the batch title-card IDs, and the actual front-plus-answer set must equal it. `persist_and_verify`
also requires `real_app_verified`. The verifier reads the current
`knowledgeBaseId` from `remnote-cli status`, checks front and answer references
plus hover previews in the web app, and records an explicit `no_combo` result
when the batch has no combo cards. The four step IDs, names, order, and executors
remain unchanged. This preserves ADR-0126's requirement that the workflow stay
independent of a particular knowledge base.
