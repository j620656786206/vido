package subtitle

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/database"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/subtitle/providers"
)

// infra-optin-usage-report-a1 — an online subtitle the engine places becomes a
// completed subtitle_runs row (route=online), so the usage report can count it.

func engineLedgerDB(t *testing.T) *sql.DB {
	t.Helper()
	// The app's own driver (sqlite-utc, #652) — the time text it writes is what
	// the window query compares in production.
	db, err := sql.Open(database.DriverName, ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

func foundProvider() *mockProvider {
	return &mockProvider{
		name: "assrt",
		searchResult: []providers.SubtitleResult{
			{ID: "1", Source: "assrt", Language: "zh-Hant", Filename: "sub.srt", Format: "srt", Downloads: 100},
		},
		downloadData: []byte("1\n00:00:01,000 --> 00:00:03,000\n這是繁體中文\n"),
	}
}

type failingLedger struct{ calls int }

func (f *failingLedger) Create(context.Context, *models.SubtitleRun) error {
	f.calls++
	return errors.New("disk full")
}

// Rule 28: the row goes through the REAL repository on a DB built by the real
// migration chain, so a column the scan forgets would surface here.
func TestEngine_Process_RecordsOnlineDeliveryInLedger(t *testing.T) {
	db := engineLedgerDB(t)
	runs := repository.NewSubtitleRunRepository(db)
	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{foundProvider()}, nil)
	engine.SetRunLedger(runs)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p", ProcessOptions{Automatic: true})
	require.True(t, result.Success)

	got, err := runs.CompletedRunsBetween(context.Background(), time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "movie-1", got[0].MediaID)
	assert.Equal(t, models.SubtitleRunMediaMovie, got[0].MediaType)
	assert.Equal(t, models.SubtitleRunCompleted, got[0].Status)
	assert.Equal(t, models.SubtitleRunRouteOnline, got[0].Route)
	assert.Equal(t, models.SubtitleRunTriggeredAuto, got[0].TriggeredBy)
	assert.Equal(t, result.SubtitlePath, got[0].OutputPath)
	require.NotNil(t, got[0].CompletedAt)
}

func TestEngine_Process_LedgerDefaultsToManual(t *testing.T) {
	db := engineLedgerDB(t)
	runs := repository.NewSubtitleRunRepository(db)
	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{foundProvider()}, nil)
	engine.SetRunLedger(runs)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p")
	require.True(t, result.Success)

	got, err := runs.CompletedRunsBetween(context.Background(), time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, models.SubtitleRunTriggeredManual, got[0].TriggeredBy,
		"the batch / manual callers pass no Automatic — they must never count as 'produced on its own'")
}

func TestEngine_Process_NotFoundWritesNoRun(t *testing.T) {
	db := engineLedgerDB(t)
	runs := repository.NewSubtitleRunRepository(db)
	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{&mockProvider{name: "assrt"}}, nil)
	engine.SetRunLedger(runs)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p", ProcessOptions{Automatic: true})
	require.False(t, result.Success)

	failed, err := runs.CountByStatus(context.Background(), models.SubtitleRunFailed)
	require.NoError(t, err)
	assert.Equal(t, 0, failed, "a not-found online search must not feed the home needs-attention cell")
	completed, err := runs.CountByStatus(context.Background(), models.SubtitleRunCompleted)
	require.NoError(t, err)
	assert.Equal(t, 0, completed)
}

func TestEngine_Process_LedgerFailureDoesNotFailTheDelivery(t *testing.T) {
	ledger := &failingLedger{}
	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{foundProvider()}, nil)
	engine.SetRunLedger(ledger)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p", ProcessOptions{Automatic: true})

	assert.True(t, result.Success, "the subtitle is already in place — bookkeeping cannot undo that")
	assert.FileExists(t, result.SubtitlePath)
	assert.Equal(t, 1, ledger.calls)
}

func TestEngine_Process_NoLedgerWiredRecordsNothing(t *testing.T) {
	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{foundProvider()}, nil)

	result := engine.Process(context.Background(), "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p", ProcessOptions{Automatic: true})

	assert.True(t, result.Success, "an engine without SetRunLedger behaves exactly as before")
}

// A caller cancelled after placement must not lose the row — the file exists.
// The engine detaches the write with context.WithoutCancel, whose Done()
// channel is nil; a spy proves the ledger never sees the caller's cancellation.
func TestEngine_Process_LedgerWriteIsDetachedFromCallerCancellation(t *testing.T) {
	spy := &ctxSpyLedger{}
	engine, mediaPath := newTestEngine(t, []providers.SubtitleProvider{foundProvider()}, nil)
	engine.SetRunLedger(spy)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := engine.Process(ctx, "movie-1", "movie", mediaPath,
		providers.SubtitleQuery{Title: "Test"}, "1080p", ProcessOptions{Automatic: true})
	require.True(t, result.Success)

	require.Equal(t, 1, spy.calls)
	assert.True(t, spy.detached, "the ledger write must not inherit the caller's cancellation")
}

type ctxSpyLedger struct {
	calls    int
	detached bool
}

func (s *ctxSpyLedger) Create(ctx context.Context, _ *models.SubtitleRun) error {
	s.calls++
	s.detached = ctx.Done() == nil
	return nil
}

func TestEngine_BareStructClockDoesNotPanic(t *testing.T) {
	e := &Engine{}
	assert.NotPanics(t, func() { _ = e.clock() })
}
