package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGeminiProvider(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		p := NewGeminiProvider("test-api-key")

		assert.Equal(t, "test-api-key", p.apiKey)
		assert.Equal(t, DefaultGeminiBaseURL, p.baseURL)
		assert.Equal(t, DefaultGeminiModel, p.model)
		assert.Equal(t, time.Duration(DefaultTimeoutSeconds)*time.Second, p.timeout)
		assert.NotNil(t, p.httpClient)
	})

	t.Run("custom configuration", func(t *testing.T) {
		customClient := &http.Client{Timeout: 5 * time.Second}
		p := NewGeminiProvider("test-api-key",
			WithGeminiBaseURL("https://custom.url"),
			WithGeminiModel("custom-model"),
			WithGeminiHTTPClient(customClient),
			WithGeminiTimeout(30*time.Second),
		)

		assert.Equal(t, "https://custom.url", p.baseURL)
		assert.Equal(t, "custom-model", p.model)
		assert.Equal(t, customClient, p.httpClient)
		assert.Equal(t, 30*time.Second, p.timeout)
	})
}

func TestGeminiProvider_Name(t *testing.T) {
	p := NewGeminiProvider("test-key")
	assert.Equal(t, ProviderGemini, p.Name())
}

func TestGeminiProvider_Parse_Success(t *testing.T) {
	// Create mock response
	mockResponse := geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Parts: []geminiPart{
						{Text: `{"title": "Attack on Titan", "year": 2013, "season": 1, "episode": 1, "media_type": "tv", "quality": "1080p", "fansub_group": "SubsPlease", "confidence": 0.95}`},
					},
				},
			},
		},
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "generateContent")
		assert.Contains(t, r.URL.RawQuery, "key=test-api-key")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key",
		WithGeminiBaseURL(server.URL),
	)

	req := &ParseRequest{
		Filename: "[SubsPlease] Shingeki no Kyojin - 01 [1080p].mkv",
	}

	result, err := p.Parse(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "Attack on Titan", result.Title)
	assert.Equal(t, 2013, result.Year)
	assert.Equal(t, 1, result.Season)
	assert.Equal(t, 1, result.Episode)
	assert.Equal(t, "tv", result.MediaType)
	assert.Equal(t, "1080p", result.Quality)
	assert.Equal(t, "SubsPlease", result.FansubGroup)
	assert.Equal(t, 0.95, result.Confidence)
}

func TestGeminiProvider_Parse_MovieSuccess(t *testing.T) {
	mockResponse := geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Parts: []geminiPart{
						{Text: `{"title": "Your Name", "year": 2016, "media_type": "movie", "quality": "1080p", "confidence": 0.9}`},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	req := &ParseRequest{Filename: "Kimi.no.Na.wa.2016.1080p.BluRay.mkv"}

	result, err := p.Parse(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "Your Name", result.Title)
	assert.Equal(t, 2016, result.Year)
	assert.Equal(t, "movie", result.MediaType)
	assert.True(t, result.IsMovie())
}

func TestGeminiProvider_Parse_ValidationError(t *testing.T) {
	p := NewGeminiProvider("test-api-key")
	req := &ParseRequest{Filename: ""}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "filename is required")
}

func TestGeminiProvider_Parse_Timeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key",
		WithGeminiBaseURL(server.URL),
		WithGeminiTimeout(50*time.Millisecond),
	)

	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAITimeout)
}

func TestGeminiProvider_Parse_QuotaExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIQuotaExceeded)
}

func TestGeminiProvider_Parse_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIProviderError)
}

func TestGeminiProvider_Parse_InvalidJSONResponse(t *testing.T) {
	mockResponse := geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Parts: []geminiPart{
						{Text: "not valid json"},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIInvalidResponse)
}

func TestGeminiProvider_Parse_MissingTitle(t *testing.T) {
	mockResponse := geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Parts: []geminiPart{
						{Text: `{"media_type": "movie", "confidence": 0.8}`},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIInvalidResponse)
}

