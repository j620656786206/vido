package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/subtitle/mine"
)

func srt(lines ...[3]string) string {
	out := ""
	for i, l := range lines {
		out += fmt.Sprintf("%d\n%s --> %s\n%s\n\n", i+1, l[0], l[1], l[2])
	}
	return out
}

// A sidecar-only show: no ffprobe/ffmpeg involved, so this runs anywhere.
func TestRun_SidecarShowLearnsNames(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644))
	}
	for ep := 1; ep <= 3; ep++ {
		base := fmt.Sprintf("Scorpion.S01E%02d", ep)
		write(base+".mkv", "") // an empty "video": ffprobe (if present) fails on it and that is fine
		// Real subtitles never repeat a line verbatim across episodes; vary
		// the context so the miner has something to tell a name from.
		enLines := [][2]string{
			{"Hey, Walter, we need you.", "嘿，華特，我們需要你"},
			{"Is Walter coming or not?", "華特到底來不來"},
			{"Ask Walter, he knows.", "去問華特，他知道"},
		}
		write(base+".en.srt", srt(
			[3]string{"00:00:01,000", "00:00:03,000", enLines[ep-1][0]},
			[3]string{"00:00:04,000", "00:00:06,000", "Tell Toby to hurry."},
			[3]string{"00:00:07,000", "00:00:09,000", "Fine."},
		))
		write(base+".zh-TW.srt", srt(
			[3]string{"00:00:01,100", "00:00:03,100", enLines[ep-1][1]},
			[3]string{"00:00:04,000", "00:00:06,200", fmt.Sprintf("叫托比快一點（%d）", ep)},
			[3]string{"00:00:07,000", "00:00:09,000", "好"},
		))
	}
	// Vido's own delivery sits beside E03 and must be ignored as a zh source.
	write("Scorpion.S01E03.zh-Hant.srt", srt([3]string{"00:00:01,000", "00:00:03,000", "機器翻的"}))
	// E04 has no official zh → skipped, reported.
	write("Scorpion.S01E04.mkv", "")
	write("Scorpion.S01E04.en.srt", srt([3]string{"00:00:01,000", "00:00:03,000", "Walter again."}))

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rep, err := run(context.Background(), dir, map[string]string{"Toby": "托比"}, mine.Options{}, time.Minute, logger)
	require.NoError(t, err)
	assert.Equal(t, 3, rep.Usable)
	require.Len(t, rep.Episodes, 4)
	assert.Equal(t, "no official zh subtitle", rep.Episodes[3].Skipped)
	assert.Equal(t, "Scorpion.S01E03.zh-TW.srt", rep.Episodes[2].ZhSource, "the .zh-Hant.srt delivery is never picked")
	assert.Equal(t, 3, rep.Episodes[0].Segments)

	byName := map[string]mine.Term{}
	for _, tm := range rep.Terms {
		byName[tm.Src] = tm
	}
	assert.Equal(t, "華特", byName["Walter"].Zh)
	assert.Equal(t, "cooccurrence", byName["Walter"].How)
	assert.Equal(t, 3, byName["Walter"].Support)
	assert.Equal(t, "known", byName["Toby"].How, "a known rendering is verified, not re-learned")
	assert.Equal(t, 3, byName["Toby"].Support)
}
