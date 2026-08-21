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
	"github.com/vido/api/internal/ai/prompts"
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

// SubtitleStateReader loads the row state the translate-only resume check
// needs (story sub-2-2a AC #3). Narrow on purpose (Rule 11), mirroring
// SubtitleStatusWriter — *repository.MovieRepository satisfies it via FindByID;
// main.go injects it.
type SubtitleStateReader interface {
	FindByID(ctx context.Context, id string) (*models.Movie, error)
}

// EpisodeSubtitleStatusWriter is the EPISODE counterpart of
// SubtitleStatusWriter (story sub-3-2). A separate narrow interface — NOT a
// widening of the movie one (Rule 11): the episode repository's method has a
// different name, takes no score, and episodes carry no search-score columns.
// *repository.EpisodeRepository satisfies it; main.go injects it.
type EpisodeSubtitleStatusWriter interface {
	UpdateEpisodeSubtitleStatus(ctx context.Context, episodeID string, status models.SubtitleStatus, path, language string) error
}

// EpisodeSubtitleStateReader is the EPISODE counterpart of SubtitleStateReader
// (story sub-3-2) — the translate-only resume check only reads
// SubtitleStatus + SubtitlePath, which the episode row carries (Story 12-2).
// *repository.EpisodeRepository satisfies it; main.go injects it.
type EpisodeSubtitleStateReader interface {
	FindByID(ctx context.Context, id string) (*models.Episode, error)
}

