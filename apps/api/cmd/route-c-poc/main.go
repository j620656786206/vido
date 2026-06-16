// Command route-c-poc is an END-TO-END feasibility spike for the subtitle
// "Route C" generation pipeline. It chains vido's REAL building blocks:
//
//	video file
//	  → [ffprobe/ffmpeg] AudioExtractorService  (extract audio track → 16kHz mono WAV)
//	  → [OpenAI]        ai.WhisperClient        (transcribe → SRT)
//	  → [parse]         subtitle.ParseSRT       (split text from timestamps)
//	  → [Claude]        services.TranslationService (+ official subtitle_translator prompt)
//	  → [in-process]    subtitle.Converter (OpenCC s2twp safety-net)
//	  → [write]         subtitle.Placer         (Movie.zh-Hant.srt, atomic, .bak backup)
//
// If this runs green on a real episode, the production Route C pipeline is feasible.
//
// Usage:
//
//	export OPENAI_API_KEY=...   # Whisper transcription
//	export CLAUDE_API_KEY=...   # LLM translation
//	go run ./cmd/route-c-poc -video "/path/to/Episode.S01E01.mkv" -clip 240 -maxblocks 40
//
// Flags:
//
//	-clip N      transcribe only the first N seconds (default 240; keeps it cheap/fast). 0 = full episode (uses chunking).
//	-maxblocks N translate only the first N subtitle blocks (default 40; 0 = all).
//	-out DIR     where to write the .srt (default: a temp dir; does NOT touch your media folder).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
)

func main() {
	video := flag.String("video", "", "path to the video file (required)")
	clip := flag.Int("clip", 240, "transcribe only the first N seconds (0 = full episode)")
	maxBlocks := flag.Int("maxblocks", 40, "translate only the first N subtitle blocks (0 = all)")
	outDir := flag.String("out", "", "output directory for the .srt (default: temp dir)")
	// NOTE: vido's hardcoded ai/claude.go DefaultClaudeModel "claude-3-5-haiku-latest"
	// now 404s (deprecated). Override with a current model id here.
	model := flag.String("model", "claude-haiku-4-5-20251001", "Claude model id for translation")
	lang := flag.String("lang", "", "Whisper source language ISO-639-1 (empty = derive from the audio track)")
	flag.Parse()

	if err := run(*video, *clip, *maxBlocks, *outDir, *model, *lang); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ POC FAILED: %v\n", err)
		os.Exit(1)
	}
}

