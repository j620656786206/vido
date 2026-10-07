// Package mine learns a show's proper-noun renderings from the official
// zh-Hant subtitles the library already has (story sub-7-5, 加速器②): for every
// episode that carries BOTH an official Chinese subtitle and an English text
// track, the two are aligned by time, English proper nouns are paired with the
// Chinese rendering that keeps showing up next to them, and the pairs become
// `official_subtitle` glossary rows for the whole show — so the episodes that
// still need a paid translation use the renderings the professionals used.
//
// Everything here is pure: file names, parsed cues and strings in, terms out.
// No LLM, no database, no ffmpeg (the caller extracts embedded tracks and
// hands the .srt paths over). Rule 19: this package may import subtitle, never
// services.
package mine

import (
	"path/filepath"
	"strings"

	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
)

// Sources is what one episode offers the miner: the official Chinese side and
// the English side, each either a sidecar path or an embedded stream index.
// Either side may be absent; Usable reports whether both are present.
type Sources struct {
	MediaPath string
	// ZhSidecars are official Traditional-Chinese sidecar files (never Vido's
	// own delivery). EnSidecars are English sidecars.
	ZhSidecars []string
	EnSidecars []string
	// ZhStreams / EnStreams are embedded text-subtitle stream indexes.
	ZhStreams []int
	EnStreams []int
}

// Usable reports whether the episode has a Chinese source AND an English one.
func (s Sources) Usable() bool {
	return (len(s.ZhSidecars) > 0 || len(s.ZhStreams) > 0) && (len(s.EnSidecars) > 0 || len(s.EnStreams) > 0)
}

// HasZh reports whether the episode already has an official Chinese subtitle —
// the same predicate eval/scan-partial-zh.sh calls has_zh.
func (s Sources) HasZh() bool { return len(s.ZhSidecars) > 0 || len(s.ZhStreams) > 0 }

// IsOfficialZhSidecar reports whether a sidecar file name is an OFFICIAL
// Traditional-Chinese subtitle: a zh-TW / cht / tc / 繁 language tag (also the
// .hi / .sdh variants), never Vido's own `.zh-Hant.srt` delivery or its .bak /
// .tmp artifacts. Mirrors eval/scan-partial-zh.sh so the two agree on what
// "partial" means.
func IsOfficialZhSidecar(name string) bool {
	base := filepath.Base(name)
	lower := strings.ToLower(base)
	ext := filepath.Ext(lower)
	if ext != ".srt" && ext != ".ass" && ext != ".ssa" {
		return false
	}
	if strings.HasSuffix(lower, ".bak") || strings.Contains(lower, ".tmp.") {
		return false
	}
	// Vido's own delivery (and its hearing-impaired/SDH variants are NOT
	// Vido's — those are the official .zh-Hant.hi / .zh-Hant.sdh files).
	if strings.HasSuffix(lower, ".zh-hant.srt") {
		return false
	}
	tags := []string{".zh-tw.", ".cht.", ".tc.", ".zh-hant.hi.", ".zh-hant.sdh.", ".zh-hk."}
	for _, t := range tags {
		if strings.Contains(lower, t) {
			return true
		}
	}
	return strings.Contains(base, "繁")
}

// IsEnglishSidecar reports whether a sidecar file name is tagged English.
func IsEnglishSidecar(name string) bool {
	lower := strings.ToLower(filepath.Base(name))
	ext := filepath.Ext(lower)
	if ext != ".srt" && ext != ".ass" && ext != ".ssa" {
		return false
	}
	if strings.HasSuffix(lower, ".bak") || strings.Contains(lower, ".tmp.") {
		return false
	}
	for _, t := range []string{".en.", ".eng.", ".en-us.", ".en-gb.", ".english."} {
		if strings.Contains(lower, t) {
			return true
		}
	}
	return false
}

// Classify builds an episode's Sources from its sidecar file names (as found
// next to the media file) and its probed embedded tracks.
func Classify(mediaPath string, sidecars []string, tracks []services.SubtitleTrack) Sources {
	s := Sources{MediaPath: mediaPath}
	var sdh []int
	for _, name := range sidecars {
		switch {
		case IsOfficialZhSidecar(name):
			s.ZhSidecars = append(s.ZhSidecars, name)
		case IsEnglishSidecar(name):
			s.EnSidecars = append(s.EnSidecars, name)
		}
	}
	for _, t := range tracks {
		if t.External || !subtitle.IsTextSubtitleCodec(t.Format) {
			continue
		}
		switch {
		case subtitle.IsChineseLanguageTag(t.Language):
			// scan-partial-zh.sh counts any chi/zho text track; a Simplified
			// track is still a human translation of the same show.
			s.ZhStreams = append(s.ZhStreams, t.StreamIndex)
		case subtitle.IsEnglishLanguageTag(t.Language):
			// disc-2026-10-mine-en-source-selection: the forced-narrative
			// track is a handful of on-screen captions, not the dialogue —
			// on See S01E07 / S02E04 / S02E05 it was stream 2, first in
			// file order, and the episode aligned 0–1 segments. Skip it;
			// put the full track before the SDH one.
			if subtitle.TrackIsForced(t) {
				continue
			}
			if subtitle.TrackIsSDH(t) {
				sdh = append(sdh, t.StreamIndex)
				continue
			}
			s.EnStreams = append(s.EnStreams, t.StreamIndex)
		}
	}
	s.EnStreams = append(s.EnStreams, sdh...)
	return s
}
