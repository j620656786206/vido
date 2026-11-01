package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// disc-2026-11-burned-in-subs-b1: one user turn = the image block, then the
// prompt; thinking off when asked; text and usage come back like a text call.
func TestClaudeProvider_CompleteImageWithUsage(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body = nil
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"content":[{"type":"text","text":"{\"burned_chinese\": false, \"lines\": []}"}],"stop_reason":"end_turn","usage":{"input_tokens":520,"output_tokens":50}}`)
	}))
	defer server.Close()
	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL), WithClaudeModel("claude-sonnet-5"))

	png := []byte("\x89PNG fake")
	res, err := p.CompleteImageWithUsage(WithThinkingOff(context.Background()), ImageCompletionRequest{
		Image: png, MediaType: "image/png", Prompt: "read the subtitles", MaxTokens: 400,
	})
	require.NoError(t, err)
	assert.Equal(t, `{"burned_chinese": false, "lines": []}`, res.Text)
	assert.Equal(t, int64(520), res.Usage.InputTokens)
	assert.Equal(t, int64(50), res.Usage.OutputTokens)

	assert.Equal(t, map[string]any{"type": "disabled"}, body["thinking"])
	assert.EqualValues(t, 400, body["max_tokens"])
	messages := body["messages"].([]any)
	require.Len(t, messages, 1)
	content := messages[0].(map[string]any)["content"].([]any)
	require.Len(t, content, 2)
	image := content[0].(map[string]any)
	assert.Equal(t, "image", image["type"])
	assert.Equal(t, map[string]any{"type": "base64", "media_type": "image/png", "data": base64.StdEncoding.EncodeToString(png)}, image["source"])
	assert.Equal(t, map[string]any{"type": "text", "text": "read the subtitles"}, content[1])

	_, err = p.CompleteImageWithUsage(context.Background(), ImageCompletionRequest{Image: png, MediaType: "image/png", Prompt: "x"})
	require.NoError(t, err)
	assert.NotContains(t, body, "thinking", "a request that did not ask keeps the model's default")
}

func TestClaudeProvider_CompleteImageWithUsage_EmptyReplyKeepsUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"content":[],"stop_reason":"refusal","usage":{"input_tokens":520,"output_tokens":3}}`)
	}))
	defer server.Close()
	res, err := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL)).
		CompleteImageWithUsage(context.Background(), ImageCompletionRequest{Image: []byte("x"), MediaType: "image/png", Prompt: "x"})
	assert.True(t, errors.Is(err, ErrAIEmptyReply), "%v", err)
	assert.ErrorIs(t, err, ErrAIInvalidResponse)
	assert.Equal(t, int64(520), res.Usage.InputTokens, "a no-text reply is still billed")

	_, err = NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL)).
		CompleteImageWithUsage(context.Background(), ImageCompletionRequest{MediaType: "image/png", Prompt: "x"})
	assert.Error(t, err, "no image, no request")
}
