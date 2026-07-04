package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/plugins"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/sse"
)

// --- fakes -------------------------------------------------------------------

// fakePollerRepo backs ListActive/UpdateStatus with an in-memory row map.
type fakePollerRepo struct {
	mu            sync.Mutex
	rows          []models.Request
	listErr       error
	listCalls     int
	statusUpdates []statusUpdate
	updateErr     error
}

type statusUpdate struct {
	id     string
	status string
	errMsg string
}

func (f *fakePollerRepo) Create(ctx context.Context, request *models.Request) error { return nil }
func (f *fakePollerRepo) List(ctx context.Context) ([]models.Request, error)        { return nil, nil }
func (f *fakePollerRepo) FindActiveByTMDbID(ctx context.Context, tmdbID int64, mediaType string) (*models.Request, error) {
	return nil, nil
}
func (f *fakePollerRepo) UpdateFulfilment(ctx context.Context, id string, status string, fulfilmentSource, externalID, errorMessage models.NullString) (time.Time, error) {
	return time.Now(), nil
}
func (f *fakePollerRepo) ListActive(ctx context.Context) ([]models.Request, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls++
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]models.Request, 0, len(f.rows))
	for _, row := range f.rows {
		switch row.Status {
		case models.RequestStatusPending, models.RequestStatusSearching, models.RequestStatusDownloading:
			out = append(out, row)
		}
	}
	return out, nil
}
func (f *fakePollerRepo) UpdateStatus(ctx context.Context, id string, status string, errMsg string) (time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return time.Time{}, f.updateErr
	}
	f.statusUpdates = append(f.statusUpdates, statusUpdate{id, status, errMsg})
	for i := range f.rows {
		if f.rows[i].ID == id {
			f.rows[i].Status = status
			if errMsg == "" {
				f.rows[i].ErrorMessage = models.NullString{}
			} else {
				f.rows[i].ErrorMessage = models.NewNullString(errMsg)
			}
		}
	}
	return time.Now(), nil
}

func (f *fakePollerRepo) updates() []statusUpdate {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]statusUpdate, len(f.statusUpdates))
	copy(out, f.statusUpdates)
	return out
}

// fakeOwnership implements AvailabilityServiceInterface.
type fakeOwnership struct {
	mu    sync.Mutex
	owned map[int64]bool
	err   error
	calls int
}

func (f *fakeOwnership) CheckOwned(ctx context.Context, tmdbIDs []int64) ([]int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	var out []int64
	for _, id := range tmdbIDs {
		if f.owned[id] {
			out = append(out, id)
		}
	}
	return out, nil
}

// fakeQueueSource implements requestQueueSource.
type fakeQueueSource struct {
	mu      sync.Mutex
	plugins map[string]*fakeDVRPlugin // nil entry = registered but GetClient errors
	health  map[string]string
	calls   int
}

func (f *fakeQueueSource) RegisteredPlugins() []string {
	names := make([]string, 0, len(f.plugins))
	for name := range f.plugins {
		names = append(names, name)
	}
	return names
}
func (f *fakeQueueSource) Health(name string) plugins.PluginHealth {
	status, ok := f.health[name]
	if !ok {
		status = plugins.HealthStatusHealthy
	}
	return plugins.PluginHealth{Status: status}
}
func (f *fakeQueueSource) GetClient(ctx context.Context, name string) (plugins.DVRPlugin, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	p := f.plugins[name]
	if p == nil {
		return nil, &plugins.PluginError{Code: plugins.ErrCodeNotConfigured, Message: "not configured"}
	}
	return p, nil
}

// fakeTorrents implements torrentSource.
type fakeTorrents struct {
	mu       sync.Mutex
	torrents []qbittorrent.Torrent
	err      error
	calls    int
}

func (f *fakeTorrents) GetAllDownloads(ctx context.Context, filter string, sortField string, order string) ([]qbittorrent.Torrent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.torrents, nil
}

// fakeScanner implements scanTrigger.
type fakeScanner struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (f *fakeScanner) StartScan(ctx context.Context) (*ScanResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &ScanResult{}, nil
}

func (f *fakeScanner) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// fakeRequestSink implements progressSink.
type fakeRequestSink struct {
	mu      sync.Mutex
	clients int
	events  []sse.Event
}

func (f *fakeRequestSink) Broadcast(event sse.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}
func (f *fakeRequestSink) ClientCount() int { return f.clients }
func (f *fakeRequestSink) all() []sse.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]sse.Event, len(f.events))
	copy(out, f.events)
	return out
}

// --- harness -----------------------------------------------------------------

