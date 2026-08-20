package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
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
		{
			name:     "multi-line block",
			response: "[1] 第一行\n第二行\n[2] 世界",
			indices:  []int{1, 2},
			want:     map[int]string{1: "第一行\n第二行", 2: "世界"},
		},
		{
			name:     "multi-line last block",
			response: "[1] 你好\n[2] 第一行\n第二行\n第三行",
			indices:  []int{1, 2},
			want:     map[int]string{1: "你好", 2: "第一行\n第二行\n第三行"},
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

// --- sub-5-5: harvest trailer (AC #2 red line 1) ---

// TestParseTranslationResponse_TrailerRedLines pins the AC #2 red line: the
// continuation rule appends un-prefixed lines to the most recent cue, so
// without trailer stripping the `===TERMS===` sentinel and every term line
// would be stitched into the LAST subtitle cue.
func TestParseTranslationResponse_TrailerRedLines(t *testing.T) {
	t.Run("last cue is byte-identical with and without a trailer", func(t *testing.T) {
		clean := "[1] 你好\n[2] 第一行\n第二行"
		withTrailer := clean + "\n===TERMS===\nDemogorgon=>魔王獸\nVecna=>維克那"

		want := parseTranslationResponse(clean, []int{1, 2})
		got := parseTranslationResponse(withTrailer, []int{1, 2})
		assert.Equal(t, want, got, "the trailer must be cut BEFORE cue parsing — red line 1")
		assert.Equal(t, "第一行\n第二行", got[2])
	})

	t.Run("no trailer — behavior byte-identical to today", func(t *testing.T) {
		got := parseTranslationResponse("[1] 你好\n[2] 世界", []int{1, 2})
		assert.Equal(t, map[int]string{1: "你好", 2: "世界"}, got)
	})

	t.Run("mid-response sentinel truncates — no cue line after it is parsed", func(t *testing.T) {
		got := parseTranslationResponse("[1] 你好\n===TERMS===\nA=>甲\n[2] 世界", []int{1, 2})
		assert.Equal(t, map[int]string{1: "你好"}, got,
			"everything from the sentinel line onward is trailer, even a [N] line")
	})
}

func TestSplitHarvestTrailer(t *testing.T) {
	t.Run("terms parsed, malformed lines skipped fail-soft", func(t *testing.T) {
		body, terms := splitHarvestTrailer(
			"[1] 你好\n===TERMS===\nDemogorgon=>魔王獸\nno separator here\n=>空來源\nEmpty target=>\nVecna=>維克那")
		assert.Equal(t, "[1] 你好", body)
		assert.Equal(t, map[string]string{"Demogorgon": "魔王獸", "Vecna": "維克那"}, terms)
	})

	t.Run("no trailer — body untouched, nil terms", func(t *testing.T) {
		body, terms := splitHarvestTrailer("[1] 你好\n[2] 世界")
		assert.Equal(t, "[1] 你好\n[2] 世界", body)
		assert.Nil(t, terms)
	})

	t.Run("sentinel with no term lines — nil terms", func(t *testing.T) {
		body, terms := splitHarvestTrailer("[1] 你好\n===TERMS===\n")
		assert.Equal(t, "[1] 你好", body)
		assert.Nil(t, terms)
	})

	t.Run("duplicate source — first occurrence wins", func(t *testing.T) {
		_, terms := splitHarvestTrailer("[1] 你好\n===TERMS===\nVecna=>維克那\nVecna=>威克納")
		assert.Equal(t, map[string]string{"Vecna": "維克那"}, terms)
	})

	t.Run("identity mapping is ignored — no rendering decision was made", func(t *testing.T) {
		// CR sub-5-5 M2: the prompt forbids listing terms kept in their
		// original form, but a disobedient model still emits them.
		_, terms := splitHarvestTrailer("[1] 你好\n===TERMS===\nVecna=>Vecna\nDemogorgon=>魔王獸")
		assert.Equal(t, map[string]string{"Demogorgon": "魔王獸"}, terms)
	})
}

func TestTranslationService_TranslateWithGlossaryHarvest_ReturnsTrailerTerms(t *testing.T) {
	mock := &mockTranslationCompleter{
		response: "[1] 魔王獸來了\n===TERMS===\nDemogorgon=>魔王獸",
	}
	svc := NewTranslationService(mock, nil)
	blocks := []TranslationBlock{
		{Index: 1, Start: "00:00:01,000", End: "00:00:04,000", Text: "The Demogorgon is coming"},
	}

	translated, terms, _, err := svc.TranslateWithGlossaryHarvest(context.Background(), blocks, nil, nil)
	require.NoError(t, err)

	require.Len(t, translated, 1)
	assert.Equal(t, "魔王獸來了", translated[0].Text,
		"the trailer must never leak into cue text (red line 1)")
	assert.Equal(t, map[string]string{"Demogorgon": "魔王獸"}, terms)

	// Back-compat delegate: same blocks, terms dropped, signature unchanged.
	viaOld, _, err := svc.TranslateWithGlossary(context.Background(), blocks, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, translated, viaOld)
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

	_, err := svc.Translate(context.Background(), blocks, progressFn)
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

// --- 9R-7: glossary-aware translation ---

func TestTranslationService_TranslateWithGlossary_InjectsGlossary(t *testing.T) {
	mock := &mockTranslationCompleter{response: "[1] 魔王獸來了"}
	svc := NewTranslationService(mock, nil)

	glossary := []GlossaryPair{
		{Source: "Demogorgon", Target: "魔王獸"},
		{Source: "", Target: "skip-me"}, // blank source must be filtered
	}
	out, _, err := svc.TranslateWithGlossary(
		context.Background(),
		[]TranslationBlock{{Index: 1, Text: "The Demogorgon is coming"}},
		glossary, nil,
	)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "魔王獸來了", out[0].Text)

	require.Len(t, mock.calls, 1)
	prompt := mock.calls[0].UserPrompt
	assert.Contains(t, prompt, "Glossary")
	assert.Contains(t, prompt, "Demogorgon → 魔王獸", "the fixed rendering must be injected")
	assert.NotContains(t, prompt, "skip-me", "blank-source entries are filtered")
}

func TestTranslationService_Translate_NoGlossary_PromptUnchanged(t *testing.T) {
	// No-regression: Translate with no glossary must NOT add a Glossary section.
	mock := &mockTranslationCompleter{response: "[1] 你好"}
	svc := NewTranslationService(mock, nil)

	_, err := svc.Translate(context.Background(), []TranslationBlock{{Index: 1, Text: "Hi"}}, nil)
	require.NoError(t, err)
	require.Len(t, mock.calls, 1)
	assert.NotContains(t, mock.calls[0].UserPrompt, "Glossary")
}

func TestTranslationService_TranslateRequest_FieldsAndGlossary(t *testing.T) {
	// Generic entry point (9R-13 metadata path): arbitrary keyed fields + glossary.
	mock := &mockTranslationCompleter{response: "[1] 一名竊賊潛入夢境\n[2] 唐姆·柯布"}
	svc := NewTranslationService(mock, nil)

	req := TranslationRequest{
		Fields: []TranslationField{
			{Key: "plot", Text: "A thief who enters dreams"},
			{Key: "character", Text: "Dom Cobb"},
		},
		Glossary: []GlossaryPair{{Source: "Dom Cobb", Target: "唐姆·柯布"}},
	}
	out, err := svc.TranslateRequest(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, "plot", out[0].Key)
	assert.Equal(t, "一名竊賊潛入夢境", out[0].Text)
	assert.Equal(t, "character", out[1].Key)
	assert.Equal(t, "唐姆·柯布", out[1].Text)

	assert.Contains(t, mock.calls[0].UserPrompt, "Dom Cobb → 唐姆·柯布")
}

func TestTranslationService_TranslateRequest_FailSoftKeepsOriginal(t *testing.T) {
	// A field with no returned translation keeps its original text.
	mock := &mockTranslationCompleter{response: "[1] 已翻譯"}
	svc := NewTranslationService(mock, nil)

	out, err := svc.TranslateRequest(context.Background(), TranslationRequest{
		Fields: []TranslationField{
			{Key: "a", Text: "translated"},
			{Key: "b", Text: "untouched"},
		},
	})
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, "已翻譯", out[0].Text)
	assert.Equal(t, "untouched", out[1].Text, "missing translation → original preserved")
}

