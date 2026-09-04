package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBudget_RecordLLM_CostAndTokens(t *testing.T) {
	b := NewBudget(0) // unlimited
	// Haiku: $1/1M in, $5/1M out. 500k in + 200k out = 0.5 + 1.0 = $1.50.
	b.RecordLLM("claude-haiku-4-5", 500_000, 200_000)
	snap := b.Snapshot()
	assert.InDelta(t, 1.5, snap.SpentUSD, 1e-9)
	assert.Equal(t, int64(500_000), snap.InputTokens)
	assert.Equal(t, int64(200_000), snap.OutputTokens)
	assert.Equal(t, 1, snap.LLMCalls)
	assert.False(t, b.Exceeded(), "unlimited budget is never exceeded")
}

func TestBudget_UnknownModelUsesFallback(t *testing.T) {
	b := NewBudget(0)
	b.RecordLLM("some-future-model", 1_000_000, 0)
	assert.InDelta(t, fallbackLLMPricing.InputPer1M.InexactFloat64(), b.SpentUSD(), 1e-9)
}

// ─── tech-money-decimal-arithmetic ────────────────────────────────────────

func TestBudget_SpendIsExactAcrossManyCalls(t *testing.T) {
	// gemini-2.5-flash-lite is $0.10 per 1M input tokens, so each of these
	// calls costs exactly $0.10. In float64, 0.1+0.1+0.1 is
	// 0.30000000000000004 — the canonical demonstration that binary doubles
	// cannot hold a tenth. The ledger must hold the number the provider bills.
	b := NewBudget(0)
	for range 3 {
		b.RecordLLM("gemini-2.5-flash-lite", 1_000_000, 0)
	}
	assert.Equal(t, "0.3", b.Spent().String(),
		"three $0.10 calls are thirty cents, not 0.30000000000000004")
}

func TestBudget_CeilingIsNotSkippedByFloatDrift(t *testing.T) {
	// CR H2: the first version of this test used $0.10 + $0.70 against a $0.80
	// ceiling and claimed float64 would miss it. It would NOT — for those
	// rates the two roundings cancel and the old code landed on 0.80000000…04,
	// bit-identical to the ceiling, so the test passed against the code it was
	// supposed to indict. A test that cannot fail on the old implementation is
	// not evidence of anything.
	//
	// These numbers were found by search and DO discriminate:
	// gemini-3.6-flash bills $0.75/1M, so 300k + 300k tokens is exactly $0.45.
	// The old `float64(tokens)/1e6*rate` accumulation gives
	// 0.44999999999999995559 — strictly BELOW the ceiling, so `spent >= max`
	// reads false and the run bills another call the user never approved.
	b := NewBudget(0.45)
	b.RecordLLM("gemini-3.6-flash", 300_000, 0)
	b.RecordLLM("gemini-3.6-flash", 300_000, 0)

	assert.Equal(t, "0.45", b.Spent().String(),
		"two 300k-token calls at $0.75/1M are forty-five cents, exactly")
	assert.True(t, b.Exceeded(),
		"the ceiling was reached exactly — a run that continues here spends more than was approved")

	// The float64 arithmetic the old ledger used, spelled out, so the reason
	// this test exists cannot rot into folklore.
	var oldFloat float64
	oldFloat += float64(300_000) / 1_000_000 * 0.75
	oldFloat += float64(300_000) / 1_000_000 * 0.75
	assert.Less(t, oldFloat, 0.45,
		"if this ever stops being true the test above has lost its teeth")
}

func TestBudget_SubCentPerTokenCostIsNotRoundedAway(t *testing.T) {
	// A single token on Sonnet costs $0.000003. Anything that meters in whole
	// cents records $0.00 for it, and a run made of a million such calls then
	// reports having spent nothing at all.
	b := NewBudget(0)
	b.RecordLLM("claude-sonnet-5", 1, 0)
	assert.Equal(t, "0.000003", b.Spent().String())
}

func TestBudget_RecordASR_ByMinutes(t *testing.T) {
	b := NewBudget(0)
	b.RecordASR(120) // 2 min * $0.006 = $0.012
	snap := b.Snapshot()
	assert.InDelta(t, 0.012, snap.SpentUSD, 1e-9)
	assert.InDelta(t, 120, snap.ASRSeconds, 1e-9)
	assert.Equal(t, 1, snap.ASRCalls)
}

func TestBudget_Exceeded(t *testing.T) {
	b := NewBudget(1.0)
	assert.False(t, b.Exceeded())
	b.RecordLLM("claude-haiku-4-5", 1_000_000, 0) // $1.00 → at ceiling
	assert.True(t, b.Exceeded())
}

func TestBudget_NilSafe(t *testing.T) {
	var b *Budget
	assert.False(t, b.Exceeded())
	b.RecordLLM("x", 1, 1) // no panic
	b.RecordASR(1)         // no panic
	assert.Equal(t, float64(0), b.SpentUSD())
	assert.Equal(t, BudgetSnapshot{}, b.Snapshot())
}

func TestBudget_ContextPlumbing(t *testing.T) {
	assert.Nil(t, BudgetFromContext(context.Background()))
	b := NewBudget(5.0)
	ctx := WithBudget(context.Background(), b)
	assert.Same(t, b, BudgetFromContext(ctx))
}

// --- sub-5-1 AC #1: rate-aware ASR metering (費率同源) ---

func TestBudget_RecordASRWithRate_SelfHostedRateIsZeroSpend(t *testing.T) {
	b := NewBudget(0)
	b.RecordASRWithRate(600, EstimatedASRPerMinuteUSD(true)) // self-hosted → $0/min
	snap := b.Snapshot()
	assert.Zero(t, snap.SpentUSD,
		"a self-hosted ASR run must record $0 — the hosted rate here would be a fabricated bill")
	// Observability still accrues: the run happened, it just cost nothing.
	assert.InDelta(t, 600, snap.ASRSeconds, 1e-9)
	assert.Equal(t, 1, snap.ASRCalls)
	assert.False(t, b.Exceeded())
}

func TestBudget_RecordASRWithRate_HostedRateMatchesRecordASR(t *testing.T) {
	// 同源斷言: RecordASR (the legacy delegate) and RecordASRWithRate fed the
	// estimator's hosted rate must record the identical spend.
	legacy, rated := NewBudget(0), NewBudget(0)
	legacy.RecordASR(120)
	rated.RecordASRWithRate(120, EstimatedASRPerMinuteUSD(false))
	assert.InDelta(t, legacy.SpentUSD(), rated.SpentUSD(), 1e-9)
	assert.InDelta(t, 0.012, rated.SpentUSD(), 1e-9) // 2 min * $0.006
}

func TestBudget_RecordASRWithRate_NilSafe(t *testing.T) {
	var b *Budget
	b.RecordASRWithRate(60, 0.006) // no panic
	assert.Zero(t, b.SpentUSD())
}
