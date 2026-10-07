package subtitle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// Implements: disc-2026-10-episode-list-subtitle-badge-c AC #3

// ScriptPeeker tells Traditional from Simplified for an embedded Chinese text
// track whose tag ("chi") and title do not say, by pulling a few minutes of
// it out with ffmpeg and reading the characters (Detect — the same rule the
// sidecar files go through).
//
// It is deliberately NOT the Extractor: that one demuxes a whole track behind
// the process-wide ExtractGate (two whole-file demuxes on one spindle both
// time out). A peek is an input-seeked read of a short window, seconds long;
// queueing it behind a 40-minute extraction would stall the whole background
// sweep (story SM ruling 3).
type ScriptPeeker struct {
	available bool
	timeout   time.Duration
	run       func(ctx context.Context, args []string) error
	logger    *slog.Logger
}

// peekWindow is one [start, start+duration) slice of the track, in seconds.
type peekWindow struct{ start, duration int }

// peekWindows: minutes 4–6.5 first (the opening is often titles and music,
// with no dialogue), then the first four minutes when that window had no
// Chinese characters (a short episode, or dialogue only at the top).
var peekWindows = []peekWindow{{240, 150}, {0, 240}}

// peekTimeout bounds one ffmpeg call. A seek plus 2.5 minutes of a subtitle
// stream reads the packets interleaved in that span — on a NAS 4K remux a few
// seconds; a minute means the disk or the file is in trouble.
const peekTimeout = 60 * time.Second

// peekMinEvidence is how many script-unique characters (Simplified-only plus
// Traditional-only) a sample needs before its ratio is trusted. Two and a
// half minutes of dialogue carry dozens; a handful means the window caught
// a few short lines.
const peekMinEvidence = 8

// NewScriptPeeker checks for ffmpeg once.
func NewScriptPeeker(logger *slog.Logger) *ScriptPeeker {
	if logger == nil {
		logger = slog.Default()
	}
	_, err := exec.LookPath("ffmpeg")
	return &ScriptPeeker{
		available: err == nil,
		timeout:   peekTimeout,
		run:       runFFmpeg,
		logger:    logger.With("service", "subtitle_script_peeker"),
	}
}

func runFFmpeg(ctx context.Context, args []string) error {
	//nolint:gosec // mediaPath comes from a trusted DB record; the output is our temp dir
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.WaitDelay = extractWaitDelay
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, stderrTail(stderr.String()))
	}
	return nil
}

// IsAvailable reports whether ffmpeg is installed.
func (p *ScriptPeeker) IsAvailable() bool { return p != nil && p.available }

// Peekable reports whether a track is worth a peek: embedded, a text codec
// (bitmap tracks have no characters to read), Chinese, and its script not
// already told by the tag or title.
func (p *ScriptPeeker) Peekable(t services.SubtitleTrack) bool {
	return !t.External && IsTextSubtitleCodec(t.Format) &&
		models.ChineseSubtitleOfTrack(t.Language, t.Title, "") == models.ChineseSubtitleZh
}

// PeekScript returns "zh-Hant" or "zh-Hans" for the track at streamIndex;
// "zh" for a genuinely mixed text, "und" when neither window had a Chinese
// character. Both of the last two are answers — "we looked and it does not
// tell" — so the sweep stores them and never samples that file again. An
// error means ffmpeg itself failed.
func (p *ScriptPeeker) PeekScript(ctx context.Context, mediaPath string, streamIndex int) (string, error) {
	if !p.IsAvailable() {
		return "", fmt.Errorf("peek script: %w", services.ErrFFmpegNotAvailable)
	}
	dir, err := os.MkdirTemp("", "vido-peek-*")
	if err != nil {
		return "", fmt.Errorf("peek script: temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	// The windows' text accumulates: a window decides only once the sample
	// holds enough characters that exist in just one script (CR M1/M2 — one
	// stray 个 is not a Simplified track, and a few lines of 好／你好嗎 are not
	// "mixed"). A thin first window adds its characters to the second's.
	var sample []byte
	for i, w := range peekWindows {
		out := filepath.Join(dir, fmt.Sprintf("window_%d.srt", i))
		args := []string{
			"-nostdin", "-y", "-v", "error",
			// -ss BEFORE -i: seek by the container index instead of decoding
			// everything up to the window.
			"-ss", strconv.Itoa(w.start),
			"-i", mediaPath,
			"-t", strconv.Itoa(w.duration),
			"-map", fmt.Sprintf("0:%d", streamIndex),
			"-c:s", "srt",
			out,
		}
		runCtx, cancel := context.WithTimeout(ctx, p.timeout)
		err := p.run(runCtx, args)
		cancel()
		if err != nil {
			return "", fmt.Errorf("peek script of stream %d in %s: %w", streamIndex, filepath.Base(mediaPath), err)
		}
		content, err := os.ReadFile(out)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("peek script: read sample: %w", err)
		}
		// A seek past the end of a short file leaves an empty (or no) file;
		// the next window gets its turn.
		sample = append(append(sample, content...), '\n')
		res := Detect(sample)
		if res.SimplifiedCount+res.TraditionalCount < peekMinEvidence {
			continue
		}
		switch res.Language {
		case LangTraditional, LangSimplified:
			return res.Language, nil
		}
		return LangAmbiguous, nil // genuinely mixed: a longer read would not settle it
	}
	if Detect(sample).TotalCJK > 0 {
		return LangAmbiguous, nil // Chinese, but too little of it to tell
	}
	return LangUndetermined, nil
}