type pollerTestEnv struct {
	poller     *RequestStatusPoller
	repo       *fakePollerRepo
	owned      *fakeOwnership
	queues     *fakeQueueSource
	torrents   *fakeTorrents
	scanner    *fakeScanner
	sink       *fakeRequestSink
	fulfilment *stubFulfilment
	clock      *time.Time
}

func newPollerTestEnv(t *testing.T) *pollerTestEnv {
	t.Helper()
	repo := &fakePollerRepo{}
	owned := &fakeOwnership{owned: map[int64]bool{}}
	queues := &fakeQueueSource{plugins: map[string]*fakeDVRPlugin{}, health: map[string]string{}}
	torrents := &fakeTorrents{}
	scanner := &fakeScanner{}
	sink := &fakeRequestSink{clients: 1}
	fulfilment := &stubFulfilment{}

	poller := NewRequestStatusPoller(repo, owned, queues, torrents, scanner, fulfilment, sink)
	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	clock := &now
	poller.now = func() time.Time { return *clock }

	return &pollerTestEnv{poller: poller, repo: repo, owned: owned, queues: queues,
		torrents: torrents, scanner: scanner, sink: sink, fulfilment: fulfilment, clock: clock}
}

func activeRow(id string, tmdbID int64, mediaType, status, externalID string) models.Request {
	row := models.Request{
		ID: id, TMDbID: tmdbID, MediaType: mediaType, Title: "t-" + id, Status: status,
		RequestedAt: time.Date(2026, 7, 5, 9, 0, 0, 0, time.UTC),
	}
	if externalID != "" {
		row.ExternalID = models.NewNullString(externalID)
		row.FulfilmentSource = models.NewNullString(models.RequestFulfilmentSourceArr)
	}
	return row
}

func queueItems(items ...plugins.QueueItem) *fakeDVRPlugin {
	return &fakeDVRPlugin{queue: items}
}

// --- tests ---------------------------------------------------------------------

func TestPoller_IdleGate_NoActiveRowsMakesZeroSourceCalls(t *testing.T) {
	env := newPollerTestEnv(t)

	env.poller.tick(context.Background())

	assert.Equal(t, 1, env.repo.listCalls, "ListActive is the idle gate")
	assert.Equal(t, 0, env.owned.calls, "no ownership call on an idle NAS")
	assert.Equal(t, 0, env.queues.calls, "no *arr call on an idle NAS")
	assert.Equal(t, 0, env.torrents.calls, "no qBT call on an idle NAS")
	assert.Empty(t, env.sink.all(), "nothing to broadcast")
}

func TestPoller_Rule1_OwnedBecomesCompleted_HookFiresExactlyOnce(t *testing.T) {
	env := newPollerTestEnv(t)
	env.repo.rows = []models.Request{activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusDownloading, "42")}
	env.owned.owned[550] = true

	var hookCalls []models.Request
	env.poller.OnRequestCompleted = func(ctx context.Context, req models.Request) {
		hookCalls = append(hookCalls, req)
	}

	env.poller.tick(context.Background())

	updates := env.repo.updates()
	require.Len(t, updates, 1)
	assert.Equal(t, statusUpdate{"r1", models.RequestStatusCompleted, ""}, updates[0])
	require.Len(t, hookCalls, 1, "13-5 seam fires on the transition edge")
	assert.Equal(t, models.RequestStatusCompleted, hookCalls[0].Status)

	// Next tick: the row is terminal → not in ListActive → no re-fire.
	env.poller.tick(context.Background())
	assert.Len(t, hookCalls, 1, "re-ticks must not re-fire the completed hook")
}

func TestPoller_Rule2_QueueMatchBecomesDownloading_WithQueueProgress(t *testing.T) {
	env := newPollerTestEnv(t)
	env.repo.rows = []models.Request{activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusSearching, "42")}
	env.queues.plugins["radarr"] = queueItems(plugins.QueueItem{
		ExternalID: 42, Title: "Fight Club", Status: "downloading", Size: 100, SizeLeft: 25, DownloadID: "HASH42",
	})

	env.poller.tick(context.Background())

	updates := env.repo.updates()
	require.Len(t, updates, 1)
	assert.Equal(t, statusUpdate{"r1", models.RequestStatusDownloading, ""}, updates[0])

	events := env.sink.all()
	require.Len(t, events, 1)
	items := events[0].Data.([]requestProgressItem)
	require.Len(t, items, 1)
	require.NotNil(t, items[0].Progress)
	assert.InDelta(t, 0.75, *items[0].Progress, 0.001, "(size-sizeleft)/size")
}

