# Review ownership boundaries

Fanloop changes are reviewed by maintenance responsibility, not by one shared
orchestration directory.

| Boundary | Paths | Owns |
| --- | --- | --- |
| Live Workflow configuration | `workflows/`, `exemplars/`, installed `current/workflows/` | Five-file Workflow definitions, bound Agent instructions, seed content and live edits |
| Workflow runtime | `internal/workflow/` | Strict loading, validation and same-root Skill resolution |
| Runtime | `cmd/`, `internal/{flow,loop,trace,card,state,store,larkexec,dashboard}/`, `errs/` | Command behavior, state changes, Lark execution and the loopback editor |
| Quality | `tests/`, `.github/workflows/ci.yml` | Capability inventory, reviewed contracts, `./tests/run-unit`, and CI `requirement-e2e` |
| Local build | `internal/{buildinfo,doctor,release}/`, `tools/release-manifest/`, `scripts/{build-local,install-local}.sh` | Immutable CLI builds, first-seed live configuration, diagnosis and nested Release switching |

Changes to `internal/idl/`, `internal/output/`, public Cobra
flags, State or Event schemas, release manifests, or reviewed contract Goldens
cross boundaries. They require review from Quality plus every affected owning
boundary. The required review check always runs `./tests/run-unit`; user-visible behavior also requires an isolated
candidate, Doctor, the affected public scenarios and fresh-context black-box acceptance. Path filters must not skip it.

Contract Goldens change only after explicit review with:

```bash
go test ./tests/contracts -update-contracts
```

## Live product inventory

Four production Workflow IDs currently ship under `workflows/<id>/`:
`technical-solution-design`, `fanloop-maintainer`, `material-flashcards`, and
`pstack-delivery`. Ownership of those directories remains the Live Workflow
configuration boundary above; this note only records the live suite list.

## Verify entry

Install, `fanloop verify doctor`, the Feature Map, and `verify.evidence.outcome` start at `.cursor/skills/verify-fanloop/SKILL.md`. `features/` in that directory is a symlink to `workflows/fanloop-maintainer/skills/fanloop-dev-verify/references/features/`.

`cmd/verify.go` and `internal/verify` implement that public CLI. The agent entry is `.cursor/skills/verify-fanloop/SKILL.md`. Go package layout stays in place.

## Outer loop

Cuts 1 through 5 are closed. A change needs a local schema-1 verify manifest with outcome `passed` before any cloud agent or parallel agent run for that change. The decision is [ADR-0116](adr/0116-require-local-verify-before-outer-loop.md). The steps are [outer-loop.md](outer-loop.md). Later work is product behavior on this path.
