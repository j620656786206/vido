package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const (
	// DefaultClaudeBaseURL is the ORIGIN of the Claude API — deliberately WITHOUT
	// the "/v1" suffix. The official SDK appends its own version path segment
	// ("v1/messages"), so a base URL ending in "/v1" would produce
	// ".../v1/v1/messages" and 404 in production (story sub-1-1 AC #6).
	// Every existing test asserts only Contains(path, "/messages"), which passes
	// for the broken form too — TestClaudeProvider_RequestPathIsV1Messages is the
	// guard. Do NOT "restore" the /v1 suffix.
	DefaultClaudeBaseURL = "https://api.anthropic.com"
	// DefaultClaudeModel is the default model to use. Must be a current,
	// non-deprecated alias: the previous default "claude-3-5-haiku-latest"
	// (Haiku 3.5) was retired 2026-02-19 and returns 404 (9R-1).
	// Override per-deployment via CLAUDE_MODEL.
	DefaultClaudeModel = "claude-haiku-4-5"
	// ClaudeAPIVersion is the required API version header. The SDK now sets this
	// header itself (it pins the same value); the constant is retained because it
	// is the documented expected value and the test suite asserts against it.
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
	// timeout is a FIXED per-attempt deadline when set (WithClaudeTimeout —
	// tests, and a deployment that wants one number). Zero, the default, means
	// every attempt derives its own deadline from RequestTimeoutFor(model,
	// max_tokens) (sub-6-2 AC #1).
	timeout  time.Duration
	governor *Governor // 9R-11: shared throttle (nil = unthrottled)

	// client is the official anthropic-sdk-go client, built ONCE in
	// NewClaudeProvider after all options are applied (Rule 14 — never per
	// request). anthropic.NewClient returns a value, not a pointer.
	client anthropic.Client
}

// Compile-time interface verification.
var (
	_ Provider         = (*ClaudeProvider)(nil)
	_ TextCompleter    = (*ClaudeProvider)(nil)
	_ CachingCompleter = (*ClaudeProvider)(nil)
	_ Pinger           = (*ClaudeProvider)(nil)
)

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

// WithClaudeTimeout pins one fixed per-attempt timeout, replacing the
// per-call RequestTimeoutFor derivation. Zero restores the derivation.
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
// Governor returns the shared AI throttle this provider was built with (nil when
// unthrottled). Exposed so a caller that REBUILDS the provider — the M1.5
// key-hot-reload holder (sub-2-1a AC #2) — can assert the same Governor instance
// carried over: a new client must never mean a new budget pool, or editing a key
// would silently reset the 9R-11 run-budget ceiling.
func (c *ClaudeProvider) Governor() *Governor { return c.governor }

// Model returns the model id this provider actually sends — the WithClaudeModel
// override when one was given, otherwise DefaultClaudeModel. It is the single
// truth every run row and cache key should record (sub-6-5): reading the env
// override instead left `subtitle_runs.model_id` empty on every default-model
// run, so a later change of the default would have matched the previous
// model's cache entries.
func (c *ClaudeProvider) Model() string { return c.model }

