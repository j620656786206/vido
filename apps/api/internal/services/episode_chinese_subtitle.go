package services

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/vido/api/internal/models"
)

// Implements: disc-2026-10-episode-list-subtitle-badge-a AC #1, #2, #4

// Chinese subtitle source kinds — AC #1 [@contract-v1].
const (
	ChineseSourceEmbedded = "embedded" // a track inside the media file
	ChineseSourceSidecar  = "sidecar"  // a subtitle file beside it
	ChineseSourceVido     = "vido"     // the subtitle Vido itself placed
)

// ChineseSubtitleSource is one Chinese source behind an episode's verdict.
// Language is zh-Hant / zh-Hans / zh-unknown; Label is the embedded track's
// title or the sidecar's filename tag ("zh-TW"), "" when there is none.
type ChineseSubtitleSource struct {
	Kind     string `json:"kind"`
	Language string `json:"language"`
	Label    string `json:"label"`
}

// episodeSubtitleVerdict is what the season list says about one episode.
type episodeSubtitleVerdict struct {
	verdict models.ChineseSubtitle
	sources []ChineseSubtitleSource
	// refresh is the subtitle_tracks JSON to write back ("" = nothing to
	// write): set when the embedded half is known and the live sidecar read
	// differs from what is stored (AC #4).
	refresh string
}

// verdictLanguage maps a per-track verdict to the source's language string.
func verdictLanguage(c models.ChineseSubtitle) string {
	switch c {
	case models.ChineseSubtitleZhHant:
		return "zh-Hant"
	case models.ChineseSubtitleZhHans:
		return "zh-Hans"
	}
	return "zh-unknown"
}

// isChineseVerdict reports whether c says "there is Chinese".
func isChineseVerdict(c models.ChineseSubtitle) bool {
	return c == models.ChineseSubtitleZhHant || c == models.ChineseSubtitleZhHans || c == models.ChineseSubtitleZh
}

// computeEpisodeSubtitle combines the stored embedded tracks, the sidecars
// (live when live is a successful read, else the stored ones) and Vido's own
// record into one verdict through models.ChineseSubtitleVerdict — the same
// rule movies, series and the library filter use.
//
// AC #2: while the embedded half has never been read, a sidecar can prove
// "has Chinese" but not "no Chinese" — an English .srt beside an unprobed
// file says nothing about the tracks inside it — so the verdict then falls
// back to Vido's record alone (usually unknown).
func computeEpisodeSubtitle(ep models.Episode, live *SidecarTracks) episodeSubtitleVerdict {
	status, language := string(ep.SubtitleStatus), ep.SubtitleLanguage.String

	var stored []SubtitleTrack
	embeddedKnown := ep.SubtitleTracks.Valid && json.Unmarshal([]byte(ep.SubtitleTracks.String), &stored) == nil
	var embedded, sidecars []SubtitleTrack
	for _, t := range stored {
		if t.External {
			sidecars = append(sidecars, t)
		} else {
			embedded = append(embedded, t)
		}
	}
	liveOK := live != nil && live.Err == nil
	if liveOK {
		sidecars = live.Tracks
	}

	var out episodeSubtitleVerdict
	if embeddedKnown {
		merged := append(append(make([]SubtitleTrack, 0, len(embedded)+len(sidecars)), embedded...), sidecars...)
		raw, _ := json.Marshal(merged)
		out.verdict = models.ChineseSubtitleVerdict(status, language, string(raw))
		if liveOK {
			canonical, _ := json.Marshal(stored)
			if stored == nil {
				canonical = []byte("[]")
			}
			if string(canonical) != string(raw) {
				out.refresh = string(raw)
			}
		}
	} else {
		raw, _ := json.Marshal(sidecars)
		if v := models.ChineseSubtitleVerdict(status, language, string(raw)); isChineseVerdict(v) {
			out.verdict = v
		} else {
			out.verdict = models.ChineseSubtitleVerdict(status, language, "")
		}
	}

	out.sources = episodeChineseSources(ep, embedded, sidecars)
	return out
}

// episodeChineseSources lists the Chinese sources, embedded first. A sidecar
// that is the file Vido placed (subtitle_path) is reported once, as "vido";
// a found Chinese record whose file is not beside the video still counts.
func episodeChineseSources(ep models.Episode, embedded, sidecars []SubtitleTrack) []ChineseSubtitleSource {
	sources := []ChineseSubtitleSource{}
	for _, t := range embedded {
		if c := models.ChineseSubtitleOfTrack(t.Language, t.Title, t.DetectedLanguage); c != "" {
			sources = append(sources, ChineseSubtitleSource{Kind: ChineseSourceEmbedded, Language: verdictLanguage(c), Label: t.Title})
		}
	}

	mediaPath := ep.FilePath.String
	dir := filepath.Dir(mediaPath)
	stem := strings.TrimSuffix(filepath.Base(mediaPath), filepath.Ext(mediaPath))
	own := ""
	if ep.SubtitlePath.Valid && ep.SubtitlePath.String != "" {
		own = filepath.Clean(ep.SubtitlePath.String)
	}
	vidoListed := false
	for _, t := range sidecars {
		c := models.ChineseSubtitleOfTrack(t.Language, "", "")
		if c == "" {
			continue
		}
		kind := ChineseSourceSidecar
		if own != "" && t.FileName != "" && filepath.Join(dir, t.FileName) == own {
			kind = ChineseSourceVido
			vidoListed = true
		}
		sources = append(sources, ChineseSubtitleSource{Kind: kind, Language: verdictLanguage(c), Label: sidecarTag(t.FileName, stem)})
	}

	if !vidoListed && ep.SubtitleStatus == models.SubtitleStatusFound {
		if c := models.ChineseSubtitleOfTrack(ep.SubtitleLanguage.String, "", ""); c != "" {
			sources = append(sources, ChineseSubtitleSource{Kind: ChineseSourceVido, Language: verdictLanguage(c)})
		}
	}
	return sources
}

// sidecarTag is the part of a sidecar's name between the video's stem and the
// extension: "See.S01E02.zh-TW.srt" beside "See.S01E02.mkv" → "zh-TW".
func sidecarTag(fileName, stem string) string {
	nameNoExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	if !strings.HasPrefix(nameNoExt, stem+".") {
		return ""
	}
	return strings.TrimPrefix(nameNoExt, stem+".")
}

// storedSidecars returns the sidecar half of an episode's stored tracks (nil
// when never read or unreadable) — handed to SidecarTrackReader so unchanged
// files are not reopened.
func storedSidecars(ep models.Episode) []SubtitleTrack {
	if !ep.SubtitleTracks.Valid {
		return nil
	}
	var stored []SubtitleTrack
	if json.Unmarshal([]byte(ep.SubtitleTracks.String), &stored) != nil {
		return nil
	}
	var side []SubtitleTrack
	for _, t := range stored {
		if t.External {
			side = append(side, t)
		}
	}
	return side
}
