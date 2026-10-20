package sidecarcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
	"github.com/vido/api/internal/subtitle/mine"
)

// TestAlignmentExam is the AC #5 exam: every film gets its original English
// subtitle, the same shifted +3 s and -2 s, the same stretched as if made for
// 25 fps and for 24 fps from 23.976 (the slow one drifts ~7 s over two
// hours), and the NEXT film's subtitle. It needs real films, so
// it only runs when VIDO_ALIGN_EXAM names a JSON list of
//
//	{"label": "A", "media": "/path.mkv", "stream": 1, "subtitle": "/path.en.srt"}
//
// or, instead of "subtitle", "subtitle_stream": N to pull an embedded text
// track with ffmpeg. It prints one line per cell and fails on any wrong cell.
// Labels only — the exam report in the repo never names a library file.
func TestAlignmentExam(t *testing.T) {
	cfgPath := os.Getenv("VIDO_ALIGN_EXAM")
	if cfgPath == "" {
		t.Skip("VIDO_ALIGN_EXAM not set — the exam needs real films")
	}
	raw, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	var films []struct {
		Label          string `json:"label"`
		Media          string `json:"media"`
		Stream         int    `json:"stream"`
		Subtitle       string `json:"subtitle"`
		SubtitleStream *int   `json:"subtitle_stream"`
	}
	require.NoError(t, json.Unmarshal(raw, &films))
	if n, err := strconv.Atoi(os.Getenv("VIDO_ALIGN_WINDOWS")); err == nil && n > 0 {
		windowCount = n
	}
	fmt.Printf("EXAM windows=%d\n", windowCount)

	dir := t.TempDir()
	originals := make([]string, len(films))
	for i, f := range films {
		originals[i] = f.Subtitle
		if f.SubtitleStream != nil {
			// Kept beside the config: pulling a track reads the whole file
			// (ten minutes for a 99 GB remux), so a rerun reuses it.
			out := filepath.Join(filepath.Dir(cfgPath), fmt.Sprintf("%s-orig.srt", f.Label))
			originals[i] = out
			if _, err := os.Stat(out); err == nil {
				continue
			}
			cmd := exec.Command("ffmpeg", "-nostdin", "-loglevel", "error", "-y", "-i", f.Media,
				"-map", fmt.Sprintf("0:%d", *f.SubtitleStream), "-c:s", "srt", out)
			b, err := cmd.CombinedOutput()
			require.NoError(t, err, string(b))
		}
	}

	checker := NewChecker(services.NewFFprobeService(1, time.Minute, nil), nil, nil)
	type dump struct {
		Label, Variant string
		Want           Result
		Offset         float64
		Starts         []int
		Curves         [][]float64
	}
	var dumps []dump
	var last dump
	checker.onCurves = func(curves [][]float64, starts []int) { last.Curves, last.Starts = curves, starts }
	if out := os.Getenv("VIDO_ALIGN_DUMP"); out != "" {
		defer func() {
			b, _ := json.Marshal(dumps)
			_ = os.WriteFile(out, b, 0o644)
		}()
	}
	wrong := 0
	for i, f := range films {
		cues, err := mine.LoadCues(originals[i])
		require.NoError(t, err)
		info, err := services.NewFFprobeService(1, time.Minute, nil).Probe(context.Background(), f.Media)
		require.NoError(t, err)
		// 24 fps against 23.976 is a 0.1% stretch. Over a 12-minute short it
		// moves the subtitle 0.7 s end to end — translating it is harmless —
		// so "aligned" is also right there. Over a feature it is seconds.
		slow := Mismatch
		if info.DurationSeconds*(24/23.976-1) <= float64(agreeMS)/1000 {
			slow = "aligned or mismatch"
		}
		variants := []struct {
			name   string
			path   string
			want   Result
			offset float64
		}{
			{"original", writeCues(t, dir, f.Label+"-orig2.srt", cues, 0, 1), Aligned, 0},
			{"+3s", writeCues(t, dir, f.Label+"-plus3.srt", cues, 3000, 1), Offset, 3},
			{"-2s", writeCues(t, dir, f.Label+"-minus2.srt", cues, -2000, 1), Offset, -2},
			{"fps", writeCues(t, dir, f.Label+"-fps.srt", cues, 0, 25/23.976), Mismatch, 0},
			{"fps 24", writeCues(t, dir, f.Label+"-fps24.srt", cues, 0, 24/23.976), slow, 0},
		}
		if len(films) > 1 {
			other := originals[(i+1)%len(films)]
			variants = append(variants, struct {
				name   string
				path   string
				want   Result
				offset float64
			}{"other film", other, Mismatch, 0})
		}
		for _, v := range variants {
			last = dump{Label: f.Label, Variant: v.name, Want: v.want, Offset: v.offset}
			got, err := checker.Check(context.Background(), f.Media, f.Stream, v.path)
			require.NoError(t, err)
			dumps = append(dumps, last)
			ok := got.Result == v.want ||
				(v.want == "aligned or mismatch" && (got.Result == Aligned || got.Result == Mismatch))
			if ok && v.want == Offset && abs(int((got.OffsetSeconds-v.offset)*1000)) > 300 {
				ok = false
			}
			mark := "ok"
			if !ok {
				mark = "WRONG"
				wrong++
			}
			var offs []string
			for _, w := range got.Windows {
				offs = append(offs, fmt.Sprintf("%+.1f(%.1f%s)", w.OffsetSeconds, w.Peak, map[bool]string{true: "*", false: ""}[w.Found]))
			}
			fmt.Printf("EXAM %s | %-10s | want %-8s got %-8s %+.1fs score %.2f %s | %dms | %s | %s\n",
				f.Label, v.name, v.want, got.Result, got.OffsetSeconds, got.Score, got.Reason, got.ElapsedMS, mark, strings.Join(offs, " "))
		}
	}
	require.Zero(t, wrong, "every exam cell must be right")
}

// writeCues writes cues shifted by shiftMS and stretched by scale.
func writeCues(t *testing.T, dir, name string, cues []mine.Cue, shiftMS int, scale float64) string {
	t.Helper()
	blocks := make([]subtitle.SubtitleBlock, 0, len(cues))
	for i, c := range cues {
		s := int(float64(c.StartMS)*scale) + shiftMS
		e := int(float64(c.EndMS)*scale) + shiftMS
		if s < 0 {
			continue
		}
		blocks = append(blocks, subtitle.SubtitleBlock{Index: i + 1, Start: srtTime(s), End: srtTime(e), Text: c.Text})
	}
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(subtitle.SerializeSRT(blocks)), 0o644))
	return path
}

func srtTime(ms int) string {
	return fmt.Sprintf("%02d:%02d:%02d,%03d", ms/3_600_000, ms/60_000%60, ms/1000%60, ms%1000)
}
