namespace go erroridl

enum ErrorType {
  unspecified = 0,
  validation  = 1,
  state       = 2,
  workflow    = 3,
  external    = 4,
  internal    = 5,
}

// Numeric values and retired field numbers must never be reused.
enum ErrorCode {
  unspecified              = 0,
  INVALID_ARGUMENT         = 1,
  INVALID_INPUT_JSON       = 2,
  ROOT_REQUIRED            = 3,
  WORKFLOW_REQUIRED        = 4,
  ALREADY_INITIALIZED      = 5,
  NOT_INITIALIZED          = 6,
  STATE_CORRUPT            = 7,
  STATE_SCHEMA_UNSUPPORTED = 8,
  STATE_CONFLICT           = 9,
  STEP_NOT_CURRENT         = 10,
  REPORT_NOT_ALLOWED       = 11,
  WORKFLOW_NOT_FOUND       = 12,
  WORKFLOW_MISMATCH        = 13,
  WORKFLOW_INVALID         = 14,
  LOCAL_COMMIT_FAILED      = 15,
  INTERNAL                 = 16,
  CONDITION_NOT_ALLOWED    = 17,
  CONDITION_CONFLICT       = 18,
  OUTPUT_INVALID           = 19,
  ROUTE_NOT_MATCHED        = 20,
  ROUTE_AMBIGUOUS          = 21,
  ROUTE_NOT_ALLOWED        = 22,
  PROTECTED_DOCUMENT       = 23,
  UPSTREAM_AUTH_FAILED     = 24,
  NETWORK_FAILED           = 25,
  TRACE_UPDATE_FAILED      = 26,
  REGISTRY_UPDATE_FAILED   = 27,
  // 28 and 29 are retired with the public update command and must not be reused.
  // 30 is retired with the hidden __install command and must not be reused.
}

struct ErrorSpec {
  1: required ErrorCode code      (
    vt.defined_only = "true",
    vt.not_in = "ErrorCode.unspecified",
  ),
  2: required ErrorType type      (
    vt.defined_only = "true",
    vt.not_in = "ErrorType.unspecified",
  ),
  3: required i32       exit_code (vt.ge = "1"),
  4: required bool      retryable,
  5: required string    hint      (vt.min_size = "1"),
}

exception PublicError {
  1: required ErrorCode          code    (
    vt.defined_only = "true",
    vt.not_in = "ErrorCode.unspecified",
  ),
  2: required string             message (vt.min_size = "1"),
  3: optional map<string,string> details,
}

