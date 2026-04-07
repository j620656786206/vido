package subtitle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle/providers"
)

// --- Mock Provider ---

type mockProvider struct {
	name         string
	searchResult []providers.SubtitleResult
	searchErr    error
	downloadData []byte
	downloadErr  error
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Search(_ context.Context, _ providers.SubtitleQuery) ([]providers.SubtitleResult, error) {
	return m.searchResult, m.searchErr
}
func (m *mockProvider) Download(_ context.Context, _ string) ([]byte, error) {
	return m.downloadData, m.downloadErr
}

// --- Mock Repository (implements SubtitleStatusUpdater) ---

type mockStatusUpdater struct {
	lastStatus models.SubtitleStatus
	lastPath   string
	lastLang   string
	lastScore  float64
	updateErr  error
	callCount  int
}

func (r *mockStatusUpdater) UpdateSubtitleStatus(_ context.Context, _ string, status models.SubtitleStatus, path, lang string, score float64) error {
	r.lastStatus = status
	r.lastPath = path
	r.lastLang = lang
	r.lastScore = score
	r.callCount++
	return r.updateErr
}

// --- Helper ---

func newTestEngine(t *testing.T, provs []providers.SubtitleProvider, movieRepo *mockStatusUpdater) (*Engine, string) {
	t.Helper()
	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "Movie.2024.1080p.mkv")
	os.WriteFile(mediaPath, []byte("fake"), 0644)

	scorer := NewScorer(NewDefaultScorerConfig())
	converter, _ := NewConverter()
	placer := NewPlacer(DefaultPlacerConfig())

	if movieRepo == nil {
		movieRepo = &mockStatusUpdater{}
	}

	engine := NewEngine(provs, scorer, converter, placer, nil, movieRepo, &mockStatusUpdater{})
	return engine, mediaPath
}

// --- Tests ---

// Task 9.3: Full pipeline happy path
func TestEngine_Process_HappyPath(t *testing.T) {
	// Traditional Chinese subtitle content
	subContent := []byte("1\n00:00:01,000 --> 00:00:03,000\n這是繁體中文\n")

	prov := &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "zh-Hant", Filename: "sub.srt", Format: "srt", Downloads: 100},
		},
		downloadData: subContent,
	}

	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{prov}, nil)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p")

	assert.True(t, result.Success)
	assert.NotEmpty(t, result.SubtitlePath)
	assert.Equal(t, "assrt", result.ProviderUsed)
	assert.FileExists(t, result.SubtitlePath)
}

// Task 9.4: Download fallback
func TestEngine_Process_DownloadFallback(t *testing.T) {
	subContent := []byte("1\n00:00:01,000 --> 00:00:03,000\n繁體字幕\n")

	failProv := &mockProvider{
		name: "zimuku",
		searchResult: []providers.SubtitleResult{
			{ID: "fail-1", Source: "zimuku", Language: "zh-Hant", Filename: "sub1.srt", Format: "srt", Downloads: 200},
		},
		downloadErr: errors.New("download failed"),
	}

	successProv := &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "ok-1", Source: "assrt", Language: "zh-Hant", Filename: "sub2.srt", Format: "srt", Downloads: 100},
		},
		downloadData: subContent,
	}

	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{failProv, successProv}, nil)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p")

	assert.True(t, result.Success)
	assert.Equal(t, "assrt", result.ProviderUsed)
}

// Task 9.5: All downloads exhausted
func TestEngine_Process_AllDownloadsFailed(t *testing.T) {
	prov := &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "zh-Hant", Filename: "sub.srt", Format: "srt"},
		},
		downloadErr: errors.New("download failed"),
	}

	movieRepo := &mockStatusUpdater{}
	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{prov}, movieRepo)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p")

	assert.False(t, result.Success)
	assert.ErrorIs(t, result.Error, ErrAllDownloadsFailed)
	assert.Equal(t, models.SubtitleStatusNotFound, movieRepo.lastStatus)
}

// Task 9.6: Partial provider failure
func TestEngine_Process_PartialProviderFailure(t *testing.T) {
	subContent := []byte("1\n00:00:01,000 --> 00:00:03,000\n繁體字幕\n")

	failProv := &mockProvider{
		name:      "zimuku",
		searchErr: errors.New("provider error"),
	}

	successProv := &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "zh-Hant", Filename: "sub.srt", Format: "srt", Downloads: 100},
		},
		downloadData: subContent,
	}

	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{failProv, successProv}, nil)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p")

	assert.True(t, result.Success, "should succeed with partial provider results")
}

