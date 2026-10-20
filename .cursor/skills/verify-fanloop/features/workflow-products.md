# Workflow products

## Sub-features

- `remnote-flashcards` HOME classification, native RemNote flashcards, Wiki links and final readback.
- Production `code-change`, `write-skill`, `daily-journal` and `general` Workflows, chosen via the
  second-brain-skills `select-workflow` Skill; code changes use `code-change` (it replaced the retired
  `fanloop-maintainer` and `remnote-direct-maintainer`, ADR-0135).
- Installed `create-fanloop-workflow` supplies a method within the production management Runtime; it does not depend on a source checkout or unpackaged examples.
- Production `manage-fanloop-workflow` lives in second-brain-skills, with analysis, candidate construction, independent review, publication and retrospective. Its `panorama_presented` Condition is an ordinary Route operand; no Runtime action is involved. Publication requires the Skills PR, merged parent PR/tree and actual gitlink, plus the synchronization proofs declared by the loaded package. Examples are test fixtures and cannot certify these production routes.

## How to get to it (user POV)

Select the matching scenario through `fanloop-workflow`, initialize the returned Workflow ID and follow each Status Prompt
and Skill path. The product is the observable artifacts and governed transitions, not only terminal completion.

## Driving it with fanloop

Create a fresh absolute Root directory per Workflow and inspect `flow status`. Initialize only on `NOT_INITIALIZED`; resume existing Runs and block on other errors. Verify the first meaningful artifact, at least one forward transition, the relevant Panorama
and the nearest governance boundary. For the `code-change` Workflow, prove the exact candidate and that each node's receipt is recorded. Do not fabricate the Main Agent decisions needed to pass Define
or Certify. Production Bundle contract tests verify Route/Condition semantics.

For an ordinary Skill request, select production `write-skill` and actually edit
a Skill in a disposable source. Follow its current bound Steps, self-check the
result, then install the verified complete immutable archive through isolated
discovery links before delivery. Read back hashes, permissions and the independent
workspace Git context where needed. Rejection restores the previous links. For a
Workflow request, select production `manage-fanloop-workflow`, construct an isolated
complete candidate and hand it to that Runtime's independent review Step; the
auxiliary create Skill adds no pre-build approval. Prove missing parent merge/tree
or ancestry receipts leave publication incomplete, and concurrent newer pointers
are preserved. Drive retrospective missing-evidence and repair routes from the
actual loaded package. Synthetic platform/Condition inputs prove orchestration
only; real approval, publication and child completion need separate actual receipts.
Routing/binding/security/approval/external-write changes still require an independent
trial; failures and candidate drift cannot reuse passes.
Compare typography, template and associated-doc tasks against direct edits with the
same model, baseline, checks and local commit/readback endpoint. Include Agent reading,
orchestration and presentation in wall time; report external waiting separately.
An explicitly deferred performance gate in the approved Acceptance Set remains
deferred, never passed; preserve failed measurements without repeating them for
delivery. Functional verification still uses the final candidate. Actually drive
risk expansion and failed checks back to the earliest responsible Step, and candidate mutation
through fresh checks before delivery; instruction inspection is not journey proof.

CI's lightweight `changes` job classifies complete Git diff paths and modes, not the
Agent's category claim. Pure configuration skips engine jobs; scripts, validation
assets and mixed changes run all jobs. Missing baselines never exempt code. Use real
temporary Git histories to exercise PR merge-base versus push before/head, renames
in both directions, executable-to-regular mode changes, symlinks and rejected local
materials. The classification script is `.github/scripts/classify-changes.py`;
`scripts/test_maintenance_checks.py` is also run by the existing unit entrypoint.
Only successful explicit `code=false` skips engine jobs. Failed/cancelled classification
and missing/invalid output must fail existing jobs before their engine steps; cover
the real first-step shell gate in `TestCIClassificationFailsClosed`. Delivery must
read same-head `changes` success and actual output even if it is not remotely required.

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

Complete packages under second-brain-skills main's `fanloop-workflow/` determine the production inventory, including code-change, write-skill, daily-journal, general, pstack-native and remnote-flashcards. New init activates a validated immutable commit snapshot; existing calls remain offline. Explicit author-source edits are immediately visible without rebuilding the CLI.
An invalid explicit source or missing bound source fails without searching other repositories or loading old copies.
Never infer the active configuration from CLI version alone; read back the selected source and exact Status Skill paths.
The example Bundle validates as an authoring candidate but is never selected, installed or loaded as production inventory.
The authoring Skill itself is part of the local Release and must be discoverable from every configured Skill root.
