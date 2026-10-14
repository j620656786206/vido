package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shrinkRetryDelays makes the package backoff instant for one test.
func shrinkRetryDelays(t *testing.T) {
	t.Helper()
	base, max := retryBaseDelay, retryMaxDelay
	retryBaseDelay, retryMaxDelay = time.Millisecond, time.Millisecond
	t.Cleanup(func() { retryBaseDelay, retryMaxDelay = base, max })
}

func TestOpenAICompat_SendsChatRequestAndReadsContent(t *testing.T) {
	var got openAICompatRequest
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		auth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		_, _ = io.WriteString(w, `{"choices":[{"index":0,"message":{"role":"assistant","content":"你好。"}}]}`)
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider(srv.URL+"/v1/", "hy-mt2-7b")
	out, err := p.CompleteText(context.Background(), "sys", "Hello.", 256)
	if err != nil || out != "你好。" {
		t.Fatalf("got %q, %v", out, err)
	}
	if got.Model != "hy-mt2-7b" || got.MaxTokens != 256 || got.Stream {
		t.Fatalf("request: %+v", got)
	}
	if len(got.Messages) != 2 || got.Messages[0].Role != "system" || got.Messages[1].Content != "Hello." {
		t.Fatalf("messages: %+v", got.Messages)
	}
	if auth != "" {
		t.Fatalf("no key configured, but sent Authorization %q", auth)
	}
}

func TestOpenAICompat_OmitsEmptySystemAndSendsBearerKey(t *testing.T) {
	var got openAICompatRequest
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider(srv.URL, "m", WithOpenAICompatAPIKey(" sk-local "))
	if _, err := p.CompleteText(context.Background(), "", "u", 0); err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 1 || got.Messages[0].Role != "user" {
		t.Fatalf("empty system prompt must not be sent: %+v", got.Messages)
	}
	if auth != "Bearer sk-local" {
		t.Fatalf("Authorization = %q", auth)
	}
}

func TestOpenAICompat_ClientErrorFailsFastWithServerMessage(t *testing.T) {
	shrinkRetryDelays(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":{"message":"model \"nope\" not found","type":"not_found_error"}}`)
	}))
	defer srv.Close()

	_, err := NewOpenAICompatProvider(srv.URL, "nope").CompleteText(context.Background(), "", "u", 0)
	if err == nil || !errors.Is(err, ErrAIProviderError) || !strings.Contains(err.Error(), `model "nope" not found`) {
		t.Fatalf("want provider error quoting the server, got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("a 404 must not retry, got %d calls", calls.Load())
	}
}

func TestOpenAICompat_ServerErrorRetriesThenSucceeds(t *testing.T) {
	shrinkRetryDelays(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"error":"Loading model"}`)
			return
		}
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"好"}}]}`)
	}))
	defer srv.Close()

	out, err := NewOpenAICompatProvider(srv.URL, "m").CompleteText(context.Background(), "", "u", 0)
	if err != nil || out != "好" || calls.Load() != 3 {
		t.Fatalf("got %q, %v after %d calls", out, err, calls.Load())
	}
}

func TestOpenAICompat_ServerErrorExhaustsRetries(t *testing.T) {
	shrinkRetryDelays(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := NewOpenAICompatProvider(srv.URL, "m").CompleteText(context.Background(), "", "u", 0)
	if !errors.Is(err, ErrAIRetriesExhausted) || int(calls.Load()) != retryMaxAttempts {
		t.Fatalf("want retries exhausted after %d calls, got %v after %d", retryMaxAttempts, err, calls.Load())
	}
}

func TestOpenAICompat_PerAttemptTimeoutIsTransient(t *testing.T) {
	shrinkRetryDelays(t)
	var calls atomic.Int32
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer srv.Close()
	defer close(release)

	p := NewOpenAICompatProvider(srv.URL, "m", WithOpenAICompatTimeout(20*time.Millisecond))
	_, err := p.CompleteText(context.Background(), "", "u", 0)
	if !errors.Is(err, ErrAITimeout) || !errors.Is(err, ErrAIRetriesExhausted) {
		t.Fatalf("want timeout retried to exhaustion, got %v", err)
	}
	if int(calls.Load()) != retryMaxAttempts {
		t.Fatalf("each attempt gets its own deadline; want %d calls, got %d", retryMaxAttempts, calls.Load())
	}
}

func TestOpenAICompat_CallerCancelStopsWithoutRetry(t *testing.T) {
	shrinkRetryDelays(t)
	var calls atomic.Int32
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer srv.Close()
	defer close(release)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := NewOpenAICompatProvider(srv.URL, "m").CompleteText(ctx, "", "u", 0)
	if !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 1 {
		t.Fatalf("caller's deadline ends the call once: %v after %d calls", err, calls.Load())
	}
}

func TestOpenAICompat_EmptyOrMalformedReplyIsInvalidResponse(t *testing.T) {
	for name, body := range map[string]string{
		"no choices":    `{"choices":[]}`,
		"blank content": `{"choices":[{"message":{"content":"  "}}]}`,
		"not json":      `<html>oops</html>`,
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, body)
			}))
			defer srv.Close()
			_, err := NewOpenAICompatProvider(srv.URL, "m").CompleteText(context.Background(), "", "u", 0)
			if !errors.Is(err, ErrAIInvalidResponse) {
				t.Fatalf("want ErrAIInvalidResponse, got %v", err)
			}
		})
	}
}

func TestOpenAICompat_MissingConfigFailsBeforeAnyCall(t *testing.T) {
	if _, err := NewOpenAICompatProvider("", "m").CompleteText(context.Background(), "", "u", 0); !errors.Is(err, ErrAIProviderError) {
		t.Fatalf("missing base URL: %v", err)
	}
	if _, err := NewOpenAICompatProvider("http://x", " ").CompleteText(context.Background(), "", "u", 0); !errors.Is(err, ErrAIProviderError) {
		t.Fatalf("missing model: %v", err)
	}
}

func TestOpenAICompat_DoesNotChargeTheBudget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"好"}}],"usage":{"prompt_tokens":3000,"completion_tokens":400}}`)
	}))
	defer srv.Close()
	b := NewBudget(1)
	ctx := WithBudget(context.Background(), b)
	if _, err := NewOpenAICompatProvider(srv.URL, "m").CompleteText(ctx, "", "u", 0); err != nil {
		t.Fatal(err)
	}
	if b.SpentUSD() != 0 {
		t.Fatalf("local inference is free; spent %v", b.SpentUSD())
	}
}

