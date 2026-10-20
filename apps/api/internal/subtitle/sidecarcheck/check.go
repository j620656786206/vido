package sidecarcheck

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle/mine"
)

// windowCount × windowLen of audio is decoded per check (AC #3): 18 minutes
// of a long film. The exam ran 6, 8, 10 and 12; only 12 separated every real
// match from every wrong one with room to spare. A var so the exam can try
// other counts.
var windowCount = 12

const (
	windowLen = 90 * time.Second
	// minWindows: a film too short for this many windows is read whole.
	minWindows = 3
	// decodeTimeout bounds one window's ffmpeg run.
	decodeTimeout = 60 * time.Second

	// CacheType is the cache_entries type of a stored verdict (AC #6).
	CacheType = "sidecar_alignment"
	cacheTTL  = 30 * 24 * time.Hour
	// algorithmVersion is part of the cache key: changing the thresholds or
	// the scoring must not serve verdicts the old code reached.
	algorithmVersion = "v1"
)

// Verdict is what Check reports.
type Verdict struct {
	Result Result `json:"result"`
	// OffsetSeconds is the overall best shift; positive = the subtitle is
	// late. Meaningful for Aligned and Offset.
	OffsetSeconds float64 `json:"offset_seconds"`
	// Reason says why for Unusable and Mismatch ("too_few_cues",
	// "not_english", "not_latin", "unreadable", "no_peak", "windows_disagree").
	Reason  string         `json:"reason,omitempty"`
	Score   float64        `json:"score"`
	Windows []WindowResult `json:"windows,omitempty"`
	// ElapsedMS is how long the check itself took (not a cache read).
	ElapsedMS int64 `json:"elapsed_ms"`
	// Cached is true when this verdict came from the cache.
	Cached bool `json:"-"`
}

// Prober reports a file's duration; *services.FFprobeService satisfies it.
type Prober interface {
	Probe(ctx context.Context, filePath string) (*services.MediaTechInfo, error)
}

// Cache stores verdicts; *repository.CacheRepository satisfies it. nil
// disables caching.
type Cache interface {
	Get(ctx context.Context, key string) (*repository.CacheEntry, error)
	Set(ctx context.Context, key string, value string, cacheType string, ttl time.Duration) error
}

// decodeFunc returns one window of the stream as 16 kHz mono s16le PCM,
// band-passed to the voice range.
// centre picks only the front-centre channel, where films put the dialogue:
// on a 5.1 mix the score and effects sit in the other five.
type decodeFunc func(ctx context.Context, mediaPath string, audioStream int, w window, centre bool) ([]byte, error)

// Checker runs CheckSidecarAlignment.
type Checker struct {
	prober    Prober
	cache     Cache
	decode    decodeFunc
	hasCentre func(ctx context.Context, mediaPath string, audioStream int) bool
	// onCurves, when set, sees every check's per-window score curves (the
	// exam saves them to tune decide offline).
	onCurves func(curves [][]float64, starts []int)
	logger   *slog.Logger
}

// NewChecker wires a checker. cache may be nil.
func NewChecker(prober Prober, cache Cache, logger *slog.Logger) *Checker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Checker{prober: prober, cache: cache, decode: decodeWindow, hasCentre: hasCentreChannel, logger: logger}
}

// ErrNoFFmpeg means the check cannot run at all. A caller must treat it as
// "unknown", never as aligned.
var ErrNoFFmpeg = errors.New("sidecar check: ffmpeg not found")

// Check tells whether the subtitle at sidecarPath matches the timing of the
// audio stream audioStream (an absolute ffmpeg stream index, as
// services.AudioTrack.Index) of mediaPath. An error means the check could not
// be done (no ffmpeg, a file unreadable, the duration unknown) — "unknown",
// never "aligned".
func (c *Checker) Check(ctx context.Context, mediaPath string, audioStream int, sidecarPath string) (Verdict, error) {
	media, err := os.Stat(mediaPath)
	if err != nil {
		return Verdict{}, fmt.Errorf("sidecar check: media: %w", err)
	}
	side, err := os.Stat(sidecarPath)
	if err != nil {
		return Verdict{}, fmt.Errorf("sidecar check: subtitle: %w", err)
	}
	key := cacheKey(mediaPath, media, sidecarPath, side, audioStream)
	if v, ok := c.cached(ctx, key); ok {
		return v, nil
	}

	started := time.Now()
	v, err := c.check(ctx, mediaPath, audioStream, sidecarPath)
	if err != nil {
		return Verdict{}, err
	}
	v.ElapsedMS = time.Since(started).Milliseconds()
	c.store(ctx, key, v)
	c.logger.Info("sidecar subtitle alignment checked",
		"result", v.Result, "offset_seconds", v.OffsetSeconds, "reason", v.Reason,
		"score", v.Score, "windows", len(v.Windows), "elapsed_ms", v.ElapsedMS)
	return v, nil
}

