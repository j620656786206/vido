package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/config"
	"github.com/vido/api/internal/database"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/parser"
	"github.com/vido/api/internal/repository"
)

// backlog-ai-parse-reads-env-keys-only-a AC #7 — AI filename parsing reads the
// key typed into the settings page / setup wizard, per call, and every paid
// provider call lands one row in ai_side_spend (metered, never capped —
// Alexyu 2026-10-21). Run against a REAL migrated database on the app's own
// driver, the REAL AES-256-GCM secrets service and the REAL KeyResolver; only
// the provider factory is fake, so no test here can spend money.

// fakeParseCall is one call a fake provider received.
type fakeParseCall struct {
	provider ai.ProviderName
	key      string
	model    string
	prompt   string
}

// fakeParseFactory builds fake providers that meter like the real ones do
// (claude.go send / gemini.go Parse: RecordLLM against the ctx Budget).
type fakeParseFactory struct {
	mu     sync.Mutex
	calls  []fakeParseCall
	builds int
	// failWith, when set, makes Parse return it AFTER metering — the "the
	// provider answered and billed, but the answer was unusable" case.
	failWith error
	// raw is the RawResponse every call returns (keyword JSON for keywords).
	raw string
}

func (f *fakeParseFactory) build(name ai.ProviderName, key, model string, _ *ai.Governor) ai.Provider {
	f.mu.Lock()
	f.builds++
	f.mu.Unlock()
	return &fakeMeteredProvider{factory: f, name: name, key: key, model: model}
}

func (f *fakeParseFactory) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *fakeParseFactory) lastCall() fakeParseCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[len(f.calls)-1]
}

type fakeMeteredProvider struct {
	factory *fakeParseFactory
	name    ai.ProviderName
	key     string
	model   string
}

func (p *fakeMeteredProvider) Parse(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error) {
	p.factory.mu.Lock()
	p.factory.calls = append(p.factory.calls, fakeParseCall{provider: p.name, key: p.key, model: p.model, prompt: req.Prompt})
	failWith, raw := p.factory.failWith, p.factory.raw
	p.factory.mu.Unlock()

	if b := ai.BudgetFromContext(ctx); b != nil {
		b.RecordLLM(p.model, 1000, 200)
	}
	if failWith != nil {
		return nil, failWith
	}
	return &ai.ParseResponse{
		Title:       "Parsed by " + string(p.name),
		MediaType:   "movie",
		Confidence:  0.9,
		RawResponse: raw,
	}, nil
}

func (p *fakeMeteredProvider) Name() ai.ProviderName { return p.name }

// newAppDriverMigratedDB is the full migration chain on the app's own driver
// (sqlite-utc), so the time text ai_side_spend stores is the production shape.
func newAppDriverMigratedDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open(database.DriverName, ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1) // :memory: is per-connection
	t.Cleanup(func() { _ = db.Close() })
	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

type sideSpendRow struct {
	kind, provider, model string
	in, out               int64
	usd                   float64
}

