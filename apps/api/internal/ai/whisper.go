package ai

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// WhisperAPIURL is the OpenAI Whisper transcription endpoint.
	WhisperAPIURL = "https://api.openai.com/v1/audio/transcriptions"
	// WhisperModel is the model identifier for Whisper.
	WhisperModel = "whisper-1"
	// WhisperMaxFileSize is the maximum file size the Whisper API accepts (25MB).
	WhisperMaxFileSize = 25 * 1024 * 1024
	// WhisperChunkTargetBytes is the per-chunk size budget (9R-3): 1MiB of
	// headroom under the API limit so file bytes + multipart overhead never
	// push the POST body past 25MiB (the POC's 413).
	WhisperChunkTargetBytes = 24 * 1024 * 1024
	// WhisperChunkDuration is the duration of each audio chunk in seconds (10 minutes).
	WhisperChunkDuration = 600
	// WhisperMaxResponseSize is the maximum Whisper API response body we'll read (10MB).
	WhisperMaxResponseSize = 10 * 1024 * 1024
)

// Whisper API errors
var (
	ErrWhisperNotConfigured = errors.New("whisper: OpenAI API key not configured")
	ErrWhisperAPIError      = errors.New("whisper: API error")
	ErrWhisperTimeout       = errors.New("whisper: request timed out")
)

// WhisperClient transcribes audio files using the OpenAI Whisper API.
type WhisperClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	logger     *slog.Logger
	// language is an optional ISO-639-1 hint (e.g. "en"). Empty = Whisper
	// auto-detects, which is UNRELIABLE for media with background speech in
	// other languages — pin it to the audio track's language when known.
	language string
	// governor is the shared AI throttle (Story 9R-11; nil = unthrottled).
	governor *Governor
	// model is the transcription model id (Story 9R-9). Defaults to WhisperModel;
	// self-hosted OpenAI-compatible engines use their own id (e.g. Speaches
	// "Systran/faster-whisper-small").
	model string
	// verboseUnsupported latches once this endpoint proves it cannot serve
	// `response_format=verbose_json` (Story 9R-5), so a ten-chunk file probes
	// once instead of ten times. Set ONLY on a permanent 4xx or an unusable
	// 200 body — never on a transient fault, which would silently disable
	// hallucination filtering for the life of the process.
	verboseUnsupported atomic.Bool
}

// WhisperOption is a functional option for configuring WhisperClient.
type WhisperOption func(*WhisperClient)

// WithWhisperBaseURL sets a custom base URL (useful for testing).
func WithWhisperBaseURL(url string) WhisperOption {
	return func(c *WhisperClient) {
		c.baseURL = url
	}
}

// WithWhisperGovernor injects the shared throttle (Story 9R-11).
func WithWhisperGovernor(g *Governor) WhisperOption {
	return func(c *WhisperClient) {
		c.governor = g
	}
}

// WithWhisperHTTPClient sets a custom HTTP client.
func WithWhisperHTTPClient(client *http.Client) WhisperOption {
	return func(c *WhisperClient) {
		c.httpClient = client
	}
}

// WithWhisperTimeout sets a custom timeout per request.
func WithWhisperTimeout(timeout time.Duration) WhisperOption {
	return func(c *WhisperClient) {
		c.timeout = timeout
	}
}

// WithWhisperLanguage pins the source language (ISO-639-1, e.g. "en") so Whisper
// does not mis-detect the language on media with mixed/background audio.
func WithWhisperLanguage(lang string) WhisperOption {
	return func(c *WhisperClient) {
		c.language = lang
	}
}

// WithWhisperModel overrides the transcription model id (Story 9R-9) — set it to
// a self-hosted engine's model id when swapping the base URL.
func WithWhisperModel(model string) WhisperOption {
	return func(c *WhisperClient) {
		c.model = model
	}
}

// IsSelfHostedASRBaseURL is the ONE self-hosted judgment (sub-5-1 CR M1):
// given the configured ASR_BASE_URL (empty = the hosted default), does the
// deployment pay the hosted per-minute rate? Both the metering side (the
// client's isSelfHosted below) and the estimate side (main.go's candidate
// service wiring) MUST derive their answer from this predicate — two
// independent detectors ("config non-empty" vs "endpoint differs") disagreed
// exactly when ASR_BASE_URL was explicitly set to the official endpoint,
// producing a $0 quote billed at the hosted rate.
func IsSelfHostedASRBaseURL(baseURL string) bool {
	return baseURL != "" && baseURL != WhisperAPIURL
}

