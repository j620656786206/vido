package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/parser"
)

// ParserServiceInterface defines the contract for filename parsing services.
type ParserServiceInterface interface {
	// ParseFilename parses a single filename and returns metadata.
	ParseFilename(filename string) *parser.ParseResult

	// ParseBatch parses multiple filenames and returns results in the same order.
	ParseBatch(filenames []string) []*parser.ParseResult

	// ParseFilenameWithContext parses a single filename with context support (for AI timeout).
	ParseFilenameWithContext(ctx context.Context, filename string) *parser.ParseResult
}

// ParserService orchestrates filename parsing using movie and TV parsers.
// It optionally integrates with AI parsing when regex fails.
type ParserService struct {
	movieParser *parser.MovieParser
	tvParser    *parser.TVParser
	aiService   AIServiceInterface
}

// NewParserService creates a new ParserService with default parsers.
func NewParserService() *ParserService {
	return &ParserService{
		movieParser: parser.NewMovieParser(),
		tvParser:    parser.NewTVParser(),
	}
}

// NewParserServiceWithAI creates a new ParserService with AI integration.
func NewParserServiceWithAI(aiService AIServiceInterface) *ParserService {
	return &ParserService{
		movieParser: parser.NewMovieParser(),
		tvParser:    parser.NewTVParser(),
		aiService:   aiService,
	}
}

// ParseFilename attempts to parse a filename and extract metadata.
// It tries TV show patterns first (more specific), then movie patterns.
// If neither works and AI is configured, it delegates to AI parsing.
// If AI is not configured, it returns ParseStatusNeedsAI.
func (s *ParserService) ParseFilename(filename string) *parser.ParseResult {
	return s.ParseFilenameWithContext(context.Background(), filename)
}

// ParseFilenameWithContext parses a filename with context support for AI timeout.
func (s *ParserService) ParseFilenameWithContext(ctx context.Context, filename string) *parser.ParseResult {
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

	// Regex failed - try AI parsing if configured
	if s.aiService != nil && s.aiService.IsConfigured() {
		slog.Info("Regex parsing failed, delegating to AI",
			"filename", filename,
			"duration_ms", time.Since(start).Milliseconds(),
		)

		aiResult, err := s.aiService.ParseFilename(ctx, filename)
		if err != nil {
			slog.Warn("AI parsing failed",
				"filename", filename,
				"error", err,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			// Return needs_ai status on AI failure
			return &parser.ParseResult{
				OriginalFilename: filename,
				Status:           parser.ParseStatusNeedsAI,
				MediaType:        parser.MediaTypeUnknown,
				Confidence:       0,
				ErrorMessage:     "AI parsing failed: " + err.Error(),
			}
		}

		// Convert AI response to ParseResult
		return convertAIResponseToParseResult(filename, aiResult, start)
	}

	// No AI configured - flag for manual review
	slog.Info("Regex parsing failed, AI not configured",
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

// convertAIResponseToParseResult converts an AI parsing response to ParseResult.
func convertAIResponseToParseResult(filename string, aiResponse *ai.ParseResponse, start time.Time) *parser.ParseResult {
	mediaType := parser.MediaTypeUnknown
	if aiResponse.MediaType == "movie" {
		mediaType = parser.MediaTypeMovie
	} else if aiResponse.MediaType == "tv" {
		mediaType = parser.MediaTypeTVShow
	}

	result := &parser.ParseResult{
		OriginalFilename: filename,
		Status:           parser.ParseStatusSuccess,
		MediaType:        mediaType,
		Title:            aiResponse.Title,
		Year:             aiResponse.Year,
		Season:           aiResponse.Season,
		Episode:          aiResponse.Episode,
		Quality:          aiResponse.Quality,
		ReleaseGroup:     aiResponse.FansubGroup,
		Confidence:       int(aiResponse.Confidence * 100),
	}

	slog.Info("Parsed with AI",
		"filename", filename,
		"title", result.Title,
		"media_type", result.MediaType,
		"confidence", result.Confidence,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return result
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
