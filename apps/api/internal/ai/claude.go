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
	// DefaultClaudeModel is the default model to use.
	DefaultClaudeModel = "claude-3-5-haiku-latest"
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

// Parse sends a filename to Claude for parsing.
func (p *ClaudeProvider) Parse(ctx context.Context, req *ParseRequest) (*ParseResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Create context with timeout per NFR-I12
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

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

	// Build URL
	url := fmt.Sprintf("%s/messages", p.baseURL)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers per Claude API spec
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", ClaudeAPIVersion)

	slog.Debug("Claude API request",
		"model", p.model,
		"filename", req.Filename,
	)

	// Execute request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			slog.Warn("Claude API timeout",
				"filename", req.Filename,
				"timeout_seconds", p.timeout.Seconds(),
			)
			return nil, ErrAITimeout
		}
		slog.Error("Claude API request failed",
			"error", err,
			"filename", req.Filename,
		)
		return nil, fmt.Errorf("%w: %v", ErrAIProviderError, err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		slog.Warn("Claude API error response",
			"status_code", resp.StatusCode,
			"body", string(respBody),
		)
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, ErrAIQuotaExceeded
		}
		return nil, fmt.Errorf("%w: status %d", ErrAIProviderError, resp.StatusCode)
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

// Claude API types

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []claudeContentBlock `json:"content"`
	StopReason string           `json:"stop_reason"`
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