// isSelfHosted reports whether this client is pointed at something other than
// the paid hosted API (sub-5-1 AC #1). Reads the ENDPOINT the client actually
// calls; equivalent to IsSelfHostedASRBaseURL over the configured override
// because NewWhisperClient defaults baseURL to WhisperAPIURL — asserted by
// TestSelfHostedJudgment_SingleSource.
func (c *WhisperClient) isSelfHosted() bool {
	return c.baseURL != WhisperAPIURL
}

// Governor returns the shared AI throttle this client was built with (nil when
// unthrottled). Exposed so a caller that REBUILDS the client — the sub-5-2 ASR
// key-hot-reload holder — can assert the same Governor instance carried over: a
// new client must never mean a new budget pool, or editing a key would silently
// reset the 9R-11 run-budget ceiling. Mirrors ClaudeProvider.Governor().
func (c *WhisperClient) Governor() *Governor { return c.governor }

// BaseURL returns the endpoint this client actually calls. Exposed for the same
// rebuild-assertion reason as Governor: the holder's fingerprint covers the
// endpoint, and a test needs to prove a base-URL change produced a different
// client rather than a silently reused one.
func (c *WhisperClient) BaseURL() string { return c.baseURL }

// NewWhisperClient creates a new Whisper API client.
func NewWhisperClient(apiKey string, opts ...WhisperOption) *WhisperClient {
	c := &WhisperClient{
		apiKey:  apiKey,
		baseURL: WhisperAPIURL,
		timeout: 5 * time.Minute,
		logger:  slog.Default().With("service", "whisper"),
		model:   WhisperModel,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{
			Timeout: c.timeout,
		}
	}

	return c
}

// Transcribe sends an audio file to the Whisper API and returns the SRT transcription.
// It uses the client-level language hint (WithWhisperLanguage) when set.
func (c *WhisperClient) Transcribe(ctx context.Context, audioPath string) (string, error) {
	return c.TranscribeWithLanguage(ctx, audioPath, c.language)
}

// TranscribeWithLanguage sends an audio file to the Whisper API with an explicit
// per-call ISO-639-1 language hint (9R-2: pinned from the selected audio track).
// An empty lang means Whisper auto-detects (only correct when the track language
// is unknown/und — auto-detection mis-fires on mixed/background audio).
func (c *WhisperClient) TranscribeWithLanguage(ctx context.Context, audioPath, lang string) (string, error) {
	detail, err := c.TranscribeDetailed(ctx, audioPath, lang)
	if err != nil {
		return "", err
	}
	return detail.SRT, nil
}

