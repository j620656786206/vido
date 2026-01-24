package douban

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScraper(t *testing.T) {
	client := NewClient(DefaultConfig(), nil)
	scraper := NewScraper(client, nil)

	assert.NotNil(t, scraper)
	assert.NotNil(t, scraper.client)
	assert.NotNil(t, scraper.logger)
}

func TestScraper_ParseDetailPage(t *testing.T) {
	t.Run("parses movie detail page from testdata", func(t *testing.T) {
		// Read test HTML file
		htmlPath := filepath.Join("testdata", "movie_detail.html")
		htmlBytes, err := os.ReadFile(htmlPath)
		require.NoError(t, err)

		client := NewClient(DefaultConfig(), nil)
		scraper := NewScraper(client, nil)

		result, err := scraper.parseDetailPage("27010768", string(htmlBytes))

		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify basic fields
		assert.Equal(t, "27010768", result.ID)
		assert.Equal(t, "寄生上流", result.Title)
		assert.Equal(t, 2019, result.Year)
		assert.Equal(t, 8.7, result.Rating)
		assert.Equal(t, 1234567, result.RatingCount)

		// Verify director
		assert.Contains(t, result.Director, "奉俊昊")

		// Verify cast
		assert.Contains(t, result.Cast, "宋康昊")
		assert.Contains(t, result.Cast, "李善均")

		// Verify genres
		assert.Contains(t, result.Genres, "剧情")
		assert.Contains(t, result.Genres, "喜剧")

		// Verify countries
		assert.Contains(t, result.Countries, "韩国")

		// Verify languages
		assert.Contains(t, result.Languages, "韩语")

		// Verify runtime
		assert.Equal(t, 132, result.Runtime)

		// Verify poster URL
		assert.Contains(t, result.PosterURL, "doubanio.com")

		// Verify summary
		assert.Contains(t, result.Summary, "基澤")

		// Verify IMDb ID
		assert.Equal(t, "tt6751668", result.IMDbID)

		// Verify media type
		assert.Equal(t, MediaTypeMovie, result.Type)
	})

	t.Run("parses minimal HTML", func(t *testing.T) {
		html := `<!DOCTYPE html>
<html>
<body>
<div id="content">
  <h1>
    <span property="v:itemreviewed">测试电影</span>
    <span class="year">(2020)</span>
  </h1>
  <strong class="rating_num">7.5</strong>
</div>
</body>
</html>`

		client := NewClient(DefaultConfig(), nil)
		scraper := NewScraper(client, nil)

		result, err := scraper.parseDetailPage("12345", html)

		require.NoError(t, err)
		assert.Equal(t, "12345", result.ID)
		assert.Equal(t, "测试电影", result.Title)
		assert.Equal(t, 2020, result.Year)
		assert.Equal(t, 7.5, result.Rating)
	})

	t.Run("handles missing fields gracefully", func(t *testing.T) {
		html := `<!DOCTYPE html>
<html>
<body>
<div id="content">
  <h1><span property="v:itemreviewed">无评分电影</span></h1>
</div>
</body>
</html>`

		client := NewClient(DefaultConfig(), nil)
		scraper := NewScraper(client, nil)

		result, err := scraper.parseDetailPage("99999", html)

		require.NoError(t, err)
		assert.Equal(t, "无评分电影", result.Title)
		assert.Equal(t, 0, result.Year)
		assert.Equal(t, 0.0, result.Rating)
		assert.Empty(t, result.Cast)
		assert.Empty(t, result.Genres)
	})

	t.Run("detects TV show", func(t *testing.T) {
		html := `<!DOCTYPE html>
<html>
<body>
<div id="content">
  <h1><span property="v:itemreviewed">电视剧测试</span><span class="year">(2021)</span></h1>
  <div id="info">
    <span class="pl">集数:</span> 16<br/>
    <span class="pl">单集片长:</span> 60分钟<br/>
  </div>
</div>
</body>
</html>`

		client := NewClient(DefaultConfig(), nil)
		scraper := NewScraper(client, nil)

		result, err := scraper.parseDetailPage("88888", html)

		require.NoError(t, err)
		assert.Equal(t, MediaTypeTV, result.Type)
		assert.Equal(t, 16, result.Episodes)
	})
}