func TestPoller_Rule2_QBTHashRefinesProgress(t *testing.T) {
	env := newPollerTestEnv(t)
	env.repo.rows = []models.Request{activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusDownloading, "42")}
	env.queues.plugins["radarr"] = queueItems(plugins.QueueItem{
		ExternalID: 42, Status: "downloading", Size: 100, SizeLeft: 90, DownloadID: "ABCDEF",
	})
	env.torrents.torrents = []qbittorrent.Torrent{{Hash: "abcdef", Progress: 0.42, Status: qbittorrent.StatusDownloading}}

	env.poller.tick(context.Background())

	events := env.sink.all()
	require.Len(t, events, 1)
	items := events[0].Data.([]requestProgressItem)
	require.NotNil(t, items[0].Progress)
	assert.InDelta(t, 0.42, *items[0].Progress, 0.001, "joined qBT progress wins over the queue estimate")
	assert.Empty(t, env.repo.updates(), "already downloading — no redundant persist")
}

func TestPoller_Rule2_ErroredQueueItemBecomesFailed(t *testing.T) {
	env := newPollerTestEnv(t)
	env.repo.rows = []models.Request{activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusDownloading, "42")}
	env.queues.plugins["radarr"] = queueItems(plugins.QueueItem{
		ExternalID: 42, Status: "failed", Size: 100, SizeLeft: 50, DownloadID: "X",
	})

	env.poller.tick(context.Background())

	updates := env.repo.updates()
	require.Len(t, updates, 1)
	assert.Equal(t, "r1", updates[0].id)
	assert.Equal(t, models.RequestStatusFailed, updates[0].status)
	assert.NotEmpty(t, updates[0].errMsg, "failed transition carries a zh-TW reason")
}

func TestPoller_Rule2_QBTErrorStateBecomesFailed(t *testing.T) {
	env := newPollerTestEnv(t)
	env.repo.rows = []models.Request{activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusDownloading, "42")}
	env.queues.plugins["radarr"] = queueItems(plugins.QueueItem{
		ExternalID: 42, Status: "downloading", Size: 100, SizeLeft: 50, DownloadID: "DEAD",
	})
	env.torrents.torrents = []qbittorrent.Torrent{{Hash: "dead", Progress: 0.5, Status: qbittorrent.StatusError}}

	env.poller.tick(context.Background())

	updates := env.repo.updates()
	require.Len(t, updates, 1)
	assert.Equal(t, models.RequestStatusFailed, updates[0].status)
}

func TestPoller_Rule3_VanishedQueueHoldsDownloading_TriggersOneScan(t *testing.T) {
	env := newPollerTestEnv(t)
	// Burst: two rows enter the import window on the same tick → ONE scan.
	env.repo.rows = []models.Request{
		activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusDownloading, "42"),
		activeRow("r2", 551, models.RequestMediaTypeMovie, models.RequestStatusDownloading, "43"),
	}
	env.queues.plugins["radarr"] = queueItems() // queue empty — downloads vanished

	env.poller.tick(context.Background())

	assert.Empty(t, env.repo.updates(), "import window HOLDS downloading — no regression to searching")
	assert.Eventually(t, func() bool { return env.scanner.count() == 1 }, time.Second, 5*time.Millisecond,
		"burst of completions shares ONE debounced scan")

	// Re-tick inside the debounce window: rows still in window, no second scan.
	env.poller.tick(context.Background())
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 1, env.scanner.count(), "no re-trigger while the window rows persist")

	// A third row enters the window AFTER the debounce interval elapses → one more scan.
	*env.clock = env.clock.Add(3 * time.Minute)
	env.repo.mu.Lock()
	env.repo.rows = append(env.repo.rows, activeRow("r3", 552, models.RequestMediaTypeMovie, models.RequestStatusDownloading, "44"))
	env.repo.mu.Unlock()
	env.poller.tick(context.Background())
	assert.Eventually(t, func() bool { return env.scanner.count() == 2 }, time.Second, 5*time.Millisecond)
}

func TestPoller_Rule4_ExternalSetNoQueueNeverDownloadedIsSearching(t *testing.T) {
	env := newPollerTestEnv(t)
	env.repo.rows = []models.Request{activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusPending, "42")}
	env.queues.plugins["radarr"] = queueItems()

	env.poller.tick(context.Background())

	updates := env.repo.updates()
	require.Len(t, updates, 1)
	assert.Equal(t, statusUpdate{"r1", models.RequestStatusSearching, ""}, updates[0])
	assert.Equal(t, 0, env.scanner.count(), "searching is not the import window")
}

