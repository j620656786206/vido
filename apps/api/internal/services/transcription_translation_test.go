package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── parseSRTToTranslationBlocks Tests (P0) ─────────────────────────────────

func TestParseSRTToTranslationBlocks_Basic(t *testing.T) {
	input := "1\n00:00:01,000 --> 00:00:04,000\nHello world\n\n2\n00:00:05,000 --> 00:00:08,000\nGoodbye\n"

	blocks, err := parseSRTToTranslationBlocks(input)
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
	blocks, err := parseSRTToTranslationBlocks("")
	require.NoError(t, err)
	assert.Nil(t, blocks)
}

func TestParseSRTToTranslationBlocks_BOM(t *testing.T) {
	input := "\xEF\xBB\xBF1\n00:00:01,000 --> 00:00:04,000\nHello\n"

	blocks, err := parseSRTToTranslationBlocks(input)
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	assert.Equal(t, 1, blocks[0].Index)
	assert.Equal(t, "Hello", blocks[0].Text)
}

func TestParseSRTToTranslationBlocks_WindowsCRLF(t *testing.T) {
	input := "1\r\n00:00:01,000 --> 00:00:04,000\r\nHello\r\n\r\n2\r\n00:00:05,000 --> 00:00:08,000\r\nWorld\r\n"

	blocks, err := parseSRTToTranslationBlocks(input)
	require.NoError(t, err)
	require.Len(t, blocks, 2)
	assert.Equal(t, "Hello", blocks[0].Text)
	assert.Equal(t, "World", blocks[1].Text)
}

func TestParseSRTToTranslationBlocks_MultiLineText(t *testing.T) {
	input := "1\n00:00:01,000 --> 00:00:04,000\nLine one\nLine two\n\n"

	blocks, err := parseSRTToTranslationBlocks(input)
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	assert.Equal(t, "Line one\nLine two", blocks[0].Text)
}

func TestParseSRTToTranslationBlocks_ExtraBlankLines(t *testing.T) {
	input := "\n\n1\n00:00:01,000 --> 00:00:04,000\nHello\n\n\n\n2\n00:00:05,000 --> 00:00:08,000\nWorld\n"

	blocks, err := parseSRTToTranslationBlocks(input)
	require.NoError(t, err)
	require.Len(t, blocks, 2)
}

func TestParseSRTToTranslationBlocks_TimestampPreservation(t *testing.T) {
	// AC #3: timestamps must be preserved exactly
	input := "1\n01:23:45,678 --> 01:23:50,999\nTest\n"

	blocks, err := parseSRTToTranslationBlocks(input)
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

	blocks1, err := parseSRTToTranslationBlocks(original)
	require.NoError(t, err)

	serialized := serializeTranslationBlocksToSRT(blocks1)

	blocks2, err := parseSRTToTranslationBlocks(serialized)
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

	zhPath, err := svc.translateSRT(context.Background(), "job-1", 1, srtContent, filePath, tmpDir)
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

	zhPath, err := svc.translateSRT(context.Background(), "job-1", 1, srtContent, filePath, tmpDir)
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

	_, err := svc.translateSRT(context.Background(), "job-1", 1, "", "test.mkv", t.TempDir())
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

	zhPath, err := svc.translateSRT(context.Background(), "job-1", 1, srtContent.String(), filePath, tmpDir)
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
	zhPath, err := svc.translateSRT(context.Background(), "job-1", 1, srtContent, filePath, tmpDir)
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
}

func (m *translationIntegrationMock) CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	m.callCount++
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

