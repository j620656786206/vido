package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/sse"
)

// SSE event types for transcription progress (AC #6)
const (
	EventTranscriptionExtracting  sse.EventType = "transcription_extracting"
	EventTranscriptionProgress    sse.EventType = "transcription_progress"
	EventTranscriptionComplete    sse.EventType = "transcription_complete"
	EventTranscriptionFailed      sse.EventType = "transcription_failed"
	EventTranscriptionTranslating sse.EventType = "translation_progress"
)

// Transcription errors
var (
	ErrTranscriptionInProgress = errors.New("transcription already in progress for this media")
	ErrTranscriptionDisabled   = errors.New("transcription disabled: OpenAI API key not configured")
)

// TranscriptionResult holds the result of a transcription job.
type TranscriptionResult struct {
	JobID       string `json:"job_id"`
	MediaID     int64  `json:"media_id"`
	SRTPath     string `json:"srt_path"`
	ZhSRTPath   string `json:"zh_srt_path,omitempty"`
	Duration    string `json:"duration"`
	Error       string `json:"error,omitempty"`
}

// TranscriptionService orchestrates the audio extraction → Whisper transcription pipeline.
type TranscriptionService struct {
	audioExtractor     *AudioExtractorService
	whisperClient      *ai.WhisperClient
	translationService *TranslationService
	sseHub             *sse.Hub
	logger             *slog.Logger
	timeout            time.Duration

	mu         sync.Mutex
	inProgress map[int64]string // mediaID → jobID
}

// NewTranscriptionService creates a new TranscriptionService.
func NewTranscriptionService(
	audioExtractor *AudioExtractorService,
	whisperClient *ai.WhisperClient,
	sseHub *sse.Hub,
	logger *slog.Logger,
) *TranscriptionService {
	if logger == nil {
		logger = slog.Default()
	}
	return &TranscriptionService{
		audioExtractor: audioExtractor,
		whisperClient:  whisperClient,
		sseHub:         sseHub,
		logger:         logger.With("service", "transcription"),
		timeout:        5 * time.Minute,
		inProgress:     make(map[int64]string),
	}
}

// SetTranslationService sets the translation service for post-transcription translation.
// Kept as a setter to avoid changing the constructor signature (backward compatible).
func (s *TranscriptionService) SetTranslationService(ts *TranslationService) {
	s.translationService = ts
}

// IsAvailable returns true if both FFmpeg and Whisper API are configured.
func (s *TranscriptionService) IsAvailable() bool {
	return s.audioExtractor != nil && s.audioExtractor.IsAvailable() && s.whisperClient != nil
}

// IsInProgress returns true if a transcription is already running for the given media ID.
func (s *TranscriptionService) IsInProgress(mediaID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.inProgress[mediaID]
	return ok
}

// StartTranscription initiates an async transcription job for a media file.
// Returns a job ID immediately. Progress is reported via SSE events.
// If translate is true and a translation service is configured, the English SRT
// will be translated to Traditional Chinese after transcription (Story 9-2b).
func (s *TranscriptionService) StartTranscription(ctx context.Context, mediaID int64, filePath string, mediaDir string, opts ...TranscriptionOption) (string, error) {
	if !s.IsAvailable() {
		return "", ErrTranscriptionDisabled
	}

	cfg := &transcriptionConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	s.mu.Lock()
	if _, exists := s.inProgress[mediaID]; exists {
		s.mu.Unlock()
		return "", ErrTranscriptionInProgress
	}
	jobID := uuid.New().String()
	s.inProgress[mediaID] = jobID
	s.mu.Unlock()

	// Run transcription pipeline in background goroutine with timeout
	go func() {
		pipelineCtx, pipelineCancel := context.WithTimeout(context.Background(), s.timeout)
		defer pipelineCancel()
		s.runPipeline(pipelineCtx, jobID, mediaID, filePath, mediaDir, cfg.translate)
	}()

	return jobID, nil
}

// TranscriptionOption configures optional transcription behavior.
type TranscriptionOption func(*transcriptionConfig)

type transcriptionConfig struct {
	translate bool
}

// WithTranslation enables post-transcription translation to Traditional Chinese.
func WithTranslation() TranscriptionOption {
	return func(c *transcriptionConfig) {
		c.translate = true
	}
}

