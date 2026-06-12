package douban

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Scraper provides Douban detail page scraping functionality
type Scraper struct {
	client    *Client
	converter *ChineseConverter
	logger    *slog.Logger
}

// NewScraper creates a new Douban scraper
func NewScraper(client *Client, logger *slog.Logger) *Scraper {
	if logger == nil {
		logger = slog.Default()
	}
	return &Scraper{
		client:    client,
		converter: NewChineseConverter(logger),
		logger:    logger,
	}
}

// ScrapeDetail scrapes metadata from a Douban detail page
func (s *Scraper) ScrapeDetail(ctx context.Context, id string) (*DetailResult, error) {
	if !s.client.IsEnabled() {
		return nil, &BlockedError{
			Reason: "client is disabled",
		}
	}

	url := DetailURL(id)
	s.logger.Info("Scraping Douban detail page",
		"id", id,
		"url", url,
	)

	body, err := s.client.GetBody(ctx, url)
	if err != nil {
		s.logger.Error("Failed to fetch detail page",
			"id", id,
			"error", err,
		)
		return nil, fmt.Errorf("failed to fetch detail page: %w", err)
	}

	result, err := s.parseDetailPage(id, body)
	if err != nil {
		s.logger.Error("Failed to parse detail page",
			"id", id,
			"error", err,
		)
		return nil, err
	}

	// Convert to Traditional Chinese (AC3: Traditional Chinese Priority)
	s.convertToTraditional(result)

	result.ScrapedAt = time.Now()

	s.logger.Info("Successfully scraped detail page",
		"id", id,
		"title", result.Title,
		"title_traditional", result.TitleTraditional,
		"year", result.Year,
		"rating", result.Rating,
	)

	return result, nil
}

// convertToTraditional converts Simplified Chinese fields to Traditional Chinese
func (s *Scraper) convertToTraditional(result *DetailResult) {
	// Convert title
	if result.Title != "" {
		traditional, err := s.converter.ConvertIfSimplified(result.Title)
		if err != nil {
			s.logger.Warn("Failed to convert title to Traditional",
				"title", result.Title,
				"error", err,
			)
		} else {
			result.TitleTraditional = traditional
		}
	}

	// Convert summary
	if result.Summary != "" {
		traditional, err := s.converter.ConvertIfSimplified(result.Summary)
		if err != nil {
			s.logger.Warn("Failed to convert summary to Traditional",
				"error", err,
			)
		} else {
			result.SummaryTraditional = traditional
		}
	}
}

// parseDetailPage parses a Douban detail page HTML
func (s *Scraper) parseDetailPage(id string, html string) (*DetailResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, &ParseError{
			Field:  "document",
			Reason: "failed to parse HTML: " + err.Error(),
		}
	}

	result := &DetailResult{
		ID:   id,
		Type: MediaTypeMovie,
	}

	// Extract title from h1 > span[property="v:itemreviewed"]
	// Per Dev Notes: #content h1 span[property="v:itemreviewed"]
	titleSel := doc.Find("#content h1 span[property='v:itemreviewed']")
	if titleSel.Length() > 0 {
		result.Title = strings.TrimSpace(titleSel.Text())
	} else {
		// Fallback to other title selectors
		titleSel = doc.Find("#content h1 span").First()
		if titleSel.Length() > 0 {
			result.Title = strings.TrimSpace(titleSel.Text())
		}
	}

	// Extract year from h1 > .year
	// Per Dev Notes: #content h1 .year
	yearSel := doc.Find("#content h1 .year")
	if yearSel.Length() > 0 {
		yearText := strings.TrimSpace(yearSel.Text())
		// Year is typically in format "(2019)"
		yearText = strings.Trim(yearText, "()")
		if year, err := strconv.Atoi(yearText); err == nil {
			result.Year = year
		}
	}

	// Extract rating from strong.rating_num
	// Per Dev Notes: strong.rating_num
	ratingSel := doc.Find("strong.rating_num, strong[property='v:average']")
	if ratingSel.Length() > 0 {
		ratingText := strings.TrimSpace(ratingSel.Text())
		if rating, err := strconv.ParseFloat(ratingText, 64); err == nil {
			result.Rating = rating
		}
	}

	// Extract rating count
	ratingCountSel := doc.Find("span[property='v:votes']")
	if ratingCountSel.Length() > 0 {
		countText := strings.TrimSpace(ratingCountSel.Text())
		if count, err := strconv.Atoi(countText); err == nil {
			result.RatingCount = count
		}
	}

	// Parse the #info section for various metadata
	s.parseInfoSection(doc, result)

	// Extract poster URL from #mainpic img
	// Per Dev Notes: #mainpic img
	posterSel := doc.Find("#mainpic img, #mainpic a img")
	if posterSel.Length() > 0 {
		if src, exists := posterSel.Attr("src"); exists {
			result.PosterURL = src
		}
	}

	// Extract summary from span[property="v:summary"]
	// Per Dev Notes: span[property="v:summary"]
	summarySel := doc.Find("span[property='v:summary'], .related-info span.all, .related-info span.short")
	if summarySel.Length() > 0 {
		summary := strings.TrimSpace(summarySel.First().Text())
		// Clean up whitespace
		summary = regexp.MustCompile(`\s+`).ReplaceAllString(summary, " ")
		result.Summary = summary
	}

	// Detect if this is a TV show based on content
	if s.detectTVShow(doc, result) {
		result.Type = MediaTypeTV
	}

	return result, nil
}

