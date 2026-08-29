# Review ownership boundaries

Commonloop changes are reviewed by maintenance responsibility, not by one shared
orchestration directory.

| Boundary | Paths | Owns |
| --- | --- | --- |
| Skill | `skills/` | Agent instructions and examples |
| Workflow | `workflows/`, `internal/workflow/` | Stage, Step, output, gate, failure target and feedback policy |
| Runtime | `cmd/`, `internal/{flow,loop,trace,card,state,store,larkexec}/`, `errs/` | Command behavior, state changes and Lark execution |
| Quality | `tests/`, `.github/workflows/ci.yml` | Capability inventory, reviewed contracts and E2E gates |
| Release | `internal/{buildinfo,doctor,release}/`, `scripts/{install,run,package-release,prepare-release}.*`, `.goreleaser.yml`, `package.json` | Matched packaging, installation, diagnosis, update and rollback |

Changes to `internal/idl/`, `internal/output/`, public Cobra
flags, State or Event schemas, release manifests, or reviewed contract Goldens
cross boundaries. They require review from Quality plus every affected owning
boundary. The required review check always runs `./tests/run-unit`; code changes also run
`./tests/run-e2e` and attach its report path. Path filters
must not skip it.

Contract Goldens change only after explicit review with:

```bash
go test ./tests/contracts -update-contracts
```