func NewClaudeProvider(apiKey string, opts ...ClaudeProviderOption) *ClaudeProvider {
	p := &ClaudeProvider{
		apiKey:  apiKey,
		baseURL: DefaultClaudeBaseURL,
		model:   DefaultClaudeModel,
	}

	for _, opt := range opts {
		opt(p)
	}

	// The deadline is the per-attempt ctx in send, and ONLY that (sub-6-2
	// AC #1): the default client carries no Timeout and this code stacks no
	// option.WithRequestTimeout on it. Two competing deadlines was the D8 bug
	// class this replaces — the shorter one always won silently, so a 60 s
	// derivation would have been cut to the client's 15 s. Before the ctx
	// deadline existed a caller-supplied client without a Timeout of its own
	// needed WithRequestTimeout to keep the bound (CR M1); the ctx covers
	// both clients now, and TestClaudeProvider_TimeoutEnforcedWithCustomHTTPClient
	// still guards it.
	//
	// One deadline the SDK adds ITSELF: Messages.New appends a 10-minute
	// non-streaming RequestTimeout, applied only when it would fire before
	// the ctx deadline. MaxRequestTimeout (180 s) guarantees it never does —
	// see the note there. A pinned WithClaudeTimeout over 10 minutes WOULD be
	// silently cut to 10; do not set one.
	if p.httpClient == nil {
		p.httpClient = &http.Client{}
	}

	// Built AFTER options are applied — otherwise WithClaudeBaseURL is silently
	// ignored and every test would hit the real API.
	clientOpts := []option.RequestOption{
		option.WithAPIKey(p.apiKey),
		option.WithBaseURL(p.baseURL),
		// D8 / NAIL 2: the SDK retries by default (max 2). retryTransient already
		// wraps every call INSIDE the Governor's budget pre-check, so SDK retries
		// would sit BELOW the budget gate and a retry storm would bypass cost
		// control entirely. Exactly one retry layer, and it is ours.
		option.WithMaxRetries(0),
		option.WithHTTPClient(p.httpClient),
		// A 2xx body that is not JSON is permanent, exactly like a malformed
		// JSON body (AC #4 trap) — without this guard the SDK's content-type
		// rejection falls into the connection-error fallback and gets retried.
		option.WithMiddleware(rejectNonJSONSuccess),
	}
	p.client = anthropic.NewClient(clientOpts...)

	return p
}

// errNonJSONResponse marks a 2xx response whose Content-Type is not JSON.
// Permanent for the same reason a malformed JSON body is: retrying a broken
// upstream that "succeeds" with garbage would silently multiply cost.
var errNonJSONResponse = errors.New("non-JSON content-type on a success response")

// rejectNonJSONSuccess is client middleware that fails fast on 2xx responses
// not declaring application/json. Error responses (non-2xx) pass through
// untouched so the *anthropic.Error path keeps its status-based classification.
func rejectNonJSONSuccess(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
	resp, err := next(req)
	if err != nil || resp == nil {
		return resp, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		ct := resp.Header.Get("Content-Type")
		if mediaType, _, parseErr := mime.ParseMediaType(ct); parseErr != nil || mediaType != "application/json" {
			resp.Body.Close()
			return nil, fmt.Errorf("%w: %q", errNonJSONResponse, ct)
		}
	}
	return resp, nil
}

// Name returns the provider name.
func (p *ClaudeProvider) Name() ProviderName {
	return ProviderClaude
}

