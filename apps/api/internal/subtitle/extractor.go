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

// defaultExtractTimeout is the configured floor of a single ffmpeg demux pass
// when SUBTITLE_EXTRACT_TIMEOUT_SECONDS is unset (sub-6-3 AC #1). It is a
// FLOOR, not the bound: the effective deadline is max(configured, file size ×
// defaultPerGBTimeout), because a 93 GB remux is pure I/O for far longer than
// ten minutes on a NAS spindle (eval-1 product problem 3 — Goodfellas timed
// out on a constant that was tuned for 20 GB).
const defaultExtractTimeout = 10 * time.Minute

// defaultPerGBTimeout is the size-aware allowance: 30 s per GB of media, the
// rate a DS920+-class NAS demuxes a remux at with headroom (93 GB → ~46 min).
// Overridable per Extractor (WithPerGBTimeout) for tests and faster hardware.
const defaultPerGBTimeout = 30 * time.Second

// bytesPerGB is the decimal gigabyte ffmpeg users think in.
const bytesPerGB = 1_000_000_000

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

// isChineseTag recognises the tags that may carry Chinese cue text.
//
// ffprobe emits ISO 639-2/B for Matroska, so a Chinese track is almost always
// the bare `chi` — the 繁/簡 distinction lives in the track TITLE ("Chinese
// (Traditional)"), which vido does not persist. That is fine: the tag only has
// to get the track EXTRACTED; `Detect` then classifies the variant from content
// (FR6), which is the same mislabel-proof rule the English path relies on.
//
// `yue` (Cantonese) is deliberately EXCLUDED. Cantonese subtitles are written in
// Traditional script, so the detector would happily label them zh-Hant and we
// would ship colloquial Cantonese as a Mandarin 繁中 track. When a file offers
// nothing but Cantonese, falling through to English-translate is the better
// answer. A Cantonese track tagged plainly `chi` (as Apple TV+ does) is still
// admitted — it is indistinguishable without the title tag — but it loses the
// tie-break to any track that is genuinely Traditional.
func isChineseTag(lang string) bool {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "chi", "zho", "zh", "zh-hans", "zh-hant", "zh-cn", "zh-tw", "zh-hk", "chs", "cht":
		return true
	default:
		return false
	}
}

// SelectCandidates filters probed tracks down to the ones M1 may extract:
// embedded (not a sidecar) and a text codec, tagged either Chinese or eng/en.
// Probe order is preserved so downstream tie-breaks stay deterministic.
//
// Chinese tracks WIN OUTRIGHT when present: a track that is already Chinese
// needs at most an OpenCC conversion, while the English path pays an LLM per
// item to produce a worse result than the human subtitle sitting in the same
// file. Live evidence (2026-07-31, owner's NAS): 135 of 157 ffprobe-scanned
// items carry BOTH an official Chinese track and an English one, i.e. the
// English-only gate sent 86% of the library down the expensive, lower-quality
// branch. The tiers are exclusive — when Chinese is present the English tracks
// are not even extracted, so this also removes ffmpeg work.
func SelectCandidates(tracks []services.SubtitleTrack) []services.SubtitleTrack {
	var chinese, english []services.SubtitleTrack
	for _, t := range tracks {
		if t.External {
			continue
		}
		if !IsTextSubtitleCodec(t.Format) {
			continue
		}
		switch {
		case isChineseTag(t.Language):
			chinese = append(chinese, t)
		case isEnglishTag(t.Language):
			english = append(english, t)
		}
	}
	if len(chinese) > 0 {
		return chinese
	}
	return english
}

// Extractor demuxes embedded text subtitle tracks into .srt files with ffmpeg.
// Construction mirrors services.FFprobeService: availability is probed once at
// startup and the instance is reused (Rule 14).
//
// Extraction is SERIALIZED process-wide through an ExtractGate (sub-6-3
// AC #2). This reverses the original ruling that the orchestrator's fixed
// concurrency of 2 (NFR-P3) was the only bound and that "a second one here
// would silently halve it": eval-1 finding 7 measured two concurrent 20 GB
// demuxes on the owner's NAS fighting over the same spindle, BOTH running past
// the ceiling and failing, while either alone took 3.5 minutes. Translation is
// not gated — only the ffmpeg subprocess is — so the two workers still
// translate concurrently; only their disk reads take turns.
type Extractor struct {
	timeout      time.Duration
	perGBTimeout time.Duration
	gate         *ExtractGate
	// fileSize answers "how big is this media file" for the size-aware
	// deadline; os.Stat in production, injectable so a test can describe a
	// 93 GB file without creating one.
	fileSize  func(path string) (int64, error)
	available bool
	logger    *slog.Logger
}

// ExtractorOption configures one optional knob of NewExtractor.
type ExtractorOption func(*Extractor)

// WithExtractGate shares one process-wide gate across every Extractor
// (Rule 14 — main.go builds it once). Without it each Extractor serializes
// only against itself, which is still correct for a single instance.
func WithExtractGate(g *ExtractGate) ExtractorOption {
	return func(e *Extractor) {
		if g != nil {
			e.gate = g
		}
	}
}

// WithPerGBTimeout overrides the size-aware allowance per gigabyte. Non-positive
// values are ignored.
func WithPerGBTimeout(d time.Duration) ExtractorOption {
	return func(e *Extractor) {
		if d > 0 {
			e.perGBTimeout = d
		}
	}
}

