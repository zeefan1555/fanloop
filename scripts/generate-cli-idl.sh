#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$ROOT"
OUTPUT_ROOT="${OUTPUT_ROOT:-$ROOT}"
IDL_GO_ROOT="$OUTPUT_ROOT/internal/idl"

THRIFTGO="${THRIFTGO:-thriftgo}"
THRIFT_VALIDATOR="${THRIFT_VALIDATOR:-thrift-gen-validator}"

if [ "$("$THRIFTGO" --version 2>&1)" != "thriftgo 0.4.3" ]; then
  echo "thriftgo v0.4.3 is required" >&2
  exit 1
fi
if [ "$("$THRIFT_VALIDATOR" --version 2>&1)" != "v0.2.5" ]; then
  echo "thrift-gen-validator v0.2.5 is required" >&2
  exit 1
fi

mkdir -p "$IDL_GO_ROOT"
"$THRIFTGO" --check-keywords -r \
  -g go:package_prefix=github.com/zeefan1555/fanloop/internal/idl,naming_style=golint,json_enum_as_text,snake_style_json_tag,always_gen_json_tag,gen_deep_equal,reserve_comments,no_default_serdes,no_processor \
  -p "validator=$THRIFT_VALIDATOR" \
  -o "$IDL_GO_ROOT" \
  idl/cli.thrift

go run ./tools/command-spec-gen -input idl/cli.thrift -output "$OUTPUT_ROOT/internal/idl/commands_gen.go"
gofmt -w "$IDL_GO_ROOT"/commonidl "$IDL_GO_ROOT"/erroridl "$IDL_GO_ROOT"/flowidl \
  "$IDL_GO_ROOT"/traceidl "$IDL_GO_ROOT"/cardidl "$IDL_GO_ROOT"/opsidl "$IDL_GO_ROOT"/storageidl "$IDL_GO_ROOT"/yamlidl "$IDL_GO_ROOT"/cliidl \
  "$IDL_GO_ROOT"/releaseidl \
  "$OUTPUT_ROOT"/internal/idl/commands_gen.go
