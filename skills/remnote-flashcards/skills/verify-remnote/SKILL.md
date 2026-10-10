---
name: verify-remnote
description: Drive the real RemNote web app (box Chrome, signed in) to prove that flashcards written by remnote-cli render correctly - blue native refs on combo fronts and answers, hover previews showing title-card backs, term card aliases, theme under daily note with its Tag. Use in persist_and_verify for real_app_verified, and whenever a user reports a ref that "did not show".
---

# Verify RemNote (real app)

CLI read-back proves stored RichText, not what the user sees. Open the cards in the real app to verify rendered references and previews.

Two real surfaces exist on the shared box:

- **Write surface:** RemNote desktop 1.28.37 (AppImage extracted under /workspace/remnote) on X display `:10`. It hosts the SDK Bridge plugin; `remnote-cli` talks to it over ws `127.0.0.1:3002`. Do not drive or restart it from here.
- **Observe surface (this skill drives it):** RemNote web app in the box Chrome, signed in to the same account and knowledge base reported by `remnote-cli status`.

Driving is done by a computerUse / browser agent through the visible browser, the way a user does. Do not use Shell CDP, injected JavaScript, internal APIs, or cookies.

## Launch

Nothing to build. Read `knowledgeBaseId` from `remnote-cli status` as `kb_id`, then open a new tab in the box Chrome at

```
https://www.remnote.com/w/<kb_id>/<remId>
```

It redirects to `/w/<kb_id>/----<remId>`. Ready when the redirect has happened and the Rem's title text is visible (not a sign-in page, not a blank editor).

Teardown is Cleanup. One tab per run; do not open a second RemNote window on the same KB in parallel with another agent.

## Doctor

Read-only. Stop and report blocked if any fails.

1. Desktop write surface up (for the CLI read-back that precedes this): `ss -ltnp | rg 3002` shows a listener and `remnote-cli status` reports a non-empty `knowledgeBaseId`. Never kill or restart that process.
2. Web app signed in: the Launch URL for the theme Rem shows its title, and the URL kept the `/w/<kb_id>/` prefix (a different KB id means the wrong account).
3. Refs load: within ~10 s every `span.rn-rem-reference` on the page has stopped reading `加载中`.

## Drive

If the batch has no combo cards, skip the browser and record `outcome: "no_combo"` with an empty `combos` array. Otherwise, read `features/INDEX.md` and drive the features touched by this batch. Always drive `combo-front-refs.md` and `combo-answer-refs.md` for every combo card in the batch.

Stable handles:

- Native ref: `span.rem-reference-container[data-rem-reference-id="<title Rem ID>"]`, visible text in inner `span.rn-rem-reference`. It renders blue. Literal text `[[name]]` with no container is a failure.
- Refs read `加载中` until loaded: wait for the real short name before judging.
- Clicking a ref only places the text cursor; it does not navigate. **Hover** the ref to open the preview popup. The popup has the tab text `预览` and shows the title card and its back (answer children).
- Answers are children of the combo Rem; expand the combo (bullet arrow) if collapsed.

## Evidence

Save under the current Fanloop requirement's `evidence/` directory (the requirement root reported by `fanloop flow status`). For each combo capture:

1. The combo front with its blue refs visible (after `加载中` is gone).
2. The expanded answers with their blue refs visible.
3. One hover preview per distinct ref on the front and answers, showing `预览` and the back.

Name the files `<combo_rem_id>-front.png`, `<combo_rem_id>-answers.png`, and `<combo_rem_id>-hover-<ref_id>.png`.

Record the `real_app_verified` object: `{outcome: "passed", surface: "web", kb_id, evidence_dir, combos: [{combo_rem_id, expected_ref_ids, front_ref_ids, answer_ref_ids, front_screenshot, answer_screenshot, hover_screenshots}]}`. Resolve `expected_ref_ids` from that combo's `preparation_receipt.combo_rem_refs`, using `flashcards_written.ids` for new `term_key` values. The union of `front_ref_ids` and `answer_ref_ids` must equal `expected_ref_ids`, and every expected ID must be a batch title-card ID. Capture the front, the expanded answers, and hover previews from the real user path. Screenshots stay out of Git.

## Cleanup

Close only the tab this run opened. Do not sign out, do not close Chrome, do not touch the desktop app on `:10`. Never delete the evidence directory; confirm the screenshot files still exist after closing the tab.
