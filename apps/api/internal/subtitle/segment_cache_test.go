package subtitle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	_ "modernc.org/sqlite"
)

// newMigratedTestDB applies the REAL migration chain to an in-memory SQLite so
// the cache and provenance paths are exercised against the shipped schema. Rule
// 15 / bugfix-20-1: a mocked repository cannot catch a column that exists in the
// table but is missing from a SELECT list.
func newMigratedTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

// memorySegmentCache is the in-test SegmentCache. It records every read and
// write so a test can prove force bypassed READS while still WRITING.
type memorySegmentCache struct {
	entries map[string]string
	reads   []string
	writes  map[string]string
	ttls    []time.Duration
	getErr  error
	setErr  error
}

func newMemorySegmentCache() *memorySegmentCache {
	return &memorySegmentCache{entries: map[string]string{}, writes: map[string]string{}}
}

func (c *memorySegmentCache) GetMany(_ context.Context, keys []string) (map[string]string, error) {
	c.reads = append(c.reads, keys...)
	if c.getErr != nil {
		return nil, c.getErr
	}
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if v, ok := c.entries[key]; ok {
			out[key] = v
		}
	}
	return out, nil
}

func (c *memorySegmentCache) Set(_ context.Context, key, value string, ttl time.Duration) error {
	if c.setErr != nil {
		return c.setErr
	}
	c.entries[key] = value
	c.writes[key] = value
	c.ttls = append(c.ttls, ttl)
	return nil
}

// ─── MetadataHash canonicalization (AC #3.1) ───────────────────────────────

func richContext() TranslateContext {
	return TranslateContext{
		Title:         "怪奇物語",
		OriginalTitle: "Stranger Things",
		Year:          2016,
		Genres:        []string{"Drama", "Sci-Fi & Fantasy", "Mystery"},
		Overview:      "When a young boy vanishes…",
		Cast:          []string{"Winona Ryder", "David Harbour"},
		Countries:     []string{"US", "CA"},
	}
}

