package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── sub-6-2 AC #1: RequestTimeoutFor ──────────────────────────────────────

func TestRequestTimeoutFor_FamilyTable(t *testing.T) {
	cases := []struct {
		model string
		want  time.Duration
	}{
		{"claude-haiku-4-5", 30 * time.Second},
		{"claude-sonnet-5", 60 * time.Second},
		{"claude-sonnet-4-6", 60 * time.Second},
		{"claude-opus-4-8", 90 * time.Second},
		{"gemini-2.5-flash-lite", 30 * time.Second},
		{"gemini-3.7-flash", 30 * time.Second},
		{"Claude-Haiku-5", 30 * time.Second}, // substring, case-insensitive: a new id needs no table edit
		{"some-future-model", 60 * time.Second},
		{"", 60 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			assert.Equal(t, tc.want, RequestTimeoutFor(tc.model, 0))
		})
	}
}

func TestRequestTimeoutFor_GrowsWithOutputTokensAndCaps(t *testing.T) {
	// +10 s per 1k output tokens, linear (not stepped): 4096 → +40.96 s.
	assert.Equal(t, 60*time.Second+40960*time.Millisecond, RequestTimeoutFor("claude-sonnet-5", 4096))
	assert.Equal(t, 30*time.Second+10*time.Second, RequestTimeoutFor("claude-haiku-4-5", 1000))
	// A Ping (max_tokens 1) is effectively the bare base.
	assert.Equal(t, 30*time.Second+10*time.Millisecond, RequestTimeoutFor("claude-haiku-4-5", 1))
	// Negative / zero add nothing.
	assert.Equal(t, 90*time.Second, RequestTimeoutFor("claude-opus-4-8", -5))
	// Capped: opus + 20k output tokens would be 290 s.
	assert.Equal(t, MaxRequestTimeout, RequestTimeoutFor("claude-opus-4-8", 20000))
	assert.Equal(t, MaxRequestTimeout, RequestTimeoutFor("claude-haiku-4-5", 1_000_000))
}

func TestRequestTimeoutFor_EveryPricedModelHasAFamilyRow(t *testing.T) {
	// The two tables are maintained together (AC #1). A model priced but not
	// matched by any family row would silently get the unknown-model base.
	for model := range defaultLLMPricing {
		matched := false
		for _, row := range llmTimeoutBase {
			if baseTimeoutFor(model) == row.base && containsFamily(model, row.family) {
				matched = true
			}
		}
		assert.True(t, matched, "priced model %q has no llmTimeoutBase family row", model)
	}
}

func containsFamily(model, family string) bool {
	return len(model) >= len(family) && func() bool {
		for i := 0; i+len(family) <= len(model); i++ {
			if model[i:i+len(family)] == family {
				return true
			}
		}
		return false
	}()
}

func TestRequestTimeoutFor_HugeMaxTokensNeverGoesNegative(t *testing.T) {
	// CR L11: multiply-then-cap overflowed time.Duration into a negative
	// deadline that expired on entry.
	for _, n := range []int{1 << 40, 1 << 62, int(^uint(0) >> 1)} {
		got := RequestTimeoutFor("claude-haiku-4-5", n)
		assert.Equal(t, MaxRequestTimeout, got, "max_tokens=%d", n)
		assert.Positive(t, got)
	}
}

// shrinkTimeoutTable rebinds the family table for one test so the derivation
// can be exercised in milliseconds — the same trick TestMain plays on the
// retry backoff. It proves the per-attempt deadline IS the derived one, which
// a 16-second sleep against the old 15 s constant would only prove slowly.
func shrinkTimeoutTable(t *testing.T, base time.Duration) {
	t.Helper()
	origTable, origUnknown := llmTimeoutBase, unknownModelTimeoutBase
	llmTimeoutBase = []modelTimeoutRow{{family: "haiku", base: base}}
	unknownModelTimeoutBase = base
	t.Cleanup(func() { llmTimeoutBase, unknownModelTimeoutBase = origTable, origUnknown })
}

