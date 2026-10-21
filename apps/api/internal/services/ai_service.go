// Package-level metering ruling — Alexyu 2026-10-21
// (backlog-ai-parse-reads-env-keys-only-a), overturning the "unmetered" half
// of sub-5-1 AC #4 and KEEPING its "uncapped" half.
//
// Every provider call in this file — ParseFilename, ParseFansubFilename,
// GenerateKeywords, both providers — runs under its OWN ai.NewBudget(0) and
// writes one ai_side_spend row when it cost money. The spend shows up in
// 「本月 AI 花費」(/subtitles/spend parse_calls / parse_usd).
//
// Why still no ceiling: these are background metadata calls fired by scans —
// there is no consent moment and no single "run" to attach a cap to, and a
// cap tripping mid-scan would strand a library half-parsed with no screen to
// say why. The ai_cache (30-day TTL) means one filename is asked about once;
// the cost is handled by making it VISIBLE, not by stopping the scan.
//
// Why its own Budget and never the ctx's: a parse fired inside a subtitle run
// must not be added to that run's spend (it would be counted twice) nor be
// able to trip that run's ceiling. NewBudget(0) accrues but Exceeded is always
// false (budget.go).
package services

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/config"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// errAINotConfiguredHealth is the health row's message when no parse key
// resolves — word for word what ServiceHealthChecker said when the service
// used to be nil, so the status page reads exactly as before.
const errAINotConfiguredHealth = "AI provider not configured"

const (
	// FansubParsingTimeout is the maximum time for fansub AI parsing (NFR-P14).
	FansubParsingTimeout = 10 * time.Second
	// KeywordGenerationTimeout is the maximum time for keyword generation.
	KeywordGenerationTimeout = 10 * time.Second
)

// AIServiceInterface defines the contract for AI parsing services.
type AIServiceInterface interface {
	// ParseFilename sends a filename to AI for parsing.
	// Uses cache-first strategy with 30-day TTL.
	ParseFilename(ctx context.Context, filename string) (*ai.ParseResponse, error)

	// ParseFansubFilename parses a fansub filename using a specialized prompt.
	// Optimized for Japanese and Chinese fansub naming conventions.
	// Uses cache-first strategy with 30-day TTL.
	ParseFansubFilename(ctx context.Context, filename string) (*ai.ParseResponse, error)

	// GenerateKeywords generates alternative search keywords for a media title.
	// Uses AI to create language variants (English, Chinese, romaji) and aliases.
	// Uses cache-first strategy with 30-day TTL.
	GenerateKeywords(ctx context.Context, title, language string) (*ai.KeywordVariants, error)

	// ClearCache removes all cached AI parsing results.
	ClearCache(ctx context.Context) (int64, error)

	// ClearExpiredCache removes expired cached entries.
	ClearExpiredCache(ctx context.Context) (int64, error)

	// GetCacheStats returns cache statistics.
	GetCacheStats(ctx context.Context) (*ai.CacheStats, error)

	// IsConfigured returns true if an AI provider is configured.
	IsConfigured() bool

	// GetProviderName returns the name of the configured AI provider.
	GetProviderName() string
}

// AIService orchestrates AI parsing with caching.
type AIService struct {
	// source resolves the provider per call (production). nil falls back to
	// the static provider below (NewAIServiceWithProvider, tests).
	source *ParseProviderHolder
	// provider is the fixed provider of NewAIServiceWithProvider. A zero
	// AIService (neither set) is "not configured".
	provider ai.Provider
	cache    *ai.Cache
	cfg      *config.Config
	// spend records one row per paid provider call; nil = not recorded.
	spend SideSpendRecorder
}

// SideSpendRecorder is where a paid parse-path call is written
// (repository.AISideSpendRepository in production).
type SideSpendRecorder interface {
	Create(ctx context.Context, row *models.AISideSpend) error
}

// Compile-time interface verification.
var _ AIServiceInterface = (*AIService)(nil)

// AIServiceOption tunes NewAIService.
type AIServiceOption func(*aiServiceOptions)

type aiServiceOptions struct {
	resolver   KeyResolver
	factory    ParseProviderFactory
	spend      SideSpendRecorder
	spendIsSet bool
}

// WithAIKeyResolver makes the service read keys through the app's resolver
// (secrets store → env). main.go always passes it; without it the service
// reads the env keys in cfg only.
func WithAIKeyResolver(r KeyResolver) AIServiceOption {
	return func(o *aiServiceOptions) { o.resolver = r }
}

