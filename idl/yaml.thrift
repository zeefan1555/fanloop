namespace go yamlidl

// Workflow YAML authoring contract.
//
// This file owns the five Workflow YAML document shapes, their enums, stable
// field IDs, and schema versions. It declares no public command or Service.

const i32 WORKFLOW_SCHEMA_VERSION  = 7
const i32 FLOW_SCHEMA_VERSION      = 4
const i32 CONDITION_SCHEMA_VERSION = 2
const i32 LOOP_SCHEMA_VERSION      = 4
const i32 PROMPT_SCHEMA_VERSION    = 1

const string PROMPT_FILE                      = "prompt.yaml"
const string OUTPUT_SOURCE_TRACE_DOCUMENT_URL = "integration.trace.document_url"

enum Executor {
  unspecified = 0,
  agent       = 1,
  human       = 2,
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

struct Step {
  1: required string   id       (
    vt.min_size = "1",
    go.tag = 'yaml:"id"',
  ),
  2: required string   name     (
    vt.min_size = "1",
    go.tag = 'yaml:"name"',
  ),
  3: required Executor executor (
    vt.defined_only = "true",
    vt.not_in = "Executor.unspecified",
    go.tag = 'yaml:"executor"',
  ),
}

struct Job {
  1: required string     id    (
    vt.min_size = "1",
    go.tag = 'yaml:"id"',
  ),
  2: required string     name  (
    vt.min_size = "1",
    go.tag = 'yaml:"name"',
  ),
  3: required list<Step> steps (go.tag = 'yaml:"steps"'),
}

struct Stage {
  1: required string    id   (
    vt.min_size = "1",
    go.tag = 'yaml:"id"',
  ),
  2: required string    name (
    vt.min_size = "1",
    go.tag = 'yaml:"name"',
  ),
  3: required list<Job> jobs (go.tag = 'yaml:"jobs"'),
}

struct PromptRef {
  1: required string file      (
    vt.min_size = "1",
    go.tag = 'yaml:"file"',
  ),
  2: required string prompt_id (
    vt.min_size = "1",
    go.tag = 'yaml:"prompt_id"',
  ),
}

struct When {
  1: required list<list<string>> any_of (
    vt.min_size = "1",
    go.tag = 'yaml:"any_of"',
  ),
}

struct FlowRoute {
  1: required PromptRef prompt_ref   (
    vt.not_nil = "true",
    go.tag = 'yaml:"prompt_ref"',
  ),
  2: required When      when         (
    vt.not_nil = "true",
    go.tag = 'yaml:"when"',
  ),
  3: optional string    next_step_id (
    vt.min_size = "1",
    go.tag = 'yaml:"next_step_id,omitempty"',
  ),
  4: optional bool      terminal     (go.tag = 'yaml:"terminal,omitempty"'),
}

struct LoopRoute {
  1: required PromptRef prompt_ref   (
    vt.not_nil = "true",
    go.tag = 'yaml:"prompt_ref"',
  ),
  2: required When      when         (
    vt.not_nil = "true",
    go.tag = 'yaml:"when"',
  ),
  3: required string    back_step_id (
    vt.min_size = "1",
    go.tag = 'yaml:"back_step_id"',
  ),
}

struct OutputDefinition {
  1: required string       key         (
    vt.min_size = "1",
    go.tag = 'yaml:"key"',
  ),
  2: required OutputType   type        (
    vt.defined_only = "true",
    vt.not_in = "OutputType.unspecified",
    go.tag = 'yaml:"type"',
  ),
  3: optional string       source      (go.tag = 'yaml:"source,omitempty"'),
  4: optional string       description (go.tag = 'yaml:"description,omitempty"'),
  5: optional list<string> values      (go.tag = 'yaml:"values,omitempty"'),
  6: optional i64          minimum     (go.tag = 'yaml:"minimum,omitempty"'),
  7: optional i64          maximum     (go.tag = 'yaml:"maximum,omitempty"'),
  8: optional i32          min_items   (go.tag = 'yaml:"min_items,omitempty"'),
  9: optional i32          max_items   (go.tag = 'yaml:"max_items,omitempty"'),
}

struct ConditionDefinition {
  1: required PromptRef        prompt_ref      (
    vt.not_nil = "true",
    go.tag = 'yaml:"prompt_ref"',
  ),
  2: required OutputDefinition output          (
    vt.not_nil = "true",
    go.tag = 'yaml:"output"',
  ),
  3: optional string           exclusive_group (go.tag = 'yaml:"exclusive_group,omitempty"'),
}

struct SkillBinding {
  1: required string id     (
    vt.min_size = "1",
    go.tag = 'yaml:"id"',
  ),
  2: required string prompt (
    vt.min_size = "1",
    go.tag = 'yaml:"prompt"',
  ),
  // Optional at the Thrift layer so YAML omission remains distinguishable from
  // an explicit false. Bundle validation requires the field to be present.
  3: optional bool optional (go.tag = 'yaml:"optional"'),
}

struct PromptDefinition {
  1: required string             prompt (
    vt.min_size = "1",
    go.tag = 'yaml:"prompt"',
  ),
  2: required list<SkillBinding> skills (go.tag = 'yaml:"skills"'),
}

struct WorkflowDocument {
  1: required i32         schema_version (
    vt.ge = "1",
    go.tag = 'yaml:"schema_version"',
  ),
  2: required string      id             (
    vt.min_size = "1",
    go.tag = 'yaml:"id"',
  ),
  // field 3 retired: Workflow has no business version.
  4: required list<Stage> stages         (go.tag = 'yaml:"stages"'),
}

struct FlowDocument {
  1: required i32                         schema_version (
    vt.ge = "1",
    go.tag = 'yaml:"schema_version"',
  ),
  2: required map<string,list<FlowRoute>> flow           (go.tag = 'yaml:"flow"'),
}

struct ConditionDocument {
  1: required i32                             schema_version (
    vt.ge = "1",
    go.tag = 'yaml:"schema_version"',
  ),
  2: required map<string,ConditionDefinition> conditions     (go.tag = 'yaml:"conditions"'),
}

struct LoopDocument {
  1: required i32                         schema_version (
    vt.ge = "1",
    go.tag = 'yaml:"schema_version"',
  ),
  2: required map<string,list<LoopRoute>> loop           (go.tag = 'yaml:"loop"'),
}

struct PromptDocument {
  1: required i32                          schema_version (
    vt.ge = "1",
    go.tag = 'yaml:"schema_version"',
  ),
  2: required map<string,PromptDefinition> prompts        (go.tag = 'yaml:"prompts"'),
}
