// Package segkey owns the ONE definition of a subtitle segment-cache key.
//
// Two legs translate subtitles: the extract leg (`internal/subtitle`, embedded
// text tracks) and the speech-recognition leg (`internal/services`, Whisper).
// Both cache per-cue translations in the shared `cache_entries` table, and Rule
// 19 forbids `services` importing `subtitle` (that would cycle). Without a
// shared home the key function would have to exist twice — and two copies of a
// hash drift silently: no error, no failing test, just a cache that stops
// hitting and a bill that quietly doubles.
//
// This package therefore sits below both: it imports only `models` and
// `ai/prompts`, which are themselves dependency-free. `subtitle` delegates to
// it (its own tests pin the result byte-for-byte), so the library cached under
// the pre-move definition keeps hitting.
package segkey

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
)

const (
	// Type tags the tier so ClearByType can evict just this family.
	Type = "subtitle_segment"

	// TTL follows the AD #4 AI-parsing precedent. Version-bumped keys orphan
	// stale entries anyway, so the TTL is a floor on storage growth rather than
	// a correctness lever.
	TTL = 30 * 24 * time.Hour

	// Prefix namespaces the key family and versions the key FORMAT (as opposed
	// to RunVersion, which versions the key's INPUTS).
	Prefix = "subseg:v1:"

	// fieldSep separates fields inside a canonical serialization. \x1f is the
	// ASCII unit separator: it cannot occur in TMDb metadata or subtitle text,
	// so "AB"+""+sep and "A"+"B"+sep can never hash alike.
	fieldSep = "\x1f"

	// payloadSep separates the cue text from the version tuple. A distinct byte
	// keeps the two halves un-forgeable against each other.
	payloadSep = "\x00"
)

// SegmentKey is the cue-grain cache key: content hash + the full RunVersion
// tuple (P1 + D4).
//
// Content, never index. SDH filtering removes cues without renumbering the
// survivors, so the same line can carry different indexes across two runs of
// the same file; an index-keyed cache would silently return a neighbouring
// cue's translation from the first filtered line onwards.
//
// All four version fields participate. Dropping prompt version or model id is
// the named silent-failure trap: changing the prompt and re-running would serve
// the previous translation back, so two pilot variants would look identical.
func SegmentKey(cueText string, v models.RunVersion) string {
	payload := cueText + payloadSep + strings.Join([]string{
		v.MetadataHash, v.GlossaryVersion, v.PromptVersion, v.ModelID,
	}, fieldSep)

	sum := sha256.Sum256([]byte(payload))
	return Prefix + hex.EncodeToString(sum[:])
}

// MetadataHash is the canonical, field-ordered digest of the FR26 show metadata
// injected into the translation prompt — the MetadataHash half of
// models.RunVersion.
//
// Determinism is the whole contract. If the same show hashes differently across
// two runs, the segment cache silently splits in half and the M1 pilot's
// prompt-variant A/B compares two populations that never shared a cache — with
// no error surfacing anywhere. Hence: a fixed field order, an unforgeable unit
// separator, and set-valued fields normalized by sorting a COPY.
//
// Genres and countries are sorted because they are SETS whose upstream ordering
// is not guaranteed stable; cast is NOT sorted because it arrives in billing
// order and is rendered in that order, so a reordering is a genuinely different
// prompt. The glossary is excluded: it has its own RunVersion field.
func MetadataHash(m prompts.MediaMetadata) string {
	fields := []string{
		m.Title,
		m.OriginalTitle,
		strconv.Itoa(m.Year),
		strings.Join(sortedCopy(m.Genres), ","),
		m.Overview,
		strings.Join(m.Cast, ","),
		strings.Join(sortedCopy(m.Countries), ","),
	}
	sum := sha256.Sum256([]byte(strings.Join(fields, fieldSep)))
	return hex.EncodeToString(sum[:])
}

// GlossaryVersionHash is the GlossaryVersion half of models.RunVersion: a
// canonical digest of the glossary pairs ACTUALLY FED into the translation
// prompt — not a DB snapshot, so cache key and prompt content agree by
// construction (a failed lookup feeds empty and hashes empty).
//
// The built-in lexicon is part of what the prompt carries — its global terms
// section and its post-processing table — so its version is folded in: an empty
// show glossary hashes the lexicon version alone. Pairs are sorted on a COPY by
// Source (MetadataHash's sortedCopy discipline: the prompt renders the caller's
// order, a hash function must not reorder it) and joined with the unforgeable
// fieldSep so "AB"+"" and "A"+"B" can never collide.
func GlossaryVersionHash(glossary []prompts.GlossaryEntry) string {
	pairs := make([]string, 0, len(glossary)+1)
	for _, e := range glossary {
		pairs = append(pairs, e.Source+fieldSep+e.Target)
	}
	sort.Strings(pairs)
	pairs = append(pairs, "lexicon"+fieldSep+prompts.LexiconVersion())
	sum := sha256.Sum256([]byte(strings.Join(pairs, fieldSep)))
	return hex.EncodeToString(sum[:])
}

// sortedCopy sorts without touching the caller's slice — the prompt renders the
// ORIGINAL order, and silently reordering a caller's metadata from a hash
// function would be an invisible side effect.
func sortedCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}
