package learning

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/agnivade/levenshtein"
)

// LearningRepositoryInterface defines the interface for learning pattern storage
type LearningRepositoryInterface interface {
	Save(ctx context.Context, mapping *FilenameMapping) error
	FindByID(ctx context.Context, id string) (*FilenameMapping, error)
	FindByExactPattern(ctx context.Context, pattern string) (*FilenameMapping, error)
	FindByFansubAndTitle(ctx context.Context, fansubGroup, titlePattern string) ([]*FilenameMapping, error)
	ListWithRegex(ctx context.Context) ([]*FilenameMapping, error)
	ListAll(ctx context.Context) ([]*FilenameMapping, error)
	Delete(ctx context.Context, id string) error
	IncrementUseCount(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
}

// MatchResult represents the result of a pattern match
type MatchResult struct {
	Pattern    *FilenameMapping
	Confidence float64
	MatchType  string // "exact", "pattern", "regex", "fuzzy"
}

// String returns a string representation of the match result
func (r *MatchResult) String() string {
	if r == nil || r.Pattern == nil {
		return "no match"
	}
	return fmt.Sprintf("Match[%s]: %s (confidence: %.2f, type: %s)",
		r.Pattern.ID, r.Pattern.TitlePattern, r.Confidence, r.MatchType)
}

// PatternMatcher finds matching patterns for new files
type PatternMatcher struct {
	repo      LearningRepositoryInterface
	extractor *PatternExtractor
	logger    *slog.Logger
}

// NewPatternMatcher creates a new pattern matcher
func NewPatternMatcher(repo LearningRepositoryInterface) *PatternMatcher {
	return &PatternMatcher{
		repo:      repo,
		extractor: NewPatternExtractor(),
		logger:    slog.Default(),
	}
}

// FindMatch attempts to find a matching pattern for the given filename
// It tries different matching strategies in order of confidence:
// 1. Exact match (confidence: 1.0)
// 2. Fansub group + title match (confidence: 0.95)
// 3. Regex pattern match (confidence: 0.9)
// 4. Fuzzy title match (confidence: 0.8-0.9)
func (m *PatternMatcher) FindMatch(ctx context.Context, filename string) (*MatchResult, error) {
	// 1. Try exact match first (highest confidence)
	if pattern, err := m.repo.FindByExactPattern(ctx, filename); err == nil && pattern != nil {
		return &MatchResult{
			Pattern:    pattern,
			Confidence: 1.0,
			MatchType:  "exact",
		}, nil
	}

	// 2. Extract pattern from filename
	extracted, err := m.extractor.Extract(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to extract pattern: %w", err)
	}

	// 3. Try fansub group + title match (high confidence)
	if extracted.FansubGroup != "" && extracted.TitlePattern != "" {
		patterns, err := m.repo.FindByFansubAndTitle(ctx, extracted.FansubGroup, extracted.TitlePattern)
		if err == nil && len(patterns) > 0 {
			return &MatchResult{
				Pattern:    patterns[0],
				Confidence: 0.95,
				MatchType:  "pattern",
			}, nil
		}
	}

	// 4. Try regex match
	allPatterns, err := m.repo.ListWithRegex(ctx)
	if err == nil {
		for _, p := range allPatterns {
			if p.PatternRegex != "" {
				re, err := regexp.Compile(p.PatternRegex)
				if err == nil && re.MatchString(filename) {
					return &MatchResult{
						Pattern:    p,
						Confidence: 0.9,
						MatchType:  "regex",
					}, nil
				}
			}
		}
	}

	// 5. Try fuzzy match (title only)
	if extracted.TitlePattern != "" {
		patterns, err := m.repo.ListAll(ctx)
		if err == nil {
			var bestMatch *FilenameMapping
			var bestSimilarity float64

			for _, p := range patterns {
				if p.TitlePattern == "" {
					continue
				}

				similarity := fuzzyMatch(extracted.TitlePattern, p.TitlePattern)
				if similarity > 0.8 && similarity > bestSimilarity {
					bestMatch = p
					bestSimilarity = similarity
				}
			}

			if bestMatch != nil {
				return &MatchResult{
					Pattern:    bestMatch,
					Confidence: bestSimilarity,
					MatchType:  "fuzzy",
				}, nil
			}
		}
	}

	// No match found
	return nil, nil
}

// fuzzyMatch calculates the similarity between two strings using Levenshtein distance
// Returns a value between 0.0 (completely different) and 1.0 (identical)
func fuzzyMatch(s1, s2 string) float64 {
	// Normalize to lowercase for case-insensitive comparison
	s1Lower := strings.ToLower(s1)
	s2Lower := strings.ToLower(s2)

	// If strings are identical, return 1.0
	if s1Lower == s2Lower {
		return 1.0
	}

	// Calculate Levenshtein distance
	distance := levenshtein.ComputeDistance(s1Lower, s2Lower)

	// Calculate max possible distance
	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}

	if maxLen == 0 {
		return 1.0 // Both strings are empty
	}

	// Convert distance to similarity (0.0 to 1.0)
	similarity := 1.0 - float64(distance)/float64(maxLen)

	// Ensure non-negative
	if similarity < 0 {
		similarity = 0
	}

	return similarity
}
