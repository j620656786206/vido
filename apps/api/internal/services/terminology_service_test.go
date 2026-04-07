package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/config"
)

// newTestClaudeProvider creates a ClaudeProvider backed by a test HTTP server.
func newTestClaudeProvider(handler http.HandlerFunc) (*ai.ClaudeProvider, *httptest.Server) {
	server := httptest.NewServer(handler)
	provider := ai.NewClaudeProvider("test-key",
		ai.WithClaudeBaseURL(server.URL),
		ai.WithClaudeHTTPClient(server.Client()),
	)
	return provider, server
}

// claudeTextResponse builds a mock Claude API response with the given text.
func claudeTextResponse(text string) []byte {
	resp := map[string]interface{}{
		"content": []map[string]string{
			{"type": "text", "text": text},
		},
		"stop_reason": "end_turn",
	}
	b, _ := json.Marshal(resp)
	return b
}

func TestTerminologyCorrectionService_Correct_Success(t *testing.T) {
	provider, server := newTestClaudeProvider(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(claudeTextResponse("這個軟體很好用"))
	})
	defer server.Close()

	svc := NewTerminologyCorrectionServiceWithProvider(provider)
	result, err := svc.Correct(context.Background(), "這個軟件很好用")

	require.NoError(t, err)
	assert.Equal(t, "這個軟體很好用", result)
}

func TestTerminologyCorrectionService_Correct_EmptyContent(t *testing.T) {
	provider, server := newTestClaudeProvider(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call API for empty content")
	})
	defer server.Close()

	svc := NewTerminologyCorrectionServiceWithProvider(provider)
	result, err := svc.Correct(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestTerminologyCorrectionService_Correct_NilProvider(t *testing.T) {
	svc := &TerminologyCorrectionService{provider: nil}
	result, err := svc.Correct(context.Background(), "some content")

	require.NoError(t, err)
	assert.Equal(t, "some content", result, "should return original when provider is nil")
}

func TestTerminologyCorrectionService_Correct_APIError_FallsBack(t *testing.T) {
	provider, server := newTestClaudeProvider(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	})
	defer server.Close()

	svc := NewTerminologyCorrectionServiceWithProvider(provider)
	original := "這個軟件很好用"
	result, err := svc.Correct(context.Background(), original)

	assert.Error(t, err, "should return error")
	assert.Equal(t, original, result, "should return original content on error (AC #4)")
}

func TestTerminologyCorrectionService_Correct_Timeout_FallsBack(t *testing.T) {
	provider, server := newTestClaudeProvider(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Simulate slow response
		w.WriteHeader(http.StatusOK)
		w.Write(claudeTextResponse("corrected"))
	})
	defer server.Close()

	svc := NewTerminologyCorrectionServiceWithProvider(provider)

	// Use a short timeout to trigger deadline exceeded
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	original := "這個軟件很好用"
	result, err := svc.Correct(ctx, original)

	assert.Error(t, err, "should return error on timeout")
	assert.Equal(t, original, result, "should return original content on timeout (AC #4)")
}

func TestTerminologyCorrectionService_Correct_QuotaExceeded_FallsBack(t *testing.T) {
	provider, server := newTestClaudeProvider(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limited"}`))
	})
	defer server.Close()

	svc := NewTerminologyCorrectionServiceWithProvider(provider)
	original := "視頻播放器"
	result, err := svc.Correct(context.Background(), original)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ai.ErrAIQuotaExceeded)
	assert.Equal(t, original, result, "should return original on quota exceeded")
}

func TestTerminologyCorrectionService_IsConfigured(t *testing.T) {
	t.Run("nil service returns false", func(t *testing.T) {
		var svc *TerminologyCorrectionService
		assert.False(t, svc.IsConfigured())
	})

	t.Run("nil provider returns false", func(t *testing.T) {
		svc := &TerminologyCorrectionService{provider: nil}
		assert.False(t, svc.IsConfigured())
	})

	t.Run("with provider returns true", func(t *testing.T) {
		provider, server := newTestClaudeProvider(nil)
		defer server.Close()
		svc := NewTerminologyCorrectionServiceWithProvider(provider)
		assert.True(t, svc.IsConfigured())
	})
}

func TestNewTerminologyCorrectionService_NoKey(t *testing.T) {
	cfg := &config.Config{ClaudeAPIKey: ""}
	svc := NewTerminologyCorrectionService(cfg)
	assert.Nil(t, svc, "should return nil when no Claude key configured (AC #2)")
}

func TestNewTerminologyCorrectionService_WithKey(t *testing.T) {
	cfg := &config.Config{ClaudeAPIKey: "test-key-123"}
	svc := NewTerminologyCorrectionService(cfg)
	require.NotNil(t, svc)
	assert.True(t, svc.IsConfigured())
}

func TestTerminologyCorrectionService_Correct_SendsSystemPrompt(t *testing.T) {
	var receivedBody map[string]interface{}

	provider, server := newTestClaudeProvider(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write(claudeTextResponse("corrected text"))
	})
	defer server.Close()

	svc := NewTerminologyCorrectionServiceWithProvider(provider)
	_, err := svc.Correct(context.Background(), "test content")
	require.NoError(t, err)

	// Verify system prompt was sent
	system, ok := receivedBody["system"].(string)
	assert.True(t, ok, "request should include system prompt")
	assert.Contains(t, system, "Taiwan", "system prompt should mention Taiwan")
	assert.Contains(t, system, "cross-strait", "system prompt should mention cross-strait")
}
