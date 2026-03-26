package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/time/rate"
)

func TestOpenSubProvider_Name(t *testing.T) {
	p := &OpenSubProvider{}
	assert.Equal(t, "opensubtitles", p.Name())
}

func TestOpenSubProvider_Disabled(t *testing.T) {
	p := NewOpenSubProvider(context.Background(), newMockSecrets(map[string]string{}))
	assert.True(t, p.disabled)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "test"})
	assert.NoError(t, err)
	assert.Nil(t, results)
}

func TestOpenSubProvider_AuthFlow(t *testing.T) {
	var loginCalled int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			atomic.AddInt32(&loginCalled, 1)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"token":  "test-jwt-token",
				"status": 200,
			})
		case "/subtitles":
			// Verify auth headers
			assert.Equal(t, "test-api-key", r.Header.Get("Api-Key"))
			assert.Equal(t, "Bearer test-jwt-token", r.Header.Get("Authorization"))

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(openSubSearchResponse{Data: nil})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := newTestOpenSubProvider(server.URL)

	_, err := p.Search(context.Background(), SubtitleQuery{Title: "test"})
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&loginCalled))
}

func TestOpenSubProvider_SearchWithIMDB(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(map[string]string{"token": "tok"})
			return
		}

		assert.Equal(t, "tt1234567", r.URL.Query().Get("imdb_id"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openSubSearchResponse{
			Data: []openSubSearchItem{
				{
					ID: "sub1",
					Attributes: openSubSearchAttrs{
						Language:      "zh-tw",
						DownloadCount: 500,
						Release:       "Movie.2024.1080p",
						UploadDate:    "2024-06-15T12:00:00Z",
						Uploader:      &openSubUploader{Name: "TestUser"},
						Files: []openSubFile{
							{FileID: 42, FileName: "movie.zh-tw.srt"},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	p := newTestOpenSubProvider(server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{ImdbID: "tt1234567"})
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, "42", results[0].ID)
	assert.Equal(t, "opensubtitles", results[0].Source)
	assert.Equal(t, "zh-tw", results[0].Language)
	assert.Equal(t, 500, results[0].Downloads)
	assert.Equal(t, "TestUser", results[0].Group)
	assert.Equal(t, "movie.zh-tw.srt", results[0].Filename)
	assert.Equal(t, "srt", results[0].Format)
}

func TestOpenSubProvider_SearchWithHash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(map[string]string{"token": "tok"})
			return
		}

		// Verify hash parameter is sent
		assert.Equal(t, "abc123hash", r.URL.Query().Get("moviehash"))
		assert.Equal(t, "tt1234567", r.URL.Query().Get("imdb_id"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openSubSearchResponse{Data: nil})
	}))
	defer server.Close()

	p := newTestOpenSubProvider(server.URL)

	_, err := p.Search(context.Background(), SubtitleQuery{
		ImdbID:   "tt1234567",
		FileHash: "abc123hash",
	})
	require.NoError(t, err)
}

func TestOpenSubProvider_SearchWithSeasonEpisode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(map[string]string{"token": "tok"})
			return
		}

		assert.Equal(t, "3", r.URL.Query().Get("season_number"))
		assert.Equal(t, "5", r.URL.Query().Get("episode_number"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openSubSearchResponse{Data: nil})
	}))
	defer server.Close()

	p := newTestOpenSubProvider(server.URL)

	_, err := p.Search(context.Background(), SubtitleQuery{
		Title:   "Test Show",
		Season:  3,
		Episode: 5,
	})
	require.NoError(t, err)
}

func TestOpenSubProvider_SearchHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(map[string]string{"token": "tok"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := newTestOpenSubProvider(server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "test"})
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestOpenSubProvider_SearchEmptyQuery(t *testing.T) {
	p := &OpenSubProvider{disabled: false}
	results, err := p.Search(context.Background(), SubtitleQuery{})
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "requires title or IMDB ID")
}

