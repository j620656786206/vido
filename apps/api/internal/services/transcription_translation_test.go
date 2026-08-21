package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
)

// Media-id fixture convention (9R-18 AC 7): media ids are UUID STRINGS —
// mirror the prod creation path (uuid.New().String()); do NOT invent numeric
// ids (fixtures reuse the uuid* consts from generation_batch_test.go).

// ─── ParseSRTToTranslationBlocks Tests (P0) ─────────────────────────────────

func TestParseSRTToTranslationBlocks_Basic(t *testing.T) {
	input := "1\n00:00:01,000 --> 00:00:04,000\nHello world\n\n2\n00:00:05,000 --> 00:00:08,000\nGoodbye\n"

	blocks, err := ParseSRTToTranslationBlocks(input)
	require.NoError(t, err)
	require.Len(t, blocks, 2)

	assert.Equal(t, 1, blocks[0].Index)
	assert.Equal(t, "00:00:01,000", blocks[0].Start)
	assert.Equal(t, "00:00:04,000", blocks[0].End)
	assert.Equal(t, "Hello world", blocks[0].Text)

	assert.Equal(t, 2, blocks[1].Index)
	assert.Equal(t, "00:00:05,000", blocks[1].Start)
	assert.Equal(t, "00:00:08,000", blocks[1].End)
	assert.Equal(t, "Goodbye", blocks[1].Text)
}

func TestParseSRTToTranslationBlocks_Empty(t *testing.T) {
	blocks, err := ParseSRTToTranslationBlocks("")
	require.NoError(t, err)
	assert.Nil(t, blocks)
}

func TestParseSRTToTranslationBlocks_BOM(t *testing.T) {
	input := "\xEF\xBB\xBF1\n00:00:01,000 --> 00:00:04,000\nHello\n"

	blocks, err := ParseSRTToTranslationBlocks(input)
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	assert.Equal(t, 1, blocks[0].Index)
	assert.Equal(t, "Hello", blocks[0].Text)
}

func TestParseSRTToTranslationBlocks_WindowsCRLF(t *testing.T) {
	input := "1\r\n00:00:01,000 --> 00:00:04,000\r\nHello\r\n\r\n2\r\n00:00:05,000 --> 00:00:08,000\r\nWorld\r\n"

	blocks, err := ParseSRTToTranslationBlocks(input)
	require.NoError(t, err)
	require.Len(t, blocks, 2)
	assert.Equal(t, "Hello", blocks[0].Text)
	assert.Equal(t, "World", blocks[1].Text)
}

func TestParseSRTToTranslationBlocks_MultiLineText(t *testing.T) {
	input := "1\n00:00:01,000 --> 00:00:04,000\nLine one\nLine two\n\n"

	blocks, err := ParseSRTToTranslationBlocks(input)
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	assert.Equal(t, "Line one\nLine two", blocks[0].Text)
}

func TestParseSRTToTranslationBlocks_ExtraBlankLines(t *testing.T) {
	input := "\n\n1\n00:00:01,000 --> 00:00:04,000\nHello\n\n\n\n2\n00:00:05,000 --> 00:00:08,000\nWorld\n"

	blocks, err := ParseSRTToTranslationBlocks(input)
	require.NoError(t, err)
	require.Len(t, blocks, 2)
}

func TestParseSRTToTranslationBlocks_TimestampPreservation(t *testing.T) {
	// AC #3: timestamps must be preserved exactly
	input := "1\n01:23:45,678 --> 01:23:50,999\nTest\n"

	blocks, err := ParseSRTToTranslationBlocks(input)
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	assert.Equal(t, "01:23:45,678", blocks[0].Start)
	assert.Equal(t, "01:23:50,999", blocks[0].End)
}

// ─── serializeTranslationBlocksToSRT Tests (P0) ─────────────────────────────

func TestSerializeTranslationBlocksToSRT_Basic(t *testing.T) {
	blocks := []TranslationBlock{
		{Index: 1, Start: "00:00:01,000", End: "00:00:04,000", Text: "你好"},
		{Index: 2, Start: "00:00:05,000", End: "00:00:08,000", Text: "世界"},
	}

	result := serializeTranslationBlocksToSRT(blocks)

	expected := "1\n00:00:01,000 --> 00:00:04,000\n你好\n\n2\n00:00:05,000 --> 00:00:08,000\n世界\n"
	assert.Equal(t, expected, result)
}

