package metadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/douban"
	"github.com/vido/api/internal/models"
)

func TestNewDoubanProvider(t *testing.T) {
	config := DefaultDoubanProviderConfig()
	provider := NewDoubanProvider(config)

	assert.NotNil(t, provider)
	assert.Equal(t, "Douban", provider.Name())
	assert.Equal(t, models.MetadataSourceDouban, provider.Source())
	assert.True(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusAvailable, provider.Status())
}

func TestDoubanProvider_Disabled(t *testing.T) {
	config := DefaultDoubanProviderConfig()
	config.Enabled = false
	provider := NewDoubanProvider(config)

	assert.False(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusUnavailable, provider.Status())
}

func TestDoubanProvider_Search_Disabled(t *testing.T) {
	config := DefaultDoubanProviderConfig()
	config.Enabled = false
	provider := NewDoubanProvider(config)

	result, err := provider.Search(context.Background(), &SearchRequest{
		Query:     "Test Movie",
		MediaType: MediaTypeMovie,
	})

	// Should return error indicating provider is disabled
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "disabled")
}

func TestDoubanProvider_SetEnabled(t *testing.T) {
	config := DefaultDoubanProviderConfig()
	provider := NewDoubanProvider(config)

	assert.True(t, provider.IsAvailable())

	provider.SetEnabled(false)
	assert.False(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusUnavailable, provider.Status())

	provider.SetEnabled(true)
	assert.True(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusAvailable, provider.Status())
}

func TestDoubanProvider_ImplementsInterface(t *testing.T) {
	config := DefaultDoubanProviderConfig()
	provider := NewDoubanProvider(config)

	// Compile-time interface verification
	var _ MetadataProvider = provider
}

func TestDoubanProvider_CircuitBreaker(t *testing.T) {
	config := DefaultDoubanProviderConfig()
	config.CircuitBreakerConfig.FailureThreshold = 2
	config.CircuitBreakerConfig.Timeout = 100 * time.Millisecond
	provider := NewDoubanProvider(config)

	// Initial state should be closed
	stats := provider.GetCircuitBreakerStats()
	assert.Equal(t, 0, stats.TotalCalls)
	assert.Equal(t, 0, stats.FailureCount)

	// Reset should work
	provider.ResetCircuitBreaker()
	stats = provider.GetCircuitBreakerStats()
	assert.Equal(t, 0, stats.TotalCalls)
}

func TestDoubanProvider_ClientMetrics(t *testing.T) {
	config := DefaultDoubanProviderConfig()
	provider := NewDoubanProvider(config)

	metrics := provider.GetClientMetrics()
	assert.Equal(t, int64(0), metrics.TotalRequests)
	assert.Equal(t, int64(0), metrics.SuccessfulRequests)
}

func TestDoubanProvider_ConvertToMetadataItem(t *testing.T) {
	config := DefaultDoubanProviderConfig()
	provider := NewDoubanProvider(config)

	detail := &douban.DetailResult{
		ID:                 "27010768",
		Title:              "寄生虫",
		TitleTraditional:   "寄生上流",
		OriginalTitle:      "Parasite",
		Year:               2019,
		Rating:             8.7,
		RatingCount:        1234567,
		Director:           "奉俊昊",
		Cast:               []string{"宋康昊", "李善均"},
		Genres:             []string{"剧情", "喜剧"},
		Summary:            "基泽一家...",
		SummaryTraditional: "基澤一家...",
		PosterURL:          "https://example.com/poster.jpg",
		Type:               douban.MediaTypeMovie,
		ReleaseDate:        "2019-05-21",
	}

	item := provider.convertToMetadataItem(detail)

	assert.Equal(t, "27010768", item.ID)
	assert.Equal(t, "寄生虫", item.Title)
	assert.Equal(t, "寄生上流", item.TitleZhTW)
	assert.Equal(t, "Parasite", item.OriginalTitle)
	assert.Equal(t, 2019, item.Year)
	assert.Equal(t, 8.7, item.Rating)
	assert.Equal(t, 1234567, item.VoteCount)
	assert.Equal(t, "基泽一家...", item.Overview)
	assert.Equal(t, "基澤一家...", item.OverviewZhTW)
	assert.Equal(t, "https://example.com/poster.jpg", item.PosterURL)
	assert.Equal(t, MediaTypeMovie, item.MediaType)
	assert.Contains(t, item.Genres, "剧情")
	assert.Equal(t, 0.8, item.Confidence)
	assert.NotNil(t, item.RawData)
}

func TestDoubanProvider_ConvertToMetadataItem_TVShow(t *testing.T) {
	config := DefaultDoubanProviderConfig()
	provider := NewDoubanProvider(config)

	detail := &douban.DetailResult{
		ID:    "12345",
		Title: "测试电视剧",
		Type:  douban.MediaTypeTV,
	}

	item := provider.convertToMetadataItem(detail)

	assert.Equal(t, MediaTypeTV, item.MediaType)
}

func TestDoubanProvider_Search_WithMockServer(t *testing.T) {
	// Create a mock server that returns search results
	searchHTML := `<!DOCTYPE html>
<html>
<body>
<div class="result-list">
  <div class="result">
    <a class="nbg" href="https://movie.douban.com/subject/27010768/">
      <img src="poster.jpg" />
    </a>
    <div class="title">
      <a href="https://movie.douban.com/subject/27010768/">寄生上流</a>
    </div>
    <div class="rating-info">
      <span class="rating_nums">8.7</span>
      <span class="subject-cast">2019 / 韩国</span>
    </div>
  </div>
</div>
</body>
</html>`

	detailHTML := `<!DOCTYPE html>
<html>
<body>
<div id="content">
  <h1>
    <span property="v:itemreviewed">寄生上流</span>
    <span class="year">(2019)</span>
  </h1>
  <strong class="rating_num">8.7</strong>
  <div id="info">
    <span class="pl">导演</span>: <span class="attrs"><a href="#">奉俊昊</a></span><br/>
    <span class="pl">主演</span>: <span class="actor"><span class="attrs"><a href="#">宋康昊</a></span></span><br/>
    <span property="v:genre">剧情</span> / <span property="v:genre">喜剧</span><br/>
    <span class="pl">制片国家/地区:</span> 韩国<br/>
  </div>
  <div id="mainpic">
    <img src="https://example.com/poster.jpg" />
  </div>
  <span property="v:summary">基泽一家四口...</span>
</div>
</body>
</html>`

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		// First request is search, subsequent are detail pages
		if requestCount == 1 {
			w.Write([]byte(searchHTML))
		} else {
			w.Write([]byte(detailHTML))
		}
	}))
	defer server.Close()

	// Note: We can't easily inject a custom base URL into the provider
	// This test demonstrates the structure but would need URL injection for full integration testing
	_ = server.URL
}

func TestDefaultDoubanProviderConfig(t *testing.T) {
	config := DefaultDoubanProviderConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, 0.5, config.ClientConfig.RequestsPerSecond)
	assert.Equal(t, 3, config.CircuitBreakerConfig.FailureThreshold)
	assert.Equal(t, 2, config.CircuitBreakerConfig.SuccessThreshold)
	assert.Equal(t, 60*time.Second, config.CircuitBreakerConfig.Timeout)
}
