package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsageReportConfig_EmptyByDefault(t *testing.T) {
	t.Setenv("VIDO_USAGE_REPORT_URL", "")
	t.Setenv("VIDO_USAGE_REPORT_WEBSITE_ID", "")
	assert.Equal(t, "", UsageReportEndpoint())
	assert.Equal(t, "", UsageReportWebsiteID())
}

func TestUsageReportConfig_TrimsWhitespaceAndTrailingSlash(t *testing.T) {
	t.Setenv("VIDO_USAGE_REPORT_URL", " https://analytics.example.com/ \n")
	t.Setenv("VIDO_USAGE_REPORT_WEBSITE_ID", " 11111111-2222-3333-4444-555555555555 ")
	assert.Equal(t, "https://analytics.example.com", UsageReportEndpoint())
	assert.Equal(t, "11111111-2222-3333-4444-555555555555", UsageReportWebsiteID())
}
