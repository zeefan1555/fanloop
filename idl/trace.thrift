include "error.thrift"

namespace go traceidl

typedef error.ErrorCode TraceTargetErrorCode

enum TraceBindEffect {
  unspecified = 0,
  bound       = 1,
  unchanged   = 2,
}

enum TraceSyncOutcome {
  unspecified = 0,
  succeeded   = 1,
  partial     = 2,
  skipped     = 3,
}

enum TraceTarget {
  unspecified      = 0,
  trace_document   = 1,
  registry         = 2,
  cli_log_document = 3,
}

enum TraceTargetStatus {
  unspecified = 0,
  succeeded   = 1,
  failed      = 2,
  skipped     = 3,
}

struct TraceTargetError {
  1: required TraceTargetErrorCode code      (
    vt.defined_only = "true",
    vt.not_in = "error.ErrorCode.unspecified",
  ),
  2: required string               message   (vt.min_size = "1"),
  3: required bool                 retryable,
}

struct TraceTargetResult {
  1: required TraceTarget       target (
    vt.defined_only = "true",
    vt.not_in = "TraceTarget.unspecified",
  ),
  2: required TraceTargetStatus status (
    vt.defined_only = "true",
    vt.not_in = "TraceTargetStatus.unspecified",
  ),
  3: optional string            reason,
  4: optional TraceTargetError  error,
}

struct TraceRegistry {
  1: required string profile    (vt.min_size = "1"),
  2: required string url        (vt.min_size = "1"),
  3: required string base_token (vt.min_size = "1"),
  4: required string table_id   (vt.min_size = "1"),
  5: required string view_id    (vt.min_size = "1"),
}

struct TraceLastSync {
  1: required string                  occurred_at (vt.min_size = "1"),
  2: required TraceSyncOutcome        outcome     (
    vt.defined_only = "true",
    vt.not_in = "TraceSyncOutcome.unspecified",
  ),
  3: required list<TraceTargetResult> targets,
}

struct TraceBindRequest {
  1: required string document_url (vt.min_size = "1"),
  2: optional string registry,
  3: optional string cli_log_document_url,
}

struct TraceBindResponse {
  1: required TraceBindEffect effect (
    vt.defined_only = "true",
    vt.not_in = "TraceBindEffect.unspecified",
  ),
}

struct TraceStatusRequest {}

struct TraceStatusResponse {
  1: optional string        document_url,
  2: optional TraceRegistry registry,
  3: required i32           event_count  (vt.ge = "0"),
  4: optional TraceLastSync last_sync,
  5: optional string        cli_log_document_url,
}

struct TraceRenderRequest {}

struct TraceRenderResponse {
  1: required i32    event_count     (vt.ge = "0"),
  2: required string projection_path (vt.min_size = "1"),
}

struct TraceSyncRequest {}

struct TraceSyncResponse {
  1: required TraceSyncOutcome        outcome (
    vt.defined_only = "true",
    vt.not_in = "TraceSyncOutcome.unspecified",
  ),
  2: required list<TraceTargetResult> targets,
}

service TraceService {
  TraceBindResponse Bind(
    1: required string           requirement_root,
    2: required TraceBindRequest request,
    3: required bool             dry_run,
  ) throws (1: error.PublicError error) (
    cli.id = "trace.bind",
    cli.summary = "Bind the Trace document",
    cli.risk = "local_write",
    cli.requirement_scope = "existing",
    cli.supports_dry_run = "true",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,ROOT_REQUIRED,NOT_INITIALIZED,STATE_CORRUPT,STATE_SCHEMA_UNSUPPORTED,WORKFLOW_MISMATCH,LOCAL_COMMIT_FAILED,PROTECTED_DOCUMENT,INTERNAL",
  )

  TraceStatusResponse Status(
    1: required string             requirement_root,
    2: required TraceStatusRequest request,
  ) throws (1: error.PublicError error) (
    cli.id = "trace.status",
    cli.summary = "Show Trace binding and projection status",
    cli.risk = "read",
    cli.requirement_scope = "existing",
    cli.supports_dry_run = "false",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,ROOT_REQUIRED,NOT_INITIALIZED,STATE_CORRUPT,STATE_SCHEMA_UNSUPPORTED,WORKFLOW_MISMATCH",
  )

  TraceRenderResponse Render(
    1: required string             requirement_root,
    2: required TraceRenderRequest request,
    3: required bool               dry_run,
  ) throws (1: error.PublicError error) (
    cli.id = "trace.render",
    cli.summary = "Render the local Events projection",
    cli.risk = "local_write",
    cli.requirement_scope = "existing",
    cli.supports_dry_run = "true",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,ROOT_REQUIRED,NOT_INITIALIZED,STATE_CORRUPT,STATE_SCHEMA_UNSUPPORTED,WORKFLOW_MISMATCH,LOCAL_COMMIT_FAILED",
  )

  TraceSyncResponse Sync(
    1: required string           requirement_root,
    2: required TraceSyncRequest request,
    3: required bool             dry_run,
  ) throws (1: error.PublicError error) (
    cli.id = "trace.sync",
    cli.summary = "Sync Trace and Registry projections",
    cli.risk = "external_write",
    cli.requirement_scope = "existing",
    cli.supports_dry_run = "true",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,ROOT_REQUIRED,NOT_INITIALIZED,STATE_CORRUPT,STATE_SCHEMA_UNSUPPORTED,WORKFLOW_MISMATCH,LOCAL_COMMIT_FAILED,UPSTREAM_AUTH_FAILED,NETWORK_FAILED,TRACE_UPDATE_FAILED,REGISTRY_UPDATE_FAILED",
  )
}
