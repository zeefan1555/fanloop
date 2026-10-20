---
status: accepted
date: 2026-10-20
supersedes_in_part: ADR-0125-make-ordinary-skill-maintenance-lightweight, ADR-0122-use-explicit-registry-update-and-local-source-distribution
preserves: ADR-0124-automate-workflow-yaml-validation, ADR-0135-retire-maintainer-workflows
---

# Route Skill maintenance through the corresponding Runtime

AGE-115's approved A/A2 scope and Round 3 decision replace the standalone
maintenance entry of ADR-0125. Dispatch confirmed this scope under the user's
2026-10-19 23:57 delegation: 「你要全程替我做决定，写在你的人设，别让我做决定了」.

New ordinary Skill creation, modification, movement and retirement use
second-brain-skills `write-skill`. New Workflow package changes use Runtime
`manage-fanloop-workflow`. Code and mixed implementation use `code-change`.
The create/edit Skills supply methods within the current bound Step; they do
not start a separate maintenance or delivery procedure. Existing Runs continue
with their pinned configuration; no compatibility layer is added.

When a bound Runtime Step calls a create/edit method, retain the caller's current
Root, workspace and candidate directory. Inspect that Root's latest public
`flow status` and verify the calling Step; do not create another Root or initialize
a nested Run. Missing or inconsistent context returns to the caller for correction.
Only a public call outside a bound Step that explicitly requests a maintenance
Run selects the corresponding Runtime. Resume an existing target Root; otherwise
create an independent absolute Root and inspect Status before initialization.
Initialize only on `NOT_INITIALIZED`, and resolve other errors before continuing.
Verification, approval and delivery remain with the current bound Step.

Ordinary Skills install the verified immutable complete Git archive before PR
delivery, with hash/permission readback and rollback. Git-dependent Skills keep
resources in that snapshot and use the existing workspace's separately verified
canonical repository context for Git operations. Workflow publication completes
the parent PR, actual gitlink and required configuration synchronization proofs.
All changes use branch PRs; the author does not approve or merge the AGE-115
candidates. Dispatch independently reviews the exact heads and handles integration.

ADR-0122's classification of examples as authoring tools is also replaced:
`examples/workflows/example/` and `examples/workflows/manage-fanloop-workflow/`
are test fixtures, with separate contract coverage. The create/edit methods use
approved Runtime contracts and complete configuration packages, without requiring
unpackaged examples. CLI distribution decisions amended by later ADRs are unchanged.
This clarification and the bound-Step rule were approved in AGE-115 Round 4.

Only these authoring clauses and ADR-0125's standalone entry, no-Requirement
orchestration and conflicting installation/delivery rules are replaced. Verification remains proportional to
changed semantics, with independent real trials for routing, permissions,
approvals and external writes. Pure configuration does not require rebuilding the
CLI or a code manifest. CI classification and exact-head applicable checks,
business Human Gates and Panorama requirements retain their existing contracts.

Acceptance: a fresh public caller selects the corresponding Runtime, creates and
inspects a separate Root when needed, and resumes existing Runs without reinitialization.
A bound-Step caller keeps its current Root and returns the candidate to that Step,
without starting another Run. Example assets are covered as fixtures separately
from the author methods. Failed status or missing delivery proof cannot
be reported as completion. No direct-main or standalone-maintenance entry remains.
