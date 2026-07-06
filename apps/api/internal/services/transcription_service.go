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
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/sse"
)

// SubtitleStatusWriter persists generation success to the movie row (Story
// 9R-16 AC 12). Narrow on purpose (Rule 11) — *repository.MovieRepository
// satisfies it; main.go injects it. Without this writeback the missing-scope
// batch enumeration never shrinks and poster badges stay 缺字幕 until a rescan.
type SubtitleStatusWriter interface {
	UpdateSubtitleStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error
}

// OpenCCConverter is the Simplified→Traditional safety net applied after LLM
// translation (Story 9R-10). Defined here so the service does not import the
// subtitle package (Rule 19); *subtitle.Converter satisfies it structurally.
type OpenCCConverter interface {
	ConvertS2TWP(content []byte) ([]byte, error)
	IsAvailable() bool
}

// SubtitlePlacer writes the final subtitle atomically (backup + correct
// filename) — the "place" stage of the Route C pipeline (Story 9R-10). Primitive
// params keep the subtitle package out of services (Rule 19); main.go adapts
// *subtitle.Placer. Returns the written path.
type SubtitlePlacer interface {
	PlaceSubtitle(mediaFilePath string, subtitleData []byte, language, format string) (string, error)
}

// SSE event types for transcription progress (AC #6).
// [@contract-v1] (stamped by 9R-18, first formalization): every payload's
// `media_id` is the movie row id — a UUID STRING, never numeric. Consumers:
// slice-1 useGenerationProgress + the ux3-subtitle-v2-batch branch.
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
	JobID     string `json:"job_id"`
	MediaID   string `json:"media_id"`
	SRTPath   string `json:"srt_path"`
	ZhSRTPath string `json:"zh_srt_path,omitempty"`
	Duration  string `json:"duration"`
	Error     string `json:"error,omitempty"`
}

// TranscriptionService orchestrates the audio extraction → Whisper transcription pipeline.
type TranscriptionService struct {
	audioExtractor     *AudioExtractorService
	asr                ai.ASRProvider
	translationService *TranslationService
	sseHub             *sse.Hub
	logger             *slog.Logger
	timeout            time.Duration
	runBudgetUSD       float64 // 9R-11: per-run AI cost ceiling (0 = unlimited)

	// 9R-10 pipeline dependencies (all optional / nil-safe).
	glossaryRepo repository.GlossaryRepositoryInterface // per-show glossary (9R-6/7)
	opencc       OpenCCConverter                        // s2twp safety net
	placer       SubtitlePlacer                         // atomic place + backup

	// 9R-16 AC 12: generation-success writeback (optional / nil-safe).
	subtitleWriter SubtitleStatusWriter

	mu         sync.Mutex
	inProgress map[string]string // mediaID (UUID string, 9R-18) → jobID
}

// NewTranscriptionService creates a new TranscriptionService.
func NewTranscriptionService(
	audioExtractor *AudioExtractorService,
	asr ai.ASRProvider,
	sseHub *sse.Hub,
	logger *slog.Logger,
) *TranscriptionService {
	if logger == nil {
		logger = slog.Default()
	}
	return &TranscriptionService{
		audioExtractor: audioExtractor,
		asr:            asr,
		sseHub:         sseHub,
		logger:         logger.With("service", "transcription"),
		timeout:        5 * time.Minute,
		inProgress:     make(map[string]string),
	}
}

// SetTranslationService sets the translation service for post-transcription translation.
// Kept as a setter to avoid changing the constructor signature (backward compatible).
func (s *TranscriptionService) SetTranslationService(ts *TranslationService) {
	s.translationService = ts
}

// SetRunBudgetUSD sets the per-run AI cost ceiling (Story 9R-11). A run that
// reaches it stops making further ASR/LLM calls. 0 = unlimited (metering only).
func (s *TranscriptionService) SetRunBudgetUSD(usd float64) {
	s.runBudgetUSD = usd
}