// WithParseProviderFactory swaps the client constructor (tests).
func WithParseProviderFactory(f ParseProviderFactory) AIServiceOption {
	return func(o *aiServiceOptions) { o.factory = f }
}

// WithSideSpendRecorder replaces the default ai_side_spend repository (tests).
func WithSideSpendRecorder(r SideSpendRecorder) AIServiceOption {
	return func(o *aiServiceOptions) { o.spend, o.spendIsSet = r, true }
}

// NewAIService builds the filename-parse service. It ALWAYS returns a service
// — with no key it is simply "not configured" until one is saved — and the
// error is kept only for signature compatibility (always nil today).
//
// Keys are resolved per call (backlog-ai-parse-reads-env-keys-only-a): a key
// saved in the setup wizard or settings page takes effect on the next parse,
// and deleting it turns AI parsing back off, with no restart. AI_PROVIDER is a
// preference, not a requirement — see NewParseProviderHolder.
//
// governor is the process-wide AI throttle (9R-11; nil = unthrottled). It is a
// parameter — not read from cfg — because config carries values, not live
// resources; main.go owns the single Governor instance (sub-5-1 CR H2: a
// Governor field left unset made the factory wiring inert in production).
func NewAIService(cfg *config.Config, db *sql.DB, governor *ai.Governor, opts ...AIServiceOption) (*AIService, error) {
	o := aiServiceOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	if o.resolver == nil {
		o.resolver = NewKeyResolver(nil, EnvKeys{Claude: cfg.GetClaudeAPIKey(), Gemini: cfg.GetGeminiAPIKey()}, nil)
	}
	if !o.spendIsSet && db != nil {
		o.spend = repository.NewAISideSpendRepository(db)
	}

	svc := &AIService{
		source: NewParseProviderHolder(o.resolver, cfg.GetAIProvider(), cfg.GetClaudeModel(), cfg.GetGeminiModel(),
			governor, o.factory, nil),
		cache: ai.NewCache(db),
		cfg:   cfg,
		spend: o.spend,
	}
	if name := svc.GetProviderName(); name != "" {
		slog.Info("AI parsing available", "provider", name)
	} else {
		slog.Info("AI parsing waiting for a key - no Gemini or Claude key yet (saving one in Settings turns it on, no restart)")
	}
	return svc, nil
}

// NewAIServiceWithProvider creates an AI service with a specific provider (for testing).
func NewAIServiceWithProvider(provider ai.Provider, cache *ai.Cache) *AIService {
	return &AIService{
		provider: provider,
		cache:    cache,
	}
}

// resolve returns the provider for this call, or ai.ErrAINotConfigured.
func (s *AIService) resolve(ctx context.Context) (ResolvedParseProvider, error) {
	if s.source != nil {
		return s.source.Resolve(ctx)
	}
	if s.provider == nil {
		return ResolvedParseProvider{}, ai.ErrAINotConfigured
	}
	return ResolvedParseProvider{Provider: s.provider, Name: s.provider.Name()}, nil
}

// meteredParse makes ONE provider call under its own uncapped Budget and
// records what it cost. The ctx's own Budget (a subtitle run's) is shadowed,
// never charged — see the package comment. Usage that came back with an
// error is recorded too: the money was spent.
func (s *AIService) meteredParse(ctx context.Context, kind models.AISideSpendKind, p ResolvedParseProvider, req *ai.ParseRequest) (*ai.ParseResponse, error) {
	budget := ai.NewBudget(0)
	resp, err := p.Provider.Parse(ai.WithBudget(ctx, budget), req)
	s.recordSideSpend(ctx, kind, p, budget)
	return resp, err
}

// recordSideSpend writes the call's row when it cost money. A write failure
// is a warning, never the parse's failure. The write ignores the call's
// cancellation (fansub/keyword timeouts) — a timed-out call can still have
// been billed.
func (s *AIService) recordSideSpend(ctx context.Context, kind models.AISideSpendKind, p ResolvedParseProvider, budget *ai.Budget) {
	snap := budget.Snapshot()
	if s.spend == nil || !snap.Spent.IsPositive() {
		return
	}
	row := &models.AISideSpend{
		Kind:         kind,
		Provider:     string(p.Name),
		ModelID:      p.Model,
		InputTokens:  snap.InputTokens,
		OutputTokens: snap.OutputTokens,
		SpentUSD:     snap.SpentUSD,
	}
	if err := s.spend.Create(context.WithoutCancel(ctx), row); err != nil {
		slog.Warn("Failed to record AI parse spend — the parse itself is unaffected",
			"error", err, "kind", kind, "provider", p.Name, "spent_usd", snap.SpentUSD)
	}
}

