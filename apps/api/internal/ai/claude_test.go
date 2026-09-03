package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- test-only wire fixtures (story sub-1-1) ---
//
// These mirror the Messages API response shape purely so the httptest servers
// below can build canned bodies. They used to be production types in claude.go;
// the SDK now owns the wire protocol, so they survive here as FIXTURE BUILDERS
// only. Nothing in production code refers to them.

type claudeResponse struct {
	Content    []claudeContentBlock `json:"content"`
	StopReason string               `json:"stop_reason"`
	Usage      claudeUsage          `json:"usage"`
}

type claudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type claudeUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

func TestNewClaudeProvider(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		p := NewClaudeProvider("test-api-key")

		assert.Equal(t, "test-api-key", p.apiKey)
		assert.Equal(t, DefaultClaudeBaseURL, p.baseURL)
		assert.Equal(t, DefaultClaudeModel, p.model)
		assert.Equal(t, time.Duration(DefaultTimeoutSeconds)*time.Second, p.timeout)
		assert.NotNil(t, p.httpClient)
	})

	t.Run("custom configuration", func(t *testing.T) {
		customClient := &http.Client{Timeout: 5 * time.Second}
		p := NewClaudeProvider("test-api-key",
			WithClaudeBaseURL("https://custom.url"),
			WithClaudeModel("custom-model"),
			WithClaudeHTTPClient(customClient),
			WithClaudeTimeout(30*time.Second),
		)

		assert.Equal(t, "https://custom.url", p.baseURL)
		assert.Equal(t, "custom-model", p.model)
		assert.Equal(t, customClient, p.httpClient)
		assert.Equal(t, 30*time.Second, p.timeout)
	})
}

func TestClaudeProvider_Name(t *testing.T) {
	p := NewClaudeProvider("test-key")
	assert.Equal(t, ProviderClaude, p.Name())
}

func TestClaudeProvider_Parse_Success(t *testing.T) {
	mockResponse := claudeResponse{
		Content: []claudeContentBlock{
			{
				Type: "text",
				Text: `{"title": "Attack on Titan", "year": 2013, "season": 1, "episode": 1, "media_type": "tv", "quality": "1080p", "fansub_group": "SubsPlease", "confidence": 0.95}`,
			},
		},
		StopReason: "end_turn",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))
		assert.Equal(t, ClaudeAPIVersion, r.Header.Get("anthropic-version"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Contains(t, r.URL.Path, "/messages")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-api-key", WithClaudeBaseURL(server.URL))
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

func TestClaudeProvider_Parse_MovieSuccess(t *testing.T) {
	mockResponse := claudeResponse{
		Content: []claudeContentBlock{
			{
				Type: "text",
				Text: `{"title": "Your Name", "year": 2016, "media_type": "movie", "quality": "1080p", "confidence": 0.9}`,
			},
		},
		StopReason: "end_turn",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-api-key", WithClaudeBaseURL(server.URL))
	req := &ParseRequest{Filename: "Kimi.no.Na.wa.2016.1080p.BluRay.mkv"}

	result, err := p.Parse(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "Your Name", result.Title)
	assert.Equal(t, 2016, result.Year)
	assert.Equal(t, "movie", result.MediaType)
	assert.True(t, result.IsMovie())
}

func TestClaudeProvider_Parse_ValidationError(t *testing.T) {
	p := NewClaudeProvider("test-api-key")
	req := &ParseRequest{Filename: ""}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "filename is required")
}

func TestClaudeProvider_Parse_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-api-key",
		WithClaudeBaseURL(server.URL),
		WithClaudeTimeout(50*time.Millisecond),
	)

	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAITimeout)
}

func TestClaudeProvider_Parse_QuotaExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-api-key", WithClaudeBaseURL(server.URL))
	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIQuotaExceeded)
}

func TestClaudeProvider_Parse_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-api-key", WithClaudeBaseURL(server.URL))
	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIProviderError)
}

