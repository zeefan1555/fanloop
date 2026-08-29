include "common.thrift"
include "error.thrift"

namespace go flowidl

// Flow public command contract.
//
// Responsibility boundary:
// - This Thrift file owns command methods, request/response types, enums,
//   dynamic Output values, and public errors.
// - Workflow YAML owns Stage/Job/Step, Prompt, Condition, Flow, and Loop.
// - Generated Go files are read-only and must not be edited by hand.

struct Requirement {
  1: required string title      (vt.min_size = "1"),
  2: optional string source_url,
}

enum EvidenceSource {
  unspecified = 0,
  human       = 1,
  system      = 2,
  ai          = 3,
  file        = 4,
  url         = 5,
}

// Evidence is audited but never participates in Route matching.
struct Evidence {
  1: required EvidenceSource source  (
    vt.defined_only = "true",
    vt.not_in = "EvidenceSource.unspecified",
  ),
  2: required string         content (vt.min_size = "1"),
  3: optional string         ref,
}

enum WorkflowStatus {
  unspecified = 0,
  running     = 1,
  completed   = 2,
}

enum StepStatus {
  unspecified = 0,
  ready       = 1,
  in_progress = 2,
  fixing      = 3,
  blocked     = 4,
  // Value 5 is retired and must not be reused.
}

enum Executor {
  unspecified = 0,
  agent       = 1,
  human       = 2,
}