// SeriesMetadataReader loads the parent SERIES row behind the FR26 media
// context an episode run feeds into its translation prompt (Story 9R-8).
// Narrow on purpose (Rule 11) — *repository.SeriesRepository satisfies it;
// main.go injects it.
//
// A separate interface rather than a widening of SubtitleStateReader /
// EpisodeSubtitleStateReader: those return *models.Movie / *models.Episode, and
// Go has no way to express "same method, different row type" in one interface.
// The movie half of the metadata read needs NO new port — SubtitleStateReader
// already hands back the complete *models.Movie.
type SeriesMetadataReader interface {
	FindByID(ctx context.Context, id string) (*models.Series, error)
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
// [@contract-v2] (stamped v1 by 9R-18, first formalization; bumped v1→v2 by
// sub-3-2 CR): every payload's `media_id` is the MEDIA row id — a movie id, or
// an episode id since the pipeline's ASR fallback leg went media-type-aware.
// Always a UUID STRING, never numeric (the 9R-18 clause, unchanged).
// Consumers: slice-1 useGenerationProgress + the ux3-subtitle-v2-batch branch —
// both filter strictly by their own tracked media id, so foreign-media events
// pass through unobserved by design.
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

	// sub-2-2a AC #3: row-state read for the translate-only resume (optional /
	// nil-safe — without it every run is a full run, the pre-2-2a behaviour).
	stateReader SubtitleStateReader

	// sub-3-2: the EPISODE writer/reader pair behind WithMediaType dispatch.
	// Optional at construction, but an EPISODE run with a nil writer fails
	// LOUDLY at the writeback (AC #7) — silently skipping would strand the row
	// and a movie-repo fallback would report a phantom missing movie.
	episodeWriter EpisodeSubtitleStatusWriter
	episodeReader EpisodeSubtitleStateReader

	// 9R-8: the parent-series row behind an episode's FR26 prompt context.
	// Optional / nil-safe — unwired means an episode translates with no media
	// context, which is exactly the pre-9R-8 behaviour.
	seriesReader SeriesMetadataReader

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

// SetSubtitleStateReader wires the row-state read behind the translate-only
// resume (story sub-2-2a AC #3). Nil-safe: when unset, every run is a full run.
func (s *TranscriptionService) SetSubtitleStateReader(r SubtitleStateReader) {
	s.stateReader = r
}

// SetEpisodeSubtitleStatusWriter wires the EPISODE writeback (story sub-3-2).
func (s *TranscriptionService) SetEpisodeSubtitleStatusWriter(w EpisodeSubtitleStatusWriter) {
	s.episodeWriter = w
}

// SetEpisodeSubtitleStateReader wires the EPISODE row-state read behind the
// translate-only resume (story sub-3-2). Nil-safe: when unset, an episode run
// is always a full run.
func (s *TranscriptionService) SetEpisodeSubtitleStateReader(r EpisodeSubtitleStateReader) {
	s.episodeReader = r
}

// SetSeriesMetadataReader wires the parent-series read behind an episode's FR26
// prompt context (Story 9R-8). Nil-safe: when unset, an episode run translates
// with no media context.
func (s *TranscriptionService) SetSeriesMetadataReader(r SeriesMetadataReader) {
	s.seriesReader = r
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

// IsAvailable returns true if both FFmpeg and speech recognition are configured.
//
// sub-5-2 AC #2: the ASR provider may be a LAZY holder whose key is resolved per
// call. Since sub-5-2 this service is constructed UNCONDITIONALLY, so "the
// provider object exists" no longer implies "ASR is configured" — without this
// probe a keyless install would report available, and the three gates that read
// it (the FR12 503, the worker-pool sweep, the legacy batch 503) would all lie.
// A plain provider (a directly-constructed WhisperClient, or a test fake) has no
// probe and stays available, exactly as before. Verbatim isomorph of
// TranslationService.IsConfigured — including that compatibility clause.
func (s *TranscriptionService) IsAvailable() bool {
	if s.audioExtractor == nil || !s.audioExtractor.IsAvailable() || s.asr == nil {
		return false
	}
	if probe, ok := s.asr.(interface {
		IsConfigured(ctx context.Context) bool
	}); ok {
		return probe.IsConfigured(context.Background())
	}
	return true
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
	cfg := newTranscriptionConfig(opts)

	// CR sub-2-2a M2: a translate-only resume needs NO ASR, so the ASR
	// availability gate must not block it — an operator who removed the ASR
	// key AFTER generating can still finish the translation.
	if !s.IsAvailable() && !(cfg.translate && s.canResumeTranslateOnly(ctx, cfg.mediaType, mediaID)) {
		return "", ErrTranscriptionDisabled
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
		_ = s.runPipeline(pipelineCtx, jobID, cfg.mediaType, mediaID, filePath, mediaDir, cfg.translate)
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
	cfg := newTranscriptionConfig(opts)

	// CR sub-2-2a M2: same gate relaxation as StartTranscription — a
	// translate-only resume needs no ASR.
	if !s.IsAvailable() && !(cfg.translate && s.canResumeTranslateOnly(ctx, cfg.mediaType, mediaID)) {
		return ErrTranscriptionDisabled
	}

	jobID, err := s.acquireJob(mediaID)
	if err != nil {
		return err
	}

	pipelineCtx, pipelineCancel := context.WithTimeout(ctx, s.timeout)
	defer pipelineCancel()
	return s.runPipeline(pipelineCtx, jobID, cfg.mediaType, mediaID, filePath, mediaDir, cfg.translate)
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
	mediaType string
}

// newTranscriptionConfig applies opts over the defaults. mediaType defaults to
// MOVIE (sub-3-2): every pre-existing caller — the manual handler and the
// 9R-16 generation batch — is movie-scoped and stays byte-unchanged with zero
// call-site edits.
func newTranscriptionConfig(opts []TranscriptionOption) *transcriptionConfig {
	cfg := &transcriptionConfig{mediaType: models.SubtitleRunMediaMovie}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// WithTranslation enables post-transcription translation to Traditional Chinese.
func WithTranslation() TranscriptionOption {
	return func(c *transcriptionConfig) {
		c.translate = true
	}
}

// WithMediaType selects which media table the run's writeback and resume-read
// dispatch to (story sub-3-2). Vocabulary is the INTERNAL movie|episode set
// (models.SubtitleRunMedia*), never the TMDB movie|tv pair. Unknown values
// fall back to the movie path (defensive — callers are in-repo).
func WithMediaType(mediaType string) TranscriptionOption {
	return func(c *transcriptionConfig) {
		if mediaType != "" {
			c.mediaType = mediaType
		}
	}
}

// TranscriptionOptionsFor resolves an option slice to the effect it would have
// on a run: (translate, mediaType). Exported for CALLER TESTS in other
// packages — TranscriptionOption is an opaque func over an unexported config,
// so a handler test cannot otherwise tell WithMediaType(episode) apart from a
// no-op. Story 9R-10a AC #5: a handler that forgets WithMediaType writes an
// episode's status into the movies table with no error at all, so "the option
// was actually sent" needs a real assertion, not a length check.
func TranscriptionOptionsFor(opts []TranscriptionOption) (translate bool, mediaType string) {
	cfg := newTranscriptionConfig(opts)
	return cfg.translate, cfg.mediaType
}

// runPipeline executes the full transcription pipeline:
// Extract audio → (optional chunk) → Whisper API → Merge SRT → Save → (optional) Translate
// It reports failures via failJob SSE AND returns the error so the synchronous
// entry (RunTranscription, 9R-16) can propagate it; the async entry discards it.
func (s *TranscriptionService) runPipeline(ctx context.Context, jobID string, mediaType string, mediaID string, filePath string, mediaDir string, translate bool) error {
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

	// sub-2-2a AC #3 (A+續跑): a previous run already produced the English SRT
	// and only translation is missing → skip extract+ASR, run translate-only.
	// The check is BOUND to subtitle_status == untranslated; bare on-disk .srt
	// presence must never trigger a resume (a user-placed subtitle is not a
	// resume point). Failure to resume degrades to a full run, never errors.
	srtContent, srtPath, resumed := s.tryTranslateOnlyResume(ctx, jobID, mediaType, mediaID, translate)

	if !resumed {
		// CR sub-2-2a M2 fallback guard: the entry gate admits ASR-less runs
		// that are resume-eligible; if the resume then degrades to a full run
		// (e.g. the SRT vanished between gate and here), the full path must
		// refuse cleanly rather than dereference a missing extractor/ASR.
		if !s.IsAvailable() {
			s.failJob(jobID, mediaID, "transcription unavailable and resume not possible")
			return ErrTranscriptionDisabled
		}

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

		srtContent, err = s.transcribeAudio(ctx, audioPath, WhisperLanguageFromTrack(selectedTrack.Language))
		if err != nil {
			s.failJob(jobID, mediaID, fmt.Sprintf("transcribe: %v", err))
			return fmt.Errorf("transcribe: %w", err)
		}

		// Phase 3: Save SRT
		baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
		srtPath = filepath.Join(mediaDir, baseName+".en.srt")

		if err := os.WriteFile(srtPath, []byte(srtContent), 0644); err != nil {
			s.failJob(jobID, mediaID, fmt.Sprintf("save SRT: %v", err))
			return fmt.Errorf("save SRT: %w", err)
		}
	}

	// Phase 3.5: Translate to Traditional Chinese (Story 9-2b) + persist the
	// generation verdict (9R-16 AC 12; en-only → untranslated per sub-2-2a).
	zhSRTPath, outcome, err := s.translateAndPersist(ctx, jobID, mediaType, mediaID, srtContent, srtPath, filePath, mediaDir, translate)
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

	// Phase 4: Complete.
	completeData := buildCompleteData(jobID, mediaID, srtPath, zhSRTPath, duration, resumed, outcome)
	s.broadcastEvent(EventTranscriptionComplete, completeData)
	return nil
}

// buildCompleteData assembles the transcription_complete SSE payload.
// Extracted pure (bugfix-j CR M2) so the partial-terminal contract is
// unit-testable. Additive keys `partial` + `english_kept_blocks` are present
// ONLY on partial terminals — absent = full success; consumers must not read
// absence as 0-vs-missing. Messages are zh-TW per the 9R-10a copy ruling
// (bugfix-j CR L1: no mixed-script strings in user-visible SSE text).
func buildCompleteData(jobID, mediaID, srtPath, zhSRTPath, duration string, resumed bool, outcome TranslationOutcome) map[string]interface{} {
	// CR sub-2-2a L1: a resumed run transcribed nothing — its completion
	// message must not claim it did.
	completeMsg := "轉錄完成"
	if resumed {
		completeMsg = "翻譯完成（自既有英文字幕續跑）"
	}
	// bugfix-j AC #3: a partial terminal must say so — the verdict already
	// recorded `untranslated`, and the completion message carries the same
	// truth plus the English-kept cue count.
	if outcome.Partial() {
		completeMsg += fmt.Sprintf("（部分翻譯失敗，%d 句保留英文）", outcome.EnglishKeptBlocks)
	}
	data := map[string]interface{}{
		"job_id":   jobID,
		"media_id": mediaID,
		"phase":    "complete",
		"srt_path": srtPath,
		"duration": duration,
		"message":  completeMsg,
	}
	if zhSRTPath != "" {
		data["zh_srt_path"] = zhSRTPath
	}
	if outcome.Partial() {
		// bugfix-j CR H2: `partial` lets the FE terminal verdict line stop
		// trusting zh_srt_path alone — the mixed file exists, but 完成 it is not.
		data["partial"] = true
		data["english_kept_blocks"] = outcome.EnglishKeptBlocks
	}
	return data
}

// resumeSource returns the recorded English SRT path when the row is
// resume-eligible: status `untranslated` + a recorded path + the file present
// on disk. Shared by the entry-gate check (canResumeTranslateOnly) and the
// pipeline resume itself. BOUND to the status (Winston guard): a user-placed
// .srt on a row in any other state must never be laundered into a resume point.
//
// sub-3-2: dispatches on mediaType — an episode run consults the EPISODE
// reader only; falling back to the movie reader would resume off the wrong
// table's row.
func (s *TranscriptionService) resumeSource(ctx context.Context, mediaType, mediaID string) (string, bool) {
	status, path, ok := s.loadSubtitleRowState(ctx, mediaType, mediaID)
	if !ok {
		return "", false
	}
	if status != models.SubtitleStatusUntranslated || path == "" {
		return "", false
	}
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
}

// loadSubtitleRowState normalizes the media-type-specific row read to the two
// fields the resume check needs. ok=false covers: reader not wired, lookup
// error, row absent — all of which degrade to a full run (Rule 13 fail-soft).
func (s *TranscriptionService) loadSubtitleRowState(ctx context.Context, mediaType, mediaID string) (models.SubtitleStatus, string, bool) {
	if mediaType == models.SubtitleRunMediaEpisode {
		if s.episodeReader == nil {
			return "", "", false
		}
		episode, err := s.episodeReader.FindByID(ctx, mediaID)
		if err != nil || episode == nil {
			return "", "", false
		}
		return episode.SubtitleStatus, episode.SubtitlePath.String, true
	}
	if s.stateReader == nil {
		return "", "", false
	}
	movie, err := s.stateReader.FindByID(ctx, mediaID)
	if err != nil || movie == nil {
		return "", "", false
	}
	return movie.SubtitleStatus, movie.SubtitlePath.String, true
}

// CanResumeTranslateOnly reports whether a MOVIE run for mediaID would resume
// translate-only (CR sub-2-2a M2). Exposed to the handler, whose surface is
// movie-only — internal callers use the media-type-aware variant.
func (s *TranscriptionService) CanResumeTranslateOnly(ctx context.Context, mediaID string) bool {
	return s.canResumeTranslateOnly(ctx, models.SubtitleRunMediaMovie, mediaID)
}

// CanResumeEpisodeTranslateOnly reports whether an EPISODE run for episodeID
// would resume translate-only (story 9R-10a AC #3). A SIBLING of
// CanResumeTranslateOnly, deliberately not a widening of it (Rule 11) — the
// same shape the EpisodeSubtitleStatusWriter/Reader pair took in sub-3-2.
//
// The movie variant hard-codes SubtitleRunMediaMovie, so calling it with an
// episode id queries the movies table, returns false forever, and 503s an
// episode that needs no ASR at all (the episode twin of CR sub-2-2a M2).
func (s *TranscriptionService) CanResumeEpisodeTranslateOnly(ctx context.Context, episodeID string) bool {
	return s.canResumeTranslateOnly(ctx, models.SubtitleRunMediaEpisode, episodeID)
}

// canResumeTranslateOnly is the media-type-aware resume check behind the
// entry gates (sub-3-2).
func (s *TranscriptionService) canResumeTranslateOnly(ctx context.Context, mediaType, mediaID string) bool {
	_, ok := s.resumeSource(ctx, mediaType, mediaID)
	return ok
}

// tryTranslateOnlyResume implements sub-2-2a AC #3: when the row says
// `untranslated` and its recorded English SRT is still on disk, return its
// content so the pipeline can skip extract+ASR. All failure modes degrade to
// (_, _, false) — a full run — never an error (Rule 13 logged degrade).
func (s *TranscriptionService) tryTranslateOnlyResume(ctx context.Context, jobID, mediaType, mediaID string, translate bool) (srtContent, srtPath string, ok bool) {
	if !translate {
		return "", "", false
	}
	path, eligible := s.resumeSource(ctx, mediaType, mediaID)
	if !eligible {
		return "", "", false
	}

	content, err := os.ReadFile(path)
	if err != nil {
		s.logger.Info("resume SRT unavailable — falling back to a full run",
			"job_id", jobID, "media_id", mediaID, "srt_path", path, "error", err)
		return "", "", false
	}

	s.logger.Info("resuming translate-only from existing English SRT",
		"job_id", jobID, "media_id", mediaID, "srt_path", path)
	return string(content), path, true
}

// translateAndPersist runs the optional translate phase and the generation
// writeback. Returns the zh-Hant path ("" when no translation happened).
// Error semantics (ruled in 9R-16 AC 6c/12, extended by sub-2-2a AC #2):
//   - ordinary translate failures are NON-FATAL (English SRT preserved,
//     deliberate swallow — logged, zh path stays empty);
//   - ai.ErrBudgetExceeded MUST propagate so a batch can pause mid-item
//     (BEFORE any writeback — a paused item is not terminal);
//   - zh-Hant FULL success writes found/zh-path/"zh-Hant";
//   - zh-Hant PARTIAL success (whole batches or single cues kept English —
//     bugfix-j) still places the mixed file (tonight-value) but writes
//     untranslated/en-path/"en": the badge must not claim 繁中已就緒 over a
//     file carrying English runs, and the EN path keeps resume translate-only;
//   - translate EXPECTED but zh empty (key unconfigured, or a non-fatal
//     translate failure) writes untranslated/en-path/"en" — 9R-16 AC 12
//     [@contract-v3] (v2→v3 by bugfix-j: `found` now requires FULL
//     translation; partial results demote to untranslated): without this the
//     badge lies 缺字幕 over a paid-for SRT and a re-click re-runs the whole
//     ASR (sub-2-2a's headline bug);
//   - translate NOT requested writes nothing (no translation was expected,
//     so `untranslated` would be a lie);
//   - a writeback failure propagates (Rule 13) — reporting success while the
//     library row still lies would break the batch-enumeration guarantee.
func (s *TranscriptionService) translateAndPersist(ctx context.Context, jobID string, mediaType string, mediaID string, srtContent, srtPath, filePath, mediaDir string, translate bool) (string, TranslationOutcome, error) {
	var zhSRTPath string
	var outcome TranslationOutcome
	if translate && s.translationService != nil && s.translationService.IsConfigured() {
		s.broadcastEvent(EventTranscriptionTranslating, map[string]interface{}{
			"job_id":     jobID,
			"media_id":   mediaID,
			"phase":      "translating",
			"percentage": 0,
			"message":    "Translating subtitles to Traditional Chinese",
		})

		zhPath, tOutcome, err := s.translateSRT(ctx, jobID, mediaType, mediaID, srtContent, filePath, mediaDir)
		if err != nil {
			// 9R-16 AC 6c: the budget sentinel MUST propagate — a mid-translate
			// ceiling hit must pause the batch, not count the item as success
			// (the English SRT stays on disk either way).
			if errors.Is(err, ai.ErrBudgetExceeded) {
				// CR sub-2-2a M1: record the on-disk truth BEFORE pausing — the
				// English SRT exists and translation is missing, which is
				// exactly what `untranslated` states. Without this, the paused
				// item resumes as a FULL run and re-pays the whole ASR — the
				// story's headline bug with a different trigger. Best-effort
				// (log-only on failure): the sentinel must survive so the
				// batch pauses; the writeback here is the optimization, not
				// the contract.
				if werr := s.writeSubtitleStatus(ctx, mediaType, mediaID,
					models.SubtitleStatusUntranslated, srtPath, "en"); werr != nil {
					s.logger.Warn("untranslated writeback before budget pause failed",
						"job_id", jobID, "media_id", mediaID, "media_type", mediaType, "error", werr)
				}
				return "", TranslationOutcome{}, fmt.Errorf("translate: %w", err)
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
			outcome = tOutcome
		}
	}

	// 9R-16 AC 12 [@contract-v3] (sub-2-2a; v2→v3 by bugfix-j): persist the
	// generation VERDICT — the resume enabler + badge truth. zh-Hant success
	// with MATERIAL English residue (H3 ruling: ≥1 batch's worth of cues or
	// ≥5% of this item's own cue count — see TranslationOutcome.DemotesVerdict)
	// → untranslated + the EN path (mixed file stays placed; resume
	// re-translates from the EN source and the placer overwrites it);
	// zh-Hant success below that bar → found (residue still disclosed via
	// the SSE partial flag); translation expected but absent → untranslated +
	// the EN path. A translate=false run writes nothing: no translation was
	// expected.
	switch {
	case zhSRTPath != "" && outcome.DemotesVerdict():
		// bugfix-j AC #2: same writeback shape as the budget-pause branch
		// (CR sub-2-2a M1) — EN path is the resume enabler. Rule 13: this
		// branch is a VERDICT write, not a best-effort optimization, so a
		// failure propagates exactly like the found/untranslated branches.
		if werr := s.writeSubtitleStatus(ctx, mediaType, mediaID,
			models.SubtitleStatusUntranslated, srtPath, "en"); werr != nil {
			return "", outcome, fmt.Errorf("update subtitle status: %w", werr)
		}
	case zhSRTPath != "":
		if werr := s.writeSubtitleStatus(ctx, mediaType, mediaID,
			models.SubtitleStatusFound, zhSRTPath, "zh-Hant"); werr != nil {
			return "", outcome, fmt.Errorf("update subtitle status: %w", werr)
		}
	case translate:
		if werr := s.writeSubtitleStatus(ctx, mediaType, mediaID,
			models.SubtitleStatusUntranslated, srtPath, "en"); werr != nil {
			return "", outcome, fmt.Errorf("update subtitle status: %w", werr)
		}
	}

	return zhSRTPath, outcome, nil
}

// writeSubtitleStatus is the sub-3-2 writeback dispatch. The movie branch
// keeps 9R-16's nil-writer semantics EXACTLY (nil = place files, persist
// nothing — a legal boot state) and always passes score 0 (a generated
// subtitle has no search score). The episode branch fails LOUDLY on a nil
// writer (AC #7): silently skipping would strand the row mid-pipeline, and
// there is no pre-sub-3-2 behavior to preserve — episode runs did not exist.
func (s *TranscriptionService) writeSubtitleStatus(ctx context.Context, mediaType, mediaID string, status models.SubtitleStatus, path, language string) error {
	if mediaType == models.SubtitleRunMediaEpisode {
		if s.episodeWriter == nil {
			return fmt.Errorf("episode subtitle writer not wired — cannot persist %s for episode %s", status, mediaID)
		}
		return s.episodeWriter.UpdateEpisodeSubtitleStatus(ctx, mediaID, status, path, language)
	}
	if s.subtitleWriter == nil {
		return nil
	}
	return s.subtitleWriter.UpdateSubtitleStatus(ctx, mediaID, status, path, language, 0)
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
		filtered, unfiltered, err := s.transcribeOne(ctx, audioPath, lang)
		if err != nil {
			return "", err
		}
		return s.guardAgainstEmptyTranscript(filtered, unfiltered), nil
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
	// 9R-5: kept alongside so the whole-FILE guard below has something to fall
	// back to. Each chunk is its own API call, which is why hallucination
	// filtering is automatically per-chunk and automatically happens before the
	// merge — no change to SplitAudioChunks/MergeSRTChunks was needed.
	var unfilteredChunks []string
	for i, chunkPath := range chunks {
		s.logger.Info("transcribing chunk",
			"chunk", i+1,
			"total", len(chunks),
		)
		filtered, unfiltered, err := s.transcribeOne(ctx, chunkPath, lang)
		if err != nil {
			return "", fmt.Errorf("transcribe chunk %d/%d: %w", i+1, len(chunks), err)
		}
		srtChunks = append(srtChunks, filtered)
		unfilteredChunks = append(unfilteredChunks, unfiltered)
	}

	return s.guardAgainstEmptyTranscript(
		ai.MergeSRTChunks(srtChunks, chunkSeconds),
		ai.MergeSRTChunks(unfilteredChunks, chunkSeconds),
	), nil
}

// transcribeOne returns both the hallucination-filtered SRT and the unfiltered
// one for a single audio file.
//
// The detailed seam is OPTIONAL (ai.DetailedTranscriber, the ai.CachingCompleter
// precedent): an ASR provider that does not implement it — or an engine that
// could not serve verbose_json — simply reports the same string twice, which
// makes the guard below a no-op rather than a failure.
func (s *TranscriptionService) transcribeOne(ctx context.Context, audioPath, lang string) (filtered, unfiltered string, err error) {
	if detailed, ok := s.asr.(ai.DetailedTranscriber); ok {
		d, err := detailed.TranscribeDetailed(ctx, audioPath, lang)
		if err != nil {
			return "", "", err
		}
		return d.SRT, d.Unfiltered, nil
	}
	srt, err := s.asr.TranscribeWithLanguage(ctx, audioPath, lang)
	if err != nil {
		return "", "", err
	}
	return srt, srt, nil
}

// guardAgainstEmptyTranscript is the 9R-5 whole-FILE safety net.
//
// A single chunk collapsing to nothing is LEGITIMATE and is the headline case:
// ten minutes of silent credits that whisper filled with an invented outro
// should leave no cues at all. An entire FILE collapsing is different — no film
// is wall-to-wall hallucination, so that outcome means the thresholds misfired.
//
// Letting it through would be actively harmful, not merely lossy: runPipeline
// writes srtContent to disk unconditionally, so the user would get a 0-byte
// .en.srt, translateSRT would then fail with "no subtitle blocks to translate",
// translateAndPersist would swallow that as non-fatal, and the row would end up
// claiming `untranslated` over an empty file — the same class of lying status
// bugfix-j was written to remove.
func (s *TranscriptionService) guardAgainstEmptyTranscript(filtered, unfiltered string) string {
	if countSRTCues(filtered) > 0 || countSRTCues(unfiltered) == 0 {
		return filtered
	}
	s.logger.Warn("hallucination filter would have emptied the entire transcript — delivering the unfiltered text instead",
		"unfiltered_cues", countSRTCues(unfiltered),
	)
	return unfiltered
}

// countSRTCues counts cues by their timestamp arrow — the one line every cue
// has and no dialogue line can contain.
func countSRTCues(srt string) int {
	return strings.Count(srt, " --> ")
}

// glossaryMediaKey resolves which glossary a run feeds from and harvests into
// (CR sub-5-5 H1). The glossary is per-SHOW: an episode run must use its
// PARENT SERIES id — the key the F6 review panel and the pipeline path already
// use — or its harvest lands in invisible episode-id rows no other episode
// ever benefits from. Movies (and series rows) key on themselves. Fail-soft:
// an unwired reader or a failed lookup falls back to mediaID (the pre-fix
// behavior — degraded consistency, never a failed run).
func (s *TranscriptionService) glossaryMediaKey(ctx context.Context, mediaType, mediaID string) string {
	if mediaType != models.SubtitleRunMediaEpisode || s.episodeReader == nil {
		return mediaID
	}
	episode, err := s.episodeReader.FindByID(ctx, mediaID)
	if err != nil || episode == nil || episode.SeriesID == "" {
		s.logger.Warn("glossary key resolve failed — falling back to the episode's own id",
			"media_id", mediaID, "error", err)
		return mediaID
	}
	return episode.SeriesID
}

// mediaMetadataFor resolves the FR26 media context this run translates with
// (Story 9R-8) — the same show-level facts the EXTRACT leg has fed into its
// system blocks since sub-1-5a (subtitle/media_store.go loadMovie /
// seriesContext → pipeline.go metadataOf).
//
// glossaryKey is the value glossaryMediaKey ALREADY resolved for this run: for
// an episode it IS the parent series id, so the episode row is never read a
// second time here.
//
// Episodes deliberately carry their PARENT SERIES' context, not their own
// title/overview — the extract leg made that call because per-episode fields
// split the segment cache's MetadataHash across every episode of one show
// (media_store.go loadEpisode). This leg has no cache to split, but diverging
// would mean the same show translates against two different contexts depending
// on which leg ran, so it mirrors the extract leg.
//
// Fail-soft throughout (Rule 13 case 3): a missing reader, a missing row or a
// database error yields the ZERO MediaMetadata, which BuildMetadataSection
// renders as "" — the prompt is then byte-identical to the pre-9R-8 one and the
// translation proceeds. Refusing to translate because a metadata row is absent
// would be strictly worse than translating with less context.
func (s *TranscriptionService) mediaMetadataFor(ctx context.Context, mediaType, mediaID, glossaryKey string) prompts.MediaMetadata {
	switch mediaType {
	case models.SubtitleRunMediaMovie:
		if s.stateReader == nil {
			// CR M1: NEVER a silent degrade. An unwired reader means every
			// movie on this leg translates context-blind — the exact class of
			// quiet downgrade main.go's sub-2-1a comment was written about.
			s.logger.Warn("no movie reader wired — translating without show context",
				"media_id", mediaID, "media_type", mediaType)
			return prompts.MediaMetadata{}
		}
		movie, err := s.stateReader.FindByID(ctx, mediaID)
		// NOTE the repository contract: a missing row comes back as an ERROR
		// wrapping sql.ErrNoRows, not (nil, nil) (movie_repository.go). Both
		// arms are guarded so neither shape can panic here.
		if err != nil || movie == nil {
			s.logger.Warn("media metadata lookup failed — translating without show context",
				"media_id", mediaID, "media_type", mediaType, "error", err)
			return prompts.MediaMetadata{}
		}
		return prompts.MediaMetadata{
			Title:         movie.Title,
			OriginalTitle: movie.OriginalTitle.String,
			Year:          releaseYear(movie.ReleaseDate),
			Genres:        movie.Genres,
			Overview:      movie.Overview.String,
			Countries:     productionCountryCodes(movie.ProductionCountries),
		}

	case models.SubtitleRunMediaEpisode, models.SubtitleRunMediaSeries:
		if s.seriesReader == nil {
			s.logger.Warn("no series reader wired — translating without show context",
				"media_id", mediaID, "media_type", mediaType)
			return prompts.MediaMetadata{}
		}
		// CR M4: glossaryMediaKey FAILS SOFT to the episode's own id when the
		// parent lookup misses. Querying the series table with an episode id
		// would then log `series_id=<an episode id>` and send an operator
		// hunting for a series row that never existed — the 9R-10a CR M1 class
		// (a real fault dressed up as a normal one). Report what actually
		// happened instead.
		if mediaType == models.SubtitleRunMediaEpisode && glossaryKey == mediaID {
			s.logger.Warn("parent series unresolved for this episode — translating without show context",
				"media_id", mediaID, "media_type", mediaType)
			return prompts.MediaMetadata{}
		}
		// For an episode glossaryKey is the parent series id; for a series run
		// it is the row's own id. Either way it is the SHOW.
		series, err := s.seriesReader.FindByID(ctx, glossaryKey)
		if err != nil || series == nil {
			s.logger.Warn("series metadata lookup failed — translating without show context",
				"media_id", mediaID, "media_type", mediaType, "series_id", glossaryKey, "error", err)
			return prompts.MediaMetadata{}
		}
		// Countries stay empty: the series table carries no production_countries
		// column, exactly as seriesContext leaves it on the extract leg.
		return prompts.MediaMetadata{
			Title:         series.Title,
			OriginalTitle: series.OriginalTitle.String,
			Year:          releaseYear(series.FirstAirDate),
			Genres:        series.Genres,
			Overview:      series.Overview.String,
		}

	default:
		return prompts.MediaMetadata{}
	}
}

// releaseYear extracts the year from an ISO date. An empty or unparseable date
// yields 0, which BuildMetadataSection omits rather than rendering a bogus
// year. Mirrors subtitle/media_store.go yearOf — duplicated rather than shared
// because services must not import subtitle (Rule 19), the same reason
// OpenCCConverter is redeclared in this file.
func releaseYear(date string) int {
	if len(date) < 4 {
		return 0
	}
	year, err := strconv.Atoi(date[:4])
	if err != nil {
		return 0
	}
	return year
}

// productionCountryCodes projects TMDb's country objects onto the ISO codes the
// prompt renders. Mirrors subtitle/media_store.go countryCodes (Rule 19 — see
// releaseYear).
func productionCountryCodes(countries []models.ProductionCountry) []string {
	if len(countries) == 0 {
		return nil
	}
	out := make([]string, 0, len(countries))
	for _, c := range countries {
		if code := strings.TrimSpace(c.ISO3166_1); code != "" {
			out = append(out, code)
		}
	}
	return out
}

// translateSRT translates English SRT content to Traditional Chinese and saves as .zh-Hant.srt.
// Returns the path to the translated SRT file plus the fidelity outcome
// (bugfix-j): a partial outcome means the placed file still carries English
// cues and the verdict must say `untranslated`, never `found`.
func (s *TranscriptionService) translateSRT(ctx context.Context, jobID string, mediaType string, mediaID string, srtContent string, filePath string, mediaDir string) (string, TranslationOutcome, error) {
	// Parse SRT into translation blocks (inline to avoid circular import with subtitle pkg)
	blocks, err := ParseSRTToTranslationBlocks(srtContent)
	if err != nil {
		return "", TranslationOutcome{}, fmt.Errorf("parse SRT for translation: %w", err)
	}

	if len(blocks) == 0 {
		return "", TranslationOutcome{}, fmt.Errorf("no subtitle blocks to translate")
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
	// CR sub-5-5 H1: feed AND harvest key on the SHOW-level glossary — for an
	// episode that is the parent series id, not the episode's own id.
	glossaryKey := s.glossaryMediaKey(ctx, mediaType, mediaID)
	glossary := s.loadGlossary(ctx, glossaryKey)
	if len(glossary) > 0 {
		s.logger.Info("translating with per-show glossary",
			"media_id", mediaID, "glossary_key", glossaryKey, "term_count", len(glossary))
	}
	// 9R-8: the FR26 show context the extract leg has always had. Reuses the
	// glossaryKey resolved just above, so an episode costs no extra row read.
	metadata := s.mediaMetadataFor(ctx, mediaType, mediaID, glossaryKey)
	// CR M2: this leg has NO prompt caching — TranslateChunk (extract leg) can
	// type-assert ai.CachingCompleter, TranslateWithGlossaryHarvest calls plain
	// CompleteText — so the media-context section is re-sent, and re-billed, on
	// EVERY batch. At batch size 10 a feature-length subtitle is ~120-180
	// batches, so a ~200-token section costs roughly 25-35k extra input tokens
	// per item and brings the 9R-11 run budget ceiling forward accordingly.
	// Rendered once here purely so an operator can tie a budget trip (or a
	// proper-noun drift complaint) to whether context was actually applied.
	metadataChars := len(prompts.BuildMetadataSection(metadata))
	s.logger.Info("translating with media context",
		"media_id", mediaID,
		"media_type", mediaType,
		"metadata_applied", metadataChars > 0,
		"metadata_chars", metadataChars,
		"resent_per_batch", true,
	)
	// sub-5-5: the harvest variant also returns the trailer's term yield —
	// written back below AFTER the translation has succeeded and placed.
	translated, harvestedTerms, outcome, err := s.translationService.TranslateWithGlossaryHarvest(
		ctx, blocks, glossary, progressFn, WithMediaMetadata(metadata))
	if err != nil {
		return "", TranslationOutcome{}, fmt.Errorf("translate: %w", err)
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
			return "", TranslationOutcome{}, fmt.Errorf("place zh-Hant SRT: %w", err)
		}
	} else {
		baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
		zhSRTPath = filepath.Join(mediaDir, baseName+".zh-Hant.srt")
		if err := os.WriteFile(zhSRTPath, []byte(zhSRT), 0644); err != nil {
			return "", TranslationOutcome{}, fmt.Errorf("save zh-Hant SRT: %w", err)
		}
	}

	// sub-5-5 AC #4 回程: the translate stage succeeded and the subtitle is on
	// disk — feed the harvested renderings back, insert-if-absent only (an
	// existing term, whatever its source or confirmed state, is never touched).
	// Fail-soft (Rule 13 case 3): a lost harvest is re-collected next run.
	harvestedNew := s.harvestGlossaryTerms(ctx, glossaryKey, harvestedTerms)

	s.logger.Info("subtitle translation complete",
		"job_id", jobID,
		"media_id", mediaID,
		"zh_srt_path", zhSRTPath,
		"block_count", len(translated),
		"glossary_terms", len(glossary),
		// sub-5-5 AC #6: NEW terms written back this run (dedupes excluded).
		"harvested_terms", harvestedNew,
		// bugfix-j: cues still carrying English after the translate stage.
		"english_kept_blocks", outcome.EnglishKeptBlocks,
	)

	return zhSRTPath, outcome, nil
}

// harvestGlossaryTerms writes the trailer yield insert-if-absent (sub-5-5,
// legacy path). Returns how many terms were actually inserted. Nil-safe: no
// glossary repo wired = nothing written, exactly the 9R-10 feed posture.
func (s *TranscriptionService) harvestGlossaryTerms(ctx context.Context, mediaID string, terms map[string]string) int {
	if s.glossaryRepo == nil || len(terms) == 0 {
		return 0
	}
	inserted := 0
	for src, zh := range terms {
		// CR sub-5-5 H2: s2twp the rendering before it becomes a MANDATORY
		// glossary feed — a raw Simplified rendering would poison every future
		// prompt for the show. Fail-soft: keep the raw value on converter error.
		if s.opencc != nil && s.opencc.IsAvailable() {
			if converted, cerr := s.opencc.ConvertS2TWP([]byte(zh)); cerr == nil {
				zh = string(converted)
			}
		}
		ok, err := s.glossaryRepo.InsertIfAbsent(ctx, &models.GlossaryTerm{
			MediaID: mediaID,
			TermSrc: src,
			TermZh:  zh,
			Source:  models.GlossarySourceSubtitle,
			// Confirmed stays false — harvested terms enter the F6 review flow.
		})
		if err != nil {
			s.logger.Warn("glossary harvest write failed — the term will be re-collected by a future run",
				"media_id", mediaID, "term_src", src, "error", err)
			continue
		}
		if ok {
			inserted++
		}
	}
	return inserted
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