// TestMetadataHash_IsCanonical is the story's named silent-failure guard:
// non-determinism here splits the segment cache in half and invalidates the M1
// pilot's A/B comparison, with nothing surfacing an error.
func TestMetadataHash_IsCanonical(t *testing.T) {
	base := MetadataHash(richContext())

	t.Run("stable across repeated calls", func(t *testing.T) {
		assert.Equal(t, base, MetadataHash(richContext()))
	})

	t.Run("struct literal field order is irrelevant", func(t *testing.T) {
		shuffled := TranslateContext{
			Countries:     []string{"US", "CA"},
			Overview:      "When a young boy vanishes…",
			Year:          2016,
			Cast:          []string{"Winona Ryder", "David Harbour"},
			OriginalTitle: "Stranger Things",
			Genres:        []string{"Drama", "Sci-Fi & Fantasy", "Mystery"},
			Title:         "怪奇物語",
		}
		assert.Equal(t, base, MetadataHash(shuffled))
	})

	t.Run("set-valued fields are order-normalized", func(t *testing.T) {
		reordered := richContext()
		reordered.Genres = []string{"Mystery", "Drama", "Sci-Fi & Fantasy"}
		reordered.Countries = []string{"CA", "US"}
		assert.Equal(t, base, MetadataHash(reordered),
			"genres and production countries are sets — TMDb ordering jitter must not split the cache")
	})

	t.Run("normalizing does not mutate the caller's slices", func(t *testing.T) {
		tctx := richContext()
		tctx.Genres = []string{"Mystery", "Drama", "Sci-Fi & Fantasy"}
		before := append([]string(nil), tctx.Genres...)
		MetadataHash(tctx)
		assert.Equal(t, before, tctx.Genres, "the hash must sort a COPY — the prompt renders the caller's order")
	})

	t.Run("cast order is significant", func(t *testing.T) {
		reordered := richContext()
		reordered.Cast = []string{"David Harbour", "Winona Ryder"}
		assert.NotEqual(t, base, MetadataHash(reordered),
			"cast arrives in billing order and is rendered in that order — a different order is a different prompt")
	})

	t.Run("every field participates", func(t *testing.T) {
		mutations := map[string]func(*TranslateContext){
			"title":          func(c *TranslateContext) { c.Title = "怪奇物語 2" },
			"original title": func(c *TranslateContext) { c.OriginalTitle = "Stranger Things 2" },
			"year":           func(c *TranslateContext) { c.Year = 2017 },
			"genres":         func(c *TranslateContext) { c.Genres = append(c.Genres, "Horror") },
			"overview":       func(c *TranslateContext) { c.Overview = "Something else entirely" },
			"cast":           func(c *TranslateContext) { c.Cast = append(c.Cast, "Millie Bobby Brown") },
			"countries":      func(c *TranslateContext) { c.Countries = append(c.Countries, "GB") },
		}
		for name, mutate := range mutations {
			t.Run(name, func(t *testing.T) {
				tctx := richContext()
				mutate(&tctx)
				assert.NotEqual(t, base, MetadataHash(tctx))
			})
		}
	})

	t.Run("field boundaries cannot be forged", func(t *testing.T) {
		a, b := richContext(), richContext()
		a.Title, a.OriginalTitle = "AB", ""
		b.Title, b.OriginalTitle = "A", "B"
		assert.NotEqual(t, MetadataHash(a), MetadataHash(b),
			"a unit separator between fields is what stops concatenation collisions")
	})

	t.Run("the glossary is deliberately excluded", func(t *testing.T) {
		// GlossaryVersion is its own RunVersion field (always "" in M1); folding
		// the entries into MetadataHash would version the same input twice.
		withGlossary := richContext()
		withGlossary.Glossary = append(withGlossary.Glossary, prompts.GlossaryEntry{Source: "Demogorgon", Target: "魔王獸"})
		assert.Equal(t, base, MetadataHash(withGlossary))
	})
}

// ─── Segment key (AC #3.3) ─────────────────────────────────────────────────

func TestSegmentKey_ContentHashedAndFullyVersioned(t *testing.T) {
	version := testVersion()
	base := segmentKey("Good morning.", version)

	assert.True(t, len(base) > len(segmentKeyPrefix), "the key must carry a digest, not just the prefix")
	assert.Equal(t, segmentKeyPrefix, base[:len(segmentKeyPrefix)])

	t.Run("same cue and version is the same key", func(t *testing.T) {
		assert.Equal(t, base, segmentKey("Good morning.", testVersion()))
	})

	t.Run("different cue text is a different key", func(t *testing.T) {
		assert.NotEqual(t, base, segmentKey("Good evening.", version))
	})

	t.Run("the key is content-hashed, never index-keyed", func(t *testing.T) {
		// P1's exact trap: SDH filtering shifts cue positions, so two runs of the
		// same file can hand the same text different indexes. Identical text must
		// still hit.
		assert.Equal(t, segmentKey("Good morning.", version), segmentKey("Good morning.", version))
	})

	t.Run("every RunVersion field participates", func(t *testing.T) {
		mutations := map[string]func(*models.RunVersion){
			"metadata hash":    func(v *models.RunVersion) { v.MetadataHash = "other-hash" },
			"glossary version": func(v *models.RunVersion) { v.GlossaryVersion = "g2" },
			"prompt version":   func(v *models.RunVersion) { v.PromptVersion = "m1-v2" },
			"model id":         func(v *models.RunVersion) { v.ModelID = "claude-sonnet-5" },
		}
		for name, mutate := range mutations {
			t.Run(name, func(t *testing.T) {
				v := testVersion()
				mutate(&v)
				assert.NotEqual(t, base, segmentKey("Good morning.", v),
					"omitting %s from the key is the silent-failure trap the M1 pilot cannot tolerate", name)
			})
		}
	})

	t.Run("version field boundaries cannot be forged", func(t *testing.T) {
		a, b := testVersion(), testVersion()
		a.MetadataHash, a.GlossaryVersion = "ab", ""
		b.MetadataHash, b.GlossaryVersion = "a", "b"
		assert.NotEqual(t, segmentKey("Good morning.", a), segmentKey("Good morning.", b))
	})

	t.Run("the cue/version boundary cannot be forged", func(t *testing.T) {
		v := testVersion()
		v.MetadataHash = ""
		assert.NotEqual(t, segmentKey("x", v), segmentKey("x"+"\x00", v))
	})
}