func sideSpendRows(t *testing.T, db *sql.DB) []sideSpendRow {
	t.Helper()
	rows, err := db.Query(`SELECT kind, provider, model_id, input_tokens, output_tokens, spent_usd FROM ai_side_spend ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()
	var out []sideSpendRow
	for rows.Next() {
		var r sideSpendRow
		require.NoError(t, rows.Scan(&r.kind, &r.provider, &r.model, &r.in, &r.out, &r.usd))
		out = append(out, r)
	}
	require.NoError(t, rows.Err())
	return out
}

type parseHarness struct {
	db       *sql.DB
	resolver KeyResolver
	settings *KeySettingsService
	factory  *fakeParseFactory
	svc      *AIService
}

func newParseHarness(t *testing.T, cfg *config.Config, opts ...AIServiceOption) *parseHarness {
	t.Helper()
	db := newAppDriverMigratedDB(t)
	secretsSvc, cleanup := newRealSecretsService(t)
	t.Cleanup(cleanup)
	resolver := NewKeyResolver(secretsSvc, EnvKeys{Claude: cfg.GetClaudeAPIKey(), Gemini: cfg.GetGeminiAPIKey()}, nil)
	factory := &fakeParseFactory{}
	all := append([]AIServiceOption{WithAIKeyResolver(resolver), WithParseProviderFactory(factory.build)}, opts...)
	svc, err := NewAIService(cfg, db, nil, all...)
	require.NoError(t, err)
	require.NotNil(t, svc, "the service is always built now — no key at boot is not 'no service'")
	return &parseHarness{
		db: db, resolver: resolver, factory: factory, svc: svc,
		settings: NewKeySettingsService(resolver, secretsSvc, true),
	}
}

const unparseableFilename = "random_video_file.mkv"

func TestAIParseKeys_NoKey_FallsBackToNeedsAIExactlyAsBefore(t *testing.T) {
	h := newParseHarness(t, &config.Config{AIProvider: "gemini"})

	assert.False(t, h.svc.IsConfigured())
	assert.Equal(t, "", h.svc.GetProviderName())
	err := h.svc.Ping(context.Background())
	require.Error(t, err)
	assert.Equal(t, "AI provider not configured", err.Error(),
		"the health row's message must read exactly as it did when the service was nil")

	result := NewParserServiceWithAI(h.svc).ParseFilename(unparseableFilename)
	assert.Equal(t, parser.ParseStatusNeedsAI, result.Status)
	assert.Equal(t, "Could not parse with standard patterns", result.ErrorMessage,
		"no key → the no-AI branch, same message as a boot without the service")
	assert.Equal(t, 0, h.factory.callCount())
	assert.Empty(t, sideSpendRows(t, h.db))
}

func TestAIParseKeys_SavedGeminiKey_ParsesWithGeminiAndRecordsSpend(t *testing.T) {
	ctx := context.Background()
	h := newParseHarness(t, &config.Config{AIProvider: "gemini"})
	require.False(t, h.svc.IsConfigured(), "precondition: keyless boot")

	require.NoError(t, h.settings.Save(ctx, map[KeyName]string{KeyGemini: "AIza-from-ui"}))

	assert.True(t, h.svc.IsConfigured(), "the next call sees the saved key — no restart")
	assert.Equal(t, "gemini", h.svc.GetProviderName())
	require.NoError(t, h.svc.Ping(ctx))

	result := NewParserServiceWithAI(h.svc).ParseFilename(unparseableFilename)
	assert.Equal(t, parser.ParseStatusSuccess, result.Status)
	assert.Equal(t, "Parsed by gemini", result.Title)
	assert.Equal(t, "gemini", result.AIProvider)

	call := h.factory.lastCall()
	assert.Equal(t, "AIza-from-ui", call.key)
	assert.Equal(t, ai.DefaultGeminiModel, call.model)

	rows := sideSpendRows(t, h.db)
	require.Len(t, rows, 1)
	assert.Equal(t, "parse", rows[0].kind)
	assert.Equal(t, "gemini", rows[0].provider)
	assert.Equal(t, ai.DefaultGeminiModel, rows[0].model)
	assert.Equal(t, int64(1000), rows[0].in)
	assert.Equal(t, int64(200), rows[0].out)
	// gemini-3.5-flash-lite: 1000 × $0.30/M + 200 × $2.50/M = $0.0008
	assert.InDelta(t, 0.0008, rows[0].usd, 1e-12)
}

func TestAIParseKeys_PreferredProviderWinsOnceItsKeyArrives_AndDeleteRevertsToNoAI(t *testing.T) {
	ctx := context.Background()
	h := newParseHarness(t, &config.Config{AIProvider: "claude"})

	// Only Gemini saved, AI_PROVIDER=claude → the other provider is used.
	require.NoError(t, h.settings.Save(ctx, map[KeyName]string{KeyGemini: "AIza-1"}))
	_, err := h.svc.ParseFilename(ctx, "first.mkv")
	require.NoError(t, err)
	assert.Equal(t, ai.ProviderGemini, h.factory.lastCall().provider)

	// Claude arrives → the preferred one takes over on the very next call.
	require.NoError(t, h.settings.Save(ctx, map[KeyName]string{KeyClaude: "sk-ant-1"}))
	assert.Equal(t, "claude", h.svc.GetProviderName())
	_, err = h.svc.ParseFilename(ctx, "second.mkv")
	require.NoError(t, err)
	call := h.factory.lastCall()
	assert.Equal(t, ai.ProviderClaude, call.provider)
	assert.Equal(t, "sk-ant-1", call.key)
	assert.Equal(t, ai.DefaultClaudeModel, call.model)

	rows := sideSpendRows(t, h.db)
	require.Len(t, rows, 2)
	assert.Equal(t, "gemini", rows[0].provider)
	assert.Equal(t, "claude", rows[1].provider)
	assert.Equal(t, ai.DefaultClaudeModel, rows[1].model)

	// Both keys deleted → back to the no-AI behaviour.
	require.NoError(t, h.settings.Save(ctx, map[KeyName]string{KeyClaude: "", KeyGemini: ""}))
	assert.False(t, h.svc.IsConfigured())
	result := NewParserServiceWithAI(h.svc).ParseFilename(unparseableFilename)
	assert.Equal(t, parser.ParseStatusNeedsAI, result.Status)
	assert.Equal(t, "Could not parse with standard patterns", result.ErrorMessage)
	assert.Equal(t, 2, h.factory.callCount(), "no provider call once the keys are gone")
}

func TestAIParseKeys_KeyChangeRebuildsClient_SameKeyReusesIt(t *testing.T) {
	ctx := context.Background()
	h := newParseHarness(t, &config.Config{AIProvider: "gemini"})
	require.NoError(t, h.settings.Save(ctx, map[KeyName]string{KeyGemini: "AIza-a"}))

	_, err := h.svc.ParseFilename(ctx, "a.mkv")
	require.NoError(t, err)
	_, err = h.svc.ParseFilename(ctx, "b.mkv")
	require.NoError(t, err)
	assert.Equal(t, 1, h.factory.builds, "same key + model + provider → one client")

	require.NoError(t, h.settings.Save(ctx, map[KeyName]string{KeyGemini: "AIza-b"}))
	_, err = h.svc.ParseFilename(ctx, "c.mkv")
	require.NoError(t, err)
	assert.Equal(t, 2, h.factory.builds, "a new key → a new client")
	assert.Equal(t, "AIza-b", h.factory.lastCall().key)
}

func TestAIParseKeys_ModelOverridesStillApply(t *testing.T) {
	ctx := context.Background()
	h := newParseHarness(t, &config.Config{AIProvider: "gemini", GeminiModel: "gemini-2.5-flash", ClaudeModel: "claude-haiku-4-5"})
	require.NoError(t, h.settings.Save(ctx, map[KeyName]string{KeyGemini: "AIza"}))
	_, err := h.svc.ParseFilename(ctx, "a.mkv")
	require.NoError(t, err)
	assert.Equal(t, "gemini-2.5-flash", h.factory.lastCall().model)

	require.NoError(t, h.settings.Save(ctx, map[KeyName]string{KeyGemini: "", KeyClaude: "sk-ant"}))
	_, err = h.svc.ParseFilename(ctx, "b.mkv")
	require.NoError(t, err)
	assert.Equal(t, "claude-haiku-4-5", h.factory.lastCall().model)

	rows := sideSpendRows(t, h.db)
	require.Len(t, rows, 2)
	assert.Equal(t, "gemini-2.5-flash", rows[0].model)
	assert.Equal(t, "claude-haiku-4-5", rows[1].model)
}

func TestAIParseKeys_CacheHitNeitherCallsNorRecords(t *testing.T) {
	ctx := context.Background()
	h := newParseHarness(t, &config.Config{AIProvider: "gemini"})
	require.NoError(t, h.settings.Save(ctx, map[KeyName]string{KeyGemini: "AIza"}))

	_, err := h.svc.ParseFilename(ctx, "same.mkv")
	require.NoError(t, err)
	_, err = h.svc.ParseFilename(ctx, "same.mkv")
	require.NoError(t, err)

	assert.Equal(t, 1, h.factory.callCount(), "the second parse is a cache hit")
	assert.Len(t, sideSpendRows(t, h.db), 1, "a cache hit writes no row")
}

func TestAIParseKeys_OuterRunBudgetIsNeverCharged(t *testing.T) {
	h := newParseHarness(t, &config.Config{AIProvider: "gemini"})
	require.NoError(t, h.settings.Save(context.Background(), map[KeyName]string{KeyGemini: "AIza"}))

	outer := ai.NewBudget(5)
	ctx := ai.WithBudget(context.Background(), outer)

	_, err := h.svc.ParseFilename(ctx, "inside-a-run.mkv")
	require.NoError(t, err)
	_, err = h.svc.ParseFansubFilename(ctx, "[Group] Show - 01.mkv")
	require.NoError(t, err)

	assert.Zero(t, outer.SpentUSD(), "a parse inside a subtitle run must not count toward that run's money")
	assert.Zero(t, outer.Snapshot().LLMCalls)
	assert.Len(t, sideSpendRows(t, h.db), 2, "it is recorded in ai_side_spend instead")
}

func TestAIParseKeys_ExhaustedOuterBudgetDoesNotBlockParsing(t *testing.T) {
	h := newParseHarness(t, &config.Config{AIProvider: "gemini"})
	require.NoError(t, h.settings.Save(context.Background(), map[KeyName]string{KeyGemini: "AIza"}))

	outer := ai.NewBudget(0.000001)
	outer.RecordLLM("claude-sonnet-5", 1000, 1000)
	require.True(t, outer.Exceeded())

	resp, err := h.svc.ParseFilename(ai.WithBudget(context.Background(), outer), "x.mkv")
	require.NoError(t, err, "parsing is metered, never capped — a run's ceiling is not the parser's")
	assert.Equal(t, "Parsed by gemini", resp.Title)
}

func TestAIParseKeys_FansubAndKeywordsRecordTheirOwnKind(t *testing.T) {
	ctx := context.Background()
	h := newParseHarness(t, &config.Config{AIProvider: "gemini"})
	require.NoError(t, h.settings.Save(ctx, map[KeyName]string{KeyGemini: "AIza"}))

	_, err := h.svc.ParseFansubFilename(ctx, "[Leopard-Raws] Kimetsu no Yaiba - 01.mkv")
	require.NoError(t, err)

	h.factory.raw = `{"original":"鬼滅之刃","english":"Demon Slayer"}`
	variants, err := h.svc.GenerateKeywords(ctx, "鬼滅之刃", "zh-TW")
	require.NoError(t, err)
	assert.Equal(t, "Demon Slayer", variants.English)

	rows := sideSpendRows(t, h.db)
	require.Len(t, rows, 2)
	assert.Equal(t, "fansub", rows[0].kind)
	assert.Equal(t, "keywords", rows[1].kind)
}

func TestAIParseKeys_FailedCallThatWasBilledIsStillRecorded(t *testing.T) {
	ctx := context.Background()
	h := newParseHarness(t, &config.Config{AIProvider: "gemini"})
	require.NoError(t, h.settings.Save(ctx, map[KeyName]string{KeyGemini: "AIza"}))
	h.factory.failWith = ai.ErrAIInvalidResponse

	_, err := h.svc.ParseFilename(ctx, "garbled.mkv")
	require.ErrorIs(t, err, ai.ErrAIInvalidResponse)

	rows := sideSpendRows(t, h.db)
	require.Len(t, rows, 1, "the money was spent even though the answer was unusable")
	assert.Greater(t, rows[0].usd, 0.0)
}

type failingSideSpendRecorder struct{ attempts int }

func (f *failingSideSpendRecorder) Create(context.Context, *models.AISideSpend) error {
	f.attempts++
	return errors.New("disk full")
}

func TestAIParseKeys_RecordFailureNeverFailsTheParse(t *testing.T) {
	ctx := context.Background()
	rec := &failingSideSpendRecorder{}
	h := newParseHarness(t, &config.Config{AIProvider: "gemini"}, WithSideSpendRecorder(rec))
	require.NoError(t, h.settings.Save(ctx, map[KeyName]string{KeyGemini: "AIza"}))

	resp, err := h.svc.ParseFilename(ctx, "a.mkv")
	require.NoError(t, err)
	assert.Equal(t, "Parsed by gemini", resp.Title)
	assert.Equal(t, 1, rec.attempts)
}

// AC #2 — the boot failure: AI_PROVIDER defaults to gemini, so a deployment
// with ONLY CLAUDE_API_KEY in its environment used to fail ai.NewProvider
// ("GEMINI_API_KEY not configured") and main.go os.Exit(1)'d. Uses the
// production factory: no call is made, only the client is chosen.
func TestAIParseKeys_ClaudeEnvOnlyBootsAndParsesWithClaude(t *testing.T) {
	db := newAppDriverMigratedDB(t)
	cfg := &config.Config{AIProvider: "gemini", ClaudeAPIKey: "sk-ant-env-only"}

	svc, err := NewAIService(cfg, db, nil)
	require.NoError(t, err, "a Claude-only environment must boot")
	require.NotNil(t, svc)
	assert.True(t, svc.IsConfigured())
	assert.Equal(t, "claude", svc.GetProviderName())
}

func TestAIParseKeys_KeylessBootStillBuildsTheService(t *testing.T) {
	db := newAppDriverMigratedDB(t)
	svc, err := NewAIService(&config.Config{AIProvider: "gemini"}, db, nil)
	require.NoError(t, err)
	require.NotNil(t, svc, "main.go wires parser / keywords / cache sweep unconditionally")
	assert.False(t, svc.IsConfigured())
}

// The unconfigured KeywordService must stay as quiet as the unwired one did:
// IsConfigured false, so the metadata orchestrator skips the phase.
func TestAIParseKeys_KeywordServiceFollowsTheKeys(t *testing.T) {
	ctx := context.Background()
	h := newParseHarness(t, &config.Config{AIProvider: "gemini"})
	ks := NewKeywordService(h.svc)
	assert.False(t, ks.IsConfigured())
	require.NoError(t, h.settings.Save(ctx, map[KeyName]string{KeyGemini: "AIza"}))
	assert.True(t, ks.IsConfigured())
}

// AC #5 — /subtitles/spend's parse_calls / parse_usd come from ai_side_spend
// over the same month window; last month's rows are not this month's money.
func TestAIParseKeys_SpendSummaryCountsThisMonthsSideSpendOnly(t *testing.T) {
	ctx := context.Background()
	db := newAppDriverMigratedDB(t)
	repo := repository.NewAISideSpendRepository(db)
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, loc)

	seed := func(kind models.AISideSpendKind, usd float64, at time.Time) {
		require.NoError(t, repo.Create(ctx, &models.AISideSpend{
			Kind: kind, Provider: "gemini", ModelID: ai.DefaultGeminiModel,
			InputTokens: 1000, OutputTokens: 200, SpentUSD: usd, CreatedAt: at,
		}))
	}
	seed(models.AISideSpendKindParse, 0.004, time.Date(2026, 10, 1, 0, 0, 1, 0, loc))      // first second of the month (local)
	seed(models.AISideSpendKindFansub, 0.003, time.Date(2026, 10, 20, 9, 0, 0, 0, loc))    // this month
	seed(models.AISideSpendKindKeywords, 0.002, time.Date(2026, 10, 21, 11, 0, 0, 0, loc)) // this month
	seed(models.AISideSpendKindParse, 0.5, time.Date(2026, 9, 30, 23, 59, 0, 0, loc))      // last month (local) — excluded
	seed(models.AISideSpendKindParse, 0.7, time.Date(2026, 11, 1, 0, 0, 0, 0, loc))        // next month — excluded

	svc := NewSubtitleSpendService(&fakeSpendRuns{}, nil, nil, nil)
	svc.SetClock(func() time.Time { return now })
	svc.SetSideSpend(repo)

	summary, err := svc.MonthSummary(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, 3, summary.ParseCalls)
	assert.InDelta(t, 0.01, summary.ParseUSD, 1e-9, "rounded to cents like every other figure")
}

func TestAIParseKeys_SpendSummaryWithNoSideSpendIsZeroZero(t *testing.T) {
	db := newAppDriverMigratedDB(t)
	svc := NewSubtitleSpendService(&fakeSpendRuns{}, nil, nil, nil)
	svc.SetSideSpend(repository.NewAISideSpendRepository(db))

	summary, err := svc.MonthSummary(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, 0, summary.ParseCalls)
	assert.Equal(t, 0.0, summary.ParseUSD)
}

func TestAIParseKeys_SpendSummaryWireNames(t *testing.T) {
	body, err := json.Marshal(&SubtitleSpendSummary{ParseCalls: 37, ParseUSD: 0.18})
	require.NoError(t, err)
	var wire map[string]any
	require.NoError(t, json.Unmarshal(body, &wire))
	assert.Equal(t, float64(37), wire["parse_calls"])
	assert.Equal(t, 0.18, wire["parse_usd"])
}