// runPipeline executes the full transcription pipeline:
// Extract audio → (optional chunk) → Whisper API → Merge SRT → Save → (optional) Translate
func (s *TranscriptionService) runPipeline(ctx context.Context, jobID string, mediaID int64, filePath string, mediaDir string, translate bool) {
	defer func() {
		s.mu.Lock()
		delete(s.inProgress, mediaID)
		s.mu.Unlock()
	}()

	startedAt := time.Now()

	// Phase 1: Extract audio
	s.broadcastEvent(EventTranscriptionExtracting, map[string]interface{}{
		"job_id":   jobID,
		"media_id": mediaID,
		"phase":    "extracting",
		"message":  "Extracting audio track from media file",
	})

	// List audio tracks and select English track
	tracks, err := s.audioExtractor.ListAudioTracks(ctx, filePath)
	if err != nil {
		s.failJob(jobID, mediaID, fmt.Sprintf("list audio tracks: %v", err))
		return
	}

	selectedTrack, err := SelectEnglishTrack(tracks)
	if err != nil {
		s.failJob(jobID, mediaID, fmt.Sprintf("select audio track: %v", err))
		return
	}

	s.logger.Info("audio track selected",
		"job_id", jobID,
		"track_index", selectedTrack.Index,
		"language", selectedTrack.Language,
	)

	// Extract audio to temp WAV
	audioPath, err := s.audioExtractor.ExtractAudio(ctx, filePath, selectedTrack.Index)
	if err != nil {
		s.failJob(jobID, mediaID, fmt.Sprintf("extract audio: %v", err))
		return
	}
	defer os.Remove(audioPath)

	// Phase 2: Transcribe
	s.broadcastEvent(EventTranscriptionProgress, map[string]interface{}{
		"job_id":   jobID,
		"media_id": mediaID,
		"phase":    "transcribing",
		"message":  "Transcribing audio with Whisper API",
	})

	srtContent, err := s.transcribeAudio(ctx, audioPath)
	if err != nil {
		s.failJob(jobID, mediaID, fmt.Sprintf("transcribe: %v", err))
		return
	}

	// Phase 3: Save SRT
	baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	srtPath := filepath.Join(mediaDir, baseName+".en.srt")

	if err := os.WriteFile(srtPath, []byte(srtContent), 0644); err != nil {
		s.failJob(jobID, mediaID, fmt.Sprintf("save SRT: %v", err))
		return
	}

	// Phase 3.5: Translate to Traditional Chinese (Story 9-2b)
	var zhSRTPath string
	if translate && s.translationService != nil && s.translationService.IsConfigured() {
		s.broadcastEvent(EventTranscriptionTranslating, map[string]interface{}{
			"job_id":     jobID,
			"media_id":   mediaID,
			"phase":      "translating",
			"percentage": 0,
			"message":    "Translating subtitles to Traditional Chinese",
		})

		zhPath, err := s.translateSRT(ctx, jobID, mediaID, srtContent, filePath, mediaDir)
		if err != nil {
			// AC #5: Translation failure is non-fatal — English SRT is still saved
			s.logger.Warn("translation failed — English SRT preserved",
				"job_id", jobID,
				"media_id", mediaID,
				"error", err,
			)
		} else {
			zhSRTPath = zhPath
		}
	}

	duration := time.Since(startedAt).Round(time.Second).String()

	s.logger.Info("transcription complete",
		"job_id", jobID,
		"media_id", mediaID,
		"srt_path", srtPath,
		"zh_srt_path", zhSRTPath,
		"duration", duration,
	)

	// Phase 4: Complete
	completeData := map[string]interface{}{
		"job_id":   jobID,
		"media_id": mediaID,
		"phase":    "complete",
		"srt_path": srtPath,
		"duration": duration,
		"message":  "Transcription complete",
	}
	if zhSRTPath != "" {
		completeData["zh_srt_path"] = zhSRTPath
	}
	s.broadcastEvent(EventTranscriptionComplete, completeData)
}

// transcribeAudio handles chunking and multi-part transcription for large files (AC #7).
func (s *TranscriptionService) transcribeAudio(ctx context.Context, audioPath string) (string, error) {
	needsChunk, err := ai.NeedsChunking(audioPath)
	if err != nil {
		return "", fmt.Errorf("check chunking: %w", err)
	}

	if !needsChunk {
		return s.whisperClient.Transcribe(ctx, audioPath)
	}

	// Split and transcribe chunks
	s.logger.Info("audio exceeds 25MB, splitting into chunks")
	chunks, err := ai.SplitAudioChunks(ctx, audioPath)
	if err != nil {
		return "", fmt.Errorf("split chunks: %w", err)
	}

	// Cleanup chunk files (skip first if it's the original)
	defer func() {
		for _, chunk := range chunks {
			if chunk != audioPath {
				os.Remove(chunk)
			}
		}
	}()

	var srtChunks []string
	for i, chunkPath := range chunks {
		s.logger.Info("transcribing chunk",
			"chunk", i+1,
			"total", len(chunks),
		)
		srt, err := s.whisperClient.Transcribe(ctx, chunkPath)
		if err != nil {
			return "", fmt.Errorf("transcribe chunk %d/%d: %w", i+1, len(chunks), err)
		}
		srtChunks = append(srtChunks, srt)
	}

	return ai.MergeSRTChunks(srtChunks, ai.WhisperChunkDuration), nil
}