// TranscribeDetailed is TranscribeWithLanguage plus what the 9R-5 hallucination
// filter dropped, so a caller that owns the WHOLE file (the chunking loop in
// services.TranscriptionService) can tell "this ten-minute chunk really was
// silent credits" apart from "the filter just deleted the entire movie".
//
// It is a SEPARATE optional interface rather than a widening of ai.ASRProvider:
// 9R-9's swap-in promise is that any OpenAI-compatible engine satisfies the
// provider interface, and growing that interface would break every alternative
// implementation. Callers type-assert DetailedTranscriber and degrade to the
// plain SRT when it is absent — the ai.CachingCompleter precedent
// (services/translation_service.go).
func (c *WhisperClient) TranscribeDetailed(ctx context.Context, audioPath, lang string) (TranscriptionDetail, error) {
	// sub-5-2 AC #3: only the PAID hosted API needs a key. A self-hosted
	// OpenAI-compatible engine (Speaches / WhisperLive / Subgen) authenticates
	// nothing, and requiring a key here forced those deployments to invent a
	// dummy OPENAI_API_KEY — while sub-5-1 was already metering them at $0 and
	// the docs advertised them as supported. Same single predicate as metering.
	if c.apiKey == "" && !c.isSelfHosted() {
		return TranscriptionDetail{}, ErrWhisperNotConfigured
	}

	audio, err := c.readAudioForUpload(audioPath)
	if err != nil {
		return TranscriptionDetail{}, err
	}
	filename := filepath.Base(audioPath)

	c.logger.Debug("Whisper API request", "file", filename)

	detail, ok, err := c.transcribeVerbose(ctx, audio, filename, lang)
	if err != nil {
		return TranscriptionDetail{}, err
	}
	if !ok {
		// 9R-5: the engine cannot serve verbose_json, so there are no segments
		// to judge. Re-ask for plain SRT — verbose_json's `text` field has no
		// timestamps, so "give up on verbose" must NOT mean "give up on
		// subtitles". Unfiltered mirrors SRT: nothing was inspected.
		srt, _, err := c.postTranscription(ctx, audio, filename, lang, transcribeFormatSRT)
		if err != nil {
			return TranscriptionDetail{}, err
		}
		detail = TranscriptionDetail{SRT: srt, Unfiltered: srt}
	}

	// 9R-11: meter ASR cost by audio minutes against the per-run budget.
	// sub-5-1 AC #1: at the SAME rate the estimator quotes for this endpoint —
	// a self-hosted server records $0 instead of a fabricated hosted-rate bill.
	//
	// 9R-5: metered EXACTLY ONCE per audio file, after the final answer is in
	// hand. The verbose→srt fallback issues a second HTTP request; billing the
	// same minutes twice would burn the 9R-11 run budget at double rate and
	// trip the ceiling halfway through a film.
	if b := BudgetFromContext(ctx); b != nil {
		if dur, _, derr := parseWAVInfo(audioPath); derr == nil {
			b.RecordASRWithRate(dur, EstimatedASRPerMinuteUSD(c.isSelfHosted()))
		}
	}

	c.logger.Info("Whisper transcription complete",
		"file", filename,
		"srt_bytes", len(detail.SRT),
		"filtered", detail.Filtered,
	)

	return detail, nil
}

// transcribeVerbose runs the verbose_json path. ok=false means "this engine
// cannot do verbose_json, fall back to srt" — never an outright failure.
func (c *WhisperClient) transcribeVerbose(ctx context.Context, audio []byte, filename, lang string) (TranscriptionDetail, bool, error) {
	if c.verboseUnsupported.Load() {
		return TranscriptionDetail{}, false, nil
	}

	body, status, err := c.postTranscription(ctx, audio, filename, lang, transcribeFormatVerboseJSON)
	if err != nil {
		// ONLY a permanent 4xx means "this engine does not implement the
		// format". A 5xx or a timeout is a transient fault that retryTransient
		// (9R-4) has already re-tried — latching on it would let one bad
		// minute disable hallucination filtering for the life of the process,
		// silently and forever.
		if status >= 400 && status < 500 {
			c.markVerboseUnsupported("engine rejected verbose_json", status, err)
			return TranscriptionDetail{}, false, nil
		}
		return TranscriptionDetail{}, false, err
	}

	vt, perr := parseVerboseTranscription(body)
	if perr != nil {
		// A 200 that is not parseable verbose_json (plain SRT echoed back) or
		// that carries a transcript with no segment array is the same class of
		// "accepts the field, does not implement it".
		c.markVerboseUnsupported("verbose_json response unusable", status, perr)
		return TranscriptionDetail{}, false, nil
	}

	// CR H1: an empty segment array with an empty transcript is SILENCE, not a
	// broken engine. Latching the fallback here disabled hallucination
	// filtering for the rest of the process the first time a chunk of credits
	// came back quiet — the exact input this story exists to handle.
	if len(vt.Segments) == 0 {
		c.logger.Info("transcription returned no speech",
			"file", filename, "duration_seconds", vt.Duration)
		return TranscriptionDetail{Filtered: true}, true, nil
	}

	kept, dropped := filterHallucinations(vt.Segments)
	detail := TranscriptionDetail{
		SRT:          segmentsToSRT(kept),
		Unfiltered:   segmentsToSRT(vt.Segments),
		SegmentsIn:   len(vt.Segments),
		SegmentsKept: len(kept),
		Filtered:     true,
	}

	for _, d := range dropped {
		c.logger.Debug("hallucination filter dropped a segment",
			"file", filename,
			"reason", d.Reason,
			"start", d.Segment.Start,
			"end", d.Segment.End,
			"text", strings.TrimSpace(d.Segment.Text),
			"no_speech_prob", d.Segment.NoSpeechProb,
			"avg_logprob", d.Segment.AvgLogprob,
			"compression_ratio", d.Segment.CompressionRatio,
		)
	}

	ratio := detail.DropRatio()
	c.logger.Info("hallucination filter applied",
		"file", filename,
		"segments_in", detail.SegmentsIn,
		"segments_kept", detail.SegmentsKept,
		"dropped_by_reason", dropReasonCounts(dropped),
		"drop_ratio", ratio,
	)
	if ratio > hallucinationDropRatioWarn {
		// The POC gap between an official subtitle and a generated one was ~5%
		// (1029 vs 1082 cues). Losing a fifth of the file says the thresholds
		// are wrong, not that the audio was quiet — and an operator has no
		// other way to see it.
		c.logger.Warn("hallucination filter dropped an unusually large share of the transcript",
			"file", filename,
			"drop_ratio", ratio,
			"segments_in", detail.SegmentsIn,
			"segments_kept", detail.SegmentsKept,
		)
	}

	return detail, true, nil
}