func TestSerializeTranslationBlocksToSRT_Empty(t *testing.T) {
	result := serializeTranslationBlocksToSRT(nil)
	assert.Equal(t, "", result)
}

func TestSerializeTranslationBlocksToSRT_MultiLine(t *testing.T) {
	blocks := []TranslationBlock{
		{Index: 1, Start: "00:00:01,000", End: "00:00:04,000", Text: "第一行\n第二行"},
	}

	result := serializeTranslationBlocksToSRT(blocks)
	assert.Contains(t, result, "第一行\n第二行")
}

func TestParseSerialize_RoundTrip(t *testing.T) {
	// P0: Parse SRT → serialize → re-parse must produce identical blocks
	original := "1\n00:00:01,000 --> 00:00:04,500\nHello world\n\n2\n00:00:05,000 --> 00:00:09,200\nLine one\nLine two\n\n3\n01:30:00,000 --> 01:30:05,500\nThe end\n"

	blocks1, err := ParseSRTToTranslationBlocks(original)
	require.NoError(t, err)

	serialized := serializeTranslationBlocksToSRT(blocks1)

	blocks2, err := ParseSRTToTranslationBlocks(serialized)
	require.NoError(t, err)

	require.Len(t, blocks2, len(blocks1))
	for i := range blocks1 {
		assert.Equal(t, blocks1[i].Index, blocks2[i].Index, "block %d index", i)
		assert.Equal(t, blocks1[i].Start, blocks2[i].Start, "block %d start", i)
		assert.Equal(t, blocks1[i].End, blocks2[i].End, "block %d end", i)
		assert.Equal(t, blocks1[i].Text, blocks2[i].Text, "block %d text", i)
	}
}

// ─── SetTranslationService Tests (P1) ────────────────────────────────────────

func TestSetTranslationService(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	assert.Nil(t, svc.translationService)

	mockProvider := &mockTranslationCompleter{}
	ts := NewTranslationService(mockProvider, nil)
	svc.SetTranslationService(ts)

	assert.NotNil(t, svc.translationService)
	assert.True(t, svc.translationService.IsConfigured())
}

func TestSetTranslationService_Nil(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(nil)
	assert.Nil(t, svc.translationService)
}

// ─── WithTranslation Option Tests (P1) ──────────────────────────────────────

func TestWithTranslation_Option(t *testing.T) {
	cfg := &transcriptionConfig{}
	assert.False(t, cfg.translate)

	opt := WithTranslation()
	opt(cfg)
	assert.True(t, cfg.translate)
}

// ─── translateSRT Integration Tests (P1) ─────────────────────────────────────

func TestTranslateSRT_Success(t *testing.T) {
	// Create a mock translation service that returns Chinese text
	mockProvider := &translationIntegrationMock{
		response: "[1] 你好世界\n[2] 再見",
	}
	translationSvc := NewTranslationService(mockProvider, nil)

	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(translationSvc)

	// Input English SRT
	srtContent := "1\n00:00:01,000 --> 00:00:04,000\nHello world\n\n2\n00:00:05,000 --> 00:00:08,000\nGoodbye\n"

	// Create temp dir for output
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "Movie.2024.1080p.mkv")

	zhPath, _, err := svc.translateSRT(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidA, srtContent, filePath, tmpDir)
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, zhPath)
	assert.True(t, strings.HasSuffix(zhPath, ".zh-Hant.srt"), "output should end with .zh-Hant.srt")

	// Verify file content
	content, err := os.ReadFile(zhPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "你好世界")
	assert.Contains(t, string(content), "再見")
	// Timestamps must be preserved (AC #3)
	assert.Contains(t, string(content), "00:00:01,000 --> 00:00:04,000")
	assert.Contains(t, string(content), "00:00:05,000 --> 00:00:08,000")
}

