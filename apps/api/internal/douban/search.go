package douban

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Searcher provides Douban search functionality
type Searcher struct {
	client *Client
	logger *slog.Logger
}

// NewSearcher creates a new Douban searcher
func NewSearcher(client *Client, logger *slog.Logger) *Searcher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Searcher{
		client: client,
		logger: logger,
	}
}

// Search searches Douban for the given query and returns results
func (s *Searcher) Search(ctx context.Context, query string, mediaType MediaType) ([]SearchResult, error) {
	if !s.client.IsEnabled() {
		return nil, &BlockedError{
			Reason: "client is disabled",
		}
	}

	url := SearchURL(query, mediaType)
	s.logger.Info("Searching Douban",
		"query", query,
		"media_type", mediaType,
		"url", url,
	)

	body, err := s.client.GetBody(ctx, url)
	if err != nil {
		s.logger.Error("Search request failed",
			"query", query,
			"error", err,
		)
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	results, err := s.parseSearchResults(body)
	if err != nil {
		s.logger.Error("Failed to parse search results",
			"query", query,
			"error", err,
		)
		return nil, err
	}

	s.logger.Info("Search completed",
		"query", query,
		"result_count", len(results),
	)

	return results, nil
}

// parseSearchResults parses the Douban search results page HTML
func (s *Searcher) parseSearchResults(html string) ([]SearchResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, &ParseError{
			Field:  "document",
			Reason: "failed to parse HTML: " + err.Error(),
		}
	}

	var results []SearchResult

	// Douban search results are in a specific structure
	// The search page uses JavaScript to render results, but there's also
	// a server-rendered version we can parse
	doc.Find(".result-list .result").Each(func(i int, sel *goquery.Selection) {
		result, err := s.parseSearchResultItem(sel)
		if err != nil {
			s.logger.Warn("Failed to parse search result item",
				"index", i,
				"error", err,
			)
			return
		}
		if result != nil {
			results = append(results, *result)
		}
	})

	// Alternative selector for different search page layouts
	if len(results) == 0 {
		doc.Find(".item-root").Each(func(i int, sel *goquery.Selection) {
			result, err := s.parseSearchResultItemAlt(sel)
			if err != nil {
				s.logger.Warn("Failed to parse search result item (alt)",
					"index", i,
					"error", err,
				)
				return
			}
			if result != nil {
				results = append(results, *result)
			}
		})
	}

	// Third alternative: direct subject items
	if len(results) == 0 {
		doc.Find(".sc-bZQynM, [data-testid='subject-item']").Each(func(i int, sel *goquery.Selection) {
			result, err := s.parseSubjectItem(sel)
			if err != nil {
				s.logger.Warn("Failed to parse subject item",
					"index", i,
					"error", err,
				)
				return
			}
			if result != nil {
				results = append(results, *result)
			}
		})
	}

	return results, nil
}

// parseSearchResultItem parses a single search result from .result element
func (s *Searcher) parseSearchResultItem(sel *goquery.Selection) (*SearchResult, error) {
	// Get the link to detail page
	linkSel := sel.Find("a.nbg, .title a, h3 a").First()
	href, exists := linkSel.Attr("href")
	if !exists {
		return nil, nil // Skip items without links
	}

	// Extract subject ID from URL
	id := extractSubjectID(href)
	if id == "" {
		return nil, nil
	}

	result := &SearchResult{
		ID:   id,
		URL:  href,
		Type: MediaTypeMovie, // Default to movie
	}

	// Get title
	titleSel := sel.Find(".title a, h3 a, .title-text").First()
	if titleSel.Length() > 0 {
		result.Title = strings.TrimSpace(titleSel.Text())
	}

	// Get year from subtitle or meta info
	subtitleSel := sel.Find(".subject-cast, .rating-info, .meta").First()
	if subtitleSel.Length() > 0 {
		text := subtitleSel.Text()
		year := extractYear(text)
		if year > 0 {
			result.Year = year
		}
	}

	// Get rating
	ratingSel := sel.Find(".rating_nums, .rating-value, span[class*='rating']").First()
	if ratingSel.Length() > 0 {
		ratingText := strings.TrimSpace(ratingSel.Text())
		if rating, err := strconv.ParseFloat(ratingText, 64); err == nil {
			result.Rating = rating
		}
	}

	// Detect media type from URL or content
	if strings.Contains(href, "/tv/") || strings.Contains(href, "tv_subject") {
		result.Type = MediaTypeTV
	}

	return result, nil
}

