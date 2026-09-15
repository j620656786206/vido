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
)

// --- fakes ---------------------------------------------------------------------

type importFakeReader struct {
	plugins.DVRPlugin // only GetImportHistory is called

	mu      sync.Mutex
	records []plugins.ImportHistoryRecord
	err     error
	calls   int
	gate    chan struct{} // when set, a fetch waits for it (or for its context)
}

func (f *importFakeReader) GetImportHistory(ctx context.Context) ([]plugins.ImportHistoryRecord, error) {
	f.mu.Lock()
	f.calls++
	gate, records, err := f.gate, append([]plugins.ImportHistoryRecord(nil), f.records...), f.err
	f.mu.Unlock()
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (f *importFakeReader) set(records []plugins.ImportHistoryRecord, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.records, f.err = records, err
}

func (f *importFakeReader) block() chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gate = make(chan struct{})
	return f.gate
}

func (f *importFakeReader) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

type importFakeClients struct {
	mu             sync.Mutex
	health         map[string]string
	readers        map[string]plugins.DVRPlugin
	getClientCalls int
}

func (f *importFakeClients) Health(name string) plugins.PluginHealth {
	f.mu.Lock()
	defer f.mu.Unlock()
	status := f.health[name]
	if status == "" {
		status = plugins.HealthStatusUnconfigured
	}
	return plugins.PluginHealth{Status: status}
}

func (f *importFakeClients) GetClient(_ context.Context, name string) (plugins.DVRPlugin, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getClientCalls++
	reader, ok := f.readers[name]
	if !ok {
		return nil, errors.New("not configured")
	}
	return reader, nil
}

func (f *importFakeClients) setHealth(name, status string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.health[name] = status
}

func (f *importFakeClients) setReader(name string, reader plugins.DVRPlugin) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.readers[name] = reader
}

type importFakeMovies map[int64]*models.Movie

func (f importFakeMovies) FindWithFileByTMDbID(_ context.Context, id int64) (*models.Movie, error) {
	return f[id], nil
}

type importFakeSeries map[int64]*models.Series

func (f importFakeSeries) FindActiveByTMDbID(_ context.Context, id int64) (*models.Series, error) {
	return f[id], nil
}

type importFakeEpisodes struct {
	bySeries map[string][]models.Episode
	calls    int
}

func (f *importFakeEpisodes) FindBySeriesID(_ context.Context, seriesID string) ([]models.Episode, error) {
	f.calls++
	return f.bySeries[seriesID], nil
}

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

type importEnv struct {
	service  *ImportStatusService
	clients  *importFakeClients
	radarr   *importFakeReader
	sonarr   *importFakeReader
	movies   importFakeMovies
	series   importFakeSeries
	episodes *importFakeEpisodes
	clock    *fakeClock
}

// newImportEnv builds a service whose named plugins are healthy.
func newImportEnv(t *testing.T, healthy ...string) *importEnv {
	t.Helper()
	env := &importEnv{
		clients:  &importFakeClients{health: map[string]string{}, readers: map[string]plugins.DVRPlugin{}},
		radarr:   &importFakeReader{},
		sonarr:   &importFakeReader{},
		movies:   importFakeMovies{},
		series:   importFakeSeries{},
		episodes: &importFakeEpisodes{bySeries: map[string][]models.Episode{}},
		clock:    &fakeClock{t: time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)},
	}
	env.clients.readers["radarr"] = env.radarr
	env.clients.readers["sonarr"] = env.sonarr
	for _, name := range healthy {
		env.clients.health[name] = plugins.HealthStatusHealthy
	}
	env.service = NewImportStatusService(env.clients, env.movies, env.series, env.episodes)
	env.service.now = env.clock.now
	env.service.firstFetchWait = 2 * time.Second
	return env
}

// waitRefreshes blocks until no background history fetch is running.
func (s *ImportStatusService) waitRefreshes() {
	for {
		s.mu.Lock()
		pending := make([]chan struct{}, 0, len(s.inflight))
		for _, done := range s.inflight {
			pending = append(pending, done)
		}
		s.mu.Unlock()
		if len(pending) == 0 {
			return
		}
		for _, done := range pending {
			<-done
		}
	}
}

