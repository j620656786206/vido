package ai

// Story sub-4-1 AC #6 — the forward cost estimate must quote the SAME rates the
// run is later billed at. These tests pin that the exported accessors are a
// read-through to the metering table, not a second copy of it.

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPricingFor_MatchesTheMeteringTable(t *testing.T) {
	for model, want := range defaultLLMPricing {
		assert.Equalf(t, want, PricingFor(model),
			"PricingFor(%q) must return exactly what RecordLLM bills at — a divergent quote is an invoice the user did not agree to", model)
	}
}

func TestPricingFor_UnknownModelUsesTheSameFallbackAsMetering(t *testing.T) {
	assert.Equal(t, fallbackLLMPricing, PricingFor("some-model-shipped-after-this-test"))
}

func TestHostedASRRate_MatchesWhatRecordASRBills(t *testing.T) {
	// RecordASR computes seconds/60*whisperPerMinuteUSD; one minute of audio is
	// therefore exactly the hosted rate.
	b := NewBudget(0)
	b.RecordASR(60)
	assert.InDelta(t, HostedASRPerMinuteUSD(), b.SpentUSD(), 1e-9,
		"the estimator's rate and the meter's rate must be the same number")
}

// The estimate must not bill a self-hosted endpoint at the hosted API's price —
// the spike ran faster-whisper locally at zero marginal cost, and quoting
// ~$0.27 per episode for that would be a fabricated invoice.
func TestEstimatedASRRate_IsZeroForSelfHosted(t *testing.T) {
	assert.Zero(t, EstimatedASRPerMinuteUSD(true))
	assert.Equal(t, HostedASRPerMinuteUSD(), EstimatedASRPerMinuteUSD(false))
}