// withFileSize overrides the size lookup (tests).
func withFileSize(fn func(path string) (int64, error)) ExtractorOption {
	return func(e *Extractor) {
		if fn != nil {
			e.fileSize = fn
		}
	}
}

// NewExtractor creates an Extractor, checking for ffmpeg via exec.LookPath.
// timeout is the configured floor (SUBTITLE_EXTRACT_TIMEOUT_SECONDS); <= 0
// means defaultExtractTimeout.
func NewExtractor(timeout time.Duration, logger *slog.Logger, opts ...ExtractorOption) *Extractor {
	if logger == nil {
		logger = slog.Default()
	}
	if timeout <= 0 {
		timeout = defaultExtractTimeout
	}

	e := &Extractor{
		timeout:      timeout,
		perGBTimeout: defaultPerGBTimeout,
		gate:         NewExtractGate(),
		fileSize:     statFileSize,
		logger:       logger.With("service", "subtitle_extractor"),
	}
	for _, opt := range opts {
		opt(e)
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		e.logger.Warn("ffmpeg not found — embedded subtitle extraction disabled")
		e.available = false
	} else {
		e.available = true
		e.logger.Info("ffmpeg available", "timeout", timeout, "per_gb_timeout", e.perGBTimeout)
	}

	return e
}

// IsAvailable reports whether ffmpeg is installed and usable.
func (e *Extractor) IsAvailable() bool {
	return e.available
}

func statFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// EffectiveTimeout is the deadline one ffmpeg pass over mediaPath gets
// (sub-6-3 AC #1): max(configured floor, size × per-GB allowance). A file
// whose size cannot be read gets the floor — the same bound as before this
// story, never less. The size is returned alongside for the caller's log
// line; 0 when unknown.
func (e *Extractor) EffectiveTimeout(mediaPath string) (time.Duration, float64) {
	size, err := e.fileSize(mediaPath)
	if err != nil || size <= 0 {
		return e.timeout, 0
	}
	gb := float64(size) / bytesPerGB
	sized := time.Duration(gb * float64(e.perGBTimeout))
	if sized > e.timeout {
		return sized, gb
	}
	return e.timeout, gb
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

	timeout, sizeGB := e.EffectiveTimeout(mediaPath)
	base := filepath.Base(mediaPath)

	// The gate wraps ONLY the ffmpeg subprocess — not the ffprobe that
	// preceded this call (a header read) and not the deadline below, which
	// must measure the demux, not the queue. The item flow's ctx may carry a
	// notifier so the user sees「等待抽軌（前方 N 件）」rather than a stalled
	// "extracting" bubble.
	// A ctx that has ALREADY ended is a plain cancellation (the sub-1-4 CR M2
	// shape), not a wait that was given up on — the gate is only asked once
	// there is something to wait for.
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, extractFailure(fmt.Sprintf("ffmpeg cancelled on %s", base), ctxErr, "")
	}
	release, err := e.gate.Acquire(ctx, extractWaitNotifierFrom(ctx))
	if err != nil {
		return nil, fmt.Errorf("%w: %s (queued behind %d): %w",
			ErrSubtitleExtractWaitAborted, base, e.gate.Waiting()+1, err)
	}
	defer release()

	extractCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := buildExtractArgs(mediaPath, tmpDir, streamIndexes)

	started := time.Now()
	e.logger.Info("embedded subtitle extraction started",
		"media", base,
		"file_size_gb", fmt.Sprintf("%.1f", sizeGB),
		"timeout_seconds", int(timeout.Seconds()),
		"stream_indexes", streamIndexes,
	)

	//nolint:gosec // mediaPath comes from a trusted DB record; tmpDir is caller-owned
	cmd := exec.CommandContext(extractCtx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// extractCtx.Err() is also non-nil when the PARENT ctx was cancelled
		// (shutdown, user abort) — only a real deadline is a timeout.
		if ctxErr := extractCtx.Err(); ctxErr != nil {
			if errors.Is(ctxErr, context.DeadlineExceeded) {
				// The deadline can also be the PARENT's (the free lane's
				// per-item bound): report ours only when it was the one
				// that fired, so the message never blames a knob that did
				// not apply.
				if ctx.Err() == nil {
					return nil, extractFailure(
						fmt.Sprintf("ffmpeg timed out after %s on %s (file %.1f GB, timeout %d s — raise SUBTITLE_EXTRACT_TIMEOUT_SECONDS for slow disks)",
							timeout, base, sizeGB, int(timeout.Seconds())),
						ctxErr, stderr.String())
				}
				return nil, extractFailure(
					fmt.Sprintf("ffmpeg stopped by the caller's deadline after %s on %s (file %.1f GB)",
						time.Since(started).Round(time.Second), base, sizeGB),
					ctxErr, stderr.String())
			}
			return nil, extractFailure(
				fmt.Sprintf("ffmpeg cancelled on %s", base),
				ctxErr, stderr.String())
		}
		return nil, extractFailure(
			fmt.Sprintf("ffmpeg exec on %s", base),
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

	e.logger.Info("embedded subtitle tracks extracted",
		"media", base,
		"track_count", len(outputs),
		"stream_indexes", streamIndexes,
		"file_size_gb", fmt.Sprintf("%.1f", sizeGB),
		"timeout_seconds", int(timeout.Seconds()),
		"elapsed", time.Since(started).Round(time.Millisecond).String(),
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
