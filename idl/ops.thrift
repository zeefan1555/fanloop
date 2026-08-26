include "error.thrift"

namespace go opsidl

struct VersionRequest {}

struct StateSchemaSupport {
  1: required list<i32> read_versions,
  2: required i32       write_version,
}

struct SkillRelease {
  1: required string name,
  2: required string version,
}

struct WorkflowRelease {
  1: required string id,
  // fields 2 and 3 retired: Workflow has no business version or default.
  4: required string digest (vt.pattern = "^sha256:[0-9a-f]{64}$"),
}

struct VersionResponse {
  1: required string                release_version,
  2: required string                commit_sha,
  3: required StateSchemaSupport    state_schema    (vt.not_nil = "true"),
  4: required list<SkillRelease>    skills,
  5: required list<WorkflowRelease> workflows,
}

struct DoctorRequest {}

enum DoctorScope {
  unspecified  = 0,
  installation = 1,
  requirement  = 2,
}

enum DoctorStatus {
  unspecified = 0,
  healthy     = 1,
  warning     = 2,
  unhealthy   = 3,
}

enum DoctorCheckStatus {
  unspecified = 0,
  passed      = 1,
  warning     = 2,
  failed      = 3,
  skipped     = 4,
}

struct DoctorCheck {
  1: required string            id,
  2: required DoctorCheckStatus status  (
    vt.defined_only = "true",
    vt.not_in = "DoctorCheckStatus.unspecified",
  ),
  3: required string            summary,
  4: optional string            hint,
}

struct DoctorResponse {
  1: required DoctorScope       scope  (
    vt.defined_only = "true",
    vt.not_in = "DoctorScope.unspecified",
  ),
  2: required DoctorStatus      status (
    vt.defined_only = "true",
    vt.not_in = "DoctorStatus.unspecified",
  ),
  3: required list<DoctorCheck> checks,
}

service OpsService {
  VersionResponse Version(
    1: required VersionRequest request,
  ) throws (1: error.PublicError error) (
    cli.id = "version",
    cli.summary = "Show release and State Schema versions",
    cli.risk = "read",
    cli.requirement_scope = "none",
    cli.supports_dry_run = "false",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,INTERNAL",
  )

  DoctorResponse Doctor(
    1: string                 requirement_root,
    2: required DoctorRequest request,
  ) throws (1: error.PublicError error) (
    cli.id = "doctor",
    cli.summary = "Diagnose the installation or one requirement",
    cli.risk = "read",
    cli.requirement_scope = "optional",
    cli.supports_dry_run = "false",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON",
  )
}