// Task 9.7: All providers fail
func TestEngine_Process_AllProvidersFail(t *testing.T) {
	prov1 := &mockProvider{name: "assrt", searchErr: errors.New("fail 1")}
	prov2 := &mockProvider{name: "zimuku", searchErr: errors.New("fail 2")}

	movieRepo := &mockStatusUpdater{}
	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{prov1, prov2}, movieRepo)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p")

	assert.False(t, result.Success)
	assert.Equal(t, models.SubtitleStatusNotFound, movieRepo.lastStatus)
}

// Task 9.8: Context cancellation
func TestEngine_Process_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	prov := &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "zh-Hant"},
		},
	}

	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{prov}, nil)

	result := engine.Process(ctx, "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p")

	// Either search fails with cancelled context or downloads fail
	assert.False(t, result.Success)
}

// Task 9.10: zh-Hans triggers conversion
func TestEngine_ConvertIfNeeded_SimplifiedConverted(t *testing.T) {
	converter, _ := NewConverter()
	engine := &Engine{converter: converter}

	// Simplified Chinese content
	data := []byte("这是简体中文测试内容")
	result, lang, err := engine.convertIfNeeded(data, ConvertAuto)
	require.NoError(t, err)
	assert.Equal(t, LangTraditional, lang)
	assert.NotEqual(t, string(data), string(result), "simplified should be converted")
}

// Task 9.11: zh-Hant skips conversion
func TestEngine_ConvertIfNeeded_TraditionalPassthrough(t *testing.T) {
	converter, _ := NewConverter()
	engine := &Engine{converter: converter}

	data := []byte("這是繁體中文測試內容")
	result, lang, err := engine.convertIfNeeded(data, ConvertAuto)
	require.NoError(t, err)
	assert.Equal(t, LangTraditional, lang)
	assert.Equal(t, string(data), string(result), "traditional should pass through")
}

// No results
func TestEngine_Process_NoResults(t *testing.T) {
	prov := &mockProvider{
		name:         "assrt",
		searchResult: nil, // no results
	}

	movieRepo := &mockStatusUpdater{}
	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{prov}, movieRepo)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p")

	assert.False(t, result.Success)
	assert.ErrorIs(t, result.Error, ErrNoResults)
}

// --- Mock Terminology Service ---

type mockTerminologyService struct {
	correctResult string
	correctErr    error
	configured    bool
	callCount     int
}

var _ services.TerminologyCorrectionServiceInterface = (*mockTerminologyService)(nil)

func (m *mockTerminologyService) Correct(_ context.Context, content string) (string, error) {
	m.callCount++
	if m.correctErr != nil {
		return content, m.correctErr
	}
	if m.correctResult != "" {
		return m.correctResult, nil
	}
	return content, nil
}

func (m *mockTerminologyService) IsConfigured() bool {
	return m.configured
}

// Story 9.1 Task 4.2: AI step is skipped when not configured
func TestEngine_Process_AICorrection_SkippedWhenNotConfigured(t *testing.T) {
	subContent := []byte("1\n00:00:01,000 --> 00:00:03,000\n這個軟件很好用\n")

	prov := &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "zh-Hans", Filename: "sub.srt", Format: "srt", Downloads: 100},
		},
		downloadData: subContent,
	}

	mockTermSvc := &mockTerminologyService{configured: false}
	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{prov}, nil)
	engine.SetTerminologyService(mockTermSvc)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p")

	assert.True(t, result.Success)
	assert.Equal(t, 0, mockTermSvc.callCount, "AI correction should not be called when not configured")
}

// Story 9.1 Task 4.2: AI step is skipped when no service set
func TestEngine_Process_AICorrection_SkippedWhenNilService(t *testing.T) {
	subContent := []byte("1\n00:00:01,000 --> 00:00:03,000\n這個軟件很好用\n")

	prov := &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "zh-Hans", Filename: "sub.srt", Format: "srt", Downloads: 100},
		},
		downloadData: subContent,
	}

	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{prov}, nil)
	// No SetTerminologyService call — service is nil

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p")

	assert.True(t, result.Success, "pipeline should succeed without AI correction service")
}