func TestPoller_Rule5_PendingRetriesFulfilment(t *testing.T) {
	env := newPollerTestEnv(t)
	env.repo.rows = []models.Request{activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusPending, "")}

	env.poller.tick(context.Background())

	assert.Equal(t, 1, env.fulfilment.calls, "stranded pending rows retry via the 13-4a service")
	assert.Equal(t, "r1", env.fulfilment.lastReq.ID)

	env.poller.tick(context.Background())
	assert.Equal(t, 2, env.fulfilment.calls, "at most once per tick per row — and once each tick")
}

func TestPoller_Rule5_NilFulfilmentIsNoOp(t *testing.T) {
	env := newPollerTestEnv(t)
	env.poller.fulfilment = nil
	env.repo.rows = []models.Request{activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusPending, "")}

	assert.NotPanics(t, func() { env.poller.tick(context.Background()) })
}

func TestPoller_BroadcastGate_ZeroClientsStillReconciles(t *testing.T) {
	env := newPollerTestEnv(t)
	env.sink.clients = 0
	env.repo.rows = []models.Request{activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusDownloading, "42")}
	env.owned.owned[550] = true

	env.poller.tick(context.Background())

	assert.Empty(t, env.sink.all(), "ClientCount 0 → no broadcast")
	updates := env.repo.updates()
	require.Len(t, updates, 1, "the reconcile is SSE-independent — DB truth still advances")
	assert.Equal(t, models.RequestStatusCompleted, updates[0].status)
}

func TestPoller_SnapshotIncludesTransitionedRows(t *testing.T) {
	env := newPollerTestEnv(t)
	env.repo.rows = []models.Request{
		activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusDownloading, "42"),
		activeRow("r2", 551, models.RequestMediaTypeMovie, models.RequestStatusSearching, "43"),
	}
	env.owned.owned[550] = true // r1 completes this tick
	env.queues.plugins["radarr"] = queueItems(plugins.QueueItem{
		ExternalID: 43, Status: "downloading", Size: 10, SizeLeft: 5, DownloadID: "H2",
	})

	env.poller.tick(context.Background())

	events := env.sink.all()
	require.Len(t, events, 1)
	assert.Equal(t, sse.EventRequestProgress, events[0].Type)
	items := events[0].Data.([]requestProgressItem)
	require.Len(t, items, 2, "the final completed frame rides the same snapshot")

	byID := map[string]requestProgressItem{}
	for _, it := range items {
		byID[it.ID] = it
	}
	assert.Equal(t, models.RequestStatusCompleted, byID["r1"].Status)
	assert.Nil(t, byID["r1"].Progress, "progress only when meaningful")
	assert.Equal(t, models.RequestStatusDownloading, byID["r2"].Status)
	require.NotNil(t, byID["r2"].Progress)
}

func TestPoller_FailSoft_QueueSourceErrorHoldsRows(t *testing.T) {
	env := newPollerTestEnv(t)
	env.repo.rows = []models.Request{activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusDownloading, "42")}
	env.queues.plugins["radarr"] = &fakeDVRPlugin{queueErr: errors.New("radarr 503")}

	env.poller.tick(context.Background())

	assert.Empty(t, env.repo.updates(), "no queue evidence → hold, never fail/regress")
	assert.Equal(t, 0, env.scanner.count(), "a dead source must not fake an import window")
	events := env.sink.all()
	require.Len(t, events, 1, "the snapshot still broadcasts held rows")
}

func TestPoller_FailSoft_UnhealthyPluginSkipsQueueCall(t *testing.T) {
	env := newPollerTestEnv(t)
	env.repo.rows = []models.Request{activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusDownloading, "42")}
	env.queues.plugins["radarr"] = queueItems()
	env.queues.health["radarr"] = plugins.HealthStatusUnhealthy

	env.poller.tick(context.Background())

	assert.Equal(t, 0, env.queues.calls, "unhealthy plugin → no GetClient/GetQueue call")
	assert.Empty(t, env.repo.updates())
}

func TestPoller_FailSoft_OwnershipErrorSkipsCompletion(t *testing.T) {
	env := newPollerTestEnv(t)
	env.repo.rows = []models.Request{activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusDownloading, "42")}
	env.owned.err = errors.New("db locked")
	env.owned.owned[550] = true // would complete, but the source is down
	env.queues.plugins["radarr"] = queueItems(plugins.QueueItem{
		ExternalID: 42, Status: "downloading", Size: 100, SizeLeft: 50, DownloadID: "H",
	})

	hookFired := false
	env.poller.OnRequestCompleted = func(ctx context.Context, req models.Request) { hookFired = true }

	env.poller.tick(context.Background())

	assert.False(t, hookFired, "completion detection skipped when CheckOwned errors")
	assert.Empty(t, env.repo.updates(), "row already downloading — held")
}