// parseInfoSection parses the #info section for director, cast, genres, etc.
func (s *Scraper) parseInfoSection(doc *goquery.Document, result *DetailResult) {
	infoSel := doc.Find("#info")
	if infoSel.Length() == 0 {
		return
	}

	infoHTML, _ := infoSel.Html()
	infoText := infoSel.Text()

	// Extract director
	// Per Dev Notes: #info span:contains("导演") + span a
	result.Director = s.extractInfoField(infoHTML, infoText, "导演")

	// Extract cast/actors
	// Per Dev Notes: #info span.actor span.attrs a
	actorSel := infoSel.Find("span.actor span.attrs a")
	if actorSel.Length() > 0 {
		actorSel.Each(func(i int, sel *goquery.Selection) {
			actor := strings.TrimSpace(sel.Text())
			if actor != "" && i < 10 { // Limit to first 10 actors
				result.Cast = append(result.Cast, actor)
			}
		})
	} else {
		// Fallback: try to extract from text
		actorText := s.extractInfoField(infoHTML, infoText, "主演")
		if actorText != "" {
			actors := strings.Split(actorText, "/")
			for i, actor := range actors {
				actor = strings.TrimSpace(actor)
				if actor != "" && i < 10 {
					result.Cast = append(result.Cast, actor)
				}
			}
		}
	}

	// Extract genres
	genreSel := infoSel.Find("span[property='v:genre']")
	if genreSel.Length() > 0 {
		genreSel.Each(func(i int, sel *goquery.Selection) {
			genre := strings.TrimSpace(sel.Text())
			if genre != "" {
				result.Genres = append(result.Genres, genre)
			}
		})
	}

	// Extract countries
	countryText := s.extractInfoField(infoHTML, infoText, "制片国家/地区")
	if countryText != "" {
		countries := strings.Split(countryText, "/")
		for _, country := range countries {
			country = strings.TrimSpace(country)
			if country != "" {
				result.Countries = append(result.Countries, country)
			}
		}
	}

	// Extract languages
	languageText := s.extractInfoField(infoHTML, infoText, "语言")
	if languageText != "" {
		languages := strings.Split(languageText, "/")
		for _, lang := range languages {
			lang = strings.TrimSpace(lang)
			if lang != "" {
				result.Languages = append(result.Languages, lang)
			}
		}
	}

	// Extract runtime for movies
	runtimeSel := infoSel.Find("span[property='v:runtime']")
	if runtimeSel.Length() > 0 {
		runtimeText := strings.TrimSpace(runtimeSel.Text())
		// Runtime is typically "142分钟" or "142 分钟"
		re := regexp.MustCompile(`(\d+)`)
		if match := re.FindString(runtimeText); match != "" {
			if runtime, err := strconv.Atoi(match); err == nil {
				result.Runtime = runtime
			}
		}
	}

	// Extract release date
	releaseDateSel := infoSel.Find("span[property='v:initialReleaseDate']")
	if releaseDateSel.Length() > 0 {
		result.ReleaseDate = strings.TrimSpace(releaseDateSel.First().Text())
	}

	// Extract IMDb ID
	imdbText := s.extractInfoField(infoHTML, infoText, "IMDb")
	if imdbText != "" {
		// IMDb ID is typically "tt1234567"
		re := regexp.MustCompile(`tt\d+`)
		if match := re.FindString(imdbText); match != "" {
			result.IMDbID = match
		}
	}

	// Extract number of episodes for TV shows
	episodesText := s.extractInfoField(infoHTML, infoText, "集数")
	if episodesText != "" {
		if episodes, err := strconv.Atoi(strings.TrimSpace(episodesText)); err == nil {
			result.Episodes = episodes
		}
	}

	// Extract original title (又名)
	aliasText := s.extractInfoField(infoHTML, infoText, "又名")
	if aliasText != "" {
		// Take the first alias as original title if different
		aliases := strings.Split(aliasText, "/")
		if len(aliases) > 0 {
			alias := strings.TrimSpace(aliases[0])
			if alias != result.Title {
				result.OriginalTitle = alias
			}
		}
	}
}