func TestTranslateSRT_FilenameConvention(t *testing.T) {
	mockProvider := &translationIntegrationMock{
		response: "[1] 翻譯",
	}
	translationSvc := NewTranslationService(mockProvider, nil)

	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(translationSvc)

	srtContent := "1\n00:00:01,000 --> 00:00:04,000\nHello\n"
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "The.Movie.2024.1080p.BluRay.mkv")

	zhPath, _, err := svc.translateSRT(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidA, srtContent, filePath, tmpDir)
	require.NoError(t, err)

	// Should follow naming convention: {basename}.zh-Hant.srt
	expectedName := "The.Movie.2024.1080p.BluRay.zh-Hant.srt"
	assert.Equal(t, expectedName, filepath.Base(zhPath))
}

func TestTranslateSRT_EmptySRT(t *testing.T) {
	mockProvider := &translationIntegrationMock{}
	translationSvc := NewTranslationService(mockProvider, nil)

	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(translationSvc)

	_, _, err := svc.translateSRT(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidA, "", "test.mkv", t.TempDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no subtitle blocks")
}

func TestTranslateSRT_PartialFailurePreservesEnglish(t *testing.T) {
	// AC #5: partial failure keeps English text for failed blocks
	// Create 15 blocks — first batch succeeds, second batch fails
	var srtContent strings.Builder
	for i := 1; i <= 15; i++ {
		srtContent.WriteString(fmt.Sprintf("%d\n00:00:%02d,000 --> 00:00:%02d,500\nEnglish line %d\n\n", i, i, i, i))
	}

	// Mock: first batch returns translations, second fails
	failMock := &translationFailOnSecondIntegrationMock{}
	for i := 1; i <= 10; i++ {
		failMock.firstResponse += fmt.Sprintf("[%d] 中文第%d行\n", i, i)
	}
	translationSvc := NewTranslationService(failMock, nil)

	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(translationSvc)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.mkv")

	zhPath, _, err := svc.translateSRT(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidA, srtContent.String(), filePath, tmpDir)
	require.NoError(t, err)

	content, err := os.ReadFile(zhPath)
	require.NoError(t, err)
	contentStr := string(content)

	// First 10 blocks should be translated
	assert.Contains(t, contentStr, "中文第1行")
	// Blocks 11-15 should retain English (AC #5)
	assert.Contains(t, contentStr, "English line 11")
	assert.Contains(t, contentStr, "English line 15")
}

func TestTranslateSRT_ProgressCallback(t *testing.T) {
	mockProvider := &translationIntegrationMock{
		response: "[1] 翻譯",
	}
	translationSvc := NewTranslationService(mockProvider, nil)

	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(translationSvc)

	srtContent := "1\n00:00:01,000 --> 00:00:04,000\nHello\n"
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.mkv")

	// translateSRT should complete without panic even with nil sseHub
	zhPath, _, err := svc.translateSRT(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidA, srtContent, filePath, tmpDir)
	require.NoError(t, err)
	assert.FileExists(t, zhPath)
}

// ─── EventType Test (P2) ────────────────────────────────────────────────────

func TestTranscriptionTranslatingEventType(t *testing.T) {
	assert.Equal(t, "translation_progress", string(EventTranscriptionTranslating))
}

// ─── Helpers ────────────────────────────────────────────────────────────────

type translationIntegrationMock struct {
	response  string
	callCount int
	// 9R-8: the system prompt of the LAST batch, so a test can assert what the
	// FR26 media-context section actually put in front of the model.
	lastSystemPrompt string
}

func (m *translationIntegrationMock) CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	m.callCount++
	m.lastSystemPrompt = systemPrompt
	return m.response, nil
}

type translationFailOnSecondIntegrationMock struct {
	firstResponse string
	callCount     int
}

func (m *translationFailOnSecondIntegrationMock) CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	m.callCount++
	if m.callCount == 1 {
		return m.firstResponse, nil
	}
	return "", context.DeadlineExceeded
}

// ─── 9R-8: FR26 media-context on the ASR leg ────────────────────────────────
//
// The extract leg has fed BuildMetadataSection into its system blocks since
// sub-1-5a (subtitle/pipeline.go buildSystemBlocks). The ASR leg — the majority
// path at 68.3% of the sampled library — translated with no idea which work the
// cues belong to. These tests pin the wiring and, just as importantly, pin the
// no-metadata path as BYTE-IDENTICAL to the pre-9R-8 prompt.