// --- sub-1-5a: TranslateChunk (single-chunk, usage-returning) ---

// cachingTranslationMock implements BOTH ai.TextCompleter and
// ai.CachingCompleter — the Claude shape.
type cachingTranslationMock struct {
	result     ai.CompletionResult
	err        error
	reqs       []ai.CompletionRequest
	plainCalls int
}

func (m *cachingTranslationMock) CompleteText(_ context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	m.plainCalls++
	return "", errors.New("CompleteText must not be used when CachingCompleter is available")
}

func (m *cachingTranslationMock) CompleteTextWithUsage(_ context.Context, req ai.CompletionRequest) (ai.CompletionResult, error) {
	m.reqs = append(m.reqs, req)
	if m.err != nil {
		return ai.CompletionResult{}, m.err
	}
	return m.result, nil
}

func chunkSystemBlocks() []ai.SystemBlock {
	return []ai.SystemBlock{
		{Text: prompts.SubtitleTranslatorSystemPrompt, CacheTTL: ai.CacheTTLNone},
		{Text: "## Media context\n", CacheTTL: ai.CacheTTLNone},
	}
}

func TestTranslationService_TranslateChunk_HappyPath(t *testing.T) {
	mock := &cachingTranslationMock{result: ai.CompletionResult{
		Text: "[4] 早安\n[9] 待會見\n[42] 這個索引沒人要",
		Usage: ai.CompletionUsage{
			InputTokens: 1200, OutputTokens: 80,
			CacheCreationInputTokens: 900, CacheReadInputTokens: 300,
		},
	}}
	svc := NewTranslationService(mock, nil)

	got, terms, usage, err := svc.TranslateChunk(context.Background(), chunkSystemBlocks(),
		[]prompts.SubtitleTranslatorBlock{{Index: 1, Text: "早安"}},
		[]prompts.SubtitleTranslatorBlock{{Index: 4, Text: "Good morning."}, {Index: 9, Text: "See you later."}},
	)
	require.NoError(t, err)

	assert.Equal(t, map[int]string{4: "早安", 9: "待會見"}, got,
		"only the requested indexes survive parseTranslationResponse")
	assert.Nil(t, terms, "no trailer in the response — nothing harvested")
	assert.Equal(t, ai.CompletionUsage{
		InputTokens: 1200, OutputTokens: 80,
		CacheCreationInputTokens: 900, CacheReadInputTokens: 300,
	}, usage)

	require.Len(t, mock.reqs, 1)
	req := mock.reqs[0]
	assert.Equal(t, chunkSystemBlocks(), req.System, "system blocks pass through in caller order")
	assert.Equal(t, TranslationMaxTokens, req.MaxTokens)
	assert.Contains(t, req.UserPrompt, "Previous context")
	assert.Contains(t, req.UserPrompt, "[4] Good morning.")
	assert.Contains(t, req.UserPrompt, "[9] See you later.")
	assert.NotContains(t, req.UserPrompt, "-->", "timestamps never leave Go (P2/FR11)")
	assert.Zero(t, mock.plainCalls)
}

