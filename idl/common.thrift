namespace go commonidl

struct WorkflowRef {
  1: required string id     (vt.min_size = "1"),
  // field 2 retired: Workflow has no business version.
  3: required string digest (vt.min_size = "1"),
}

enum CommandRisk {
  unspecified    = 0,
  read           = 1,
  local_write    = 2,
  external_write = 3,
}

enum RequirementScope {
  unspecified = 0,
  none        = 1,
  new         = 2,
  existing    = 3,
  optional    = 4,
}

struct DriftNotice {
  1: required list<string> components,
  2: required string       command,
}

struct Notice {
  // Field 1 is retired with UpdateNotice and must not be reused.
  2: optional DriftNotice  drift,
}

// JsonValue is encoded as natural JSON by commonidl/json_value.go.
union JsonValue {
  1: bool                  null_value,
  2: string                string_value,
  3: bool                  bool_value,
  4: i64                   integer_value,
  5: double                number_value,
  6: list<JsonValue>       list_value,
  7: map<string,JsonValue> object_value,
}