func TestClaudeProvider_Parse_InvalidJSONResponse(t *testing.T) {
	mockResponse := claudeResponse{
		Content: []claudeContentBlock{
			{Type: "text", Text: "not valid json"},
		},
		StopReason: "end_turn",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-api-key", WithClaudeBaseURL(server.URL))
	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIInvalidResponse)
}

func TestClaudeProvider_Parse_EmptyResponse(t *testing.T) {
	mockResponse := claudeResponse{
		Content:    []claudeContentBlock{},
		StopReason: "end_turn",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-api-key", WithClaudeBaseURL(server.URL))
	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIInvalidResponse)
}

// --- CompleteText tests (Story 9-1) ---

func TestClaudeProvider_CompleteText_Success(t *testing.T) {
	var receivedReq map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))
		assert.Equal(t, ClaudeAPIVersion, r.Header.Get("anthropic-version"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		json.NewDecoder(r.Body).Decode(&receivedReq)

		resp := claudeResponse{
			Content:    []claudeContentBlock{{Type: "text", Text: "這個軟體很好用"}},
			StopReason: "end_turn",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-api-key", WithClaudeBaseURL(server.URL))
	result, err := p.CompleteText(context.Background(), "system prompt", "user prompt", 2048)

	require.NoError(t, err)
	assert.Equal(t, "這個軟體很好用", result)

	// Verify system prompt is in the request. It now serializes as an ARRAY of
	// content blocks rather than a bare string — that shape is what makes
	// cache_control possible at all (story sub-1-1 AC #5); the hand-rolled
	// client's plain-string `system` structurally could not express it.
	systemBlocks, ok := receivedReq["system"].([]interface{})
	require.True(t, ok, "system must serialize as an array of content blocks")
	require.Len(t, systemBlocks, 1)
	assert.Equal(t, "system prompt", systemBlocks[0].(map[string]interface{})["text"])
	// Verify max_tokens
	assert.Equal(t, float64(2048), receivedReq["max_tokens"])
	// Verify messages
	messages := receivedReq["messages"].([]interface{})
	assert.Len(t, messages, 1)
	msg := messages[0].(map[string]interface{})
	assert.Equal(t, "user", msg["role"])
	// content is the Messages API's canonical content-block array (the SDK always
	// emits the structured form; the hand-rolled client sent a bare string).
	content, ok := msg["content"].([]interface{})
	require.True(t, ok, "message content must serialize as an array of content blocks")
	require.Len(t, content, 1)
	block := content[0].(map[string]interface{})
	assert.Equal(t, "text", block["type"])
	assert.Equal(t, "user prompt", block["text"])
}

// TestClaudeProvider_CompleteText_SystemFieldSerialization asserts the WIRE BODY
// the SDK actually sends (story sub-1-1 AC #2). It previously marshalled the
// hand-rolled claudeRequest struct; that struct no longer exists in production,
// so the assertion moved to where it always belonged — the real request body,
// captured by an httptest server.
func TestClaudeProvider_CompleteText_SystemFieldSerialization(t *testing.T) {
	newCapturingProvider := func(t *testing.T, captured *map[string]interface{}) *ClaudeProvider {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var body map[string]interface{}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			*captured = body
			resp := claudeResponse{
				Content:    []claudeContentBlock{{Type: "text", Text: "ok"}},
				StopReason: "end_turn",
			}
			require.NoError(t, json.NewEncoder(w).Encode(resp))
		}))
		t.Cleanup(server.Close)
		return NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))
	}

	t.Run("system field included when non-empty", func(t *testing.T) {
		var body map[string]interface{}
		p := newCapturingProvider(t, &body)

		_, err := p.CompleteText(context.Background(), "You are a helpful assistant", "hello", 1024)
		require.NoError(t, err)

		system, ok := body["system"]
		require.True(t, ok, "system must be present in the request body when the system prompt is non-empty")
		blocks, ok := system.([]interface{})
		require.True(t, ok, "system must serialize as an array of content blocks (required for cache_control support)")
		require.Len(t, blocks, 1)
		block := blocks[0].(map[string]interface{})
		assert.Equal(t, "You are a helpful assistant", block["text"])
	})

	t.Run("system field omitted when empty", func(t *testing.T) {
		var body map[string]interface{}
		p := newCapturingProvider(t, &body)

		_, err := p.CompleteText(context.Background(), "", "hello", 1024)
		require.NoError(t, err)

		_, ok := body["system"]
		assert.False(t, ok, "system must be absent from the request body when the system prompt is empty")
	})
}

