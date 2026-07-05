package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	// DefaultClaudeBaseURL is the base URL for Claude API.
	DefaultClaudeBaseURL = "https://api.anthropic.com/v1"
	// DefaultClaudeModel is the default model to use. Must be a current,
	// non-deprecated alias: the previous default "claude-3-5-haiku-latest"
	// (Haiku 3.5) was retired 2026-02-19 and returns 404 (9R-1).
	// Override per-deployment via CLAUDE_MODEL.
	DefaultClaudeModel = "claude-haiku-4-5"
	// ClaudeAPIVersion is the required API version header.
	ClaudeAPIVersion = "2023-06-01"
	// ClaudeMaxTokens is the max tokens for response.
	ClaudeMaxTokens = 1024
)

// ClaudeProvider implements the Provider interface for Anthropic's Claude AI.
type ClaudeProvider struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	timeout    time.Duration
	governor   *Governor // 9R-11: shared throttle (nil = unthrottled)
}

// Compile-time interface verification.
var _ Provider = (*ClaudeProvider)(nil)

// ClaudeProviderOption is a functional option for configuring ClaudeProvider.
type ClaudeProviderOption func(*ClaudeProvider)

// WithClaudeBaseURL sets a custom base URL.
func WithClaudeBaseURL(url string) ClaudeProviderOption {
	return func(p *ClaudeProvider) {
		p.baseURL = url
	}
}

// WithClaudeModel sets a custom model.
func WithClaudeModel(model string) ClaudeProviderOption {
	return func(p *ClaudeProvider) {
		p.model = model
	}
}

// WithClaudeHTTPClient sets a custom HTTP client (useful for testing).
func WithClaudeHTTPClient(client *http.Client) ClaudeProviderOption {
	return func(p *ClaudeProvider) {
		p.httpClient = client
	}
}

// WithClaudeTimeout sets a custom timeout.
func WithClaudeTimeout(timeout time.Duration) ClaudeProviderOption {
	return func(p *ClaudeProvider) {
		p.timeout = timeout
	}
}

// WithClaudeGovernor injects the shared throttle (Story 9R-11).
func WithClaudeGovernor(g *Governor) ClaudeProviderOption {
	return func(p *ClaudeProvider) {
		p.governor = g
	}
}

// NewClaudeProvider creates a new Claude AI provider.
func NewClaudeProvider(apiKey string, opts ...ClaudeProviderOption) *ClaudeProvider {
	p := &ClaudeProvider{
		apiKey:  apiKey,
		baseURL: DefaultClaudeBaseURL,
		model:   DefaultClaudeModel,
		timeout: DefaultTimeoutSeconds * time.Second,
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.httpClient == nil {
		p.httpClient = &http.Client{
			Timeout: p.timeout,
		}
	}

	return p
}

// Name returns the provider name.
func (p *ClaudeProvider) Name() ProviderName {
	return ProviderClaude
}

// doRequest POSTs a marshaled Messages API body with bounded retry on
// transient failures (9R-4): network errors, per-attempt timeouts, 429 and
// 5xx retry with exponential backoff; other 4xx (including the 9R-1 404
// model guard) fail immediately.
func (p *ClaudeProvider) doRequest(ctx context.Context, body []byte) ([]byte, error) {
	url := fmt.Sprintf("%s/messages", p.baseURL)

	// 9R-11: budget pre-check + shared throttle around the retrying request.
	respBody, err := governed(ctx, p.governor, "claude.messages", func() ([]byte, error) {
		return retryTransient(ctx, "claude.messages", func() ([]byte, bool, error) {
			attemptCtx, cancel := context.WithTimeout(ctx, p.timeout)
			defer cancel()

			httpReq, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, url, bytes.NewReader(body))
			if err != nil {
				return nil, false, fmt.Errorf("failed to create request: %w", err)
			}
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("x-api-key", p.apiKey)
			httpReq.Header.Set("anthropic-version", ClaudeAPIVersion)

			resp, err := p.httpClient.Do(httpReq)
			if err != nil {
				if attemptCtx.Err() == context.DeadlineExceeded {
					slog.Warn("Claude API timeout", "timeout_seconds", p.timeout.Seconds())
					return nil, true, ErrAITimeout
				}
				return nil, true, fmt.Errorf("%w: %v", ErrAIProviderError, err)
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, true, fmt.Errorf("failed to read response: %w", err)
			}

			if resp.StatusCode != http.StatusOK {
				slog.Warn("Claude API error response",
					"status_code", resp.StatusCode,
					"body", string(respBody),
				)
				if resp.StatusCode == http.StatusTooManyRequests {
					return nil, true, ErrAIQuotaExceeded
				}
				if resp.StatusCode == http.StatusNotFound {
					slog.Error("Claude model not found — the configured model id is deprecated or invalid",
						"model", p.model,
					)
					return nil, false, fmt.Errorf("%w: status 404: model %q not found (deprecated or invalid model id — set CLAUDE_MODEL to a current model)", ErrAIProviderError, p.model)
				}
				return nil, isTransientStatus(resp.StatusCode), fmt.Errorf("%w: status %d", ErrAIProviderError, resp.StatusCode)
			}

			return respBody, false, nil
		})
	})
	if err != nil {
		return nil, err
	}

	// 9R-11: meter token usage against the per-run budget (best-effort — a
	// usage-less response just records zero).
	if b := BudgetFromContext(ctx); b != nil {
		var u struct {
			Usage claudeUsage `json:"usage"`
		}
		if json.Unmarshal(respBody, &u) == nil {
			b.RecordLLM(p.model, u.Usage.InputTokens, u.Usage.OutputTokens)
		}
	}
	return respBody, nil
}

