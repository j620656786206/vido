// Package preview runs 「試跑 20 句」 (story sub-7-8c): a model nobody has graded
// is tried on the first PreviewCues golden cues with the box's own key, judged
// the same way cmd/grade judges, and the result is kept on THIS box only.
//
// It is its own package because it needs both services (the translator, the
// Claude holder) and eval (which imports subtitle, which imports services) —
// inside services that would be an import cycle.
package preview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/eval"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
	"github.com/vido/api/internal/zhtw"
)

// Preview runs are not media runs: they do not touch subtitle_runs and are
// not part of the monthly spend summary (documented in the UI copy).

const (
	// PreviewCues is how many golden cues a preview translates.
	PreviewCues = 20
	// previewCooldown is the per-model gap between two previews on one box —
	// a double-click must not buy the same 20 cues twice.
	previewCooldown = 60 * time.Second
	// Preview budget: the estimate ×3, clamped. The floor keeps a tiny Haiku
	// estimate from tripping the ceiling on the first batch; the cap is the
	// most a「約 $0.0x」button may ever cost.
	previewBudgetFloorUSD = 0.05
	previewBudgetCapUSD   = 0.20
	// settingKeyLocalGradePrefix + model id = where a box keeps its own result.
	settingKeyLocalGradePrefix = "models.local_grade."
)

// Token heuristics for the estimate. They are deliberately coarse and
// documented as such: the figure on the button is 「約」, and the real charge is
// what the run reports afterwards.
const (
	estTokensPerCueIn      = 60  // one English cue inside the prompt, with its index
	estTokensPerCueOut     = 45  // one zh-TW line back
	estJudgeTokensPerCueIn = 160 // cue + refs + candidate as JSON
	estJudgeTokensPerCueOu = 25
	estCharsPerToken       = 3 // mixed zh/en system text
)

// LocalGrade is one box's own preview result — stored per model, returned on
// GET /settings/models as `local_grade`.
//
// [@contract-v1] — additive field on the sub-6-8a model entry.
type LocalGrade struct {
	ModelID     string    `json:"model_id"`
	Cues        int       `json:"cues"`
	ZeroRate    float64   `json:"zero_rate"`
	NaturalRate float64   `json:"natural_rate"`
	CostUSD     float64   `json:"cost_usd"`
	JudgeModel  string    `json:"judge_model"`
	GradedAt    time.Time `json:"graded_at"`
	// Incomplete is non-empty when the budget stopped the run early; the rates
	// then cover fewer cues than Cues says and the UI must say so.
	Incomplete string `json:"incomplete,omitempty"`
}

// ErrPreviewTooSoon is returned when the same model was previewed less than
// previewCooldown ago on this box.
var ErrPreviewTooSoon = errors.New("model preview: too soon")

// ErrPreviewUnsupportedModel is returned for a model this deployment cannot run.
var ErrPreviewUnsupportedModel = errors.New("model preview: unsupported model")

// completerSource is the narrow slice of TranslationDispatcher the preview
// needs: a completer per model, and "does this provider have a key right now"
// to pick the judge (disc-2026-10-preview-judge-needs-claude).
type completerSource interface {
	GetFor(ctx context.Context, model string) (ai.TextCompleter, error)
	ProviderConfigured(ctx context.Context, provider string) bool
}

// modelSupport is the narrow slice of ModelCatalogService the preview needs.
type modelSupport interface {
	Supports(ctx context.Context, model string) bool
}

// localizationReader is the narrow slice of LocalizationSettingsService the
// preview needs: the dial in force, so the preview translates the way the
// box's real runs do.
type localizationReader interface {
	Resolve(ctx context.Context) services.LocalizationSettings
}

// ModelPreviewService runs and stores previews.
type Service struct {
	completers   completerSource
	catalog      modelSupport
	localization localizationReader
	settings     repository.SettingsRepositoryInterface
	now          func() time.Time
	logger       *slog.Logger

	// run is the seam tests replace: it is eval.Run with the real translator
	// and judge wired, and nothing in Preview depends on how it is built.
	run func(ctx context.Context, model string, cues []eval.Cue) (eval.Report, error)

	mu       sync.Mutex
	lastRun  map[string]time.Time
	inFlight map[string]bool
}

// NewService wires the service. localization may be nil (default
// level); settings may be nil (results are returned but not stored).
func NewService(completers completerSource, catalog modelSupport, localization localizationReader, settings repository.SettingsRepositoryInterface, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Service{
		completers: completers, catalog: catalog, localization: localization, settings: settings,
		now:      time.Now,
		logger:   logger.With("service", "model_preview"),
		lastRun:  map[string]time.Time{},
		inFlight: map[string]bool{},
	}
	s.run = s.realRun
	return s
}

// WithClock replaces the clock (Rule 23).
func (s *Service) WithClock(now func() time.Time) *Service {
	s.now = now
	return s
}

