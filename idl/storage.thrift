include "common.thrift"

namespace go storageidl

// Durable Requirement storage contract.
//
// This file owns every Commonloop-defined structured JSON/JSONL document under
// .commonloop. It defines file schemas only: no public CLI methods or RPC service.

const i32 FLOW_STATE_SCHEMA_VERSION      = 12
const i32 EVENT_SCHEMA_VERSION           = 12
const i32 OUTPUT_REGISTRY_SCHEMA_VERSION = 3
const i32 CARD_PROJECTION_SCHEMA_VERSION = 5
const i32 CARD_BINDING_SCHEMA_VERSION      = 2
const i32 TRACE_CONFIG_SCHEMA_VERSION      = 2
const i32 CLI_EXECUTION_LOG_SCHEMA_VERSION = 2

struct CLIExecutionLogEntry {
  1:  required i32    schema_version   (vt.ge = "1"),
  2:  required string invocation_id   (vt.min_size = "1"),
  3:  required string started_at      (vt.min_size = "1"),
  4:  required i64    duration_ms     (vt.ge = "0"),
  5:  required string command_id      (vt.min_size = "1"),
  6:  required string cli_version     (vt.min_size = "1"),
  7:  required string release_version (vt.min_size = "1"),
  8:  required string commit_sha      (vt.min_size = "1"),
  9:  required bool   dry_run,
  10: required i32    exit_code       (vt.ge = "0"),
  11: optional string error_code,
  12: required list<string> arguments,
  13: required string       stdin,
  14: required string       stdout,
  15: required string       stderr,
}

enum StepStatus {
  unspecified = 0,
  ready       = 1,
  in_progress = 2,
  fixing      = 3,
  blocked     = 4,
}

enum EvidenceSource {
  unspecified = 0,
  human       = 1,
  system      = 2,
  ai          = 3,
  file        = 4,
  url         = 5,
}

enum OutputType {
  unspecified = 0,
  string      = 1,
  boolean     = 2,
  integer     = 3,
  path        = 4,
  url         = 5,
  url_list    = 6,
  enum_value  = 7,
  object      = 8,
}

enum RegistryProfile {
  unspecified = 0,
  production  = 1,
  test        = 2,
}

enum EventKind {
  unspecified          = 0,
  flow_initialized     = 1,
  flow_progressed      = 2,
  flow_result          = 3,
  trace_document_bound = 4,
  trace_sync_started   = 5,
  trace_synced         = 6,
}

enum ResultEffect {
  unspecified = 0,
  advanced    = 1,
  looped      = 2,
  completed   = 3,
}

