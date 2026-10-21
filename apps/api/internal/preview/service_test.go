package preview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/eval"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

type stubCatalog struct{ ok map[string]bool }

func (s stubCatalog) Supports(_ context.Context, m string) bool { return s.ok[m] }

type stubLevel struct{ level prompts.LocalizationLevel }

func (s stubLevel) Resolve(context.Context) services.LocalizationSettings {
	return services.LocalizationSettings{Level: s.level, Source: services.LocalizationSourceSettings}
}

// memSettings is the slice of SettingsRepositoryInterface the service uses.
type memSettings struct{ rows map[string]*models.Setting }

func (m *memSettings) Set(_ context.Context, s *models.Setting) error {
	if m.rows == nil {
		m.rows = map[string]*models.Setting{}
	}
	cp := *s
	m.rows[s.Key] = &cp
	return nil
}
func (m *memSettings) Get(_ context.Context, key string) (*models.Setting, error) {
	if s, ok := m.rows[key]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}
func (m *memSettings) GetAll(context.Context) ([]models.Setting, error) {
	var out []models.Setting
	for _, s := range m.rows {
		out = append(out, *s)
	}
	return out, nil
}
func (m *memSettings) Delete(_ context.Context, key string) error { delete(m.rows, key); return nil }
func (m *memSettings) GetString(ctx context.Context, key string) (string, error) {
	s, err := m.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return s.Value, nil
}
func (m *memSettings) GetInt(context.Context, string) (int, error)   { return 0, errors.New("n/a") }
func (m *memSettings) GetBool(context.Context, string) (bool, error) { return false, errors.New("n/a") }
func (m *memSettings) SetString(ctx context.Context, key, value string) error {
	return m.Set(ctx, &models.Setting{Key: key, Value: value, Type: "string"})
}
func (m *memSettings) SetInt(context.Context, string, int) error   { return nil }
func (m *memSettings) SetBool(context.Context, string, bool) error { return nil }

func newTestService(t *testing.T, store *memSettings) (*Service, *time.Time) {
	t.Helper()
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	s := NewService(nil, stubCatalog{ok: map[string]bool{"claude-opus-4-8": true}}, stubLevel{prompts.LocalizationStandard}, store, nil).
		WithClock(func() time.Time { return now })
	s.run = func(_ context.Context, model string, cues []eval.Cue) (eval.Report, error) {
		return eval.Report{ModelID: model, Cues: len(cues), ZeroRate: 0.05, NaturalRate: 0.7, CostUSD: 0.0123, JudgeModel: eval.DefaultJudgeModel, Grade: "A"}, nil
	}
	return s, &now
}

func TestPreview_RunsFirst20StoresAndReturns(t *testing.T) {
	store := &memSettings{}
	s, _ := newTestService(t, store)
	var seen int
	inner := s.run
	s.run = func(ctx context.Context, m string, cues []eval.Cue) (eval.Report, error) {
		seen = len(cues)
		require.Equal(t, "tos-01", cues[0].ID, "the preview uses the golden sample from the top")
		return inner(ctx, m, cues)
	}
	g, err := s.Preview(context.Background(), "claude-opus-4-8")
	require.NoError(t, err)
	assert.Equal(t, PreviewCues, seen)
	assert.Equal(t, 0.0123, g.CostUSD)
	assert.Equal(t, 0.05, g.ZeroRate)
	assert.Equal(t, "claude-opus-4-8", g.ModelID)
	assert.Empty(t, g.Incomplete)

	row, ok := store.rows[settingKeyLocalGradePrefix+"claude-opus-4-8"]
	require.True(t, ok, "stored under models.local_grade.<id>")
	assert.Equal(t, "json", row.Type)
	var back LocalGrade
	require.NoError(t, json.Unmarshal([]byte(row.Value), &back))
	assert.Equal(t, g, back)

	all := s.LocalGrades(context.Background())
	assert.Equal(t, g, all["claude-opus-4-8"])
}