var t0 = time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)

func event(hash, eventType string, minute int, tmdb int64) plugins.ImportHistoryRecord {
	return plugins.ImportHistoryRecord{DownloadID: hash, EventType: eventType, Date: t0.Add(time.Duration(minute) * time.Minute), TMDbID: tmdb}
}

func episodeImport(hash string, minute int, tmdb int64, season, episode int, path string) plugins.ImportHistoryRecord {
	r := event(hash, plugins.HistoryEventImported, minute, tmdb)
	r.SeasonNumber, r.EpisodeNumber, r.ImportedPath = season, episode, path
	return r
}

func withFile(path string) models.NullString { return models.NewNullString(path) }

// --- what it says --------------------------------------------------------------

func TestImportStatus_NothingConfigured_SaysNothingAndReadsNothing(t *testing.T) {
	env := newImportEnv(t)
	assert.Nil(t, env.service.Resolve(context.Background(), []string{"abc"}))
	assert.Zero(t, env.clients.getClientCalls, "an unconfigured plugin must not cost a key decrypt")
}

func TestImportStatus_MovieVidoHas_InLibrary(t *testing.T) {
	env := newImportEnv(t, "radarr")
	env.radarr.set([]plugins.ImportHistoryRecord{
		event("ABCDEF", plugins.HistoryEventGrabbed, 0, 603),
		event("ABCDEF", plugins.HistoryEventImported, 5, 603),
	}, nil)
	env.movies[603] = &models.Movie{ID: "movie-1", FilePath: withFile("/media/movies/matrix.mkv")}

	// qBittorrent hashes are lower-case; *arr stores them upper-case.
	got := env.service.Resolve(context.Background(), []string{"abcdef"})

	require.Contains(t, got, "abcdef")
	assert.Equal(t, &DownloadImportStatus{
		State: ImportStateInLibrary, Source: "radarr", MediaType: "movie", MediaID: "movie-1",
	}, got["abcdef"])
}

func TestImportStatus_MovieVidoDoesNotHave_AwaitingScan(t *testing.T) {
	env := newImportEnv(t, "radarr")
	env.radarr.set([]plugins.ImportHistoryRecord{event("AAA", plugins.HistoryEventImported, 5, 603)}, nil)

	got := env.service.Resolve(context.Background(), []string{"aaa"})["aaa"]
	require.NotNil(t, got)
	assert.Equal(t, ImportStateAwaitingScan, got.State)
	assert.Empty(t, got.MediaID, "no link to a copy Vido does not have")
}

func TestImportStatus_ImportWithoutTMDbID_SaysNothing(t *testing.T) {
	env := newImportEnv(t, "radarr")
	env.radarr.set([]plugins.ImportHistoryRecord{event("AAA", plugins.HistoryEventImported, 5, 0)}, nil)
	assert.Empty(t, env.service.Resolve(context.Background(), []string{"aaa"}))
}

func TestImportStatus_NotImported_NewestEventDecides(t *testing.T) {
	env := newImportEnv(t, "radarr")
	env.radarr.set([]plugins.ImportHistoryRecord{
		event("WAIT", plugins.HistoryEventGrabbed, 0, 1),
		event("FAIL", plugins.HistoryEventGrabbed, 0, 2),
		event("FAIL", plugins.HistoryEventDownloadFailed, 9, 2),
		event("SKIP", plugins.HistoryEventGrabbed, 0, 3),
		event("SKIP", plugins.HistoryEventDownloadIgnored, 9, 3),
		event("RETRY", plugins.HistoryEventDownloadFailed, 1, 4),
		event("RETRY", plugins.HistoryEventGrabbed, 9, 4), // re-grabbed after the failure
	}, nil)

	got := env.service.Resolve(context.Background(), []string{"wait", "fail", "skip", "retry"})
	assert.Equal(t, ImportStateAwaitingImport, got["wait"].State)
	assert.Equal(t, ImportStateImportFailed, got["fail"].State)
	assert.Equal(t, ImportStateImportIgnored, got["skip"].State)
	assert.Equal(t, ImportStateAwaitingImport, got["retry"].State)
}