enum ProgressStatus {
  unspecified = 0,
  in_progress = 1,
  fixing      = 2,
  blocked     = 3,
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

// Agent-submitted value. The Condition definition owns the key.
struct OutputValue {
  1: required OutputType       type  (
    vt.defined_only = "true",
    vt.not_in = "OutputType.unspecified",
  ),
  2: required common.JsonValue value (vt.not_nil = "true"),
}

struct OutputSpec {
  1: required string       key         (vt.min_size = "1"),
  2: required OutputType   type        (
    vt.defined_only = "true",
    vt.not_in = "OutputType.unspecified",
  ),
  3: optional string       description,
  4: optional list<string> values,
  5: optional i64          minimum,
  6: optional i64          maximum,
  7: optional i32          min_items,
  8: optional i32          max_items,
  9: optional string       source,
}

struct RegisteredOutput {
  1: required OutputType       type             (
    vt.defined_only = "true",
    vt.not_in = "OutputType.unspecified",
  ),
  2: required common.JsonValue value            (vt.not_nil = "true"),
  3: required string           producer_step_id (vt.min_size = "1"),
}

struct CurrentContext {
  // Fields 1-3 are retired and must not be reused.
  4:  required string   stage_id   (vt.min_size = "1"),
  5:  required string   stage_name (vt.min_size = "1"),
  6:  required string   job_id     (vt.min_size = "1"),
  7:  required string   job_name   (vt.min_size = "1"),
  8:  required string   step_id    (vt.min_size = "1"),
  9:  required string   step_name  (vt.min_size = "1"),
  10: required Executor executor   (
    vt.defined_only = "true",
    vt.not_in = "Executor.unspecified",
  ),
}

struct Execution {
  1: required StepStatus     status   (
    vt.defined_only = "true",
    vt.not_in = "StepStatus.unspecified",
  ),
  2: optional string         summary,
  3: required list<Evidence> evidence,
}

struct Skill {
  1: required string id       (vt.min_size = "1"),
  2: required string prompt   (vt.min_size = "1"),
  3: required bool   optional,
  4: required string path     (vt.min_size = "1"),
}

struct Prompt {
  1: required string      content (vt.min_size = "1"),
  2: required list<Skill> skills,
}

struct ConditionView {
  1: required string     id              (vt.min_size = "1"),
  2: required Prompt     prompt          (vt.not_nil = "true"),
  3: required OutputSpec output          (vt.not_nil = "true"),
  4: optional string     exclusive_group,
}

struct RouteWhen {
  1: required list<list<string>> any_of (vt.min_size = "1"),
}

enum RouteDirection {
  unspecified = 0,
  flow        = 1,
  loop        = 2,
}

union RouteSelection {
  1: string next_step_id (vt.min_size = "1"),
  2: string back_step_id (vt.min_size = "1"),
  3: bool   terminal,
}

struct AvailableRoute {
  1: required RouteDirection direction (
    vt.defined_only = "true",
    vt.not_in = "RouteDirection.unspecified",
  ),
  2: required RouteWhen      when      (vt.not_nil = "true"),
  3: required RouteSelection route     (vt.not_nil = "true"),
  4: required Prompt         prompt    (vt.not_nil = "true"),
}

struct CurrentTask {
  1: required CurrentContext      context    (vt.not_nil = "true"),
  2: required Execution           execution  (vt.not_nil = "true"),
  3: required Prompt              prompt     (vt.not_nil = "true"),
  4: required list<ConditionView> conditions,
  // Fields 5-6 are retired and must not be reused.
  7: required list<AvailableRoute> available_routes,
}

struct FlowState {
  1: required WorkflowStatus               status  (
    vt.defined_only = "true",
    vt.not_in = "WorkflowStatus.unspecified",
  ),
  2: optional CurrentTask                  current,
  3: required map<string,RegisteredOutput> outputs,
}

enum InitEffect {
  unspecified = 0,
  initialized = 1,
}

struct FlowInitRequest {
  1: required string      workflow    (vt.min_size = "1"),
  2: required Requirement requirement (vt.not_nil = "true"),
}

struct FlowInitResponse {
  1: required InitEffect         effect   (
    vt.defined_only = "true",
    vt.not_in = "InitEffect.unspecified",
  ),
  2: required common.WorkflowRef workflow (vt.not_nil = "true"),
  3: required FlowState          state    (vt.not_nil = "true"),
}

struct FlowStatusRequest {}

struct FlowStatusResponse {
  1: required Requirement        requirement (vt.not_nil = "true"),
  2: required common.WorkflowRef workflow    (vt.not_nil = "true"),
  3: required FlowState          state       (vt.not_nil = "true"),
}

enum ProgressEffect {
  unspecified    = 0,
  status_updated = 1,
}

struct FlowProgressRequest {
  1: required string         step_id  (vt.min_size = "1"),
  2: required ProgressStatus status   (
    vt.defined_only = "true",
    vt.not_in = "ProgressStatus.unspecified",
  ),
  3: required string         summary  (vt.min_size = "1"),
  4: required list<Evidence> evidence,
}

struct FlowProgressResponse {
  1: required ProgressEffect effect (
    vt.defined_only = "true",
    vt.not_in = "ProgressEffect.unspecified",
  ),
  2: required FlowState      state  (vt.not_nil = "true"),
}

struct ConditionResult {
  1: required string      condition_id (vt.min_size = "1"),
  2: required OutputValue output       (vt.not_nil = "true"),
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

struct Transition {
  1: required TransitionDirection direction    (
    vt.defined_only = "true",
    vt.not_in = "TransitionDirection.unspecified",
  ),
  2: required string              from_step_id (vt.min_size = "1"),
  3: optional string              to_step_id,
}

struct FlowResultRequest {
  1: required string                step_id           (vt.min_size = "1"),
  2: required list<ConditionResult> condition_results (vt.min_size = "1"),
  3: required list<Evidence>        evidence,
  4: required string                summary           (vt.min_size = "1"),
  // Field 5 is retired and must not be reused.
  6: required RouteSelection route (vt.not_nil = "true"),
}

struct FlowResultResponse {
  1: required ResultEffect effect              (
    vt.defined_only = "true",
    vt.not_in = "ResultEffect.unspecified",
  ),
  2: optional string       event_id,
  3: required Transition   transition          (vt.not_nil = "true"),
  4: required FlowState    state               (vt.not_nil = "true"),
  5: required list<string> invalidated_outputs,
}

// One method maps to one public CLI command:
// Init     -> commonloop flow init
// Status   -> commonloop flow status
// Progress -> commonloop flow report progress
// Result   -> commonloop flow report result
service FlowService {
  FlowInitResponse Init(
    1: required string          requirement_root,
    2: required FlowInitRequest request,
    3: required bool            dry_run,
  ) throws (1: error.PublicError error) (
    cli.id = "flow.init",
    cli.summary = "Initialize a requirement",
    cli.risk = "local_write",
    cli.requirement_scope = "new",
    cli.supports_dry_run = "true",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,ROOT_REQUIRED,WORKFLOW_REQUIRED,ALREADY_INITIALIZED,WORKFLOW_NOT_FOUND,WORKFLOW_INVALID,LOCAL_COMMIT_FAILED,INTERNAL",
  )

  FlowStatusResponse Status(
    1: required string            requirement_root,
    2: required FlowStatusRequest request,
  ) throws (1: error.PublicError error) (
    cli.id = "flow.status",
    cli.summary = "Show current Workflow state",
    cli.risk = "read",
    cli.requirement_scope = "existing",
    cli.supports_dry_run = "false",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,ROOT_REQUIRED,NOT_INITIALIZED,STATE_CORRUPT,STATE_SCHEMA_UNSUPPORTED,WORKFLOW_MISMATCH,INTERNAL",
  )

  FlowProgressResponse Progress(
    1: required string              requirement_root,
    2: required FlowProgressRequest request,
    3: required bool                dry_run,
  ) throws (1: error.PublicError error) (
    cli.id = "flow.report.progress",
    cli.summary = "Update current Step execution facts",
    cli.risk = "local_write",
    cli.requirement_scope = "existing",
    cli.supports_dry_run = "true",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,ROOT_REQUIRED,NOT_INITIALIZED,STATE_CORRUPT,STATE_SCHEMA_UNSUPPORTED,STATE_CONFLICT,STEP_NOT_CURRENT,REPORT_NOT_ALLOWED,WORKFLOW_MISMATCH,WORKFLOW_INVALID,LOCAL_COMMIT_FAILED,INTERNAL",
  )

  FlowResultResponse Result(
    1: required string            requirement_root,
    2: required FlowResultRequest request,
    3: required bool              dry_run,
  ) throws (1: error.PublicError error) (
    cli.id = "flow.report.result",
    cli.summary = "Submit current Step Condition results and route the Workflow",
    cli.risk = "local_write",
    cli.requirement_scope = "existing",
    cli.supports_dry_run = "true",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,ROOT_REQUIRED,NOT_INITIALIZED,STATE_CORRUPT,STATE_SCHEMA_UNSUPPORTED,STATE_CONFLICT,STEP_NOT_CURRENT,REPORT_NOT_ALLOWED,WORKFLOW_MISMATCH,WORKFLOW_INVALID,LOCAL_COMMIT_FAILED,INTERNAL,CONDITION_NOT_ALLOWED,CONDITION_CONFLICT,OUTPUT_INVALID,ROUTE_NOT_MATCHED,ROUTE_AMBIGUOUS,ROUTE_NOT_ALLOWED",
  )
}
