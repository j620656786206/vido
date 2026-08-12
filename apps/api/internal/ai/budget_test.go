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
	assert.InDelta(t, fallbackLLMPricing.InputPer1M, b.SpentUSD(), 1e-9)
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
