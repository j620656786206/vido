package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/plugins"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/repository"
)

// fakeQueueCleaner records RemoveFailedQueueItems calls and can fail, and
// can assert the row was still failed when it ran (the order AC #2b needs).
type fakeQueueCleaner struct {
	calls   []string
	removed int
	err     error
	during  func()
}

func (f *fakeQueueCleaner) DiscardFailedDownloads(ctx context.Context, plugin string, externalID int64) (int, error) {
	f.calls = append(f.calls, plugin)
	if f.during != nil {
		f.during()
	}
	return f.removed, f.err
}

type cancelRetryFixture struct {
	repo    *repository.RequestRepository
	svc     *RequestService
	ful     *stubFulfilment
	cleaner *fakeQueueCleaner
}

func newCancelRetryFixture(t *testing.T) *cancelRetryFixture {
	t.Helper()
	repo := repository.NewRequestRepository(newMigratedServicesTestDB(t))
	svc := NewRequestService(repo, nil, nil, nil, nil)
	ful := &stubFulfilment{}
	cleaner := &fakeQueueCleaner{}
	svc.SetFulfilmentService(ful)
	svc.SetQueueCleaner(cleaner)
	return &cancelRetryFixture{repo: repo, svc: svc, ful: ful, cleaner: cleaner}
}

// seed creates a row and moves it to status (with an *arr link when external != "").
func (f *cancelRetryFixture) seed(t *testing.T, tmdbID int64, mediaType, status, external string) *models.Request {
	t.Helper()
	ctx := context.Background()
	req := &models.Request{TMDbID: tmdbID, MediaType: mediaType, Title: "片"}
	require.NoError(t, f.repo.Create(ctx, req))
	if status != models.RequestStatusPending || external != "" {
		var src, ext, msg models.NullString
		if external != "" {
			src = models.NewNullString(models.RequestFulfilmentSourceArr)
			ext = models.NewNullString(external)
		}
		if status == models.RequestStatusFailed {
			msg = models.NewNullString("失敗了")
		}
		_, err := f.repo.UpdateFulfilment(ctx, req.ID, status, src, ext, msg)
		require.NoError(t, err)
	}
	row, err := f.repo.FindByID(ctx, req.ID)
	require.NoError(t, err)
	return row
}

func TestCancelRequest_StateMatrix(t *testing.T) {
	f := newCancelRetryFixture(t)
	ctx := context.Background()

	pending := f.seed(t, 1, models.RequestMediaTypeMovie, models.RequestStatusPending, "")
	require.NoError(t, f.svc.CancelRequest(ctx, pending.ID))
	_, err := f.repo.FindByID(ctx, pending.ID)
	assert.ErrorIs(t, err, repository.ErrRequestNotFound)

	for i, status := range []string{models.RequestStatusSearching, models.RequestStatusDownloading, models.RequestStatusCompleted, models.RequestStatusFailed} {
		row := f.seed(t, int64(10+i), models.RequestMediaTypeMovie, status, "5")
		err := f.svc.CancelRequest(ctx, row.ID)
		assert.ErrorIs(t, err, ErrRequestNotCancellable, status)
	}

	assert.ErrorIs(t, f.svc.CancelRequest(ctx, "missing"), repository.ErrRequestNotFound)
}