// metadataMovieReader is a SubtitleStateReader that serves one movie row (or an
// error). callCount proves the metadata path does not re-read what it already has.
type metadataMovieReader struct {
	movie     *models.Movie
	err       error
	callCount int
}

func (r *metadataMovieReader) FindByID(ctx context.Context, id string) (*models.Movie, error) {
	r.callCount++
	if r.err != nil {
		return nil, r.err
	}
	return r.movie, nil
}

// metadataEpisodeReader is an EpisodeSubtitleStateReader for the episode leg.
type metadataEpisodeReader struct {
	episode   *models.Episode
	err       error
	callCount int
}

func (r *metadataEpisodeReader) FindByID(ctx context.Context, id string) (*models.Episode, error) {
	r.callCount++
	if r.err != nil {
		return nil, r.err
	}
	return r.episode, nil
}

func fixtureMovie() *models.Movie {
	return &models.Movie{
		ID:            uuidA,
		Title:         "The Invisible Agent",
		OriginalTitle: models.NewNullString("Der Unsichtbare Agent"),
		ReleaseDate:   "1998-07-14",
		Genres:        []string{"Science Fiction", "Thriller"},
		Overview:      models.NewNullString("A field operative discovers\na way to vanish."),
		ProductionCountries: []models.ProductionCountry{
			{ISO3166_1: "DE", Name: "Germany"},
		},
	}
}

func TestTranslateSRT_MovieMetadataReachesTheSystemPrompt(t *testing.T) {
	mockProvider := &translationIntegrationMock{response: "[1] 你好世界"}
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(NewTranslationService(mockProvider, nil))
	svc.SetSubtitleStateReader(&metadataMovieReader{movie: fixtureMovie()})

	tmpDir := t.TempDir()
	_, _, err := svc.translateSRT(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidA,
		"1\n00:00:01,000 --> 00:00:04,000\nHello world\n", filepath.Join(tmpDir, "movie.mkv"), tmpDir)
	require.NoError(t, err)

	sys := mockProvider.lastSystemPrompt
	assert.Contains(t, sys, "## Media context — background only, do NOT translate or output this section:")
	assert.Contains(t, sys, "- Title: The Invisible Agent")
	assert.Contains(t, sys, "- Original title: Der Unsichtbare Agent")
	assert.Contains(t, sys, "- Year: 1998")
	assert.Contains(t, sys, "- Genres: Science Fiction, Thriller")
	assert.Contains(t, sys, "- Production countries: DE")
	// collapseLines folds the multi-line overview onto one line.
	assert.Contains(t, sys, "- Overview: A field operative discovers a way to vanish.")
	// The invariant translator prompt stays FIRST — same stable-prefix order as
	// the extract leg's buildSystemBlocks (block[0] invariant, block[1] per-show).
	assert.True(t, strings.HasPrefix(sys, prompts.SubtitleTranslatorSystemPrompt),
		"the invariant system prompt must stay the stable prefix")
}

func TestTranslateSRT_MetadataLookupFailureKeepsThePromptByteIdentical(t *testing.T) {
	mockProvider := &translationIntegrationMock{response: "[1] 你好世界"}
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(NewTranslationService(mockProvider, nil))
	// Rule 13 fail-soft. NOTE the repo contract: a missing row comes back as an
	// error WRAPPING sql.ErrNoRows, never (nil, nil) — movie_repository.go:106.
	svc.SetSubtitleStateReader(&metadataMovieReader{
		err: fmt.Errorf("movie with id %s not found: %w", uuidA, sql.ErrNoRows),
	})

	tmpDir := t.TempDir()
	zhPath, _, err := svc.translateSRT(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidA,
		"1\n00:00:01,000 --> 00:00:04,000\nHello world\n", filepath.Join(tmpDir, "movie.mkv"), tmpDir)
	require.NoError(t, err, "a metadata miss must never fail the translation")
	assert.FileExists(t, zhPath)

	assert.Equal(t, prompts.SubtitleTranslatorSystemPrompt, mockProvider.lastSystemPrompt,
		"no metadata must yield the pre-9R-8 prompt byte-for-byte")
}

