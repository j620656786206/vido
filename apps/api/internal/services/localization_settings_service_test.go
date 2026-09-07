package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
)

// ─── sub-7-4 AC #3: precedence settings > env > default, validation ─────────

func TestLocalizationSettings_Precedence(t *testing.T) {
	ctx := context.Background()

	t.Run("nothing set → default", func(t *testing.T) {
		repo := &MockSettingsRepository{}
		repo.On("GetString", mock.Anything, SettingKeyLocalizationLevel).Return("", fmt.Errorf("setting with key %s not found", SettingKeyLocalizationLevel))
		got := NewLocalizationSettingsService(repo, "", nil).Resolve(ctx)
		assert.Equal(t, LocalizationSettings{Level: prompts.LocalizationStandard, Source: LocalizationSourceDefault}, got)
	})
	t.Run("env only", func(t *testing.T) {
		repo := &MockSettingsRepository{}
		repo.On("GetString", mock.Anything, SettingKeyLocalizationLevel).Return("", fmt.Errorf("setting with key %s not found", SettingKeyLocalizationLevel))
		got := NewLocalizationSettingsService(repo, "ott", nil).Resolve(ctx)
		assert.Equal(t, LocalizationSettings{Level: prompts.LocalizationOTT, Source: LocalizationSourceEnv}, got)
	})
	t.Run("settings beats env", func(t *testing.T) {
		repo := &MockSettingsRepository{}
		repo.On("GetString", mock.Anything, SettingKeyLocalizationLevel).Return("literal", nil)
		got := NewLocalizationSettingsService(repo, "ott", nil).Resolve(ctx)
		assert.Equal(t, LocalizationSettings{Level: prompts.LocalizationLiteral, Source: LocalizationSourceSettings}, got)
	})
	t.Run("garbage in settings falls through to env", func(t *testing.T) {
		repo := &MockSettingsRepository{}
		repo.On("GetString", mock.Anything, SettingKeyLocalizationLevel).Return("netflix", nil)
		got := NewLocalizationSettingsService(repo, "literal", nil).Resolve(ctx)
		assert.Equal(t, LocalizationSettings{Level: prompts.LocalizationLiteral, Source: LocalizationSourceEnv}, got)
	})
	t.Run("repo error falls through, never fails the run", func(t *testing.T) {
		repo := &MockSettingsRepository{}
		repo.On("GetString", mock.Anything, SettingKeyLocalizationLevel).Return("", errors.New("db locked"))
		assert.Equal(t, prompts.LocalizationStandard, NewLocalizationSettingsService(repo, "", nil).Level(ctx))
	})
	t.Run("nil repo", func(t *testing.T) {
		assert.Equal(t, prompts.LocalizationOTT, NewLocalizationSettingsService(nil, "ott", nil).Level(ctx))
	})
}

func TestLocalizationSettings_Set(t *testing.T) {
	ctx := context.Background()
	repo := &MockSettingsRepository{}
	repo.On("Set", mock.Anything, mock.MatchedBy(func(s *models.Setting) bool {
		return s.Key == SettingKeyLocalizationLevel && s.Value == "ott" && s.Type == "string"
	})).Return(nil)
	svc := NewLocalizationSettingsService(repo, "", nil)

	saved, err := svc.Set(ctx, " OTT ")
	require.NoError(t, err)
	assert.Equal(t, LocalizationSettings{Level: prompts.LocalizationOTT, Source: LocalizationSourceSettings}, saved)
	repo.AssertExpectations(t)

	_, err = svc.Set(ctx, "netflix")
	var ve *models.ValidationError
	require.ErrorAs(t, err, &ve, "an unknown taste is rejected, never silently defaulted")

	_, err = NewLocalizationSettingsService(nil, "", nil).Set(ctx, "ott")
	require.Error(t, err)
}

// ─── the ASR leg: level reaches the prompt, lexicon reaches the output ─────

