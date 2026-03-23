package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	zimukuBaseURL          = "https://zimuku.org"
	zimukuHTTPTimeout      = 30 * time.Second
	zimukuMinDelay         = 1 * time.Second
	zimukuMaxDelay         = 3 * time.Second
	zimukuMaxResponseBytes = 2 << 20  // 2 MB max for HTML pages
	zimukuMaxDownloadBytes = 50 << 20 // 50 MB max for subtitle files
	zimukuUserAgent        = "Vido/1.0 (NAS Media Manager)"
)

// Sentinel errors for Zimuku-specific failure modes.
var (
	// ErrCaptchaDetected indicates Zimuku returned a CAPTCHA challenge page.
	// The provider does NOT attempt to solve CAPTCHAs — the engine should
	// fall back to other sources.
	ErrCaptchaDetected = errors.New("zimuku: CAPTCHA challenge detected")
)

// ErrParseFailure indicates the HTML structure changed and expected elements
// could not be found. Includes the CSS selector that failed and optional context.
type ErrParseFailure struct {
	Selector string
	Context  string
}

func (e *ErrParseFailure) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("zimuku: parse failure — selector %q not found (%s)", e.Selector, e.Context)
	}
	return fmt.Sprintf("zimuku: parse failure — selector %q not found", e.Selector)
}

// Common browser User-Agent strings for rotation.
var defaultUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
}

// ZimukuProvider implements SubtitleProvider for the Zimuku (字幕庫) subtitle source.
type ZimukuProvider struct {
	httpClient   *http.Client
	userAgents   []string
	testBaseURL  string // override for testing; empty = use zimukuBaseURL
	skipDelays   bool   // skip random delays in tests
}

// NewZimukuProvider creates a Zimuku subtitle provider with anti-scraping measures.
func NewZimukuProvider() *ZimukuProvider {
	return &ZimukuProvider{
		httpClient: &http.Client{
			Timeout: zimukuHTTPTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("zimuku: too many redirects")
				}
				return nil
			},
		},
		userAgents: defaultUserAgents,
	}
}

// Name returns the provider identifier.
func (p *ZimukuProvider) Name() string {
	return "zimuku"
}

func (p *ZimukuProvider) baseURL() string {
	if p.testBaseURL != "" {
		return p.testBaseURL
	}
	return zimukuBaseURL
}

func (p *ZimukuProvider) randomUserAgent() string {
	return p.userAgents[rand.Intn(len(p.userAgents))]
}

func (p *ZimukuProvider) randomDelay(ctx context.Context) error {
	if p.skipDelays {
		return nil
	}
	delay := zimukuMinDelay + time.Duration(rand.Int63n(int64(zimukuMaxDelay-zimukuMinDelay)))
	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *ZimukuProvider) setBrowserHeaders(req *http.Request) {
	req.Header.Set("User-Agent", p.randomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9,en;q=0.8")
	req.Header.Set("Referer", p.baseURL()+"/")
}

// detectCaptcha checks if the response body contains CAPTCHA indicators.
func detectCaptcha(body []byte) bool {
	s := string(body)
	return strings.Contains(s, "captcha") ||
		strings.Contains(s, "CAPTCHA") ||
		strings.Contains(s, "驗證碼") ||
		strings.Contains(s, "验证码") ||
		strings.Contains(s, "recaptcha")
}

// Search scrapes the Zimuku search results page for subtitles.
func (p *ZimukuProvider) Search(ctx context.Context, query SubtitleQuery) ([]SubtitleResult, error) {
	if query.Title == "" {
		return nil, fmt.Errorf("zimuku: search title is required")
	}

	if err := p.randomDelay(ctx); err != nil {
		return nil, fmt.Errorf("zimuku: delay interrupted: %w", err)
	}

	searchURL := fmt.Sprintf("%s/search?q=%s", p.baseURL(), url.QueryEscape(query.Title))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("zimuku: create request: %w", err)
	}
	p.setBrowserHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zimuku: search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, zimukuMaxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("zimuku: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("zimuku: search returned HTTP %d", resp.StatusCode)
	}

	if detectCaptcha(body) {
		return nil, ErrCaptchaDetected
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("zimuku: parse HTML: %w", err)
	}

	return p.parseSearchResults(doc)
}