func TestPreview_RefusesUnsupportedAndThrottles(t *testing.T) {
	s, now := newTestService(t, &memSettings{})
	_, err := s.Preview(context.Background(), "gemini-2.5-flash")
	assert.ErrorIs(t, err, ErrPreviewUnsupportedModel)
	_, err = s.Preview(context.Background(), "")
	assert.ErrorIs(t, err, ErrPreviewUnsupportedModel)

	_, err = s.Preview(context.Background(), "claude-opus-4-8")
	require.NoError(t, err)
	_, err = s.Preview(context.Background(), "claude-opus-4-8")
	assert.ErrorIs(t, err, ErrPreviewTooSoon, "same model inside the cooldown")
	*now = now.Add(previewCooldown + time.Second)
	_, err = s.Preview(context.Background(), "claude-opus-4-8")
	assert.NoError(t, err, "cooldown over")
}

func TestPreview_IncompleteAndErrorsPassThrough(t *testing.T) {
	s, _ := newTestService(t, &memSettings{})
	s.run = func(context.Context, string, []eval.Cue) (eval.Report, error) {
		return eval.Report{Cues: 20, Grade: eval.GradeIncomplete, Incomplete: "translate: budget", CostUSD: 0.2}, nil
	}
	g, err := s.Preview(context.Background(), "claude-opus-4-8")
	require.NoError(t, err)
	assert.Equal(t, "translate: budget", g.Incomplete, "a budget stop is a result the user paid for, not an error")

	s2, _ := newTestService(t, &memSettings{})
	s2.run = func(context.Context, string, []eval.Cue) (eval.Report, error) {
		return eval.Report{}, ai.ErrAINotConfigured
	}
	_, err = s2.Preview(context.Background(), "claude-opus-4-8")
	assert.ErrorIs(t, err, ai.ErrAINotConfigured)
}

func TestEstimateAndBudget(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestService(t, &memSettings{})
	haiku := s.EstimateUSD(ctx, "claude-haiku-4-5")
	sonnet := s.EstimateUSD(ctx, "claude-sonnet-5")
	opus := s.EstimateUSD(ctx, "claude-opus-4-8")
	assert.Greater(t, haiku, 0.0)
	assert.Less(t, haiku, sonnet)
	assert.Less(t, sonnet, opus)
	assert.Less(t, opus, 0.10, "20 cues should never be quoted at more than a dime")
	assert.Equal(t, 0.0, s.EstimateUSD(ctx, "not-a-model"))
	// Budget = estimate ×3, clamped to [floor, cap].
	for _, m := range []string{"claude-haiku-4-5", "claude-sonnet-5", "claude-opus-4-8"} {
		b := s.BudgetUSD(ctx, m)
		assert.GreaterOrEqual(t, b, previewBudgetFloorUSD, m)
		assert.LessOrEqual(t, b, previewBudgetCapUSD, m)
		assert.GreaterOrEqual(t, b, s.EstimateUSD(ctx, m)*2, "%s: ceiling must leave real headroom over the estimate", m)
	}
	assert.Equal(t, previewBudgetFloorUSD, s.BudgetUSD(ctx, "not-a-model"), "unpriced → floor, never 0 (a 0 ceiling would stop the first call)")
}

func TestLocalGrades_SkipsUnreadableRows(t *testing.T) {
	store := &memSettings{}
	_ = store.SetString(context.Background(), settingKeyLocalGradePrefix+"broken", "{not json")
	_ = store.SetString(context.Background(), "unrelated.key", "x")
	s, _ := newTestService(t, store)
	assert.Empty(t, s.LocalGrades(context.Background()))
}

// ---- disc-2026-10-preview-judge-needs-claude: who judges, decided per run ----

// fakeKeys is a completerSource whose keys can change between runs, the way a
// key saved on the settings page reaches the next run. GetFor refuses a
// provider with no key exactly like the dispatcher does (ErrAINotConfigured),
// and hands back a completer that answers as a translator or as a judge.
type fakeKeys struct {
	claude, gemini bool
	asked          []string // every model GetFor was asked for
	judgedBy       []string // the model id whose completer answered a judge prompt
	translations   map[int]string
}

