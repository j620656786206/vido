package config

import (
	"os"
	"strings"
)

// Version returns the version this Vido build reports
// (infra-optin-usage-report-a1), or "dev" when none was set (source
// checkout, `go run`, CI test builds).
//
// It is read from the VIDO_VERSION environment variable, which the runtime
// stage of the Docker image sets from a build-arg: the semver of a `v*.*.*`
// tag ("0.1.2"), or "<branch>-<short sha>" for a branch build ("main-81a4e5c")
// — `latest` is built from main, so a bare "main" would tell every NAS apart
// from nothing. An env var rather than an -ldflags -X value on purpose: a
// value baked into the Go binary changes on every commit and would make the
// Go build layer miss the BuildKit cache on every build, frontend-only PRs
// included; an ENV line in the final stage costs nothing to rebuild.
func Version() string {
	if v := strings.TrimSpace(os.Getenv("VIDO_VERSION")); v != "" {
		return v
	}
	return "dev"
}
