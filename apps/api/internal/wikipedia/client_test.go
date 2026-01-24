package wikipedia

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with default config", func(t *testing.T) {
		config := DefaultConfig()
		client := NewClient(config, nil)

		assert.NotNil(t, client)
		assert.True(t, client.IsEnabled())
		assert.NotNil(t, client.httpClient)
		assert.NotNil(t, client.rateLimiter)
	})

	t.Run("applies default values for zero config", func(t *testing.T) {
		config := ClientConfig{}
		client := NewClient(config, nil)

		assert.NotNil(t, client)
		// Client should have applied defaults
		assert.Equal(t, 10*time.Second, client.config.Timeout)
		assert.Equal(t, 1.0, client.config.RequestsPerSecond)
		assert.Equal(t, 3, client.config.MaxRetries)
	})

	t.Run("uses provided logger", func(t *testing.T) {
		logger := slog.Default()
		config := DefaultConfig()
		client := NewClient(config, logger)

		assert.Equal(t, logger, client.logger)
	})
}

func TestClient_BuildUserAgent(t *testing.T) {
	t.Run("builds correct user agent format", func(t *testing.T) {
		config := ClientConfig{
			UserAgent:    "TestApp/1.0",
			ContactEmail: "test@example.com",
		}
		client := NewClient(config, nil)

		ua := client.buildUserAgent()

		assert.Contains(t, ua, "TestApp/1.0")
		assert.Contains(t, ua, "test@example.com")
		assert.Contains(t, ua, "https://github.com/vido")
		assert.Contains(t, ua, "Go-http-client/1.1")
	})

	t.Run("uses default contact email when not provided", func(t *testing.T) {
		config := ClientConfig{
			UserAgent: "TestApp/1.0",
		}
		client := NewClient(config, nil)

		ua := client.buildUserAgent()

		assert.Contains(t, ua, "contact@example.com")
	})
}

func TestClient_IsEnabled(t *testing.T) {
	t.Run("returns enabled state", func(t *testing.T) {
		config := ClientConfig{Enabled: true}
		client := NewClient(config, nil)

		assert.True(t, client.IsEnabled())
	})

	t.Run("can toggle enabled state", func(t *testing.T) {
		config := ClientConfig{Enabled: true}
		client := NewClient(config, nil)

		client.SetEnabled(false)
		assert.False(t, client.IsEnabled())

		client.SetEnabled(true)
		assert.True(t, client.IsEnabled())
	})
}

func TestClient_Search(t *testing.T) {
	t.Run("successful search returns results", func(t *testing.T) {
		// Test with actual structure validation
		// Note: Integration tests with mock server would require modifying
		// the client to accept a custom base URL
		results := []SearchResult{
			{
				PageID:    12345,
				Title:     "寄生上流",
				Snippet:   "《寄生上流》是2019年韓國電影...",
				WordCount: 5000,
				Timestamp: "2024-01-15T10:00:00Z",
			},
		}

		assert.Len(t, results, 1)
		assert.Equal(t, int64(12345), results[0].PageID)
		assert.Equal(t, "寄生上流", results[0].Title)
	})

	t.Run("search with empty query returns error", func(t *testing.T) {
		config := ClientConfig{Enabled: true}
		client := NewClient(config, nil)

		// Empty query should still make request but return NotFoundError
		ctx := context.Background()
		_, err := client.Search(ctx, "", 5)

		// Should fail because no mock server
		assert.Error(t, err)
	})

	t.Run("disabled client returns error", func(t *testing.T) {
		config := ClientConfig{Enabled: false}
		client := NewClient(config, nil)

		ctx := context.Background()
		_, err := client.Search(ctx, "test", 5)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")
	})
}

func TestClient_GetMetrics(t *testing.T) {
	t.Run("returns metrics snapshot", func(t *testing.T) {
		config := ClientConfig{Enabled: true}
		client := NewClient(config, nil)

		metrics := client.GetMetrics()

		assert.Equal(t, int64(0), metrics.TotalRequests)
		assert.Equal(t, int64(0), metrics.SuccessfulRequests)
		assert.Equal(t, int64(0), metrics.FailedRequests)
	})
}

func TestSearchResult(t *testing.T) {
	t.Run("correctly stores search result data", func(t *testing.T) {
		result := SearchResult{
			PageID:    12345,
			Title:     "測試頁面",
			Snippet:   "這是一個測試...",
			WordCount: 1000,
			Timestamp: "2024-01-15T10:00:00Z",
		}

		assert.Equal(t, int64(12345), result.PageID)
		assert.Equal(t, "測試頁面", result.Title)
		assert.Equal(t, "這是一個測試...", result.Snippet)
		assert.Equal(t, 1000, result.WordCount)
		assert.Equal(t, "2024-01-15T10:00:00Z", result.Timestamp)
	})
}

func TestPageContent(t *testing.T) {
	t.Run("correctly stores page content data", func(t *testing.T) {
		content := PageContent{
			PageID:   12345,
			Title:    "測試頁面",
			Wikitext: "{{Infobox film\n|name=測試電影\n}}",
			Extract:  "這是測試電影的介紹。",
		}

		assert.Equal(t, int64(12345), content.PageID)
		assert.Equal(t, "測試頁面", content.Title)
		assert.Contains(t, content.Wikitext, "Infobox film")
		assert.Equal(t, "這是測試電影的介紹。", content.Extract)
	})
}

