package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTextCompleter implements ai.TextCompleter for testing.
type mockTranslationCompleter struct {
	response string
	err      error
	calls    []mockTranslationCall
}

type mockTranslationCall struct {
	SystemPrompt string
	UserPrompt   string
	MaxTokens    int
}

func (m *mockTranslationCompleter) CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	m.calls = append(m.calls, mockTranslationCall{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		MaxTokens:    maxTokens,
	})
	return m.response, m.err
}

func TestNewTranslationService_NilProvider(t *testing.T) {
	svc := NewTranslationService(nil, nil)
	assert.Nil(t, svc)
}

func TestTranslationService_IsConfigured(t *testing.T) {
	mock := &mockTranslationCompleter{}
	svc := NewTranslationService(mock, nil)
	assert.True(t, svc.IsConfigured())

	var nilSvc *TranslationService
	assert.False(t, nilSvc.IsConfigured())
}

func TestTranslationService_Translate_BasicBatch(t *testing.T) {
	mock := &mockTranslationCompleter{
		response: "[1] 你好，你好嗎？\n[2] 我很好，謝謝。",
	}

	svc := NewTranslationService(mock, nil)
	require.NotNil(t, svc)

	blocks := []TranslationBlock{
		{Index: 1, Start: "00:00:01,000", End: "00:00:04,000", Text: "Hello, how are you?"},
		{Index: 2, Start: "00:00:05,000", End: "00:00:08,000", Text: "I'm doing fine, thanks."},
	}

	result, err := svc.Translate(context.Background(), blocks, nil)
	require.NoError(t, err)
	require.Len(t, result, 2)

	// Timestamps must be preserved (AC #3)
	assert.Equal(t, "00:00:01,000", result[0].Start)
	assert.Equal(t, "00:00:04,000", result[0].End)
	assert.Equal(t, "00:00:05,000", result[1].Start)
	assert.Equal(t, "00:00:08,000", result[1].End)

	// Text should be translated
	assert.Equal(t, "你好，你好嗎？", result[0].Text)
	assert.Equal(t, "我很好，謝謝。", result[1].Text)
}

func TestTranslationService_Translate_MultipleBatches(t *testing.T) {
	// Generate 15 blocks (should be 2 batches: 10 + 5)
	var blocks []TranslationBlock
	for i := 1; i <= 15; i++ {
		blocks = append(blocks, TranslationBlock{
			Index: i,
			Start: fmt.Sprintf("00:00:%02d,000", i),
			End:   fmt.Sprintf("00:00:%02d,000", i+1),
			Text:  fmt.Sprintf("Line %d", i),
		})
	}

	customMock := &translationBatchMock{responses: make(map[int]string)}
	for i := 1; i <= 10; i++ {
		customMock.responses[0] += fmt.Sprintf("[%d] 第%d行\n", i, i)
	}
	for i := 11; i <= 15; i++ {
		customMock.responses[1] += fmt.Sprintf("[%d] 第%d行\n", i, i)
	}

	svc := NewTranslationService(customMock, nil)
	result, err := svc.Translate(context.Background(), blocks, nil)
	require.NoError(t, err)
	require.Len(t, result, 15)

	// Verify 2 API calls were made
	assert.Equal(t, 2, customMock.callCount)
}

type translationBatchMock struct {
	responses map[int]string
	callCount int
}

func (m *translationBatchMock) CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	idx := m.callCount
	m.callCount++
	if resp, ok := m.responses[idx]; ok {
		return resp, nil
	}
	return "", errors.New("unexpected call")
}

func TestTranslationService_Translate_ContextPassing(t *testing.T) {
	// Verify that context blocks are passed to subsequent batches (AC #2)
	trackingMock := &translationTrackingMock{responses: make(map[int]string)}

	// 12 blocks → batch 1 (1-10), batch 2 (11-12)
	var blocks []TranslationBlock
	for i := 1; i <= 12; i++ {
		blocks = append(blocks, TranslationBlock{
			Index: i,
			Start: fmt.Sprintf("00:00:%02d,000", i),
			End:   fmt.Sprintf("00:00:%02d,000", i+1),
			Text:  fmt.Sprintf("Line %d", i),
		})
	}

	for i := 1; i <= 10; i++ {
		trackingMock.responses[0] += fmt.Sprintf("[%d] 第%d行\n", i, i)
	}
	for i := 11; i <= 12; i++ {
		trackingMock.responses[1] += fmt.Sprintf("[%d] 第%d行\n", i, i)
	}

	svc := NewTranslationService(trackingMock, nil)
	_, err := svc.Translate(context.Background(), blocks, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, trackingMock.callCount)

	// Second call should contain "Previous context" with last 5 translated blocks
	secondPrompt := trackingMock.prompts[1]
	assert.Contains(t, secondPrompt, "Previous context")
}

