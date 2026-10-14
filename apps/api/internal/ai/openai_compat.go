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
	"sync"
	"time"
)

// OpenAICompatProvider is a TextCompleter for any server that speaks the
// OpenAI chat-completions shape — Ollama, llama.cpp's llama-server, vLLM,
// LM Studio, and Google's OpenAI-compatible Gemini endpoint.
// spike-local-translation-golden-eval uses it so cmd/grade can score a model
// running on the NAS; gemini-translation-dispatch uses it as the app's Gemini
// translation client (GeminiProviderHolder).
//
// Deliberate differences from the native cloud providers:
//   - The timeout is the caller's, not RequestTimeoutFor's. That table is
//     sized for cloud latency and capped at MaxRequestTimeout (180 s); a 7–12B
//     model on a CPU-only NAS can need several minutes for one batch.
//   - Billing is OPT-IN (WithOpenAICompatBilling): local inference costs $0
//     and a price lookup for an uncatalogued model id would be meaningless; a
//     hosted endpoint turns it on and is metered at defaultLLMPricing.
//   - Pacing is available (WithOpenAICompatMinInterval): free tiers are
//     limited per minute, and the package's retry waits only 1–2 s.
type OpenAICompatProvider struct {
	baseURL string
	model   string
	apiKey  string
	timeout time.Duration
	client  *http.Client
	// governor is the shared AI throttle (nil = unthrottled, the grade-tool
	// default).
	governor *Governor
	// billing: record each reply's `usage` on the run budget at the model's
	// catalog price.
	billing bool
	// minInterval spaces consecutive calls at least this far apart — a free
	// tier's requests-per-minute limit (Gemini free: 5 RPM, 2026-10-14).
	// Zero = no pacing.
	minInterval time.Duration
	paceMu      sync.Mutex
	paceNext    time.Time
}

// GeminiOpenAIBaseURL is Google's OpenAI-compatible endpoint for Gemini
// (https://ai.google.dev/gemini-api/docs/openai, verified 2026-10-14).
const GeminiOpenAIBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"

// DefaultOpenAICompatTimeout is the per-attempt deadline when the caller sets
// none: generous on purpose, see the type comment.
const DefaultOpenAICompatTimeout = 10 * time.Minute

// openAICompatErrBodyLimit caps how much of an error body is quoted back.
const openAICompatErrBodyLimit = 512

// Compile-time proof: the translation stack reads usage through this seam.
var _ CachingCompleter = (*OpenAICompatProvider)(nil)

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

// WithOpenAICompatGovernor injects the shared AI throttle (budget pre-check,
// concurrency, QPS), the same one every hosted provider runs under.
func WithOpenAICompatGovernor(g *Governor) OpenAICompatOption {
	return func(p *OpenAICompatProvider) { p.governor = g }
}

// WithOpenAICompatBilling records each reply's token usage on the run budget
// at the model's catalog price (gemini-translation-dispatch). Use it for a
// HOSTED OpenAI-compatible API; never for a local engine.
func WithOpenAICompatBilling(on bool) OpenAICompatOption {
	return func(p *OpenAICompatProvider) { p.billing = on }
}

// WithOpenAICompatMinInterval spaces calls at least d apart (free-tier RPM
// limits). Retries inside one call are not paced — they are already backed
// off — so pick d with a little headroom (13 s for a 5 RPM tier).
func WithOpenAICompatMinInterval(d time.Duration) OpenAICompatOption {
	return func(p *OpenAICompatProvider) {
		if d > 0 {
			p.minInterval = d
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

// MinInterval reports the pacing gap this provider keeps between calls.
func (p *OpenAICompatProvider) MinInterval() time.Duration { return p.minInterval }

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
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
	} `json:"usage"`
}

// openAICompatReply is one successful attempt's text and usage.
type openAICompatReply struct {
	text  string
	usage CompletionUsage
}

// CompleteText sends one chat completion: an optional system message, then
// the user message. Transient failures (network, timeout, 429, 5xx) retry with
// the package's bounded backoff; other 4xx fail at once with the server's own
// error text, so "model not found" reads as such instead of as a retry storm.
func (p *OpenAICompatProvider) CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	res, err := p.CompleteTextWithUsage(ctx, CompletionRequest{
		System:     []SystemBlock{{Text: systemPrompt}},
		UserPrompt: userPrompt,
		MaxTokens:  maxTokens,
	})
	if err != nil {
		return "", err
	}
	return res.Text, nil
}

