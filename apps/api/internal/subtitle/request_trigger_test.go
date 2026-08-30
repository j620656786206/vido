package subtitle

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/subtitle/providers"
)

// --- fakes -----------------------------------------------------------------

type triggerFakeMovieFinder struct {
	movie *models.Movie
	err   error
}

func (f *triggerFakeMovieFinder) FindByTMDbID(_ context.Context, _ int64) (*models.Movie, error) {
	return f.movie, f.err
}

type triggerFakeSeriesFinder struct {
	series *models.Series
	err    error
}

func (f *triggerFakeSeriesFinder) FindByTMDbID(_ context.Context, _ int64) (*models.Series, error) {
	return f.series, f.err
}

type triggerEngineCall struct {
	mediaID    string
	mediaType  string
	filePath   string
	query      providers.SubtitleQuery
	resolution string
	opts       []ProcessOptions
}

type spyTriggerEngine struct {
	mu     sync.Mutex
	calls  []triggerEngineCall
	result EngineResult
}

func (s *spyTriggerEngine) Process(_ context.Context, mediaID, mediaType, mediaFilePath string,
	query providers.SubtitleQuery, mediaResolution string, opts ...ProcessOptions) EngineResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, triggerEngineCall{
		mediaID: mediaID, mediaType: mediaType, filePath: mediaFilePath,
		query: query, resolution: mediaResolution, opts: opts,
	})
	return s.result
}

func (s *spyTriggerEngine) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func newTrigger(movies movieByTMDbFinder, series seriesByTMDbFinder, engine batchEngine) *RequestCompletionTrigger {
	return NewRequestCompletionTrigger(movies, series, engine)
}

func movieRequest() models.Request {
	return models.Request{ID: "req-1", TMDbID: 550, MediaType: models.RequestMediaTypeMovie, Title: "鬥陣俱樂部"}
}

// --- tests -----------------------------------------------------------------

func TestRequestTrigger_MovieNeedingSearchFiresEngineWithCNPolicy(t *testing.T) {
	movie := &models.Movie{
		ID:                      "movie-1",
		Title:                   "無間道",
		SubtitleStatus:          models.SubtitleStatusNotSearched,
		FilePath:                models.NewNullString("/media/movies/infernal.mkv"),
		ProductionCountriesJSON: models.NewNullString(`[{"iso_3166_1":"CN","name":"China"},{"iso_3166_1":"HK","name":"Hong Kong"}]`),
	}
	engine := &spyTriggerEngine{result: EngineResult{Success: true, Language: "zh-TW"}}
	trigger := newTrigger(&triggerFakeMovieFinder{movie: movie}, &triggerFakeSeriesFinder{}, engine)

	trigger.run(movieRequest())

	require.Equal(t, 1, engine.callCount())
	call := engine.calls[0]
	assert.Equal(t, "movie-1", call.mediaID)
	assert.Equal(t, "movie", call.mediaType)
	assert.Equal(t, "/media/movies/infernal.mkv", call.filePath)
	assert.Equal(t, "無間道", call.query.Title)
	require.Len(t, call.opts, 1)
	assert.Equal(t, "CN,HK", call.opts[0].ProductionCountry, "CN policy must ride in like the batch path")
}

func TestRequestTrigger_NotFoundStatusIsRetried(t *testing.T) {
	movie := &models.Movie{
		ID:             "movie-2",
		Title:          "Old Miss",
		SubtitleStatus: models.SubtitleStatusNotFound, // a previous search came up empty
	}
	engine := &spyTriggerEngine{}
	trigger := newTrigger(&triggerFakeMovieFinder{movie: movie}, &triggerFakeSeriesFinder{}, engine)

	trigger.run(movieRequest())

	assert.Equal(t, 1, engine.callCount(), "not_found media deserves a fresh search on a new request")
}

func TestRequestTrigger_IdempotentWhenSubtitleAlreadyResolved(t *testing.T) {
	for _, status := range []models.SubtitleStatus{
		models.SubtitleStatusFound,
		models.SubtitleStatusSearching,
	} {
		engine := &spyTriggerEngine{}
		trigger := newTrigger(&triggerFakeMovieFinder{movie: &models.Movie{
			ID: "movie-3", Title: "X", SubtitleStatus: status,
		}}, &triggerFakeSeriesFinder{}, engine)

		trigger.run(movieRequest())

		assert.Equal(t, 0, engine.callCount(),
			"media with subtitle_status=%s must not be re-searched", status)
	}
}

func TestRequestTrigger_UnresolvableMovieSkipsQuietly(t *testing.T) {
	engine := &spyTriggerEngine{}
	trigger := newTrigger(&triggerFakeMovieFinder{err: errors.New("movie with tmdb_id 550 not found")},
		&triggerFakeSeriesFinder{}, engine)

	trigger.run(movieRequest()) // must not panic

	assert.Equal(t, 0, engine.callCount())
}

func TestRequestTrigger_TVRequestSearchesSeries(t *testing.T) {
	series := &models.Series{
		ID:             "series-1",
		Title:          "如懿傳",
		SubtitleStatus: models.SubtitleStatusNotSearched,
	}
	engine := &spyTriggerEngine{result: EngineResult{Success: false, Error: errors.New("no match")}}
	trigger := newTrigger(&triggerFakeMovieFinder{}, &triggerFakeSeriesFinder{series: series}, engine)

	trigger.run(models.Request{ID: "req-2", TMDbID: 999, MediaType: models.RequestMediaTypeTV, Title: "如懿傳"})

	require.Equal(t, 1, engine.callCount())
	call := engine.calls[0]
	assert.Equal(t, "series-1", call.mediaID)
	assert.Equal(t, "series", call.mediaType)
	assert.Equal(t, "如懿傳", call.query.Title)
}

func TestRequestTrigger_OnRequestCompletedIsAsyncAndSerialized(t *testing.T) {
	movie := &models.Movie{
		ID: "movie-4", Title: "A", SubtitleStatus: models.SubtitleStatusNotSearched,
	}
	engine := &spyTriggerEngine{}
	trigger := newTrigger(&triggerFakeMovieFinder{movie: movie}, &triggerFakeSeriesFinder{}, engine)

	// The seam call must return immediately (poller tick must never block) and
	// both completions must be processed.
	trigger.OnRequestCompleted(context.Background(), movieRequest())
	trigger.OnRequestCompleted(context.Background(), movieRequest())
	trigger.wg.Wait()

	assert.Equal(t, 2, engine.callCount())
}

func TestRequestTrigger_EngineFailureNeverPanics(t *testing.T) {
	movie := &models.Movie{
		ID: "movie-5", Title: "B", SubtitleStatus: models.SubtitleStatusNotSearched,
	}
	engine := &spyTriggerEngine{result: EngineResult{Success: false, Error: errors.New("provider down")}}
	trigger := newTrigger(&triggerFakeMovieFinder{movie: movie}, &triggerFakeSeriesFinder{}, engine)

	trigger.run(movieRequest()) // failure is logged, nothing propagates

	assert.Equal(t, 1, engine.callCount())
}
