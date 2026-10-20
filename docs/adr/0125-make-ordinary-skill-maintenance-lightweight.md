---
status: accepted
date: 2026-10-02
amends: ADR-0109, ADR-0119, ADR-0123, ADR-0124
superseded_in_part_by: ADR-0144-route-skill-maintenance-through-runtime
---

# Ordinary Skill Maintenance Has No Separate Delivery Ceremony

The standalone maintenance entry, no-Requirement orchestration, installation and
delivery endpoint below are superseded by [ADR-0144](0144-route-skill-maintenance-through-runtime.md).
Its proportional verification, risk-sensitive independent trials, CI classification
and code/configuration boundaries remain effective.

## Direct Repository Skill Maintenance (2026-10-03)

The user approved a standalone `.agents/skills/edit-fanloop-skills/SKILL.md`:
edit, validate the affected content, commit/push, create or update one PR, merge
after applicable exact-head checks, and read back the actual local source.
Pure configuration does not initialize Maintainer, create a Requirement, advance
four Steps, present maintenance Panoramas or produce stage reports. This replaces
the pure-configuration orchestration requirements below and in ADR-0109/0119/0124;
business Workflow Human Gates, Panorama Conditions and risk-sensitive independent
trials are unchanged. Code and mixed implementation still use Maintainer.

The Skill covers Workflow YAML, non-executable Skill Markdown in `skills/` and
`.agents/skills/`, and associated documentation. It reuses the existing validator
and public CLI, with verification proportional to changed semantics. No new
Workflow, script framework, runtime, dependency or CLI installation is introduced.
Read `FANLOOP_CONFIG_SOURCE` and the configured data home's `config/current` to
identify the actual source; fast-forward that repository after merging and verify
its loaded paths and file identities. Do not overwrite local edits, silently
rebind the source or confuse `config/current` with the CLI's `current` directory.

The CI allowlist additionally accepts ordinary, non-executable Markdown below
`.agents/skills/<skill>/`. Existing mode, asset-directory, rename, mixed-diff and
fail-closed checks remain. The user explicitly included this narrow selector
policy change and its focused regression checks in the direct delivery; changes
to the selector itself and validation assets still run applicable code CI. This
does not make `.github/`, executable scripts or unknown paths configuration-only.

Acceptance: a fresh Agent can use the standalone Skill without starting a
maintenance Workflow; ordinary repository Skill text classifies as configuration,
while scripts, executable Markdown, symlinks, verification assets and mixed or
renamed code changes do not. Delivery ends at a merged PR and verified local source,
unless the user explicitly selects an earlier stopping point.

## Earlier Four-Step Optimization (2026-10-02)

The user approved keeping the existing configuration/code channels and four Steps,
while making ordinary Skill maintenance cost approximately the same as direct
editing with the same checks and delivery endpoint. No new Workflow or runtime is
introduced. The initial measured target is at most 30 seconds additional wall time;
measure Agent reading, orchestration and presentation, not just CLI execution.

Configuration maintenance uses one continuing Agent and one local reading artifact,
`delivery-record.md`. Required per-stage Lark reports are removed only from the
configuration routes. Raw tool evidence remains separate and is referenced, not
duplicated into prose reports. Same-goal associated-file discovery updates the
actual file list without returning to Define. Goal, acceptance, permission,
side-effect or category changes still return to Define.

Ordinary template and Prompt changes receive actual self-checks with representative
examples; typography may receive a justified exemption. Routing, bindings,
security, permissions, approval or external-write behavior retain a fresh independent
trial. Record which kind of verification actually happened. Invalid configuration,
failed examples and candidate drift still prevent delivery. This revises only
ADR-0119's 2026-09-23 configuration-trial supplement, not its Panorama requirements.
Every Step continues to read current Status and display its real Panorama once.

Validation follows changed semantics rather than file extensions. Prompt prose
needs strict affected-package loading and relevant behavior checks; structural
configuration needs the affected Workflow route matrix. The existing validator's
repeatable `--workflow` selects matrices, defaults to all and rejects unknown IDs;
the full source manifest and unchanged-source check remain. Parser-negative tests
remain engine regression, not a manually repeated suite for each prose edit. This
amends the validation breadth in ADR-0124-automate-workflow-yaml-validation.

CI classifies the complete Git diff, including both paths and modes of renames.
Only ordinary non-executable Skill Markdown, package-root five YAML, ordinary docs
Markdown and root AGENTS.md/README.md can skip engine jobs. Scripts, validation
assets, executable entries, symlinks, unknown paths and mixed changes retain all
code checks. Missing baselines cannot classify as configuration; classification
errors fail delivery. The lightweight job retains the local-material prohibition.
This selector is not evidence of configuration behavior and does not weaken local
checks. Keep existing job names and honor all applicable exact-head required checks.
Only successful classification with explicit `code=false` may skip engine jobs.
Failure, cancellation or missing/invalid output starts the existing jobs and fails
their first step before engine work, preserving existing required-check protection.
Delivery explicitly reads the same-head `changes` success and actual `code` output,
even when that job is absent from the remote required set; local inference is not
remote evidence. Code and mixed delivery require `code=true`.

Code and mixed changes retain the old full verification, independent certification,
main-Agent decisions and exact-candidate delivery. Configuration uses authorized
Git delivery and actual loaded-source readback, never CLI rebuild/install. The
local package truth from ADR-0124-load-local-workflow-skill-packages remains.

Acceptance compares three real paired edits (typography, template, associated docs)
with the same model, baseline, checks and local commit/readback endpoint. Report
real times and external waiting separately; a missed target is not a pass.

On 2026-10-02 the user approved delivering the current optimization with only the
30-second performance gate deferred to subsequent optimization. The three initial
paired trials remain failed; they are not repeated or relabeled for this delivery.
The final candidate still requires ordinary functional completion, actual risk,
failure and stale-evidence journeys, independent review/blackbox, applicable CI and
exact-commit delivery. Four Steps and per-entry Panorama remain unchanged; the
proposed one-execution/one-Panorama redesign is not approved.

Approval: the original plan's requirements SHA-256 was
`5186391b0a9bb54064ebfa1a02e37beb82c79e4d8d078c87cb9be6d8d625acd8`.
Main-Agent approval of the revised scope covers SHA-256
`7f0c871d70c18393116f52177d6f0f8b60d39749a50c4e03d6a22511e8c1b8f7`,
host turn `01a0fc0f-5ff0-7911-8690-e20acc5139c7`.
