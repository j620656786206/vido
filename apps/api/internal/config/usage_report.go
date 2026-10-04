package config

import (
	"os"
	"strings"
)

// UsageReportEndpoint is the base URL of the receiver of the opt-in anonymous
// usage report (infra-optin-usage-report-a2) — the maintainer's self-hosted
// Umami. Like VIDO_VERSION it is set in the runtime stage of the release image
// from a CI build-arg, and is empty in every local build and fork, which makes
// the feature unavailable rather than pointing anywhere by default.
func UsageReportEndpoint() string {
	return strings.TrimRight(strings.TrimSpace(os.Getenv("VIDO_USAGE_REPORT_URL")), "/")
}

// UsageReportWebsiteID is the receiver's website id for Vido. Not a secret:
// every report carries it, and it only routes, it does not authenticate.
func UsageReportWebsiteID() string {
	return strings.TrimSpace(os.Getenv("VIDO_USAGE_REPORT_WEBSITE_ID"))
}