func TestTranslationService_TranslateChunk_NonCachingProviderDegrades(t *testing.T) {
	// Gemini shape: ai.TextCompleter only, no usage reporting.
	mock := &mockTranslationCompleter{response: "[1] 早安"}
	svc := NewTranslationService(mock, nil)

	got, terms, usage, err := svc.TranslateChunk(context.Background(), chunkSystemBlocks(), nil,
		[]prompts.SubtitleTranslatorBlock{{Index: 1, Text: "Good morning."}})
	require.NoError(t, err, "a non-caching provider degrades, it does not fail")

	assert.Equal(t, map[int]string{1: "早安"}, got)
	assert.Nil(t, terms)
	assert.Equal(t, ai.CompletionUsage{}, usage, "usage is unavailable, not invented")

	require.Len(t, mock.calls, 1)
	assert.Contains(t, mock.calls[0].SystemPrompt, prompts.SubtitleTranslatorSystemPrompt,
		"system blocks are flattened for the single-string API")
	assert.Contains(t, mock.calls[0].SystemPrompt, "## Media context")
	assert.Equal(t, TranslationMaxTokens, mock.calls[0].MaxTokens)
}

func TestTranslationService_TranslateChunk_PropagatesProviderError(t *testing.T) {
	sentinel := errors.New("upstream exploded")
	svc := NewTranslationService(&cachingTranslationMock{err: sentinel}, nil)

	got, terms, usage, err := svc.TranslateChunk(context.Background(), chunkSystemBlocks(), nil,
		[]prompts.SubtitleTranslatorBlock{{Index: 1, Text: "Good morning."}})

	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel, "the cause is chained, not flattened (Rule 13)")
	assert.Nil(t, got)
	assert.Nil(t, terms)
	assert.Equal(t, ai.CompletionUsage{}, usage)
}

