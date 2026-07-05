package ai

import (
	"context"
	"log/slog"
	"sync"
)

// ModelPricing is the per-1M-token USD price for an LLM model (Story 9R-11
// metering). Whisper is priced separately per audio minute.
type ModelPricing struct {
	InputPer1M  float64
	OutputPer1M float64
}

// defaultLLMPricing holds published USD/1M-token prices for the models vido may
// use. Unknown models fall back to the Haiku tier so metering never silently
// under-counts. Update alongside DefaultClaudeModel.
var defaultLLMPricing = map[string]ModelPricing{
	"claude-haiku-4-5":  {InputPer1M: 1.0, OutputPer1M: 5.0},
	"claude-sonnet-5":   {InputPer1M: 3.0, OutputPer1M: 15.0},
	"claude-opus-4-8":   {InputPer1M: 5.0, OutputPer1M: 25.0},
	"claude-sonnet-4-6": {InputPer1M: 3.0, OutputPer1M: 15.0},
}

// fallbackLLMPricing is used when the model id isn't in the table.
var fallbackLLMPricing = ModelPricing{InputPer1M: 1.0, OutputPer1M: 5.0}

// whisperPerMinuteUSD is the OpenAI Whisper API price per audio minute.
const whisperPerMinuteUSD = 0.006

func llmPricing(model string) ModelPricing {
	if p, ok := defaultLLMPricing[model]; ok {
		return p
	}
	return fallbackLLMPricing
}

// Budget meters token usage and cost for one batch run and enforces an optional
// USD ceiling (Story 9R-11 AC #2). It is created per run (per transcription /
// translation job) and carried through the call chain via context, so a batch
// over many files shares one ceiling. A nil Budget means "no metering / no cap".
type Budget struct {
	maxUSD float64 // 0 = unlimited

	mu           sync.Mutex
	spentUSD     float64
	inputTokens  int64
	outputTokens int64
	llmCalls     int
	asrSeconds   float64
	asrCalls     int
}

// NewBudget creates a per-run budget with an optional USD ceiling (<=0 means no
// ceiling — metering still accrues, but Exceeded is always false).
func NewBudget(maxUSD float64) *Budget {
	return &Budget{maxUSD: maxUSD}
}

// Exceeded reports whether the accrued spend has reached the ceiling.
func (b *Budget) Exceeded() bool {
	if b == nil || b.maxUSD <= 0 {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.spentUSD >= b.maxUSD
}

// RecordLLM adds one LLM call's token usage + cost.
func (b *Budget) RecordLLM(model string, inputTokens, outputTokens int64) {
	if b == nil {
		return
	}
	p := llmPricing(model)
	cost := float64(inputTokens)/1_000_000*p.InputPer1M + float64(outputTokens)/1_000_000*p.OutputPer1M
	b.mu.Lock()
	b.inputTokens += inputTokens
	b.outputTokens += outputTokens
	b.spentUSD += cost
	b.llmCalls++
	spent := b.spentUSD
	b.mu.Unlock()
	slog.Info("AI usage recorded (LLM)",
		"model", model, "input_tokens", inputTokens, "output_tokens", outputTokens,
		"call_cost_usd", cost, "run_spent_usd", spent, "run_budget_usd", b.maxUSD,
	)
}

// RecordASR adds one ASR (Whisper) call's audio-minute cost.
func (b *Budget) RecordASR(audioSeconds float64) {
	if b == nil {
		return
	}
	cost := audioSeconds / 60.0 * whisperPerMinuteUSD
	b.mu.Lock()
	b.asrSeconds += audioSeconds
	b.spentUSD += cost
	b.asrCalls++
	spent := b.spentUSD
	b.mu.Unlock()
	slog.Info("AI usage recorded (ASR)",
		"audio_seconds", audioSeconds, "call_cost_usd", cost,
		"run_spent_usd", spent, "run_budget_usd", b.maxUSD,
	)
}

// SpentUSD returns the accrued spend.
func (b *Budget) SpentUSD() float64 {
	if b == nil {
		return 0
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.spentUSD
}

// BudgetSnapshot is a point-in-time view of a run's metering.
type BudgetSnapshot struct {
	SpentUSD     float64
	BudgetUSD    float64
	InputTokens  int64
	OutputTokens int64
	LLMCalls     int
	ASRSeconds   float64
	ASRCalls     int
}

// Snapshot returns the current metering totals for logging/reporting.
func (b *Budget) Snapshot() BudgetSnapshot {
	if b == nil {
		return BudgetSnapshot{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return BudgetSnapshot{
		SpentUSD: b.spentUSD, BudgetUSD: b.maxUSD,
		InputTokens: b.inputTokens, OutputTokens: b.outputTokens, LLMCalls: b.llmCalls,
		ASRSeconds: b.asrSeconds, ASRCalls: b.asrCalls,
	}
}

// budgetCtxKey plumbs a per-run Budget through the call chain without changing
// every method signature.
type budgetCtxKey struct{}

// WithBudget attaches a per-run Budget to ctx.
func WithBudget(ctx context.Context, b *Budget) context.Context {
	return context.WithValue(ctx, budgetCtxKey{}, b)
}

// BudgetFromContext returns the Budget on ctx, or nil if none.
func BudgetFromContext(ctx context.Context) *Budget {
	if b, ok := ctx.Value(budgetCtxKey{}).(*Budget); ok {
		return b
	}
	return nil
}
