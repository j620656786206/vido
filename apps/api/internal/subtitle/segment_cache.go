package subtitle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// ─── Storage port ──────────────────────────────────────────────────────────

// SegmentCache is the narrow port over the AD #4 tier-2 cache. It is
// deliberately NOT repository.CacheRepositoryInterface: the item flow needs two
// of its seven methods and wants misses expressed by ABSENCE from a map rather
// than a nil *CacheEntry, so tests fake two methods instead of seven.
//
// The read side is batch-shaped on purpose (sub-1-5b CR M3, Alexyu ruling
// 2026-08-03): the item flow reads one key per cue, and a per-key Get here is
// an N+1 against SQLite that silently re-emerges whenever the surrounding
// architecture changes — one query per track keeps the shape safe by
// construction rather than by the current timings.
//
// This is the SEGMENT cache — post-OpenCC translated text per cue, keyed by
// content + RunVersion. It has nothing to do with the PROMPT cache, which lives
// provider-side and is recorded in subtitle_runs.cache_enabled (AC #4).
type SegmentCache interface {
	// GetMany returns the found values keyed by cache key; a missing key is
	// simply absent. Absence is not an error.
	GetMany(ctx context.Context, keys []string) (map[string]string, error)
	// Set writes value under key with the given TTL.
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}

const (
	// segmentCacheType tags the tier so ClearByType can evict just this family.
	segmentCacheType = "subtitle_segment"

	// segmentCacheTTL follows the AD #4 AI-parsing precedent. Version-bumped
	// keys orphan stale entries anyway, so the TTL is a floor on storage growth
	// rather than a correctness lever.
	segmentCacheTTL = 30 * 24 * time.Hour

	// segmentKeyPrefix namespaces the key family and versions the key FORMAT
	// (as opposed to RunVersion, which versions the key's INPUTS).
	segmentKeyPrefix = "subseg:v1:"

	// fieldSep separates fields inside a canonical serialization. \x1f is the
	// ASCII unit separator: it cannot occur in TMDb metadata or subtitle text,
	// so "AB"+""+sep and "A"+"B"+sep can never hash alike.
	fieldSep = "\x1f"

	// payloadSep separates the cue text from the version tuple. A distinct byte
	// keeps the two halves un-forgeable against each other.
	payloadSep = "\x00"
)

// segmentCacheRepository adapts repository.CacheRepositoryInterface to
// SegmentCache. It lives here rather than in the repository package because the
// boolean-miss shape is this pipeline's convenience, not a storage concern.
type segmentCacheRepository struct {
	cache repository.CacheRepositoryInterface
}

// NewSegmentCacheRepository wires the segment cache onto the shared
// cache_entries table. sub-1-6 constructs it; no migration is involved — the
// table has existed since the tiered-cache work (AD #4).
func NewSegmentCacheRepository(cache repository.CacheRepositoryInterface) SegmentCache {
	return &segmentCacheRepository{cache: cache}
}

func (r *segmentCacheRepository) GetMany(ctx context.Context, keys []string) (map[string]string, error) {
	entries, err := r.cache.GetMany(ctx, keys)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(entries))
	for key, entry := range entries {
		out[key] = entry.Value
	}
	return out, nil
}

func (r *segmentCacheRepository) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.cache.Set(ctx, key, value, segmentCacheType, ttl)
}

// ─── Keying (AC #3.1, #3.3) ────────────────────────────────────────────────

// MetadataHash is the canonical, field-ordered digest of the FR26 show metadata
// injected into the translation prompt — the MetadataHash half of
// models.RunVersion (sub-1-2 defers its definition here, where the metadata
// shape lives).
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
func MetadataHash(tctx TranslateContext) string {
	fields := []string{
		tctx.Title,
		tctx.OriginalTitle,
		strconv.Itoa(tctx.Year),
		strings.Join(sortedCopy(tctx.Genres), ","),
		tctx.Overview,
		strings.Join(tctx.Cast, ","),
		strings.Join(sortedCopy(tctx.Countries), ","),
	}
	sum := sha256.Sum256([]byte(strings.Join(fields, fieldSep)))
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

// segmentKey is the cue-grain cache key: content hash + the full RunVersion
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
func segmentKey(cueText string, v models.RunVersion) string {
	payload := cueText + payloadSep + strings.Join([]string{
		v.MetadataHash, v.GlossaryVersion, v.PromptVersion, v.ModelID,
	}, fieldSep)

	sum := sha256.Sum256([]byte(payload))
	return segmentKeyPrefix + hex.EncodeToString(sum[:])
}

// GlossaryVersionHash is the GlossaryVersion half of models.RunVersion
// (sub-5-5 AC #5): a canonical digest of the glossary pairs ACTUALLY FED into
// the translation prompt — not a DB snapshot, so cache key and prompt content
// agree by construction (a failed lookup feeds empty and hashes empty).
//
// Empty glossary → "" — backward compatibility with every M1 cache entry,
// which was written under GlossaryVersion "". Pairs are sorted on a COPY by
// Source (MetadataHash's sortedCopy discipline: the prompt renders the
// caller's order, a hash function must not reorder it) and joined with the
// unforgeable fieldSep so "AB"+"" and "A"+"B" can never collide.
func GlossaryVersionHash(glossary []prompts.GlossaryEntry) string {
	if len(glossary) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(glossary))
	for _, e := range glossary {
		pairs = append(pairs, e.Source+fieldSep+e.Target)
	}
	sort.Strings(pairs)
	sum := sha256.Sum256([]byte(strings.Join(pairs, fieldSep)))
	return hex.EncodeToString(sum[:])
}

