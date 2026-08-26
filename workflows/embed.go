package workflows

import "embed"

// Files contains the immutable workflows shipped with this binary.
//
//go:embed */workflow.yaml */flow.yaml */condition.yaml */loop.yaml */prompt.yaml
var Files embed.FS
