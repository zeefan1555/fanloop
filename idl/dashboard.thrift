include "error.thrift"

namespace go dashboardidl

struct DashboardStartRequest {}

struct DashboardStartResponse {
  1: required string url,
}

service DashboardService {
  DashboardStartResponse Start(
    1: required DashboardStartRequest request,
  ) throws (1: error.PublicError error) (
    cli.id = "dashboard",
    cli.summary = "Start the local Workflow configuration dashboard",
    cli.risk = "local_write",
    cli.requirement_scope = "none",
    cli.supports_dry_run = "false",
      cli.errors = "INVALID_ARGUMENT,INVALID_INPUT_JSON,INTERNAL",
  )
}