func TestTranslationService_TranslateChunk_EmptyBlocksIsANoOp(t *testing.T) {
	mock := &cachingTranslationMock{}
	svc := NewTranslationService(mock, nil)

	got, terms, usage, err := svc.TranslateChunk(context.Background(), chunkSystemBlocks(), nil, nil)

	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Nil(t, terms)
	assert.Equal(t, ai.CompletionUsage{}, usage)
	assert.Empty(t, mock.reqs, "no cues means no request")
}

// ─── bugfix-j: TranslationOutcome — partial results must not vanish ─────────

// bugfix-j AC #1/AC #4(a): a failed batch keeps English wholesale (AC #5 of
// 9-2b) — the outcome must count those cues instead of dropping a bool into a
// slog.Warn, because the verdict layer demotes found → untranslated on it.
func TestTranslationService_Harvest_PartialOutcome_BatchFailure(t *testing.T) {
	// bugfix-j CR L2: the fixture below (10-line first response, 15 blocks,
	// expected kept=5) assumes the batch size — pin it so a constant change
	// fails HERE with a clear message, not as an opaque count mismatch.
	require.Equal(t, 10, prompts.SubtitleTranslatorBatchSize,
		"fixture assumes batch size 10 — rebuild the response/counts if this changes")
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
	translated, _, outcome, err := svc.TranslateWithGlossaryHarvest(context.Background(), blocks, nil, nil)
	require.NoError(t, err, "partial failure stays non-fatal (AC #5 of 9-2b)")
	require.Len(t, translated, 15)

	assert.True(t, outcome.Partial(), "a failed batch is a partial outcome")
	assert.Equal(t, 5, outcome.EnglishKeptBlocks, "blocks 11-15 kept English")
	assert.Equal(t, 15, outcome.TotalBlocks)
	assert.Equal(t, "第一行", translated[0].Text)
	assert.Equal(t, "English line 11", translated[10].Text)
}

// bugfix-j AC #1/AC #4(b): a response missing a single block is ALSO partial —
// the silent per-cue keep-English path (:252-258 pre-fix) must count too.
func TestTranslationService_Harvest_PartialOutcome_PerCueMiss(t *testing.T) {
	// Two blocks in, response covers only block 1.
	mock := &translationIntegrationMock{response: "[1] 你好"}
	blocks := []TranslationBlock{
		{Index: 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "Hello"},
		{Index: 2, Start: "00:00:03,000", End: "00:00:04,000", Text: "Still English"},
	}

	svc := NewTranslationService(mock, nil)
	translated, _, outcome, err := svc.TranslateWithGlossaryHarvest(context.Background(), blocks, nil, nil)
	require.NoError(t, err)
	require.Len(t, translated, 2)

	assert.True(t, outcome.Partial(), "a per-cue miss is a partial outcome")
	assert.Equal(t, 1, outcome.EnglishKeptBlocks)
	assert.Equal(t, 2, outcome.TotalBlocks)
	assert.Equal(t, "你好", translated[0].Text)
	assert.Equal(t, "Still English", translated[1].Text, "the missed cue keeps English (unchanged behavior)")
}

// bugfix-j AC #4(c) regression pin: a fully-translated run is NOT partial —
// the found verdict path stays byte-identical for full successes.
func TestTranslationService_Harvest_FullSuccessNotPartial(t *testing.T) {
	mock := &translationIntegrationMock{response: "[1] 你好\n[2] 世界"}
	blocks := []TranslationBlock{
		{Index: 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "Hello"},
		{Index: 2, Start: "00:00:03,000", End: "00:00:04,000", Text: "World"},
	}

	svc := NewTranslationService(mock, nil)
	translated, _, outcome, err := svc.TranslateWithGlossaryHarvest(context.Background(), blocks, nil, nil)
	require.NoError(t, err)
	require.Len(t, translated, 2)

	assert.False(t, outcome.Partial())
	assert.Equal(t, 0, outcome.EnglishKeptBlocks)
	assert.Equal(t, 2, outcome.TotalBlocks)
}

