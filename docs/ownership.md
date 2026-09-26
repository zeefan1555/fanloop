# Review ownership boundaries

Fanloop changes are reviewed by maintenance responsibility, not by one shared
orchestration directory.

| Boundary | Paths | Owns |
| --- | --- | --- |
| Live Workflow configuration | Fornax published packages, installed `current/workflows/` | Five-file Workflow definitions, bound Agent instructions, complete resources and publication validation |
| Workflow runtime | `internal/workflow/` | Strict loading, validation and same-root Skill resolution |
| Runtime | `cmd/`, `internal/{flow,loop,trace,card,state,store,larkexec,dashboard}/`, `errs/` | Command behavior, state changes, Lark execution and the loopback live/candidate viewer |
| Quality | `tests/`, `.github/workflows/ci.yml` | Capability inventory, reviewed contracts, `./tests/run-unit`, and CI `requirement-e2e` |
| Configuration refresh | `cmd/update.go`, `internal/workflow/fornax.go` | Explicit Fornax discovery/download, full-tree validation, locking and atomic `current/workflows` replacement |
| Local build | `internal/{buildinfo,doctor,release}/`, `tools/release-manifest/`, `scripts/{build-local,install-local}.sh`, `package.json` | Immutable local CLI builds, offline installation, diagnosis and nested Release switching |

Changes to `internal/idl/`, `internal/output/`, public Cobra
flags, State or Event schemas, release manifests, or reviewed contract Goldens
cross boundaries. They require review from Quality plus every affected owning
boundary. For code changes, the required review check runs `./tests/run-unit`; user-visible behavior also requires an isolated
candidate, Doctor, the affected public scenarios and fresh-context black-box acceptance. Code changes must not skip it. Pure Workflow YAML and Skill instruction/reference edits use the configuration validation path in `AGENTS.md`: bundle loading, schema, references, required Skills and route checks with an existing compatible CLI. They do not require a build, code test run, contract Golden update or CLI installation. Executable scripts and changes to the loader, Schema or Runtime remain code changes.

Contract Goldens change only after explicit review with:

```bash
go test ./tests/contracts -update-contracts
```

## Live product inventory

Three production Workflow IDs are hosted in Fornax and synchronized under `current/workflows/<id>/`:
`fanloop-maintainer`, `remnote-flashcards`, and
`pstack-native`. Ownership of those directories remains the Live Workflow
configuration boundary above; this note only records the live suite list.

## Verify entry

Install, `fanloop verify doctor`, the Feature Map, and `verify.evidence.outcome` start at `.cursor/skills/verify-fanloop/SKILL.md`. `features/` in that directory is the real, code-owned Feature Map. Production configuration owns no duplicate map.

`cmd/verify.go` and `internal/verify` implement that public CLI. The agent entry is `.cursor/skills/verify-fanloop/SKILL.md`. Go package layout stays in place.

## Outer loop

Cuts 1 through 5 are closed. A code change needs a local schema-1 verify manifest with outcome `passed` before any cloud agent or parallel agent run for that change. The decision is [ADR-0116](adr/0116-require-local-verify-before-outer-loop.md). The steps are [outer-loop.md](outer-loop.md). Later work is product behavior on this path.
