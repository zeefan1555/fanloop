# Workflow products

## Sub-features

- `technical-solution-design` document production and human gates.
- `material-flashcards` recall-goal planning, preview approval and Decks persistence.
- `fanloop-maintainer` four-step Define, Build, Certify and Deliver loop.

## How to get to it (user POV)

Select the matching scenario through `fanloop-workflow`, initialize the returned Workflow ID and follow each Status Prompt
and Skill path. The product is the observable artifacts and governed transitions, not only terminal completion.

## Driving it with fanloop

Use a fresh Root per Workflow. Verify the first meaningful artifact, at least one forward transition, the relevant Panorama
and the nearest governance boundary. For `fanloop-maintainer`, prove the exact candidate and four-step Panorama; check that
`define_verification_contract`, `build_until_verified`, `certify_candidate` and `merge_and_update_local` exist in order and
that retired nine-step IDs and human jump routes are absent. Do not fabricate the Main Agent decisions needed to pass Define
or Certify. Production Bundle contract tests verify Route/Condition semantics.

`verify smoke` proves installation, initialization, one Route transition, evidence and cleanup. It does not replace an
Acceptance Set or full Workflow lifecycle. Compare Status Outputs and created reports with the Workflow's public descriptions.

## Gotchas

Workflow Steps, Conditions and Routes evolve independently. Read the current five-file bundle and Status rather than using
historical counts. Deliver must not modify a candidate when main advances; it returns to Build for a new candidate and full
recertification. Workflow-specific behavior stays in YAML and Skills, not generic verify Runtime code.
