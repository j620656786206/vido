package services

import (
	"context"
	"time"

	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/segkey"
)

// ─── Segment store (disc-2026-09-generation-resume-a-translation-cache) ────
//
// A feature-length subtitle costs ~$2.7 and ~191 model calls to translate. Until
// this story, a run that stopped part-way — the batch money ceiling, a deadline,
// a restart — threw every finished batch away, so the next run started at cue 1
// and stopped in the same place, forever. Each finished batch now lands here the
// moment it comes back.
//
// The rows are the SAME rows the extract leg (`internal/subtitle`) has used
// since sub-1-5b: the shared `cache_entries` table, type `subtitle_segment`,
// TTL 30 days, keys from `internal/segkey`. No migration is involved.

// SegmentStore is the narrow port over the per-cue translation cache (Rule 11),
// the same two-method shape `subtitle.SegmentCache` uses: the flow needs two of
// the repository's seven methods and wants a miss expressed by ABSENCE from the
// map rather than a nil entry.
//
// The read side is batch-shaped on purpose: one key per cue means a per-key Get
// is an N+1 against SQLite, and a 1,910-cue film would pay it 1,910 times.
type SegmentStore interface {
	// GetMany returns the found values keyed by cache key; a missing key is
	// simply absent. Absence is not an error.
	GetMany(ctx context.Context, keys []string) (map[string]string, error)
	// Set writes value under key with the given TTL.
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}

// segmentStoreRepository adapts repository.CacheRepositoryInterface to
// SegmentStore, tagging every row with segkey.Type so both legs' entries share
// one family (and one ClearByType eviction).
type segmentStoreRepository struct {
	cache repository.CacheRepositoryInterface
}

// NewSegmentStore wires the segment cache onto the shared `cache_entries`
// table. Byte-compatible with subtitle.NewSegmentCacheRepository by
// construction: same key function, same type tag, same TTL.
func NewSegmentStore(cache repository.CacheRepositoryInterface) SegmentStore {
	return &segmentStoreRepository{cache: cache}
}

func (r *segmentStoreRepository) GetMany(ctx context.Context, keys []string) (map[string]string, error) {
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

func (r *segmentStoreRepository) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.cache.Set(ctx, key, value, segkey.Type, ttl)
}

var _ SegmentStore = (*segmentStoreRepository)(nil)
