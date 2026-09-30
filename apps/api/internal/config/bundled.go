package config

import "strings"

// bundledTMDbKey is the TMDb key baked into release builds so a fresh install
// gets metadata without the owner applying for a TMDb account (sub-7-7a).
//
// It is EMPTY in source and in every local build. The Docker build injects it
// with
//
//	-ldflags "-X github.com/vido/api/internal/config.bundledTMDbKey=$KEY"
//
// where $KEY comes from a BuildKit secret mount (see the api-builder stage in
// the root Dockerfile and `.github/workflows/docker.yml`). It must never be
// read from an environment variable — that would make it indistinguishable
// from the user's own `TMDB_API_KEY` in logs and in the settings page.
//
// Resolution order is settings → env → bundled (services.KeyResolver), so a
// user who brings their own key is never routed through this one.
var bundledTMDbKey string

// BundledTMDbKey returns the build-time TMDb key, or "" when this binary was
// built without one (source checkout, `go run`, CI test builds).
func BundledTMDbKey() string {
	return strings.TrimSpace(bundledTMDbKey)
}

// HasBundledTMDbKey reports whether this binary carries a bundled TMDb key.
// Log THIS, never the key: it is the only thing support ever needs to know.
func HasBundledTMDbKey() bool {
	return BundledTMDbKey() != ""
}
