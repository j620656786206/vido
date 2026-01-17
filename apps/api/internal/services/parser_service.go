package services

import (
	"log/slog"
	"time"

	"github.com/vido/api/internal/parser"
)

// ParserServiceInterface defines the contract for filename parsing services.
type ParserServiceInterface interface {
	// ParseFilename parses a single filename and returns metadata.
	ParseFilename(filename string) *parser.ParseResult

	// ParseBatch parses multiple filenames and returns results in the same order.
	ParseBatch(filenames []string) []*parser.ParseResult
}

// ParserService orchestrates filename parsing using movie and TV parsers.
type ParserService struct {
	movieParser *parser.MovieParser
	tvParser    *parser.TVParser
}

// NewParserService creates a new ParserService with default parsers.
func NewParserService() *ParserService {
	return &ParserService{
		movieParser: parser.NewMovieParser(),
		tvParser:    parser.NewTVParser(),
	}
}

// ParseFilename attempts to parse a filename and extract metadata.
// It tries TV show patterns first (more specific), then movie patterns.
// If neither works, it returns a result with ParseStatusNeedsAI.
func (s *ParserService) ParseFilename(filename string) *parser.ParseResult {
	start := time.Now()

	// Try TV show pattern first (more specific patterns)
	if s.tvParser.CanParse(filename) {
		result := s.tvParser.Parse(filename)
		if result.Status == parser.ParseStatusSuccess {
			result.Confidence = calculateConfidence(result)
			slog.Debug("Parsed as TV show",
				"filename", filename,
				"title", result.Title,
				"season", result.Season,
				"episode", result.Episode,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return result
		}
	}

	// Try movie pattern
	if s.movieParser.CanParse(filename) {
		result := s.movieParser.Parse(filename)
		if result.Status == parser.ParseStatusSuccess {
			result.Confidence = calculateConfidence(result)
			slog.Debug("Parsed as movie",
				"filename", filename,
				"title", result.Title,
				"year", result.Year,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return result
		}
	}

	// Failed to parse - needs AI
	slog.Info("Regex parsing failed, flagging for AI",
		"filename", filename,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return &parser.ParseResult{
		OriginalFilename: filename,
		Status:           parser.ParseStatusNeedsAI,
		MediaType:        parser.MediaTypeUnknown,
		Confidence:       0,
		ErrorMessage:     "Could not parse with standard patterns",
	}
}

// ParseBatch parses multiple filenames and returns results in the same order.
func (s *ParserService) ParseBatch(filenames []string) []*parser.ParseResult {
	results := make([]*parser.ParseResult, len(filenames))
	for i, filename := range filenames {
		results[i] = s.ParseFilename(filename)
	}
	return results
}

// calculateConfidence determines a confidence score based on how much metadata was extracted.
func calculateConfidence(result *parser.ParseResult) int {
	if result.Status != parser.ParseStatusSuccess {
		return 0
	}

	// Base confidence
	confidence := 50

	// Add confidence for each piece of metadata found
	if result.Title != "" {
		confidence += 10
	}
	if result.Year > 0 {
		confidence += 10
	}
	if result.Season > 0 || result.Episode > 0 {
		confidence += 10
	}
	if result.Quality != "" {
		confidence += 5
	}
	if result.Source != "" {
		confidence += 5
	}
	if result.VideoCodec != "" {
		confidence += 5
	}
	if result.ReleaseGroup != "" {
		confidence += 5
	}

	// Cap at 100
	if confidence > 100 {
		confidence = 100
	}

	return confidence
}