// isTimeoutErr reports whether err represents a request timeout, covering both
// http.Client.Timeout (a net.Error with Timeout() == true) and a caller-supplied
// context deadline.
func isTimeoutErr(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// isDecodeErr reports whether err came from decoding a malformed response body.
// The SDK decodes with encoding/json (requestconfig.go: json.NewDecoder(...).Decode),
// so a garbage 200 body surfaces as *json.SyntaxError / *json.UnmarshalTypeError.
func isDecodeErr(err error) bool {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	return errors.As(err, &syntaxErr) || errors.As(err, &typeErr)
}

// attemptTimeout is the deadline one attempt gets: the pinned WithClaudeTimeout
// when set, otherwise RequestTimeoutFor(model, max_tokens).
func (p *ClaudeProvider) attemptTimeout(maxTokens int64) time.Duration {
	if p.timeout > 0 {
		return p.timeout
	}
	return RequestTimeoutFor(p.model, int(maxTokens))
}

// probeTimeout is Ping's per-attempt deadline: the pinned WithClaudeTimeout
// when set, otherwise the fixed ProbeRequestTimeout.
func (p *ClaudeProvider) probeTimeout() time.Duration {
	if p.timeout > 0 {
		return p.timeout
	}
	return ProbeRequestTimeout
}

// classifyErr maps an SDK error onto this package's sentinels and reports
// whether the failure is worth retrying. SDK error types must never leak past
// this function. timeout is the deadline the failed attempt ran under — logged
// with the model so a timeout line says what bound it actually hit (AC #4).
func (p *ClaudeProvider) classifyErr(err error, timeout time.Duration) (retryable bool, mapped error) {
	// A malformed 200 body. NOTE: with the SDK the decode happens INSIDE
	// Messages.New, i.e. inside the retry loop — the hand-rolled client decoded
	// outside it. Classifying this as retryable would turn one garbage response
	// into three real requests, so it is deliberately permanent.
	if isDecodeErr(err) || errors.Is(err, errNonJSONResponse) {
		return false, fmt.Errorf("%w: %v", ErrAIInvalidResponse, err)
	}

	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		slog.Warn("Claude API error response",
			"status_code", apiErr.StatusCode,
			"body", apiErr.Error(),
			// The unmodified API error body — Error() is the SDK's formatted
			// summary; the raw JSON is what the hand-rolled client used to log.
			"raw_json", apiErr.RawJSON(),
		)
		switch apiErr.StatusCode {
		case http.StatusTooManyRequests:
			return true, ErrAIQuotaExceeded
		case http.StatusUnauthorized, http.StatusForbidden:
			// Not transient: a rejected key will be rejected again. Wrapping both
			// sentinels keeps existing ErrAIProviderError consumers unchanged
			// while letting the key-test endpoint identify a bad key exactly.
			return false, fmt.Errorf("%w: %w: status %d", ErrAIProviderError, ErrAIUnauthorized, apiErr.StatusCode)
		case http.StatusNotFound:
			// 9R-1: a deprecated/invalid model id returns 404. This diagnostic was
			// hard-won during that incident — keep it verbatim.
			slog.Error("Claude model not found — the configured model id is deprecated or invalid",
				"model", p.model,
			)
			return false, fmt.Errorf("%w: %w: status 404: model %q not found (deprecated or invalid model id — set CLAUDE_MODEL to a current model)", ErrAIProviderError, ErrAIModelNotFound, p.model)
		}
		return isTransientStatus(apiErr.StatusCode), fmt.Errorf("%w: status %d", ErrAIProviderError, apiErr.StatusCode)
	}

	if isTimeoutErr(err) {
		slog.Warn("Claude API timeout", "model", p.model, "timeout_seconds", timeout.Seconds())
		return true, ErrAITimeout
	}

	// Connection-level failure.
	return true, fmt.Errorf("%w: %v", ErrAIProviderError, err)
}

// send issues one logical Messages request with bounded retry on transient
// failures (9R-4) under the shared budget/throttle gate (9R-11).
//
// The nesting is load-bearing (D8):
//
//	governed(...)              budget pre-check + rate token + concurrency slot, ONCE
//	  └── retryTransient(...)  3 attempts, exponential backoff
//	        └── Messages.New   SDK, retries DISABLED, one fresh deadline per attempt
//
// The deadline is per ATTEMPT (the gemini.go / whisper.go shape): a timeout on
// attempt 1 must not eat the window of attempts 2 and 3. It is derived from
// this request's max_tokens, so a 10-cue Sonnet chunk waits ~100 s while a
// Ping waits 60 s (sub-6-2 AC #1).
func (p *ClaudeProvider) send(ctx context.Context, params anthropic.MessageNewParams) (*anthropic.Message, error) {
	return p.sendWithin(ctx, params, p.attemptTimeout(params.MaxTokens))
}

// sendWithin is send with an explicit per-attempt deadline — the one seam
// that lets Ping run the same governed/retry/classify path under its own
// probe bound.
func (p *ClaudeProvider) sendWithin(ctx context.Context, params anthropic.MessageNewParams, timeout time.Duration) (*anthropic.Message, error) {
	msg, err := governed(ctx, p.governor, "claude.messages", func() (*anthropic.Message, error) {
		return retryTransient(ctx, "claude.messages", func() (*anthropic.Message, bool, error) {
			attemptCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			m, err := p.client.Messages.New(attemptCtx, params)
			if err != nil {
				retryable, mapped := p.classifyErr(err, timeout)
				return nil, retryable, mapped
			}
			return m, false, nil
		})
	})
	if err != nil {
		return nil, err
	}

	// 9R-11: meter token usage against the per-run budget.
	if b := BudgetFromContext(ctx); b != nil {
		b.RecordLLM(p.model, msg.Usage.InputTokens, msg.Usage.OutputTokens)
	}
	return msg, nil
}