func TestClaudeProvider_CompleteText_MaxTokensDefaulting(t *testing.T) {
	var receivedReq map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewDecoder(r.Body).Decode(&receivedReq)
		resp := claudeResponse{
			Content:    []claudeContentBlock{{Type: "text", Text: "ok"}},
			StopReason: "end_turn",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))

	t.Run("zero defaults to ClaudeMaxTokens", func(t *testing.T) {
		_, err := p.CompleteText(context.Background(), "sys", "usr", 0)
		require.NoError(t, err)
		assert.Equal(t, float64(ClaudeMaxTokens), receivedReq["max_tokens"])
	})

	t.Run("negative defaults to ClaudeMaxTokens", func(t *testing.T) {
		_, err := p.CompleteText(context.Background(), "sys", "usr", -1)
		require.NoError(t, err)
		assert.Equal(t, float64(ClaudeMaxTokens), receivedReq["max_tokens"])
	})
}

func TestClaudeProvider_CompleteText_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key",
		WithClaudeBaseURL(server.URL),
		WithClaudeTimeout(50*time.Millisecond),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := p.CompleteText(ctx, "sys", "usr", 1024)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAITimeout)
}

func TestClaudeProvider_CompleteText_QuotaExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate_limited"}`))
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))
	_, err := p.CompleteText(context.Background(), "sys", "usr", 1024)

	assert.ErrorIs(t, err, ErrAIQuotaExceeded)
}

func TestClaudeProvider_CompleteText_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal"}`))
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))
	_, err := p.CompleteText(context.Background(), "sys", "usr", 1024)

	assert.ErrorIs(t, err, ErrAIProviderError)
}

func TestClaudeProvider_CompleteText_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := claudeResponse{Content: []claudeContentBlock{}, StopReason: "end_turn"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))
	_, err := p.CompleteText(context.Background(), "sys", "usr", 1024)

	assert.ErrorIs(t, err, ErrAIInvalidResponse)
}