// Parse sends a filename to Claude for parsing.
func (p *ClaudeProvider) Parse(ctx context.Context, req *ParseRequest) (*ParseResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Build prompt
	prompt := req.Prompt
	if prompt == "" {
		prompt = fmt.Sprintf(DefaultPrompt, req.Filename)
	}

	// Build request body
	claudeReq := claudeRequest{
		Model:     p.model,
		MaxTokens: ClaudeMaxTokens,
		Messages: []claudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	slog.Debug("Claude API request",
		"model", p.model,
		"filename", req.Filename,
	)

	// Execute with bounded transient retry (9R-4)
	respBody, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	// Parse Claude response
	var claudeResp claudeResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		slog.Error("Failed to parse Claude response",
			"error", err,
			"body", string(respBody),
		)
		return nil, fmt.Errorf("%w: %v", ErrAIInvalidResponse, err)
	}

	// Extract text from response
	text := claudeResp.GetText()
	if text == "" {
		slog.Warn("Empty response from Claude",
			"filename", req.Filename,
		)
		return nil, ErrAIInvalidResponse
	}

	// Parse the JSON response into ParseResponse
	result, err := parseJSONResponse(text)
	if err != nil {
		slog.Error("Failed to parse JSON from Claude response",
			"error", err,
			"raw_response", text,
		)
		return nil, fmt.Errorf("%w: %v", ErrAIInvalidResponse, err)
	}

	result.RawResponse = text

	slog.Info("Claude parsed filename",
		"filename", req.Filename,
		"title", result.Title,
		"media_type", result.MediaType,
		"confidence", result.Confidence,
	)

	return result, nil
}

// CompleteText sends a system+user prompt pair to Claude and returns the raw text response.
// Unlike Parse, this does not expect JSON output — it returns the text as-is.
// The caller controls the timeout via the provided context.
func (p *ClaudeProvider) CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	if maxTokens <= 0 {
		maxTokens = ClaudeMaxTokens
	}

	claudeReq := claudeRequest{
		Model:     p.model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages: []claudeMessage{
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}

	body, err := json.Marshal(claudeReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	slog.Debug("Claude CompleteText request", "model", p.model)

	// Execute with bounded transient retry (9R-4)
	respBody, err := p.doRequest(ctx, body)
	if err != nil {
		return "", err
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return "", fmt.Errorf("%w: %v", ErrAIInvalidResponse, err)
	}

	text := claudeResp.GetText()
	if text == "" {
		return "", ErrAIInvalidResponse
	}

	return text, nil
}

// Claude API types

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system,omitempty"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content    []claudeContentBlock `json:"content"`
	StopReason string               `json:"stop_reason"`
	Usage      claudeUsage          `json:"usage"`
}

// claudeUsage carries the token counts the Messages API returns (9R-11 metering).
type claudeUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

type claudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// GetText extracts the text from the first content block.
func (r *claudeResponse) GetText() string {
	for _, block := range r.Content {
		if block.Type == "text" {
			return block.Text
		}
	}
	return ""
}