// ─── Split / merge (AC #3.5) ───────────────────────────────────────────────

func TestSplitCachedCues_HitsNeverReachTheTranslator(t *testing.T) {
	version := testVersion()
	source := cues("Good morning.", "This software is great.", "See you later.")

	cache := newMemorySegmentCache()
	cache.entries[segmentKey(source[1].Text, version)] = "這個軟體很好用"

	p := NewPipeline(&fakeTranslator{}, &recordingConverter{}, nil, WithSegmentCache(cache))
	hits, misses := p.splitCachedCues(context.Background(), source, version, false)

	assert.Equal(t, map[int]string{2: "這個軟體很好用"}, hits)
	require.Len(t, misses, 2)
	assert.Equal(t, []int{1, 3}, []int{misses[0].Index, misses[1].Index},
		"only the misses may be sent to the LLM, and they keep their source order")
}

func TestSplitCachedCues_ForceBypassesReadsButNotWrites(t *testing.T) {
	version := testVersion()
	source := cues("Good morning.")

	cache := newMemorySegmentCache()
	cache.entries[segmentKey(source[0].Text, version)] = "早安（快取）"

	p := NewPipeline(&fakeTranslator{}, &recordingConverter{}, nil, WithSegmentCache(cache))
	hits, misses := p.splitCachedCues(context.Background(), source, version, true)

	assert.Empty(t, hits, "force must bypass cache READS")
	assert.Len(t, misses, 1)
	assert.Empty(t, cache.reads, "a forced run must not even query the cache")

	p.storeCachedCues(context.Background(), source, map[int]string{1: "早安（新）"}, version)
	assert.Equal(t, "早安（新）", cache.writes[segmentKey(source[0].Text, version)],
		"force still WRITES — a re-run refreshes the cache rather than orphaning it")
}

func TestSplitCachedCues_ReadFailureDegradesToAMiss(t *testing.T) {
	source := cues("Good morning.", "See you later.")
	cache := newMemorySegmentCache()
	// One cue IS cached — but the read is one batched query per track now, so a
	// failure degrades the WHOLE track to misses, cached entry included.
	cache.entries[segmentKey(source[0].Text, testVersion())] = "早安"
	cache.getErr = errors.New("sqlite is busy")

	p := NewPipeline(&fakeTranslator{}, &recordingConverter{}, nil, WithSegmentCache(cache))
	hits, misses := p.splitCachedCues(context.Background(), source, testVersion(), false)

	assert.Empty(t, hits)
	assert.Len(t, misses, 2, "a cache read failure costs tokens, never correctness")
}

func TestSplitCachedCues_NilCacheTreatsEveryCueAsAMiss(t *testing.T) {
	source := cues("Good morning.", "See you later.")
	p := NewPipeline(&fakeTranslator{}, &recordingConverter{}, nil)

	hits, misses := p.splitCachedCues(context.Background(), source, testVersion(), false)
	assert.Empty(t, hits)
	assert.Len(t, misses, 2)
}