func TestClaudeProvider_CompleteText_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json at all`))
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))
	_, err := p.CompleteText(context.Background(), "sys", "usr", 1024)

	assert.ErrorIs(t, err, ErrAIInvalidResponse)
}

// TestClaudeResponse_GetText retargets the old claudeResponse.GetText() table at
// textFromMessage, which extracts the first text block from an SDK
// *anthropic.Message (story sub-1-1 AC #2/#4). All four original cases survive.
//
// The message is built by decoding a canned wire body rather than by
// constructing anthropic.Message literals, so the union discriminator
// (ContentBlockUnion.Type) is populated the same way a real response would
// populate it — a hand-built literal would silently skip AsAny()'s switch.
func TestClaudeResponse_GetText(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "text content",
			body: `{"content":[{"type":"text","text":"hello"}]}`,
			want: "hello",
		},
		{
			name: "empty content",
			body: `{"content":[]}`,
			want: "",
		},
		{
			name: "non-text content",
			body: `{"content":[{"type":"thinking","thinking":"data"}]}`,
			want: "",
		},
		{
			name: "multiple blocks returns first text",
			body: `{"content":[{"type":"thinking","thinking":"image_data"},{"type":"text","text":"actual text"}]}`,
			want: "actual text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg anthropic.Message
			require.NoError(t, json.Unmarshal([]byte(tt.body), &msg))
			assert.Equal(t, tt.want, textFromMessage(&msg))
		})
	}

	t.Run("nil message", func(t *testing.T) {
		assert.Equal(t, "", textFromMessage(nil))
	})
}

// --- 9R-1: stale default model fix ---

func TestDefaultClaudeModel_CurrentAndCarriedInRequestBody(t *testing.T) {
	// AC1/AC3: default is non-empty and NOT the deprecated Haiku 3.5 alias
	// (retired 2026-02-19 -> 404).
	assert.NotEmpty(t, DefaultClaudeModel)
	assert.NotEqual(t, "claude-3-5-haiku-latest", DefaultClaudeModel)

	// AC3: the request body carries the default model.
	var receivedReq map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewDecoder(r.Body).Decode(&receivedReq)
		resp := claudeResponse{
			Content:    []claudeContentBlock{{Type: "text", Text: "ok"}},
			StopReason: "end_turn",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))
	_, err := p.CompleteText(context.Background(), "", "hello", 128)
	require.NoError(t, err)
	assert.Equal(t, DefaultClaudeModel, receivedReq["model"])
}

func TestClaudeProvider_NotFoundGuard_NamesBadModel(t *testing.T) {
	// AC3: a 404 not_found_error must surface an error naming the bad model.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"type":"error","error":{"type":"not_found_error","message":"model: bogus-model"}}`))
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL), WithClaudeModel("bogus-model"))

	_, err := p.CompleteText(context.Background(), "", "hello", 128)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAIProviderError)
	assert.ErrorIs(t, err, ErrAIModelNotFound, "sub-2-1a CR H2: the key-test endpoint classifies on this sentinel")
	assert.Contains(t, err.Error(), "bogus-model")

	_, err = p.Parse(context.Background(), &ParseRequest{Filename: "Some.Movie.2020.mkv"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAIProviderError)
	assert.ErrorIs(t, err, ErrAIModelNotFound)
	assert.Contains(t, err.Error(), "bogus-model")
}

func TestNewProvider_ClaudeModelOverride(t *testing.T) {
	// AC2: model id is config-overridable, not only a constant.
	provider, err := NewProvider(FactoryConfig{
		ProviderName: "claude",
		ClaudeAPIKey: "test-key",
		ClaudeModel:  "claude-opus-4-8",
	})
	require.NoError(t, err)
	cp, ok := provider.(*ClaudeProvider)
	require.True(t, ok)
	assert.Equal(t, "claude-opus-4-8", cp.model)

	provider, err = NewProvider(FactoryConfig{
		ProviderName: "claude",
		ClaudeAPIKey: "test-key",
	})
	require.NoError(t, err)
	cp, ok = provider.(*ClaudeProvider)
	require.True(t, ok)
	assert.Equal(t, DefaultClaudeModel, cp.model)
}

// --- 9R-11: metering + budget cutoff through the client ---

