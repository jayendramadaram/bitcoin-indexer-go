package path

import (
	"path/filepath"
	"runtime"
)

var (
	_, b, _, _               = runtime.Caller(0)
	ProjectRoot              = filepath.Join(filepath.Dir(b), "../")
	DefaultConfigPath string = filepath.Join(ProjectRoot, "config", "config.toml")
)