// TestOpenAICompat_RealOllamaShapes replays bodies recorded from Ollama 0.40.3
// on the NAS (Rule 28): a real success and a real model-not-found error.
func TestOpenAICompat_RealOllamaShapes(t *testing.T) {
	shrinkRetryDelays(t)
	for _, tc := range []struct {
		fixture string
		status  int
		check   func(t *testing.T, out string, err error)
	}{
		{"testdata/openai_compat_ollama_ok.json", http.StatusOK, func(t *testing.T, out string, err error) {
			if err != nil || out != "你真是一個混蛋，湯姆。" {
				t.Fatalf("got %q, %v", out, err)
			}
		}},
		{"testdata/openai_compat_ollama_404.json", http.StatusNotFound, func(t *testing.T, _ string, err error) {
			if !errors.Is(err, ErrAIProviderError) || !strings.Contains(err.Error(), "model 'no-such-model' not found") {
				t.Fatalf("want the server's own message, got %v", err)
			}
		}},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			body, err := os.ReadFile(tc.fixture)
			if err != nil {
				t.Fatal(err)
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write(body)
			}))
			defer srv.Close()
			out, err := NewOpenAICompatProvider(srv.URL, "hy-mt2-7b").CompleteText(context.Background(), "", "u", 0)
			tc.check(t, out, err)
		})
	}
}

// ─── gemini-translation-dispatch: billing, usage, pacing ───────────────────