func (f *fakeKeys) ProviderConfigured(_ context.Context, provider string) bool {
	switch provider {
	case ai.ProviderNameClaude:
		return f.claude
	case ai.ProviderNameGemini:
		return f.gemini
	}
	return false
}

func (f *fakeKeys) GetFor(_ context.Context, model string) (ai.TextCompleter, error) {
	f.asked = append(f.asked, model)
	switch ai.ProviderOf(model) {
	case ai.ProviderNameClaude:
		if !f.claude {
			return nil, fmt.Errorf("%w: no Claude API key configured", ai.ErrAINotConfigured)
		}
	case ai.ProviderNameGemini:
		if !f.gemini {
			return nil, fmt.Errorf("%w: no Gemini API key configured", ai.ErrAINotConfigured)
		}
	default:
		return nil, fmt.Errorf("%w: %q", ai.ErrAIModelNotFound, model)
	}
	return fakeCompleter{keys: f, model: model}, nil
}

type fakeCompleter struct {
	keys  *fakeKeys
	model string
}

func (c fakeCompleter) CompleteText(_ context.Context, sys, usr string, _ int) (string, error) {
	if sys == eval.JudgeSystemPrompt() {
		c.keys.judgedBy = append(c.keys.judgedBy, c.model)
		var items []eval.JudgeItem
		if err := json.Unmarshal([]byte(usr[strings.Index(usr, "["):]), &items); err != nil {
			return "", err
		}
		out := make([]eval.JudgeScore, 0, len(items))
		for _, it := range items {
			out = append(out, eval.JudgeScore{ID: it.ID, Score: 2})
		}
		b, _ := json.Marshal(out)
		return string(b), nil
	}
	// Translator: answer every cue with its first reference rendering, so the
	// rule check passes and the cues actually reach the judge.
	var sb strings.Builder
	for i := 1; i <= PreviewCues; i++ {
		fmt.Fprintf(&sb, "[%d] %s\n", i, c.keys.translations[i])
	}
	return sb.String(), nil
}

const geminiModelUnderTest = "gemini-3.5-flash-lite"

// newRealRunService wires the real realRun (no s.run seam) to fake keys.
func newRealRunService(t *testing.T, keys *fakeKeys, store *memSettings) (*Service, *time.Time) {
	t.Helper()
	golden, err := eval.Golden()
	require.NoError(t, err)
	keys.translations = map[int]string{}
	for i := 0; i < PreviewCues && i < len(golden); i++ {
		keys.translations[i+1] = golden[i].Refs[0]
	}
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	s := NewService(keys, stubCatalog{ok: map[string]bool{geminiModelUnderTest: true, "claude-haiku-4-5": true}},
		stubLevel{prompts.LocalizationStandard}, store, nil).
		WithClock(func() time.Time { return now })
	return s, &now
}

func TestPreview_GeminiOnlyInstall_JudgedByGemini(t *testing.T) {
	keys := &fakeKeys{gemini: true}
	store := &memSettings{}
	s, _ := newRealRunService(t, keys, store)

	g, err := s.Preview(context.Background(), geminiModelUnderTest)
	require.NoError(t, err, "a Gemini-only install must be able to preview (was 409 AI_NOT_CONFIGURED)")
	assert.Equal(t, eval.GeminiFallbackJudgeModel, g.JudgeModel)
	assert.Equal(t, "gemini-3.7-flash", g.JudgeModel)
	require.NotEmpty(t, keys.judgedBy, "the judge must actually be called")
	for _, m := range keys.judgedBy {
		assert.Equal(t, eval.GeminiFallbackJudgeModel, m)
	}
	assert.NotContains(t, keys.asked, eval.DefaultJudgeModel, "no Claude key: Claude is never asked for")

	var back LocalGrade
	require.NoError(t, json.Unmarshal([]byte(store.rows[settingKeyLocalGradePrefix+geminiModelUnderTest].Value), &back))
	assert.Equal(t, eval.GeminiFallbackJudgeModel, back.JudgeModel, "the stored grade names the judge that scored it")
}

