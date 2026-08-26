include "ops.thrift"

namespace go releaseidl

const i32 RELEASE_MANIFEST_SCHEMA_VERSION = 2

struct CLIRelease {
  1: required string version (vt.pattern = "^[0-9A-Za-z][0-9A-Za-z._+-]*$"),
}

struct SkillArtifact {
  1: required string name    (vt.pattern = "^[a-z][a-z0-9-]*$"),
  2: required string version (vt.pattern = "^[0-9A-Za-z][0-9A-Za-z._+-]*$"),
  3: required string path    (vt.min_size = "1"),
  4: required string sha256  (vt.pattern = "^sha256:[0-9a-f]{64}$"),
}

struct WorkflowArtifact {
  1: required string id     (vt.pattern = "^[a-z][a-z0-9-]*$"),
  // fields 2 and 3 retired: Workflow has no business version or default.
  4: required string path   (vt.min_size = "1"),
  5: required string sha256 (vt.pattern = "^sha256:[0-9a-f]{64}$"),
}

struct PlatformAsset {
  1: required string os            (vt.pattern = "^(darwin|linux)$"),
  2: required string arch          (vt.pattern = "^(amd64|arm64)$"),
  3: required string file          (vt.min_size = "1"),
  4: required string sha256        (vt.pattern = "^sha256:[0-9a-f]{64}$"),
  5: required string binary_sha256 (vt.pattern = "^sha256:[0-9a-f]{64}$"),
}

struct ReleaseManifest {
  1: required i32                    schema_version  (vt.const = "2"),
  2: required string                 release_version (vt.pattern = "^[0-9A-Za-z][0-9A-Za-z._+-]*$"),
  3: required CLIRelease             cli             (vt.not_nil = "true"),
  4: required ops.StateSchemaSupport state_schema    (vt.not_nil = "true"),
  5: required list<SkillArtifact>    skills          (vt.min_size = "1"),
  6: required list<WorkflowArtifact> workflows       (vt.min_size = "1"),
  7: required list<PlatformAsset>    assets          (
    vt.min_size = "4",
    vt.max_size = "4",
  ),
}