func (c *Checker) check(ctx context.Context, mediaPath string, audioStream int, sidecarPath string) (Verdict, error) {
	loaded, err := mine.LoadCues(sidecarPath)
	if err != nil {
		return Verdict{Result: Unusable, Reason: "unreadable"}, nil
	}
	texts := make([]string, len(loaded))
	cues := make([]cue, len(loaded))
	for i, l := range loaded {
		texts[i] = l.Text
		cues[i] = cue{StartMS: l.StartMS, EndMS: l.EndMS}
	}
	if ok, why := englishVerdict(texts); !ok {
		return Verdict{Result: Unusable, Reason: why}, nil
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return Verdict{}, ErrNoFFmpeg
	}
	info, err := c.prober.Probe(ctx, mediaPath)
	if err != nil {
		return Verdict{}, fmt.Errorf("sidecar check: probe: %w", err)
	}
	if info == nil || info.DurationSeconds <= 0 {
		return Verdict{}, errors.New("sidecar check: duration unknown")
	}

	line := timeline(cues)
	windows := planWindows(int(info.DurationSeconds*1000), windowCount, int(windowLen.Milliseconds()), minWindows)
	curves := make([][]float64, 0, len(windows))
	starts := make([]int, 0, len(windows))
	centre := c.hasCentre(ctx, mediaPath, audioStream)
	for _, w := range windows {
		pcm, err := c.decode(ctx, mediaPath, audioStream, w, centre)
		if err != nil {
			return Verdict{}, fmt.Errorf("sidecar check: decode at %ds: %w", w.StartMS/1000, err)
		}
		curves = append(curves, shiftScores(frameEnergies(pcm), w.StartMS, line))
		starts = append(starts, w.StartMS)
	}

	if c.onCurves != nil {
		c.onCurves(curves, starts)
	}
	return decide(curves, starts), nil
}

// decodeWindow runs ffmpeg over one window. -ss before -i seeks without
// reading what comes before it.
func decodeWindow(ctx context.Context, mediaPath string, audioStream int, w window, centre bool) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, decodeTimeout)
	defer cancel()
	filter := "highpass=f=300,lowpass=f=3400"
	if centre {
		filter = "pan=mono|c0=FC," + filter
	}
	//nolint:gosec // mediaPath is a library file, the rest are numbers
	cmd := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-hide_banner", "-loglevel", "error",
		"-ss", strconv.FormatFloat(float64(w.StartMS)/1000, 'f', 3, 64),
		"-t", strconv.FormatFloat(float64(w.LenMS)/1000, 'f', 3, 64),
		"-i", mediaPath,
		"-map", fmt.Sprintf("0:%d", audioStream),
		"-af", filter,
		"-ac", "1", "-ar", strconv.Itoa(sampleRate),
		"-f", "s16le", "-")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}
	return out, nil
}

// centreLayouts are the ffmpeg channel layouts with a front-centre channel.
// pan=…FC on a layout without one yields silence, not an error, so the
// layout is asked first.
var centreLayouts = map[string]bool{
	"3.0": true, "3.0(back)": true, "3.1": true, "4.0": true, "4.1": true,
	"5.0": true, "5.0(side)": true, "5.1": true, "5.1(side)": true,
	"6.0": true, "6.1": true, "6.1(back)": true, "6.1(front)": true,
	"7.0": true, "7.0(front)": true, "7.1": true, "7.1(wide)": true, "7.1(wide-side)": true,
	"7.1(top)": true, "hexagonal": true, "octagonal": true,
}

// hasCentreChannel asks ffprobe for the stream's channel layout. Unknown
// means no: the whole mix is the safe default.
func hasCentreChannel(ctx context.Context, mediaPath string, audioStream int) bool {
	ctx, cancel := context.WithTimeout(ctx, decodeTimeout)
	defer cancel()
	//nolint:gosec // mediaPath is a library file
	out, err := exec.CommandContext(ctx, "ffprobe", "-v", "error",
		"-select_streams", strconv.Itoa(audioStream),
		"-show_entries", "stream=channel_layout", "-of", "csv=p=0", mediaPath).Output()
	if err != nil {
		return false
	}
	return centreLayouts[strings.TrimSpace(string(out))]
}

// cacheKey changes whenever either file, the stream or the algorithm does.
func cacheKey(mediaPath string, media os.FileInfo, sidecarPath string, side os.FileInfo, audioStream int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%d|%s|%d|%d|%d|%s|%s",
		mediaPath, media.Size(), media.ModTime().UnixNano(),
		sidecarPath, side.Size(), side.ModTime().UnixNano(),
		audioStream, algorithmVersion, tunables())))
	return CacheType + ":" + hex.EncodeToString(h[:16])
}

// tunables is every number that changes a verdict, so retuning one can never
// serve a verdict the old numbers reached (CR: a bump of algorithmVersion is
// easy to forget).
func tunables() string {
	return fmt.Sprintf("w%d×%d|g%.2f|p%.2f|h%.2f|e%d|a%d|al%d|c%d|l%.2f|f%.2f|d%.3f",
		windowCount, windowLen.Milliseconds(), minGlobalPeak, minWindowPeak, minHalfPeak,
		edgeMS, agreeMS, alignedMS, minCues, minLatinShare, minFunctionWordShare, maxDiacriticShare)
}

func (c *Checker) cached(ctx context.Context, key string) (Verdict, bool) {
	if c.cache == nil {
		return Verdict{}, false
	}
	entry, err := c.cache.Get(ctx, key)
	if err != nil || entry == nil {
		return Verdict{}, false
	}
	var v Verdict
	if err := json.Unmarshal([]byte(entry.Value), &v); err != nil {
		return Verdict{}, false
	}
	v.Cached = true
	return v, true
}

func (c *Checker) store(ctx context.Context, key string, v Verdict) {
	if c.cache == nil {
		return
	}
	data, err := json.Marshal(v)
	if err != nil {
		c.logger.Warn("sidecar check: could not encode the verdict", "error", err)
		return
	}
	if err := c.cache.Set(ctx, key, string(data), CacheType, cacheTTL); err != nil {
		c.logger.Warn("sidecar check: could not cache the verdict", "error", err)
	}
}