// judgeFor decides who judges a preview, from the keys in force NOW — read
// through the same dispatcher/resolver the translation uses, so a key saved
// on the settings page counts on the next call, no restart. The quote
// (EstimateUSD/BudgetUSD) and the run (realRun) both call it, so the button
// prices the judge the run will actually use.
//
//   - a Claude key → eval.DefaultJudgeModel, as before: the score stays
//     comparable with the Claude-judged rating table;
//   - no Claude key, a Gemini key → eval.GeminiFallbackJudgeModel, so a
//     Gemini-only install can try before it spends;
//   - neither → ai.ErrAINotConfigured (409 AI_NOT_CONFIGURED).
func (s *Service) judgeFor(ctx context.Context) (string, error) {
	if s.completers != nil {
		for _, judge := range []string{eval.DefaultJudgeModel, eval.GeminiFallbackJudgeModel} {
			if s.completers.ProviderConfigured(ctx, ai.ProviderOf(judge)) {
				return judge, nil
			}
		}
	}
	return "", fmt.Errorf("%w: no Claude or Gemini key for the preview judge", ai.ErrAINotConfigured)
}

// quoteJudge is judgeFor for the price on the button. With no key at all there
// is nothing to run (the catalog lists no models), so the quote keeps the
// historic Claude-judge price instead of failing the model list.
func (s *Service) quoteJudge(ctx context.Context) string {
	if judge, err := s.judgeFor(ctx); err == nil {
		return judge
	}
	return eval.DefaultJudgeModel
}

// EstimateUSD is the figure for the button: PreviewCues through the model plus
// the judge this box would use right now, from the published per-token rates.
// 0 when the model is unpriced.
func (s *Service) EstimateUSD(ctx context.Context, model string) float64 {
	return s.estimateUSD(ctx, model, s.quoteJudge(ctx))
}

func (s *Service) estimateUSD(ctx context.Context, model, judgeModel string) float64 {
	if !ai.HasPricing(model) {
		return 0
	}
	level := prompts.DefaultLocalizationLevel
	if s.localization != nil {
		level = s.localization.Resolve(ctx).Level
	}
	systemTokens := float64(len(prompts.ComposeInvariantSystemPrompt(level))) / estCharsPerToken
	batchSize := prompts.SubtitleTranslatorBatchSizeFor(model)
	batches := float64((PreviewCues + batchSize - 1) / batchSize)
	inTokens := batches*systemTokens + float64(PreviewCues*estTokensPerCueIn) + float64((batches-1)*prompts.SubtitleTranslatorContextWindow*estTokensPerCueIn)
	outTokens := float64(PreviewCues * estTokensPerCueOut)
	p := ai.PricingFor(model)
	usd := inTokens*p.InputPer1M.InexactFloat64()/1e6 + outTokens*p.OutputPer1M.InexactFloat64()/1e6

	j := ai.PricingFor(judgeModel)
	judgeIn := float64(len(eval.JudgeSystemPrompt()))/estCharsPerToken + float64(PreviewCues*estJudgeTokensPerCueIn)
	judgeOut := float64(PreviewCues * estJudgeTokensPerCueOu)
	usd += judgeIn*j.InputPer1M.InexactFloat64()/1e6 + judgeOut*j.OutputPer1M.InexactFloat64()/1e6
	return usd
}

// BudgetUSD is the ceiling a preview of this model runs under, with the judge
// this box would use right now.
func (s *Service) BudgetUSD(ctx context.Context, model string) float64 {
	return s.budgetUSD(ctx, model, s.quoteJudge(ctx))
}

func (s *Service) budgetUSD(ctx context.Context, model, judgeModel string) float64 {
	b := s.estimateUSD(ctx, model, judgeModel) * 3
	if b < previewBudgetFloorUSD {
		b = previewBudgetFloorUSD
	}
	if b > previewBudgetCapUSD {
		b = previewBudgetCapUSD
	}
	return b
}

// Preview translates the first PreviewCues golden cues with the model under
// the user's key, judges them, stores the result and returns it.
func (s *Service) Preview(ctx context.Context, model string) (LocalGrade, error) {
	if model == "" || s.catalog == nil || !s.catalog.Supports(ctx, model) {
		return LocalGrade{}, fmt.Errorf("%w: %q", ErrPreviewUnsupportedModel, model)
	}
	if err := s.acquire(model); err != nil {
		return LocalGrade{}, err
	}
	defer s.release(model)

	all, err := eval.Golden()
	if err != nil {
		return LocalGrade{}, err
	}
	cues := all
	if len(cues) > PreviewCues {
		cues = cues[:PreviewCues]
	}

	rep, err := s.run(ctx, model, cues)
	if err != nil {
		return LocalGrade{}, err
	}
	grade := LocalGrade{
		ModelID: model, Cues: rep.Cues, ZeroRate: rep.ZeroRate, NaturalRate: rep.NaturalRate,
		CostUSD: rep.CostUSD, JudgeModel: rep.JudgeModel, GradedAt: s.now(), Incomplete: rep.Incomplete,
	}
	if err := s.store(ctx, grade); err != nil {
		// The user paid and got a number; losing the record is a warning,
		// not a failed request.
		s.logger.Warn("local grade not stored", "model_id", model, "error", err)
	}
	s.logger.Info("model preview finished", "model_id", model, "zero_rate", grade.ZeroRate,
		"natural_rate", grade.NaturalRate, "cost_usd", grade.CostUSD, "incomplete", grade.Incomplete)
	return grade, nil
}

