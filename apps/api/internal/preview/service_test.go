package preview

import (
	"context"
	"encoding/json"
	"errors"
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
	s, _ := newTestService(t, &memSettings{})
	haiku := s.EstimateUSD("claude-haiku-4-5")
	sonnet := s.EstimateUSD("claude-sonnet-5")
	opus := s.EstimateUSD("claude-opus-4-8")
	assert.Greater(t, haiku, 0.0)
	assert.Less(t, haiku, sonnet)
	assert.Less(t, sonnet, opus)
	assert.Less(t, opus, 0.10, "20 cues should never be quoted at more than a dime")
	assert.Equal(t, 0.0, s.EstimateUSD("not-a-model"))
	// Budget = estimate ×3, clamped to [floor, cap].
	for _, m := range []string{"claude-haiku-4-5", "claude-sonnet-5", "claude-opus-4-8"} {
		b := s.BudgetUSD(m)
		assert.GreaterOrEqual(t, b, previewBudgetFloorUSD, m)
		assert.LessOrEqual(t, b, previewBudgetCapUSD, m)
		assert.GreaterOrEqual(t, b, s.EstimateUSD(m)*2, "%s: ceiling must leave real headroom over the estimate", m)
	}
	assert.Equal(t, previewBudgetFloorUSD, s.BudgetUSD("not-a-model"), "unpriced → floor, never 0 (a 0 ceiling would stop the first call)")
}

func TestLocalGrades_SkipsUnreadableRows(t *testing.T) {
	store := &memSettings{}
	_ = store.SetString(context.Background(), settingKeyLocalGradePrefix+"broken", "{not json")
	_ = store.SetString(context.Background(), "unrelated.key", "x")
	s, _ := newTestService(t, store)
	assert.Empty(t, s.LocalGrades(context.Background()))
}
