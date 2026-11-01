package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg" // ffmpeg hands frames over as JPEG
	"image/png"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Frame sampling for the burned-in check (disc-2026-11-burned-in-subs-b1) —
// exactly what passed the exam in disc-2026-11-burned-in-subs-a
// (eval/burned-in-exam-2026-11/extract_frames.sh): six frames at 15–90% of the
// runtime, 480 px wide, the bottom 32% of each, stacked top to bottom.
var burnedInSamplePercents = []int{15, 30, 45, 60, 75, 90}

const (
	burnedInFrameWidth = 480
	// burnedInStripShare is the bottom share of a frame kept: where open
	// captions sit, without the rest of the picture to pay tokens for.
	burnedInStripShare = 0.32
	// burnedInMinFrames: fewer frames than this and the season is not judged
	// (a broken file, or a 4K Dolby Vision decode error like 《金色》's).
	burnedInMinFrames = 3
)

// ErrTooFewFrames means fewer than burnedInMinFrames frames could be read.
var ErrTooFewFrames = errors.New("too few frames to judge burned-in subtitles")

// BurnedInFrameGrabber reads frames out of a media file.
type BurnedInFrameGrabber interface {
	// Duration is the container duration in seconds.
	Duration(ctx context.Context, path string) (float64, error)
	// Frame is one encoded image (any format image.Decode reads) of the
	// frame at atSeconds, burnedInFrameWidth wide.
	Frame(ctx context.Context, path string, atSeconds float64) ([]byte, error)
}

// FFmpegFrameGrabber is the ffprobe/ffmpeg BurnedInFrameGrabber.
type FFmpegFrameGrabber struct {
	// Timeout bounds each ffprobe/ffmpeg call; zero means one minute (a seek
	// into a large file on a NAS disk can take a while).
	Timeout time.Duration
}

func (g FFmpegFrameGrabber) timeout() time.Duration {
	if g.Timeout > 0 {
		return g.Timeout
	}
	return time.Minute
}

// Duration implements BurnedInFrameGrabber.
func (g FFmpegFrameGrabber) Duration(ctx context.Context, path string) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, g.timeout())
	defer cancel()
	//nolint:gosec // path comes from a trusted DB record
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error",
		"-show_entries", "format=duration", "-of", "csv=p=0", path)
	cmd.WaitDelay = 10 * time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	seconds, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil || seconds <= 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return 0, fmt.Errorf("ffprobe duration: unreadable %q", strings.TrimSpace(string(out)))
	}
	return seconds, nil
}

// Frame implements BurnedInFrameGrabber: one JPEG on stdout. -nostdin keeps
// ffmpeg from reading the server's stdin.
func (g FFmpegFrameGrabber) Frame(ctx context.Context, path string, atSeconds float64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, g.timeout())
	defer cancel()
	//nolint:gosec // path comes from a trusted DB record
	cmd := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-v", "error",
		"-ss", strconv.FormatFloat(atSeconds, 'f', 3, 64), "-i", path,
		"-frames:v", "1", "-vf", fmt.Sprintf("scale=%d:-2", burnedInFrameWidth), "-q:v", "4",
		"-f", "image2pipe", "-c:v", "mjpeg", "pipe:1")
	cmd.WaitDelay = 10 * time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg frame at %.0fs: %w: %s", atSeconds, err, strings.TrimSpace(stderr.String()))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("ffmpeg frame at %.0fs: no image (%s)", atSeconds, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// stackFrameStrips decodes the frames, keeps the bottom burnedInStripShare of
// each and stacks them top to bottom into one PNG — pure Go, no cgo. Frames
// that do not decode are skipped; fewer than burnedInMinFrames left is
// ErrTooFewFrames.
func stackFrameStrips(frames [][]byte) ([]byte, error) {
	var strips []image.Image
	width, height := 0, 0
	for _, f := range frames {
		img, _, err := image.Decode(bytes.NewReader(f))
		if err != nil {
			continue
		}
		b := img.Bounds()
		stripH := int(math.Round(float64(b.Dy()) * burnedInStripShare))
		if b.Dx() == 0 || stripH == 0 {
			continue
		}
		strip := image.Rect(b.Min.X, b.Max.Y-stripH, b.Max.X, b.Max.Y)
		strips = append(strips, subImage(img, strip))
		width = max(width, b.Dx())
		height += stripH
	}
	if len(strips) < burnedInMinFrames {
		return nil, fmt.Errorf("%w: %d of %d", ErrTooFewFrames, len(strips), len(frames))
	}

	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), image.Black, image.Point{}, draw.Src)
	y := 0
	for _, s := range strips {
		b := s.Bounds()
		draw.Draw(canvas, image.Rect(0, y, b.Dx(), y+b.Dy()), s, b.Min, draw.Src)
		y += b.Dy()
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return nil, fmt.Errorf("encode stacked frames: %w", err)
	}
	return buf.Bytes(), nil
}

// subImage crops img to r, copying only when the decoder's type cannot crop.
func subImage(img image.Image, r image.Rectangle) image.Image {
	if s, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	}); ok {
		return s.SubImage(r)
	}
	dst := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(dst, dst.Bounds(), img, r.Min, draw.Src)
	return dst
}