func TestScraper_ScrapeDetail(t *testing.T) {
	t.Run("returns error when client is disabled", func(t *testing.T) {
		config := DefaultConfig()
		config.Enabled = false
		client := NewClient(config, nil)
		scraper := NewScraper(client, nil)

		_, err := scraper.ScrapeDetail(context.Background(), "12345")

		require.Error(t, err)
		var blockedErr *BlockedError
		assert.ErrorAs(t, err, &blockedErr)
	})
}

func TestScraper_ScrapeByURL(t *testing.T) {
	t.Run("extracts ID and scrapes", func(t *testing.T) {
		config := DefaultConfig()
		config.Enabled = false // Disable to avoid actual request
		client := NewClient(config, nil)
		scraper := NewScraper(client, nil)

		// Will fail because client is disabled, but ID extraction should work
		_, err := scraper.ScrapeByURL(context.Background(), "https://movie.douban.com/subject/27010768/")

		// Should fail with disabled error, not parse error
		var blockedErr *BlockedError
		assert.ErrorAs(t, err, &blockedErr)
	})

	t.Run("returns error for invalid URL", func(t *testing.T) {
		client := NewClient(DefaultConfig(), nil)
		scraper := NewScraper(client, nil)

		_, err := scraper.ScrapeByURL(context.Background(), "https://movie.douban.com/top250")

		require.Error(t, err)
		var parseErr *ParseError
		assert.ErrorAs(t, err, &parseErr)
	})
}

func TestScraper_ExtractInfoField(t *testing.T) {
	scraper := NewScraper(NewClient(DefaultConfig(), nil), nil)

	tests := []struct {
		name      string
		html      string
		text      string
		fieldName string
		want      string
	}{
		{
			name:      "extracts director from text",
			html:      "",
			text:      "导演: 奉俊昊\n编剧: 奉俊昊",
			fieldName: "导演",
			want:      "奉俊昊",
		},
		{
			name:      "extracts country from text",
			html:      "",
			text:      "制片国家/地区: 韩国 / 美国",
			fieldName: "制片国家/地区",
			want:      "韩国 / 美国",
		},
		{
			name:      "handles Chinese colon",
			html:      "",
			text:      "语言：韩语 / 英语",
			fieldName: "语言",
			want:      "韩语 / 英语",
		},
		{
			name:      "returns empty for missing field",
			html:      "",
			text:      "导演: 某某",
			fieldName: "编剧",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scraper.extractInfoField(tt.html, tt.text, tt.fieldName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScraper_ConvertToTraditional(t *testing.T) {
	client := NewClient(DefaultConfig(), nil)
	scraper := NewScraper(client, nil)

	t.Run("converts simplified title to traditional", func(t *testing.T) {
		result := &DetailResult{
			Title:   "寄生虫", // Simplified
			Summary: "基泽一家四口全是无业游民", // Simplified
		}

		scraper.convertToTraditional(result)

		// TitleTraditional should be set
		assert.NotEmpty(t, result.TitleTraditional)
		// SummaryTraditional should be set
		assert.NotEmpty(t, result.SummaryTraditional)
	})

	t.Run("handles empty fields gracefully", func(t *testing.T) {
		result := &DetailResult{
			Title:   "",
			Summary: "",
		}

		// Should not panic
		scraper.convertToTraditional(result)

		assert.Empty(t, result.TitleTraditional)
		assert.Empty(t, result.SummaryTraditional)
	})

	t.Run("preserves traditional chinese", func(t *testing.T) {
		result := &DetailResult{
			Title:   "寄生蟲", // Already Traditional
			Summary: "基澤一家四口全是無業遊民", // Already Traditional
		}

		scraper.convertToTraditional(result)

		// Should still set the traditional fields (converter handles this)
		assert.NotEmpty(t, result.TitleTraditional)
		assert.NotEmpty(t, result.SummaryTraditional)
	})
}

func TestScraper_DetectTVShow(t *testing.T) {
	client := NewClient(DefaultConfig(), nil)
	scraper := NewScraper(client, nil)

	t.Run("detects TV show by episodes", func(t *testing.T) {
		html := `<html><body><div id="info">集数: 16</div></body></html>`
		result := &DetailResult{Episodes: 16}

		// Parse doc for testing
		doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
		isTV := scraper.detectTVShow(doc, result)

		assert.True(t, isTV)
	})

	t.Run("detects movie by default", func(t *testing.T) {
		html := `<html><body><div id="info">片长: 120分钟</div></body></html>`
		result := &DetailResult{}

		doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
		isTV := scraper.detectTVShow(doc, result)

		assert.False(t, isTV)
	})
}
