include "error.thrift"

namespace go verifyidl

enum VerificationOutcome {
  unspecified = 0,
  passed      = 1,
  failed      = 2,
  blocked     = 3,
}

enum VerificationCheckStatus {
  unspecified = 0,
  passed      = 1,
  failed      = 2,
  skipped     = 3,
}

struct VerificationCheck {
  1: required string                  id,
  2: required VerificationCheckStatus status,
  3: required string                  summary,
  4: optional string                  evidence_ref,
  5: optional string                  hint,
}

struct VerifyDoctorRequest {}

struct VerifyDoctorResponse {
  1: required VerificationOutcome     outcome,
  2: required list<VerificationCheck> checks,
}

struct VerifySmokeRequest {
  1: optional string evidence_dir,
}

struct VerifySmokeResponse {
  1: required VerificationOutcome     outcome,
  2: required string                  run_id,
  3: required string                  evidence_dir,
  4: required string                  release_version,
  5: required string                  commit_sha,
  6: required string                  binary_sha256,
  7: required list<VerificationCheck> checks,
  8: required bool                    cleanup_complete,
}

struct VerifySnapshotRequest {
  1: optional string evidence_dir,
}

struct VerifySnapshotResponse {
  1: required string       snapshot_dir,
  2: required list<string> files,
}

struct VerifyCleanupRequest {
  1: required string run_id,
}

struct VerifyCleanupResponse {
  1: required string       run_id,
  2: required list<string> paths,
  3: required bool         removed,
}

service VerifyService {
  VerifyDoctorResponse Doctor(
    1: required VerifyDoctorRequest request,
  ) throws (1: error.PublicError error) (
    cli.id = "verify.doctor",
    cli.summary = "Diagnose the verification environment",
    cli.risk = "read",
    cli.requirement_scope = "none",
    cli.supports_dry_run = "false",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,INTERNAL",
  )

  VerifySmokeResponse Smoke(
    1: required VerifySmokeRequest request,
  ) throws (1: error.PublicError error) (
    cli.id = "verify.smoke",
    cli.summary = "Run the public CLI verification smoke journey",
    cli.risk = "local_write",
    cli.requirement_scope = "none",
    cli.supports_dry_run = "false",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,LOCAL_COMMIT_FAILED,INTERNAL",
  )

  VerifySnapshotResponse Snapshot(
    1: string                         requirement_root,
    2: required VerifySnapshotRequest request,
  ) throws (1: error.PublicError error) (
    cli.id = "verify.snapshot",
    cli.summary = "Capture requirement verification evidence",
    cli.risk = "local_write",
    cli.requirement_scope = "existing",
    cli.supports_dry_run = "false",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,ROOT_REQUIRED,NOT_INITIALIZED,STATE_CORRUPT,STATE_SCHEMA_UNSUPPORTED,WORKFLOW_MISMATCH,LOCAL_COMMIT_FAILED,INTERNAL",
  )

  VerifyCleanupResponse Cleanup(
    1: bool                          dry_run,
    2: required VerifyCleanupRequest request,
  ) throws (1: error.PublicError error) (
    cli.id = "verify.cleanup",
    cli.summary = "Clean resources owned by one verification run",
    cli.risk = "local_write",
    cli.requirement_scope = "none",
    cli.supports_dry_run = "true",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,LOCAL_COMMIT_FAILED,INTERNAL",
  )
}