func TestImportStatus_UnknownTorrent_IsAbsent(t *testing.T) {
	env := newImportEnv(t, "radarr", "sonarr")
	env.radarr.set([]plugins.ImportHistoryRecord{event("KNOWN", plugins.HistoryEventGrabbed, 0, 603)}, nil)
	got := env.service.Resolve(context.Background(), []string{"known", "a-book-torrent"})
	assert.Contains(t, got, "known")
	assert.NotContains(t, got, "a-book-torrent")
}

func TestImportStatus_ThePluginThatImportedItSpeaks(t *testing.T) {
	env := newImportEnv(t, "radarr", "sonarr")
	env.radarr.set([]plugins.ImportHistoryRecord{event("BOTH", plugins.HistoryEventGrabbed, 0, 1)}, nil)
	env.sonarr.set([]plugins.ImportHistoryRecord{episodeImport("BOTH", 5, 1399, 1, 1, "/data/media/tv/s01e01.mkv")}, nil)

	got := env.service.Resolve(context.Background(), []string{"both"})["both"]
	require.NotNil(t, got)
	assert.Equal(t, "sonarr", got.Source)
}

func TestImportStatus_SeasonPack_CountsEpisodesVidoHas(t *testing.T) {
	env := newImportEnv(t, "sonarr")
	env.sonarr.set([]plugins.ImportHistoryRecord{
		event("PACK", plugins.HistoryEventGrabbed, 0, 1399),
		episodeImport("PACK", 5, 1399, 1, 1, "/data/media/tv/Show/S01E01.mkv"),
		episodeImport("PACK", 5, 1399, 1, 2, "/data/media/tv/Show/S01E02.mkv"),
	}, nil)
	env.series[1399] = &models.Series{ID: "series-1"}
	env.episodes.bySeries["series-1"] = []models.Episode{
		{SeasonNumber: 1, EpisodeNumber: 1, FilePath: withFile("/media/tv/Show/S01E01.mkv")},
		{SeasonNumber: 1, EpisodeNumber: 2}, // row, no file yet
	}

	got := env.service.Resolve(context.Background(), []string{"pack"})["pack"]
	require.NotNil(t, got)
	assert.Equal(t, ImportStateAwaitingScan, got.State)
	assert.Equal(t, "series-1", got.MediaID)
	assert.Equal(t, 2, *got.EpisodesImported)
	assert.Equal(t, 1, *got.EpisodesInLibrary)

	env.episodes.bySeries["series-1"][1].FilePath = withFile("/media/tv/Show/S01E02.mkv")
	got = env.service.Resolve(context.Background(), []string{"pack"})["pack"]
	assert.Equal(t, ImportStateInLibrary, got.State)
	assert.Equal(t, 2, *got.EpisodesInLibrary)
}

func TestImportStatus_MultiEpisodeFile_CountsAsBothEpisodes(t *testing.T) {
	env := newImportEnv(t, "sonarr")
	env.sonarr.set([]plugins.ImportHistoryRecord{
		episodeImport("DOUBLE", 5, 1399, 1, 1, "/data/media/tv/Show/S01E01E02.mkv"),
		episodeImport("DOUBLE", 5, 1399, 1, 2, "/data/media/tv/Show/S01E01E02.mkv"),
	}, nil)
	env.series[1399] = &models.Series{ID: "series-1"}
	// Vido ingests one row for the file.
	env.episodes.bySeries["series-1"] = []models.Episode{
		{SeasonNumber: 1, EpisodeNumber: 1, FilePath: withFile("/media/tv/Show/S01E01E02.mkv")},
	}

	got := env.service.Resolve(context.Background(), []string{"double"})["double"]
	require.NotNil(t, got)
	assert.Equal(t, ImportStateInLibrary, got.State, "one file, both episodes — never a forever 1/2")
	assert.Equal(t, 2, *got.EpisodesInLibrary)
}