func TestOpenAICompat_WithBillingRecordsUsageAtTheCatalogPrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"好"}}],"usage":{"prompt_tokens":1000000,"completion_tokens":1000000}}`)
	}))
	defer srv.Close()
	b := NewBudget(100)
	ctx := WithBudget(context.Background(), b)
	p := NewOpenAICompatProvider(srv.URL, "gemini-3.5-flash-lite", WithOpenAICompatBilling(true))
	res, err := p.CompleteTextWithUsage(ctx, CompletionRequest{UserPrompt: "u", MaxTokens: 10})
	require.NoError(t, err)
	assert.Equal(t, "好", res.Text)
	assert.Equal(t, int64(1000000), res.Usage.InputTokens)
	assert.Equal(t, int64(1000000), res.Usage.OutputTokens)
	// 1M in at $0.30 + 1M out at $2.50 = $2.80 — the catalog row, not a guess.
	want := llmPricing("gemini-3.5-flash-lite")
	assert.InDelta(t, want.InputPer1M.Add(want.OutputPer1M).InexactFloat64(), b.SpentUSD(), 1e-9)
}

func TestOpenAICompat_SystemBlocksAreFlattenedInOrder(t *testing.T) {
	var got openAICompatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	defer srv.Close()
	_, err := NewOpenAICompatProvider(srv.URL, "m").CompleteTextWithUsage(context.Background(), CompletionRequest{
		System:     []SystemBlock{{Text: "rules"}, {Text: ""}, {Text: "glossary", CacheTTL: CacheTTL1h}},
		UserPrompt: "u",
	})
	require.NoError(t, err)
	require.Len(t, got.Messages, 2)
	assert.Equal(t, "rules\n\nglossary", got.Messages[0].Content)
}

func TestOpenAICompat_MinIntervalSpacesCalls(t *testing.T) {
	var stamps []time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stamps = append(stamps, time.Now())
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	defer srv.Close()
	p := NewOpenAICompatProvider(srv.URL, "m", WithOpenAICompatMinInterval(60*time.Millisecond))
	for i := 0; i < 3; i++ {
		_, err := p.CompleteText(context.Background(), "", "u", 1)
		require.NoError(t, err)
	}
	require.Len(t, stamps, 3)
	assert.GreaterOrEqual(t, stamps[1].Sub(stamps[0]), 55*time.Millisecond)
	assert.GreaterOrEqual(t, stamps[2].Sub(stamps[1]), 55*time.Millisecond)
}

func TestOpenAICompat_CancelledPacingWaitReturnsCtxErr(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	defer srv.Close()
	p := NewOpenAICompatProvider(srv.URL, "m", WithOpenAICompatMinInterval(time.Hour))
	_, err := p.CompleteText(context.Background(), "", "u", 1) // first: no wait
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = p.CompleteText(ctx, "", "u", 1)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int32(1), calls.Load(), "a cancelled wait never reaches the server")
}

func TestOpenAICompat_ExhaustedBudgetSkipsTheCallWhenGoverned(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer srv.Close()
	b := NewBudget(0.001)
	b.RecordLLM("claude-haiku-4-5", 1_000_000, 0)
	require.True(t, b.Exceeded())
	p := NewOpenAICompatProvider(srv.URL, "m", WithOpenAICompatGovernor(NewGovernor(1, 0, 1)))
	_, err := p.CompleteText(WithBudget(context.Background(), b), "", "u", 1)
	assert.ErrorIs(t, err, ErrBudgetExceeded)
	assert.Zero(t, calls.Load())
}

func TestOpenAICompat_QuotaWallBecomesQuotaExceededAfterRetries(t *testing.T) {
	shrinkRetryDelays(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		// Real Gemini body shape (2026-10-14), shortened.
		_, _ = io.WriteString(w, `{"error":{"code":429,"message":"You exceeded your current quota, please check your plan and billing details. Quota exceeded for metric: generativelanguage.googleapis.com/generate_content_free_tier_requests, limit: 20","status":"RESOURCE_EXHAUSTED"}}`)
	}))
	defer srv.Close()
	_, err := NewOpenAICompatProvider(srv.URL, "gemini-3.5-flash-lite").CompleteText(context.Background(), "", "u", 1)
	assert.ErrorIs(t, err, ErrAIQuotaExceeded, "a quota wall is the account saying no, the class the pipeline stops on")
	assert.Equal(t, int32(retryMaxAttempts), calls.Load(), "a per-minute window may refill, so it is still retried first")
}

func TestOpenAICompat_PlainRateLimitStaysTransient(t *testing.T) {
	shrinkRetryDelays(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"error":{"message":"slow down"}}`)
	}))
	defer srv.Close()
	_, err := NewOpenAICompatProvider(srv.URL, "m").CompleteText(context.Background(), "", "u", 1)
	assert.ErrorIs(t, err, ErrAIRetriesExhausted)
	assert.NotErrorIs(t, err, ErrAIQuotaExceeded)
}

func TestOpenAICompat_ExhaustedBudgetDoesNotWaitForAPacingSlot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	defer srv.Close()
	p := NewOpenAICompatProvider(srv.URL, "m", WithOpenAICompatMinInterval(time.Hour))
	_, err := p.CompleteText(context.Background(), "", "u", 1)
	require.NoError(t, err)
	b := NewBudget(0.001)
	b.RecordLLM("claude-haiku-4-5", 1_000_000, 0)
	start := time.Now()
	_, err = p.CompleteText(WithBudget(context.Background(), b), "", "u", 1)
	assert.ErrorIs(t, err, ErrBudgetExceeded)
	assert.Less(t, time.Since(start), time.Second)
}