// markVerboseUnsupported latches the fallback so a ten-chunk file probes once
// instead of ten times.
func (c *WhisperClient) markVerboseUnsupported(reason string, status int, cause error) {
	if c.verboseUnsupported.Swap(true) {
		return
	}
	c.logger.Warn("falling back to plain SRT — hallucination filtering disabled for this endpoint",
		"reason", reason,
		"status_code", status,
		"self_hosted", c.isSelfHosted(),
		"base_url", c.baseURL,
		"error", cause,
	)
}

// readAudioForUpload loads the audio file with the 9R-3 size guard applied
// BEFORE the read, so an oversized file is refused rather than buffered.
func (c *WhisperClient) readAudioForUpload(audioPath string) ([]byte, error) {
	file, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("whisper: open audio file: %w", err)
	}
	defer file.Close()

	// Fail loudly instead of silently truncating (9R-3): oversized input here
	// means the chunking layer misbehaved — truncated audio would silently
	// lose dialogue.
	if info, err := file.Stat(); err == nil && info.Size() > WhisperMaxFileSize {
		return nil, fmt.Errorf("whisper: audio file %q is %d bytes, exceeds the %d-byte API limit — chunking failed upstream", filepath.Base(audioPath), info.Size(), int64(WhisperMaxFileSize))
	}

	audio, err := io.ReadAll(io.LimitReader(file, WhisperMaxFileSize))
	if err != nil {
		return nil, fmt.Errorf("whisper: read audio data: %w", err)
	}
	return audio, nil
}