func TestImportStatus_DifferentNumberingMatchesByFileName(t *testing.T) {
	env := newImportEnv(t, "sonarr")
	env.sonarr.set([]plugins.ImportHistoryRecord{
		episodeImport("ANIME", 5, 1399, 2, 5, "/data/media/tv/Show/Show - S02E05.mkv"),
	}, nil)
	env.series[1399] = &models.Series{ID: "series-1"}
	// Vido stored the absolute number; the renamed file is the same.
	env.episodes.bySeries["series-1"] = []models.Episode{
		{SeasonNumber: 1, EpisodeNumber: 30, FilePath: withFile("/media/tv/Show/Show - S02E05.mkv")},
	}

	got := env.service.Resolve(context.Background(), []string{"anime"})["anime"]
	require.NotNil(t, got)
	assert.Equal(t, ImportStateInLibrary, got.State)
}

func TestImportStatus_LibraryReadOncePerSeriesPerPage(t *testing.T) {
	env := newImportEnv(t, "sonarr")
	env.sonarr.set([]plugins.ImportHistoryRecord{
		episodeImport("E1", 5, 1399, 1, 1, "/a/S01E01.mkv"),
		episodeImport("E2", 5, 1399, 1, 2, "/a/S01E02.mkv"),
		episodeImport("E3", 5, 1399, 1, 3, "/a/S01E03.mkv"),
	}, nil)
	env.series[1399] = &models.Series{ID: "series-1"}

	env.service.Resolve(context.Background(), []string{"e1", "e2", "e3"})
	assert.Equal(t, 1, env.episodes.calls)
}

// --- when it reads *arr ----------------------------------------------------------

func TestImportStatus_StaleSnapshotServedWhileRefreshing(t *testing.T) {
	env := newImportEnv(t, "radarr")
	env.radarr.set([]plugins.ImportHistoryRecord{event("AAA", plugins.HistoryEventGrabbed, 0, 603)}, nil)
	env.service.Resolve(context.Background(), []string{"aaa"})

	// the import happens; the next refresh is slow
	env.radarr.set([]plugins.ImportHistoryRecord{
		event("AAA", plugins.HistoryEventGrabbed, 0, 603),
		event("AAA", plugins.HistoryEventImported, 5, 603),
	}, nil)
	gate := env.radarr.block()
	env.clock.advance(importHistoryTTL + time.Second)

	start := time.Now()
	got := env.service.Resolve(context.Background(), []string{"aaa"})
	assert.Less(t, time.Since(start), time.Second, "the page must not wait for *arr once a snapshot exists")
	assert.Equal(t, ImportStateAwaitingImport, got["aaa"].State, "last known answer while refreshing")

	close(gate)
	env.service.waitRefreshes()
	got = env.service.Resolve(context.Background(), []string{"aaa"})
	assert.Equal(t, ImportStateAwaitingScan, got["aaa"].State)
	assert.Equal(t, 2, env.radarr.callCount())
}

func TestImportStatus_CancelledRequestDoesNotCancelTheFetch(t *testing.T) {
	env := newImportEnv(t, "radarr")
	env.radarr.set([]plugins.ImportHistoryRecord{event("AAA", plugins.HistoryEventGrabbed, 0, 603)}, nil)
	gate := env.radarr.block()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the user navigated away
	assert.Empty(t, env.service.Resolve(ctx, []string{"aaa"}))

	close(gate)
	env.service.waitRefreshes()
	got := env.service.Resolve(context.Background(), []string{"aaa"})
	assert.Contains(t, got, "aaa", "the shared fetch finished for everyone else")
	assert.Equal(t, 1, env.radarr.callCount())
}

func TestImportStatus_SlowFirstFetchDoesNotHoldThePage(t *testing.T) {
	env := newImportEnv(t, "radarr")
	env.service.firstFetchWait = 50 * time.Millisecond
	env.radarr.set([]plugins.ImportHistoryRecord{event("AAA", plugins.HistoryEventGrabbed, 0, 603)}, nil)
	gate := env.radarr.block()

	start := time.Now()
	assert.Empty(t, env.service.Resolve(context.Background(), []string{"aaa"}))
	assert.Less(t, time.Since(start), time.Second)

	close(gate)
	env.service.waitRefreshes()
	assert.Contains(t, env.service.Resolve(context.Background(), []string{"aaa"}), "aaa")
}

