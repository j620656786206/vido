package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/sse"
)

// fakeProgressSink is a minimal progressSink: a configurable ClientCount plus captured Broadcast
// events, so the gate and payload can be asserted directly with no real hub/HTTP (AC6). An optional
// signal channel gets a non-blocking send on each Broadcast so the ticker-driven test can wait on a
// broadcast instead of sleeping a fixed duration.
type fakeProgressSink struct {
	mu          sync.Mutex
	clientCount int
	events      []sse.Event
	signal      chan struct{}
}

func (f *fakeProgressSink) Broadcast(event sse.Event) {
	f.mu.Lock()
	f.events = append(f.events, event)
	f.mu.Unlock()
	if f.signal != nil {
		select {
		case f.signal <- struct{}{}:
		default:
		}
	}
}

func (f *fakeProgressSink) ClientCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.clientCount
}

func (f *fakeProgressSink) broadcastCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

func (f *fakeProgressSink) lastEvent() (sse.Event, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.events) == 0 {
		return sse.Event{}, false
	}
	return f.events[len(f.events)-1], true
}

// Compile-time check the fake satisfies the narrow sink the broadcaster depends on.
var _ progressSink = (*fakeProgressSink)(nil)

// mockDownloadSvc is a minimal DownloadServiceInterface for the broadcaster: it counts
// GetAllDownloads calls and returns a configurable snapshot/error. The other five methods are unused
// no-ops (the broadcaster only ever calls GetAllDownloads).
type mockDownloadSvc struct {
	mu       sync.Mutex
	calls    int
	torrents []qbittorrent.Torrent
	err      error

	// lastArgs captures the (filter, sort, order) of the most recent call so the AC3 poll contract
	// can be asserted directly — a regression that changes the filter/sort must fail a test.
	lastFilter string
	lastSort   string
	lastOrder  string
}

func (m *mockDownloadSvc) GetAllDownloads(_ context.Context, filter string, sortField string, order string) ([]qbittorrent.Torrent, error) {
	m.mu.Lock()
	m.calls++
	m.lastFilter, m.lastSort, m.lastOrder = filter, sortField, order
	m.mu.Unlock()
	return m.torrents, m.err
}

func (m *mockDownloadSvc) args() (string, string, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastFilter, m.lastSort, m.lastOrder
}

func (m *mockDownloadSvc) GetDownloadDetails(_ context.Context, _ string) (*qbittorrent.TorrentDetails, error) {
	return nil, nil
}
func (m *mockDownloadSvc) GetDownloadCounts(_ context.Context) (*qbittorrent.DownloadCounts, error) {
	return nil, nil
}
func (m *mockDownloadSvc) PauseDownload(_ context.Context, _ string) error  { return nil }
func (m *mockDownloadSvc) ResumeDownload(_ context.Context, _ string) error { return nil }
func (m *mockDownloadSvc) RemoveDownload(_ context.Context, _ string, _ bool) error {
	return nil
}

func (m *mockDownloadSvc) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

// Compile-time check the mock satisfies the interface the broadcaster depends on.
var _ DownloadServiceInterface = (*mockDownloadSvc)(nil)

func TestDownloadProgressBroadcaster_DefaultInterval(t *testing.T) {
	// Locks decision #2 (~2s cadence) so a silent change to the default is caught.
	b := NewDownloadProgressBroadcaster(&mockDownloadSvc{}, &fakeProgressSink{})
	assert.Equal(t, 2*time.Second, b.interval)
}