func okClaudeServer(t *testing.T, delay time.Duration, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			hits.Add(1)
		}
		time.Sleep(delay)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(claudeResponse{
			Content: []claudeContentBlock{{Type: "text", Text: "早安"}}, StopReason: "end_turn",
		})
	}))
}

// TestClaudeProvider_DerivedTimeoutIsThePerAttemptDeadline (AC #5b): with no
// pinned timeout the deadline each attempt runs under is
// RequestTimeoutFor(model, max_tokens) — a response slower than that times
// out, one faster than it succeeds, and there is no client-level Timeout
// competing with it.
func TestClaudeProvider_DerivedTimeoutIsThePerAttemptDeadline(t *testing.T) {
	shrinkTimeoutTable(t, 40*time.Millisecond)

	t.Run("slower than the derived deadline times out on every attempt", func(t *testing.T) {
		var hits atomic.Int32
		srv := okClaudeServer(t, 150*time.Millisecond, &hits)
		defer srv.Close()

		p := NewClaudeProvider("k", WithClaudeBaseURL(srv.URL), WithClaudeModel("claude-haiku-4-5"))
		_, err := p.CompleteText(context.Background(), "", "hi", 1)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAITimeout)
		assert.ErrorIs(t, err, ErrAIRetriesExhausted, "three timeouts in a row are a transient-exhausted verdict")
		assert.EqualValues(t, retryMaxAttempts, hits.Load(), "a fresh deadline per attempt: every attempt reaches the server")
	})

	t.Run("faster than the derived deadline succeeds", func(t *testing.T) {
		srv := okClaudeServer(t, 5*time.Millisecond, nil)
		defer srv.Close()

		p := NewClaudeProvider("k", WithClaudeBaseURL(srv.URL), WithClaudeModel("claude-haiku-4-5"))
		got, err := p.CompleteText(context.Background(), "", "hi", 1)

		require.NoError(t, err)
		assert.Equal(t, "早安", got)
	})

	t.Run("max_tokens widens the deadline", func(t *testing.T) {
		// base 40 ms + 10 s per 1k tokens: 20 tokens → +200 ms, so a 150 ms
		// response that timed out at max_tokens=1 now fits.
		srv := okClaudeServer(t, 150*time.Millisecond, nil)
		defer srv.Close()

		p := NewClaudeProvider("k", WithClaudeBaseURL(srv.URL), WithClaudeModel("claude-haiku-4-5"))
		got, err := p.CompleteText(context.Background(), "", "hi", 20)

		require.NoError(t, err, "the deadline follows what the call asks for, not a constant")
		assert.Equal(t, "早安", got)
	})
}

// TestClaudeProvider_PingUsesTheProbeBoundNotTheFamilyDerivation (CR M6): a
// key probe must not inherit a translation chunk's minute-scale deadline.
func TestClaudeProvider_PingUsesTheProbeBoundNotTheFamilyDerivation(t *testing.T) {
	shrinkTimeoutTable(t, 5*time.Millisecond)
	srv := pingServer(t, 200, `{"id":"m","type":"message","role":"assistant","model":"x","content":[],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`)
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(40 * time.Millisecond)
		srv.Config.Handler.ServeHTTP(w, r)
	}))
	defer srv.Close()
	defer slow.Close()

	p := NewClaudeProvider("k", WithClaudeBaseURL(slow.URL), WithClaudeModel("claude-haiku-4-5"))
	assert.Equal(t, ProbeRequestTimeout, p.probeTimeout())
	require.NoError(t, p.Ping(context.Background()), "a 40 ms probe under a 10 s bound passes even though the family table says 5 ms")

	_, err := p.CompleteText(context.Background(), "", "hi", 1)
	assert.ErrorIs(t, err, ErrAITimeout, "the same server IS too slow for the (shrunk) family derivation — the probe bound is separate")
}

func TestClaudeProvider_PinnedTimeoutOverridesDerivation(t *testing.T) {
	shrinkTimeoutTable(t, time.Hour)
	srv := okClaudeServer(t, 60*time.Millisecond, nil)
	defer srv.Close()

	p := NewClaudeProvider("k", WithClaudeBaseURL(srv.URL), WithClaudeTimeout(10*time.Millisecond))
	_, err := p.CompleteText(context.Background(), "", "hi", 4096)

	assert.ErrorIs(t, err, ErrAITimeout, "WithClaudeTimeout pins one number regardless of model/max_tokens")
}

