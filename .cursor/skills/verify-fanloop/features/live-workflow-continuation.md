# Live Workflow continuation

## Sub-features

- Continue a current-schema Requirement after valid live Workflow or Skill edits.
- Preserve the creating CLI Release version and Workflow ID in every durable fact.
- Reject missing current Steps, incompatible Outputs, corrupt Events and new Results that violate current Routes.
- Select the exact bound controller before State decoding at both direct and npm entrypoints.
- Leave invalid Workflow and Route diagnosis, including the Doctor report protocol, to that selected controller rather than the caller's YAML decoder.
- Keep the actual configuration pin authoritative over global current, caller overrides and historical journals.
- Bind new Requirements with the Go controller transaction; explicit protect preserves business bytes and freezes author/live only when requested.
- Reject malformed bindings and resource drift without fallback; retain prepared journals and identifiable staging for explicit protect recovery.
- Reject unsupported unbound Storage schemas without migration. M1 retains State/Event 16; later incompatible-Schema continuity needs its own real cross-release proof.

## How to get to it (user POV)

Create and advance a Requirement, then edit the bound local Workflow package before the next CLI invocation. The
Requirement keeps its Workflow ID while Status, Prompt, Skill paths and new Route validation use the latest live files.

## Driving it with fanloop

Use one isolated install and a fresh Requirement Root. Copy the distributed packages or synthetic fixtures into a separate
verifier-owned writable repository with complete packages under `fanloop-workflow/`, then set the absolute `FANLOOP_CONFIG_SOURCE`
to that copy. Keep the frozen candidate and read-only verification inputs unchanged throughout the run.
Run `flow init`, advance within the approved synthetic scope to create historical Events, and edit a Prompt and Skill only
in the writable copy. Use the same candidate binary for `flow status`, a dry-run and real `flow report`, Card render,
Trace status/render and `doctor --root`; the next invocation must read the edits without publication or reinstallation.
In separate Roots, prove a removed current Step or incompatible
Output/Event facts fail closed, historical accepted Results are not re-decided, and a new Result that misses the latest
Route is rejected atomically. Present an unsupported unbound schema and verify `STATE_SCHEMA_UNSUPPORTED` with no
business write; the execution log may record the failed invocation.

Use a genuine older Release to create or explicitly protect a separate Root. Advance global CLI/configuration to the
candidate, retire the original source Release/configuration only inside the fixture, then enter every tested command through
the candidate direct binary or npm launcher. Verify dispatch to the exact known bound binary, local resource identity,
Prompt/Skill paths and durable State/Event continuity. Running the old binary directly or relabeling the current source with
two versions is not candidate-entry or incompatible-Schema proof.

In a separate author fixture created by the genuine older Release, make its YAML invalid after binding. Compare status and
`doctor --root` through the exact old binary and both candidate entrypoints: stdout, stderr, exit code and CLI execution-log
version must retain the old interpreter's protocol, with unchanged State/Output/Event bytes. This error-ownership check is
not positive incompatible-YAML or incompatible-State continuity proof. Damaged immutable resource identity must still fail
at selection; ordinary configuration read and publication must still reject invalid Workflow packages.

For immutable snapshots, compare the original publication identity, configuration bytes and executable bits with the
Root-local capture. For author/live, verify init retains the editable source and emits its notice; edit Prompt/Skill files,
then explicitly protect and retire the source. The frozen capture must retain those edits without synthetic publication
identity. Rebind to a distinct installed snapshot and prove subsequent direct/Node reads use the actual new pin, not the
journal's prior target; only explicit rebind uses the caller data home for target selection.

Exercise prepared initialize rejection and same-intent protect retry separately. Protect may resume only with unchanged
recorded Release/configuration resources and State/Output/Events bytes. Inspect retained journals/staging on failure and
prove unknown or drifted paths are neither removed nor replaced. Check the complete resource identity, including file
permissions, file/directory sets and empty directories. A healthy no-journal controller remains selectable; a legacy
missing pin is not evidence of immutable configuration protection.

Malformed journal, pin, layout and resource cases must fail closed before business execution. Compare the real Go and
Node public INTERNAL stderr envelope, command/Root/dry-run metadata, empty stdout and exit 5. Go records one unified CLI
log entry; a Node rejection before a trusted CLI starts records zero new CLI log lines and executes no fallback/reporter.

## Gotchas

Do not hand-edit State or Events to manufacture the positive live-edit case. Workflow YAML and bound Skills must come
from the same live root on every invocation. There is no Workflow digest gate, packaged Workflow fallback or State migration.
Release resource identity covers the CLI and both entrypoints; immutable configuration identity is checked separately.
Prepared initialize is never silently completed, even when State already exists. Merging M1 does not automatically protect
pre-existing user Roots, and same-Schema tests do not satisfy the later incompatible-Schema acceptance.