func TestTranslateSRT_NoReaderWiredKeepsThePromptByteIdentical(t *testing.T) {
	mockProvider := &translationIntegrationMock{response: "[1] 你好世界"}
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(NewTranslationService(mockProvider, nil))
	// No readers wired at all — the pre-9R-8 deployment shape.

	tmpDir := t.TempDir()
	_, _, err := svc.translateSRT(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidA,
		"1\n00:00:01,000 --> 00:00:04,000\nHello world\n", filepath.Join(tmpDir, "movie.mkv"), tmpDir)
	require.NoError(t, err)

	assert.Equal(t, prompts.SubtitleTranslatorSystemPrompt, mockProvider.lastSystemPrompt)
}

// TestSubtitleTranslatorPromptVersion_NotBumpedBy9R8 pins the deliberate
// non-bump. 9R-8 adds NO text to any prompt surface in subtitle_translator.go —
// it composes the already-shipped BuildMetadataSection at the ASR call site.
// Bumping would re-key the EXTRACT leg's whole segment cache (RunVersion embeds
// the prompt version) to re-translate a library that gained nothing, while the
// ASR leg — which has no segment cache at all — would gain nothing either.
func TestSubtitleTranslatorPromptVersion_NotBumpedBy9R8(t *testing.T) {
	assert.Equal(t, "m1-v2", prompts.SubtitleTranslatorPromptVersion)
}

// metadataSeriesReader is a SeriesMetadataReader serving one series row.
type metadataSeriesReader struct {
	series    *models.Series
	err       error
	callCount int
	lastID    string
}

func (r *metadataSeriesReader) FindByID(ctx context.Context, id string) (*models.Series, error) {
	r.callCount++
	r.lastID = id
	if r.err != nil {
		return nil, r.err
	}
	return r.series, nil
}

// TestTranslateSRT_EpisodeUsesParentSeriesMetadata pins BOTH halves of AC #1:
// an episode translates against its SHOW's context (mirroring the extract leg's
// loadEpisode decision), and the episode row is read exactly ONCE — the
// metadata path reuses the series id glossaryMediaKey already resolved.
func TestTranslateSRT_EpisodeUsesParentSeriesMetadata(t *testing.T) {
	mockProvider := &translationIntegrationMock{response: "[1] 你好世界"}
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(NewTranslationService(mockProvider, nil))

	episodes := &metadataEpisodeReader{episode: &models.Episode{
		ID:       uuidB,
		SeriesID: uuidC,
		// An episode-level title that must NOT reach the prompt.
		Title: models.NewNullString("The Body"),
	}}
	series := &metadataSeriesReader{series: &models.Series{
		ID:            uuidC,
		Title:         "Buffy the Vampire Slayer",
		OriginalTitle: models.NewNullString("Buffy the Vampire Slayer"),
		FirstAirDate:  "1997-03-10",
		Genres:        []string{"Drama", "Fantasy"},
		Overview:      models.NewNullString("A teenager battles the forces of darkness."),
	}}
	svc.SetEpisodeSubtitleStateReader(episodes)
	svc.SetSeriesMetadataReader(series)

	tmpDir := t.TempDir()
	_, _, err := svc.translateSRT(context.Background(), "job-1", models.SubtitleRunMediaEpisode, uuidB,
		"1\n00:00:01,000 --> 00:00:04,000\nHello world\n", filepath.Join(tmpDir, "s05e16.mkv"), tmpDir)
	require.NoError(t, err)

	sys := mockProvider.lastSystemPrompt
	assert.Contains(t, sys, "- Title: Buffy the Vampire Slayer")
	assert.Contains(t, sys, "- Year: 1997")
	assert.Contains(t, sys, "- Genres: Drama, Fantasy")
	// The episode's OWN title must not leak in. Paired with the positive Title
	// assertion above so this cannot pass by the section being absent entirely
	// (CR L2 — a lone NotContains proves nothing).
	assert.NotContains(t, sys, "The Body",
		"episodes carry SHOW-level context — a per-episode title would diverge from the extract leg")

	assert.Equal(t, uuidC, series.lastID, "the series read must key on the PARENT series id")
	assert.Equal(t, 1, series.callCount)
	assert.Equal(t, 1, episodes.callCount,
		"glossaryMediaKey already resolved the series id — the metadata path must not re-read the episode")
}