// parseSearchResults extracts subtitle entries from a Zimuku search results page.
// Expected HTML structure:
//
//	<div class="search-result">
//	  <div class="item">
//	    <a href="/detail/12345" class="title">字幕標題</a>
//	    <span class="lang">繁體中文</span>
//	    <span class="downloads">1234</span>
//	    <span class="group">字幕組名稱</span>
//	  </div>
//	</div>
func (p *ZimukuProvider) parseSearchResults(doc *goquery.Document) ([]SubtitleResult, error) {
	container := doc.Find(".search-result")
	if container.Length() == 0 {
		// Try alternative selectors
		container = doc.Find("#subtitle_list, .subtitle-list, table.table tbody")
		if container.Length() == 0 {
			slog.Debug("Zimuku: no search results container found")
			return nil, nil // No results, not an error
		}
	}

	var results []SubtitleResult
	container.Find(".item, tr").Each(func(i int, s *goquery.Selection) {
		result := SubtitleResult{Source: "zimuku"}

		// Extract title and detail URL
		titleLink := s.Find("a.title, a[href*='/detail/'], td a")
		if titleLink.Length() > 0 {
			result.Filename = strings.TrimSpace(titleLink.Text())
			if href, exists := titleLink.Attr("href"); exists {
				result.ID = href
				result.DownloadURL = href
			}
		}

		// Extract language
		langEl := s.Find(".lang, .language, span[class*='lang']")
		if langEl.Length() > 0 {
			result.Language = mapZimukuLanguage(strings.TrimSpace(langEl.Text()))
		}

		// Extract download count
		dlEl := s.Find(".downloads, .download-count, span[class*='download']")
		if dlEl.Length() > 0 {
			if count, err := strconv.Atoi(strings.TrimSpace(dlEl.Text())); err == nil {
				result.Downloads = count
			}
		}

		// Extract group
		groupEl := s.Find(".group, .uploader, span[class*='group']")
		if groupEl.Length() > 0 {
			result.Group = strings.TrimSpace(groupEl.Text())
		}

		if result.ID != "" && result.Filename != "" {
			results = append(results, result)
		}
	})

	return results, nil
}

// mapZimukuLanguage maps Zimuku language labels to standard language codes.
func mapZimukuLanguage(label string) string {
	switch {
	case strings.Contains(label, "繁體"), strings.Contains(label, "繁体"):
		return "zh-Hant"
	case strings.Contains(label, "簡體"), strings.Contains(label, "简体"):
		return "zh-Hans"
	case strings.Contains(label, "雙語"), strings.Contains(label, "双语"):
		return "zh"
	case strings.Contains(label, "英"), strings.Contains(label, "English"):
		return "en"
	default:
		return label
	}
}

// Download fetches a subtitle file from Zimuku by navigating to the detail page
// and extracting the download link.
func (p *ZimukuProvider) Download(ctx context.Context, id string) ([]byte, error) {
	if id == "" {
		return nil, fmt.Errorf("zimuku: download ID is required")
	}

	// Step 1: Fetch detail page
	if err := p.randomDelay(ctx); err != nil {
		return nil, fmt.Errorf("zimuku: delay interrupted: %w", err)
	}

	detailURL := id
	if !strings.HasPrefix(id, "http") {
		detailURL = p.baseURL() + id
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, detailURL, nil)
	if err != nil {
		return nil, fmt.Errorf("zimuku: create detail request: %w", err)
	}
	p.setBrowserHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zimuku: detail request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, zimukuMaxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("zimuku: read detail response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("zimuku: detail returned HTTP %d", resp.StatusCode)
	}

	if detectCaptcha(body) {
		return nil, ErrCaptchaDetected
	}

	// Step 2: Parse detail page for download link
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("zimuku: parse detail HTML: %w", err)
	}

	downloadLink := ""
	doc.Find("a.download, a[href*='/download/'], a[href*='/dld/']").Each(func(i int, s *goquery.Selection) {
		if downloadLink == "" {
			if href, exists := s.Attr("href"); exists {
				downloadLink = href
			}
		}
	})

	if downloadLink == "" {
		return nil, &ErrParseFailure{
			Selector: "a.download, a[href*='/download/']",
			Context:  "detail page missing download link",
		}
	}

	if !strings.HasPrefix(downloadLink, "http") {
		downloadLink = p.baseURL() + downloadLink
	}

	// Step 3: Download subtitle file
	if err := p.randomDelay(ctx); err != nil {
		return nil, fmt.Errorf("zimuku: delay interrupted: %w", err)
	}

	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadLink, nil)
	if err != nil {
		return nil, fmt.Errorf("zimuku: create download request: %w", err)
	}
	p.setBrowserHeaders(dlReq)

	dlResp, err := p.httpClient.Do(dlReq)
	if err != nil {
		return nil, fmt.Errorf("zimuku: download request failed: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("zimuku: download returned HTTP %d", dlResp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(dlResp.Body, zimukuMaxDownloadBytes))
	if err != nil {
		return nil, fmt.Errorf("zimuku: read download response: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("zimuku: downloaded empty subtitle file for %s", id)
	}

	return data, nil
}

// Compile-time interface verification.
var _ SubtitleProvider = (*ZimukuProvider)(nil)
