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

struct VersionResponse {
  1: required string                release_version,
  2: required string                commit_sha,
  3: required StateSchemaSupport    state_schema    (vt.not_nil = "true"),
  4: required list<SkillRelease>    skills,
  // field 5 retired: Workflows are live configuration, not Release artifacts.
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

// drifted, missing, and unreleased remain so older values still decode.
// config drift does not emit them. The released package is the reference.
enum ConfigDriftStatus {
  unspecified    = 0,
  in_sync        = 1,
  drifted        = 2,
  unavailable    = 3,
  local_behind   = 4,
  local_modified = 5,
}

enum WorkflowMirrorStatus {
  unspecified    = 0,
  in_sync        = 1,
  missing        = 2,
  unreleased     = 3,
  drifted        = 4,
  unavailable    = 5,
  local_behind   = 6,
  local_modified = 7,
}

struct ConfigDriftRequest {}

struct WorkflowMirrorCheck {
  1: required string workflow_id (vt.min_size = "1"),
  2: required string skill_key   (vt.min_size = "1"),
  3: required WorkflowMirrorStatus status (
    vt.defined_only = "true",
    vt.not_in = "WorkflowMirrorStatus.unspecified",
  ),
  4: optional string skill_id         (vt.min_size = "1"),
  5: optional string released_version (vt.min_size = "1"),
  6: optional string detail           (vt.min_size = "1"),
}

struct ConfigDriftResponse {
  1: required ConfigDriftStatus status (
    vt.defined_only = "true",
    vt.not_in = "ConfigDriftStatus.unspecified",
  ),
  2: optional string configuration_commit (vt.min_size = "1"),
  3: required list<WorkflowMirrorCheck> workflows (vt.min_size = "1"),
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

  ConfigDriftResponse ConfigDrift(
    1: required ConfigDriftRequest request,
  ) throws (1: error.PublicError error) (
    cli.id = "config.drift",
    cli.summary = "Compare the active Workflow snapshot with the latest Fornax release",
    cli.risk = "read",
    cli.requirement_scope = "none",
    cli.supports_dry_run = "false",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,INTERNAL",
  )
}
