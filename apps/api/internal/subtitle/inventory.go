package subtitle

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vido/api/internal/services"
)

// Subtitle inventory (bugfix-subtitle-dialog-real-inventory): what subtitles a
// title ACTUALLY has, read on demand for ONE media file. Two halves of very
// different cost, kept separate on purpose:
//
//   - ListSidecars: one directory read plus the head of each subtitle file —
//     cheap enough to summarise a whole season from one folder read
//     (disc-2026-10-episode-list-subtitle-badge builds on it).
//   - the embedded half: one ffprobe of the media file — never run per episode
//     on a season expand (red line 3).

// LangChineseUnknown is Chinese whose script cannot be told — a mixed file, or
// an embedded `chi` track with no telling title. Deliberately NOT "zh": the
// web client reads "zh" as Traditional.
const LangChineseUnknown = "zh-unknown"

// Inventory section statuses.
const (
	InventoryOK          = "ok"
	InventoryUnavailable = "unavailable" // ffprobe is not installed
	InventoryFailed      = "failed"      // the read was attempted and failed
)

// SidecarFile is one subtitle file beside the media file.
type SidecarFile struct {
	FileName string `json:"file_name"`
	// Language: zh-Hant / zh-Hans / zh-unknown decided by CONTENT; otherwise the
	// lowercased filename tag (e.g. "en"), or "und".
	Language string `json:"language"`
	Format   string `json:"format"`
	// IsVidoOutput marks the subtitle Vido itself placed (the row's
	// subtitle_path), so the client shows it once, as the engine's row.
	IsVidoOutput bool `json:"is_vido_output"`
}

// EmbeddedTrack is one subtitle stream inside the media file.
type EmbeddedTrack struct {
	StreamIndex int    `json:"stream_index"`
	Language    string `json:"language"`
	Title       string `json:"title,omitempty"`
	Format      string `json:"format"`
	// Text is false for image-based tracks (PGS, VobSub).
	Text bool `json:"text"`
}

// SidecarSection / EmbeddedSection carry their own status so one half failing
// never hides the other.
type SidecarSection struct {
	Status string        `json:"status"`
	Files  []SidecarFile `json:"files"`
}

type EmbeddedSection struct {
	Status string          `json:"status"`
	Tracks []EmbeddedTrack `json:"tracks"`
}

// Inventory is the full answer for one media file.
type Inventory struct {
	Sidecars SidecarSection  `json:"sidecars"`
	Embedded EmbeddedSection `json:"embedded"`
}

// TrackProber is the expensive half: *services.FFprobeService.
type TrackProber interface {
	IsAvailable() bool
	Probe(ctx context.Context, filePath string) (*services.MediaTechInfo, error)
}

// sidecarFormats are the text subtitle files listed (image .sub/.idx is not).
var sidecarFormats = map[string]string{".srt": "srt", ".ass": "ass", ".ssa": "ssa", ".vtt": "vtt"}

// BuildInventory reads both halves for one media file. ownSubtitlePath is the
// subtitle Vido placed for this row ("" when none). prober may be nil.
func BuildInventory(ctx context.Context, mediaPath, ownSubtitlePath string, prober TrackProber) Inventory {
	inv := Inventory{
		Sidecars: SidecarSection{Status: InventoryOK, Files: []SidecarFile{}},
		Embedded: EmbeddedSection{Status: InventoryOK, Tracks: []EmbeddedTrack{}},
	}

	if files, err := ListSidecars(mediaPath, ownSubtitlePath); err != nil {
		inv.Sidecars.Status = InventoryFailed
	} else {
		inv.Sidecars.Files = append(inv.Sidecars.Files, files...)
	}

	switch {
	case prober == nil || !prober.IsAvailable():
		inv.Embedded.Status = InventoryUnavailable
	default:
		info, err := prober.Probe(ctx, mediaPath)
		if err != nil || info == nil {
			inv.Embedded.Status = InventoryFailed
			break
		}
		for _, t := range info.SubtitleTracks {
			if t.External {
				continue
			}
			inv.Embedded.Tracks = append(inv.Embedded.Tracks, EmbeddedTrack{
				StreamIndex: t.StreamIndex,
				Language:    embeddedLanguage(t.Language, t.Title),
				Title:       t.Title,
				Format:      t.Format,
				Text:        IsTextSubtitleCodec(t.Format),
			})
		}
	}
	return inv
}

