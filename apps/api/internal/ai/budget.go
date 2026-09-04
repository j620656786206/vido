package ai

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// ModelPricing is the per-1M-token USD price for an LLM model (Story 9R-11
// metering). Whisper is priced separately per audio minute.
//
// tech-money-decimal-arithmetic: these are DECIMALS built from decimal
// STRINGS, never float64 literals. `0.30` has no exact binary representation,
// so a float64 rate is already ~1e-17 off the published price before a single
// token is counted — and this table multiplies by token counts in the millions
// and accumulates across thousands of calls per run. The rate a provider
// publishes is a decimal figure; storing it as one is the only way the number
// we meter is the number they charge.
type ModelPricing struct {
	InputPer1M  decimal.Decimal
	OutputPer1M decimal.Decimal
}

// price builds a ModelPricing from the published decimal strings. Panics on a
// malformed literal, which is what you want for a compile-time-ish table: a
// typo in a price must never boot.
func price(inputPer1M, outputPer1M string) ModelPricing {
	return ModelPricing{
		InputPer1M:  decimal.RequireFromString(inputPer1M),
		OutputPer1M: decimal.RequireFromString(outputPer1M),
	}
}

// perMillion is the 10^-6 shift that turns "price per 1M tokens" into "price
// per token". Applied with decimal.Shift, which moves the exponent and is
// therefore EXACT — unlike dividing by 1e6, which would round at
// decimal.DivisionPrecision.
const perMillionShift = -6

// defaultLLMPricing holds published USD/1M-token prices for the models vido may
// use. Unknown models fall back to the Haiku tier so metering never silently
// under-counts. Update alongside DefaultClaudeModel.
var defaultLLMPricing = map[string]ModelPricing{
	"claude-haiku-4-5":  price("1.00", "5.00"),
	"claude-sonnet-5":   price("3.00", "15.00"),
	"claude-opus-4-8":   price("5.00", "25.00"),
	"claude-sonnet-4-6": price("3.00", "15.00"),
	// Gemini rows (sub-5-1 AC #2): a Gemini call must never silently fall into
	// the Haiku-tier fallback and record a fabricated number. Verified
	// 2026-08-12 at https://ai.google.dev/gemini-api/docs/pricing.
	// NOTE: gemini-2.0-flash was SHUT DOWN by Google on 2026-06-01 and is no
	// longer the default (bugfix-gemini-default-model-retired bumped it to
	// gemini-2.5-flash-lite). The row STAYS: a deployment that pinned it via
	// GEMINI_MODEL must meter at its final published rate rather than inherit
	// the fallback tier and record a fabricated number.
	// Rates below re-verified 2026-08-24 at https://ai.google.dev/gemini-api/docs/pricing.
	"gemini-2.0-flash":      price("0.10", "0.40"),
	"gemini-2.5-flash":      price("0.30", "2.50"),
	"gemini-2.5-flash-lite": price("0.10", "0.40"),
	// 3.x rows exist because GEMINI_MODEL can now select them; the 3.6/3.7
	// figures are the promotional rate published through 2026-12-31.
	"gemini-3.5-flash-lite": price("0.30", "2.50"),
	"gemini-3.6-flash":      price("0.75", "3.75"),
	"gemini-3.7-flash":      price("0.75", "3.75"),
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
var fallbackLLMPricing = price("1.00", "5.00")

// whisperPerMinuteUSD is the OpenAI Whisper API price per audio minute.
var whisperPerMinuteUSD = decimal.RequireFromString("0.006")

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

// HasPricing reports whether a model has a REAL row in the table, as opposed
// to silently inheriting the Haiku-tier fallback. An estimator must be able to
// tell the two apart: scaling a quote by a fallback price would quote an
// unknown (possibly premium) model at the cheapest tier, and a quote that
// surprises upward is worse than no quote.
func HasPricing(model string) bool {
	_, ok := defaultLLMPricing[model]
	return ok
}

// HostedASRPerMinuteUSD is the per-audio-minute price of the hosted Whisper
// API — the rate RecordASR bills at.
//
// ⚠️ It applies to the HOSTED endpoint only. A deployment pointing ASR_BASE_URL
// at a self-hosted server (Speaches, faster-whisper, …) pays nothing per
// minute, so an estimator must not apply this rate blindly — see
// EstimatedASRPerMinuteUSD.
func HostedASRPerMinuteUSD() float64 { return whisperPerMinuteUSD.InexactFloat64() }

// HostedASRPerMinute is the same rate as an exact decimal, for the metering
// path. The float64 twin above stays for estimators and the wire.
func HostedASRPerMinute() decimal.Decimal { return whisperPerMinuteUSD }

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
	return EstimatedASRPerMinute(selfHosted).InexactFloat64()
}

// EstimatedASRPerMinute is the exact-decimal form of EstimatedASRPerMinuteUSD.
func EstimatedASRPerMinute(selfHosted bool) decimal.Decimal {
	if selfHosted {
		return decimal.Zero
	}
	return whisperPerMinuteUSD
}

// Budget meters token usage and cost for one batch run and enforces an optional
// USD ceiling (Story 9R-11 AC #2). It is created per run (per transcription /
// translation job) and carried through the call chain via context, so a batch
// over many files shares one ceiling. A nil Budget means "no metering / no cap".
type Budget struct {
	maxUSD decimal.Decimal // zero = unlimited

	mu           sync.Mutex
	spentUSD     decimal.Decimal
	inputTokens  int64
	outputTokens int64
	llmCalls     int
	asrSeconds   float64
	asrCalls     int
}