// extractInfoField extracts a field value from the #info section
func (s *Scraper) extractInfoField(html, text, fieldName string) string {
	// Try to find the field in text format "字段: 值"
	re := regexp.MustCompile(fieldName + `[:\s：]+([^<\n]+)`)
	if matches := re.FindStringSubmatch(text); len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}

	// Try alternative pattern
	re = regexp.MustCompile(fieldName + `</span>[:\s：]*([^<]+)`)
	if matches := re.FindStringSubmatch(html); len(matches) >= 2 {
		value := strings.TrimSpace(matches[1])
		// Clean up HTML entities
		value = strings.ReplaceAll(value, "&nbsp;", " ")
		return value
	}

	return ""
}

// detectTVShow detects if the content is a TV show based on various indicators
func (s *Scraper) detectTVShow(doc *goquery.Document, result *DetailResult) bool {
	// Check for episode count
	if result.Episodes > 0 {
		return true
	}

	// Check URL patterns in breadcrumbs
	breadcrumbs := doc.Find(".tags-body, .rec-bd").Text()
	if strings.Contains(breadcrumbs, "电视剧") || strings.Contains(breadcrumbs, "剧集") {
		return true
	}

	// Check for TV-specific genres
	for _, genre := range result.Genres {
		if genre == "电视剧" || genre == "剧集" || genre == "综艺" || genre == "动画" {
			return true
		}
	}

	// Check the info section for TV indicators
	infoText := doc.Find("#info").Text()
	if strings.Contains(infoText, "集数") || strings.Contains(infoText, "单集片长") {
		return true
	}

	return false
}

// maxReviewComments caps how many short comments (短評) the review summary returns.
const maxReviewComments = 5

var (
	// totalCommentsRe extracts the subject's full short-comment count from the
	// section header (e.g. "全部 152340 条").
	totalCommentsRe = regexp.MustCompile(`全部\s*(\d+)\s*条`)
	// allstarRe extracts the star rating from a Douban "allstarNN" class
	// (allstar40 → 4 stars). NN is a multiple of 10 on a 0-50 scale.
	allstarRe = regexp.MustCompile(`allstar(\d+)`)
	// whitespaceRe collapses runs of whitespace in comment text.
	whitespaceRe = regexp.MustCompile(`\s+`)
)

// ScrapeReviewSummary scrapes the top short comments (短評) for a Douban subject
// (Story 12-6 AC #2/#3). It reuses the subject page the rating path already knows
// (DetailURL) because the short-comments block is rendered inline on
// movie.douban.com/subject/{id}/ — no extra request. Comment text is converted to
// Traditional Chinese (AC #3). A block/parse failure is returned so the caller can
// degrade the review section to omitted while keeping the direct link (AC #5).
func (s *Scraper) ScrapeReviewSummary(ctx context.Context, id string) (*ReviewSummaryResult, error) {
	if !s.client.IsEnabled() {
		return nil, &BlockedError{
			Reason: "client is disabled",
		}
	}

	url := DetailURL(id)
	s.logger.Info("Scraping Douban review summary",
		"id", id,
		"url", url,
	)

	body, err := s.client.GetBody(ctx, url)
	if err != nil {
		s.logger.Error("Failed to fetch subject page for review summary",
			"id", id,
			"error", err,
		)
		return nil, fmt.Errorf("failed to fetch subject page: %w", err)
	}

	result, err := s.parseReviewSummary(id, body)
	if err != nil {
		s.logger.Error("Failed to parse review summary",
			"id", id,
			"error", err,
		)
		return nil, err
	}

	// Convert each comment to Traditional Chinese (AC #3) — reuses the same
	// ChineseConverter the rating-title path uses (convertToTraditional style).
	s.convertCommentsToTraditional(result)

	s.logger.Info("Successfully scraped review summary",
		"id", id,
		"total_comments", result.TotalComments,
		"top_comments", len(result.TopComments),
	)

	return result, nil
}