// SetGlossaryRepository wires the per-show glossary (Story 9R-10). When set,
// translation is glossary-aware (proper nouns render consistently). Nil-safe.
func (s *TranscriptionService) SetGlossaryRepository(repo repository.GlossaryRepositoryInterface) {
	s.glossaryRepo = repo
}

// SetOpenCCConverter wires the Simplified→Traditional safety net (Story 9R-10).
func (s *TranscriptionService) SetOpenCCConverter(c OpenCCConverter) {
	s.opencc = c
}

// SetPlacer wires atomic subtitle placement + backup (Story 9R-10).
func (s *TranscriptionService) SetPlacer(p SubtitlePlacer) {
	s.placer = p
}

// SetSubtitleStatusWriter wires the generation-success DB writeback (Story
// 9R-16 AC 12). Nil-safe: when unset, generation places files but persists
// nothing (pre-9R-16 behavior).
func (s *TranscriptionService) SetSubtitleStatusWriter(w SubtitleStatusWriter) {
	s.subtitleWriter = w
}

// loadGlossary returns the per-show glossary as translation pairs, or nil when
// no repo is wired or the lookup fails (fail-soft — a glossary miss must never
// block generation). Uses ALL terms (confirmed + auto-mined) for maximum
// intra-run consistency; the F6 review UI lets users correct mistakes.
func (s *TranscriptionService) loadGlossary(ctx context.Context, mediaID string) []GlossaryPair {
	if s.glossaryRepo == nil {
		return nil
	}
	m, err := s.glossaryRepo.LookupByMedia(ctx, mediaID, false)
	if err != nil {
		s.logger.Warn("glossary lookup failed — translating without glossary",
			"media_id", mediaID, "error", err)
		return nil
	}
	if len(m) == 0 {
		return nil
	}
	pairs := make([]GlossaryPair, 0, len(m))
	for src, zh := range m {
		pairs = append(pairs, GlossaryPair{Source: src, Target: zh})
	}
	return pairs
}

// IsAvailable returns true if both FFmpeg and Whisper API are configured.
func (s *TranscriptionService) IsAvailable() bool {
	return s.audioExtractor != nil && s.audioExtractor.IsAvailable() && s.asr != nil
}

// IsInProgress returns true if a transcription is already running for the given media ID.
func (s *TranscriptionService) IsInProgress(mediaID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.inProgress[mediaID]
	return ok
}

// StartTranscription initiates an async transcription job for a media file.
// Returns a job ID immediately. Progress is reported via SSE events.
// If translate is true and a translation service is configured, the English SRT
// will be translated to Traditional Chinese after transcription (Story 9-2b).
func (s *TranscriptionService) StartTranscription(ctx context.Context, mediaID string, filePath string, mediaDir string, opts ...TranscriptionOption) (string, error) {
	if !s.IsAvailable() {
		return "", ErrTranscriptionDisabled
	}

	cfg := &transcriptionConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	jobID, err := s.acquireJob(mediaID)
	if err != nil {
		return "", err
	}

	// Run transcription pipeline in background goroutine with timeout.
	// Deliberately detached from the request ctx (context.Background()) so the
	// job outlives the HTTP request. Fire-and-forget: the pipeline error is
	// intentionally discarded here — every failure path already reports via
	// failJob SSE (9R-16 AC 6a ruling; sync callers use RunTranscription).
	go func() {
		pipelineCtx, pipelineCancel := context.WithTimeout(context.Background(), s.timeout)
		defer pipelineCancel()
		_ = s.runPipeline(pipelineCtx, jobID, mediaID, filePath, mediaDir, cfg.translate)
	}()

	return jobID, nil
}

// RunTranscription is the SYNCHRONOUS pipeline entry (Story 9R-16 AC 6a): it
// shares the same per-media single-flight map as StartTranscription, runs the
// pipeline inline, and RETURNS the pipeline error (the async path reports via
// failJob SSE only). ⚠️ The timeout derives from the CALLER's ctx — NOT the
// async path's context.Background() detach — so a batch's shared ai.Budget
// (a ctx value) and cancel propagation flow through.
func (s *TranscriptionService) RunTranscription(ctx context.Context, mediaID string, filePath string, mediaDir string, opts ...TranscriptionOption) error {
	if !s.IsAvailable() {
		return ErrTranscriptionDisabled
	}

	cfg := &transcriptionConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	jobID, err := s.acquireJob(mediaID)
	if err != nil {
		return err
	}

	pipelineCtx, pipelineCancel := context.WithTimeout(ctx, s.timeout)
	defer pipelineCancel()
	return s.runPipeline(pipelineCtx, jobID, mediaID, filePath, mediaDir, cfg.translate)
}

