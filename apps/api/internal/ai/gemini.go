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
	// DefaultGeminiBaseURL is the base URL for Gemini API.
	DefaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"
	// DefaultGeminiModel is the default model to use.
	DefaultGeminiModel = "gemini-2.0-flash"
	// DefaultTimeoutSeconds is the default timeout per NFR-I12.
	DefaultTimeoutSeconds = 15
)

// GeminiProvider implements the Provider interface for Google's Gemini AI.
type GeminiProvider struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	timeout    time.Duration
}

// Compile-time interface verification.
var _ Provider = (*GeminiProvider)(nil)

// GeminiProviderOption is a functional option for configuring GeminiProvider.
type GeminiProviderOption func(*GeminiProvider)

// WithGeminiBaseURL sets a custom base URL.
func WithGeminiBaseURL(url string) GeminiProviderOption {
	return func(p *GeminiProvider) {
		p.baseURL = url
	}
}

// WithGeminiModel sets a custom model.
func WithGeminiModel(model string) GeminiProviderOption {
	return func(p *GeminiProvider) {
		p.model = model
	}
}

// WithGeminiHTTPClient sets a custom HTTP client (useful for testing).
func WithGeminiHTTPClient(client *http.Client) GeminiProviderOption {
	return func(p *GeminiProvider) {
		p.httpClient = client
	}
}

// WithGeminiTimeout sets a custom timeout.
func WithGeminiTimeout(timeout time.Duration) GeminiProviderOption {
	return func(p *GeminiProvider) {
		p.timeout = timeout
	}
}

// NewGeminiProvider creates a new Gemini AI provider.
func NewGeminiProvider(apiKey string, opts ...GeminiProviderOption) *GeminiProvider {
	p := &GeminiProvider{
		apiKey:  apiKey,
		baseURL: DefaultGeminiBaseURL,
		model:   DefaultGeminiModel,
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
func (p *GeminiProvider) Name() ProviderName {
	return ProviderGemini
}

// Parse sends a filename to Gemini for parsing.
func (p *GeminiProvider) Parse(ctx context.Context, req *ParseRequest) (*ParseResponse, error) {
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
	geminiReq := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
		},
	}

	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, p.model, p.apiKey)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	slog.Debug("Gemini API request",
		"model", p.model,
		"filename", req.Filename,
	)

	// Execute request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			slog.Warn("Gemini API timeout",
				"filename", req.Filename,
				"timeout_seconds", p.timeout.Seconds(),
			)
			return nil, ErrAITimeout
		}
		slog.Error("Gemini API request failed",
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
		slog.Warn("Gemini API error response",
			"status_code", resp.StatusCode,
			"body", string(respBody),
		)
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, ErrAIQuotaExceeded
		}
		return nil, fmt.Errorf("%w: status %d", ErrAIProviderError, resp.StatusCode)
	}

	// Parse Gemini response
	var geminiResp geminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		slog.Error("Failed to parse Gemini response",
			"error", err,
			"body", string(respBody),
		)
		return nil, fmt.Errorf("%w: %v", ErrAIInvalidResponse, err)
	}

	// Extract text from response
	text := geminiResp.GetText()
	if text == "" {
		slog.Warn("Empty response from Gemini",
			"filename", req.Filename,
		)
		return nil, ErrAIInvalidResponse
	}

	// Parse the JSON response into ParseResponse
	result, err := parseJSONResponse(text)
	if err != nil {
		slog.Error("Failed to parse JSON from Gemini response",
			"error", err,
			"raw_response", text,
		)
		return nil, fmt.Errorf("%w: %v", ErrAIInvalidResponse, err)
	}

	result.RawResponse = text

	slog.Info("Gemini parsed filename",
		"filename", req.Filename,
		"title", result.Title,
		"media_type", result.MediaType,
		"confidence", result.Confidence,
	)

	return result, nil
}

// Gemini API types

type geminiRequest struct {
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	ResponseMimeType string `json:"responseMimeType,omitempty"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

// GetText extracts the text from the first candidate.
func (r *geminiResponse) GetText() string {
	if len(r.Candidates) == 0 {
		return ""
	}
	if len(r.Candidates[0].Content.Parts) == 0 {
		return ""
	}
	return r.Candidates[0].Content.Parts[0].Text
}

// parseJSONResponse parses the AI's JSON response into ParseResponse.
func parseJSONResponse(text string) (*ParseResponse, error) {
	var result ParseResponse
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, err
	}

	// Validate required fields
	if result.Title == "" {
		return nil, fmt.Errorf("missing required field: title")
	}
	if result.MediaType == "" {
		return nil, fmt.Errorf("missing required field: media_type")
	}
	if result.MediaType != "movie" && result.MediaType != "tv" {
		return nil, fmt.Errorf("invalid media_type: %s", result.MediaType)
	}

	return &result, nil
}
