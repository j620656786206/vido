package subtitle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vido/api/internal/services"
)

// defaultExtractTimeout bounds a single ffmpeg demux pass. A 20 GB remux on a
// DS920+ is minutes of pure I/O, so the ceiling is generous by design; callers
// that know better pass their own.
const defaultExtractTimeout = 10 * time.Minute

// stderrTailBytes caps how much ffmpeg stderr is carried into the wrapped error
// message (Rule 13 — context without unbounded log lines).
const stderrTailBytes = 512

// textSubtitleCodecs are the ffprobe codec names that carry extractable cue
// text. `mov_text` is MP4's flavour; `webvtt` appears in web-sourced remuxes.
var textSubtitleCodecs = map[string]struct{}{
	"subrip":   {},
	"srt":      {},
	"ass":      {},
	"ssa":      {},
	"mov_text": {},
	"webvtt":   {},
}

// imageSubtitleCodecs are bitmap tracks. They are never extraction candidates —
// turning them into text needs OCR, which M1 does not do (FR5 routes them to
// "no usable text source"). Listed explicitly so the classification is auditable
// rather than "anything not in the text set".
var imageSubtitleCodecs = map[string]struct{}{
	"hdmv_pgs_subtitle": {},
	"dvd_subtitle":      {},
	"dvb_subtitle":      {},
	"xsub":              {},
}

// IsTextSubtitleCodec reports whether an ffprobe codec name carries cue text.
func IsTextSubtitleCodec(codec string) bool {
	_, ok := textSubtitleCodecs[strings.ToLower(strings.TrimSpace(codec))]
	return ok
}

// IsImageSubtitleCodec reports whether an ffprobe codec name is a bitmap track.
func IsImageSubtitleCodec(codec string) bool {
	_, ok := imageSubtitleCodecs[strings.ToLower(strings.TrimSpace(codec))]
	return ok
}

// isEnglishTag implements the P0 language gate: M1 accepts ONLY tracks tagged
// eng/en. `und` is NEVER treated as English — an untagged track is exactly the
// case where guessing mistranslates, so the pipeline fails closed.
func isEnglishTag(lang string) bool {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "eng", "en":
		return true
	default:
		return false
	}
}

// SelectCandidates filters probed tracks down to the ones M1 may extract:
// embedded (not a sidecar), a text codec, and tagged eng/en. Probe order is
// preserved so downstream tie-breaks stay deterministic.
func SelectCandidates(tracks []services.SubtitleTrack) []services.SubtitleTrack {
	var candidates []services.SubtitleTrack
	for _, t := range tracks {
		if t.External {
			continue
		}
		if !IsTextSubtitleCodec(t.Format) {
			continue
		}
		if !isEnglishTag(t.Language) {
			continue
		}
		candidates = append(candidates, t)
	}
	return candidates
}

// Extractor demuxes embedded text subtitle tracks into .srt files with ffmpeg.
// Construction mirrors services.FFprobeService: availability is probed once at
// startup and the instance is reused (Rule 14). There is deliberately NO
// internal semaphore — the orchestrator's fixed concurrency of 2 (NFR-P3) is
// the only bound, and a second one here would silently halve it.
type Extractor struct {
	timeout   time.Duration
	available bool
	logger    *slog.Logger
}

// NewExtractor creates an Extractor, checking for ffmpeg via exec.LookPath.
func NewExtractor(timeout time.Duration, logger *slog.Logger) *Extractor {
	if logger == nil {
		logger = slog.Default()
	}
	if timeout <= 0 {
		timeout = defaultExtractTimeout
	}

	e := &Extractor{
		timeout: timeout,
		logger:  logger.With("service", "subtitle_extractor"),
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		e.logger.Warn("ffmpeg not found — embedded subtitle extraction disabled")
		e.available = false
	} else {
		e.available = true
		e.logger.Info("ffmpeg available", "timeout", timeout)
	}

	return e
}

// IsAvailable reports whether ffmpeg is installed and usable.
func (e *Extractor) IsAvailable() bool {
	return e.available
}