// CompleteTextWithUsage is CompleteText plus the reply's token usage. The
// system blocks are flattened in order (this wire format has no cache
// control); the cache dimensions are always zero.
func (p *OpenAICompatProvider) CompleteTextWithUsage(ctx context.Context, req CompletionRequest) (CompletionResult, error) {
	if p.baseURL == "" || p.model == "" {
		return CompletionResult{}, fmt.Errorf("%w: openai-compatible provider needs a base URL and a model", ErrAIProviderError)
	}
	msgs := make([]openAICompatMessage, 0, 2)
	if sys := flattenSystem(req.System); sys != "" {
		msgs = append(msgs, openAICompatMessage{Role: "system", Content: sys})
	}
	msgs = append(msgs, openAICompatMessage{Role: "user", Content: req.UserPrompt})
	body, err := json.Marshal(openAICompatRequest{Model: p.model, Messages: msgs, MaxTokens: req.MaxTokens})
	if err != nil {
		return CompletionResult{}, fmt.Errorf("%w: encode request: %v", ErrAIProviderError, err)
	}
	// An exhausted run budget must not sit out a pacing slot first (CR L1).
	if b := BudgetFromContext(ctx); b != nil && b.Exceeded() {
		return CompletionResult{}, ErrBudgetExceeded
	}
	if err := p.pace(ctx); err != nil {
		return CompletionResult{}, err
	}
	reply, err := governed(ctx, p.governor, "openai-compat:"+p.model, func() (openAICompatReply, error) {
		return retryTransient(ctx, "openai-compat:"+p.model, func() (openAICompatReply, bool, error) {
			return p.attempt(ctx, body)
		})
	})
	if err != nil {
		// A quota wall that outlived every retry (Gemini's free tier: 5 a
		// minute, 20 a day on some models) is the account saying no, not the
		// network — the same class Claude's 429 maps to, so the pipeline's
		// "a quota stops the run, it does not degrade" rule applies (CR M1).
		if errors.Is(err, ErrAIRetriesExhausted) && errors.Is(err, errOpenAICompatQuota) {
			return CompletionResult{}, fmt.Errorf("%w: %w", ErrAIQuotaExceeded, err)
		}
		return CompletionResult{}, err
	}
	if p.billing {
		if b := BudgetFromContext(ctx); b != nil {
			b.RecordLLM(p.model, reply.usage.InputTokens, reply.usage.OutputTokens)
		}
	}
	return CompletionResult{Text: reply.text, Usage: reply.usage}, nil
}

// flattenSystem joins ordered system blocks into one system message.
func flattenSystem(blocks []SystemBlock) string {
	var sb strings.Builder
	for _, b := range blocks {
		if strings.TrimSpace(b.Text) == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(b.Text)
	}
	return sb.String()
}

// pace waits until minInterval has passed since the previous call's slot.
// Calls take slots in order; a cancelled ctx ends the wait.
func (p *OpenAICompatProvider) pace(ctx context.Context) error {
	if p.minInterval <= 0 {
		return nil
	}
	p.paceMu.Lock()
	now := time.Now()
	slot := p.paceNext
	if slot.Before(now) {
		slot = now
	}
	p.paceNext = slot.Add(p.minInterval)
	p.paceMu.Unlock()
	wait := time.Until(slot)
	if wait <= 0 {
		return nil
	}
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// attempt is one HTTP round trip; it reports whether its failure is worth
// another try.
func (p *OpenAICompatProvider) attempt(ctx context.Context, body []byte) (openAICompatReply, bool, error) {
	var none openAICompatReply
	attemptCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return none, false, fmt.Errorf("%w: build request: %v", ErrAIProviderError, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			// The caller's ctx ended the call — not ours to retry.
			return none, false, ctx.Err()
		}
		if errors.Is(attemptCtx.Err(), context.DeadlineExceeded) {
			return none, true, fmt.Errorf("%w: no reply within %s", ErrAITimeout, p.timeout)
		}
		return none, true, fmt.Errorf("%w: %v", ErrAIProviderError, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		if ctx.Err() != nil {
			return none, false, ctx.Err()
		}
		if errors.Is(attemptCtx.Err(), context.DeadlineExceeded) {
			return none, true, fmt.Errorf("%w: no reply within %s", ErrAITimeout, p.timeout)
		}
		return none, true, fmt.Errorf("%w: read body: %v", ErrAIProviderError, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		text := openAICompatErrorText(data)
		if resp.StatusCode == http.StatusTooManyRequests && openAICompatIsQuota(text) {
			// Retryable (a per-minute window refills), but tagged so that
			// running out of retries reads as a quota, not a flaky server.
			return none, true, fmt.Errorf("%w: %w: HTTP 429: %s", ErrAIProviderError, errOpenAICompatQuota, text)
		}
		return none, isTransientStatus(resp.StatusCode),
			fmt.Errorf("%w: HTTP %d: %s", ErrAIProviderError, resp.StatusCode, text)
	}

	var out openAICompatResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return none, false, fmt.Errorf("%w: %v", ErrAIInvalidResponse, err)
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return none, false, ErrAIInvalidResponse
	}
	return openAICompatReply{
		text:  out.Choices[0].Message.Content,
		usage: CompletionUsage{InputTokens: out.Usage.PromptTokens, OutputTokens: out.Usage.CompletionTokens},
	}, false, nil
}

// errOpenAICompatQuota tags a 429 whose body says the ACCOUNT's quota is
// spent (Gemini: "You exceeded your current quota" / RESOURCE_EXHAUSTED).
var errOpenAICompatQuota = errors.New("quota exceeded")

// openAICompatIsQuota recognises a quota 429 by the words every such body
// carries (Gemini 2026-10-14; OpenAI "insufficient_quota").
func openAICompatIsQuota(text string) bool {
	lower := strings.ToLower(text)
	for _, needle := range []string{"quota", "resource_exhausted"} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
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