func TestClaudeProvider_MetersUsageToBudget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1000,"output_tokens":500}}`))
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL), WithClaudeModel("claude-haiku-4-5"))
	b := NewBudget(0)
	ctx := WithBudget(context.Background(), b)

	_, err := p.CompleteText(ctx, "", "hi", 64)
	require.NoError(t, err)

	snap := b.Snapshot()
	assert.Equal(t, int64(1000), snap.InputTokens)
	assert.Equal(t, int64(500), snap.OutputTokens)
	assert.Equal(t, 1, snap.LLMCalls)
	// Haiku: 1000/1M*$1 + 500/1M*$5 = 0.001 + 0.0025 = 0.0035
	assert.InDelta(t, 0.0035, snap.SpentUSD, 1e-9)
}

func TestClaudeProvider_BudgetCutoffStopsCall(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		hits.Add(1)
		w.Write([]byte(`{"content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn"}`))
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL), WithClaudeModel("claude-haiku-4-5"))
	b := NewBudget(1.0)
	b.RecordLLM("claude-haiku-4-5", 2_000_000, 0) // $2 → over ceiling
	ctx := WithBudget(context.Background(), b)

	_, err := p.CompleteText(ctx, "", "hi", 64)
	require.ErrorIs(t, err, ErrBudgetExceeded)
	assert.Equal(t, int32(0), hits.Load(), "no HTTP call once the budget is blown")
}

// --- story sub-1-1 AC #7: guards the pre-existing suite structurally cannot provide ---

// TestClaudeProvider_SDKRetriesDisabled is the NAIL 2 proof (AC #3). The SDK
// retries twice by default and retryTransient retries three times; if the SDK's
// retries were left on, one logical call would make up to 2x3 = 6 real requests
// while the Governor's budget pre-check ran ONCE — silently bypassing cost
// control. No other test in the suite can observe that.
func TestClaudeProvider_SDKRetriesDisabled(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))
	_, err := p.CompleteText(context.Background(), "", "hello", 64)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAIProviderError)
	assert.Equal(t, int32(retryMaxAttempts), hits.Load(),
		"exactly retryMaxAttempts real requests — retryTransient is the ONLY retry layer (D8)")
}

// TestClaudeProvider_RequestPathIsV1Messages locks AC #6. Every pre-existing test
// asserts Contains(path, "/messages"), which passes for "/v1/messages",
// "/v1/v1/messages" AND "/messages" alike — so a base URL that kept its "/v1"
// suffix would pass the entire suite and 404 in production.
func TestClaudeProvider_RequestPathIsV1Messages(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		resp := claudeResponse{Content: []claudeContentBlock{{Type: "text", Text: "ok"}}, StopReason: "end_turn"}
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))
	_, err := p.CompleteText(context.Background(), "", "hello", 64)
	require.NoError(t, err)

	assert.Equal(t, "/v1/messages", gotPath,
		"the SDK appends v1/messages itself — DefaultClaudeBaseURL must NOT carry a /v1 suffix")
}

// TestClaudeProvider_MalformedJSONNotRetried locks the AC #4 trap. The hand-rolled
// client decoded OUTSIDE retryTransient, so a garbage 200 body cost exactly one
// request. The SDK decodes INSIDE Messages.New — i.e. inside the retry loop — so
// classifying a decode failure as retryable would silently triple the cost of
// every malformed response.
func TestClaudeProvider_MalformedJSONNotRetried(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json at all`))
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))
	_, err := p.CompleteText(context.Background(), "", "hello", 64)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAIInvalidResponse)
	assert.Equal(t, int32(1), hits.Load(), "a malformed body is permanent — it must NOT be retried")
}

