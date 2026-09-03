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
// `Dune: Part Two`, `S.W.A.T.`, `Mr. Robot`, `[REC]` and `Amélie.2001` stay
// titles. Rules, any one of which is enough:
//
//   - a video-container extension at the end (`.mkv`, `.mp4`, …)
//   - a STRONG release token anywhere — a resolution (`2160p`) or a
//     source/codec tag (`WEB-DL`, `BluRay`, `REMUX`, `x265`, `HEVC`, `HDR10`) —
//     spaces or dots, it does not matter; no film is called that
//   - a leading bracket tag (`[bitsearch.to] …`, `[字幕组] …`) FOLLOWED by
//     dotted words with a release year — `[REC]` alone is a film
//   - dotted words (≥ 2 dots, no spaces) carrying a dot-delimited year:
//     `Youth.2017.HQ`, `Ip.Man.2008.Extended`
//
// Two-letter tags (`DV`, `DD`, `DTS`, `AAC`) are deliberately NOT signals on
// their own: they have no discriminating power inside a dotted title
// (sub-6-7 CR M6) and RE2's `\b` is ASCII-only, so they would also fire next
// to CJK characters.
func LooksLikeFilename(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if videoExtensionPattern.MatchString(s) {
		return true
	}
	if strongReleaseTokenPattern.MatchString(s) {
		return true
	}
	dotted := !strings.Contains(s, " ") && strings.Count(s, ".") >= 2
	if dotted && dottedYearPattern.MatchString(s) {
		return true
	}
	if strings.HasPrefix(s, "[") {
		// `[tag] Dotted.Words.2025…` — the bracket alone is not evidence.
		rest := strings.TrimSpace(s[strings.IndexByte(s, ']')+1:])
		if strings.IndexByte(s, ']') > 0 && strings.Count(rest, ".") >= 2 && dottedYearPattern.MatchString(rest) {
			return true
		}
	}
	return false
}

// UntrustedTitleReason is the ONE rule both translation legs apply before a
// row's identity reaches the prompt (Rule 19 mirror: subtitle/media_store.go
// and services/transcription_service.go must agree). It returns "" when the
// title can be trusted as show context, otherwise the reason to log:
//
//   - "filename-shaped": the title is a release name. Blank Title and
//     OriginalTitle. If the row is ALSO unmatched, blank Year and Overview as
//     well — they came from the same filename parse. A matched row keeps
//     TMDb's Year/Overview (sub-6-7 CR H3: those are trustworthy).
//
// An unmatched row whose title is NOT filename-shaped is trusted: the parser
// produced a clean name, or the user typed one in the metadata editor
// (sub-6-7 CR M4) — discarding that would throw away real context.
func UntrustedTitleReason(title string, tmdbMatched bool) string {
	if !LooksLikeFilename(title) {
		return ""
	}
	if tmdbMatched {
		return "filename-shaped"
	}
	return "unmatched-filename-shaped"
}

var (
	videoExtensionPattern = regexp.MustCompile(`(?i)\.(mkv|mp4|avi|m4v|ts|wmv|mov|webm)$`)
	// Tokens that appear in release names and never in a film's title.
	strongReleaseTokenPattern = regexp.MustCompile(`(?i)(^|[\s.\[\]_-])(\d{3,4}p|WEB-?DL|WEBRip|BluRay|BDRip|HDRip|REMUX|HEVC|x26[45]|H\.?26[45]|HDR10\+?)($|[\s.\[\]_-])`)
	// A bare four-digit year between dots or at either end of a dotted name.
	dottedYearPattern = regexp.MustCompile(`(^|\.)(19|20)\d{2}(\.|$)`)
)