// ParseFilename parses a filename using AI with cache-first strategy.
func (s *AIService) ParseFilename(ctx context.Context, filename string) (*ai.ParseResponse, error) {
	p, err := s.resolve(ctx)
	if err != nil {
		return nil, err
	}

	// Check cache first
	cached, err := s.cache.Get(ctx, filename)
	if err != nil {
		slog.Warn("Cache lookup failed, proceeding with AI",
			"error", err,
			"filename", filename,
		)
		// Continue to AI parsing
	} else if cached != nil {
		slog.Debug("Cache hit for AI parsing",
			"filename", filename,
			"title", cached.Title,
		)
		return cached, nil
	}

	// Parse with AI
	req := &ai.ParseRequest{
		Filename: filename,
	}

	response, err := s.meteredParse(ctx, models.AISideSpendKindParse, p, req)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if err := s.cache.Set(ctx, filename, p.Name, "", response); err != nil {
		slog.Warn("Failed to cache AI response",
			"error", err,
			"filename", filename,
		)
		// Don't fail the request if caching fails
	}

	return response, nil
}

// ParseFansubFilename parses a fansub filename using a specialized prompt.
// Optimized for Japanese and Chinese fansub naming conventions.
// Enforces 10-second timeout per NFR-P14.
func (s *AIService) ParseFansubFilename(ctx context.Context, filename string) (*ai.ParseResponse, error) {
	start := time.Now()

	p, err := s.resolve(ctx)
	if err != nil {
		return nil, err
	}

	// Use fansub-specific cache key prefix to avoid collisions
	cacheKey := "fansub:" + filename

	// Check cache first
	cached, err := s.cache.Get(ctx, cacheKey)
	if err != nil {
		slog.Warn("Cache lookup failed for fansub parsing, proceeding with AI",
			"error", err,
			"filename", filename,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		// Continue to AI parsing
	} else if cached != nil {
		slog.Debug("Cache hit for fansub AI parsing",
			"filename", filename,
			"title", cached.Title,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return cached, nil
	}

	// Apply 10-second timeout for fansub parsing (NFR-P14)
	ctx, cancel := context.WithTimeout(ctx, FansubParsingTimeout)
	defer cancel()

	// Build fansub-specific prompt
	fansubPrompt := prompts.BuildFansubPrompt(filename)

	// Parse with AI using specialized prompt
	req := &ai.ParseRequest{
		Filename: filename,
		Prompt:   fansubPrompt,
	}

	slog.Debug("Starting fansub AI parsing",
		"filename", filename,
		"timeout_seconds", FansubParsingTimeout.Seconds(),
	)

	response, err := s.meteredParse(ctx, models.AISideSpendKindFansub, p, req)
	if err != nil {
		duration := time.Since(start)
		slog.Error("AI fansub parsing failed",
			"error", err,
			"filename", filename,
			"duration_ms", duration.Milliseconds(),
			"timeout_exceeded", ctx.Err() == context.DeadlineExceeded,
		)
		return nil, err
	}

	duration := time.Since(start)

	// Validate response confidence
	if response.Confidence < 0.3 {
		slog.Warn("Low confidence fansub parsing result",
			"filename", filename,
			"confidence", response.Confidence,
			"title", response.Title,
			"duration_ms", duration.Milliseconds(),
		)
	}

	// Cache the result with fansub prefix
	if err := s.cache.Set(ctx, cacheKey, p.Name, "fansub", response); err != nil {
		slog.Warn("Failed to cache fansub AI response",
			"error", err,
			"filename", filename,
		)
		// Don't fail the request if caching fails
	}

	slog.Info("Fansub filename parsed successfully",
		"filename", filename,
		"title", response.Title,
		"fansub_group", response.FansubGroup,
		"confidence", response.Confidence,
		"duration_ms", duration.Milliseconds(),
	)

	return response, nil
}

// GenerateKeywords generates alternative search keywords for a media title.
// Uses AI to create language variants (English, Chinese, romaji) and aliases.
// Uses cache-first strategy with 30-day TTL.
func (s *AIService) GenerateKeywords(ctx context.Context, title, language string) (*ai.KeywordVariants, error) {
	start := time.Now()

	p, err := s.resolve(ctx)
	if err != nil {
		return nil, err
	}

	// Validate title
	if title == "" {
		return nil, errors.New("title is required for keyword generation")
	}

	// Use keyword-specific cache key prefix to avoid collisions
	cacheKey := "keywords:" + title

	// Check cache first
	cached, err := s.cache.Get(ctx, cacheKey)
	if err != nil {
		slog.Warn("Cache lookup failed for keyword generation, proceeding with AI",
			"error", err,
			"title", title,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		// Continue to AI generation
	} else if cached != nil {
		slog.Debug("Cache hit for keyword generation",
			"title", title,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		// Parse the cached RawResponse as KeywordVariants
		variants, err := ai.KeywordVariantsFromJSON(cached.RawResponse)
		if err != nil {
			slog.Warn("Failed to parse cached keyword variants, regenerating",
				"error", err,
				"title", title,
			)
			// Continue to AI generation
		} else {
			return variants, nil
		}
	}

	// Apply timeout for keyword generation
	ctx, cancel := context.WithTimeout(ctx, KeywordGenerationTimeout)
	defer cancel()

	// Build keyword generation prompt
	keywordPrompt := prompts.BuildKeywordPrompt(title, language)

	// Generate with AI using keyword prompt
	req := &ai.ParseRequest{
		Filename: title, // Using Filename field for the title
		Prompt:   keywordPrompt,
	}

	slog.Debug("Starting keyword generation",
		"title", title,
		"language", language,
		"timeout_seconds", KeywordGenerationTimeout.Seconds(),
	)

	response, err := s.meteredParse(ctx, models.AISideSpendKindKeywords, p, req)
	if err != nil {
		duration := time.Since(start)
		slog.Error("AI keyword generation failed",
			"error", err,
			"title", title,
			"duration_ms", duration.Milliseconds(),
			"timeout_exceeded", ctx.Err() == context.DeadlineExceeded,
		)
		return nil, err
	}

	duration := time.Since(start)

	// Parse the AI response as KeywordVariants
	variants, err := ai.KeywordVariantsFromJSON(response.RawResponse)
	if err != nil {
		slog.Error("Failed to parse keyword variants from AI response",
			"error", err,
			"title", title,
			"raw_response", response.RawResponse,
		)
		return nil, err
	}

	// Cache the result with keyword prefix
	// Store the response for caching (with RawResponse containing the keyword JSON)
	if err := s.cache.Set(ctx, cacheKey, p.Name, "keywords", response); err != nil {
		slog.Warn("Failed to cache keyword generation response",
			"error", err,
			"title", title,
		)
		// Don't fail the request if caching fails
	}

	slog.Info("Keywords generated successfully",
		"title", title,
		"language", language,
		"keyword_count", variants.Count(),
		"duration_ms", duration.Milliseconds(),
	)

	return variants, nil
}

// ClearCache removes all cached AI parsing results.
func (s *AIService) ClearCache(ctx context.Context) (int64, error) {
	return s.cache.ClearAll(ctx)
}

// ClearExpiredCache removes expired cached entries.
func (s *AIService) ClearExpiredCache(ctx context.Context) (int64, error) {
	return s.cache.ClearExpired(ctx)
}

// GetCacheStats returns cache statistics.
func (s *AIService) GetCacheStats(ctx context.Context) (*ai.CacheStats, error) {
	return s.cache.Stats(ctx)
}

// IsConfigured reports whether a parse key resolves RIGHT NOW (secrets store
// → env). Asked per call by the parser and the keyword phase, so saving or
// deleting a key in the settings page takes effect without a restart.
func (s *AIService) IsConfigured() bool {
	return s.GetProviderName() != ""
}

// GetProviderName returns the provider the next call would use, "" when no
// key resolves.
func (s *AIService) GetProviderName() string {
	if s.source != nil {
		return s.source.ProviderName(context.Background())
	}
	if s.provider == nil {
		return ""
	}
	return string(s.provider.Name())
}

// Ping checks if the AI service is accessible.
// Implements health.Pingable interface for health monitoring. Judged per call:
// with no key it fails with the exact message the health row showed when the
// service used to be nil at boot.
func (s *AIService) Ping(ctx context.Context) error {
	if !s.IsConfigured() {
		return errors.New(errAINotConfiguredHealth)
	}
	// AI service is considered healthy if it's configured
	// Actual API connectivity is verified on first use
	return nil
}