const map<ErrorCode,ErrorSpec> ERROR_SPECS = {
  ErrorCode.INVALID_ARGUMENT: {"code": ErrorCode.INVALID_ARGUMENT, "type": ErrorType.validation, "exit_code": 2, "retryable": false, "hint": "Check the command arguments and retry."},
  ErrorCode.INVALID_INPUT_JSON: {"code": ErrorCode.INVALID_INPUT_JSON, "type": ErrorType.validation, "exit_code": 2, "retryable": false, "hint": "Pass one JSON object matching the command Request."},
  ErrorCode.ROOT_REQUIRED: {"code": ErrorCode.ROOT_REQUIRED, "type": ErrorType.validation, "exit_code": 2, "retryable": false, "hint": "Pass the absolute requirement directory."},
  ErrorCode.WORKFLOW_REQUIRED: {"code": ErrorCode.WORKFLOW_REQUIRED, "type": ErrorType.validation, "exit_code": 2, "retryable": false, "hint": "Set workflow in the Init Request."},
  ErrorCode.ALREADY_INITIALIZED: {"code": ErrorCode.ALREADY_INITIALIZED, "type": ErrorType.state, "exit_code": 1, "retryable": false, "hint": "Use flow status or choose another root."},
  ErrorCode.NOT_INITIALIZED: {"code": ErrorCode.NOT_INITIALIZED, "type": ErrorType.state, "exit_code": 1, "retryable": false, "hint": "Run flow init for this requirement root."},
  ErrorCode.STATE_CORRUPT: {"code": ErrorCode.STATE_CORRUPT, "type": ErrorType.state, "exit_code": 5, "retryable": false, "hint": "Repair or restore .fanloop local state."},
  ErrorCode.STATE_SCHEMA_UNSUPPORTED: {"code": ErrorCode.STATE_SCHEMA_UNSUPPORTED, "type": ErrorType.state, "exit_code": 5, "retryable": false, "hint": "Use a CLI release that supports this State Schema."},
  ErrorCode.STATE_CONFLICT: {"code": ErrorCode.STATE_CONFLICT, "type": ErrorType.state, "exit_code": 1, "retryable": true, "hint": "Read flow status again and retry from the current Step."},
  ErrorCode.STEP_NOT_CURRENT: {"code": ErrorCode.STEP_NOT_CURRENT, "type": ErrorType.state, "exit_code": 1, "retryable": false, "hint": "Read flow status again and use its current Step ID."},
  ErrorCode.REPORT_NOT_ALLOWED: {"code": ErrorCode.REPORT_NOT_ALLOWED, "type": ErrorType.state, "exit_code": 1, "retryable": false, "hint": "Follow the latest flow status."},
  ErrorCode.WORKFLOW_NOT_FOUND: {"code": ErrorCode.WORKFLOW_NOT_FOUND, "type": ErrorType.workflow, "exit_code": 1, "retryable": false, "hint": "Choose a Workflow shipped with this release."},
  ErrorCode.WORKFLOW_MISMATCH: {"code": ErrorCode.WORKFLOW_MISMATCH, "type": ErrorType.workflow, "exit_code": 1, "retryable": false, "hint": "Use the release containing the bound Workflow."},
  ErrorCode.WORKFLOW_INVALID: {"code": ErrorCode.WORKFLOW_INVALID, "type": ErrorType.workflow, "exit_code": 5, "retryable": false, "hint": "Fix the bundled Workflow before release."},
  ErrorCode.LOCAL_COMMIT_FAILED: {"code": ErrorCode.LOCAL_COMMIT_FAILED, "type": ErrorType.state, "exit_code": 5, "retryable": false, "hint": "Fix the local filesystem failure and retry."},
  ErrorCode.INTERNAL: {"code": ErrorCode.INTERNAL, "type": ErrorType.internal, "exit_code": 5, "retryable": false, "hint": "Report this runtime error."},
  ErrorCode.CONDITION_NOT_ALLOWED: {"code": ErrorCode.CONDITION_NOT_ALLOWED, "type": ErrorType.validation, "exit_code": 2, "retryable": false, "hint": "Submit only Conditions listed by the latest flow status."},
  ErrorCode.CONDITION_CONFLICT: {"code": ErrorCode.CONDITION_CONFLICT, "type": ErrorType.validation, "exit_code": 2, "retryable": false, "hint": "Remove duplicate or mutually exclusive Condition results."},
  ErrorCode.OUTPUT_INVALID: {"code": ErrorCode.OUTPUT_INVALID, "type": ErrorType.validation, "exit_code": 2, "retryable": false, "hint": "Match the Condition OutputSpec returned by flow status."},
  ErrorCode.ROUTE_NOT_MATCHED: {"code": ErrorCode.ROUTE_NOT_MATCHED, "type": ErrorType.state, "exit_code": 1, "retryable": false, "hint": "Report a complete Condition combination from flow status."},
  ErrorCode.ROUTE_AMBIGUOUS: {"code": ErrorCode.ROUTE_AMBIGUOUS, "type": ErrorType.workflow, "exit_code": 5, "retryable": false, "hint": "Fix ambiguous Workflow Routes before retrying."},
  ErrorCode.ROUTE_NOT_ALLOWED: {"code": ErrorCode.ROUTE_NOT_ALLOWED, "type": ErrorType.validation, "exit_code": 2, "retryable": false, "hint": "Choose one Route returned by the latest flow status."},
  ErrorCode.PROTECTED_DOCUMENT: {"code": ErrorCode.PROTECTED_DOCUMENT, "type": ErrorType.workflow, "exit_code": 1, "retryable": false, "hint": "Bind a dedicated Trace document."},
  ErrorCode.UPSTREAM_AUTH_FAILED: {"code": ErrorCode.UPSTREAM_AUTH_FAILED, "type": ErrorType.external, "exit_code": 1, "retryable": false, "hint": "Authenticate lark-cli as a user."},
  ErrorCode.NETWORK_FAILED: {"code": ErrorCode.NETWORK_FAILED, "type": ErrorType.external, "exit_code": 1, "retryable": true, "hint": "Check network connectivity and retry."},
  ErrorCode.TRACE_UPDATE_FAILED: {"code": ErrorCode.TRACE_UPDATE_FAILED, "type": ErrorType.external, "exit_code": 1, "retryable": true, "hint": "Check Trace document access and retry."},
  ErrorCode.REGISTRY_UPDATE_FAILED: {"code": ErrorCode.REGISTRY_UPDATE_FAILED, "type": ErrorType.external, "exit_code": 1, "retryable": true, "hint": "Check Registry access and retry."},
}