// resolveBudget returns the ctx-attached ai.Budget when one is present (9R-16
// AC 6b — a generation batch attaches ONE shared Budget so the whole batch
// spends from one envelope), else creates the per-run budget as before (9R-11)
// and attaches it.
func (s *TranscriptionService) resolveBudget(ctx context.Context) (*ai.Budget, context.Context) {
	if b := ai.BudgetFromContext(ctx); b != nil {
		return b, ctx
	}
	b := ai.NewBudget(s.runBudgetUSD)
	return b, ai.WithBudget(ctx, b)
}

// acquireJob registers a media ID in the single-flight map shared by the async
// and sync entries, returning the new job ID or ErrTranscriptionInProgress.
// runPipeline's deferred cleanup releases the slot.
func (s *TranscriptionService) acquireJob(mediaID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.inProgress[mediaID]; exists {
		return "", ErrTranscriptionInProgress
	}
	jobID := uuid.New().String()
	s.inProgress[mediaID] = jobID
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
// It reports failures via failJob SSE AND returns the error so the synchronous
// entry (RunTranscription, 9R-16) can propagate it; the async entry discards it.
func (s *TranscriptionService) runPipeline(ctx context.Context, jobID string, mediaID string, filePath string, mediaDir string, translate bool) error {
	defer func() {
		s.mu.Lock()
		delete(s.inProgress, mediaID)
		s.mu.Unlock()
	}()

	startedAt := time.Now()

	// 9R-11: one per-run budget spans BOTH transcription and translation of
	// this media so ASR + LLM share the ceiling; logged at the end.
	budget, ctx := s.resolveBudget(ctx)
	defer func() {
		snap := budget.Snapshot()
		s.logger.Info("transcription run AI usage",
			"job_id", jobID, "media_id", mediaID,
			"spent_usd", snap.SpentUSD, "budget_usd", snap.BudgetUSD,
			"llm_input_tokens", snap.InputTokens, "llm_output_tokens", snap.OutputTokens,
			"llm_calls", snap.LLMCalls, "asr_seconds", snap.ASRSeconds, "asr_calls", snap.ASRCalls,
		)
	}()

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
		return fmt.Errorf("list audio tracks: %w", err)
	}

	selectedTrack, err := SelectEnglishTrack(tracks)
	if err != nil {
		s.failJob(jobID, mediaID, fmt.Sprintf("select audio track: %v", err))
		return fmt.Errorf("select audio track: %w", err)
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
		return fmt.Errorf("extract audio: %w", err)
	}
	defer os.Remove(audioPath)

	// Phase 2: Transcribe
	s.broadcastEvent(EventTranscriptionProgress, map[string]interface{}{
		"job_id":   jobID,
		"media_id": mediaID,
		"phase":    "transcribing",
		"message":  "Transcribing audio with Whisper API",
	})

	srtContent, err := s.transcribeAudio(ctx, audioPath, WhisperLanguageFromTrack(selectedTrack.Language))
	if err != nil {
		s.failJob(jobID, mediaID, fmt.Sprintf("transcribe: %v", err))
		return fmt.Errorf("transcribe: %w", err)
	}

	// Phase 3: Save SRT
	baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	srtPath := filepath.Join(mediaDir, baseName+".en.srt")

	if err := os.WriteFile(srtPath, []byte(srtContent), 0644); err != nil {
		s.failJob(jobID, mediaID, fmt.Sprintf("save SRT: %v", err))
		return fmt.Errorf("save SRT: %w", err)
	}

	// Phase 3.5: Translate to Traditional Chinese (Story 9-2b) + persist
	// generation success (Story 9R-16 AC 12).
	zhSRTPath, err := s.translateAndPersist(ctx, jobID, mediaID, srtContent, filePath, mediaDir, translate)
	if err != nil {
		s.failJob(jobID, mediaID, err.Error())
		return err
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
	return nil
}

// translateAndPersist runs the optional translate phase and the 9R-16 AC 12
// generation-success writeback. Returns the zh-Hant path ("" for en-only runs).
// Error semantics (ruled in 9R-16 AC 6c/12):
//   - ordinary translate failures are NON-FATAL (English SRT preserved,
//     deliberate swallow — logged, zh path stays empty, no writeback);
//   - ai.ErrBudgetExceeded MUST propagate so a batch can pause mid-item;
//   - a writeback failure propagates (Rule 13) — reporting success while the
//     library row still says 缺字幕 would break the batch-enumeration guarantee.
func (s *TranscriptionService) translateAndPersist(ctx context.Context, jobID string, mediaID string, srtContent, filePath, mediaDir string, translate bool) (string, error) {
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
			// 9R-16 AC 6c: the budget sentinel MUST propagate — a mid-translate
			// ceiling hit must pause the batch, not count the item as success
			// (the English SRT stays on disk either way).
			if errors.Is(err, ai.ErrBudgetExceeded) {
				return "", fmt.Errorf("translate: %w", err)
			}
			// AC #5 (9-2b): other translation failures are non-fatal — English
			// SRT is still saved. Deliberate swallow, ruled in 9R-16 AC 6c.
			s.logger.Warn("translation failed — English SRT preserved",
				"job_id", jobID,
				"media_id", mediaID,
				"error", err,
			)
		} else {
			zhSRTPath = zhPath
		}
	}

	// 9R-16 AC 12: persist generation success — the resume enabler + badge
	// truth. Only after a successful zh-Hant place (en-only runs write
	// nothing; failed runs never reach here).
	if zhSRTPath != "" && s.subtitleWriter != nil {
		if werr := s.subtitleWriter.UpdateSubtitleStatus(ctx, mediaID,
			models.SubtitleStatusFound, zhSRTPath, "zh-Hant", 0); werr != nil {
			return "", fmt.Errorf("update subtitle status: %w", werr)
		}
	}

	return zhSRTPath, nil
}