func TestGeminiProvider_Parse_EmptyResponse(t *testing.T) {
	mockResponse := geminiResponse{
		Candidates: []geminiCandidate{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIInvalidResponse)
}

func TestGeminiResponse_GetText(t *testing.T) {
	tests := []struct {
		name     string
		response geminiResponse
		want     string
	}{
		{
			name: "normal response",
			response: geminiResponse{
				Candidates: []geminiCandidate{
					{
						Content: geminiContent{
							Parts: []geminiPart{{Text: "hello"}},
						},
					},
				},
			},
			want: "hello",
		},
		{
			name:     "empty candidates",
			response: geminiResponse{Candidates: []geminiCandidate{}},
			want:     "",
		},
		{
			name: "empty parts",
			response: geminiResponse{
				Candidates: []geminiCandidate{
					{Content: geminiContent{Parts: []geminiPart{}}},
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.response.GetText())
		})
	}
}

func TestParseJSONResponse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *ParseResponse
		wantErr bool
	}{
		{
			name:  "valid movie response",
			input: `{"title": "Inception", "year": 2010, "media_type": "movie", "confidence": 0.95}`,
			want: &ParseResponse{
				Title:      "Inception",
				Year:       2010,
				MediaType:  "movie",
				Confidence: 0.95,
			},
			wantErr: false,
		},
		{
			name:  "valid tv response",
			input: `{"title": "Breaking Bad", "season": 1, "episode": 1, "media_type": "tv", "confidence": 0.9}`,
			want: &ParseResponse{
				Title:      "Breaking Bad",
				Season:     1,
				Episode:    1,
				MediaType:  "tv",
				Confidence: 0.9,
			},
			wantErr: false,
		},
		{
			name:    "missing title",
			input:   `{"media_type": "movie", "confidence": 0.8}`,
			wantErr: true,
		},
		{
			name:    "missing media_type",
			input:   `{"title": "Test", "confidence": 0.8}`,
			wantErr: true,
		},
		{
			name:    "invalid media_type",
			input:   `{"title": "Test", "media_type": "anime", "confidence": 0.8}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			input:   `not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseJSONResponse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.Title, result.Title)
				assert.Equal(t, tt.want.MediaType, result.MediaType)
			}
		})
	}
}

// --- sub-5-1 AC #2: Gemini rides the same governed+retry+metering stack as Claude ---

// geminiOKBody is a minimal valid generateContent response WITH the API's own
// token accounting, as the real endpoint returns it.
func geminiOKBody(promptTokens, candidateTokens int64) []byte {
	body, _ := json.Marshal(map[string]interface{}{
		"candidates": []map[string]interface{}{
			{"content": map[string]interface{}{"parts": []map[string]interface{}{
				{"text": `{"title": "Test Movie", "media_type": "movie", "confidence": 0.9}`},
			}}},
		},
		"usageMetadata": map[string]interface{}{
			"promptTokenCount":     promptTokens,
			"candidatesTokenCount": candidateTokens,
			"totalTokenCount":      promptTokens + candidateTokens,
		},
	})
	return body
}

func TestGeminiProvider_Parse_MetersUsageIntoCtxBudget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(geminiOKBody(1_000_000, 2_000_000))
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	b := NewBudget(0)
	_, err := p.Parse(WithBudget(context.Background(), b), &ParseRequest{Filename: "test.mkv"})
	require.NoError(t, err)

	snap := b.Snapshot()
	assert.Equal(t, int64(1_000_000), snap.InputTokens, "promptTokenCount → input tokens")
	assert.Equal(t, int64(2_000_000), snap.OutputTokens, "candidatesTokenCount → output tokens")
	assert.Equal(t, 1, snap.LLMCalls)
	// Priced at the gemini-2.0-flash row ($0.10 in / $0.40 out per 1M) — NOT
	// the Haiku fallback, which would fabricate $1 + $10 here.
	assert.InDelta(t, 0.10+0.80, snap.SpentUSD, 1e-9)
}

func TestGeminiProvider_Parse_NoCtxBudgetIsNoOp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(geminiOKBody(100, 50))
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	_, err := p.Parse(context.Background(), &ParseRequest{Filename: "test.mkv"})
	require.NoError(t, err, "the unmetered parse path (AC #4 ruling) must keep working without a Budget")
}

func TestGeminiProvider_Parse_BudgetCutoffShortCircuits(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		w.Write(geminiOKBody(100, 50))
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	b := NewBudget(0.001)
	b.RecordLLM("gemini-2.0-flash", 1_000_000, 0) // $0.10 → over the ceiling
	_, err := p.Parse(WithBudget(context.Background(), b), &ParseRequest{Filename: "test.mkv"})
	require.ErrorIs(t, err, ErrBudgetExceeded)
	assert.Equal(t, int32(0), atomic.LoadInt32(&hits), "an exhausted budget must stop the call BEFORE the wire")
}