// ListSidecars lists the subtitle files beside mediaPath (same file stem,
// text formats, no .bak / .tmp. artifacts) with ONE directory read. Chinese is
// decided by content — a ".zh-TW.srt" full of Simplified is Simplified.
func ListSidecars(mediaPath, ownSubtitlePath string) ([]SidecarFile, error) {
	dir := filepath.Dir(mediaPath)
	stem := strings.TrimSuffix(filepath.Base(mediaPath), filepath.Ext(mediaPath))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	own := ""
	if ownSubtitlePath != "" {
		own = filepath.Clean(ownSubtitlePath)
	}

	files := []SidecarFile{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lower := strings.ToLower(name)
		format, ok := sidecarFormats[filepath.Ext(lower)]
		if !ok || strings.Contains(lower, ".tmp.") {
			continue
		}
		nameNoExt := strings.TrimSuffix(name, filepath.Ext(name))
		if nameNoExt != stem && !strings.HasPrefix(nameNoExt, stem+".") {
			continue
		}
		path := filepath.Join(dir, name)
		files = append(files, SidecarFile{
			FileName:     name,
			Language:     sidecarLanguage(path, strings.TrimPrefix(strings.TrimPrefix(nameNoExt, stem), ".")),
			Format:       format,
			IsVidoOutput: own != "" && filepath.Clean(path) == own,
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].FileName < files[j].FileName })
	return files, nil
}

// sidecarLanguage decides a sidecar's language: Chinese from the text, else the
// first filename tag segment ("en" of "en.sdh"), never a Chinese tag the text
// does not back up.
func sidecarLanguage(path, tag string) string {
	if head, err := readHead(path); err == nil {
		switch Detect(head).Language {
		case LangTraditional:
			return LangTraditional
		case LangSimplified:
			return LangSimplified
		case LangAmbiguous:
			return LangChineseUnknown
		}
	}
	first := strings.ToLower(strings.SplitN(tag, ".", 2)[0])
	switch {
	case first == "" || isChineseTag(first):
		return LangUndetermined
	case isEnglishTag(first):
		return "en"
	default:
		return first
	}
}

func readHead(path string) ([]byte, error) {
	f, err := os.Open(path) //nolint:gosec // a sidecar beside a library file
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, maxDetectionBytes))
}

// embeddedLanguage normalises an embedded track's language. ffprobe only says
// "chi"; the title tag usually tells the script ("繁體", "Chinese
// (Simplified)", "Cantonese"). Untold Chinese stays zh-unknown.
func embeddedLanguage(lang, title string) string {
	l := strings.ToLower(strings.TrimSpace(lang))
	t := strings.ToLower(title)
	has := func(words ...string) bool {
		for _, w := range words {
			if strings.Contains(t, strings.ToLower(w)) {
				return true
			}
		}
		return false
	}
	switch {
	case l == "yue" || ((isChineseTag(l) || l == "") && has("粵", "粤", "cantonese", "yue")):
		return "yue"
	case l == "zh-hant" || l == "zh-tw" || l == "zh-hk" || l == "cht":
		return LangTraditional
	case l == "zh-hans" || l == "zh-cn" || l == "chs":
		return LangSimplified
	case isChineseTag(l):
		switch {
		case has("繁", "traditional", "hant", "zh-tw", "taiwan", "hong kong", "zh-hk"):
			return LangTraditional
		case has("简", "簡", "simplified", "hans", "zh-cn"):
			return LangSimplified
		}
		return LangChineseUnknown
	case isEnglishTag(l):
		return "en"
	case l == "":
		return LangUndetermined
	}
	return l
}