// AC #5: the poller snapshot still holds the row when the user cancels; its
// in-flight fulfilment write then lands on nothing and nothing comes back.
func TestCancelRequest_RaceWithPollerFulfilment(t *testing.T) {
	f := newCancelRetryFixture(t)
	ctx := context.Background()
	row := f.seed(t, 1, models.RequestMediaTypeMovie, models.RequestStatusPending, "")

	snapshot, err := f.repo.ListActive(ctx) // the poller's tick read
	require.NoError(t, err)
	require.Len(t, snapshot, 1)

	require.NoError(t, f.svc.CancelRequest(ctx, row.ID))

	// The poller's FulfilRequest finished AddMovie and writes the transition.
	_, err = f.repo.UpdateFulfilment(ctx, snapshot[0].ID, models.RequestStatusSearching,
		models.NewNullString(models.RequestFulfilmentSourceArr), models.NewNullString("99"), models.NullString{})
	assert.ErrorIs(t, err, repository.ErrRequestNotFound, "logged by FulfilmentService, never resurrects the row")

	all, err := f.repo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestRetryRequest_TerminalFailedRefulfils(t *testing.T) {
	f := newCancelRetryFixture(t)
	ctx := context.Background()
	row := f.seed(t, 1, models.RequestMediaTypeTV, models.RequestStatusFailed, "")
	f.ful.simulate = func(r *models.Request) {
		assert.Equal(t, models.RequestStatusPending, r.Status, "fulfilment runs on the reset row")
		r.Status = models.RequestStatusSearching
	}

	got, err := f.svc.RetryRequest(ctx, row.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, f.ful.calls)
	assert.Equal(t, models.RequestStatusSearching, got.Status, "the response carries what fulfilment produced")
	assert.Empty(t, f.cleaner.calls, "nothing in *arr to clean")

	stored, _ := f.repo.FindByID(ctx, row.ID)
	assert.Equal(t, models.RequestStatusPending, stored.Status)
	assert.False(t, stored.ErrorMessage.Valid)
}

func TestRetryRequest_QueueFailedDiscardsThenSearches(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mediaType string
		plugin    string
		removed   int
	}{
		{"movie, broken download removed", models.RequestMediaTypeMovie, "radarr", 1},
		{"tv, queue entry already gone", models.RequestMediaTypeTV, "sonarr", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCancelRetryFixture(t)
			ctx := context.Background()
			row := f.seed(t, 1, tc.mediaType, models.RequestStatusFailed, "42")
			f.cleaner.removed = tc.removed
			f.cleaner.during = func() {
				now, _ := f.repo.FindByID(ctx, row.ID)
				assert.Equal(t, models.RequestStatusFailed, now.Status, "discard happens BEFORE the reset")
			}

			got, err := f.svc.RetryRequest(ctx, row.ID)
			require.NoError(t, err)
			assert.Equal(t, []string{tc.plugin}, f.cleaner.calls)
			assert.Equal(t, models.RequestStatusSearching, got.Status)
			assert.Equal(t, "42", got.ExternalID.String, "the *arr link is kept")
			assert.False(t, got.ErrorMessage.Valid)
			assert.Zero(t, f.ful.calls, "*arr already has the title")
		})
	}
}

// CR: a retry whose cleanup fails must not pretend — the row stays failed
// (the next poller tick would only flip a reset row back).
func TestRetryRequest_CleanupFailureKeepsRowFailed(t *testing.T) {
	f := newCancelRetryFixture(t)
	ctx := context.Background()
	row := f.seed(t, 1, models.RequestMediaTypeMovie, models.RequestStatusFailed, "42")
	f.cleaner.err = errors.New("radarr down")

	_, err := f.svc.RetryRequest(ctx, row.ID)
	assert.ErrorIs(t, err, ErrRetryCleanupFailed)
	stored, _ := f.repo.FindByID(ctx, row.ID)
	assert.Equal(t, models.RequestStatusFailed, stored.Status)
	assert.True(t, stored.ErrorMessage.Valid)
}

// CR: a newer active request for the title is refused BEFORE *arr is told to
// blocklist and re-search anything.
func TestRetryRequest_DuplicateCheckedBeforeCleanup(t *testing.T) {
	f := newCancelRetryFixture(t)
	ctx := context.Background()
	failed := f.seed(t, 1, models.RequestMediaTypeMovie, models.RequestStatusFailed, "42")
	f.seed(t, 1, models.RequestMediaTypeMovie, models.RequestStatusPending, "")

	_, err := f.svc.RetryRequest(ctx, failed.ID)
	assert.ErrorIs(t, err, repository.ErrRequestDuplicate)
	assert.Empty(t, f.cleaner.calls)
}

func TestRetryRequest_Rejections(t *testing.T) {
	f := newCancelRetryFixture(t)
	ctx := context.Background()

	_, err := f.svc.RetryRequest(ctx, "missing")
	assert.ErrorIs(t, err, repository.ErrRequestNotFound)

	for i, status := range []string{models.RequestStatusPending, models.RequestStatusSearching, models.RequestStatusDownloading, models.RequestStatusCompleted} {
		row := f.seed(t, int64(10+i), models.RequestMediaTypeMovie, status, "")
		_, err := f.svc.RetryRequest(ctx, row.ID)
		assert.ErrorIs(t, err, ErrRequestNotRetryable, status)
	}

	// The user already re-requested the title after it failed.
	failed := f.seed(t, 20, models.RequestMediaTypeMovie, models.RequestStatusFailed, "")
	f.seed(t, 20, models.RequestMediaTypeMovie, models.RequestStatusPending, "")
	_, err = f.svc.RetryRequest(ctx, failed.ID)
	assert.ErrorIs(t, err, repository.ErrRequestDuplicate)
	assert.Zero(t, f.ful.calls)
}

