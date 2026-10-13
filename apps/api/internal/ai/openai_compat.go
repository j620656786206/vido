package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAICompatProvider is a TextCompleter for any server that speaks the
// OpenAI chat-completions shape — Ollama, llama.cpp's llama-server, vLLM,
// LM Studio. spike-local-translation-golden-eval uses it so cmd/grade can
// score a model running on the NAS; the app's translation path does not use
// it yet (that dispatch is backlog-gemini-translation-dispatch).
//
// Two deliberate differences from the cloud providers:
//   - The timeout is the caller's, not RequestTimeoutFor's. That table is
//     sized for cloud latency and capped at MaxRequestTimeout (180 s); a 7–12B
//     model on a CPU-only NAS can need several minutes for one batch.
//   - Nothing is recorded on the ctx Budget: local inference costs $0, and a
//     price lookup for an uncatalogued model id would be meaningless.
type OpenAICompatProvider struct {
	baseURL string
	model   string
	apiKey  string
	timeout time.Duration
	client  *http.Client
}

// DefaultOpenAICompatTimeout is the per-attempt deadline when the caller sets
// none: generous on purpose, see the type comment.
const DefaultOpenAICompatTimeout = 10 * time.Minute

// openAICompatErrBodyLimit caps how much of an error body is quoted back.
const openAICompatErrBodyLimit = 512

// OpenAICompatOption configures an OpenAICompatProvider.
type OpenAICompatOption func(*OpenAICompatProvider)

// WithOpenAICompatAPIKey sends "Authorization: Bearer <key>". Empty = no
// header at all (Ollama and llama-server need none by default).
func WithOpenAICompatAPIKey(key string) OpenAICompatOption {
	return func(p *OpenAICompatProvider) { p.apiKey = strings.TrimSpace(key) }
}

// WithOpenAICompatTimeout sets the per-attempt deadline (each retry gets its
// own). Zero or negative keeps DefaultOpenAICompatTimeout.
func WithOpenAICompatTimeout(d time.Duration) OpenAICompatOption {
	return func(p *OpenAICompatProvider) {
		if d > 0 {
			p.timeout = d
		}
	}
}

// WithOpenAICompatHTTPClient replaces the HTTP client (tests).
func WithOpenAICompatHTTPClient(c *http.Client) OpenAICompatOption {
	return func(p *OpenAICompatProvider) {
		if c != nil {
			p.client = c
		}
	}
}

// NewOpenAICompatProvider builds a completer for baseURL (the part before
// "/chat/completions", e.g. "http://nas:11434/v1") serving model.
func NewOpenAICompatProvider(baseURL, model string, opts ...OpenAICompatOption) *OpenAICompatProvider {
	p := &OpenAICompatProvider{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		model:   strings.TrimSpace(model),
		timeout: DefaultOpenAICompatTimeout,
		client:  &http.Client{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Model returns the model id requests are sent with.
func (p *OpenAICompatProvider) Model() string { return p.model }

type openAICompatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAICompatRequest struct {
	Model     string                `json:"model"`
	Messages  []openAICompatMessage `json:"messages"`
	MaxTokens int                   `json:"max_tokens,omitempty"`
	Stream    bool                  `json:"stream"`
}

type openAICompatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// CompleteText sends one chat completion: an optional system message, then
// the user message. Transient failures (network, timeout, 429, 5xx) retry with
// the package's bounded backoff; other 4xx fail at once with the server's own
// error text, so "model not found" reads as such instead of as a retry storm.
func (p *OpenAICompatProvider) CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	if p.baseURL == "" || p.model == "" {
		return "", fmt.Errorf("%w: openai-compatible provider needs a base URL and a model", ErrAIProviderError)
	}
	msgs := make([]openAICompatMessage, 0, 2)
	if systemPrompt != "" {
		msgs = append(msgs, openAICompatMessage{Role: "system", Content: systemPrompt})
	}
	msgs = append(msgs, openAICompatMessage{Role: "user", Content: userPrompt})
	body, err := json.Marshal(openAICompatRequest{Model: p.model, Messages: msgs, MaxTokens: maxTokens})
	if err != nil {
		return "", fmt.Errorf("%w: encode request: %v", ErrAIProviderError, err)
	}
	return retryTransient(ctx, "openai-compat:"+p.model, func() (string, bool, error) {
		return p.attempt(ctx, body)
	})
}

// attempt is one HTTP round trip; it reports whether its failure is worth
// another try.
func (p *OpenAICompatProvider) attempt(ctx context.Context, body []byte) (string, bool, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", false, fmt.Errorf("%w: build request: %v", ErrAIProviderError, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			// The caller's ctx ended the call — not ours to retry.
			return "", false, ctx.Err()
		}
		if errors.Is(attemptCtx.Err(), context.DeadlineExceeded) {
			return "", true, fmt.Errorf("%w: no reply within %s", ErrAITimeout, p.timeout)
		}
		return "", true, fmt.Errorf("%w: %v", ErrAIProviderError, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		if ctx.Err() != nil {
			return "", false, ctx.Err()
		}
		if errors.Is(attemptCtx.Err(), context.DeadlineExceeded) {
			return "", true, fmt.Errorf("%w: no reply within %s", ErrAITimeout, p.timeout)
		}
		return "", true, fmt.Errorf("%w: read body: %v", ErrAIProviderError, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", isTransientStatus(resp.StatusCode),
			fmt.Errorf("%w: HTTP %d: %s", ErrAIProviderError, resp.StatusCode, openAICompatErrorText(data))
	}

	var out openAICompatResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return "", false, fmt.Errorf("%w: %v", ErrAIInvalidResponse, err)
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return "", false, ErrAIInvalidResponse
	}
	return out.Choices[0].Message.Content, false, nil
}

// openAICompatErrorText pulls the human-readable message out of an OpenAI-style
// {"error":{"message":…}} (or {"error":"…"}) body, falling back to the raw
// body, truncated.
func openAICompatErrorText(data []byte) string {
	var shaped struct {
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(data, &shaped) == nil && len(shaped.Error) > 0 {
		var obj struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(shaped.Error, &obj) == nil && obj.Message != "" {
			return openAICompatTruncate(obj.Message)
		}
		var s string
		if json.Unmarshal(shaped.Error, &s) == nil && s != "" {
			return openAICompatTruncate(s)
		}
	}
	return openAICompatTruncate(strings.TrimSpace(string(data)))
}

func openAICompatTruncate(s string) string {
	if len(s) > openAICompatErrBodyLimit {
		return s[:openAICompatErrBodyLimit] + "…"
	}
	return s
}