func TestPreview_ClaudeKeyPresent_JudgeStaysClaude(t *testing.T) {
	keys := &fakeKeys{claude: true, gemini: true}
	s, _ := newRealRunService(t, keys, &memSettings{})

	g, err := s.Preview(context.Background(), geminiModelUnderTest)
	require.NoError(t, err)
	assert.Equal(t, eval.DefaultJudgeModel, g.JudgeModel, "with a Claude key nothing changes: Sonnet judges")
	require.NotEmpty(t, keys.judgedBy)
	for _, m := range keys.judgedBy {
		assert.Equal(t, eval.DefaultJudgeModel, m)
	}
	assert.NotContains(t, keys.asked, eval.GeminiFallbackJudgeModel)
}

func TestPreview_NoKeys_NotConfigured(t *testing.T) {
	keys := &fakeKeys{}
	s, _ := newRealRunService(t, keys, &memSettings{})
	_, err := s.Preview(context.Background(), geminiModelUnderTest)
	assert.ErrorIs(t, err, ai.ErrAINotConfigured)
	assert.Empty(t, keys.judgedBy)
}

func TestPreview_ClaudeKeySavedLater_NextPreviewJudgedByClaude(t *testing.T) {
	keys := &fakeKeys{gemini: true}
	s, now := newRealRunService(t, keys, &memSettings{})

	g, err := s.Preview(context.Background(), geminiModelUnderTest)
	require.NoError(t, err)
	assert.Equal(t, eval.GeminiFallbackJudgeModel, g.JudgeModel)

	keys.claude = true // saved on the settings page; no restart
	keys.judgedBy = nil
	*now = now.Add(previewCooldown + time.Second)
	g, err = s.Preview(context.Background(), geminiModelUnderTest)
	require.NoError(t, err)
	assert.Equal(t, eval.DefaultJudgeModel, g.JudgeModel, "the judge is decided per run, not once at boot")
	require.NotEmpty(t, keys.judgedBy)
	assert.Equal(t, eval.DefaultJudgeModel, keys.judgedBy[0])
}

func TestEstimateAndBudget_GeminiJudgePricesTheButton(t *testing.T) {
	ctx := context.Background()
	geminiOnly, _ := newRealRunService(t, &fakeKeys{gemini: true}, &memSettings{})
	withClaude, _ := newRealRunService(t, &fakeKeys{claude: true, gemini: true}, &memSettings{})

	gem := geminiOnly.EstimateUSD(ctx, geminiModelUnderTest)
	cla := withClaude.EstimateUSD(ctx, geminiModelUnderTest)
	assert.Greater(t, gem, 0.0)
	assert.Less(t, gem, cla, "a Gemini judge is cheaper than Sonnet, and the quote must show it")

	// The judge's share of the quote is priced at the judge's own rate.
	jg, jc := ai.PricingFor(eval.GeminiFallbackJudgeModel), ai.PricingFor(eval.DefaultJudgeModel)
	judgeIn := float64(len(eval.JudgeSystemPrompt()))/estCharsPerToken + float64(PreviewCues*estJudgeTokensPerCueIn)
	judgeOut := float64(PreviewCues * estJudgeTokensPerCueOu)
	wantDelta := judgeIn*(jc.InputPer1M.InexactFloat64()-jg.InputPer1M.InexactFloat64())/1e6 +
		judgeOut*(jc.OutputPer1M.InexactFloat64()-jg.OutputPer1M.InexactFloat64())/1e6
	assert.InDelta(t, wantDelta, cla-gem, 1e-12)

	b := geminiOnly.BudgetUSD(ctx, geminiModelUnderTest)
	assert.GreaterOrEqual(t, b, previewBudgetFloorUSD)
	assert.LessOrEqual(t, b, previewBudgetCapUSD)
}