// removableDVR is a DVR plugin fake with the QueueRemover capability.
type removableDVR struct {
	*fakeDVRPlugin
	removedIDs []string
	removeErr  error
}

func (r *removableDVR) RemoveQueueItems(ctx context.Context, externalID int64, downloadIDs []string) (int, error) {
	r.removedIDs = append(r.removedIDs, downloadIDs...)
	return len(downloadIDs), r.removeErr
}

type oneClientSource struct{ client plugins.DVRPlugin }

func (o oneClientSource) RegisteredPlugins() []string { return []string{"radarr"} }
func (o oneClientSource) Health(string) plugins.PluginHealth {
	return plugins.PluginHealth{Status: plugins.HealthStatusHealthy}
}
func (o oneClientSource) GetClient(context.Context, string) (plugins.DVRPlugin, error) {
	return o.client, nil
}

// CR: the cleaner removes exactly what the poller would call failed — *arr
// "failed", or a qBittorrent error torrent — and never a stalled-but-healthy
// download that *arr merely flags "warning".
func TestDVRQueueCleaner_RemovesWhatThePollerCallsFailed(t *testing.T) {
	queue := []plugins.QueueItem{
		{ExternalID: 42, Status: "failed", DownloadID: "ARRFAILED"},
		{ExternalID: 42, Status: "warning", DownloadID: "QBTERROR"},
		{ExternalID: 42, Status: "warning", DownloadID: "STALLED"},
		{ExternalID: 42, Status: "downloading", DownloadID: "HEALTHY"},
		{ExternalID: 7, Status: "failed", DownloadID: "OTHERTITLE"},
	}
	torrents := &fakeTorrents{torrents: []qbittorrent.Torrent{
		{Hash: "qbterror", Status: qbittorrent.StatusError},
		{Hash: "stalled", Status: qbittorrent.StatusStalled},
		{Hash: "healthy", Status: qbittorrent.StatusDownloading},
	}}

	t.Run("with qBittorrent evidence", func(t *testing.T) {
		dvr := &removableDVR{fakeDVRPlugin: &fakeDVRPlugin{queue: queue}}
		n, err := NewDVRQueueCleaner(oneClientSource{dvr}, torrents).DiscardFailedDownloads(context.Background(), "radarr", 42)
		require.NoError(t, err)
		assert.Equal(t, 2, n)
		assert.ElementsMatch(t, []string{"ARRFAILED", "QBTERROR"}, dvr.removedIDs)
	})

	t.Run("qBittorrent unavailable → only *arr's own failed", func(t *testing.T) {
		dvr := &removableDVR{fakeDVRPlugin: &fakeDVRPlugin{queue: queue}}
		down := &fakeTorrents{err: errors.New("qbt down")}
		_, err := NewDVRQueueCleaner(oneClientSource{dvr}, down).DiscardFailedDownloads(context.Background(), "radarr", 42)
		require.NoError(t, err)
		assert.Equal(t, []string{"ARRFAILED"}, dvr.removedIDs)
	})

	t.Run("nothing failed → nothing removed, not an error", func(t *testing.T) {
		dvr := &removableDVR{fakeDVRPlugin: &fakeDVRPlugin{queue: queue[3:4]}}
		n, err := NewDVRQueueCleaner(oneClientSource{dvr}, torrents).DiscardFailedDownloads(context.Background(), "radarr", 42)
		require.NoError(t, err)
		assert.Zero(t, n)
		assert.Empty(t, dvr.removedIDs)
	})

	t.Run("queue unreadable → error", func(t *testing.T) {
		dvr := &removableDVR{fakeDVRPlugin: &fakeDVRPlugin{queueErr: errors.New("radarr down")}}
		_, err := NewDVRQueueCleaner(oneClientSource{dvr}, torrents).DiscardFailedDownloads(context.Background(), "radarr", 42)
		assert.Error(t, err)
	})
}
