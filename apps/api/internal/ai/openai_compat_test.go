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