// runVersion assembles the tuple this run is identified by. GlossaryVersion
// (empty from M1 until sub-5-5 wired the feed) hashes the pairs the prompt
// actually carries, so a grown glossary re-keys the cache and re-translation
// picks the new renderings up — the D4 "向前補一致性" mechanism.
func (p *Pipeline) runVersion(ctx context.Context, tctx TranslateContext) models.RunVersion {
	return models.RunVersion{
		MetadataHash:    MetadataHash(tctx),
		GlossaryVersion: GlossaryVersionHash(tctx.Glossary),
		PromptVersion:   prompts.SubtitleTranslatorPromptVersion,
		ModelID:         p.currentModelID(ctx),
	}
}

// ─── Split / merge (AC #3.5) ───────────────────────────────────────────────

// splitCachedCues partitions a routed track into cues already translated under
// this exact RunVersion (hits, keyed by cue Index) and the cues that still need
// the LLM (misses, in source order). ONE batched read for the whole track —
// per-cue Gets are an N+1 (CR M3).
//
// force bypasses READS only: the operator asked for a fresh translation, but
// the write-back below still refreshes the entry rather than orphaning it.
func (p *Pipeline) splitCachedCues(ctx context.Context, source []SubtitleBlock, v models.RunVersion, force bool) (map[int]string, []SubtitleBlock) {
	hits := make(map[int]string)
	misses := make([]SubtitleBlock, 0, len(source))

	if p.cache == nil || force {
		return hits, append(misses, source...)
	}

	keys := make([]string, len(source))
	for i, b := range source {
		keys[i] = segmentKey(b.Text, v)
	}

	values, err := p.cache.GetMany(ctx, keys)
	if err != nil {
		// Deliberate Rule 13 case-3 discard: a cache read failure costs tokens,
		// never correctness — the whole track simply becomes misses and is
		// translated. Failing the item here would turn a transient SQLite
		// hiccup into a lost episode.
		p.logger.Warn("segment cache read failed — treating the whole track as misses",
			"cue_count", len(source), "error", err)
		return hits, append(misses, source...)
	}

	for i, b := range source {
		if value, ok := values[keys[i]]; ok {
			hits[b.Index] = value
			continue
		}
		misses = append(misses, b)
	}
	return hits, misses
}

// storeCachedCues writes the FINAL post-OpenCC text of the cues this run
// actually translated. Delivering a hit later must be byte-identical to
// delivering the freshly-translated cue, which is only true if what we store is
// what we ship — not the pre-conversion text.
//
// A write failure is logged and swallowed (Rule 13 case 3): the translation
// already succeeded, and failing the item over a cache miss-to-be would be
// strictly worse than paying for it again next run.
func (p *Pipeline) storeCachedCues(ctx context.Context, translated []SubtitleBlock, final map[int]string, v models.RunVersion) {
	if p.cache == nil {
		return
	}

	failures := 0
	for _, b := range translated {
		text, ok := final[b.Index]
		if !ok {
			continue
		}
		if err := p.cache.Set(ctx, segmentKey(b.Text, v), text, segmentCacheTTL); err != nil {
			failures++
			if failures == 1 {
				p.logger.Warn("segment cache write failed — the next run pays for these cues again",
					"cue_index", b.Index, "error", err)
			}
		}
	}
	if failures > 0 {
		p.logger.Warn("segment cache writes failed", "failed_cues", failures, "total_cues", len(translated))
	}
}

// mergeCues rebuilds the full track from the cache hits and the freshly
// translated blocks, keyed by cue Index and ordered by the SOURCE track.
//
// Index/Start/End come from source by construction, which is what keeps the
// FR17 invariant true across a partial-cache run: the merge cannot reorder,
// drop, or renumber anything.
//
// A cue covered by NEITHER side is a hard error rather than a fallback. It is
// impossible today (every cue is a hit or a miss, and TranslateTrack returns
// the whole subset it was given), but leaving the source text in place would
// ship an ENGLISH cue inside a .zh-Hant.srt — and checkTimestampInvariant could
// never catch it, because that guard only compares Index/Start/End. This is the
// one place where "impossible by construction" has to be enforced rather than
// asserted in a comment.
func mergeCues(source []SubtitleBlock, hits map[int]string, translated []SubtitleBlock) ([]SubtitleBlock, error) {
	byIndex := make(map[int]string, len(translated))
	for _, b := range translated {
		byIndex[b.Index] = b.Text
	}

	out := make([]SubtitleBlock, len(source))
	copy(out, source)

	var uncovered []int
	for i := range out {
		if text, ok := byIndex[out[i].Index]; ok {
			out[i].Text = text
			continue
		}
		if text, ok := hits[out[i].Index]; ok {
			out[i].Text = text
			continue
		}
		uncovered = append(uncovered, out[i].Index)
	}

	if len(uncovered) > 0 {
		return nil, fmt.Errorf("%w: %d cue(s) came back from neither the segment cache nor the translator (first: %d)",
			ErrSubtitleTranslateFailed, len(uncovered), uncovered[0])
	}
	return out, nil
}

// compile-time proof that the shipped repository satisfies the narrow port.
var _ SegmentCache = (*segmentCacheRepository)(nil)