func run(video string, clip, maxBlocks int, outDir, model, lang string) error {
	ctx := context.Background()

	// ── Stage 0: preconditions ───────────────────────────────────────────
	fmt.Println("══════════ Route C end-to-end POC ══════════")
	if video == "" {
		return fmt.Errorf("-video is required")
	}
	absVideo, err := filepath.Abs(video)
	if err != nil {
		return fmt.Errorf("resolve video path: %w", err)
	}
	if _, err := os.Stat(absVideo); err != nil {
		return fmt.Errorf("video not found: %s", absVideo)
	}
	openaiKey := os.Getenv("OPENAI_API_KEY")
	claudeKey := os.Getenv("CLAUDE_API_KEY")
	if openaiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY not set (needed for Whisper transcription)")
	}
	if claudeKey == "" {
		return fmt.Errorf("CLAUDE_API_KEY not set (needed for LLM translation)")
	}
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("%s not found in PATH — install ffmpeg (e.g. `brew install ffmpeg`)", bin)
		}
	}
	fmt.Printf("video    : %s\n", absVideo)
	fmt.Printf("clip     : %d sec   maxblocks: %d\n\n", clip, maxBlocks)

	// ── Stage 1a: list + select audio track (REAL AudioExtractorService) ─
	t0 := time.Now()
	extractor := services.NewAudioExtractorService(1, 5*time.Minute, nil)
	if !extractor.IsAvailable() {
		return fmt.Errorf("ffmpeg not available")
	}
	tracks, err := extractor.ListAudioTracks(ctx, absVideo)
	if err != nil {
		return fmt.Errorf("[stage1] list audio tracks: %w", err)
	}
	track, err := services.SelectEnglishTrack(tracks)
	if err != nil {
		return fmt.Errorf("[stage1] select track: %w", err)
	}
	fmt.Printf("➊ audio tracks: %d found → selected stream %d (lang=%s, codec=%s, ch=%d)\n",
		len(tracks), track.Index, track.Language, track.Codec, track.Channels)

	// Derive the Whisper language hint from the track when not overridden.
	if lang == "" {
		if l := track.Language; l != "" && l != "und" && len(l) >= 2 {
			lang = strings.ToLower(l[:2]) // "eng" → "en"
		}
	}

	// ── Stage 1b: extract audio → WAV (clip for cost, else real full) ────
	var wavPath string
	if clip > 0 {
		wavPath, err = extractClip(ctx, absVideo, track.Index, clip)
	} else {
		wavPath, err = extractor.ExtractAudio(ctx, absVideo, track.Index)
	}
	if err != nil {
		return fmt.Errorf("[stage1] extract audio: %w", err)
	}
	defer os.Remove(wavPath)
	if fi, err := os.Stat(wavPath); err == nil {
		fmt.Printf("  extracted WAV: %s (%.1f MB)  [%.1fs]\n\n", filepath.Base(wavPath), float64(fi.Size())/1e6, time.Since(t0).Seconds())
	}

	// ── Stage 2a: transcribe via Whisper (REAL ai.WhisperClient) ─────────
	t1 := time.Now()
	fmt.Printf("  (Whisper language pinned to: %q)\n", lang)
	whisper := ai.NewWhisperClient(openaiKey, ai.WithWhisperLanguage(lang), ai.WithWhisperTimeout(8*time.Minute))
	srt, err := transcribeWAV(ctx, whisper, wavPath)
	if err != nil {
		return fmt.Errorf("[stage2] whisper transcribe: %w", err)
	}
	blocks, err := subtitle.ParseSRT(srt)
	if err != nil {
		return fmt.Errorf("[stage2] parse SRT: %w", err)
	}
	fmt.Printf("➋ transcribed: %d subtitle blocks  [%.1fs]\n", len(blocks), time.Since(t1).Seconds())
	printBlocksPreview("   transcript", blocks, 4)
	if len(blocks) == 0 {
		return fmt.Errorf("[stage2] Whisper returned 0 blocks (no speech detected?)")
	}

	// cap blocks for cost
	transBlocks := blocks
	if maxBlocks > 0 && len(transBlocks) > maxBlocks {
		transBlocks = transBlocks[:maxBlocks]
		fmt.Printf("   (translating first %d of %d blocks)\n", maxBlocks, len(blocks))
	}

	// ── Stage 2b: translate via Claude (REAL services.TranslationService) ─
	t2 := time.Now()
	provider := ai.NewClaudeProvider(claudeKey, ai.WithClaudeModel(model))
	var completer ai.TextCompleter = provider
	fmt.Printf("   (translating with model: %s)\n", model)
	translator := services.NewTranslationService(completer, nil)
	if translator == nil || !translator.IsConfigured() {
		return fmt.Errorf("[stage2] translation service not configured")
	}
	in := make([]services.TranslationBlock, len(transBlocks))
	for i, b := range transBlocks {
		in[i] = services.TranslationBlock{Index: b.Index, Start: b.Start, End: b.End, Text: b.Text}
	}
	translated, err := translator.Translate(ctx, in, func(p float64) {
		fmt.Printf("\r   translating… %.0f%%", p)
	})
	if err != nil {
		return fmt.Errorf("[stage2] translate: %w", err)
	}
	fmt.Printf("\r➌ translated: %d blocks → 繁中  [%.1fs]\n", len(translated), time.Since(t2).Seconds())
	printSideBySide("   原文 → 繁中", transBlocks, translated, 6)

	// ── Stage 3a: OpenCC s2twp safety-net (REAL subtitle.Converter) ──────
	outBlocks := make([]subtitle.SubtitleBlock, len(translated))
	for i, b := range translated {
		outBlocks[i] = subtitle.SubtitleBlock{Index: b.Index, Start: b.Start, End: b.End, Text: b.Text}
	}
	finalSRT := subtitle.SerializeSRT(outBlocks)
	if conv, cerr := subtitle.NewConverter(); cerr == nil && conv.IsAvailable() {
		if converted, err := conv.ConvertS2TWP([]byte(finalSRT)); err == nil {
			finalSRT = string(converted)
			fmt.Printf("➍ OpenCC s2twp normalization applied (simplified→Taiwan-traditional safety pass)\n")
		}
	}

	// ── Stage 3b: write file (REAL subtitle.Placer) ─────────────────────
	if outDir == "" {
		outDir = filepath.Join(os.TempDir(), "route-c-poc")
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("[stage3] mkdir out: %w", err)
	}
	// Point MediaFilePath at the out dir (NOT the user's media folder) so the
	// placer writes there; Placer only needs the dir to exist, not the media file.
	placeMedia := filepath.Join(outDir, filepath.Base(absVideo))
	placer := subtitle.NewPlacer(subtitle.DefaultPlacerConfig())
	res, err := placer.Place(subtitle.PlaceRequest{
		MediaFilePath: placeMedia,
		SubtitleData:  []byte(finalSRT),
		Language:      "zh-Hant",
		Format:        "srt",
	})
	if err != nil {
		return fmt.Errorf("[stage3] place subtitle: %w", err)
	}

	fmt.Printf("\n✅ POC PASSED — full pipeline ran end-to-end.\n")
	fmt.Printf("   output: %s\n", res.SubtitlePath)
	return nil
}