// translateSRT translates English SRT content to Traditional Chinese and saves as .zh-Hant.srt.
// Returns the path to the translated SRT file.
func (s *TranscriptionService) translateSRT(ctx context.Context, jobID string, mediaID int64, srtContent string, filePath string, mediaDir string) (string, error) {
	// Parse SRT into translation blocks (inline to avoid circular import with subtitle pkg)
	blocks, err := ParseSRTToTranslationBlocks(srtContent)
	if err != nil {
		return "", fmt.Errorf("parse SRT for translation: %w", err)
	}

	if len(blocks) == 0 {
		return "", fmt.Errorf("no subtitle blocks to translate")
	}

	s.logger.Info("starting subtitle translation",
		"job_id", jobID,
		"media_id", mediaID,
		"block_count", len(blocks),
	)

	// Translate with progress reporting via SSE (AC #6)
	progressFn := func(pct float64) {
		s.broadcastEvent(EventTranscriptionTranslating, map[string]interface{}{
			"job_id":     jobID,
			"media_id":   mediaID,
			"phase":      "translating",
			"percentage": pct,
			"message":    fmt.Sprintf("Translating subtitles: %.0f%%", pct),
		})
	}

	translated, err := s.translationService.Translate(ctx, blocks, progressFn)
	if err != nil {
		return "", fmt.Errorf("translate: %w", err)
	}

	// Serialize back to SRT format
	zhSRT := serializeTranslationBlocksToSRT(translated)

	// Save zh-Hant SRT file
	baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	zhSRTPath := filepath.Join(mediaDir, baseName+".zh-Hant.srt")

	if err := os.WriteFile(zhSRTPath, []byte(zhSRT), 0644); err != nil {
		return "", fmt.Errorf("save zh-Hant SRT: %w", err)
	}

	s.logger.Info("subtitle translation complete",
		"job_id", jobID,
		"media_id", mediaID,
		"zh_srt_path", zhSRTPath,
		"block_count", len(translated),
	)

	return zhSRTPath, nil
}

// srtTimestampPattern matches SRT timestamp lines: 00:00:01,000 --> 00:00:04,000
// Mirrors subtitle.ParseSRT's validation to reject malformed timestamps.
var srtTimestampPattern = regexp.MustCompile(`^(\d{2}:\d{2}:\d{2},\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2},\d{3})`)

// ParseSRTToTranslationBlocks parses SRT content into TranslationBlocks.
// Inline SRT parser. services ↛ subtitle — see project-context.md Rule 19. Mirrors subtitle.ParseSRT validation.
func ParseSRTToTranslationBlocks(content string) ([]TranslationBlock, error) {
	if content == "" {
		return nil, nil
	}

	content = strings.TrimPrefix(content, "\xEF\xBB\xBF")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	lines := strings.Split(content, "\n")
	var blocks []TranslationBlock

	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			i++
			continue
		}

		// Parse block index
		index, err := strconv.Atoi(line)
		if err != nil {
			i++
			continue
		}
		i++

		// Parse timestamp line with regex validation
		if i >= len(lines) {
			break
		}
		tsLine := strings.TrimSpace(lines[i])
		matches := srtTimestampPattern.FindStringSubmatch(tsLine)
		if matches == nil {
			continue
		}
		start := matches[1]
		end := matches[2]
		i++

		// Collect text lines
		var textLines []string
		for i < len(lines) {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "" {
				i++
				break
			}
			textLines = append(textLines, trimmed)
			i++
		}

		blocks = append(blocks, TranslationBlock{
			Index: index,
			Start: start,
			End:   end,
			Text:  strings.Join(textLines, "\n"),
		})
	}

	return blocks, nil
}

// serializeTranslationBlocksToSRT converts TranslationBlocks back to SRT format.
func serializeTranslationBlocksToSRT(blocks []TranslationBlock) string {
	if len(blocks) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, b := range blocks {
		sb.WriteString(fmt.Sprintf("%d\n", b.Index))
		sb.WriteString(fmt.Sprintf("%s --> %s\n", b.Start, b.End))
		sb.WriteString(b.Text)
		sb.WriteString("\n")
		if i < len(blocks)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (s *TranscriptionService) failJob(jobID string, mediaID int64, errMsg string) {
	s.logger.Error("transcription failed",
		"job_id", jobID,
		"media_id", mediaID,
		"error", errMsg,
	)
	s.broadcastEvent(EventTranscriptionFailed, map[string]interface{}{
		"job_id":   jobID,
		"media_id": mediaID,
		"phase":    "failed",
		"error":    errMsg,
		"message":  "Transcription failed: " + errMsg,
	})
}

func (s *TranscriptionService) broadcastEvent(eventType sse.EventType, data interface{}) {
	if s.sseHub == nil {
		return
	}
	s.sseHub.Broadcast(sse.Event{
		ID:   uuid.New().String(),
		Type: eventType,
		Data: data,
	})
}
