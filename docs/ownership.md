# Review ownership boundaries

Fanloop changes are reviewed by maintenance responsibility, not by one shared
orchestration directory.

| Boundary | Paths | Owns |
| --- | --- | --- |
| Product workflows | `second-brain-agent/plugins/<plugin>/` | Codex plugin manifests and Skills; no production Fanloop Workflow packages |
| Workflow runtime | `internal/workflow/` | Strict loading, validation and same-root Skill resolution |
| Runtime | `cmd/`, `internal/{flow,loop,trace,card,state,store,larkexec,dashboard}/`, `errs/` | Command behavior, state changes, Lark execution and the loopback read-only viewer and explicit source editor |
| Quality | `tests/`, `.github/workflows/ci.yml` | Capability inventory, reviewed contracts, `./tests/run-unit`, and CI `requirement-e2e` |
| Configuration source | `internal/configuration/`, `internal/workflow/{sources,configuration}.go` | Git main snapshots, explicit author source, complete validation and atomic activation |
| Fornax mirror | `internal/fornaxmirror/`, `tools/workflow-package/`, `cmd/config.go`, `idl/ops.thrift` ConfigDrift symbols | Loop-compatible archives, credential-free mirror mapping, released-package readback and read-only drift |
| Local build | `internal/{buildinfo,doctor,release}/`, `tools/release-manifest/`, `scripts/{build-local,install-local}.sh`, `package.json` | Immutable local CLI builds, offline installation, diagnosis and nested Release switching |
| Distribution | `internal/distribution/`, `cmd/update.go`, `packaging/npm/`, `scripts/{build-release.sh,verify-release-assets.js}`, `.github/workflows/release.yml` | GitHub Release assets, checksum verification, the npm launcher, and `fanloop update` / `fanloop update config` |

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

There are no current production Workflow IDs. Product flows live in `second-brain-agent/plugins/` and execute directly in Codex. Fanloop keeps its generic Runtime, loader and synthetic fixtures so historical Requirements pinned to immutable snapshots remain readable. Do not restore `fanloop-workflow`, `create-fanloop-workflow`, `edit-fanloop-skills` or `plugins/fanloop-remnote`; do not turn a plugin into a Workflow wrapper. `examples/workflows/manage-fanloop-workflow` is a loader fixture and is not a product entry.

## Verify entry

Install, `fanloop verify doctor`, the Feature Map, and `verify.evidence.outcome` start at `.cursor/skills/verify-fanloop/SKILL.md`. `features/` in that directory is the real, code-owned Feature Map. Production configuration owns no duplicate map.

`cmd/verify.go` and `internal/verify` implement that public CLI. The agent entry is `.cursor/skills/verify-fanloop/SKILL.md`. Go package layout stays in place.

## Outer loop

Cuts 1 through 5 are closed. A code change needs a local schema-1 verify manifest with outcome `passed` before any cloud agent or parallel agent run for that change. The decision is [ADR-0116](adr/0116-require-local-verify-before-outer-loop.md). The steps are [outer-loop.md](outer-loop.md). Later work is product behavior on this path.
