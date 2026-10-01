// Command mine runs 加速器② (story sub-7-5a) over a show's folder and prints
// the proper-noun renderings it can learn from the official zh-Hant subtitles
// already there — the acceptance run for AC #6 (Scorpion S01 on the NAS), and
// the dry run an operator can do before sub-7-5b wires this into the scanner.
//
//	go run ./cmd/mine --dir "/media/TV/Scorpion/Season 01"
//	go run ./cmd/mine --dir … --known known.json      # {"Walter O'Brien":"華特·歐布萊恩", …}
//	go run ./cmd/mine --dir … --json > terms.json
//
// Needs ffprobe (to see embedded tracks) and ffmpeg (to pull them out); runs
// inside the Vido container on the NAS. Sidecar-only shows need neither.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
	"github.com/vido/api/internal/subtitle/mine"
)

var videoExts = map[string]bool{".mkv": true, ".mp4": true, ".avi": true, ".mov": true, ".m4v": true, ".ts": true, ".wmv": true}

type episodeReport struct {
	Media    string `json:"media"`
	ZhSource string `json:"zh_source"`
	EnSource string `json:"en_source"`
	EnCues   int    `json:"en_cues"`
	ZhCues   int    `json:"zh_cues"`
	Segments int    `json:"segments"`
	Skipped  string `json:"skipped,omitempty"`
}

type report struct {
	Dir      string          `json:"dir"`
	Episodes []episodeReport `json:"episodes"`
	Usable   int             `json:"usable_episodes"`
	Terms    []mine.Term     `json:"terms"`
	segments []mine.Segment
}

func main() {
	dir := flag.String("dir", "", "folder with the show's episodes (scanned non-recursively)")
	knownPath := flag.String("known", "", "JSON object of already-trusted renderings {\"English\": \"中文\"}")
	asJSON := flag.Bool("json", false, "print the full report as JSON instead of a table")
	minSeg := flag.Int("min-segments", 0, "override Options.MinSegments (default 3)")
	timeout := flag.Duration("extract-timeout", 10*time.Minute, "ffmpeg timeout per episode")
	dump := flag.String("dump-segments", "", "also write every aligned segment to this JSON file (for tuning the miner offline)")
	flag.Parse()
	if *dir == "" {
		fmt.Fprintln(os.Stderr, "--dir is required")
		os.Exit(2)
	}
	known := map[string]string{}
	if *knownPath != "" {
		b, err := os.ReadFile(*knownPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "known:", err)
			os.Exit(2)
		}
		if err := json.Unmarshal(b, &known); err != nil {
			fmt.Fprintln(os.Stderr, "known:", err)
			os.Exit(2)
		}
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	rep, err := run(context.Background(), *dir, known, mine.Options{MinSegments: *minSeg}, *timeout, logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mine:", err)
		os.Exit(1)
	}
	if *dump != "" {
		b, _ := json.MarshalIndent(rep.segments, "", " ")
		if err := os.WriteFile(*dump, b, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "dump:", err)
			os.Exit(1)
		}
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rep)
		return
	}
	printTable(rep)
}

func run(ctx context.Context, dir string, known map[string]string, opts mine.Options, timeout time.Duration, logger *slog.Logger) (*report, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	prober := services.NewFFprobeService(1, 60*time.Second, logger)
	extractor := subtitle.NewExtractor(timeout, logger)
	tmp, err := os.MkdirTemp("", "vido-mine")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	rep := &report{Dir: dir}
	var all []mine.Segment
	for _, name := range names {
		if !videoExts[strings.ToLower(filepath.Ext(name))] {
			continue
		}
		media := filepath.Join(dir, name)
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		var sidecars []string
		for _, n := range names {
			if n != name && strings.HasPrefix(n, stem+".") {
				sidecars = append(sidecars, n)
			}
		}
		var tracks []services.SubtitleTrack
		if prober.IsAvailable() {
			if info, perr := prober.Probe(ctx, media); perr == nil && info != nil {
				tracks = info.SubtitleTracks
			}
		}
		src := mine.Classify(media, sidecars, tracks)
		er := episodeReport{Media: name}
		if !src.Usable() {
			switch {
			case src.HasZh():
				er.Skipped = "has zh, no English text track"
			default:
				er.Skipped = "no official zh subtitle"
			}
			rep.Episodes = append(rep.Episodes, er)
			continue
		}
		zhCues, zhFrom, err := loadSide(ctx, extractor, media, tmp, dir, src.ZhSidecars, src.ZhStreams)
		if err != nil {
			er.Skipped = "zh: " + err.Error()
			rep.Episodes = append(rep.Episodes, er)
			continue
		}
		enCues, enFrom, err := loadSide(ctx, extractor, media, tmp, dir, src.EnSidecars, src.EnStreams)
		if err != nil {
			er.Skipped = "en: " + err.Error()
			rep.Episodes = append(rep.Episodes, er)
			continue
		}
		segs := mine.Align(enCues, zhCues)
		er.ZhSource, er.EnSource, er.EnCues, er.ZhCues, er.Segments = zhFrom, enFrom, len(enCues), len(zhCues), len(segs)
		rep.Episodes = append(rep.Episodes, er)
		rep.Usable++
		all = append(all, segs...)
	}
	opts.Known = known
	rep.Terms = mine.Mine(all, opts)
	rep.segments = all
	return rep, nil
}

// loadSide reads one side's cues: the first parseable sidecar wins, otherwise
// the first embedded stream is extracted.
func loadSide(ctx context.Context, extractor *subtitle.Extractor, media, tmp, dir string, sidecars []string, streams []int) ([]mine.Cue, string, error) {
	for _, sc := range sidecars {
		cues, err := mine.LoadCues(filepath.Join(dir, sc))
		if err == nil && len(cues) > 0 {
			return cues, sc, nil
		}
	}
	if len(streams) == 0 {
		return nil, "", errors.New("no readable sidecar and no embedded track")
	}
	if !extractor.IsAvailable() {
		return nil, "", errors.New("embedded track needs ffmpeg, which is not available here")
	}
	out, err := extractor.Extract(ctx, media, tmp, streams[:1])
	if err != nil {
		return nil, "", err
	}
	path, ok := out[streams[0]]
	if !ok {
		return nil, "", fmt.Errorf("stream %d not extracted", streams[0])
	}
	cues, err := mine.LoadCues(path)
	if err != nil {
		return nil, "", err
	}
	return cues, fmt.Sprintf("embedded stream %d", streams[0]), nil
}

func printTable(rep *report) {
	fmt.Printf("%s\n", rep.Dir)
	for _, e := range rep.Episodes {
		if e.Skipped != "" {
			fmt.Printf("  – %-50s %s\n", trunc(e.Media, 50), e.Skipped)
			continue
		}
		fmt.Printf("  ✓ %-50s en=%d zh=%d segments=%d  (%s / %s)\n", trunc(e.Media, 50), e.EnCues, e.ZhCues, e.Segments, e.EnSource, e.ZhSource)
	}
	fmt.Printf("\n%d usable episode(s), %d term(s):\n", rep.Usable, len(rep.Terms))
	for _, t := range rep.Terms {
		fmt.Printf("  %-28s → %-16s support %d/%d  %s\n", t.Src, t.Zh, t.Support, t.Segments, t.How)
	}
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