func TestOpenSubProvider_DownloadSuccess(t *testing.T) {
	subtitleContent := []byte("1\n00:00:01,000 --> 00:00:03,000\n你好世界\n")

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		switch {
		case r.URL.Path == "/login":
			json.NewEncoder(w).Encode(map[string]string{"token": "tok"})
		case r.URL.Path == "/download":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(openSubDownloadResponse{
				Link:     fmt.Sprintf("http://%s/files/sub.srt", r.Host),
				FileName: "sub.srt",
			})
		case r.URL.Path == "/files/sub.srt":
			w.Write(subtitleContent)
		default:
			t.Fatalf("unexpected request %d: %s", count, r.URL.Path)
		}
	}))
	defer server.Close()

	p := newTestOpenSubProvider(server.URL)

	data, err := p.Download(context.Background(), "42")
	require.NoError(t, err)
	assert.Equal(t, subtitleContent, data)
}

func TestOpenSubProvider_DownloadDisabled(t *testing.T) {
	p := &OpenSubProvider{disabled: true}
	data, err := p.Download(context.Background(), "42")
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestOpenSubProvider_DownloadInvalidID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(map[string]string{"token": "tok"})
			return
		}
	}))
	defer server.Close()

	p := newTestOpenSubProvider(server.URL)

	data, err := p.Download(context.Background(), "not-a-number")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "invalid file ID")
}

func TestOpenSubProvider_RateLimiting429(t *testing.T) {
	var attempt int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(map[string]string{"token": "tok"})
			return
		}

		count := atomic.AddInt32(&attempt, 1)
		if count == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openSubSearchResponse{Data: nil})
	}))
	defer server.Close()

	p := newTestOpenSubProvider(server.URL)

	start := time.Now()
	results, err := p.Search(context.Background(), SubtitleQuery{Title: "test"})
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond, "should wait for Retry-After")
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempt))
}

func TestOpenSubProvider_TokenRefresh(t *testing.T) {
	var loginCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			atomic.AddInt32(&loginCount, 1)
			json.NewEncoder(w).Encode(map[string]string{"token": "new-token"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openSubSearchResponse{Data: nil})
	}))
	defer server.Close()

	p := newTestOpenSubProvider(server.URL)
	// Set expired token
	p.authToken = "old-token"
	p.tokenExpiry = time.Now().Add(-1 * time.Hour)

	_, err := p.Search(context.Background(), SubtitleQuery{Title: "test"})
	require.NoError(t, err)

	// Should have re-authenticated
	assert.Equal(t, int32(1), atomic.LoadInt32(&loginCount))
	assert.Equal(t, "new-token", p.authToken)
}

func TestParseRetryAfter(t *testing.T) {
	assert.Equal(t, 5*time.Second, parseRetryAfter("5"))
	assert.Equal(t, 30*time.Second, parseRetryAfter("30"))
	assert.Equal(t, 60*time.Second, parseRetryAfter("120")) // capped at 60
	assert.Equal(t, 5*time.Second, parseRetryAfter(""))
	assert.Equal(t, 5*time.Second, parseRetryAfter("invalid"))
}

func TestCalculateOpenSubHash(t *testing.T) {
	// Create a test file with known content — all zeros
	dir := t.TempDir()
	path := filepath.Join(dir, "testfile.mkv")

	// 200KB of zeros: hash = fileSize + sum_first_64KB(zeros) + sum_last_64KB(zeros)
	// = 204800 + 0 + 0 = 204800 = 0x0000000000032000
	fileSize := 200 * 1024
	data := make([]byte, fileSize)
	err := os.WriteFile(path, data, 0644)
	require.NoError(t, err)

	hash, err := CalculateOpenSubHash(path)
	require.NoError(t, err)
	assert.Len(t, hash, 16) // 64-bit hex = 16 chars
	assert.Regexp(t, `^[0-9a-f]{16}$`, hash)

	// Known value: for all-zero file, hash = file size only
	expectedHash := fmt.Sprintf("%016x", fileSize)
	assert.Equal(t, expectedHash, hash, "all-zero file hash should equal file size")

	// Deterministic: same file → same hash
	hash2, err := CalculateOpenSubHash(path)
	require.NoError(t, err)
	assert.Equal(t, hash, hash2)
}

