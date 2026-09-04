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
	//
	// Keep this pointing at a model that still exists. The previous value,
	// gemini-2.0-flash, was shut down by Google on 2026-06-01 and stayed the
	// default for ~3 months, so every Gemini deployment 404'd on every call with
	// no way to route around it. gemini-2.5-flash-lite was verified present on
	// ai.google.dev/gemini-api/docs/pricing on 2026-08-24 and carries the same
	// $0.10/$0.40 per-1M rate the retired model had, so the bump is cost-neutral.
	// Override per deployment with GEMINI_MODEL (mirrors CLAUDE_MODEL, story 9R-1).
	DefaultGeminiModel = "gemini-2.5-flash-lite"
)

// GeminiProvider implements the Provider interface for Google's Gemini AI.
type GeminiProvider struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	// timeout is a FIXED per-attempt deadline when set (WithGeminiTimeout).
	// Zero, the default, derives each attempt's deadline from
	// RequestTimeoutFor(model, max_output_tokens) (sub-6-2 AC #1, same rule as
	// claude.go so the two providers cannot drift).
	timeout  time.Duration
	governor *Governor // sub-5-1: shared throttle (nil = unthrottled)
}

// Compile-time interface verification.
var _ Provider = (*GeminiProvider)(nil)

var _ Pinger = (*GeminiProvider)(nil)

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

// WithGeminiTimeout pins one fixed per-attempt timeout, replacing the
// per-call RequestTimeoutFor derivation. Zero restores the derivation.
func WithGeminiTimeout(timeout time.Duration) GeminiProviderOption {
	return func(p *GeminiProvider) {
		p.timeout = timeout
	}
}

// WithGeminiGovernor injects the shared AI throttle (sub-5-1 AC #2, mirroring
// WithWhisperGovernor/WithClaudeGovernor) so Gemini calls draw from the same
// concurrency/QPS pool — and hit the same budget pre-check — as every other
// provider.
func WithGeminiGovernor(g *Governor) GeminiProviderOption {
	return func(p *GeminiProvider) {
		p.governor = g
	}
}

// NewGeminiProvider creates a new Gemini AI provider.
func NewGeminiProvider(apiKey string, opts ...GeminiProviderOption) *GeminiProvider {
	p := &GeminiProvider{
		apiKey:  apiKey,
		baseURL: DefaultGeminiBaseURL,
		model:   DefaultGeminiModel,
	}

	for _, opt := range opts {
		opt(p)
	}

	// No client-level Timeout: the per-attempt ctx in Parse/Ping is the one
	// deadline (sub-6-2 AC #1) — a second one on the client would silently win
	// whenever it was the shorter, which is how the old 15 s constant kept
	// biting after the ctx existed.
	if p.httpClient == nil {
		p.httpClient = &http.Client{}
	}

	return p
}

// attemptTimeout is the deadline one attempt gets: the pinned WithGeminiTimeout
// when set, otherwise RequestTimeoutFor(model, maxOutputTokens).
func (p *GeminiProvider) attemptTimeout(maxOutputTokens int) time.Duration {
	if p.timeout > 0 {
		return p.timeout
	}
	return RequestTimeoutFor(p.model, maxOutputTokens)
}

// Name returns the provider name.
func (p *GeminiProvider) Name() ProviderName {
	return ProviderGemini
}

