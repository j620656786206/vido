package wikipedia

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"unicode"
)

// SearchOptions provides additional options for searching Wikipedia
type SearchOptions struct {
	// Limit is the maximum number of results to return (default: 5, max: 10)
	Limit int
	// MediaType filters results by media type
	MediaType MediaType
	// Year filters results by release year (if available)
	Year int
	// PreferTraditionalChinese prioritizes Traditional Chinese content
	PreferTraditionalChinese bool
}

// DefaultSearchOptions returns sensible defaults for search
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		Limit:                    5,
		PreferTraditionalChinese: true,
	}
}

// Searcher provides advanced search functionality for Wikipedia
type Searcher struct {
	client *Client
	logger *slog.Logger
}

// NewSearcher creates a new Wikipedia searcher
func NewSearcher(client *Client, logger *slog.Logger) *Searcher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Searcher{
		client: client,
		logger: logger,
	}
}

// Search performs an advanced search with result ranking
func (s *Searcher) Search(ctx context.Context, query string, opts SearchOptions) ([]RankedResult, error) {
	if opts.Limit <= 0 || opts.Limit > 10 {
		opts.Limit = 5
	}

	// Search Wikipedia with a larger limit to allow for ranking
	results, err := s.client.Search(ctx, query, opts.Limit*2)
	if err != nil {
		return nil, err
	}

	// Rank the results
	ranked := s.rankResults(query, results, opts)

	// Limit to requested number
	if len(ranked) > opts.Limit {
		ranked = ranked[:opts.Limit]
	}

	s.logger.Debug("Wikipedia search completed with ranking",
		"query", query,
		"total_results", len(results),
		"ranked_results", len(ranked),
	)

	return ranked, nil
}

// RankedResult is a search result with a confidence score
type RankedResult struct {
	SearchResult
	// Confidence is the match confidence (0-1 scale)
	Confidence float64
	// MatchType indicates how the result matched
	MatchType string
}

// rankResults assigns confidence scores and sorts results
func (s *Searcher) rankResults(query string, results []SearchResult, opts SearchOptions) []RankedResult {
	ranked := make([]RankedResult, 0, len(results))
	queryLower := strings.ToLower(query)
	queryNormalized := normalizeText(query)

	for _, result := range results {
		confidence := s.calculateConfidence(queryLower, queryNormalized, result, opts)
		matchType := s.determineMatchType(queryLower, queryNormalized, result)

		ranked = append(ranked, RankedResult{
			SearchResult: result,
			Confidence:   confidence,
			MatchType:    matchType,
		})
	}

	// Sort by confidence (descending)
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Confidence > ranked[j].Confidence
	})

	return ranked
}

// calculateConfidence calculates the match confidence score
func (s *Searcher) calculateConfidence(queryLower, queryNormalized string, result SearchResult, opts SearchOptions) float64 {
	titleLower := strings.ToLower(result.Title)
	titleNormalized := normalizeText(result.Title)
	snippetLower := strings.ToLower(result.Snippet)

	var confidence float64

	// Exact title match (highest confidence)
	if titleNormalized == queryNormalized {
		confidence = 1.0
	} else if titleLower == queryLower {
		confidence = 0.95
	} else if strings.Contains(titleNormalized, queryNormalized) {
		// Title contains query
		confidence = 0.8
	} else if strings.Contains(queryNormalized, titleNormalized) {
		// Query contains title (e.g., query is "寄生上流 電影" and title is "寄生上流")
		confidence = 0.75
	} else if strings.Contains(titleLower, queryLower) || strings.Contains(queryLower, titleLower) {
		confidence = 0.7
	} else if strings.Contains(snippetLower, queryLower) {
		// Query found in snippet
		confidence = 0.5
	} else {
		// Fuzzy match - check for common characters
		confidence = calculateCharacterOverlap(queryNormalized, titleNormalized)
	}

	// Boost for media type indicators in title
	if opts.MediaType != "" {
		if containsMediaTypeIndicator(result.Title, opts.MediaType) {
			confidence += 0.1
		}
	}

	// Boost for year match in snippet
	if opts.Year > 0 {
		yearStr := strings.Builder{}
		yearStr.WriteString(string(rune('0' + (opts.Year/1000)%10)))
		yearStr.WriteString(string(rune('0' + (opts.Year/100)%10)))
		yearStr.WriteString(string(rune('0' + (opts.Year/10)%10)))
		yearStr.WriteString(string(rune('0' + opts.Year%10)))
		if strings.Contains(snippetLower, yearStr.String()) {
			confidence += 0.05
		}
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// determineMatchType determines how the query matched the result
func (s *Searcher) determineMatchType(queryLower, queryNormalized string, result SearchResult) string {
	titleNormalized := normalizeText(result.Title)
	titleLower := strings.ToLower(result.Title)

	if titleNormalized == queryNormalized || titleLower == queryLower {
		return "exact"
	}
	if strings.Contains(titleNormalized, queryNormalized) || strings.Contains(titleLower, queryLower) {
		return "title_contains"
	}
	if strings.Contains(queryNormalized, titleNormalized) || strings.Contains(queryLower, titleLower) {
		return "query_contains"
	}
	if strings.Contains(strings.ToLower(result.Snippet), queryLower) {
		return "snippet"
	}
	return "fuzzy"
}

// normalizeText removes punctuation and normalizes whitespace
func normalizeText(text string) string {
	var result strings.Builder
	prevSpace := false

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			result.WriteRune(unicode.ToLower(r))
			prevSpace = false
		} else if unicode.IsSpace(r) && !prevSpace {
			result.WriteRune(' ')
			prevSpace = true
		}
	}

	return strings.TrimSpace(result.String())
}

// calculateCharacterOverlap calculates the overlap between two strings
func calculateCharacterOverlap(s1, s2 string) float64 {
	if len(s1) == 0 || len(s2) == 0 {
		return 0
	}

	chars1 := make(map[rune]int)
	chars2 := make(map[rune]int)

	for _, r := range s1 {
		chars1[r]++
	}
	for _, r := range s2 {
		chars2[r]++
	}

	overlap := 0
	total := 0

	for r, count := range chars1 {
		total += count
		if c2, ok := chars2[r]; ok {
			if count < c2 {
				overlap += count
			} else {
				overlap += c2
			}
		}
	}

	if total == 0 {
		return 0
	}

	return float64(overlap) / float64(total)
}

// containsMediaTypeIndicator checks if title contains media type indicators
func containsMediaTypeIndicator(title string, mediaType MediaType) bool {
	titleLower := strings.ToLower(title)

	switch mediaType {
	case MediaTypeMovie:
		return strings.Contains(titleLower, "電影") ||
			strings.Contains(titleLower, "film") ||
			strings.Contains(titleLower, "movie")
	case MediaTypeTV:
		return strings.Contains(titleLower, "電視劇") ||
			strings.Contains(titleLower, "電視節目") ||
			strings.Contains(titleLower, "tv") ||
			strings.Contains(titleLower, "series")
	case MediaTypeAnime:
		return strings.Contains(titleLower, "動畫") ||
			strings.Contains(titleLower, "anime") ||
			strings.Contains(titleLower, "動漫")
	}

	return false
}

// SearchByTitle searches for a specific title and returns the best match
func (s *Searcher) SearchByTitle(ctx context.Context, title string, mediaType MediaType) (*RankedResult, error) {
	opts := SearchOptions{
		Limit:                    5,
		MediaType:                mediaType,
		PreferTraditionalChinese: true,
	}

	results, err := s.Search(ctx, title, opts)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, &NotFoundError{Query: title}
	}

	// Return the best match
	return &results[0], nil
}