// TestClaudeProvider_CompleteTextWithUsage_CacheControlAndUsage covers AC #5:
// ordered system blocks, per-block cache_control, and both cache-token
// dimensions reaching the caller.
func TestClaudeProvider_CompleteTextWithUsage_CacheControlAndUsage(t *testing.T) {
	newProvider := func(t *testing.T, captured *map[string]interface{}, respBody string) *ClaudeProvider {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]interface{}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			if captured != nil {
				*captured = body
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(respBody))
		}))
		t.Cleanup(server.Close)
		return NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))
	}

	const okBody = `{"content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn"}`

	t.Run("blocks keep order and only the marked block carries cache_control", func(t *testing.T) {
		var body map[string]interface{}
		p := newProvider(t, &body, okBody)

		_, err := p.CompleteTextWithUsage(context.Background(), CompletionRequest{
			System: []SystemBlock{
				{Text: "stable rules"},
				{Text: "per-show metadata", CacheTTL: CacheTTL1h},
			},
			UserPrompt: "translate",
			MaxTokens:  256,
		})
		require.NoError(t, err)

		blocks, ok := body["system"].([]interface{})
		require.True(t, ok, "system must be an array of content blocks")
		require.Len(t, blocks, 2)

		first := blocks[0].(map[string]interface{})
		second := blocks[1].(map[string]interface{})
		assert.Equal(t, "stable rules", first["text"], "prefix order is semantic and must be preserved")
		assert.Equal(t, "per-show metadata", second["text"])

		_, firstHasCC := first["cache_control"]
		assert.False(t, firstHasCC, "an unmarked block must not become a cache breakpoint")

		cc, ok := second["cache_control"].(map[string]interface{})
		require.True(t, ok, "the marked block must carry cache_control")
		assert.Equal(t, "ephemeral", cc["type"])
		assert.Equal(t, "1h", cc["ttl"])
	})

	t.Run("CacheTTL5m emits the default ephemeral breakpoint", func(t *testing.T) {
		var body map[string]interface{}
		p := newProvider(t, &body, okBody)

		_, err := p.CompleteTextWithUsage(context.Background(), CompletionRequest{
			System:     []SystemBlock{{Text: "stable rules", CacheTTL: CacheTTL5m}},
			UserPrompt: "translate",
		})
		require.NoError(t, err)

		blocks := body["system"].([]interface{})
		cc, ok := blocks[0].(map[string]interface{})["cache_control"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "ephemeral", cc["type"])
		assert.Empty(t, cc["ttl"], "the 5m default is implicit — no explicit ttl is sent")
	})

	t.Run("CacheTTLNone everywhere emits no cache_control at all", func(t *testing.T) {
		var body map[string]interface{}
		p := newProvider(t, &body, okBody)

		_, err := p.CompleteTextWithUsage(context.Background(), CompletionRequest{
			System:     []SystemBlock{{Text: "a"}, {Text: "b"}},
			UserPrompt: "translate",
		})
		require.NoError(t, err)

		for i, raw := range body["system"].([]interface{}) {
			_, has := raw.(map[string]interface{})["cache_control"]
			assert.Falsef(t, has, "block %d must not carry cache_control", i)
		}
	})

	t.Run("cache token dimensions reach the caller", func(t *testing.T) {
		p := newProvider(t, nil, `{"content":[{"type":"text","text":"translated"}],"stop_reason":"end_turn",
			"usage":{"input_tokens":120,"output_tokens":34,"cache_creation_input_tokens":4096,"cache_read_input_tokens":2048}}`)

		res, err := p.CompleteTextWithUsage(context.Background(), CompletionRequest{
			System:     []SystemBlock{{Text: "stable", CacheTTL: CacheTTL1h}},
			UserPrompt: "translate",
		})
		require.NoError(t, err)

		assert.Equal(t, "translated", res.Text)
		assert.Equal(t, int64(120), res.Usage.InputTokens)
		assert.Equal(t, int64(34), res.Usage.OutputTokens)
		assert.Equal(t, int64(4096), res.Usage.CacheCreationInputTokens)
		assert.Equal(t, int64(2048), res.Usage.CacheReadInputTokens)
	})

	t.Run("zero cache tokens are surfaced, not hidden", func(t *testing.T) {
		// The exact signal sub-1-5b reads to detect a silently-inert prefix cache.
		p := newProvider(t, nil, okBody)

		res, err := p.CompleteTextWithUsage(context.Background(), CompletionRequest{
			System:     []SystemBlock{{Text: "too short to cache", CacheTTL: CacheTTL1h}},
			UserPrompt: "translate",
		})
		require.NoError(t, err)
		assert.Zero(t, res.Usage.CacheCreationInputTokens)
		assert.Zero(t, res.Usage.CacheReadInputTokens)
	})
}

// --- code-review follow-ups (story sub-1-1, adversarial CR 2026-07-28) ---