// Ping implements Pinger for Gemini (sub-6-6 AC #2 — 同級化 with claude.go):
// one generateContent call with max_output_tokens=1 under governed+retry,
// judged on the HTTP verdict only. The key travels in the x-goog-api-key
// header, never the URL, so a wrapped *url.Error cannot leak it into the
// key-test handler's log line (Rule 27 ⑤ / NFR-S1; CR H7 — Parse still uses
// the query form and is untouched here).
//
// Status mapping: 401/403 → ErrAIUnauthorized; 400 → ErrAIUnauthorized ONLY
// when Google's error body says so (`API_KEY_INVALID` reason or "API key not
// valid" message) — a bare 400 is INVALID_ARGUMENT for many other reasons
// (malformed body, an unsupported field, a thinking-model token floor) and
// is reported as a provider error instead of「金鑰無效」(CR H3); 404 →
// ErrAIModelNotFound; 429 → ErrAIQuotaExceeded; timeout → ErrAITimeout.
//
// Not wired to the key-test endpoint yet: `TestKeyRequest` has only a
// `claude` field and the holder is Claude-only — the Gemini probe is reachable
// once a Gemini key row exists on the settings page (sub-7-8 / models
// endpoint). It exists now so the two providers cannot drift.
func (p *GeminiProvider) Ping(ctx context.Context) error {
	body, err := json.Marshal(geminiRequest{
		Contents:         []geminiContent{{Parts: []geminiPart{{Text: "hi"}}}},
		GenerationConfig: geminiGenerationConfig{MaxOutputTokens: 1},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	url := fmt.Sprintf("%s/models/%s:generateContent", p.baseURL, p.model)
	timeout := p.attemptTimeout(1)

	_, err = governed(ctx, p.governor, "gemini.ping", func() (struct{}, error) {
		return retryTransient(ctx, "gemini.ping", func() (struct{}, bool, error) {
			attemptCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			httpReq, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, url, bytes.NewReader(body))
			if err != nil {
				return struct{}{}, false, fmt.Errorf("failed to create request: %w", err)
			}
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("x-goog-api-key", p.apiKey)
			resp, err := p.httpClient.Do(httpReq)
			if err != nil {
				if attemptCtx.Err() == context.DeadlineExceeded {
					return struct{}{}, true, ErrAITimeout
				}
				return struct{}{}, true, fmt.Errorf("%w: %v", ErrAIProviderError, err)
			}
			defer resp.Body.Close()
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
			switch {
			case resp.StatusCode >= 200 && resp.StatusCode < 300:
				return struct{}{}, false, nil
			case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
				return struct{}{}, false, fmt.Errorf("%w: %w: status %d", ErrAIProviderError, ErrAIUnauthorized, resp.StatusCode)
			case resp.StatusCode == http.StatusBadRequest && geminiBodySaysBadKey(respBody):
				return struct{}{}, false, fmt.Errorf("%w: %w: status 400 API_KEY_INVALID", ErrAIProviderError, ErrAIUnauthorized)
			case resp.StatusCode == http.StatusNotFound:
				return struct{}{}, false, fmt.Errorf("%w: %w: status 404: model %q not found", ErrAIProviderError, ErrAIModelNotFound, p.model)
			case resp.StatusCode == http.StatusTooManyRequests:
				return struct{}{}, true, ErrAIQuotaExceeded
			default:
				return struct{}{}, isTransientStatus(resp.StatusCode), fmt.Errorf("%w: status %d", ErrAIProviderError, resp.StatusCode)
			}
		})
	})
	return err
}

// geminiBodySaysBadKey reports whether a 400 body is Google's bad-key shape.
func geminiBodySaysBadKey(body []byte) bool {
	return bytes.Contains(body, []byte("API_KEY_INVALID")) || bytes.Contains(body, []byte("API key not valid"))
}

// Parse sends a filename to Gemini for parsing.
//
// sub-5-1 AC #2: the call runs under the SAME resilience/metering stack as
// claude.go's send — governed() (budget pre-check + rate token + concurrency
// slot, ONCE) wrapping retryTransient (3 attempts, exponential backoff), with
// token usage metered against a ctx-attached Budget afterward. The nesting
// order is load-bearing (D8): retries must sit INSIDE the budget gate.
func (p *GeminiProvider) Parse(ctx context.Context, req *ParseRequest) (*ParseResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

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

	slog.Debug("Gemini API request",
		"model", p.model,
		"filename", req.Filename,
	)
	// A parse sets no max_output_tokens, so the family base alone applies.
	timeout := p.attemptTimeout(0)

	geminiResp, err := governed(ctx, p.governor, "gemini.generate", func() (*geminiResponse, error) {
		return retryTransient(ctx, "gemini.generate", func() (*geminiResponse, bool, error) {
			// Per-attempt timeout (the whisper.go shape) — a timeout on attempt
			// 1 must not consume the deadline of attempts 2 and 3. NOTE (CR
			// M3): the bound is therefore per-ATTEMPT; the worst-case logical
			// call is 3×timeout + backoff, matching claude.go — which also
			// retries 429/timeout/5xx with no outer deadline. That parity IS
			// the AC (同級化); pinned by the 429-attempt-count test.
			attemptCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			httpReq, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, url, bytes.NewReader(body))
			if err != nil {
				return nil, false, fmt.Errorf("failed to create request: %w", err)
			}
			httpReq.Header.Set("Content-Type", "application/json")

			resp, err := p.httpClient.Do(httpReq)
			if err != nil {
				if attemptCtx.Err() == context.DeadlineExceeded {
					slog.Warn("Gemini API timeout",
						"model", p.model,
						"filename", req.Filename,
						"timeout_seconds", timeout.Seconds(),
					)
					return nil, true, ErrAITimeout
				}
				slog.Error("Gemini API request failed",
					"error", err,
					"filename", req.Filename,
				)
				return nil, true, fmt.Errorf("%w: %v", ErrAIProviderError, err)
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, true, fmt.Errorf("failed to read response: %w", err)
			}

			if resp.StatusCode != http.StatusOK {
				slog.Warn("Gemini API error response",
					"status_code", resp.StatusCode,
					"body", string(respBody),
				)
				if resp.StatusCode == http.StatusTooManyRequests {
					return nil, true, ErrAIQuotaExceeded
				}
				return nil, isTransientStatus(resp.StatusCode), fmt.Errorf("%w: status %d", ErrAIProviderError, resp.StatusCode)
			}

			// A malformed 200 body is PERMANENT (the claude.go classifyErr
			// rationale): retrying garbage would turn one broken response into
			// three real requests.
			var parsed geminiResponse
			if err := json.Unmarshal(respBody, &parsed); err != nil {
				slog.Error("Failed to parse Gemini response",
					"error", err,
					"body", string(respBody),
				)
				return nil, false, fmt.Errorf("%w: %v", ErrAIInvalidResponse, err)
			}
			return &parsed, false, nil
		})
	})
	if err != nil {
		return nil, err
	}

	// sub-5-1 AC #2: meter token usage against the per-run budget (claude.go
	// send precedent). Without a ctx Budget this is a no-op — the parse paths
	// are ruled unmetered-by-design (AC #4, backlog-parse-path-ai-metering).
	if b := BudgetFromContext(ctx); b != nil {
		usage := geminiResp.UsageMetadata
		// CR M4: a success with zero-value usage means the upstream omitted
		// usageMetadata (proxy, API-shape drift). Metering $0 silently would
		// disable the budget ceiling while real money is spent — the
		// under-count twin of the fabricated-fallback-number AC #2 bans. Warn
		// loudly; the call is still recorded so LLMCalls stays honest.
		if usage.PromptTokenCount == 0 && usage.CandidatesTokenCount == 0 {
			slog.Warn("Gemini response carried no usageMetadata — metering $0 for this call; the budget ceiling cannot see its real cost",
				"model", p.model, "filename", req.Filename)
		}
		b.RecordLLM(p.model, usage.PromptTokenCount, usage.CandidatesTokenCount)
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
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generation_config,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	ResponseMimeType string `json:"response_mime_type,omitempty"`
	// MaxOutputTokens bounds the reply; Ping sets 1 so the probe costs nothing.
	MaxOutputTokens int `json:"max_output_tokens,omitempty"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
	// UsageMetadata carries the API's own token accounting (sub-5-1 AC #2) —
	// what RecordLLM meters. Absent on older mocks it zero-values, recording a
	// $0 call rather than failing the parse.
	UsageMetadata geminiUsageMetadata `json:"usageMetadata"`
}

// geminiUsageMetadata mirrors the generateContent usageMetadata block.
type geminiUsageMetadata struct {
	PromptTokenCount     int64 `json:"promptTokenCount"`
	CandidatesTokenCount int64 `json:"candidatesTokenCount"`
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