enum TransitionDirection {
  unspecified = 0,
  flow        = 1,
  loop        = 2,
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

enum TraceTargetErrorCode {
  unspecified            = 0,
  trace_update_failed    = 1,
  registry_update_failed = 2,
  network_failed         = 3,
  upstream_auth_failed   = 4,
}

struct Requirement {
  1: required string title      (vt.min_size = "1"),
  2: optional string source_url,
}

struct Release {
  1: required string             version  (vt.min_size = "1"),
  2: required common.WorkflowRef workflow (vt.not_nil = "true"),
}

struct Evidence {
  1: required EvidenceSource source  (
    vt.defined_only = "true",
    vt.not_in = "EvidenceSource.unspecified",
  ),
  2: required string         content (vt.min_size = "1"),
  3: optional string         ref,
}

struct OutputValue {
  1: required OutputType       type  (
    vt.defined_only = "true",
    vt.not_in = "OutputType.unspecified",
  ),
  2: required common.JsonValue value (vt.not_nil = "true"),
}

struct ConditionResult {
  1: required string      condition_id (vt.min_size = "1"),
  2: required OutputValue output       (vt.not_nil = "true"),
}

struct RegisteredOutput {
  1: required OutputType       type             (
    vt.defined_only = "true",
    vt.not_in = "OutputType.unspecified",
  ),
  2: required common.JsonValue value            (vt.not_nil = "true"),
  3: required string           producer_step_id (vt.min_size = "1"),
}

struct TraceBinding {
  1: required string          document_url (vt.min_size = "1"),
  2: required RegistryProfile registry     (
    vt.defined_only = "true",
    vt.not_in = "RegistryProfile.unspecified",
  ),
  3: optional string          cli_log_document_url,
}

struct Integrations {
  1: optional TraceBinding trace,
}

struct FlowState {
  1:  required i32            schema_version       (vt.ge = "1"),
  2:  required Requirement    requirement          (vt.not_nil = "true"),
  3:  required Release        release              (vt.not_nil = "true"),
  4:  optional string         current_step_id,
  5:  optional StepStatus     current_step_status  (
    vt.defined_only = "true",
    vt.not_in = "StepStatus.unspecified",
  ),
  6:  optional string         current_step_summary,
  7:  optional list<Evidence> current_evidence,
  8:  required Integrations   integrations         (vt.not_nil = "true"),
  9:  required string         last_event_id        (vt.min_size = "1"),
  10: required string         created_at           (vt.min_size = "1"),
  11: required string         updated_at           (vt.min_size = "1"),
}

struct OutputRegistry {
  1: required i32                          schema_version (vt.ge = "1"),
  2: required common.WorkflowRef           workflow       (vt.not_nil = "true"),
  3: required map<string,RegisteredOutput> outputs,
}

struct OutputChanges {
  1: optional list<string> accepted,
  2: optional list<string> invalidated,
}

struct Transition {
  1: required TransitionDirection direction    (
    vt.defined_only = "true",
    vt.not_in = "TransitionDirection.unspecified",
  ),
  2: required string              from_step_id (vt.min_size = "1"),
  3: optional string              to_step_id,
}

struct FlowInitializedPayload {
  1: required string     step_id     (vt.min_size = "1"),
  2: required StepStatus step_status (
    vt.defined_only = "true",
    vt.not_in = "StepStatus.unspecified",
  ),
}

struct FlowProgressedPayload {
  1: required string         from_step_id     (vt.min_size = "1"),
  2: required StepStatus     from_step_status (
    vt.defined_only = "true",
    vt.not_in = "StepStatus.unspecified",
  ),
  3: required StepStatus     to_step_status   (
    vt.defined_only = "true",
    vt.not_in = "StepStatus.unspecified",
  ),
  4: required string         summary          (vt.min_size = "1"),
  5: optional list<Evidence> evidence,
}

struct FlowResultPayload {
  1: required list<ConditionResult> condition_results (vt.min_size = "1"),
  2: optional list<Evidence>        evidence,
  3: required string                summary           (vt.min_size = "1"),
  4: required ResultEffect          effect            (
    vt.defined_only = "true",
    vt.not_in = "ResultEffect.unspecified",
  ),
  5: required Transition            transition        (vt.not_nil = "true"),
  6: required OutputChanges         output_changes    (vt.not_nil = "true"),
}

struct TraceDocumentBoundPayload {
  1: optional string          previous_document_url,
  2: required string          document_url          (vt.min_size = "1"),
  3: optional RegistryProfile previous_registry     (
    vt.defined_only = "true",
    vt.not_in = "RegistryProfile.unspecified",
  ),
  4: required RegistryProfile registry              (
    vt.defined_only = "true",
    vt.not_in = "RegistryProfile.unspecified",
  ),
  5: optional string          previous_cli_log_document_url,
  6: optional string          cli_log_document_url,
}

struct TraceSyncStartedPayload {
  1: required list<TraceTarget> targets (vt.min_size = "1"),
}

struct TraceTargetError {
  1: required TraceTargetErrorCode code      (
    vt.defined_only = "true",
    vt.not_in = "TraceTargetErrorCode.unspecified",
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

struct TraceSyncedPayload {
  1: required TraceSyncOutcome        outcome (
    vt.defined_only = "true",
    vt.not_in = "TraceSyncOutcome.unspecified",
  ),
  2: required list<TraceTargetResult> targets (vt.min_size = "1"),
}

union EventPayload {
  1: FlowInitializedPayload    flow_initialized,
  2: FlowProgressedPayload     flow_progressed,
  3: FlowResultPayload         flow_result,
  4: TraceDocumentBoundPayload trace_document_bound,
  5: TraceSyncStartedPayload   trace_sync_started,
  6: TraceSyncedPayload        trace_synced,
}

struct Event {
  1: required i32                schema_version     (vt.ge = "1"),
  2: required string             event_id           (vt.min_size = "1"),
  3: required string             occurred_at        (vt.min_size = "1"),
  4: required EventKind          kind               (
    vt.defined_only = "true",
    vt.not_in = "EventKind.unspecified",
  ),
  5: required string             command            (vt.min_size = "1"),
  6: required common.WorkflowRef workflow           (vt.not_nil = "true"),
  7: optional string             caused_by_event_id,
  8: required EventPayload       payload            (vt.not_nil = "true"),
}

struct CardProjection {
  1:  required i32                          schema_version       (vt.ge = "1"),
  2:  required Requirement                  requirement          (vt.not_nil = "true"),
  3:  required Release                      release              (vt.not_nil = "true"),
  4:  optional string                       current_step_id,
  5:  optional StepStatus                   current_step_status  (
    vt.defined_only = "true",
    vt.not_in = "StepStatus.unspecified",
  ),
  6:  optional string                       current_step_summary,
  7:  optional list<Evidence>               current_evidence,
  8:  required map<string,RegisteredOutput> outputs,
  9:  optional string                       trace_document_url,
  10: required string                       source_event_id      (vt.min_size = "1"),
  11: required string                       updated_at           (vt.min_size = "1"),
  12: optional string                       cli_log_document_url,
}

struct CardBinding {
  1: required i32    schema_version (vt.ge = "1"),
  2: required string chat_id        (vt.min_size = "1"),
  3: required string session_id     (vt.min_size = "1"),
}

struct TraceConfig {
  1:  required i32             schema_version                 (vt.ge = "1"),
  2:  optional string          trace_document_url,
  3:  optional RegistryProfile registry_profile               (
    vt.defined_only = "true",
    vt.not_in = "RegistryProfile.unspecified",
  ),
  4:  optional string          registry_url,
  5:  optional string          registry_base_token,
  6:  optional string          registry_table_id,
  7:  optional string          registry_view_id,
  8:  required string          last_sync_at,
  9:  required string          last_sync_error,
  10: required string          trace_document_last_sync_at,
  11: required string          trace_document_last_sync_error,
  12: required string          registry_last_sync_at,
  13: required string          registry_last_sync_error,
  14: optional string          cli_log_document_url,
  15: required string          cli_log_document_last_sync_at,
  16: required string          cli_log_document_last_sync_error,
}
