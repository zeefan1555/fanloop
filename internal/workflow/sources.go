package workflow

import (
	"io/fs"

	builtin "github.com/zeefan1555/commonloop/workflows"
)

func sources() []fs.FS { return []fs.FS{builtin.Files} }
