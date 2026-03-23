package providers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- HTML Fixtures ---

const searchResultsHTML = `<!DOCTYPE html>
<html>
<body>
<div class="search-result">
  <div class="item">
    <a href="/detail/12345" class="title">進擊的巨人 第三季</a>
    <span class="lang">繁體中文</span>
    <span class="downloads">5678</span>
    <span class="group">字幕組A</span>
  </div>
  <div class="item">
    <a href="/detail/12346" class="title">進擊的巨人 第三季 簡體</a>
    <span class="lang">簡體中文</span>
    <span class="downloads">1234</span>
    <span class="group">字幕組B</span>
  </div>
  <div class="item">
    <a href="/detail/12347" class="title">進擊的巨人 雙語字幕</a>
    <span class="lang">雙語</span>
    <span class="downloads">890</span>
    <span class="group">字幕組C</span>
  </div>
  <div class="item">
    <a href="/detail/12348" class="title">Attack on Titan S3</a>
    <span class="lang">English</span>
    <span class="downloads">456</span>
    <span class="group">SubGroup</span>
  </div>
</div>
</body>
</html>`

const detailPageHTML = `<!DOCTYPE html>
<html>
<body>
<h1>進擊的巨人 第三季 字幕</h1>
<div class="subtitle-info">
  <a href="/download/sub_12345.srt" class="download">下載字幕</a>
</div>
</body>
</html>`

const captchaPageHTML = `<!DOCTYPE html>
<html>
<body>
<div class="captcha">
  <p>請完成驗證碼</p>
  <div class="recaptcha"></div>
</div>
</body>
</html>`

const emptySearchHTML = `<!DOCTYPE html>
<html>
<body>
<div class="no-results">
  <p>沒有找到相關字幕</p>
</div>
</body>
</html>`

// --- Tests ---

func TestZimukuProvider_Name(t *testing.T) {
	p := NewZimukuProvider()
	assert.Equal(t, "zimuku", p.Name())
}

func TestZimukuProvider_SearchSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/search", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("q"))
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(searchResultsHTML))
	}))
	defer server.Close()

	p := newTestZimukuProvider(server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "進擊的巨人"})
	require.NoError(t, err)
	require.Len(t, results, 4)

	// Verify first result
	assert.Equal(t, "/detail/12345", results[0].ID)
	assert.Equal(t, "進擊的巨人 第三季", results[0].Filename)
	assert.Equal(t, "zimuku", results[0].Source)
	assert.Equal(t, "zh-Hant", results[0].Language)
	assert.Equal(t, 5678, results[0].Downloads)
	assert.Equal(t, "字幕組A", results[0].Group)
}

func TestZimukuProvider_LanguageMapping(t *testing.T) {
	tests := []struct {
		label    string
		expected string
	}{
		{"繁體中文", "zh-Hant"},
		{"繁体中文", "zh-Hant"},
		{"簡體中文", "zh-Hans"},
		{"简体中文", "zh-Hans"},
		{"雙語", "zh"},
		{"双语", "zh"},
		{"English", "en"},
		{"英文", "en"},
		{"Unknown", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapZimukuLanguage(tt.label))
		})
	}
}

func TestZimukuProvider_SearchCaptchaDetected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(captchaPageHTML))
	}))
	defer server.Close()

	p := newTestZimukuProvider(server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "test"})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrCaptchaDetected))
	assert.Nil(t, results)
}

func TestZimukuProvider_SearchEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(emptySearchHTML))
	}))
	defer server.Close()

	p := newTestZimukuProvider(server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "nonexistent"})
	assert.NoError(t, err)
	assert.Nil(t, results)
}

func TestZimukuProvider_SearchEmptyTitle(t *testing.T) {
	p := NewZimukuProvider()
	results, err := p.Search(context.Background(), SubtitleQuery{Title: ""})
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "title is required")
}

func TestZimukuProvider_SearchHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"403 Forbidden", http.StatusForbidden},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			p := newTestZimukuProvider(server.URL)

			results, err := p.Search(context.Background(), SubtitleQuery{Title: "test"})
			assert.Error(t, err)
			assert.Nil(t, results)
		})
	}
}

func TestZimukuProvider_SearchParseFailure(t *testing.T) {
	// Malformed HTML that won't parse properly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><p>Just text, no results container</p></body></html>`))
	}))
	defer server.Close()

	p := newTestZimukuProvider(server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "test"})
	// No error — just empty results (no search container found)
	assert.NoError(t, err)
	assert.Nil(t, results)
}