// WhisperLanguageFromTrack maps an ffprobe audio-track language tag (ISO-639-2,
// e.g. "eng") to the ISO-639-1 hint Whisper expects (9R-2). Returns "" (Whisper
// auto-detect) ONLY when the tag is missing/und/unknown — pinning a wrong
// language is worse than auto-detecting.
func WhisperLanguageFromTrack(trackLang string) string {
	lang := strings.ToLower(strings.TrimSpace(trackLang))
	switch lang {
	case "", "und":
		return ""
	}
	if len(lang) == 2 {
		return lang
	}
	iso3to1 := map[string]string{
		"eng": "en",
		"jpn": "ja",
		"chi": "zh", "zho": "zh",
		"kor": "ko",
		"fra": "fr", "fre": "fr",
		"deu": "de", "ger": "de",
		"spa": "es",
		"ita": "it",
		"rus": "ru",
		"por": "pt",
		"tha": "th",
		"vie": "vi",
		"ara": "ar",
		"hin": "hi",
		"nld": "nl", "dut": "nl",
		"pol": "pl",
		"tur": "tr",
		"swe": "sv",
		"nor": "no",
		"dan": "da",
		"fin": "fi",
		"ind": "id",
		"msa": "ms", "may": "ms",
	}
	if iso1, ok := iso3to1[lang]; ok {
		return iso1
	}
	return ""
}