// Ping implements Pinger: one max_tokens=1 Messages call through the same
// governed/retry/classify path as every real request — so it is rate-limited
// and, if the ctx carried a Budget, metered and budget-gated like any call.
// The key-test handler's request ctx carries no Budget, so in practice the
// probe is neither. The VERDICT is transport-only: 401/403 →
// ErrAIUnauthorized, 404 → ErrAIModelNotFound, timeouts and 5xx → their
// sentinels; a 2xx with an empty content array is a PASS. The deadline is
// ProbeRequestTimeout, not the family derivation (sub-6-2 CR M6).
func (p *ClaudeProvider) Ping(ctx context.Context) error {
	_, err := p.sendWithin(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 1,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("hi")),
		},
	}, p.probeTimeout())
	return err
}

// textFromMessage returns the text of the first text content block, or "".
func textFromMessage(msg *anthropic.Message) string {
	if msg == nil {
		return ""
	}
	for _, block := range msg.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			return t.Text
		}
	}
	return ""
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

	slog.Debug("Claude API request",
		"model", p.model,
		"filename", req.Filename,
	)

	msg, err := p.send(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: int64(ClaudeMaxTokens),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, err
	}

	text := textFromMessage(msg)
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
	req := CompletionRequest{UserPrompt: userPrompt, MaxTokens: maxTokens}
	if systemPrompt != "" {
		req.System = []SystemBlock{{Text: systemPrompt}}
	}

	res, err := p.CompleteTextWithUsage(ctx, req)
	if err != nil {
		return "", err
	}
	return res.Text, nil
}

// CompleteTextWithUsage is the caching-aware completion entry point: it accepts
// ordered system blocks (each optionally a cache_control breakpoint) and returns
// token usage alongside the text.
//
// [@contract-v1] — see CachingCompleter in provider.go. Consumer: sub-1-5a.
func (p *ClaudeProvider) CompleteTextWithUsage(ctx context.Context, req CompletionRequest) (CompletionResult, error) {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = ClaudeMaxTokens
	}

	// Logged here, not in CompleteText — both entry points funnel through this
	// method, so this is the one place that observes every completion request.
	slog.Debug("Claude completion request",
		"model", p.model,
		"system_blocks", len(req.System),
		"max_tokens", maxTokens,
	)

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: int64(maxTokens),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(req.UserPrompt)),
		},
	}
	// Order is semantic, not cosmetic: prompt caching is a prefix match, so the
	// caller's slice order is preserved exactly. An empty System slice leaves the
	// field unset, so no "system" key is emitted at all.
	if len(req.System) > 0 {
		blocks := make([]anthropic.TextBlockParam, 0, len(req.System))
		for _, b := range req.System {
			block := anthropic.TextBlockParam{Text: b.Text}
			switch b.CacheTTL {
			case CacheTTLNone:
				// not a cache breakpoint
			case CacheTTL5m:
				block.CacheControl = anthropic.NewCacheControlEphemeralParam()
			case CacheTTL1h:
				block.CacheControl = anthropic.CacheControlEphemeralParam{
					TTL: anthropic.CacheControlEphemeralTTLTTL1h,
				}
			default:
				// An unrecognized TTL silently emitting no cache_control is the
				// exact "silently-inert cache" failure mode this API exists to
				// surface (AC #5) — be loud about it.
				slog.Warn("unknown CacheTTL — no cache_control emitted",
					"cache_ttl", string(b.CacheTTL),
				)
			}
			blocks = append(blocks, block)
		}
		params.System = blocks
	}

	msg, err := p.send(ctx, params)
	if err != nil {
		return CompletionResult{}, err
	}

	text := textFromMessage(msg)
	if text == "" {
		return CompletionResult{}, ErrAIInvalidResponse
	}

	return CompletionResult{
		Text: text,
		Usage: CompletionUsage{
			InputTokens:              msg.Usage.InputTokens,
			OutputTokens:             msg.Usage.OutputTokens,
			CacheCreationInputTokens: msg.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     msg.Usage.CacheReadInputTokens,
		},
	}, nil
}