func TestGeminiProvider_Parse_RetriesTransient5xxThenSucceeds(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(geminiOKBody(100, 50))
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	result, err := p.Parse(context.Background(), &ParseRequest{Filename: "test.mkv"})
	require.NoError(t, err)
	assert.Equal(t, "Test Movie", result.Title)
	assert.Equal(t, int32(2), atomic.LoadInt32(&hits), "one transient 503 → exactly one retry")
}

func TestGeminiProvider_Parse_MalformedBodyIsPermanent(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{not json"))
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	_, err := p.Parse(context.Background(), &ParseRequest{Filename: "test.mkv"})
	require.ErrorIs(t, err, ErrAIInvalidResponse)
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits),
		"a garbage 200 body must NOT be retried — one broken response must not become three real requests")
}

// CR L2: observable governance, not field identity — with a 1-slot Governor,
// two concurrent Parses must serialize on the wire, which also proves the
// slot is RELEASED (a leaked slot would deadlock the second call).
func TestGeminiProvider_GovernorSerializesConcurrentCalls(t *testing.T) {
	var inFlight, maxInFlight int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := atomic.AddInt32(&inFlight, 1)
		for {
			prev := atomic.LoadInt32(&maxInFlight)
			if cur <= prev || atomic.CompareAndSwapInt32(&maxInFlight, prev, cur) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		atomic.AddInt32(&inFlight, -1)
		w.Header().Set("Content-Type", "application/json")
		w.Write(geminiOKBody(100, 50))
	}))
	defer server.Close()

	g := NewGovernor(1, 0, 0)
	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL), WithGeminiGovernor(g))

	done := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := p.Parse(context.Background(), &ParseRequest{Filename: "test.mkv"})
			done <- err
		}()
	}
	for i := 0; i < 2; i++ {
		require.NoError(t, <-done)
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&maxInFlight),
		"a 1-slot governor must never allow 2 in-flight Gemini calls")
}

// CR M3 pin: 429 is transient (claude.go parity), so a quota-exhausted key
// costs exactly retryMaxAttempts wire hits per logical call — never more,
// and the sentinel still surfaces for callers.
func TestGeminiProvider_Parse_QuotaExceededRetriesExactlyMaxAttempts(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	_, err := p.Parse(context.Background(), &ParseRequest{Filename: "test.mkv"})
	require.ErrorIs(t, err, ErrAIQuotaExceeded)
	assert.Equal(t, int32(retryMaxAttempts), atomic.LoadInt32(&hits),
		"the 1→3 request amplification is a deliberate claude-parity tradeoff and must stay bounded at retryMaxAttempts")
}

// CR M4: a 200 without usageMetadata meters $0 — that must be LOUD, because a
// silent zero disables the budget ceiling while real money is spent.
func TestGeminiProvider_Parse_MissingUsageMetadataWarnsAndMetersZero(t *testing.T) {
	body, _ := json.Marshal(map[string]interface{}{
		"candidates": []map[string]interface{}{
			{"content": map[string]interface{}{"parts": []map[string]interface{}{
				{"text": `{"title": "Test Movie", "media_type": "movie", "confidence": 0.9}`},
			}}},
		},
		// no usageMetadata block at all
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer server.Close()

	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	b := NewBudget(0)
	_, err := p.Parse(WithBudget(context.Background(), b), &ParseRequest{Filename: "test.mkv"})
	require.NoError(t, err)

	snap := b.Snapshot()
	assert.Zero(t, snap.SpentUSD)
	assert.Equal(t, 1, snap.LLMCalls, "the call itself is still recorded so LLMCalls stays honest")
	assert.Contains(t, logs.String(), "no usageMetadata",
		"the zero-usage condition must be observable in logs, not silent")
}

// CR L2: pin the EXACT published rates (source: ai.google.dev/gemini-api/docs/
// pricing, verified 2026-08-12) — a NotEqual-to-fallback assertion would let a
// typo'd rate sail through.
func TestGeminiPricing_RowsMatchThePublishedRates(t *testing.T) {
	require.Contains(t, defaultLLMPricing, DefaultGeminiModel,
		"DefaultGeminiModel must carry its own pricing row — falling into the Haiku-tier fallback fabricates numbers")
	assert.Equal(t, ModelPricing{InputPer1M: 0.10, OutputPer1M: 0.40}, defaultLLMPricing["gemini-2.0-flash"])
	assert.Equal(t, ModelPricing{InputPer1M: 0.30, OutputPer1M: 2.50}, defaultLLMPricing["gemini-2.5-flash"])
	assert.Equal(t, ModelPricing{InputPer1M: 0.10, OutputPer1M: 0.40}, defaultLLMPricing["gemini-2.5-flash-lite"])
}