func TestStoreCachedCues_WritesFinalTextAtTheThirtyDayTTL(t *testing.T) {
	version := testVersion()
	source := cues("Good morning.", "See you later.")
	cache := newMemorySegmentCache()

	p := NewPipeline(&fakeTranslator{}, &recordingConverter{}, nil, WithSegmentCache(cache))
	// Only cue 1 was translated this run; cue 2 came from the cache and must not
	// be rewritten.
	p.storeCachedCues(context.Background(), source[:1], map[int]string{1: "早安"}, version)

	assert.Equal(t, map[string]string{segmentKey(source[0].Text, version): "早安"}, cache.writes)
	assert.Equal(t, []time.Duration{segmentCacheTTL}, cache.ttls)
	assert.Equal(t, 30*24*time.Hour, segmentCacheTTL)
}

func TestStoreCachedCues_WriteFailureIsNonFatal(t *testing.T) {
	cache := newMemorySegmentCache()
	cache.setErr = errors.New("disk full")
	source := cues("Good morning.")

	p := NewPipeline(&fakeTranslator{}, &recordingConverter{}, nil, WithSegmentCache(cache))
	// No panic, no error return: a cache write is an optimisation, and failing
	// the item after a successful translation would be strictly worse.
	p.storeCachedCues(context.Background(), source, map[int]string{1: "早安"}, testVersion())
	assert.Empty(t, cache.writes)
}

func TestMergeCues_RestoresFullTrackOrderAndIdentity(t *testing.T) {
	source := cues("Good morning.", "This software is great.", "See you later.")
	// Cue 2 came from the cache, 1 and 3 from this run's translation.
	merged, err := mergeCues(source, map[int]string{2: "這個軟體很好用"}, []SubtitleBlock{
		{Index: 1, Start: source[0].Start, End: source[0].End, Text: "早安"},
		{Index: 3, Start: source[2].Start, End: source[2].End, Text: "待會見"},
	})
	require.NoError(t, err)

	require.Len(t, merged, 3)
	assert.Equal(t, []string{"早安", "這個軟體很好用", "待會見"},
		[]string{merged[0].Text, merged[1].Text, merged[2].Text})
	for i := range source {
		assert.Equal(t, source[i].Index, merged[i].Index)
		assert.Equal(t, source[i].Start, merged[i].Start)
		assert.Equal(t, source[i].End, merged[i].End)
	}
}

// ─── RunVersion assembly (AC #3.2) ─────────────────────────────────────────

func TestPipeline_RunVersion(t *testing.T) {
	p := NewPipeline(&fakeTranslator{}, &recordingConverter{}, nil, WithModelID("claude-haiku-4-5"))
	tctx := richContext()

	v := p.runVersion(tctx)
	assert.Equal(t, MetadataHash(tctx), v.MetadataHash)
	assert.Empty(t, v.GlossaryVersion, "M1 ships no glossary — the field is versioned now, populated in P2")
	assert.Equal(t, prompts.SubtitleTranslatorPromptVersion, v.PromptVersion)
	assert.Equal(t, "claude-haiku-4-5", v.ModelID)
}

// ─── Repository adapter (AC #3.4, #3.6) ────────────────────────────────────

func TestSegmentCacheRepository_RoundTripsThroughCacheEntries(t *testing.T) {
	db := newMigratedTestDB(t)
	adapter := NewSegmentCacheRepository(repository.NewCacheRepository(db))
	ctx := context.Background()

	require.NoError(t, adapter.Set(ctx, "subseg:v1:present", "早安", segmentCacheTTL))

	// One batched read covering a hit and a miss: absence is not an error, the
	// missing key is simply not in the map.
	values, err := adapter.GetMany(ctx, []string{"subseg:v1:present", "subseg:v1:absent"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"subseg:v1:present": "早安"}, values)

	var cacheType string
	require.NoError(t, db.QueryRow(`SELECT type FROM cache_entries WHERE key = ?`, "subseg:v1:present").Scan(&cacheType))
	assert.Equal(t, segmentCacheType, cacheType, "the tier must be identifiable for ClearByType")
}

