include "common.thrift"
include "error.thrift"

namespace go cardidl

enum CardView {
  unspecified = 0,
  current     = 1,
  panorama    = 2,
}

enum CardFormat {
  unspecified = 0,
  markdown    = 1,
  lark_json   = 2,
}

struct CardRenderRequest {
  1: required CardView   view   (
    vt.defined_only = "true",
    vt.not_in = "CardView.unspecified",
  ),
  2: required CardFormat format (
    vt.defined_only = "true",
    vt.not_in = "CardFormat.unspecified",
  ),
}

struct CardRenderResponse {
  1: required CardFormat       format        (
    vt.defined_only = "true",
    vt.not_in = "CardFormat.unspecified",
  ),
  2: required common.JsonValue content       (vt.not_nil = "true"),
  3: optional string           snapshot_path,
}

service CardService {
  CardRenderResponse Render(
    1: required string            requirement_root,
    2: required CardRenderRequest request,
    3: required bool              dry_run,
  ) throws (1: error.PublicError error) (
    cli.id = "card.render",
    cli.summary = "Render a deterministic Card",
    cli.risk = "local_write",
    cli.requirement_scope = "existing",
    cli.supports_dry_run = "true",
    cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,ROOT_REQUIRED,NOT_INITIALIZED,STATE_CORRUPT,STATE_SCHEMA_UNSUPPORTED,WORKFLOW_MISMATCH,LOCAL_COMMIT_FAILED,INTERNAL",
  )
}