// Story 9.1: AI correction is applied when configured and Chinese content detected
func TestEngine_Process_AICorrection_Applied(t *testing.T) {
	subContent := []byte("1\n00:00:01,000 --> 00:00:03,000\n这个软件很好用\n")

	prov := &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "zh-Hans", Filename: "sub.srt", Format: "srt", Downloads: 100},
		},
		downloadData: subContent,
	}

	mockTermSvc := &mockTerminologyService{
		configured:    true,
		correctResult: "1\n00:00:01,000 --> 00:00:03,000\n這個軟體很好用\n",
	}
	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{prov}, nil)
	engine.SetTerminologyService(mockTermSvc)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p")

	assert.True(t, result.Success)
	assert.Equal(t, 1, mockTermSvc.callCount, "AI correction should be called once")
}

// Story 9.1: AI correction failure falls back to OpenCC output (AC #4)
func TestEngine_Process_AICorrection_FallbackOnError(t *testing.T) {
	subContent := []byte("1\n00:00:01,000 --> 00:00:03,000\n这个软件很好用\n")

	prov := &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "zh-Hans", Filename: "sub.srt", Format: "srt", Downloads: 100},
		},
		downloadData: subContent,
	}

	mockTermSvc := &mockTerminologyService{
		configured: true,
		correctErr: errors.New("API error"),
	}
	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{prov}, nil)
	engine.SetTerminologyService(mockTermSvc)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p")

	assert.True(t, result.Success, "pipeline should succeed even if AI correction fails")
	assert.Equal(t, 1, mockTermSvc.callCount, "AI correction should have been attempted")
}

// Story 9.1: AI correction is skipped for CN content (ConvertNever policy)
func TestEngine_Process_AICorrection_SkippedForCNContent(t *testing.T) {
	subContent := []byte("1\n00:00:01,000 --> 00:00:03,000\n这是简体中文\n")

	prov := &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "zh-Hans", Filename: "sub.srt", Format: "srt", Downloads: 100},
		},
		downloadData: subContent,
	}

	mockTermSvc := &mockTerminologyService{configured: true}
	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{prov}, nil)
	engine.SetTerminologyService(mockTermSvc)

	// CN production country → ConvertNever
	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p",
		ProcessOptions{ProductionCountry: "CN"})

	assert.True(t, result.Success)
	assert.Equal(t, 0, mockTermSvc.callCount, "AI correction should be skipped for CN content")
}

// Story 9.1 TA: AI correction is skipped for non-Chinese content (English subs)
func TestEngine_Process_AICorrection_SkippedForNonChinese(t *testing.T) {
	subContent := []byte("1\n00:00:01,000 --> 00:00:03,000\nThis is English subtitle text\n")

	prov := &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "en", Filename: "sub.srt", Format: "srt", Downloads: 100},
		},
		downloadData: subContent,
	}

	mockTermSvc := &mockTerminologyService{configured: true}
	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{prov}, nil)
	engine.SetTerminologyService(mockTermSvc)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p")

	assert.True(t, result.Success)
	assert.Equal(t, 0, mockTermSvc.callCount, "AI correction should be skipped for non-Chinese content")
}

// Story 9.1 TA: AI correction applies to Traditional Chinese content (post-OpenCC)
func TestEngine_Process_AICorrection_AppliesTraditionalChinese(t *testing.T) {
	// Content already in Traditional Chinese (from OpenCC or original)
	subContent := []byte("1\n00:00:01,000 --> 00:00:03,000\n這個軟件在操作系統上運行\n")

	prov := &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "zh-Hant", Filename: "sub.srt", Format: "srt", Downloads: 100},
		},
		downloadData: subContent,
	}

	mockTermSvc := &mockTerminologyService{
		configured:    true,
		correctResult: "1\n00:00:01,000 --> 00:00:03,000\n這個軟體在作業系統上運行\n",
	}
	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{prov}, nil)
	engine.SetTerminologyService(mockTermSvc)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p")

	assert.True(t, result.Success)
	assert.Equal(t, 1, mockTermSvc.callCount, "AI correction should apply to Traditional Chinese content")
}

// Story 9.1 TA: SetTerminologyService allows late binding
func TestEngine_SetTerminologyService(t *testing.T) {
	engine := &Engine{}
	assert.Nil(t, engine.terminologyService)

	svc := &mockTerminologyService{configured: true}
	engine.SetTerminologyService(svc)
	assert.Equal(t, svc, engine.terminologyService)
}
