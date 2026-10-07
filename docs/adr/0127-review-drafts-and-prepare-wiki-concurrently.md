---
status: accepted
date: 2026-10-03
amends: ADR-0117, ADR-0126-remnote-bounded-workflows-and-maintenance
---

# Review Drafts And Prepare Wiki Concurrently

ADR-0129 replaces the default Wiki worker and the `wiki_ready` join for ordinary
learning batches. Term search or create, then title-card rem-refs, is the path
that leaves prepare. Concurrent draft review, the four step IDs, batch limits,
and the retrospective rules below remain except where ADR-0129 states otherwise.

The approved speed plan replaces the ordering and review policy of ADR-0117 and
ADR-0126. Keep one Stage, one Job and four Agent steps:
`resolve_home_tag -> prepare_cards_and_wiki -> persist_and_verify -> retrospect_run`.
Remove the old write-first routes; no compatibility branch or engine change.

Reuse source material and resolve learning goals, classification, duplicates and
terms once. A separate read-only reviewer checks each draft batch immediately.
Workers share source references and temporary material, not persistent knowledge
copies. CLI connections remain serialized. ADR-0129 replaces the Wiki worker and
the `wiki_ready` join: ordinary batches gate on `terms_ready` and
`learning_preflight_passed` instead. YAML still enforces the conjunction; Agents
must validate the evidence, since the existing engine does not certify its
semantic truth. The wiki worker, real page IDs, and zero-card proof below apply
only when the user explicitly asks for a wiki mutation.

New terminology pages contain a clear definition, necessary example, sources and
relevant references. Do not require background, debates or open questions, or
delete existing content. Default scope is terms needed for the learning goals;
an explicit full-material terminology request includes all applicable terms.
Ordinary words and temporary values do not become pages.

On an explicit wiki request, inject real Wiki IDs before read-only preview and
card persistence. Ordinary batches inject title-card Rem IDs instead. Preserve
ordinary STAR and original-text modes, 0..N goals, native IDs/history, placement,
7-new-card/10-page/100-KiB limits, unknown-write reconciliation and three-round
failure stop. Batch failures repair only failed objects, never recreate the batch.
Compare final text to the reviewed draft; changed text requires renewed review.

Direct CLI batch Wiki verification covers affected objects, not the whole Wiki.
`verificationScope: affected_objects` must never count as final whole-Wiki proof.
After the last Wiki mutation, run the existing full-root `verify scope` zero-card
check once; relevant later mutation invalidates it. Pure reuse and answer-only
link changes check affected pages/answers/references. Summary output is a projection,
not permission to omit checks or alter failure exit codes. CLI implementation and
native performance acceptance belong to a separate Direct maintenance candidate.

The last step records measured phases, unknown time and zero or one generic
improvement in existing run records. Suggestions use evidence, repeated mechanism,
proposed change and verification method. Queue them in the shared maintenance list;
do not edit rules, upgrade tools or create Wiki/card content during retrospective.
No speculative cache, new service, plugin or Fanloop engine changes.

Acceptance requires complete bundle loading, affected route matrix and fresh
independent behavior trial. Synthetic routing receipts prove orchestration only;
native persistence/recovery and fixed-workload timing need separate CLI evidence.
No unmeasured latency claim, waived quality gate or discarded user content.