// TestClaudeProvider_TimeoutEnforcedWithCustomHTTPClient locks review finding M1.
// The hand-rolled doRequest wrapped EVERY attempt in
// context.WithTimeout(ctx, p.timeout) regardless of which http.Client was in
// use; the SDK build only carried the timeout on the default-built client, so
// WithClaudeHTTPClient + WithClaudeTimeout silently lost the deadline. Guarded
// via option.WithRequestTimeout on the custom-client branch.
func TestClaudeProvider_TimeoutEnforcedWithCustomHTTPClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		resp := claudeResponse{Content: []claudeContentBlock{{Type: "text", Text: "late"}}, StopReason: "end_turn"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key",
		WithClaudeBaseURL(server.URL),
		WithClaudeHTTPClient(&http.Client{}), // deliberately carries NO Timeout of its own
		WithClaudeTimeout(50*time.Millisecond),
	)

	_, err := p.CompleteText(context.Background(), "", "hello", 64)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAITimeout,
		"WithClaudeTimeout must be enforced per attempt even with a caller-supplied client")
}

// TestClaudeProvider_NonJSONSuccessNotRetried locks review finding M2. A 2xx
// whose Content-Type is not JSON (e.g. a broken proxy serving an HTML page with
// status 200) is permanent, exactly like a malformed JSON body — without the
// rejectNonJSONSuccess middleware the SDK's content-type rejection error is
// neither an *anthropic.Error nor a *json.SyntaxError, so it fell into the
// connection-error fallback and was retried 3x.
func TestClaudeProvider_NonJSONSuccessNotRetried(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>totally not the Messages API</body></html>`))
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))
	_, err := p.CompleteText(context.Background(), "", "hello", 64)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAIInvalidResponse,
		"a non-JSON 2xx is an invalid response, not a provider/connection error")
	assert.Equal(t, int32(1), hits.Load(), "a non-JSON 2xx is permanent — it must NOT be retried")
}

// sub-6-5: Model() is the single truth for run rows and cache keys.
func TestClaudeProvider_ModelAccessor(t *testing.T) {
	assert.Equal(t, DefaultClaudeModel, NewClaudeProvider("k").Model())
	assert.Equal(t, "claude-sonnet-5", NewClaudeProvider("k", WithClaudeModel("claude-sonnet-5")).Model())
}

// ─── sub-6-6: Ping judges transport only ────────────────────────────────────

func pingServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func TestClaudeProvider_Ping_EmptyContentIsValid(t *testing.T) {
	// Sonnet 5 at max_tokens=1 routinely returns no text block; the key is fine.
	srv := pingServer(t, 200, `{"id":"m","type":"message","role":"assistant","model":"claude-sonnet-5","content":[],"stop_reason":"max_tokens","usage":{"input_tokens":1,"output_tokens":0}}`)
	defer srv.Close()
	assert.NoError(t, NewClaudeProvider("k", WithClaudeBaseURL(srv.URL)).Ping(context.Background()))
}

func TestClaudeProvider_Ping_Unauthorized(t *testing.T) {
	srv := pingServer(t, 401, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
	defer srv.Close()
	err := NewClaudeProvider("bad", WithClaudeBaseURL(srv.URL)).Ping(context.Background())
	assert.ErrorIs(t, err, ErrAIUnauthorized)
}

func TestClaudeProvider_Ping_ModelNotFound(t *testing.T) {
	srv := pingServer(t, 404, `{"type":"error","error":{"type":"not_found_error","message":"model: nope"}}`)
	defer srv.Close()
	err := NewClaudeProvider("k", WithClaudeBaseURL(srv.URL), WithClaudeModel("nope")).Ping(context.Background())
	assert.ErrorIs(t, err, ErrAIModelNotFound)
}

func TestClaudeProvider_Ping_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()
	prevBase, prevMax := retryBaseDelay, retryMaxDelay
	retryBaseDelay, retryMaxDelay = 5*time.Millisecond, 10*time.Millisecond
	t.Cleanup(func() { retryBaseDelay, retryMaxDelay = prevBase, prevMax })
	err := NewClaudeProvider("k", WithClaudeBaseURL(srv.URL), WithClaudeTimeout(50*time.Millisecond)).Ping(context.Background())
	assert.ErrorIs(t, err, ErrAITimeout)
}
