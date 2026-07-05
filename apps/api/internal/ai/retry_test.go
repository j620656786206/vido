package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain shrinks retry backoff for the whole ai package test suite —
// retry LOGIC is under test, wall-clock delays are not.
func TestMain(m *testing.M) {
	retryBaseDelay = 5 * time.Millisecond
	retryMaxDelay = 20 * time.Millisecond
	os.Exit(m.Run())
}

// --- 9R-4: transient retry/backoff ---

func TestRetryTransient_TransientThenSuccess(t *testing.T) {
	calls := 0
	got, err := retryTransient(context.Background(), "test.op", func() (string, bool, error) {
		calls++
		if calls < 3 {
			return "", true, errors.New("transient")
		}
		return "ok", false, nil
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", got)
	assert.Equal(t, 3, calls)
}

func TestRetryTransient_PermanentNoRetry(t *testing.T) {
	calls := 0
	_, err := retryTransient(context.Background(), "test.op", func() (string, bool, error) {
		calls++
		return "", false, errors.New("permanent 400")
	})
	require.Error(t, err)
	assert.Equal(t, 1, calls, "permanent errors must not retry")
}

func TestRetryTransient_CapsAttempts(t *testing.T) {
	calls := 0
	_, err := retryTransient(context.Background(), "test.op", func() (string, bool, error) {
		calls++
		return "", true, errors.New("always transient")
	})
	require.Error(t, err)
	assert.Equal(t, retryMaxAttempts, calls)
}

func TestClaudeProvider_RetriesTransientThenSucceeds(t *testing.T) {
	// AC #3: simulate a transient failure then success (5xx -> 200).
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"content":[{"type":"text","text":"recovered"}],"stop_reason":"end_turn"}`))
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))
	result, err := p.CompleteText(context.Background(), "", "hello", 64)
	require.NoError(t, err)
	assert.Equal(t, "recovered", result)
	assert.Equal(t, int32(2), hits.Load())
}

func TestClaudeProvider_NoRetryOnPermanent4xx(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	p := NewClaudeProvider("test-key", WithClaudeBaseURL(server.URL))
	_, err := p.CompleteText(context.Background(), "", "hello", 64)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAIProviderError)
	assert.Equal(t, int32(1), hits.Load(), "4xx (non-429) must not retry")
}

func TestWhisperClient_RetriesTransientThenSucceeds(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte("1\n00:00:00,000 --> 00:00:01,000\nrecovered\n"))
	}))
	defer server.Close()

	audio := writeTempAudio(t)
	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))
	srt, err := c.Transcribe(context.Background(), audio)
	require.NoError(t, err)
	assert.Contains(t, srt, "recovered")
	assert.Equal(t, int32(2), hits.Load())
}