// TestSegmentCacheRepository_GetManySpansTheChunkBoundary — the repository
// chunks its IN (...) at 500 placeholders to stay under SQLite's historical
// 999-variable floor. A 1000-cue episode is exactly the real workload that
// crosses that boundary, so the seam gets its own test: every key on both
// sides of the chunk edge must come back.
func TestSegmentCacheRepository_GetManySpansTheChunkBoundary(t *testing.T) {
	db := newMigratedTestDB(t)
	adapter := NewSegmentCacheRepository(repository.NewCacheRepository(db))
	ctx := context.Background()

	const n = 1201 // 3 chunks: 500 + 500 + 201
	keys := make([]string, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("subseg:v1:chunk-%04d", i)
		if i%3 != 0 { // two thirds present, one third missing
			require.NoError(t, adapter.Set(ctx, keys[i], fmt.Sprintf("譯文-%d", i), segmentCacheTTL))
		}
	}

	values, err := adapter.GetMany(ctx, keys)
	require.NoError(t, err)

	want := 0
	for i, key := range keys {
		if i%3 == 0 {
			assert.NotContains(t, values, key)
			continue
		}
		want++
		assert.Equal(t, fmt.Sprintf("譯文-%d", i), values[key])
	}
	assert.Len(t, values, want, "every present key across all three chunks must come back")
}

// TestMergeCues_DesyncIsCaughtByTheInvariant documents why ProcessItem re-runs
// checkTimestampInvariant over the FULL merged track rather than trusting
// TranslateTrack's own check: TranslateTrack only sees the miss subset, so a
// merge that reordered, dropped or renumbered cues would be invisible to it.
// The merge cannot desync today — that is the point; the guard exists so a
// future refactor fails loudly instead of shipping subtitles that drift.
func TestMergeCues_DesyncIsCaughtByTheInvariant(t *testing.T) {
	source := cues("Good morning.", "See you later.")
	merged, err := mergeCues(source, map[int]string{2: "待會見"}, []SubtitleBlock{
		{Index: 1, Start: source[0].Start, End: source[0].End, Text: "早安"},
	})
	require.NoError(t, err)
	require.NoError(t, checkTimestampInvariant(source, merged))

	// Simulate the class of bug the guard is there for.
	merged[1].Start = "99:99:99,999"
	assert.ErrorIs(t, checkTimestampInvariant(source, merged), ErrSubtitleTimestampMismatch)

	assert.Error(t, checkTimestampInvariant(source, merged[:1]), "a dropped cue must not pass either")
}

// TestMergeCues_AnUncoveredCueIsAHardError is the gap checkTimestampInvariant
// structurally CANNOT close: that guard compares Index/Start/End only, so a cue
// covered by neither the cache nor the translator would keep its ENGLISH source
// text and sail straight through into a .zh-Hant.srt. The merge has to catch it
// itself.
func TestMergeCues_AnUncoveredCueIsAHardError(t *testing.T) {
	source := cues("Good morning.", "This software is great.", "See you later.")

	// Cue 2 is in neither bucket — the shape a future refactor of the split
	// would produce.
	merged, err := mergeCues(source, map[int]string{1: "早安"}, []SubtitleBlock{
		{Index: 3, Start: source[2].Start, End: source[2].End, Text: "待會見"},
	})

	require.Error(t, err, "shipping the English source under a zh-Hant name must never be the fallback")
	assert.ErrorIs(t, err, ErrSubtitleTranslateFailed)
	assert.Contains(t, err.Error(), "first: 2")
	assert.Nil(t, merged)

	// Proof that the invariant alone would NOT have caught it.
	silent := make([]SubtitleBlock, len(source))
	copy(silent, source)
	silent[0].Text, silent[2].Text = "早安", "待會見"
	require.NoError(t, checkTimestampInvariant(source, silent),
		"the timestamp guard passes an untranslated cue — which is exactly why mergeCues must not")
}
