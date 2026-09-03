package prompts

import (
	"regexp"
	"strings"
)

// LooksLikeFilename reports whether a "title" is really a release filename —
// the shape an unmatched library row carries after the scanner falls back to
// the file's own name. Sending that into the translation prompt as the show's
// Title is noise, not context (sub-6-7; eval-1 finding 12 —
// `[bitsearch.to] Wake.Up...NAHOM.mkv` went out as the film's name and into
// the metadata hash).
//
// Deliberately conservative: every rule needs a token no real title has, so
// `Dune: Part Two`, `S.W.A.T.` and `Mr. Robot` stay titles. Three signals:
//
//   - a video-container extension at the end (`.mkv`, `.mp4`, …)
//   - a tracker / group tag in leading square brackets (`[bitsearch.to] …`)
//   - dot-separated words (≥ 3 segments, no spaces) carrying a release token —
//     a resolution, a source/codec tag, or a bare year
func LooksLikeFilename(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if videoExtensionPattern.MatchString(s) {
		return true
	}
	if strings.HasPrefix(s, "[") {
		return true
	}
	if !strings.Contains(s, " ") && strings.Count(s, ".") >= 2 && releaseTokenPattern.MatchString(s) {
		return true
	}
	return false
}

var (
	videoExtensionPattern = regexp.MustCompile(`(?i)\.(mkv|mp4|avi|m4v|ts|wmv|mov|webm)$`)
	// Tokens that appear in release names and never in a film's title:
	// resolution (`2160p`), source / codec / HDR tags, or a dot-delimited year.
	releaseTokenPattern = regexp.MustCompile(`(?i)(\b\d{3,4}p\b|\bWEB-?DL\b|\bWEBRip\b|\bBluRay\b|\bBDRip\b|\bHDRip\b|\bREMUX\b|\bHEVC\b|\bx26[45]\b|\bH\.?26[45]\b|\bHDR10?\b|\bDV\b|\bAAC\b|\bDTS\b|\bDDP?5?\b|(^|\.)(19|20)\d{2}(\.|$))`)
)