// bugfix-j CR M3: a bare "[N]" line (model emitted the index with no
// translation) must (a) NOT pollute the PREVIOUS cue with the literal "[N]"
// text, (b) count as a per-cue miss (English kept), and (c) still accept a
// continuation line as that cue's first text.
func TestParseTranslationResponse_BareIndexLine(t *testing.T) {
	// (a)+(b): bare [2] between two translated cues.
	res, _ := parseTranslationResponseWithTerms("[1] 你好\n[2]\n[3] 世界", []int{1, 2, 3})
	assert.Equal(t, "你好", res[1], "previous cue must not absorb the literal \"[2]\" line")
	_, ok := res[2]
	assert.False(t, ok, "an empty translation is a MISSING translation")
	assert.Equal(t, "世界", res[3])

	// trailing-whitespace variant of the same shape.
	res2, _ := parseTranslationResponseWithTerms("[1] 你好\n[2]   ", []int{1, 2})
	assert.Equal(t, "你好", res2[1])
	_, ok2 := res2[2]
	assert.False(t, ok2)

	// (c): continuation after a bare index belongs to THAT cue.
	res3, _ := parseTranslationResponseWithTerms("[1]\n第一行接續", []int{1})
	assert.Equal(t, "第一行接續", res3[1], "continuation is the cue's first text, no leading newline")
}

// bugfix-j CR M3 (outcome side): the bare-index miss flows into the partial
// outcome — the cue keeps English and EnglishKeptBlocks counts it.
func TestTranslationService_Harvest_BareIndexCountsAsKept(t *testing.T) {
	mock := &translationIntegrationMock{response: "[1] 你好\n[2]"}
	blocks := []TranslationBlock{
		{Index: 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "Hello"},
		{Index: 2, Start: "00:00:03,000", End: "00:00:04,000", Text: "Still English"},
	}
	svc := NewTranslationService(mock, nil)
	translated, _, outcome, err := svc.TranslateWithGlossaryHarvest(context.Background(), blocks, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "你好", translated[0].Text, "cue 1 unpolluted")
	assert.Equal(t, "Still English", translated[1].Text)
	assert.Equal(t, 1, outcome.EnglishKeptBlocks)
	assert.True(t, outcome.Partial())
}

// bugfix-j H3 ruling (option B「門檻制」): demotion fires only on MATERIAL
// English residue — ≥1 batch's worth of cues (an English RUN) or ≥5% of the
// item's OWN cue count (per-movie/per-episode denominator, per the ruling).
// Disclosure (Partial) stays unconditional either way.
func TestTranslationOutcome_DemotesVerdict(t *testing.T) {
	// The absolute bar IS the batch size — a failed batch always demotes. If
	// the batch size changes, the boundary cases below must be re-derived.
	require.Equal(t, 10, prompts.SubtitleTranslatorBatchSize,
		"demoteAbsoluteKeptBlocks tracks the batch size — update the cases below")

	cases := []struct {
		name    string
		kept    int
		total   int
		demotes bool
	}{
		{"full success never demotes", 0, 100, false},
		{"exactly 5% demotes (1/20)", 1, 20, true},
		{"just under 5% stays found (1/21)", 1, 21, false},
		{"scattered misses under both bars (9/300 = 3%)", 9, 300, false},
		{"a whole batch demotes regardless of percent (10/300 ≈ 3.3%)", 10, 300, true},
		{"above absolute bar (25/1000 = 2.5%)", 25, 1000, true},
		{"small file: 1/2 = 50% demotes", 1, 2, true},
		{"tiny file below batch, at percent bar (1/10 = 10%)", 1, 10, true},
		{"everything kept demotes", 40, 40, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := TranslationOutcome{EnglishKeptBlocks: tc.kept, TotalBlocks: tc.total}
			assert.Equal(t, tc.demotes, o.DemotesVerdict())
			assert.Equal(t, tc.kept > 0, o.Partial(), "disclosure is unconditional — independent of the threshold")
		})
	}
}