func TestImportStatus_FailuresBackOff(t *testing.T) {
	env := newImportEnv(t, "radarr")
	env.radarr.set(nil, errors.New("dial tcp: i/o timeout"))

	assert.Empty(t, env.service.Resolve(context.Background(), []string{"aaa"}))
	assert.Equal(t, 1, env.radarr.callCount())

	env.service.Resolve(context.Background(), []string{"aaa"})
	assert.Equal(t, 1, env.radarr.callCount(), "no retry inside the first minute")

	env.clock.advance(importBackoffBase + time.Second)
	env.service.Resolve(context.Background(), []string{"aaa"})
	assert.Equal(t, 2, env.radarr.callCount())

	env.clock.advance(importBackoffBase + time.Second) // second failure waits two minutes
	env.service.Resolve(context.Background(), []string{"aaa"})
	assert.Equal(t, 2, env.radarr.callCount())

	env.radarr.set([]plugins.ImportHistoryRecord{event("AAA", plugins.HistoryEventGrabbed, 0, 603)}, nil)
	env.clock.advance(importBackoffBase)
	got := env.service.Resolve(context.Background(), []string{"aaa"})
	assert.Equal(t, 3, env.radarr.callCount())
	assert.Contains(t, got, "aaa", "recovered")
}

func TestImportStatus_UnhealthyPluginServesItsSnapshotWithoutCallingIt(t *testing.T) {
	env := newImportEnv(t, "radarr")
	env.radarr.set([]plugins.ImportHistoryRecord{event("AAA", plugins.HistoryEventImported, 5, 603)}, nil)
	env.service.Resolve(context.Background(), []string{"aaa"})

	env.clients.setHealth("radarr", plugins.HealthStatusUnhealthy)
	env.clock.advance(importHistoryTTL + time.Second)
	got := env.service.Resolve(context.Background(), []string{"aaa"})
	assert.Contains(t, got, "aaa", "imports never un-happen")
	assert.Equal(t, 1, env.radarr.callCount(), "an unreachable plugin is not called on the request path")

	env.clock.advance(importStaleMax)
	assert.Empty(t, env.service.Resolve(context.Background(), []string{"aaa"}), "too old to vouch for")
}

func TestImportStatus_SwitchedOffPluginForgetsItsSnapshot(t *testing.T) {
	env := newImportEnv(t, "radarr")
	env.radarr.set([]plugins.ImportHistoryRecord{event("AAA", plugins.HistoryEventImported, 5, 603)}, nil)
	env.service.Resolve(context.Background(), []string{"aaa"})

	env.clients.setHealth("radarr", plugins.HealthStatusUnconfigured)
	assert.Empty(t, env.service.Resolve(context.Background(), []string{"aaa"}))

	env.clients.setHealth("radarr", plugins.HealthStatusHealthy)
	env.service.Resolve(context.Background(), []string{"aaa"})
	assert.Equal(t, 2, env.radarr.callCount(), "switched back on: read fresh, not the old snapshot")
}

func TestImportStatus_AnotherServerDropsTheOldSnapshot(t *testing.T) {
	env := newImportEnv(t, "radarr")
	env.radarr.set([]plugins.ImportHistoryRecord{event("AAA", plugins.HistoryEventImported, 5, 603)}, nil)
	env.service.Resolve(context.Background(), []string{"aaa"})

	other := &importFakeReader{}
	other.set(nil, errors.New("connection refused"))
	env.clients.setReader("radarr", other)
	env.clock.advance(importHistoryTTL + time.Second)

	env.service.Resolve(context.Background(), []string{"aaa"}) // serves stale, refresh fails against the new server
	env.service.waitRefreshes()
	assert.Empty(t, env.service.Resolve(context.Background(), []string{"aaa"}),
		"the old snapshot describes the previous server's downloads")
}
