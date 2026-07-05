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
}

// WhisperOption is a functional option for configuring WhisperClient.
type WhisperOption func(*WhisperClient)

// WithWhisperBaseURL sets a custom base URL (useful for testing).
func WithWhisperBaseURL(url string) WhisperOption {
	return func(c *WhisperClient) {
		c.baseURL = url
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

// NewWhisperClient creates a new Whisper API client.
func NewWhisperClient(apiKey string, opts ...WhisperOption) *WhisperClient {
	c := &WhisperClient{
		apiKey:  apiKey,
		baseURL: WhisperAPIURL,
		timeout: 5 * time.Minute,
		logger:  slog.Default().With("service", "whisper"),
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
	if c.apiKey == "" {
		return "", ErrWhisperNotConfigured
	}

	// Build multipart form body once; per-attempt timeouts are applied inside
	// the retry loop below.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add audio file
	file, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("whisper: open audio file: %w", err)
	}
	defer file.Close()

	// Fail loudly instead of silently truncating (9R-3): oversized input here
	// means the chunking layer misbehaved — truncated audio would silently
	// lose dialogue.
	if info, err := file.Stat(); err == nil && info.Size() > WhisperMaxFileSize {
		return "", fmt.Errorf("whisper: audio file %q is %d bytes, exceeds the %d-byte API limit — chunking failed upstream", filepath.Base(audioPath), info.Size(), int64(WhisperMaxFileSize))
	}

	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return "", fmt.Errorf("whisper: create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("whisper: copy audio data: %w", err)
	}

	// Add model and response format fields
	if err := writer.WriteField("model", WhisperModel); err != nil {
		return "", fmt.Errorf("whisper: write model field: %w", err)
	}
	if err := writer.WriteField("response_format", "srt"); err != nil {
		return "", fmt.Errorf("whisper: write format field: %w", err)
	}
	// Pin language when known — avoids unreliable auto-detection (e.g. an English
	// episode mis-detected as Chinese due to a few seconds of background TV audio).
	if lang != "" {
		if err := writer.WriteField("language", lang); err != nil {
			return "", fmt.Errorf("whisper: write language field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("whisper: close writer: %w", err)
	}

	bodyBytes := body.Bytes()
	contentType := writer.FormDataContentType()

	c.logger.Debug("Whisper API request", "file", filepath.Base(audioPath))

	// Execute with bounded transient retry (9R-4): a single transient timeout
	// previously killed a full multi-chunk transcription run.
	srt, err := retryTransient(ctx, "whisper.transcribe", func() (string, bool, error) {
		attemptCtx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()

		req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, c.baseURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return "", false, fmt.Errorf("whisper: create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
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

		if resp.StatusCode != http.StatusOK {
			c.logger.Warn("Whisper API error",
				"status_code", resp.StatusCode,
				"body", string(respBody),
			)
			return "", isTransientStatus(resp.StatusCode), fmt.Errorf("%w: status %d — %s", ErrWhisperAPIError, resp.StatusCode, string(respBody))
		}

		return string(respBody), false, nil
	})
	if err != nil {
		return "", err
	}

	c.logger.Info("Whisper transcription complete", "file", filepath.Base(audioPath), "srt_bytes", len(srt))

	return srt, nil
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
