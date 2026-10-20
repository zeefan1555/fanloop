# Review ownership boundaries

Fanloop changes are reviewed by maintenance responsibility, not by one shared
orchestration directory.

| Boundary | Paths | Owns |
| --- | --- | --- |
| Live Workflow configuration | `fanloop-workflow/<workflow-id>/` complete packages in second-brain-skills | Five-file Workflow definitions, bound Agent instructions, complete resources and local source validation |
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

Existing production work, including `remnote-flashcards`, uses the current selector and its matching verification suite.

Production Workflow IDs live only in second-brain-skills main under `fanloop-workflow/<id>/`; discover them through its current `select-workflow` Skill. New ordinary Skill changes use `write-skill`, and Workflow package changes use Runtime `manage-fanloop-workflow`. `second-brain-skills-maintainer` continues existing Runs only. Validation, independent review, branch PR delivery and parent gitlink readback belong to the selected Runtime steps. ADR-0135's retirement of fanloop-maintainer and remnote-direct-maintainer remains; code changes use `code-change`. `examples/workflows/manage-fanloop-workflow` is a loader fixture and is not the production entry.

## Verify entry

Install, `fanloop verify doctor`, the Feature Map, and `verify.evidence.outcome` start at `.cursor/skills/verify-fanloop/SKILL.md`. `features/` in that directory is the real, code-owned Feature Map. Production configuration owns no duplicate map.

`cmd/verify.go` and `internal/verify` implement that public CLI. The agent entry is `.cursor/skills/verify-fanloop/SKILL.md`. Go package layout stays in place.

## Outer loop

Cuts 1 through 5 are closed. A code change needs a local schema-1 verify manifest with outcome `passed` before any cloud agent or parallel agent run for that change. The decision is [ADR-0116](adr/0116-require-local-verify-before-outer-loop.md). The steps are [outer-loop.md](outer-loop.md). Later work is product behavior on this path.