func TestDownloadProgressBroadcaster_Tick(t *testing.T) {
	ctx := context.Background()

	t.Run("clients connected → polls once and broadcasts one download_progress event with the snapshot (AC7a)", func(t *testing.T) {
		snapshot := []qbittorrent.Torrent{
			{Hash: "abc123", Name: "Movie.2024.mkv", Progress: 0.5, Status: qbittorrent.StatusDownloading},
		}
		svc := &mockDownloadSvc{torrents: snapshot}
		sink := &fakeProgressSink{clientCount: 2}
		b := NewDownloadProgressBroadcaster(svc, sink)

		b.tick(ctx)

		assert.Equal(t, 1, svc.callCount(), "one tick must poll GetAllDownloads exactly once")
		// AC3 poll contract: filter=all (every status), sort=added_on/desc (mirrors GET /downloads
		// defaults). Locked so a regression changing the filter/sort fails here.
		gotFilter, gotSort, gotOrder := svc.args()
		assert.Equal(t, "all", gotFilter)
		assert.Equal(t, "added_on", gotSort)
		assert.Equal(t, "desc", gotOrder)
		require.Equal(t, 1, sink.broadcastCount(), "one tick must broadcast exactly once")

		ev, ok := sink.lastEvent()
		require.True(t, ok)
		assert.Equal(t, sse.EventDownloadProgress, ev.Type)
		assert.NotEmpty(t, ev.ID, "event must carry a UUID id")

		got, ok := ev.Data.([]qbittorrent.Torrent)
		require.True(t, ok, "Data must be the []qbittorrent.Torrent snapshot (AC2 [@contract-v1] shape)")
		assert.Equal(t, snapshot, got)
	})

	t.Run("no clients → NO poll, NO broadcast (the gate, AC7b / Rule 14)", func(t *testing.T) {
		svc := &mockDownloadSvc{torrents: []qbittorrent.Torrent{{Hash: "x"}}}
		sink := &fakeProgressSink{clientCount: 0}
		b := NewDownloadProgressBroadcaster(svc, sink)

		b.tick(ctx)

		assert.Equal(t, 0, svc.callCount(), "GetAllDownloads must NOT be called when ClientCount()==0")
		assert.Equal(t, 0, sink.broadcastCount(), "no broadcast when nobody is watching")
	})

	t.Run("poll error is swallowed, no broadcast, loop-safe (AC7c / Rule 13)", func(t *testing.T) {
		svc := &mockDownloadSvc{err: assert.AnError}
		sink := &fakeProgressSink{clientCount: 1}
		b := NewDownloadProgressBroadcaster(svc, sink)

		assert.NotPanics(t, func() { b.tick(ctx) })
		assert.Equal(t, 1, svc.callCount())
		assert.Equal(t, 0, sink.broadcastCount(), "a failed poll must not broadcast a stale/empty snapshot")
	})

	t.Run("qBittorrent ConnectionError is swallowed quietly (AC4 DEBUG path)", func(t *testing.T) {
		svc := &mockDownloadSvc{err: &qbittorrent.ConnectionError{
			Code:    qbittorrent.ErrCodeNotConfigured,
			Message: "qBittorrent not configured",
		}}
		sink := &fakeProgressSink{clientCount: 1}
		b := NewDownloadProgressBroadcaster(svc, sink)

		assert.NotPanics(t, func() { b.tick(ctx) })
		assert.Equal(t, 1, svc.callCount())
		assert.Equal(t, 0, sink.broadcastCount())
	})

	t.Run("nil snapshot is normalized to an empty array (payload is never null)", func(t *testing.T) {
		svc := &mockDownloadSvc{torrents: nil} // success, but no torrents
		sink := &fakeProgressSink{clientCount: 1}
		b := NewDownloadProgressBroadcaster(svc, sink)

		b.tick(ctx)

		require.Equal(t, 1, sink.broadcastCount())
		ev, _ := sink.lastEvent()
		got, ok := ev.Data.([]qbittorrent.Torrent)
		require.True(t, ok, "Data must be a []qbittorrent.Torrent even when empty")
		assert.NotNil(t, got, "nil snapshot must be normalized to a non-nil empty slice")
		assert.Empty(t, got)
	})
}