// parseReviewSummary extracts the short-comment summary from a Douban subject page.
// Selectors mirror the inline #comments-section block (validated against the saved
// testdata fixture). Markup drift degrades to an empty TopComments list rather than
// an error (Rule 27 Pillar 3) — an empty summary omits the review block upstream.
func (s *Scraper) parseReviewSummary(id string, html string) (*ReviewSummaryResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, &ParseError{
			Field:  "document",
			Reason: "failed to parse HTML: " + err.Error(),
		}
	}

	result := &ReviewSummaryResult{ID: id}

	// Total short-comment count from the section header ("全部 N 条").
	section := doc.Find("#comments-section")
	if section.Length() == 0 {
		section = doc.Find("#hot-comments")
	}
	if section.Length() > 0 {
		if m := totalCommentsRe.FindStringSubmatch(section.Text()); len(m) == 2 {
			if n, convErr := strconv.Atoi(m[1]); convErr == nil {
				result.TotalComments = n
			}
		}
	}

	// Top comment items (capped). Prefer the canonical #comments-section block,
	// falling back to the older #hot-comments / .comment-list layouts.
	items := doc.Find("#comments-section .comment-item")
	if items.Length() == 0 {
		items = doc.Find("#hot-comments .comment-item, .comment-list .comment-item")
	}

	items.EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		if len(result.TopComments) >= maxReviewComments {
			return false
		}
		comment := parseReviewComment(sel)
		// Skip empty rows (ads / placeholder items) — a comment with no text is
		// not useful for the summary.
		if comment.Text != "" {
			result.TopComments = append(result.TopComments, comment)
		}
		return true
	})

	// Fall back to the on-page comment-item count when the "全部 N 条" header is
	// absent. Use items.Length() (all comment rows on the page) rather than
	// len(TopComments) — the latter is capped at maxReviewComments and would
	// understate the count as exactly the cap (CR L2).
	if result.TotalComments == 0 {
		result.TotalComments = items.Length()
	}

	return result, nil
}

// parseReviewComment extracts a single short comment (author + star rating + text)
// from a .comment-item selection.
func parseReviewComment(sel *goquery.Selection) ReviewComment {
	comment := ReviewComment{}

	comment.Author = strings.TrimSpace(sel.Find(".comment-info a").First().Text())

	// Star rating lives in an "allstarNN rating" class span inside .comment-info.
	sel.Find(".comment-info span, span.rating").EachWithBreak(func(_ int, span *goquery.Selection) bool {
		class, _ := span.Attr("class")
		if m := allstarRe.FindStringSubmatch(class); len(m) == 2 {
			if n, convErr := strconv.Atoi(m[1]); convErr == nil {
				comment.Rating = n / 10 // allstar40 → 4 stars
				return false
			}
		}
		return true
	})

	text := strings.TrimSpace(sel.Find(".short").First().Text())
	if text == "" {
		text = strings.TrimSpace(sel.Find(".comment-content").First().Text())
	}
	comment.Text = strings.TrimSpace(whitespaceRe.ReplaceAllString(text, " "))

	return comment
}

// convertCommentsToTraditional converts each comment's Simplified-Chinese text to
// Traditional (AC #3), reusing the scraper's existing ChineseConverter.
func (s *Scraper) convertCommentsToTraditional(result *ReviewSummaryResult) {
	for i := range result.TopComments {
		text := result.TopComments[i].Text
		if text == "" {
			continue
		}
		traditional, err := s.converter.ConvertIfSimplified(text)
		if err != nil {
			s.logger.Warn("Failed to convert comment to Traditional",
				"error", err,
			)
			continue
		}
		result.TopComments[i].Text = traditional
	}
}

// ScrapeByURL scrapes metadata from a direct Douban URL
func (s *Scraper) ScrapeByURL(ctx context.Context, url string) (*DetailResult, error) {
	id := extractSubjectID(url)
	if id == "" {
		return nil, &ParseError{
			Field:  "url",
			Reason: "could not extract subject ID from URL: " + url,
		}
	}

	return s.ScrapeDetail(ctx, id)
}