func TestGeminiProvider_DerivedTimeoutIsThePerAttemptDeadline(t *testing.T) {
	shrinkTimeoutTable(t, 40*time.Millisecond)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewGeminiProvider("k", WithGeminiBaseURL(srv.URL))
	assert.Zero(t, p.httpClient.Timeout)
	assert.Equal(t, ProbeRequestTimeout, p.probeTimeout(), "Ping runs under the probe bound (claude.go parity)")
	_, err := p.Parse(context.Background(), &ParseRequest{Filename: "x.mkv"})

	assert.ErrorIs(t, err, ErrAITimeout)
	assert.ErrorIs(t, err, ErrAIRetriesExhausted)
	assert.EqualValues(t, retryMaxAttempts, hits.Load())
}

// ─── sub-6-2 AC #3 groundwork: ErrAIRetriesExhausted + RetryObserver ───────

func TestRetryTransient_ExhaustionWrapsSentinelAndKeepsLeadingCode(t *testing.T) {
	_, err := retryTransient(context.Background(), "test.op", func() (string, bool, error) {
		return "", true, ErrAITimeout
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAITimeout, "the last attempt's own sentinel still classifies")
	assert.ErrorIs(t, err, ErrAIRetriesExhausted)
	assert.True(t, len(err.Error()) > 0 && err.Error()[:len("AI_TIMEOUT")] == "AI_TIMEOUT",
		"the leading error code (Rule 7) is unchanged: %q", err.Error())
}

func TestRetryTransient_PermanentFailureIsNotExhausted(t *testing.T) {
	boom := errors.New("401")
	_, err := retryTransient(context.Background(), "test.op", func() (string, bool, error) {
		return "", false, boom
	})

	assert.ErrorIs(t, err, boom)
	assert.NotErrorIs(t, err, ErrAIRetriesExhausted, "a permanent rejection never claims the provider was merely flaky")
}

func TestRetryTransient_CancelledWindowIsNotExhausted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	_, err := retryTransient(ctx, "test.op", func() (string, bool, error) {
		calls++
		cancel() // the caller ends the window after the first attempt
		return "", true, ErrAITimeout
	})

	assert.Equal(t, 1, calls)
	assert.ErrorIs(t, err, ErrAITimeout)
	assert.NotErrorIs(t, err, ErrAIRetriesExhausted, "the caller's ctx, not the provider, ended the window")
}

func TestRetryTransient_ObserverSeesEachRetryButNotTheVerdict(t *testing.T) {
	var notices []RetryNotice
	ctx := WithRetryObserver(context.Background(), func(n RetryNotice) { notices = append(notices, n) })

	_, err := retryTransient(ctx, "claude.messages", func() (string, bool, error) {
		return "", true, ErrAITimeout
	})
	require.ErrorIs(t, err, ErrAIRetriesExhausted)

	// 3 attempts → 2 retries → 2 notices; the final failure is the return value.
	require.Len(t, notices, retryMaxAttempts-1)
	assert.Equal(t, 1, notices[0].Attempt)
	assert.Equal(t, 2, notices[1].Attempt)
	for _, n := range notices {
		assert.Equal(t, "claude.messages", n.Op)
		assert.Equal(t, retryMaxAttempts, n.MaxAttempts)
		assert.ErrorIs(t, n.Err, ErrAITimeout)
		assert.Positive(t, n.Delay)
	}
}

func TestRetryTransient_ObserverIsOptional(t *testing.T) {
	assert.Nil(t, RetryObserverFrom(context.Background()))
	ctx := WithRetryObserver(context.Background(), nil)
	assert.Nil(t, RetryObserverFrom(ctx), "a nil observer leaves the ctx untouched")

	got, err := retryTransient(ctx, "x", func() (string, bool, error) { return "ok", false, nil })
	require.NoError(t, err)
	assert.Equal(t, "ok", got)
}