func TestDownloadProgressBroadcaster_Run(t *testing.T) {
	t.Run("ticker drives the gated poll+broadcast (exercises case <-ticker.C)", func(t *testing.T) {
		sig := make(chan struct{}, 8)
		svc := &mockDownloadSvc{torrents: []qbittorrent.Torrent{{Hash: "abc"}}}
		sink := &fakeProgressSink{clientCount: 1, signal: sig}
		b := NewDownloadProgressBroadcaster(svc, sink)

		done := make(chan struct{})
		go func() {
			b.run(context.Background(), 5*time.Millisecond)
			close(done)
		}()

		// Wait on the first ticker-driven broadcast (signal, not a fixed sleep) — deleting the
		// `case <-ticker.C: b.tick(ctx)` path would hang this test.
		select {
		case <-sig:
		case <-time.After(2 * time.Second):
			t.Fatal("ticker never drove a broadcast")
		}
		assert.GreaterOrEqual(t, sink.broadcastCount(), 1)
		assert.GreaterOrEqual(t, svc.callCount(), 1)

		// Join the run goroutine so it cannot outlive the test (no cross-test goroutine bleed).
		b.Stop()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("run did not return after Stop()")
		}
	})
}

func TestDownloadProgressBroadcaster_WarnThrottle(t *testing.T) {
	ctx := context.Background()

	// A persistent non-connection error must WARN once, then throttle repeats to DEBUG; a successful
	// poll clears the throttle so the next distinct failure WARNs again. The log LEVEL is not directly
	// assertable, so we lock the state machine that drives it (b.lastPollErr).
	svc := &mockDownloadSvc{err: assert.AnError}
	sink := &fakeProgressSink{clientCount: 1}
	b := NewDownloadProgressBroadcaster(svc, sink)

	b.tick(ctx) // first failure → WARN, lastPollErr set
	assert.Equal(t, assert.AnError.Error(), b.lastPollErr)

	b.tick(ctx) // same error → DEBUG (throttled); state unchanged
	assert.Equal(t, assert.AnError.Error(), b.lastPollErr)
	assert.Equal(t, 0, sink.broadcastCount(), "errors never broadcast")

	svc.mu.Lock()
	svc.err = nil
	svc.torrents = []qbittorrent.Torrent{{Hash: "ok"}}
	svc.mu.Unlock()

	b.tick(ctx) // success → throttle cleared, one broadcast
	assert.Equal(t, "", b.lastPollErr, "a successful poll must reset the WARN throttle")
	assert.Equal(t, 1, sink.broadcastCount())
}

func TestDownloadProgressBroadcaster_Stop(t *testing.T) {
	t.Run("Stop is idempotent (never started, and called twice)", func(t *testing.T) {
		b := NewDownloadProgressBroadcaster(&mockDownloadSvc{}, &fakeProgressSink{})
		assert.NotPanics(t, func() {
			b.Stop()
			b.Stop()
		})
	})

	t.Run("run returns promptly on Stop()", func(t *testing.T) {
		// A 10-minute interval can never tick within the test window, so ONLY Stop() can end run.
		b := NewDownloadProgressBroadcaster(&mockDownloadSvc{}, &fakeProgressSink{clientCount: 1})

		done := make(chan struct{})
		go func() {
			b.run(context.Background(), 10*time.Minute)
			close(done)
		}()

		b.Stop()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("run did not return promptly after Stop()")
		}
	})

	t.Run("run returns promptly on context cancellation", func(t *testing.T) {
		b := NewDownloadProgressBroadcaster(&mockDownloadSvc{}, &fakeProgressSink{clientCount: 1})

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			b.run(ctx, 10*time.Minute)
			close(done)
		}()

		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("run did not return on context cancellation")
		}
	})

	t.Run("Start delegates to run and returns after Stop() (public entrypoint)", func(t *testing.T) {
		// clientCount:0 keeps the default ~2s interval harmless; Stop() closes stopCh so Start returns
		// immediately regardless of the interval — no 2s wait.
		b := NewDownloadProgressBroadcaster(&mockDownloadSvc{}, &fakeProgressSink{clientCount: 0})

		done := make(chan struct{})
		go func() {
			b.Start(context.Background())
			close(done)
		}()

		b.Stop()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("Start did not return after Stop()")
		}
	})
}
