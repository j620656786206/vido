package subtitle

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vido/api/internal/models"
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
	entries, err := os.ReadDir(filepath.Dir(mediaPath))
	if err != nil {
		return nil, err
	}
	return sidecarsFromEntries(entries, mediaPath, ownSubtitlePath), nil
}

// sidecarsFromEntries is ListSidecars over an already-read directory, so a
// season whose episodes share one folder reads it once
// (disc-2026-10-episode-list-subtitle-badge-a AC #3).
func sidecarsFromEntries(entries []os.DirEntry, mediaPath, ownSubtitlePath string) []SidecarFile {
	dir := filepath.Dir(mediaPath)
	stem := strings.TrimSuffix(filepath.Base(mediaPath), filepath.Ext(mediaPath))
	own := ""
	if ownSubtitlePath != "" {
		own = filepath.Clean(ownSubtitlePath)
	}

	files := []SidecarFile{}
	for _, e := range entries {
		format, tag, ok := matchSidecar(e, stem)
		if !ok {
			continue
		}
		path := filepath.Join(dir, e.Name())
		files = append(files, SidecarFile{
			FileName:     e.Name(),
			Language:     sidecarLanguage(path, tag),
			Format:       format,
			IsVidoOutput: own != "" && filepath.Clean(path) == own,
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].FileName < files[j].FileName })
	return files
}

// matchSidecar reports whether a directory entry is a text subtitle file of
// the video whose file stem is stem, with its format and filename tag
// ("zh-TW" of "Ep.zh-TW.srt", "" for "Ep.srt").
func matchSidecar(e os.DirEntry, stem string) (format, tag string, ok bool) {
	if e.IsDir() {
		return "", "", false
	}
	name := e.Name()
	lower := strings.ToLower(name)
	format, ok = sidecarFormats[filepath.Ext(lower)]
	if !ok || strings.Contains(lower, ".tmp.") {
		return "", "", false
	}
	nameNoExt := strings.TrimSuffix(name, filepath.Ext(name))
	if nameNoExt != stem && !strings.HasPrefix(nameNoExt, stem+".") {
		return "", "", false
	}
	return format, strings.TrimPrefix(strings.TrimPrefix(nameNoExt, stem), "."), true
}

// SidecarTrackReader implements services.SidecarTrackReader: the sidecar half
// of an episode's subtitle_tracks, as services.SubtitleTrack values, reading
// each directory once however many episodes live in it. Language is the
// content verdict (zh-Hant / zh-Hans / zh-unknown) or the filename tag; the
// file name goes in FileName, never in Title, so a ".zh-TW" name cannot
// overrule what the text says.
//
// Reading the text is the expensive part on a NAS mount (an open plus up to
// 100KB per file), so a sidecar whose name and size:mtime match what was
// stored keeps its stored Language without being opened again.
type SidecarTrackReader struct{}

// ReadSidecarTracks lists the sidecars of every media path. known maps a media
// path to the sidecar tracks stored for it (may be nil).
func (SidecarTrackReader) ReadSidecarTracks(mediaPaths []string, known map[string][]services.SubtitleTrack) map[string]services.SidecarTracks {
	out := make(map[string]services.SidecarTracks, len(mediaPaths))
	dirs := map[string][]os.DirEntry{}
	dirErrs := map[string]error{}
	for _, mediaPath := range mediaPaths {
		dir := filepath.Dir(mediaPath)
		if _, read := dirs[dir]; !read {
			if _, failed := dirErrs[dir]; !failed {
				if entries, err := os.ReadDir(dir); err != nil {
					dirErrs[dir] = err
				} else {
					dirs[dir] = entries
				}
			}
		}
		if err, failed := dirErrs[dir]; failed {
			out[mediaPath] = services.SidecarTracks{Err: err}
			continue
		}

		prior := map[string]services.SubtitleTrack{}
		for _, t := range known[mediaPath] {
			if t.External && t.FileName != "" && t.FileSig != "" && t.Language != "" {
				prior[t.FileName] = t
			}
		}
		stem := strings.TrimSuffix(filepath.Base(mediaPath), filepath.Ext(mediaPath))
		tracks := []services.SubtitleTrack{}
		for _, e := range dirs[dir] {
			format, tag, ok := matchSidecar(e, stem)
			if !ok {
				continue
			}
			sig := ""
			if info, err := e.Info(); err == nil {
				sig = services.FileSignature(info)
			}
			lang := ""
			if p, seen := prior[e.Name()]; seen && sig != "" && p.FileSig == sig {
				lang = p.Language
			} else {
				lang = sidecarLanguage(filepath.Join(dir, e.Name()), tag)
			}
			tracks = append(tracks, services.SubtitleTrack{
				Language: lang,
				Format:   format,
				External: true,
				FileName: e.Name(),
				FileSig:  sig,
			})
		}
		sort.Slice(tracks, func(i, j int) bool { return tracks[i].FileName < tracks[j].FileName })
		out[mediaPath] = services.SidecarTracks{Tracks: tracks}
	}
	return out
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
// (Simplified)", "Cantonese"). Untold Chinese stays zh-unknown. The title
// keywords are models.ChineseTitleScript — the same list the library's
// "has Chinese subtitles" verdict reads (disc-2026-10-subtitle-filter-
// disagrees-with-badges), so this dialog and the library badge agree.
func embeddedLanguage(lang, title string) string {
	l := strings.ToLower(strings.TrimSpace(lang))
	script := models.ChineseTitleScript(title)
	switch {
	case l == "yue" || ((isChineseTag(l) || l == "") && script == models.TitleScriptCantonese):
		return "yue"
	case l == "zh-hant" || l == "zh-tw" || l == "zh-hk" || l == "cht":
		return LangTraditional
	case l == "zh-hans" || l == "zh-cn" || l == "chs":
		return LangSimplified
	case isChineseTag(l):
		switch script {
		case models.TitleScriptTraditional:
			return LangTraditional
		case models.TitleScriptSimplified:
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
