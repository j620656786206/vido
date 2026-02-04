package services

import (
	"context"
	"log/slog"
	"strings"
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
// It optionally integrates with AI parsing when regex fails, and can check
// learned patterns for previously corrected filenames.
type ParserService struct {
	movieParser     *parser.MovieParser
	tvParser        *parser.TVParser
	aiService       AIServiceInterface
	learningService LearningServiceInterface
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

// NewParserServiceWithLearning creates a new ParserService with AI and learning integration.
func NewParserServiceWithLearning(aiService AIServiceInterface, learningService LearningServiceInterface) *ParserService {
	return &ParserService{
		movieParser:     parser.NewMovieParser(),
		tvParser:        parser.NewTVParser(),
		aiService:       aiService,
		learningService: learningService,
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

	// 1. Check learned patterns first (highest priority)
	if s.learningService != nil {
		result, err := s.parseWithLearnedPattern(ctx, filename, start)
		if err != nil {
			slog.Warn("Error checking learned patterns", "error", err, "filename", filename)
			// Continue with other parsing methods
		} else if result != nil {
			return result
		}
	}

	// Check if this looks like a fansub filename
	fansubDetection := ai.DetectFansub(filename)

	// If fansub detected with high confidence, skip regex and go straight to AI fansub parser
	if fansubDetection.IsFansub && fansubDetection.Confidence >= 0.6 {
		return s.parseWithFansubAI(ctx, filename, fansubDetection, start)
	}

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

	// Regex failed - check if it's a fansub with lower confidence
	if fansubDetection.IsFansub {
		return s.parseWithFansubAI(ctx, filename, fansubDetection, start)
	}

	// Try generic AI parsing if configured
	if s.aiService != nil && s.aiService.IsConfigured() {
		slog.Info("Regex parsing failed, delegating to generic AI",
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
		return convertAIResponseToParseResult(filename, aiResult, "ai", s.aiService.GetProviderName(), start)
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

// parseWithFansubAI handles fansub filename parsing using the specialized AI fansub parser.
func (s *ParserService) parseWithFansubAI(ctx context.Context, filename string, detection *ai.FansubDetectionResult, start time.Time) *parser.ParseResult {
	if s.aiService == nil || !s.aiService.IsConfigured() {
		slog.Info("Fansub detected but AI not configured",
			"filename", filename,
			"bracket_type", detection.BracketType,
			"group_name", detection.GroupName,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return &parser.ParseResult{
			OriginalFilename: filename,
			Status:           parser.ParseStatusNeedsAI,
			MediaType:        parser.MediaTypeUnknown,
			ReleaseGroup:     detection.GroupName,
			Confidence:       0,
			ErrorMessage:     "Fansub filename detected, AI required for parsing",
		}
	}

	slog.Info("Fansub detected, using specialized AI parser",
		"filename", filename,
		"bracket_type", detection.BracketType,
		"group_name", detection.GroupName,
		"detection_confidence", detection.Confidence,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	aiResult, err := s.aiService.ParseFansubFilename(ctx, filename)
	if err != nil {
		slog.Warn("AI fansub parsing failed",
			"filename", filename,
			"error", err,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return &parser.ParseResult{
			OriginalFilename: filename,
			Status:           parser.ParseStatusNeedsAI,
			MediaType:        parser.MediaTypeUnknown,
			ReleaseGroup:     detection.GroupName,
			Confidence:       0,
			ErrorMessage:     "AI fansub parsing failed: " + err.Error(),
		}
	}

	// Convert AI response to ParseResult with fansub source
	return convertAIResponseToParseResult(filename, aiResult, "ai_fansub", s.aiService.GetProviderName(), start)
}

// convertAIResponseToParseResult converts an AI parsing response to ParseResult.
// The source parameter indicates the AI parsing method used ("ai" or "ai_fansub").
// The providerName parameter identifies which AI provider was used.
func convertAIResponseToParseResult(filename string, aiResponse *ai.ParseResponse, source string, providerName string, start time.Time) *parser.ParseResult {
	duration := time.Since(start)

	mediaType := parser.MediaTypeUnknown
	if aiResponse.MediaType == "movie" {
		mediaType = parser.MediaTypeMovie
	} else if aiResponse.MediaType == "tv" {
		mediaType = parser.MediaTypeTVShow
	}

	// Determine metadata source
	metadataSource := parser.MetadataSourceAI
	if source == "ai_fansub" {
		metadataSource = parser.MetadataSourceAIFansub
	}

	result := &parser.ParseResult{
		OriginalFilename: filename,
		Status:           parser.ParseStatusSuccess,
		MediaType:        mediaType,
		Title:            aiResponse.Title,
		Year:             aiResponse.Year,
		Season:           aiResponse.Season,
		Episode:          aiResponse.Episode,
		Quality:          normalizeQuality(aiResponse.Quality),
		Source:           normalizeSource(aiResponse.Source),
		VideoCodec:       normalizeCodec(aiResponse.Codec),
		ReleaseGroup:     aiResponse.FansubGroup,
		Language:         aiResponse.Language,
		Confidence:       int(aiResponse.Confidence * 100),
		MetadataSource:   metadataSource,
		ParseDurationMs:  duration.Milliseconds(),
		AIProvider:       providerName,
	}

	slog.Info("Parsed with AI",
		"filename", filename,
		"title", result.Title,
		"media_type", result.MediaType,
		"source_method", source,
		"fansub_group", result.ReleaseGroup,
		"language", result.Language,
		"confidence", result.Confidence,
		"duration_ms", duration.Milliseconds(),
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

// normalizeQuality converts various quality representations to standard values.
// Handles formats like "1080p", "1920x1080", "4K", "2160p", etc.
// Returns lowercase standard format (e.g., "1080p", "720p", "2160p").
func normalizeQuality(quality string) string {
	if quality == "" {
		return ""
	}

	quality = strings.ToLower(quality)

	// Handle dimension formats like "1920x1080" and various quality representations
	switch {
	case strings.Contains(quality, "2160") || strings.Contains(quality, "4k") || strings.Contains(quality, "uhd"):
		return "2160p"
	case strings.Contains(quality, "1920") || strings.Contains(quality, "1080"):
		return "1080p"
	case strings.Contains(quality, "1280") || strings.Contains(quality, "720"):
		return "720p"
	case strings.Contains(quality, "480"):
		return "480p"
	case strings.Contains(quality, "360"):
		return "360p"
	default:
		// Return lowercase for consistency
		return quality
	}
}

// normalizeSource converts various source representations to standard values.
// Handles formats like "BD", "Blu-ray", "BluRay", "WEB-DL", etc.
func normalizeSource(source string) string {
	if source == "" {
		return ""
	}

	source = strings.ToLower(source)

	switch {
	case strings.Contains(source, "bd") || strings.Contains(source, "blu"):
		return "BD"
	case strings.Contains(source, "web"):
		return "WEB"
	case strings.Contains(source, "hdtv") || strings.Contains(source, "tv"):
		return "TV"
	case strings.Contains(source, "dvd"):
		return "DVD"
	default:
		return strings.ToUpper(source)
	}
}

// normalizeCodec converts various codec representations to standard values.
// Handles formats like "x264", "h.264", "x265", "HEVC", etc.
func normalizeCodec(codec string) string {
	if codec == "" {
		return ""
	}

	codec = strings.ToLower(codec)

	switch {
	case strings.Contains(codec, "x265") || strings.Contains(codec, "hevc") || strings.Contains(codec, "h.265") || strings.Contains(codec, "h265"):
		return "x265"
	case strings.Contains(codec, "x264") || strings.Contains(codec, "avc") || strings.Contains(codec, "h.264") || strings.Contains(codec, "h264"):
		return "x264"
	case strings.Contains(codec, "av1"):
		return "AV1"
	case strings.Contains(codec, "vp9"):
		return "VP9"
	case strings.Contains(codec, "aac"):
		return "AAC"
	case strings.Contains(codec, "flac"):
		return "FLAC"
	case strings.Contains(codec, "dts"):
		return "DTS"
	default:
		return strings.ToUpper(codec)
	}
}

// parseWithLearnedPattern checks if a learned pattern matches the filename.
// Returns the parse result if a match is found with sufficient confidence.
func (s *ParserService) parseWithLearnedPattern(ctx context.Context, filename string, start time.Time) (*parser.ParseResult, error) {
	match, err := s.learningService.FindMatchingPattern(ctx, filename)
	if err != nil {
		return nil, err
	}

	if match == nil {
		return nil, nil
	}

	// Require minimum confidence for learned pattern match
	if match.Confidence < 0.8 {
		slog.Debug("Learned pattern match below threshold",
			"filename", filename,
			"patternId", match.Pattern.ID,
			"confidence", match.Confidence,
		)
		return nil, nil
	}

	slog.Info("Applied learned pattern",
		"filename", filename,
		"patternId", match.Pattern.ID,
		"pattern", match.Pattern.Pattern,
		"confidence", match.Confidence,
		"matchType", match.MatchType,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	// Increment use count for the pattern
	if err := s.learningService.ApplyPattern(ctx, match.Pattern.ID); err != nil {
		slog.Warn("Failed to increment pattern use count", "error", err, "patternId", match.Pattern.ID)
		// Continue anyway - this is not critical
	}

	// Determine media type from the pattern
	mediaType := parser.MediaTypeUnknown
	if match.Pattern.MetadataType == "movie" {
		mediaType = parser.MediaTypeMovie
	} else if match.Pattern.MetadataType == "series" {
		mediaType = parser.MediaTypeTVShow
	}

	return &parser.ParseResult{
		OriginalFilename:  filename,
		Status:            parser.ParseStatusSuccess,
		MediaType:         mediaType,
		Title:             match.Pattern.TitlePattern,
		ReleaseGroup:      match.Pattern.FansubGroup,
		Confidence:        int(match.Confidence * 100),
		MetadataSource:    parser.MetadataSourceLearned,
		ParseDurationMs:   time.Since(start).Milliseconds(),
		LearnedPatternID:  match.Pattern.ID,
		LearnedTmdbID:     match.Pattern.TmdbID,
		LearnedMetadataID: match.Pattern.MetadataID,
	}, nil
}