// NewBudget creates a per-run budget with an optional USD ceiling (<=0 means no
// ceiling — metering still accrues, but Exceeded is always false).
func NewBudget(maxUSD float64) *Budget {
	// The ceiling arrives as a float64 from config / the consent screen.
	// NewFromFloat takes the SHORTEST decimal that round-trips that float, so
	// a user who typed 5.00 gets exactly 5, not 4.99999999999999911182…
	//
	// ⚠️ CR H1: NewFromFloat PANICS on NaN/±Inf, and `strconv.ParseFloat`
	// accepts the literal strings "NaN"/"Inf" without error — so a typo'd
	// AI_RUN_BUDGET_USD used to be inert (a NaN ceiling silently never
	// triggered) and would instead have taken the whole process down: this
	// constructor runs inside WorkerPool's bare `go func`, which has no
	// recover(), so gin's Recovery middleware never sees it. A misconfigured
	// env var must not be able to kill the backend.
	if math.IsNaN(maxUSD) || math.IsInf(maxUSD, 0) {
		slog.Warn("AI_RUN_BUDGET_USD is not a finite number — running WITHOUT a ceiling",
			"value", maxUSD)
		return &Budget{}
	}
	return &Budget{maxUSD: decimal.NewFromFloat(maxUSD)}
}

// Exceeded reports whether the accrued spend has reached the ceiling.
func (b *Budget) Exceeded() bool {
	if b == nil || !b.maxUSD.IsPositive() {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.spentUSD.GreaterThanOrEqual(b.maxUSD)
}

// RecordLLM adds one LLM call's token usage + cost.
func (b *Budget) RecordLLM(model string, inputTokens, outputTokens int64) {
	if b == nil {
		return
	}
	p := llmPricing(model)
	// EXACT: multiply token counts by the published per-1M price, then shift
	// the decimal point six places. No division, so nothing rounds — the cost
	// of a call is the same figure the provider's own invoice arithmetic
	// produces, at any token count.
	cost := decimal.NewFromInt(inputTokens).Mul(p.InputPer1M).
		Add(decimal.NewFromInt(outputTokens).Mul(p.OutputPer1M)).
		Shift(perMillionShift)
	b.mu.Lock()
	b.inputTokens += inputTokens
	b.outputTokens += outputTokens
	b.spentUSD = b.spentUSD.Add(cost)
	b.llmCalls++
	spent := b.spentUSD
	b.mu.Unlock()
	slog.Info("AI usage recorded (LLM)",
		"model", model, "input_tokens", inputTokens, "output_tokens", outputTokens,
		"call_cost_usd", cost.String(), "run_spent_usd", spent.String(),
		"run_budget_usd", b.maxUSD.String(),
	)
}

// RecordASR adds one ASR (Whisper) call's audio-minute cost at the HOSTED API
// rate. Kept as a thin delegate so pre-sub-5-1 callers and tests keep their
// exact semantics; rate-aware callers use RecordASRWithRate.
func (b *Budget) RecordASR(audioSeconds float64) {
	b.RecordASRWithRate(audioSeconds, whisperPerMinuteUSD.InexactFloat64())
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
	// ⚠️ The one operation here that CANNOT be exact: seconds → minutes divides
	// by 60, and 1/60 is a repeating fraction in base 10 just as it is in base
	// 2. Decimal does not fix that; what it fixes is that the quotient is then
	// rounded ONCE at decimal.DivisionPrecision (16 significant digits) instead
	// of carrying binary representation error into every subsequent addition.
	// 16 digits is ~13 orders of magnitude below a cent on any realistic
	// audio length.
	cost := decimal.NewFromFloat(audioSeconds).
		Mul(decimal.NewFromFloat(perMinuteUSD)).
		Div(decimal.NewFromInt(60))
	b.mu.Lock()
	b.asrSeconds += audioSeconds
	b.spentUSD = b.spentUSD.Add(cost)
	b.asrCalls++
	spent := b.spentUSD
	b.mu.Unlock()
	slog.Info("AI usage recorded (ASR)",
		"audio_seconds", audioSeconds, "call_cost_usd", cost.String(),
		"run_spent_usd", spent.String(), "run_budget_usd", b.maxUSD.String(),
	)
}

// SpentUSD returns the accrued spend.
func (b *Budget) SpentUSD() float64 {
	if b == nil {
		return 0
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.spentUSD.InexactFloat64()
}

// Spent is the accrued spend WITHOUT the float64 narrowing SpentUSD applies.
// Use this wherever the number is compared or accumulated further; SpentUSD is
// for the wire and for logs.
func (b *Budget) Spent() decimal.Decimal {
	if b == nil {
		return decimal.Zero
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.spentUSD
}

// BudgetSnapshot is a point-in-time view of a run's metering.
type BudgetSnapshot struct {
	SpentUSD float64
	// Spent is the same figure WITHOUT the float64 narrowing — for callers
	// that subtract or compare it (CR M3: the per-item spend delta did
	// `snap.SpentUSD - scope.spentUSDAtStart` in plain float64, which is the
	// one money computation this codebase had already caught going slightly
	// negative — hence the `if spent < 0` guard beside it).
	Spent        decimal.Decimal
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
		SpentUSD: b.spentUSD.InexactFloat64(), Spent: b.spentUSD,
		BudgetUSD:   b.maxUSD.InexactFloat64(),
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
