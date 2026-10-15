package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// disc-2026-10-translation-batch-truncation: a translation request asks for
// no reasoning tokens; every other request is untouched.
func TestClaudeProvider_ThinkingOffIsSentOnlyWhenAsked(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body = nil
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn"}`)
	}))
	defer server.Close()
	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))

	_, err := p.CompleteTextWithUsage(WithThinkingOff(context.Background()), CompletionRequest{UserPrompt: "translate", MaxTokens: 256})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"type": "disabled"}, body["thinking"])

	_, err = p.CompleteText(context.Background(), "", "judge this", 256)
	require.NoError(t, err)
	assert.NotContains(t, body, "thinking", "a request that did not ask keeps the model's default")
}

// The S01E04 failure shape: 4096 output tokens, stop_reason=max_tokens, no
// text block (all of it went to thinking).
func TestClaudeProvider_MaxTokensWithNoTextIsTruncated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"content":[{"type":"thinking","thinking":"","signature":"x"}],"stop_reason":"max_tokens","usage":{"input_tokens":957,"output_tokens":4096}}`)
	}))
	defer server.Close()
	_, err := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL)).CompleteTextWithUsage(context.Background(), CompletionRequest{UserPrompt: "x", MaxTokens: 4096})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAIResponseTruncated), "%v", err)
	assert.True(t, errors.Is(err, ErrAIInvalidResponse), "old checks keep matching")
}

func TestClaudeProvider_EmptyReplyThatIsNotTruncatedStaysInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"content":[],"stop_reason":"end_turn"}`)
	}))
	defer server.Close()
	_, err := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL)).CompleteTextWithUsage(context.Background(), CompletionRequest{UserPrompt: "x"})
	assert.ErrorIs(t, err, ErrAIInvalidResponse)
	assert.NotErrorIs(t, err, ErrAIResponseTruncated)
}

func TestOpenAICompat_LengthFinishWithNoTextIsTruncated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":""},"finish_reason":"length"}]}`)
	}))
	defer server.Close()
	_, err := NewOpenAICompatProvider(server.URL, "gemini-3.5-flash-lite").CompleteText(context.Background(), "", "u", 1)
	assert.ErrorIs(t, err, ErrAIResponseTruncated)
	assert.ErrorIs(t, err, ErrAIInvalidResponse)
}
