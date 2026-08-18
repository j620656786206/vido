package services

// Story sub-5-4 — the route cache.
//
// The F14 sweep classifies every item missing a zh-Hant subtitle, and every
// item whose track list is not already persisted costs one ffprobe. Episodes
// are ALWAYS in that class (their table has no tech-info columns at all), so a
// 500-episode library pays 500 probes — again after every batch, again after
// every restart, even when not one byte changed. This file makes the second
// sweep cheap by remembering the verdict against the file's identity.
//
// It caches the VERDICT, not the tracks: storing tracks would be rebuilding a
// tech-info table for episodes, which is a different story
// (`backlog-episode-tech-info-parity`).

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/vido/api/internal/repository"
)

// ─── Storage port ──────────────────────────────────────────────────────────

// RouteCache is the narrow port (Rule 11) over the shared `cache_entries`
// table. It is deliberately NOT repository.CacheRepositoryInterface: the sweep
// needs two of its seven methods and wants a miss expressed by ABSENCE from a
// map rather than a nil *CacheEntry, so a fake implements two methods instead
// of seven. Same shape, same reason, as subtitle.SegmentCache.
//
// The read side is batch-only ON PURPOSE. A per-key Get here is an N+1 against
// SQLite that a 1,200-episode library would issue 1,200 times before it could
// show a list — the exact anti-pattern sub-1-5b CR M3 corrected in the segment
// cache. Keeping the port batch-shaped makes that safe by construction rather
// than by whoever writes the next caller.
type RouteCache interface {
	// GetMany returns the found values keyed by cache key; a missing key is
	// simply absent. Absence is not an error.
	GetMany(ctx context.Context, keys []string) (map[string]string, error)
	// Set writes value under key with the given TTL.
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}

const (
	// routeCacheType tags the family so ClearByType can evict just these
	// entries, and so the shared expiry sweep (infra-cache-entries-expiry-sweep)
	// covers them with no new wiring.
	routeCacheType = "subtitle_route"

	// routeCacheTTL follows the AD #4 AI-parsing precedent. Correctness is the
	// KEY's job (a changed file changes its key and therefore misses), so the
	// TTL is only a floor on storage growth — it stops a deleted movie's
	// verdict from squatting in the table forever.
	routeCacheTTL = 30 * 24 * time.Hour

	// routeVersion versions the PREDICATE, not the data.
	//
	// BUMP RULE — bump this whenever the meaning of a route verdict changes:
	// any change to subtitle.PredictFromTracks / Router.PredictRoute (reached
	// here through the RoutePredictor port), to SelectCandidates' notion of a
	// "usable text track", or to the RoutePrediction constants themselves.
	// Every cached verdict was written by the OLD predicate; leaving the
	// version alone would serve those stale answers for up to routeCacheTTL,
	// and nothing else in the system would notice. Bumping strands them all at
	// once — the whole point of putting the version in the key.
	//
	// This is the sub-1-5b segment-cache split applied to a different axis:
	// the key PREFIX versions the key FORMAT, RunVersion versions the key's
	// INPUTS; here the version tags the LOGIC that produced the value.
	routeVersion = 1
)

// routeCacheKey is the [@contract-v1] key format:
//
//	subtitle:route:v{routeVersion}:{mediaID}:{fileSize}:{mtimeUnix}
//
// Invalidation lives in the key, never in the TTL. size + mtime is the SAME
// pair scanner_service.go:477-478 has used to answer "did this file change?"
// since story 7-2 — deliberately the same, because two subsystems holding two
// different answers to that question is how a library ends up half-stale.
//
// KNOWN PRECISION LIMIT (CR sub-5-4 L2): mtimeUnix is truncated to SECONDS.
// A file rewritten within the same second as its cached stat, at the same
// byte size, produces an identical key and therefore a stale hit. This is
// accepted, not overlooked: the scanner's change detection has the same
// effective granularity, and a same-second same-size re-encode is not a
// scenario a media library produces.
//
// version is a parameter rather than a direct read of routeVersion so a test
// can prove a bump actually strands the old entries.
func routeCacheKey(version int, mediaID string, fileSize, mtimeUnix int64) string {
	return fmt.Sprintf("subtitle:route:v%d:%s:%d:%d", version, mediaID, fileSize, mtimeUnix)
}

// isKnownRoute rejects a stored value this build no longer understands.
//
// A verdict written by a future/foreign build is not a verdict — treating an
// unrecognised string as a route would let it flow into the quote as an empty
// RoutePrediction, which prices at $0 and is invisible in the summary counts.
// Falling through to a probe is always safe.
func isKnownRoute(route RoutePrediction) bool {
	switch route {
	case RouteExtract, RouteASR, RouteSkipped:
		return true
	default:
		return false
	}
}

// ─── File identity ─────────────────────────────────────────────────────────

// fileIdentityFunc reports the size and modification time the cache key is
// built from. Injected so tests own invalidation outright instead of touching
// a real disk; production always uses osFileIdentity.
type fileIdentityFunc func(path string) (fileSize, mtimeUnix int64, err error)

// osFileIdentity is one stat syscall — microseconds against the hundreds of
// milliseconds an ffprobe costs, which is what makes the lookup worth doing at
// all.
func osFileIdentity(path string) (int64, int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	return info.Size(), info.ModTime().Unix(), nil
}

// ─── Repository adapter ────────────────────────────────────────────────────

// routeCacheRepository adapts repository.CacheRepositoryInterface to
// RouteCache. It lives here rather than in the repository package because the
// value-not-entry shape is this sweep's convenience, not a storage concern.
type routeCacheRepository struct {
	cache repository.CacheRepositoryInterface
}

// NewRouteCacheRepository wires the route cache onto the shared `cache_entries`
// table. No migration is involved — the table has existed since AD #4.
func NewRouteCacheRepository(cache repository.CacheRepositoryInterface) RouteCache {
	return &routeCacheRepository{cache: cache}
}

func (r *routeCacheRepository) GetMany(ctx context.Context, keys []string) (map[string]string, error) {
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

func (r *routeCacheRepository) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.cache.Set(ctx, key, value, routeCacheType, ttl)
}

// compile-time proof that the shipped repository satisfies the narrow port.
var _ RouteCache = (*routeCacheRepository)(nil)