func TestCalculateOpenSubHash_KnownContent(t *testing.T) {
	// Create a file with known non-zero content for deterministic hash verification
	dir := t.TempDir()
	path := filepath.Join(dir, "known.mkv")

	fileSize := 128 * 1024 // exactly 128KB (minimum valid size)
	data := make([]byte, fileSize)
	// Fill first 64KB with 0x01 pattern
	for i := 0; i < 64*1024; i++ {
		data[i] = 0x01
	}
	// Fill last 64KB with 0x02 pattern
	for i := 64 * 1024; i < 128*1024; i++ {
		data[i] = 0x02
	}

	err := os.WriteFile(path, data, 0644)
	require.NoError(t, err)

	hash, err := CalculateOpenSubHash(path)
	require.NoError(t, err)
	assert.Len(t, hash, 16)

	// Compute expected hash manually:
	// First 64KB: 8192 uint64 words, each = 0x0101010101010101
	// Sum = 8192 * 0x0101010101010101
	// Last 64KB: 8192 uint64 words, each = 0x0202020202020202
	// Sum = 8192 * 0x0202020202020202
	// Hash = fileSize + firstSum + lastSum
	var firstSum, lastSum uint64
	firstWord := uint64(0x0101010101010101)
	lastWord := uint64(0x0202020202020202)
	firstSum = 8192 * firstWord
	lastSum = 8192 * lastWord
	expectedHash := fmt.Sprintf("%016x", uint64(fileSize)+firstSum+lastSum)

	assert.Equal(t, expectedHash, hash, "known content hash should match manual calculation")
}

func TestCalculateOpenSubHash_FileTooSmall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tiny.mkv")
	err := os.WriteFile(path, []byte("too small"), 0644)
	require.NoError(t, err)

	_, err = CalculateOpenSubHash(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too small")
}

func TestCalculateOpenSubHash_FileNotFound(t *testing.T) {
	_, err := CalculateOpenSubHash("/nonexistent/file.mkv")
	assert.Error(t, err)
}

func TestOpenSubProvider_RateLimiter(t *testing.T) {
	var searchCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(map[string]string{"token": "tok"})
			return
		}
		atomic.AddInt32(&searchCount, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openSubSearchResponse{Data: nil})
	}))
	defer server.Close()

	p := newTestOpenSubProvider(server.URL)

	// Make 5 rapid requests — should be throttled
	// With burst=2, first 2 go immediately, then ~500ms each for 3rd, 4th, 5th
	start := time.Now()
	for i := 0; i < 5; i++ {
		_, _ = p.Search(context.Background(), SubtitleQuery{Title: "test"})
	}
	elapsed := time.Since(start)

	// With 2 req/s limit and burst=2, 5 requests should take at least ~1200ms
	// (2 burst + 3 throttled at 500ms each = ~1500ms minimum)
	assert.GreaterOrEqual(t, elapsed, 1200*time.Millisecond, "rate limiter should throttle requests")
	assert.Equal(t, int32(5), atomic.LoadInt32(&searchCount))
}

func TestOpenSubProvider_RateLimiterContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(map[string]string{"token": "tok"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openSubSearchResponse{Data: nil})
	}))
	defer server.Close()

	// Use a very restrictive limiter (1 req/s, burst 1) to make the test fast
	p := newTestOpenSubProvider(server.URL)
	p.rateLimiter = rate.NewLimiter(rate.Limit(1), 1)

	// Exhaust the burst token
	_, _ = p.Search(context.Background(), SubtitleQuery{Title: "test"})

	// Now cancel context before the rate limiter can issue the next token
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := p.Search(ctx, SubtitleQuery{Title: "test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limiter")
}

// --- Helper ---

func newTestOpenSubProvider(baseURL string) *OpenSubProvider {
	return &OpenSubProvider{
		apiKey:      "test-api-key",
		username:    "testuser",
		password:    "testpass",
		disabled:    false,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		rateLimiter: rate.NewLimiter(rate.Limit(openSubRateLimit), openSubRateBurst),
		testBaseURL: baseURL,
	}
}