// LocalGrades returns every stored preview on this box keyed by model id.
func (s *Service) LocalGrades(ctx context.Context) map[string]LocalGrade {
	out := map[string]LocalGrade{}
	if s.settings == nil {
		return out
	}
	all, err := s.settings.GetAll(ctx)
	if err != nil {
		s.logger.Warn("local grades unreadable", "error", err)
		return out
	}
	for _, st := range all {
		if !strings.HasPrefix(st.Key, settingKeyLocalGradePrefix) {
			continue
		}
		var g LocalGrade
		if err := json.Unmarshal([]byte(st.Value), &g); err != nil || g.ModelID == "" {
			s.logger.Warn("local grade row unreadable — skipped", "key", st.Key, "error", err)
			continue
		}
		out[g.ModelID] = g
	}
	return out
}

func (s *Service) store(ctx context.Context, g LocalGrade) error {
	if s.settings == nil {
		return nil
	}
	b, err := json.Marshal(g)
	if err != nil {
		return err
	}
	return s.settings.Set(ctx, &models.Setting{Key: settingKeyLocalGradePrefix + g.ModelID, Value: string(b), Type: "json"})
}

// acquire enforces the cooldown and single-flight per model.
func (s *Service) acquire(model string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inFlight[model] {
		return fmt.Errorf("%w: a preview of %s is already running", ErrPreviewTooSoon, model)
	}
	if last, ok := s.lastRun[model]; ok {
		if wait := previewCooldown - s.now().Sub(last); wait > 0 {
			return fmt.Errorf("%w: retry in %ds", ErrPreviewTooSoon, int(wait.Seconds())+1)
		}
	}
	s.inFlight[model] = true
	s.lastRun[model] = s.now()
	return nil
}

func (s *Service) release(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.inFlight, model)
}

// realRun wires eval.Run to the box's own key, the dial in force, OpenCC and
// the lexicon — the same path cmd/grade takes, under the preview budget.
func (s *Service) realRun(ctx context.Context, model string, cues []eval.Cue) (eval.Report, error) {
	completer, err := s.completers.GetFor(ctx, model)
	if err != nil {
		return eval.Report{}, err
	}
	// Decided once per run and used for the judge, the budget and the report,
	// so the three can never disagree.
	judgeModel, err := s.judgeFor(ctx)
	if err != nil {
		return eval.Report{}, err
	}
	judge, err := s.completers.GetFor(ctx, judgeModel)
	if err != nil {
		return eval.Report{}, err
	}
	level := prompts.DefaultLocalizationLevel
	if s.localization != nil {
		level = s.localization.Resolve(ctx).Level
	}
	budget := ai.NewBudget(s.budgetUSD(ctx, model, judgeModel))
	ctx = ai.WithBudget(ctx, budget)
	ctx = ai.WithModelID(ctx, model)

	translator := services.NewTranslationService(completer, nil)
	translate := func(ctx context.Context, cues []eval.Cue) (map[string]string, error) {
		blocks := make([]services.TranslationBlock, len(cues))
		for i, c := range cues {
			blocks[i] = services.TranslationBlock{Index: i + 1, Start: c.Start, End: c.End, Text: c.Text}
		}
		out, _, err := translator.TranslateWithGlossary(ctx, blocks, nil, nil, services.WithLocalizationLevel(level))
		got := make(map[string]string, len(out))
		for _, b := range out {
			if b.Index >= 1 && b.Index <= len(cues) {
				got[cues[b.Index-1].ID] = b.Text
			}
		}
		return got, err
	}

	// Delivery's finishing step (zhtw.Finalize); no title context, so the
	// mainland exemption never applies here.
	var conv zhtw.Converter
	if c, cerr := subtitle.NewConverter(); cerr == nil && c.IsAvailable() {
		conv = c
	}
	finalize := func(raw string) string {
		out, _ := zhtw.Finalize(conv, raw, nil)
		return out
	}

	return eval.Run(ctx, eval.Options{
		Cues: cues, Translate: translate, Finalize: finalize,
		Judge: judge, JudgeModel: judgeModel,
		ModelID: model, PromptVersion: prompts.PromptVersionFor(level), LexiconVersion: prompts.LexiconVersion(),
		// The 20-cue preview sends no name glossary, so it is the v1 exam —
		// not comparable to a golden-v2 rating (golden-v2-exam-with-name-glossary).
		SampleVersion: eval.SampleVersionNoGlossary,
		Spent:         budget.SpentUSD, Now: s.now,
	})
}