// parseSearchResultItemAlt parses a single search result from .item-root element
func (s *Searcher) parseSearchResultItemAlt(sel *goquery.Selection) (*SearchResult, error) {
	// Get the link
	linkSel := sel.Find("a").First()
	href, exists := linkSel.Attr("href")
	if !exists {
		return nil, nil
	}

	id := extractSubjectID(href)
	if id == "" {
		return nil, nil
	}

	result := &SearchResult{
		ID:   id,
		URL:  href,
		Type: MediaTypeMovie,
	}

	// Get title from various possible elements
	titleSel := sel.Find(".title, .item-title, [class*='title']").First()
	if titleSel.Length() > 0 {
		result.Title = strings.TrimSpace(titleSel.Text())
	} else {
		// Try getting from link text
		result.Title = strings.TrimSpace(linkSel.Text())
	}

	// Get year
	metaSel := sel.Find(".meta, .abstract, .info").First()
	if metaSel.Length() > 0 {
		year := extractYear(metaSel.Text())
		if year > 0 {
			result.Year = year
		}
	}

	// Get rating
	ratingSel := sel.Find("[class*='rating'] span, .rating_nums").First()
	if ratingSel.Length() > 0 {
		ratingText := strings.TrimSpace(ratingSel.Text())
		if rating, err := strconv.ParseFloat(ratingText, 64); err == nil {
			result.Rating = rating
		}
	}

	return result, nil
}

// parseSubjectItem parses a subject item from newer search layouts
func (s *Searcher) parseSubjectItem(sel *goquery.Selection) (*SearchResult, error) {
	// Find the link
	linkSel := sel.Find("a[href*='subject']").First()
	href, exists := linkSel.Attr("href")
	if !exists {
		return nil, nil
	}

	id := extractSubjectID(href)
	if id == "" {
		return nil, nil
	}

	result := &SearchResult{
		ID:   id,
		URL:  href,
		Type: MediaTypeMovie,
	}

	// Get title
	titleSel := sel.Find("span[class*='title'], div[class*='title']").First()
	if titleSel.Length() > 0 {
		result.Title = strings.TrimSpace(titleSel.Text())
	}

	return result, nil
}

// extractSubjectID extracts the subject ID from a Douban URL
func extractSubjectID(url string) string {
	// Match patterns like /subject/1292052/ or /subject/1292052
	re := regexp.MustCompile(`/subject/(\d+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// extractYear extracts a 4-digit year from text
func extractYear(text string) int {
	// Look for 4-digit year pattern (1900-2099)
	re := regexp.MustCompile(`(19|20)\d{2}`)
	match := re.FindString(text)
	if match != "" {
		year, _ := strconv.Atoi(match)
		return year
	}
	return 0
}

// SearchByID searches for a specific Douban subject ID
func (s *Searcher) SearchByID(ctx context.Context, id string) (*SearchResult, error) {
	if !s.client.IsEnabled() {
		return nil, &BlockedError{
			Reason: "client is disabled",
		}
	}

	url := DetailURL(id)
	s.logger.Info("Fetching Douban subject by ID",
		"id", id,
		"url", url,
	)

	resp, err := s.client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subject: %w", err)
	}
	defer resp.Body.Close()

	// Check for 404
	if resp.StatusCode == 404 {
		return nil, &NotFoundError{ID: id}
	}

	// We just need to confirm it exists and extract basic info
	return &SearchResult{
		ID:   id,
		URL:  url,
		Type: MediaTypeMovie,
	}, nil
}
