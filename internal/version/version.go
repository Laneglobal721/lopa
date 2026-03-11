package version

import (
	"fmt"
	"runtime"
)

// These variables are meant to be overridden via -ldflags.
//
// Example:
//
//	go build -ldflags "-X github.com/yanjiulab/lopa/internal/version.Version=v0.1.0 -X github.com/yanjiulab/lopa/internal/version.Commit=$(git rev-parse --short HEAD)"
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "unknown"
)

// String returns a human-friendly version string.
func String(app string) string {
	if app == "" {
		app = "lopa"
	}
	return fmt.Sprintf(
		"%s %s (commit=%s, date=%s, builtBy=%s, go=%s)",
		app,
		Version,
		Commit,
		Date,
		BuiltBy,
		runtime.Version(),
	)
}