// Extract demuxes every requested stream index in ONE ffmpeg invocation (FR3 —
// a single read pass over the media) and returns stream index → extracted .srt
// path. Output ALWAYS lands in tmpDir, never beside the media: placer.go is the
// sole writer into the media folder (D3), and a stray `.eng.srt` sidecar would
// be auto-detected by Plex/Jellyfin/Video Station.
//
// Failure (non-zero exit, timeout, or a missing output file) wraps
// ErrSubtitleExtractFailed with an ffmpeg stderr tail.
func (e *Extractor) Extract(ctx context.Context, mediaPath, tmpDir string, streamIndexes []int) (map[int]string, error) {
	if !e.available {
		return nil, fmt.Errorf("subtitle extract: %w", services.ErrFFmpegNotAvailable)
	}
	if len(streamIndexes) == 0 {
		return nil, fmt.Errorf("subtitle extract: no stream indexes supplied for %s", filepath.Base(mediaPath))
	}

	extractCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	args := buildExtractArgs(mediaPath, tmpDir, streamIndexes)

	//nolint:gosec // mediaPath comes from a trusted DB record; tmpDir is caller-owned
	cmd := exec.CommandContext(extractCtx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// extractCtx.Err() is also non-nil when the PARENT ctx was cancelled
		// (shutdown, user abort) — only a real deadline is a timeout.
		if ctxErr := extractCtx.Err(); ctxErr != nil {
			if errors.Is(ctxErr, context.DeadlineExceeded) {
				return nil, extractFailure(
					fmt.Sprintf("ffmpeg timed out after %s on %s", e.timeout, filepath.Base(mediaPath)),
					ctxErr, stderr.String())
			}
			return nil, extractFailure(
				fmt.Sprintf("ffmpeg cancelled on %s", filepath.Base(mediaPath)),
				ctxErr, stderr.String())
		}
		return nil, extractFailure(
			fmt.Sprintf("ffmpeg exec on %s", filepath.Base(mediaPath)),
			err, stderr.String())
	}

	outputs := make(map[int]string, len(streamIndexes))
	for _, idx := range streamIndexes {
		out := trackOutputPath(tmpDir, idx)
		if _, err := os.Stat(out); err != nil {
			return nil, extractFailure(
				fmt.Sprintf("ffmpeg produced no output for stream %d of %s", idx, filepath.Base(mediaPath)),
				err, stderr.String())
		}
		outputs[idx] = out
	}

	e.logger.Debug("embedded subtitle tracks extracted",
		"media", filepath.Base(mediaPath),
		"track_count", len(outputs),
		"stream_indexes", streamIndexes,
	)

	return outputs, nil
}

// buildExtractArgs assembles the single ffmpeg invocation. Every candidate uses
// `-c:s srt`, never `-c:s copy`: copy into an .srt muxer works only when the
// source codec is already subrip, so ass/ssa/mov_text would fail. This is a
// text-format transform, NOT a media re-encode — video and audio streams are
// never selected (`-map 0:{n}` picks subtitle streams only), so FR2's
// "no re-encode" guarantee holds.
func buildExtractArgs(mediaPath, tmpDir string, streamIndexes []int) []string {
	args := []string{"-nostdin", "-y", "-i", mediaPath}
	for _, idx := range streamIndexes {
		args = append(args,
			"-map", fmt.Sprintf("0:%d", idx),
			"-c:s", "srt",
			trackOutputPath(tmpDir, idx),
		)
	}
	return args
}

// trackOutputPath is the caller-owned temp destination for one extracted track.
func trackOutputPath(tmpDir string, streamIndex int) string {
	return filepath.Join(tmpDir, fmt.Sprintf("track_%d.srt", streamIndex))
}

// extractFailure wraps a failure as the sub-1-3 ErrSubtitleExtractFailed
// sentinel so callers classify with errors.Is, and CHAINS the cause with a
// second %w — the orchestrator (1.5b) must be able to tell a cancellation
// (errors.Is(err, context.Canceled)) apart from a genuine ffmpeg failure.
// The ffmpeg stderr tail rides in the message (Rule 13).
func extractFailure(context string, cause error, stderr string) error {
	msg := context
	if tail := stderrTail(stderr); tail != "" {
		msg = fmt.Sprintf("%s (ffmpeg stderr: %s)", msg, tail)
	}
	if cause != nil {
		return fmt.Errorf("%w: %s: %w", ErrSubtitleExtractFailed, msg, cause)
	}
	return fmt.Errorf("%w: %s", ErrSubtitleExtractFailed, msg)
}

// stderrTail returns the trailing stderrTailBytes of ffmpeg's stderr — the tail
// is where the actual failure line sits, ahead of it is banner noise. The cut
// is advanced to a rune boundary so a non-ASCII filename in ffmpeg's output
// cannot leave invalid UTF-8 in the error message or logs.
func stderrTail(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > stderrTailBytes {
		s = s[len(s)-stderrTailBytes:]
		for len(s) > 0 && !utf8.RuneStart(s[0]) {
			s = s[1:]
		}
	}
	return s
}