func TestPoller_FailSoft_QBTDownStillUsesQueueProgress(t *testing.T) {
	env := newPollerTestEnv(t)
	env.repo.rows = []models.Request{activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusDownloading, "42")}
	env.queues.plugins["radarr"] = queueItems(plugins.QueueItem{
		ExternalID: 42, Status: "downloading", Size: 100, SizeLeft: 30, DownloadID: "H",
	})
	env.torrents.err = &qbittorrent.ConnectionError{Code: qbittorrent.ErrCodeConnectionFailed, Message: "down"}

	env.poller.tick(context.Background())

	events := env.sink.all()
	require.Len(t, events, 1)
	items := events[0].Data.([]requestProgressItem)
	require.NotNil(t, items[0].Progress)
	assert.InDelta(t, 0.7, *items[0].Progress, 0.001, "queue-based % works without qBT")
}

func TestPoller_TVRowsJoinSonarrQueue(t *testing.T) {
	env := newPollerTestEnv(t)
	env.repo.rows = []models.Request{activeRow("r1", 1399, models.RequestMediaTypeTV, models.RequestStatusSearching, "7")}
	env.queues.plugins["radarr"] = queueItems(plugins.QueueItem{ExternalID: 7, Status: "downloading", Size: 10, SizeLeft: 5})
	env.queues.plugins["sonarr"] = queueItems(plugins.QueueItem{ExternalID: 7, Status: "downloading", Size: 10, SizeLeft: 2, DownloadID: "TV"})

	env.poller.tick(context.Background())

	// The tv row must join the SONARR queue (media_type routing), not radarr's
	// coincidentally-matching ExternalID.
	events := env.sink.all()
	items := events[0].Data.([]requestProgressItem)
	require.NotNil(t, items[0].Progress)
	assert.InDelta(t, 0.8, *items[0].Progress, 0.001)
}

func TestPoller_LifecycleStartStop(t *testing.T) {
	env := newPollerTestEnv(t)
	env.poller.interval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		env.poller.Start(ctx)
		close(done)
	}()

	assert.Eventually(t, func() bool {
		env.repo.mu.Lock()
		defer env.repo.mu.Unlock()
		return env.repo.listCalls >= 2
	}, time.Second, 5*time.Millisecond, "ticker drives ticks")

	env.poller.Stop()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Start did not return after Stop")
	}
	env.poller.Stop() // idempotent
}

func TestMapQueueItemStatus(t *testing.T) {
	// The qBT-status→request-status mapping function (AC #2 table).
	cases := []struct {
		qbt  qbittorrent.TorrentStatus
		want queueDerivedState
	}{
		{qbittorrent.StatusDownloading, queueStateDownloading},
		{qbittorrent.StatusQueued, queueStateDownloading},
		{qbittorrent.StatusChecking, queueStateDownloading},
		{qbittorrent.StatusStalled, queueStateDownloading},
		{qbittorrent.StatusCompleted, queueStateImportWindow},
		{qbittorrent.StatusSeeding, queueStateImportWindow},
		{qbittorrent.StatusError, queueStateFailed},
	}
	for _, c := range cases {
		t.Run(string(c.qbt), func(t *testing.T) {
			assert.Equal(t, c.want, mapTorrentToQueueState(c.qbt))
		})
	}
}

func TestPoller_QBTCompletedEntersImportWindow(t *testing.T) {
	// qBT says the torrent finished while *arr still lists it → import window:
	// hold downloading + scan trigger (AC #2 mapping row → rule 3).
	env := newPollerTestEnv(t)
	env.repo.rows = []models.Request{activeRow("r1", 550, models.RequestMediaTypeMovie, models.RequestStatusDownloading, "42")}
	env.queues.plugins["radarr"] = queueItems(plugins.QueueItem{
		ExternalID: 42, Status: "downloading", Size: 100, SizeLeft: 0, DownloadID: "FIN",
	})
	env.torrents.torrents = []qbittorrent.Torrent{{Hash: "fin", Progress: 1.0, Status: qbittorrent.StatusCompleted}}

	env.poller.tick(context.Background())

	assert.Empty(t, env.repo.updates(), "hold downloading through the import window")
	assert.Eventually(t, func() bool { return env.scanner.count() == 1 }, time.Second, 5*time.Millisecond)
}

// Compile-time guards: the narrow deps stay satisfied by the real types.
var (
	_ requestQueueSource = (*plugins.Manager)(nil)
	_ torrentSource      = (DownloadServiceInterface)(nil)
	_ scanTrigger        = (*ScannerService)(nil)
)
