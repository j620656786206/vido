package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// disc-2026-09-asr-quota-429-treated-as-transient — the body OpenAI sends for
// an account with no money left (NAS log 2026-09-18 23:33).
const openAIInsufficientQuota = `{"error":{"message":"You exceeded your current quota, please check your plan and billing details.","type":"insufficient_quota","param":null,"code":"insufficient_quota"}}`

func TestWhisperClient_EmptyAccountStopsAtOnceAndKeepsVerbose(t *testing.T) {
	var calls atomic.Int32
	rec := &formatRecorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		rec.record(t, r)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(openAIInsufficientQuota))
	}))
	defer server.Close()

	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))
	_, err := c.Transcribe(context.Background(), writeTempAudio(t))

	require.Error(t, err)
	var credit *CreditExhaustedError
	require.True(t, errors.As(err, &credit), "%v", err)
	assert.Equal(t, "openai", credit.Provider)
	assert.Equal(t, OpenAIBillingURL, credit.BillingURL)
	assert.ErrorIs(t, err, ErrAIQuotaExceeded, "the pipeline's stop-the-run rule keys on this")
	assert.ErrorIs(t, err, ErrWhisperAPIError)
	assert.NotContains(t, err.Error(), "{", "no JSON body in the error")
	assert.Equal(t, int32(1), calls.Load(), "waiting does not refill an account — one request, no retry, no srt fallback")

	// Filtering must still be on: the next request asks for verbose_json again.
	_, _ = c.Transcribe(context.Background(), writeTempAudio(t))
	assert.Equal(t, []string{"verbose_json", "verbose_json"}, rec.seen())
}

func TestWhisperClient_RateLimitStillRetriesAndDoesNotLatch(t *testing.T) {
	var calls atomic.Int32
	rec := &formatRecorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		rec.record(t, r)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"Rate limit reached","type":"requests","code":"rate_limit_exceeded"}}`))
	}))
	defer server.Close()

	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))
	_, err := c.Transcribe(context.Background(), writeTempAudio(t))

	require.Error(t, err)
	var credit *CreditExhaustedError
	assert.False(t, errors.As(err, &credit), "a burst limit is not an empty account")
	assert.Equal(t, int32(retryMaxAttempts), calls.Load(), "a burst limit clears — keep retrying it")
	assert.False(t, c.verboseUnsupported.Load(), "a rate limit says nothing about verbose_json")
	for _, f := range rec.seen() {
		assert.Equal(t, "verbose_json", f)
	}
}

func TestWhisperClient_RejectedKeyDoesNotLatchSRT(t *testing.T) {
	rec := &formatRecorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.record(t, r)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"Incorrect API key provided","code":"invalid_api_key"}}`))
	}))
	defer server.Close()

	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))
	_, err := c.Transcribe(context.Background(), writeTempAudio(t))

	require.ErrorIs(t, err, ErrWhisperAPIError)
	assert.False(t, c.verboseUnsupported.Load())
	assert.Equal(t, []string{"verbose_json"}, rec.seen(), "no plain-SRT retry of a rejected key")
}

func TestOpenAICreditExhausted(t *testing.T) {
	for _, body := range []string{
		openAIInsufficientQuota,
		`{"error":{"code":"billing_hard_limit_reached"}}`,
		`{"error":{"type":"CREDIT_BALANCE_EXHAUSTED"}}`,
	} {
		assert.NotNil(t, openAICreditExhausted(body), body)
	}
	for _, body := range []string{"", `{"error":{"code":"rate_limit_exceeded"}}`, `{"error":{"message":"unsupported response_format"}}`} {
		assert.Nil(t, openAICreditExhausted(body), body)
	}
}

func TestClaudeProvider_EmptyBalanceIsCreditExhausted(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"Your credit balance is too low to access the Anthropic API. Please go to Plans & Billing to upgrade or purchase credits."}}`))
	}))
	defer server.Close()

	p := NewClaudeProvider("test-api-key", WithClaudeBaseURL(server.URL))
	_, err := p.Parse(context.Background(), &ParseRequest{Filename: "test.mkv"})

	var credit *CreditExhaustedError
	require.True(t, errors.As(err, &credit), "%v", err)
	assert.Equal(t, "anthropic", credit.Provider)
	assert.Equal(t, AnthropicBillingURL, credit.BillingURL)
	assert.ErrorIs(t, err, ErrAIQuotaExceeded)
	assert.Equal(t, int32(1), calls.Load(), "not retried")
}

func TestClaudeProvider_OtherBadRequestUnchanged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens: too large"}}`))
	}))
	defer server.Close()

	p := NewClaudeProvider("test-api-key", WithClaudeBaseURL(server.URL))
	_, err := p.Parse(context.Background(), &ParseRequest{Filename: "test.mkv"})

	var credit *CreditExhaustedError
	assert.False(t, errors.As(err, &credit))
	assert.ErrorIs(t, err, ErrAIProviderError)
}
