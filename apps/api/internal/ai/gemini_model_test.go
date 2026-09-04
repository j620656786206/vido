package ai

// Story bugfix-gemini-default-model-retired — Google shut gemini-2.0-flash down on
// 2026-06-01, and it had been vido's DefaultGeminiModel ever since, with no
// GEMINI_MODEL escape hatch. These tests pin both halves of the fix so a future
// retirement is caught by a red test rather than by 404s in production.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultGeminiModelIsNotTheRetiredOne(t *testing.T) {
	assert.NotEqual(t, "gemini-2.0-flash", DefaultGeminiModel,
		"gemini-2.0-flash was shut down 2026-06-01 — a default pointing at it 404s every call")
	assert.Equal(t, "gemini-2.5-flash-lite", DefaultGeminiModel,
		"verified present on ai.google.dev/gemini-api/docs/pricing on 2026-08-24, at the same price tier as the retired default")
}

func TestDefaultGeminiModelIsPricedNotFallenBackOn(t *testing.T) {
	// A default the pricing table does not know would meter at the fallback tier
	// and record a number the invoice disagrees with.
	pricing, ok := defaultLLMPricing[DefaultGeminiModel]
	require.True(t, ok, "the default model must have an explicit pricing row")
	assert.NotEqual(t, fallbackLLMPricing, pricing,
		"the default must not be metered at the catch-all fallback rate")
}

func TestNewGeminiProviderUsesTheDefaultModel(t *testing.T) {
	p := NewGeminiProvider("key")
	assert.Equal(t, DefaultGeminiModel, p.model)
}

func TestWithGeminiModelOverridesTheDefault(t *testing.T) {
	p := NewGeminiProvider("key", WithGeminiModel("gemini-3.7-flash"))
	assert.Equal(t, "gemini-3.7-flash", p.model)
}

func TestFactoryAppliesGeminiModelOverride(t *testing.T) {
	provider, err := NewProvider(FactoryConfig{
		ProviderName: string(ProviderGemini),
		GeminiAPIKey: "key",
		GeminiModel:  "gemini-2.5-flash",
	})
	require.NoError(t, err)

	gemini, ok := provider.(*GeminiProvider)
	require.True(t, ok, "expected a *GeminiProvider")
	assert.Equal(t, "gemini-2.5-flash", gemini.model,
		"a GEMINI_MODEL override that never reaches the provider is a silent no-op (Rule 15)")
}

func TestFactoryFallsBackToDefaultWhenNoGeminiModelSet(t *testing.T) {
	provider, err := NewProvider(FactoryConfig{
		ProviderName: string(ProviderGemini),
		GeminiAPIKey: "key",
	})
	require.NoError(t, err)

	gemini, ok := provider.(*GeminiProvider)
	require.True(t, ok, "expected a *GeminiProvider")
	assert.Equal(t, DefaultGeminiModel, gemini.model)
}

// Every model a GEMINI_MODEL override can realistically select needs a real row,
// otherwise metering silently reports the fallback tier as fact.
func TestGeminiPricingRowsAreExplicit(t *testing.T) {
	tests := []struct {
		model  string
		input  string
		output string
		why    string
	}{
		{"gemini-2.5-flash-lite", "0.10", "0.40", "current default"},
		{"gemini-2.5-flash", "0.30", "2.50", ""},
		{"gemini-3.5-flash-lite", "0.30", "2.50", ""},
		{"gemini-3.6-flash", "0.75", "3.75", "promotional rate through 2026-12-31"},
		{"gemini-3.7-flash", "0.75", "3.75", "promotional rate through 2026-12-31"},
		{
			model: "gemini-2.0-flash", input: "0.10", output: "0.40",
			why: "retired 2026-06-01 but kept: a deployment that pinned it must meter at its final published rate, not inherit the fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got, ok := defaultLLMPricing[tt.model]
			require.Truef(t, ok, "missing pricing row for %s (%s)", tt.model, tt.why)
			assert.Equal(t, price(tt.input, tt.output), got)
			assert.NotEqual(t, fallbackLLMPricing, got, "must not resolve to the catch-all tier")
		})
	}
}

// The fallback being dearer than every real Gemini rate is deliberate: an unknown
// model over-counts rather than under-counts.
func TestFallbackTierIsNotCheaperThanAnyGeminiRow(t *testing.T) {
	for model, p := range defaultLLMPricing {
		if len(model) < 6 || model[:6] != "gemini" {
			continue
		}
		assert.Truef(t, p.InputPer1M.LessThanOrEqual(fallbackLLMPricing.InputPer1M),
			"%s input rate exceeds the fallback, so an unknown model would under-count", model)
		assert.Truef(t, p.OutputPer1M.LessThanOrEqual(fallbackLLMPricing.OutputPer1M),
			"%s output rate exceeds the fallback, so an unknown model would under-count", model)
	}
}
