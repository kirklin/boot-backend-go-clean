// Package version provides build-time version information.
// Values are injected via -ldflags at compile time.
//
//	go build -ldflags="-X github.com/kirklin/boot-backend-go-clean/pkg/version.Version=1.0.0"
package version

// These variables are set at build time via -ldflags.
var (
	// Version is the semantic version of the application (e.g. "1.2.3").
	Version = "dev"

	// GitCommit is the git commit hash of the build.
	GitCommit = "unknown"

	// BuildTime is the UTC timestamp of the build.
	BuildTime = "unknown"
)
