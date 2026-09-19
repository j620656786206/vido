package subtitle

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/segkey"
)

// ─── disc-2026-09-generation-resume-a-translation-cache AC #1 ──────────────
//
// The speech-recognition leg is getting the same per-cue translation cache the
// extract leg has had since sub-1-5b. There must be exactly ONE definition of
// what a cache key is: two definitions drift, and a drifted key is invisible —
// no error, just a cache that stops hitting and a bill that quietly doubles.
//
// The key functions therefore MOVED to internal/segkey (importable by both
// `services` and `subtitle`; Rule 19 forbids services → subtitle). These tests
// pin that the move was byte-for-byte: every key this package produces today is
// the key segkey produces, so the library already cached under the old
// definition keeps hitting after the move.

func TestSegkeyParity_SegmentKey(t *testing.T) {
	versions := []models.RunVersion{
		{},
		{MetadataHash: "m", GlossaryVersion: "g", PromptVersion: "p", ModelID: "claude-sonnet-5"},
		// Field-boundary forgery: "ab"+"" and "a"+"b" must not hash alike.
		{MetadataHash: "ab", GlossaryVersion: ""},
		{MetadataHash: "a", GlossaryVersion: "b"},
	}
	texts := []string{"", "Hello.", "多行\n第二行", "Text with \x1f separator"}
	seen := map[string]bool{}
	for _, v := range versions {
		for _, text := range texts {
			assert.Equal(t, segmentKey(text, v), segkey.SegmentKey(text, v),
				"segkey must reproduce this package's key exactly — otherwise the move orphans every cached cue")
			seen[segmentKey(text, v)] = true
		}
	}
	assert.Len(t, seen, len(versions)*len(texts), "no collisions across the matrix")
}

func TestSegkeyParity_MetadataHash(t *testing.T) {
	cases := []TranslateContext{
		{},
		{Title: "Goblet of Fire", OriginalTitle: "Harry Potter and the Goblet of Fire", Year: 2005,
			Genres: []string{"Fantasy", "Adventure"}, Overview: "A tournament.",
			Cast: []string{"Daniel Radcliffe", "Emma Watson"}, Countries: []string{"GB", "US"}},
		// Sets are sorted on a COPY, cast is NOT (billing order is meaningful).
		{Genres: []string{"Adventure", "Fantasy"}, Countries: []string{"US", "GB"}},
		{Cast: []string{"Emma Watson", "Daniel Radcliffe"}},
	}
	for _, tctx := range cases {
		md := prompts.MediaMetadata{
			Title: tctx.Title, OriginalTitle: tctx.OriginalTitle, Year: tctx.Year,
			Genres: tctx.Genres, Overview: tctx.Overview, Cast: tctx.Cast, Countries: tctx.Countries,
		}
		assert.Equal(t, MetadataHash(tctx), segkey.MetadataHash(md))
	}
	// The glossary rides its own RunVersion field, never the metadata hash.
	withGlossary := TranslateContext{Title: "X", Glossary: []prompts.GlossaryEntry{{Source: "a", Target: "b"}}}
	assert.Equal(t, MetadataHash(TranslateContext{Title: "X"}), MetadataHash(withGlossary))
}

func TestSegkeyParity_GlossaryVersionHash(t *testing.T) {
	cases := [][]prompts.GlossaryEntry{
		nil,
		{},
		{{Source: "Dumbledore", Target: "鄧不利多"}},
		{{Source: "Dumbledore", Target: "鄧不利多"}, {Source: "Snape", Target: "石內卜"}},
		// Sorted on a copy: the caller's order must not change the hash.
		{{Source: "Snape", Target: "石內卜"}, {Source: "Dumbledore", Target: "鄧不利多"}},
	}
	for _, g := range cases {
		assert.Equal(t, GlossaryVersionHash(g), segkey.GlossaryVersionHash(g))
	}
	assert.Equal(t, segkey.GlossaryVersionHash(cases[3]), segkey.GlossaryVersionHash(cases[4]),
		"pair order is the caller's, not the hash's")
	assert.NotEqual(t, segkey.GlossaryVersionHash(nil), segkey.GlossaryVersionHash(cases[2]),
		"a grown glossary re-keys the cache — that is the 向前補一致性 mechanism")
}

// The storage constants move with the key: a second copy of "subtitle_segment"
// or of the 30-day TTL is the same silent-drift risk as a second key function.
func TestSegkeyParity_StorageConstants(t *testing.T) {
	assert.Equal(t, segmentCacheType, segkey.Type)
	assert.Equal(t, segmentCacheTTL, segkey.TTL)
	assert.Equal(t, segmentKeyPrefix, segkey.Prefix)
}