func TestZimukuProvider_DownloadSuccess(t *testing.T) {
	subtitleContent := []byte("1\n00:00:01,000 --> 00:00:03,000\n你好世界\n")

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)

		if count == 1 {
			// Detail page request
			assert.Equal(t, "/detail/12345", r.URL.Path)
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(detailPageHTML))
		} else {
			// Download request
			assert.Equal(t, "/download/sub_12345.srt", r.URL.Path)
			w.Write(subtitleContent)
		}
	}))
	defer server.Close()

	p := newTestZimukuProvider(server.URL)

	data, err := p.Download(context.Background(), "/detail/12345")
	require.NoError(t, err)
	assert.Equal(t, subtitleContent, data)
}

func TestZimukuProvider_DownloadCaptchaOnDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(captchaPageHTML))
	}))
	defer server.Close()

	p := newTestZimukuProvider(server.URL)

	data, err := p.Download(context.Background(), "/detail/12345")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrCaptchaDetected))
	assert.Nil(t, data)
}

func TestZimukuProvider_DownloadNoLink(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><h1>Detail page without download link</h1></body></html>`))
	}))
	defer server.Close()

	p := newTestZimukuProvider(server.URL)

	data, err := p.Download(context.Background(), "/detail/12345")
	assert.Error(t, err)
	assert.Nil(t, data)

	var parseErr *ErrParseFailure
	assert.True(t, errors.As(err, &parseErr))
	assert.Contains(t, parseErr.Selector, "download")
}

func TestZimukuProvider_DownloadHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := newTestZimukuProvider(server.URL)

	data, err := p.Download(context.Background(), "/detail/12345")
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestZimukuProvider_DownloadEmptyID(t *testing.T) {
	p := NewZimukuProvider()
	data, err := p.Download(context.Background(), "")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "ID is required")
}

func TestZimukuProvider_DownloadRejectsAbsoluteURL(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"http URL", "http://evil.com/steal-creds"},
		{"https URL", "https://169.254.169.254/latest/meta-data/"},
		{"protocol-relative URL", "//evil.com/ssrf"},
		{"no leading slash", "detail/12345"},
	}

	p := NewZimukuProvider()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := p.Download(context.Background(), tt.id)
			assert.ErrorIs(t, err, ErrInvalidID)
			assert.Nil(t, data)
		})
	}
}

func TestZimukuProvider_DownloadCaptchaOnFileResponse(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count == 1 {
			// Detail page — normal
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(detailPageHTML))
		} else {
			// Download endpoint returns CAPTCHA instead of file
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(captchaPageHTML))
		}
	}))
	defer server.Close()

	p := newTestZimukuProvider(server.URL)
	data, err := p.Download(context.Background(), "/detail/12345")
	assert.ErrorIs(t, err, ErrCaptchaDetected)
	assert.Nil(t, data)
}

func TestZimukuProvider_UserAgentRotation(t *testing.T) {
	var mu sync.Mutex
	agents := make(map[string]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		agents[r.Header.Get("User-Agent")] = true
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(emptySearchHTML))
	}))
	defer server.Close()

	p := newTestZimukuProvider(server.URL)

	// Make enough requests to likely get different user agents
	for i := 0; i < 20; i++ {
		_, _ = p.Search(context.Background(), SubtitleQuery{Title: "test"})
	}

	mu.Lock()
	agentCount := len(agents)
	mu.Unlock()
	// With 10 UAs and 20 requests, we should have at least 2 different ones
	assert.GreaterOrEqual(t, agentCount, 2, "should rotate between multiple user agents")
}

func TestZimukuProvider_ContextCancellation(t *testing.T) {
	p := newTestZimukuProvider("http://localhost:1") // Will not connect

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := p.Search(ctx, SubtitleQuery{Title: "test"})
	assert.Error(t, err)
}

func TestErrParseFailure_Error(t *testing.T) {
	err := &ErrParseFailure{Selector: ".download", Context: "detail page"}
	assert.Contains(t, err.Error(), ".download")
	assert.Contains(t, err.Error(), "detail page")

	err2 := &ErrParseFailure{Selector: ".item"}
	assert.Contains(t, err2.Error(), ".item")
	assert.NotContains(t, err2.Error(), "()")
}

func TestDetectCaptcha(t *testing.T) {
	assert.True(t, detectCaptcha([]byte("please solve the captcha")))
	assert.True(t, detectCaptcha([]byte("請完成驗證碼")))
	assert.True(t, detectCaptcha([]byte("recaptcha challenge")))
	assert.False(t, detectCaptcha([]byte("normal search results page")))
}

// --- Helper ---

// newTestZimukuProvider creates a ZimukuProvider pointing at a test server
// with delays disabled for fast tests.
func newTestZimukuProvider(baseURL string) *ZimukuProvider {
	return &ZimukuProvider{
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		userAgents:  defaultUserAgents,
		testBaseURL: baseURL,
		skipDelays:  true,
	}
}