func TestTranslateSRT_EpisodeWithoutSeriesReaderStaysByteIdentical(t *testing.T) {
	mockProvider := &translationIntegrationMock{response: "[1] 你好世界"}
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(NewTranslationService(mockProvider, nil))
	svc.SetEpisodeSubtitleStateReader(&metadataEpisodeReader{
		episode: &models.Episode{ID: uuidB, SeriesID: uuidC},
	})
	// No series reader wired — the pre-9R-8 deployment shape.

	tmpDir := t.TempDir()
	_, _, err := svc.translateSRT(context.Background(), "job-1", models.SubtitleRunMediaEpisode, uuidB,
		"1\n00:00:01,000 --> 00:00:04,000\nHello world\n", filepath.Join(tmpDir, "s05e16.mkv"), tmpDir)
	require.NoError(t, err)

	assert.Equal(t, prompts.SubtitleTranslatorSystemPrompt, mockProvider.lastSystemPrompt)
}

// TestTranslateSRT_UnresolvedParentSeriesSkipsTheSeriesLookup covers CR M4:
// glossaryMediaKey fails soft to the EPISODE's own id when the parent lookup
// misses. Querying the series table with that id would log a series_id that is
// really an episode id and send an operator hunting a row that never existed
// (the 9R-10a CR M1 class), so the metadata path must not make the call at all.
func TestTranslateSRT_UnresolvedParentSeriesSkipsTheSeriesLookup(t *testing.T) {
	mockProvider := &translationIntegrationMock{response: "[1] 你好世界"}
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(NewTranslationService(mockProvider, nil))

	episodes := &metadataEpisodeReader{err: fmt.Errorf("episode lookup: %w", sql.ErrNoRows)}
	series := &metadataSeriesReader{series: &models.Series{ID: uuidC, Title: "Buffy the Vampire Slayer"}}
	svc.SetEpisodeSubtitleStateReader(episodes)
	svc.SetSeriesMetadataReader(series)

	tmpDir := t.TempDir()
	_, _, err := svc.translateSRT(context.Background(), "job-1", models.SubtitleRunMediaEpisode, uuidB,
		"1\n00:00:01,000 --> 00:00:04,000\nHello world\n", filepath.Join(tmpDir, "s05e16.mkv"), tmpDir)
	require.NoError(t, err)

	assert.Equal(t, 0, series.callCount,
		"an unresolved parent must not be looked up under the episode's own id")
	assert.Equal(t, prompts.SubtitleTranslatorSystemPrompt, mockProvider.lastSystemPrompt)
}

// ─── 9R-8: the Rule 19 duplicates of subtitle/media_store.go helpers ─────────
//
// releaseYear and productionCountryCodes exist ONLY because services must not
// import subtitle. Duplicated logic drifts, so both are pinned directly rather
// than left to a single happy-path integration assertion (CR M3).

func TestReleaseYear(t *testing.T) {
	cases := []struct {
		name string
		date string
		want int
	}{
		{"full ISO date", "1998-07-14", 1998},
		{"year only", "2016", 2016},
		{"empty", "", 0},
		{"too short", "199", 0},
		{"non-numeric placeholder", "TBAX", 0},
		{"mixed junk", "19x8-01-01", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, releaseYear(tc.date))
		})
	}
}

func TestProductionCountryCodes(t *testing.T) {
	assert.Nil(t, productionCountryCodes(nil))
	assert.Nil(t, productionCountryCodes([]models.ProductionCountry{}))

	assert.Equal(t, []string{"US", "DE"}, productionCountryCodes([]models.ProductionCountry{
		{ISO3166_1: "US", Name: "United States of America"},
		{ISO3166_1: "DE", Name: "Germany"},
	}))

	// Blank / whitespace-only codes are dropped, mirroring media_store.go
	// countryCodes — otherwise the prompt renders a dangling ", ".
	assert.Equal(t, []string{"JP"}, productionCountryCodes([]models.ProductionCountry{
		{ISO3166_1: "  ", Name: "Blank"},
		{ISO3166_1: " JP ", Name: "Japan"},
		{ISO3166_1: "", Name: "Empty"},
	}))
}