// extractClip mirrors AudioExtractorService.ExtractAudio (audio_extractor_service.go:150)
// but limits to the first `seconds` to keep the POC cheap. Same 16kHz mono PCM WAV.
func extractClip(ctx context.Context, video string, trackIndex, seconds int) (string, error) {
	tmp, err := os.CreateTemp("", "route-c-poc-*.wav")
	if err != nil {
		return "", err
	}
	out := tmp.Name()
	tmp.Close()
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", video,
		"-t", strconv.Itoa(seconds),
		"-map", fmt.Sprintf("0:%d", trackIndex),
		"-vn", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1",
		"-y", out,
	)
	if o, err := cmd.CombinedOutput(); err != nil {
		os.Remove(out)
		return "", fmt.Errorf("ffmpeg clip: %w\n%s", err, string(o))
	}
	return out, nil
}

// transcribeWAV transcribes a WAV. If it exceeds the Whisper 25MB limit it
// segments reliably with ffmpeg's segment muxer and offsets each piece's
// timestamps.
//
// NOTE: we deliberately do NOT use vido's ai.SplitAudioChunks here — it relies
// on getWAVDuration() which mis-parses ffmpeg's WAV header (assumes data size at
// fixed bytes 40-43), returns a wrong/small duration, decides "no split needed",
// and hands the whole oversized file to Whisper → HTTP 413. That is a real
// production bug (NeedsChunking is size-based; SplitAudioChunks is duration-based
// — they disagree). Backlog fix required.
const pocSegmentSeconds = 300 // ~9.6MB per 16kHz-mono segment — lighter requests, less timeout risk

func transcribeWAV(ctx context.Context, w *ai.WhisperClient, wav string) (string, error) {
	needs, _ := ai.NeedsChunking(wav)
	if !needs {
		return w.Transcribe(ctx, wav)
	}

	segDir, err := os.MkdirTemp("", "rc-seg")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(segDir)

	seg := exec.CommandContext(ctx, "ffmpeg", "-i", wav,
		"-f", "segment", "-segment_time", strconv.Itoa(pocSegmentSeconds),
		"-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", "-y",
		filepath.Join(segDir, "seg-%03d.wav"),
	)
	if out, err := seg.CombinedOutput(); err != nil {
		return "", fmt.Errorf("segment wav: %w\n%s", err, string(out))
	}
	segs, _ := filepath.Glob(filepath.Join(segDir, "seg-*.wav"))
	sort.Strings(segs)
	fmt.Printf("  (chunked into %d segments of %ds)\n", len(segs), pocSegmentSeconds)

	var all []subtitle.SubtitleBlock
	for i, s := range segs {
		// vido's WhisperClient has NO retry — a single transient timeout kills the
		// whole transcription. Retry here (backlog: add retry/backoff to ai/ layer).
		var srt string
		var terr error
		for attempt := 1; attempt <= 3; attempt++ {
			srt, terr = w.Transcribe(ctx, s)
			if terr == nil {
				break
			}
			fmt.Printf("\n  segment %d attempt %d failed (%v) — retrying\n", i, attempt, terr)
		}
		if terr != nil {
			return "", fmt.Errorf("segment %d after 3 attempts: %w", i, terr)
		}
		blks, _ := subtitle.ParseSRT(srt)
		off := time.Duration(i*pocSegmentSeconds) * time.Second
		for _, b := range blks {
			b.Start = shiftTS(b.Start, off)
			b.End = shiftTS(b.End, off)
			b.Index = len(all) + 1
			all = append(all, b)
		}
		fmt.Printf("\r  transcribed segment %d/%d", i+1, len(segs))
	}
	fmt.Println()
	return subtitle.SerializeSRT(all), nil
}

// shiftTS adds an offset to an SRT timestamp "HH:MM:SS,mmm".
func shiftTS(ts string, off time.Duration) string {
	var h, m, s, ms int
	fmt.Sscanf(strings.ReplaceAll(ts, ",", ":"), "%d:%d:%d:%d", &h, &m, &s, &ms)
	d := time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s)*time.Second + time.Duration(ms)*time.Millisecond + off
	h = int(d / time.Hour)
	d -= time.Duration(h) * time.Hour
	m = int(d / time.Minute)
	d -= time.Duration(m) * time.Minute
	s = int(d / time.Second)
	d -= time.Duration(s) * time.Second
	ms = int(d / time.Millisecond)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

func printBlocksPreview(label string, blocks []subtitle.SubtitleBlock, n int) {
	fmt.Printf("%s (first %d):\n", label, n)
	for i, b := range blocks {
		if i >= n {
			break
		}
		fmt.Printf("     [%d] %s  %s\n", b.Index, b.Start, oneLine(b.Text))
	}
}

func printSideBySide(label string, orig []subtitle.SubtitleBlock, zh []services.TranslationBlock, n int) {
	fmt.Printf("%s (first %d):\n", label, n)
	for i := 0; i < n && i < len(orig) && i < len(zh); i++ {
		fmt.Printf("     [%d] %s\n          ↳ %s\n", orig[i].Index, oneLine(orig[i].Text), oneLine(zh[i].Text))
	}
}

func oneLine(s string) string { return strings.ReplaceAll(strings.TrimSpace(s), "\n", " / ") }