func TestTranslateSRT_LocalizationLevelAndLexiconOnTheASRLeg(t *testing.T) {
	mockProvider := &translationIntegrationMock{response: "[1] 這個視頻的質量很好"}
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(NewTranslationService(mockProvider, nil))
	svc.SetLocalizationLevelSource(func(context.Context) prompts.LocalizationLevel { return prompts.LocalizationOTT })

	tmpDir := t.TempDir()
	zhPath, _, err := svc.translateSRT(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidA,
		"1\n00:00:01,000 --> 00:00:04,000\nThe video quality is great\n", filepath.Join(tmpDir, "movie.mkv"), tmpDir)
	require.NoError(t, err)

	assert.Contains(t, mockProvider.lastSystemPrompt, "## Localization style (ott)")
	assert.True(t, strings.HasPrefix(mockProvider.lastSystemPrompt, prompts.ComposeInvariantSystemPrompt(prompts.LocalizationOTT)))

	written := readFileString(t, zhPath)
	assert.Contains(t, written, "這個影片的品質很好", "lexicon applied after OpenCC on the ASR leg too")
	assert.Contains(t, written, "00:00:01,000 --> 00:00:04,000", "timestamps untouched")
}

func TestTranslateSRT_LexiconSkipsMainlandContent(t *testing.T) {
	mockProvider := &translationIntegrationMock{response: "[1] 這個視頻的質量很好"}
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(NewTranslationService(mockProvider, nil))
	svc.SetSubtitleStateReader(&metadataMovieReader{movie: &models.Movie{
		ID: uuidA, Title: "讓子彈飛", TMDbID: models.NewNullInt64(1),
		ProductionCountries: []models.ProductionCountry{{ISO3166_1: "CN", Name: "China"}},
	}})

	tmpDir := t.TempDir()
	zhPath, _, err := svc.translateSRT(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidA,
		"1\n00:00:01,000 --> 00:00:04,000\nThe video quality is great\n", filepath.Join(tmpDir, "movie.mkv"), tmpDir)
	require.NoError(t, err)
	assert.Contains(t, readFileString(t, zhPath), "這個視頻的質量很好", "CN content keeps its own vocabulary")
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}

// CR round: a level pinned on the ctx by the pipeline (the one its run row
// recorded) beats a fresh settings read — a dial flipped mid-job must not make
// the recorded provenance lie.
func TestTranslateSRT_PinnedContextLevelBeatsTheSource(t *testing.T) {
	mockProvider := &translationIntegrationMock{response: "[1] 你好"}
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(NewTranslationService(mockProvider, nil))
	svc.SetLocalizationLevelSource(func(context.Context) prompts.LocalizationLevel { return prompts.LocalizationOTT })

	ctx := prompts.ContextWithLocalizationLevel(context.Background(), prompts.LocalizationLiteral)
	tmpDir := t.TempDir()
	_, _, err := svc.translateSRT(ctx, "job-1", models.SubtitleRunMediaMovie, uuidA,
		"1\n00:00:01,000 --> 00:00:04,000\nHello\n", filepath.Join(tmpDir, "movie.mkv"), tmpDir)
	require.NoError(t, err)
	assert.Contains(t, mockProvider.lastSystemPrompt, "## Localization style (literal)")
}

// CR round: a harvested rendering goes through the lexicon before it becomes
// a per-show glossary row on the ASR leg too.
func TestHarvestGlossaryTerms_AppliesTheLexicon(t *testing.T) {
	repo := &stubGlossaryRepo{}
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetGlossaryRepository(repo)
	n := svc.harvestGlossaryTerms(context.Background(), "m1", map[string]string{"smartphone": "智能手機"}, []string{"US"})
	assert.Equal(t, 1, n)
	assert.Equal(t, "智慧型手機", repo.inserted["smartphone"])

	cn := &stubGlossaryRepo{}
	svc.SetGlossaryRepository(cn)
	svc.harvestGlossaryTerms(context.Background(), "m2", map[string]string{"smartphone": "智能手機"}, []string{"CN"})
	assert.Equal(t, "智能手機", cn.inserted["smartphone"], "CN content keeps its vocabulary")
}
