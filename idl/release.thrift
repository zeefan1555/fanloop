include "ops.thrift"

namespace go releaseidl

const i32 RELEASE_MANIFEST_SCHEMA_VERSION = 4

struct CLIRelease {
  1: required string version (vt.pattern = "^[0-9A-Za-z][0-9A-Za-z._+-]*$"),
  2: required string binary_sha256 (vt.pattern = "^sha256:[0-9a-f]{64}$"),
}

struct SkillArtifact {
  1: required string name    (vt.pattern = "^[a-z][a-z0-9-]*$"),
  2: required string version (vt.pattern = "^[0-9A-Za-z][0-9A-Za-z._+-]*$"),
  3: required string path    (vt.min_size = "1"),
  4: required string sha256  (vt.pattern = "^sha256:[0-9a-f]{64}$"),
}

struct ReleaseManifest {
  1: required i32                    schema_version  (vt.const = "4"),
  2: required string                 release_version (vt.pattern = "^[0-9A-Za-z][0-9A-Za-z._+-]*$"),
  3: required CLIRelease             cli             (vt.not_nil = "true"),
  4: required ops.StateSchemaSupport state_schema    (vt.not_nil = "true"),
  5: required list<SkillArtifact>    skills          (vt.min_size = "1"),
  // field 6 retired: Workflows are live configuration outside the Release.
  // field 7 retired: local builds do not contain distribution archives.
}
