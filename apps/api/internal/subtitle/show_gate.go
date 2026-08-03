package subtitle

import (
	"context"
	"sync"
	"time"
)

const (
	// showWarmWindow is how long we assume a show's prompt prefix stays live
	// provider-side. Deliberately SHORTER than the 1h cache_control TTL the
	// system blocks carry (AC #4.1): if the two were equal, an item entering at
	// T+59m59s would skip the gate and then miss an entry that expired a second
	// later, and every remaining episode would pay the full prefix again.
	showWarmWindow = 50 * time.Minute

	// showEntryTTL bounds the latch map (Rule 14). One long-lived Pipeline
	// serves an entire library, so entries are pruned once they can no longer
	// affect any decision — an entry older than this is stale by definition,
	// since the warm window has long since closed.
	showEntryTTL = 2 * showWarmWindow
)

// showGate is D10's per-show first-request latch.
//
// A cache entry is only readable once the first response has begun (architecture
// fact 6), so N workers translating the same show would each write the same
// prefix and pay the 2× write premium N times. The gate lets the FIRST request
// for a show run alone and releases the rest when it returns.
//
// This is deliberately an ORCHESTRATOR concern: sub-1-6's worker pool stays a
// generic executor with no show-awareness (AD #5).
type showGate struct {
	mu      sync.Mutex
	entries map[string]*showGateEntry
	now     func() time.Time
}

type showGateEntry struct {
	// inFlight is non-nil while a leader holds the gate; it is closed on
	// release, which is what unparks the waiters.
	inFlight chan struct{}
	// warmedAt is when the last leader returned. Zero = never warmed.
	warmedAt time.Time
	// touchedAt keeps a currently-leading entry from being pruned out from
	// under its own waiters.
	touchedAt time.Time
}

func newShowGate(now func() time.Time) *showGate {
	if now == nil {
		now = time.Now
	}
	return &showGate{entries: map[string]*showGateEntry{}, now: now}
}

// enter returns once this caller may issue the show's next LLM request. The
// returned release marks the show warm and unparks any waiters; it is
// idempotent and MUST be called on every path.
//
// An empty key bypasses the gate entirely — that is a movie, and nothing shares
// its prompt prefix.
func (g *showGate) enter(ctx context.Context, key string) (func(), error) {
	if key == "" {
		return func() {}, nil
	}

	for {
		g.mu.Lock()
		now := g.now()
		g.pruneLocked(now)

		entry, ok := g.entries[key]
		switch {
		case !ok:
			entry = &showGateEntry{}
			g.entries[key] = entry

		case entry.inFlight != nil:
			// Someone is warming this prefix. Park until they return, then
			// re-evaluate: by that point the entry is normally warm, and if it
			// went stale in the meantime we legitimately become the next leader.
			inFlight := entry.inFlight
			g.mu.Unlock()
			select {
			case <-inFlight:
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}

		case !entry.warmedAt.IsZero() && now.Sub(entry.warmedAt) < showWarmWindow:
			// The prefix is live: no serialization needed.
			entry.touchedAt = now
			g.mu.Unlock()
			return func() {}, nil
		}

		// Become the leader.
		entry.inFlight = make(chan struct{})
		entry.touchedAt = now
		inFlight := entry.inFlight
		g.mu.Unlock()

		var once sync.Once
		return func() {
			once.Do(func() {
				g.mu.Lock()
				entry.warmedAt = g.now()
				entry.touchedAt = entry.warmedAt
				entry.inFlight = nil
				g.mu.Unlock()
				close(inFlight)
			})
		}, nil
	}
}

// pruneLocked drops entries that can no longer influence a decision. A leading
// entry is never pruned — its waiters hold a reference to the very channel that
// would be dropped.
func (g *showGate) pruneLocked(now time.Time) {
	for key, entry := range g.entries {
		if entry.inFlight != nil {
			continue
		}
		if now.Sub(entry.touchedAt) >= showEntryTTL {
			delete(g.entries, key)
		}
	}
}

// size reports the number of tracked shows (Rule 14 bound assertions).
func (g *showGate) size() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.entries)
}

// enterShowGate wraps the gate for the translate loop: only the item's FIRST
// LLM request is serialized, and only when the item belongs to a show.
func (p *Pipeline) enterShowGate(ctx context.Context, firstRequest bool) (func(), error) {
	if !firstRequest || p.gate == nil {
		return func() {}, nil
	}
	scope := processScopeFrom(ctx)
	if scope == nil {
		return func() {}, nil
	}
	return p.gate.enter(ctx, scope.showKey)
}