// transcribeAudio handles chunking and multi-part transcription for large files (AC #7).
// lang is the ISO-639-1 hint derived from the selected audio track ("" = auto-detect).
func (s *TranscriptionService) transcribeAudio(ctx context.Context, audioPath, lang string) (string, error) {
	needsChunk, err := ai.NeedsChunking(audioPath)
	if err != nil {
		return "", fmt.Errorf("check chunking: %w", err)
	}

	if !needsChunk {
		return s.asr.TranscribeWithLanguage(ctx, audioPath, lang)
	}

	// Split and transcribe chunks
	s.logger.Info("audio exceeds 25MB, splitting into chunks")
	chunks, chunkSeconds, err := ai.SplitAudioChunks(ctx, audioPath)
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
		srt, err := s.asr.TranscribeWithLanguage(ctx, chunkPath, lang)
		if err != nil {
			return "", fmt.Errorf("transcribe chunk %d/%d: %w", i+1, len(chunks), err)
		}
		srtChunks = append(srtChunks, srt)
	}

	return ai.MergeSRTChunks(srtChunks, chunkSeconds), nil
}

// translateSRT translates English SRT content to Traditional Chinese and saves as .zh-Hant.srt.
// Returns the path to the translated SRT file.
func (s *TranscriptionService) translateSRT(ctx context.Context, jobID string, mediaID string, srtContent string, filePath string, mediaDir string) (string, error) {
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

	// 9R-10: glossary-aware translation — proper nouns render consistently
	// across the whole subtitle and across runs (keystone payoff).
	glossary := s.loadGlossary(ctx, mediaID)
	if len(glossary) > 0 {
		s.logger.Info("translating with per-show glossary", "media_id", mediaID, "term_count", len(glossary))
	}
	translated, err := s.translationService.TranslateWithGlossary(ctx, blocks, glossary, progressFn)
	if err != nil {
		return "", fmt.Errorf("translate: %w", err)
	}

	// Serialize back to SRT format.
	zhSRT := serializeTranslationBlocksToSRT(translated)

	// 9R-10: OpenCC s2twp safety net — guarantee Traditional output even if the
	// LLM slips a Simplified character through. Fail-soft: on converter error,
	// keep the LLM output rather than losing the subtitle.
	if s.opencc != nil && s.opencc.IsAvailable() {
		if converted, cerr := s.opencc.ConvertS2TWP([]byte(zhSRT)); cerr == nil {
			zhSRT = string(converted)
		} else {
			s.logger.Warn("OpenCC safety-net conversion failed — keeping LLM output",
				"media_id", mediaID, "error", cerr)
		}
	}

	// 9R-10: place the subtitle. Prefer the injected Placer (atomic write +
	// .bak backup + normalized filename); fall back to a direct write when no
	// placer is wired so the pipeline still functions.
	var zhSRTPath string
	if s.placer != nil {
		zhSRTPath, err = s.placer.PlaceSubtitle(filePath, []byte(zhSRT), "zh-Hant", "srt")
		if err != nil {
			return "", fmt.Errorf("place zh-Hant SRT: %w", err)
		}
	} else {
		baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
		zhSRTPath = filepath.Join(mediaDir, baseName+".zh-Hant.srt")
		if err := os.WriteFile(zhSRTPath, []byte(zhSRT), 0644); err != nil {
			return "", fmt.Errorf("save zh-Hant SRT: %w", err)
		}
	}

	s.logger.Info("subtitle translation complete",
		"job_id", jobID,
		"media_id", mediaID,
		"zh_srt_path", zhSRTPath,
		"block_count", len(translated),
		"glossary_terms", len(glossary),
	)

	return zhSRTPath, nil
}

// srtTimestampPattern matches SRT timestamp lines: 00:00:01,000 --> 00:00:04,000
// Mirrors subtitle.ParseSRT's validation to reject malformed timestamps.
var srtTimestampPattern = regexp.MustCompile(`^(\d{2}:\d{2}:\d{2},\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2},\d{3})`)

// ParseSRTToTranslationBlocks parses SRT content into TranslationBlocks.
// Inline SRT parser. services ↛ subtitle — see project-context.md Rule 19.
// Mirrors subtitle.ParseSRT validation. Exported (rather than kept private)
// only so the external-test-package parity check in srt_parity_test.go can
// call it cross-package; external runtime callers should use subtitle.ParseSRT.
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

func (s *TranscriptionService) failJob(jobID string, mediaID string, errMsg string) {
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
