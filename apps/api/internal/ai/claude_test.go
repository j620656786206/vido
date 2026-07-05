package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-api-key", WithClaudeBaseURL(server.URL))
	result, err := p.CompleteText(context.Background(), "system prompt", "user prompt", 2048)

	require.NoError(t, err)
	assert.Equal(t, "這個軟體很好用", result)

	// Verify system prompt is in the request
	assert.Equal(t, "system prompt", receivedReq["system"])
	// Verify max_tokens
	assert.Equal(t, float64(2048), receivedReq["max_tokens"])
	// Verify messages
	messages := receivedReq["messages"].([]interface{})
	assert.Len(t, messages, 1)
	msg := messages[0].(map[string]interface{})
	assert.Equal(t, "user", msg["role"])
	assert.Equal(t, "user prompt", msg["content"])
}

func TestClaudeProvider_CompleteText_SystemFieldSerialization(t *testing.T) {
	t.Run("system field included when non-empty", func(t *testing.T) {
		req := claudeRequest{
			Model:     "test-model",
			MaxTokens: 1024,
			System:    "You are a helpful assistant",
			Messages:  []claudeMessage{{Role: "user", Content: "hello"}},
		}
		data, err := json.Marshal(req)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"system":"You are a helpful assistant"`)
	})

	t.Run("system field omitted when empty", func(t *testing.T) {
		req := claudeRequest{
			Model:     "test-model",
			MaxTokens: 1024,
			System:    "",
			Messages:  []claudeMessage{{Role: "user", Content: "hello"}},
		}
		data, err := json.Marshal(req)
		require.NoError(t, err)
		assert.NotContains(t, string(data), `"system"`)
	})
}

func TestClaudeProvider_CompleteText_MaxTokensDefaulting(t *testing.T) {
	var receivedReq map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json at all`))
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))
	_, err := p.CompleteText(context.Background(), "sys", "usr", 1024)

	assert.ErrorIs(t, err, ErrAIInvalidResponse)
}

func TestClaudeResponse_GetText(t *testing.T) {
	tests := []struct {
		name     string
		response claudeResponse
		want     string
	}{
		{
			name: "text content",
			response: claudeResponse{
				Content: []claudeContentBlock{
					{Type: "text", Text: "hello"},
				},
			},
			want: "hello",
		},
		{
			name:     "empty content",
			response: claudeResponse{Content: []claudeContentBlock{}},
			want:     "",
		},
		{
			name: "non-text content",
			response: claudeResponse{
				Content: []claudeContentBlock{
					{Type: "image", Text: "data"},
				},
			},
			want: "",
		},
		{
			name: "multiple blocks returns first text",
			response: claudeResponse{
				Content: []claudeContentBlock{
					{Type: "image", Text: "image_data"},
					{Type: "text", Text: "actual text"},
				},
			},
			want: "actual text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.response.GetText())
		})
	}
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
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"type":"error","error":{"type":"not_found_error","message":"model: bogus-model"}}`))
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL), WithClaudeModel("bogus-model"))

	_, err := p.CompleteText(context.Background(), "", "hello", 128)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAIProviderError)
	assert.Contains(t, err.Error(), "bogus-model")

	_, err = p.Parse(context.Background(), &ParseRequest{Filename: "Some.Movie.2020.mkv"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAIProviderError)
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
