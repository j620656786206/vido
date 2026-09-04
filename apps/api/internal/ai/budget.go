package ai

import (
	"context"
	"log/slog"
	"sync"
	"time"
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
	// Gemini rows (sub-5-1 AC #2): a Gemini call must never silently fall into
	// the Haiku-tier fallback and record a fabricated number. Verified
	// 2026-08-12 at https://ai.google.dev/gemini-api/docs/pricing.
	// NOTE: gemini-2.0-flash was SHUT DOWN by Google on 2026-06-01 and is no
	// longer the default (bugfix-gemini-default-model-retired bumped it to
	// gemini-2.5-flash-lite). The row STAYS: a deployment that pinned it via
	// GEMINI_MODEL must meter at its final published rate rather than inherit
	// the fallback tier and record a fabricated number.
	// Rates below re-verified 2026-08-24 at https://ai.google.dev/gemini-api/docs/pricing.
	"gemini-2.0-flash":      {InputPer1M: 0.10, OutputPer1M: 0.40},
	"gemini-2.5-flash":      {InputPer1M: 0.30, OutputPer1M: 2.50},
	"gemini-2.5-flash-lite": {InputPer1M: 0.10, OutputPer1M: 0.40},
	// 3.x rows exist because GEMINI_MODEL can now select them; the 3.6/3.7
	// figures are the promotional rate published through 2026-12-31.
	"gemini-3.5-flash-lite": {InputPer1M: 0.30, OutputPer1M: 2.50},
	"gemini-3.6-flash":      {InputPer1M: 0.75, OutputPer1M: 3.75},
	"gemini-3.7-flash":      {InputPer1M: 0.75, OutputPer1M: 3.75},
}

// llmTimeoutBase is the per-family request-timeout base RequestTimeoutFor
// (timeout.go) starts from — kept HERE, beside the pricing table, because both
// are "what do we know about this model id" and a model added to one belongs
// in the other (sub-6-2 AC #1). Matched by substring, first row wins; the
// values are the observed p99 of a 10-cue chunk on each family, with headroom.
// Vars (not consts) so the package test suite can shrink them the way it
// shrinks the retry backoff.
var llmTimeoutBase = []modelTimeoutRow{
	{family: "haiku", base: 30 * time.Second},
	{family: "sonnet", base: 60 * time.Second},
	{family: "opus", base: 90 * time.Second},
	{family: "gemini", base: 30 * time.Second},
}

// unknownModelTimeoutBase is the Sonnet-class base an unlisted model id gets.
var unknownModelTimeoutBase = 60 * time.Second

// modelTimeoutRow is one family → base pair of llmTimeoutBase.
type modelTimeoutRow struct {
	family string
	base   time.Duration
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

// PricingFor exposes the same table the retrospective metering uses, so a
// FORWARD cost estimate (story sub-4-1) quotes exactly what the run will later
// be billed at. Deliberately a read-through to llmPricing rather than a second
// table: two copies of a price list drift, and the drift shows up as a quote
// the invoice disagrees with.
func PricingFor(model string) ModelPricing { return llmPricing(model) }

// HostedASRPerMinuteUSD is the per-audio-minute price of the hosted Whisper
// API — the rate RecordASR bills at.
//
// ⚠️ It applies to the HOSTED endpoint only. A deployment pointing ASR_BASE_URL
// at a self-hosted server (Speaches, faster-whisper, …) pays nothing per
// minute, so an estimator must not apply this rate blindly — see
// EstimatedASRPerMinuteUSD.
func HostedASRPerMinuteUSD() float64 { return whisperPerMinuteUSD }

// EstimatedASRPerMinuteUSD returns the rate a cost ESTIMATE should use for the
// currently-wired ASR endpoint: 0 when the endpoint is self-hosted, the hosted
// price otherwise.
//
// selfHosted is the caller's answer to "is ASR_BASE_URL pointed somewhere other
// than the paid API?" — config lives above this package, so the decision is
// passed in rather than read here.
//
// This is ALSO the metering side's rate resolver (sub-5-1 AC #1): the Whisper
// client passes EstimatedASRPerMinuteUSD(isSelfHosted) to RecordASRWithRate, so
// the estimate and the retrospective spend figure are the same number by
// construction.
func EstimatedASRPerMinuteUSD(selfHosted bool) float64 {
	if selfHosted {
		return 0
	}
	return whisperPerMinuteUSD
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

// RecordASR adds one ASR (Whisper) call's audio-minute cost at the HOSTED API
// rate. Kept as a thin delegate so pre-sub-5-1 callers and tests keep their
// exact semantics; rate-aware callers use RecordASRWithRate.
func (b *Budget) RecordASR(audioSeconds float64) {
	b.RecordASRWithRate(audioSeconds, whisperPerMinuteUSD)
}

// RecordASRWithRate adds one ASR call's audio-minute cost at an explicit
// per-minute rate (sub-5-1 AC #1). The rate comes from
// EstimatedASRPerMinuteUSD so estimate and metering can never disagree — a
// self-hosted endpoint records $0 while still accruing asrSeconds/asrCalls for
// observability.
func (b *Budget) RecordASRWithRate(audioSeconds, perMinuteUSD float64) {
	if b == nil {
		return
	}
	cost := audioSeconds / 60.0 * perMinuteUSD
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