// buildTranscribeBody assembles the multipart payload for one response format.
// Built per format because response_format is a form field: the verbose→srt
// fallback needs a second body, not a second file read.
func (c *WhisperClient) buildTranscribeBody(audio []byte, filename, lang, format string) ([]byte, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, "", fmt.Errorf("whisper: create form file: %w", err)
	}
	if _, err := part.Write(audio); err != nil {
		return nil, "", fmt.Errorf("whisper: copy audio data: %w", err)
	}

	if err := writer.WriteField("model", c.model); err != nil {
		return nil, "", fmt.Errorf("whisper: write model field: %w", err)
	}
	if err := writer.WriteField("response_format", format); err != nil {
		return nil, "", fmt.Errorf("whisper: write format field: %w", err)
	}
	// Pin language when known — avoids unreliable auto-detection (e.g. an English
	// episode mis-detected as Chinese due to a few seconds of background TV audio).
	if lang != "" {
		if err := writer.WriteField("language", lang); err != nil {
			return nil, "", fmt.Errorf("whisper: write language field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("whisper: close writer: %w", err)
	}
	return body.Bytes(), writer.FormDataContentType(), nil
}

// postTranscription performs one transcription request under the shared
// throttle + per-run budget (9R-11) with bounded transient retry (9R-4). The
// returned status is the last HTTP status observed (0 when the request never
// got a response), which is how the caller tells "this engine does not support
// verbose_json" (4xx) from "the network hiccuped" (5xx / timeout).
func (c *WhisperClient) postTranscription(ctx context.Context, audio []byte, filename, lang, format string) (string, int, error) {
	bodyBytes, contentType, err := c.buildTranscribeBody(audio, filename, lang, format)
	if err != nil {
		return "", 0, err
	}

	lastStatus := 0
	out, err := governed(ctx, c.governor, "whisper.transcribe", func() (string, error) {
		return retryTransient(ctx, "whisper.transcribe", func() (string, bool, error) {
			attemptCtx, cancel := context.WithTimeout(ctx, c.timeout)
			defer cancel()

			req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, c.baseURL, bytes.NewReader(bodyBytes))
			if err != nil {
				return "", false, fmt.Errorf("whisper: create request: %w", err)
			}
			// Omit the header entirely rather than sending a bare `Bearer `
			// (sub-5-2 AC #3): an empty credential is malformed and some
			// self-hosted engines reject it outright.
			if c.apiKey != "" {
				req.Header.Set("Authorization", "Bearer "+c.apiKey)
			}
			req.Header.Set("Content-Type", contentType)

			resp, err := c.httpClient.Do(req)
			if err != nil {
				if attemptCtx.Err() == context.DeadlineExceeded {
					return "", true, ErrWhisperTimeout
				}
				return "", true, fmt.Errorf("%w: %v", ErrWhisperAPIError, err)
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(io.LimitReader(resp.Body, WhisperMaxResponseSize))
			if err != nil {
				return "", true, fmt.Errorf("whisper: read response: %w", err)
			}

			lastStatus = resp.StatusCode
			if resp.StatusCode != http.StatusOK {
				c.logger.Warn("Whisper API error",
					"status_code", resp.StatusCode,
					"body", string(respBody),
				)
				return "", isTransientStatus(resp.StatusCode), fmt.Errorf("%w: status %d — %s", ErrWhisperAPIError, resp.StatusCode, string(respBody))
			}

			return string(respBody), false, nil
		})
	})
	if err != nil {
		return "", lastStatus, err
	}
	return out, lastStatus, nil
}

// NeedsChunking returns true if the audio file exceeds the per-chunk size
// budget (9R-3: budget = API limit minus multipart headroom, so the decision
// agrees with what SplitAudioChunks produces and what Transcribe can send).
func NeedsChunking(audioPath string) (bool, error) {
	info, err := os.Stat(audioPath)
	if err != nil {
		return false, fmt.Errorf("stat audio file: %w", err)
	}
	return info.Size() > WhisperChunkTargetBytes, nil
}

// SplitAudioChunks splits a WAV file into chunks that each fit the per-chunk
// size budget. It returns the chunk paths AND the chunk duration in seconds
// actually used (callers MUST pass that value to MergeSRTChunks so merged
// timestamps stay contiguous — 9R-3). Caller is responsible for cleanup.
//
// 9R-3: the split decision is SIZE-consistent with NeedsChunking — the chunk
// duration is derived from the WAV byte rate so that duration*byteRate stays
// under WhisperChunkTargetBytes, and the duration itself comes from a
// chunk-walking WAV parser that tolerates ffmpeg's extra header chunks (the
// old fixed-offset read misparsed those headers, skipped splitting, and sent
// the whole oversized file -> HTTP 413).
func SplitAudioChunks(ctx context.Context, audioPath string) ([]string, int, error) {
	info, err := os.Stat(audioPath)
	if err != nil {
		return nil, 0, fmt.Errorf("stat audio file: %w", err)
	}

	duration, byteRate, err := parseWAVInfo(audioPath)
	if err != nil {
		return nil, 0, fmt.Errorf("get audio duration: %w", err)
	}

	// Chunk seconds bounded by BOTH the nominal duration cap and the size
	// budget (guards against byte rates higher than the expected 16kHz mono).
	chunkSeconds := WhisperChunkDuration
	if byteRate > 0 {
		if maxSec := int(uint32(WhisperChunkTargetBytes) / byteRate); maxSec < chunkSeconds {
			chunkSeconds = maxSec
		}
	}
	if chunkSeconds < 1 {
		chunkSeconds = 1
	}

	if info.Size() <= WhisperChunkTargetBytes && duration <= float64(chunkSeconds) {
		return []string{audioPath}, chunkSeconds, nil
	}

	var chunks []string
	for start := 0; start < int(duration); start += chunkSeconds {
		chunkFile, err := os.CreateTemp("", fmt.Sprintf("vido-chunk-%d-*.wav", start))
		if err != nil {
			// Cleanup already created chunks
			for _, c := range chunks {
				os.Remove(c)
			}
			return nil, 0, fmt.Errorf("create chunk temp file: %w", err)
		}
		chunkPath := chunkFile.Name()
		chunkFile.Close()

		//nolint:gosec // audioPath comes from our own temp extraction
		cmd := execCommandContext(ctx, "ffmpeg",
			"-i", audioPath,
			"-ss", fmt.Sprintf("%d", start),
			"-t", fmt.Sprintf("%d", chunkSeconds),
			"-acodec", "pcm_s16le",
			"-ar", "16000",
			"-ac", "1",
			"-y",
			chunkPath,
		)

		if output, err := cmd.CombinedOutput(); err != nil {
			for _, c := range chunks {
				os.Remove(c)
			}
			os.Remove(chunkPath)
			return nil, 0, fmt.Errorf("ffmpeg chunk split at %ds: %w — %s", start, err, string(output))
		}

		// Defensive: never hand an oversized chunk to the API (the 413 class).
		if ci, err := os.Stat(chunkPath); err == nil && ci.Size() > WhisperMaxFileSize {
			for _, c := range chunks {
				os.Remove(c)
			}
			os.Remove(chunkPath)
			return nil, 0, fmt.Errorf("chunk at %ds is %d bytes, exceeds Whisper %d-byte limit", start, ci.Size(), int64(WhisperMaxFileSize))
		}

		chunks = append(chunks, chunkPath)
	}

	return chunks, chunkSeconds, nil
}

// execCommandContext wraps exec.CommandContext to allow testing
var execCommandContext = execCommandContextReal

func execCommandContextReal(ctx context.Context, name string, args ...string) command {
	return execCmd{exec.CommandContext(ctx, name, args...)}
}

// command interface for testing
type command interface {
	CombinedOutput() ([]byte, error)
}

type execCmd struct {
	*exec.Cmd
}

// MergeSRTChunks merges multiple SRT strings from chunked transcription, adjusting timestamps.
func MergeSRTChunks(chunks []string, chunkDuration int) string {
	if len(chunks) == 0 {
		return ""
	}
	if len(chunks) == 1 {
		return chunks[0]
	}

	var merged bytes.Buffer
	seqNum := 1

	for i, chunk := range chunks {
		offsetSeconds := i * chunkDuration
		adjusted := adjustSRTTimestamps(chunk, offsetSeconds, &seqNum)
		merged.WriteString(adjusted)
	}

	return merged.String()
}

// getWAVDuration calculates the audio duration of a WAV file.
func getWAVDuration(path string) (float64, error) {
	duration, _, err := parseWAVInfo(path)
	return duration, err
}

// parseWAVInfo walks the RIFF chunk list to find the fmt and data chunks
// (9R-3: header-robust — ffmpeg and other muxers may emit extra chunks such
// as LIST/INFO between fmt and data, which breaks fixed-offset header reads
// and silently yields a garbage duration).
func parseWAVInfo(path string) (duration float64, byteRate uint32, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	riff := make([]byte, 12)
	if _, err := io.ReadFull(f, riff); err != nil {
		return 0, 0, fmt.Errorf("read WAV header: %w", err)
	}
	if string(riff[0:4]) != "RIFF" || string(riff[8:12]) != "WAVE" {
		return 0, 0, fmt.Errorf("not a WAV file")
	}

	var dataSize uint32
	chunkHdr := make([]byte, 8)
	for {
		if _, err := io.ReadFull(f, chunkHdr); err != nil {
			break // end of file — evaluate what we found below
		}
		id := string(chunkHdr[0:4])
		size := binary.LittleEndian.Uint32(chunkHdr[4:8])

		switch id {
		case "fmt ":
			fmtChunk := make([]byte, size)
			if _, err := io.ReadFull(f, fmtChunk); err != nil {
				return 0, 0, fmt.Errorf("read fmt chunk: %w", err)
			}
			if size >= 12 {
				byteRate = binary.LittleEndian.Uint32(fmtChunk[8:12])
			}
		case "data":
			dataSize = size
			// data payload does not need to be read for duration math.
			if _, err := f.Seek(int64(size), io.SeekCurrent); err != nil {
				return 0, 0, fmt.Errorf("seek past data chunk: %w", err)
			}
		default:
			if _, err := f.Seek(int64(size), io.SeekCurrent); err != nil {
				return 0, 0, fmt.Errorf("seek past %q chunk: %w", id, err)
			}
		}
		// RIFF chunks are word-aligned: odd sizes are padded with one byte.
		if size%2 == 1 {
			if _, err := f.Seek(1, io.SeekCurrent); err != nil {
				break
			}
		}
	}

	if byteRate == 0 {
		return 0, 0, fmt.Errorf("invalid WAV byte rate")
	}
	if dataSize == 0 {
		return 0, 0, fmt.Errorf("WAV data chunk not found")
	}

	return float64(dataSize) / float64(byteRate), byteRate, nil
}

// adjustSRTTimestamps adjusts SRT timestamp lines by an offset and renumbers sequences.
func adjustSRTTimestamps(srt string, offsetSeconds int, seqNum *int) string {
	if offsetSeconds == 0 && *seqNum == 1 {
		// First chunk, no adjustment needed; just count sequences
		result := &bytes.Buffer{}
		lines := splitLines(srt)
		for i := 0; i < len(lines); i++ {
			line := lines[i]
			// Check if this is a sequence number line (digits only, followed by timestamp line)
			if isSequenceNumber(line) && i+1 < len(lines) && isTimestampLine(lines[i+1]) {
				fmt.Fprintf(result, "%d\n", *seqNum)
				*seqNum++
				continue
			}
			result.WriteString(line)
			result.WriteByte('\n')
		}
		return result.String()
	}

	result := &bytes.Buffer{}
	lines := splitLines(srt)
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if isSequenceNumber(line) && i+1 < len(lines) && isTimestampLine(lines[i+1]) {
			fmt.Fprintf(result, "%d\n", *seqNum)
			*seqNum++
			continue
		}
		if isTimestampLine(line) {
			adjusted := offsetTimestampLine(line, offsetSeconds)
			result.WriteString(adjusted)
			result.WriteByte('\n')
			continue
		}
		result.WriteString(line)
		result.WriteByte('\n')
	}
	return result.String()
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func isSequenceNumber(line string) bool {
	if len(line) == 0 {
		return false
	}
	for _, c := range line {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isTimestampLine(line string) bool {
	// SRT timestamp format: 00:00:00,000 --> 00:00:00,000
	return len(line) >= 29 && line[2] == ':' && line[5] == ':' && line[8] == ','
}

func offsetTimestampLine(line string, offsetSeconds int) string {
	// Parse: 00:00:00,000 --> 00:00:00,000
	if len(line) < 29 {
		return line
	}

	start := parseSRTTimestamp(line[0:12])
	end := parseSRTTimestamp(line[17:29])

	start += offsetSeconds * 1000
	end += offsetSeconds * 1000

	return fmt.Sprintf("%s --> %s", formatSRTTimestamp(start), formatSRTTimestamp(end))
}

// parseSRTTimestamp parses "HH:MM:SS,mmm" to milliseconds.
func parseSRTTimestamp(ts string) int {
	if len(ts) < 12 {
		return 0
	}
	// Validate digit positions to prevent garbage output from malformed SRT
	for _, i := range []int{0, 1, 3, 4, 6, 7, 9, 10, 11} {
		if ts[i] < '0' || ts[i] > '9' {
			return 0
		}
	}
	h := int(ts[0]-'0')*10 + int(ts[1]-'0')
	m := int(ts[3]-'0')*10 + int(ts[4]-'0')
	s := int(ts[6]-'0')*10 + int(ts[7]-'0')
	ms := int(ts[9]-'0')*100 + int(ts[10]-'0')*10 + int(ts[11]-'0')
	return h*3600000 + m*60000 + s*1000 + ms
}

// formatSRTTimestamp formats milliseconds to "HH:MM:SS,mmm".
func formatSRTTimestamp(ms int) string {
	h := ms / 3600000
	ms %= 3600000
	m := ms / 60000
	ms %= 60000
	s := ms / 1000
	ms %= 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}