type translationTrackingMock struct {
	responses map[int]string
	callCount int
	prompts   []string
}

func (m *translationTrackingMock) CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	idx := m.callCount
	m.callCount++
	m.prompts = append(m.prompts, userPrompt)
	if resp, ok := m.responses[idx]; ok {
		return resp, nil
	}
	return "", errors.New("unexpected call")
}

func TestTranslationService_Translate_PartialFailure(t *testing.T) {
	// AC #5: on error, keep English text for failed blocks
	failingMock := &translationFailOnSecondMock{
		firstResponse: "[1] 第一行\n[2] 第二行\n[3] 第三行\n[4] 第四行\n[5] 第五行\n[6] 第六行\n[7] 第七行\n[8] 第八行\n[9] 第九行\n[10] 第十行",
	}

	var blocks []TranslationBlock
	for i := 1; i <= 15; i++ {
		blocks = append(blocks, TranslationBlock{
			Index: i,
			Start: fmt.Sprintf("00:00:%02d,000", i),
			End:   fmt.Sprintf("00:00:%02d,000", i+1),
			Text:  fmt.Sprintf("English line %d", i),
		})
	}

	svc := NewTranslationService(failingMock, nil)
	result, err := svc.Translate(context.Background(), blocks, nil)

	// Should return partial result with warning, not hard error
	require.NoError(t, err)
	require.Len(t, result, 15)

	// First 10 blocks should be translated
	assert.Equal(t, "第一行", result[0].Text)

	// Blocks 11-15 should retain English (fallback)
	assert.Equal(t, "English line 11", result[10].Text)
	assert.Equal(t, "English line 15", result[14].Text)
}

type translationFailOnSecondMock struct {
	firstResponse string
	callCount     int
}

func (m *translationFailOnSecondMock) CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	m.callCount++
	if m.callCount == 1 {
		return m.firstResponse, nil
	}
	return "", errors.New("API error: rate limit exceeded")
}

func TestTranslationService_Translate_EmptyBlocks(t *testing.T) {
	mock := &mockTranslationCompleter{}
	svc := NewTranslationService(mock, nil)

	result, err := svc.Translate(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Empty(t, result)
	assert.Empty(t, mock.calls)
}

func TestTranslationService_ParseTranslationResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		indices  []int
		want     map[int]string
	}{
		{
			name:     "basic",
			response: "[1] 你好\n[2] 世界",
			indices:  []int{1, 2},
			want:     map[int]string{1: "你好", 2: "世界"},
		},
		{
			name:     "with extra whitespace",
			response: "[1]  你好 \n[2]  世界 ",
			indices:  []int{1, 2},
			want:     map[int]string{1: "你好", 2: "世界"},
		},
		{
			name:     "missing index",
			response: "[1] 你好",
			indices:  []int{1, 2},
			want:     map[int]string{1: "你好"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTranslationResponse(tt.response, tt.indices)
			for idx, text := range tt.want {
				assert.Equal(t, text, got[idx], "index %d", idx)
			}
		})
	}
}

func TestTranslationService_Translate_ProgressCallback(t *testing.T) {
	mock := &mockTranslationCompleter{
		response: "[1] 翻譯",
	}

	var progressUpdates []float64
	progressFn := func(pct float64) {
		progressUpdates = append(progressUpdates, pct)
	}

	svc := NewTranslationService(mock, nil)
	blocks := []TranslationBlock{
		{Index: 1, Start: "00:00:01,000", End: "00:00:04,000", Text: "Hello"},
	}

	_, err := svc.TranslateWithProgress(context.Background(), blocks, progressFn)
	require.NoError(t, err)

	// Should have at least one progress update
	assert.NotEmpty(t, progressUpdates)
	// Last update should be 100%
	if len(progressUpdates) > 0 {
		last := progressUpdates[len(progressUpdates)-1]
		assert.True(t, last >= 99.0, "final progress should be ~100%%, got %f", last)
	}
}

func TestTranslationService_Translate_CancelledContext(t *testing.T) {
	mock := &mockTranslationCompleter{
		err: context.Canceled,
	}

	svc := NewTranslationService(mock, nil)
	blocks := []TranslationBlock{
		{Index: 1, Start: "00:00:01,000", End: "00:00:04,000", Text: "Hello"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := svc.Translate(ctx, blocks, nil)

	// Should propagate context cancellation
	assert.True(t, errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "cancel"),
		"should return context cancelled error, got: %v", err)
}
