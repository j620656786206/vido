package config

import "strings"

// buildVersion is the release version baked in at build time
// (infra-optin-usage-report-a1). The Docker build injects it with
//
//	-ldflags "-X github.com/vido/api/internal/config.buildVersion=$VIDO_VERSION"
//
// where $VIDO_VERSION is docker/metadata-action's version output: the semver
// of a `v*.*.*` tag without the leading v (e.g. "0.1.2"), or the branch name
// for a branch build. EMPTY in source and every local build.
var buildVersion string

// Version returns the version this binary was built as, or "dev" when it was
// built without one (source checkout, `go run`, CI test builds).
func Version() string {
	if v := strings.TrimSpace(buildVersion); v != "" {
		return v
	}
	return "dev"
}
