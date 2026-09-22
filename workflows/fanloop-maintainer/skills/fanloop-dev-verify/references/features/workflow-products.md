# Workflow products

## Sub-features

- `technical-solution-design` document production and human gates.
- `remnote-flashcards` HOME classification, native RemNote flashcards, Wiki links and final readback.
- `fanloop-maintainer` four-step Define, Build, Certify and Deliver loop.
- `pstack-delivery` thin orchestration over native pstack skills under the workflow bundle.

## How to get to it (user POV)

Select the matching scenario through `fanloop-workflow`, initialize the returned Workflow ID and follow each Status Prompt
and Skill path. The product is the observable artifacts and governed transitions, not only terminal completion.

## Driving it with fanloop

Use a fresh Root per Workflow. Verify the first meaningful artifact, at least one forward transition, the relevant Panorama
and the nearest governance boundary. For `fanloop-maintainer`, prove the exact candidate and four-step Panorama; check that
`define_verification_contract`, `build_until_verified`, `certify_candidate` and `merge_and_update_local` exist in order and
that retired nine-step IDs and human jump routes are absent. Do not fabricate the Main Agent decisions needed to pass Define
or Certify. Production Bundle contract tests verify Route/Condition semantics.

For `remnote-flashcards`, verify exactly four agent Steps in order: `resolve_home_tag`, `write_flashcards`,
`update_wiki_links`, `verify_result`. Status resolves six Skills from that same live bundle: `remnote-structure`,
`remnote-cli`, `remnote-flashcards`, `remnote-wiki`, `remnote-verify`, and optional `control-remnote`.
Use isolated synthetic ID receipts to prove four forward/terminal Routes, ten repair Routes, output invalidation,
blocked/rejected request atomicity, and zero-new-card completion. The retired `material-flashcards` ID must be rejected.
These receipts prove orchestration only; real RemNote business acceptance requires separately authorized object readback.
Wiki creation and new card references follow card writing; only final verification ends the workflow.

`verify smoke` proves installation, initialization, one Route transition, evidence and cleanup. It does not replace an
Acceptance Set or full Workflow lifecycle. Compare Status Outputs and created reports with the Workflow's public descriptions.

## Gotchas

Workflow Steps, Conditions and Routes evolve independently. Read the current five-file bundle and Status rather than using
historical counts. Deliver must not modify a candidate when main advances; it returns to Build for a new candidate and full
recertification. Workflow-specific behavior stays in YAML and Skills, not generic verify Runtime code.

Existing installations preserve live Workflow bytes on install. For this replacement, separately verify the scoped
addition of `remnote-flashcards` and removal of `material-flashcards` in an isolated live copy, then at delivery;
also synchronize this product-description file only. All other pre-existing live bytes must remain unchanged;
do not seed unrelated missing workflows. Source first-install inventory and an existing live inventory may differ.
Never infer deployment from CLI version alone.