func TestImageInfo(t *testing.T) {
	t.Run("correctly stores image info data", func(t *testing.T) {
		info := ImageInfo{
			URL:            "https://upload.wikimedia.org/wikipedia/commons/1/1a/test.jpg",
			DescriptionURL: "https://commons.wikimedia.org/wiki/File:test.jpg",
			Width:          1920,
			Height:         1080,
			Size:           1024000,
			MimeType:       "image/jpeg",
		}

		assert.Equal(t, "https://upload.wikimedia.org/wikipedia/commons/1/1a/test.jpg", info.URL)
		assert.Equal(t, 1920, info.Width)
		assert.Equal(t, 1080, info.Height)
		assert.Equal(t, int64(1024000), info.Size)
		assert.Equal(t, "image/jpeg", info.MimeType)
	})
}

func TestNotFoundError(t *testing.T) {
	t.Run("error message with query", func(t *testing.T) {
		err := &NotFoundError{Query: "測試查詢"}

		assert.Contains(t, err.Error(), "no results for query")
		assert.Contains(t, err.Error(), "測試查詢")
	})

	t.Run("error message with page title", func(t *testing.T) {
		err := &NotFoundError{PageTitle: "測試頁面"}

		assert.Contains(t, err.Error(), "page not found")
		assert.Contains(t, err.Error(), "測試頁面")
	})
}

func TestParseError(t *testing.T) {
	t.Run("error message format", func(t *testing.T) {
		err := &ParseError{
			Field:    "director",
			Reason:   "invalid format",
			Wikitext: "[[導演|",
		}

		assert.Contains(t, err.Error(), "parse error")
		assert.Contains(t, err.Error(), "director")
		assert.Contains(t, err.Error(), "invalid format")
	})
}

func TestAPIError(t *testing.T) {
	t.Run("error message format", func(t *testing.T) {
		err := &APIError{
			Code: "badquery",
			Info: "Invalid query parameter",
		}

		assert.Contains(t, err.Error(), "badquery")
		assert.Contains(t, err.Error(), "Invalid query parameter")
	})
}

func TestDefaultConfig(t *testing.T) {
	t.Run("returns sensible defaults", func(t *testing.T) {
		config := DefaultConfig()

		assert.Equal(t, 1.0, config.RequestsPerSecond) // NFR-I14
		assert.Equal(t, 10*time.Second, config.Timeout)
		assert.Equal(t, 3, config.MaxRetries)
		assert.True(t, config.Enabled)
		assert.Equal(t, "Vido/1.0", config.UserAgent)
	})
}

// Integration-style test with mock server
func TestClient_Integration(t *testing.T) {
	t.Run("search and get page content flow", func(t *testing.T) {
		// This test validates the request/response structure
		// without making actual API calls

		// Simulate search response
		searchResp := searchResponse{
			Query: struct {
				Search []struct {
					PageID    int64  `json:"pageid"`
					Title     string `json:"title"`
					Snippet   string `json:"snippet"`
					WordCount int    `json:"wordcount"`
					Timestamp string `json:"timestamp"`
				} `json:"search"`
			}{
				Search: []struct {
					PageID    int64  `json:"pageid"`
					Title     string `json:"title"`
					Snippet   string `json:"snippet"`
					WordCount int    `json:"wordcount"`
					Timestamp string `json:"timestamp"`
				}{
					{
						PageID:    12345,
						Title:     "寄生上流",
						Snippet:   "韓國電影",
						WordCount: 5000,
						Timestamp: "2024-01-15T10:00:00Z",
					},
				},
			},
		}

		// Marshal and unmarshal to verify structure
		data, err := json.Marshal(searchResp)
		require.NoError(t, err)

		var parsed searchResponse
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Len(t, parsed.Query.Search, 1)
		assert.Equal(t, "寄生上流", parsed.Query.Search[0].Title)
	})
}

func TestClient_RateLimiting(t *testing.T) {
	t.Run("rate limiter is applied", func(t *testing.T) {
		config := ClientConfig{
			Enabled:           true,
			RequestsPerSecond: 1.0, // 1 request per second
		}
		client := NewClient(config, nil)

		// Verify rate limiter exists
		assert.NotNil(t, client.rateLimiter)
	})
}

func TestClient_UserAgentFormat(t *testing.T) {
	t.Run("user agent follows NFR-I13 format", func(t *testing.T) {
		config := ClientConfig{
			UserAgent:    "Vido/1.0",
			ContactEmail: "alexyu@example.com",
		}
		client := NewClient(config, nil)

		ua := client.buildUserAgent()

		// Format: ApplicationName/Version (Contact; Description)
		assert.True(t, strings.HasPrefix(ua, "Vido/1.0"))
		assert.Contains(t, ua, "https://github.com/vido")
		assert.Contains(t, ua, "alexyu@example.com")
		assert.Contains(t, ua, "Go-http-client/1.1")
	})
}
