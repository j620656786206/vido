package subtitle

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/subtitle/providers"
)

// Story 13-5 (G-5/P3-005, artery #5): when a request transitions into
// `completed`, the media just landed in the library — kick the Epic 8 subtitle
// search for it so the 想要 → 下載 → 字幕 pipeline closes without a manual step.
//
// This type implements the poller's OnRequestCompleted seam (13-3a scouted it
// deliberately: the seam fires exactly once per transition INTO completed, so
// re-polls never re-fire — first-order idempotence lives on the edge). Second-
// order idempotence lives here: media whose subtitle_status is anything other
// than 需要字幕 (not_searched / not_found) is skipped, so a restart replaying a
// transition — or a request for media that already has subtitles — never
// double-searches. Failures are logged and NEVER touch the request row: the
// request is completed because the media landed; subtitles are best-effort on
// top.

// movieByTMDbFinder is the narrow port over MovieRepository this trigger needs.
type movieByTMDbFinder interface {
	FindByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error)
}

// seriesByTMDbFinder is the narrow port over SeriesRepository this trigger needs.
type seriesByTMDbFinder interface {
	FindByTMDbID(ctx context.Context, tmdbID int64) (*models.Series, error)
}

// defaultTriggerTimeout bounds one search run (2 providers + download +
// convert + place). Generous next to the <60s success criterion — the ceiling
// exists so a hung provider cannot pin the serializing mutex forever.
const defaultTriggerTimeout = 5 * time.Minute

// RequestCompletionTrigger wires request completion to the subtitle engine.
type RequestCompletionTrigger struct {
	movies movieByTMDbFinder
	series seriesByTMDbFinder
	engine batchEngine // the same narrow port the batch loop uses

	// mu serializes searches — the batch loop is deliberately sequential for
	// provider politeness, and this trigger honors the same discipline when
	// several requests complete on one poll tick.
	mu      sync.Mutex
	timeout time.Duration

	// wg lets tests wait for the async run; production never waits on it.
	wg sync.WaitGroup
}

// NewRequestCompletionTrigger creates the 13-5 trigger.
func NewRequestCompletionTrigger(movies movieByTMDbFinder, series seriesByTMDbFinder, engine batchEngine) *RequestCompletionTrigger {
	return &RequestCompletionTrigger{
		movies:  movies,
		series:  series,
		engine:  engine,
		timeout: defaultTriggerTimeout,
	}
}

// OnRequestCompleted matches the poller seam signature. It must never block a
// poll tick, so the work runs on its own goroutine with a background context —
// the tick's ctx dies with the tick, while the search legitimately outlives it.
func (t *RequestCompletionTrigger) OnRequestCompleted(_ context.Context, req models.Request) {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.run(req)
	}()
}

func (t *RequestCompletionTrigger) run(req models.Request) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()

	switch req.MediaType {
	case models.RequestMediaTypeMovie:
		t.triggerMovie(ctx, req)
	case models.RequestMediaTypeTV:
		t.triggerSeries(ctx, req)
	default:
		slog.Warn("Request-completion subtitle trigger: unknown media type — skipped",
			"request_id", req.ID, "media_type", req.MediaType)
	}
}

// needsSearch is the idempotence gate: only media still waiting for a subtitle
// is searched — mirroring the batch collector's "needing subtitles" definition.
func needsSearch(status models.SubtitleStatus) bool {
	return status == models.SubtitleStatusNotSearched || status == models.SubtitleStatusNotFound
}

func (t *RequestCompletionTrigger) triggerMovie(ctx context.Context, req models.Request) {
	movie, err := t.movies.FindByTMDbID(ctx, req.TMDbID)
	if err != nil || movie == nil {
		// The poller declares completed once the media landed, but the parse
		// that stamps tmdb_id may still be in flight — a real race, not a bug.
		// The next avenue for this media is the normal batch/manual search.
		slog.Info("Request-completion subtitle trigger: movie not resolvable yet — skipped",
			"request_id", req.ID, "tmdb_id", req.TMDbID, "error", err)
		return
	}
	if !needsSearch(movie.SubtitleStatus) {
		slog.Debug("Request-completion subtitle trigger: movie already has a subtitle outcome — skipped",
			"request_id", req.ID, "media_id", movie.ID, "subtitle_status", string(movie.SubtitleStatus))
		return
	}

	// CN policy rides in exactly as the batch path passes it: production
	// countries decide whether 簡體 stays unconverted (Epic 8 ConversionPolicy).
	country := ""
	if countries, err := movie.GetProductionCountries(); err == nil {
		codes := make([]string, 0, len(countries))
		for _, c := range countries {
			codes = append(codes, c.ISO3166_1)
		}
		country = strings.Join(codes, ",")
	}
	filePath := ""
	if movie.FilePath.Valid {
		filePath = movie.FilePath.String
	}

	slog.Info("Request completed — starting automatic subtitle search",
		"request_id", req.ID, "media_id", movie.ID, "title", movie.Title)
	result := t.engine.Process(ctx, movie.ID, "movie", filePath,
		providers.SubtitleQuery{Title: movie.Title}, "",
		ProcessOptions{ProductionCountry: country})
	t.logResult(req.ID, movie.ID, movie.Title, result)
}

func (t *RequestCompletionTrigger) triggerSeries(ctx context.Context, req models.Request) {
	series, err := t.series.FindByTMDbID(ctx, req.TMDbID)
	if err != nil || series == nil {
		slog.Info("Request-completion subtitle trigger: series not resolvable yet — skipped",
			"request_id", req.ID, "tmdb_id", req.TMDbID, "error", err)
		return
	}
	if !needsSearch(series.SubtitleStatus) {
		slog.Debug("Request-completion subtitle trigger: series already has a subtitle outcome — skipped",
			"request_id", req.ID, "media_id", series.ID, "subtitle_status", string(series.SubtitleStatus))
		return
	}

	filePath := ""
	if series.FilePath.Valid {
		filePath = series.FilePath.String
	}

	slog.Info("Request completed — starting automatic subtitle search",
		"request_id", req.ID, "media_id", series.ID, "title", series.Title)
	// Series-level search mirrors the batch path: no production_countries on
	// the Series model → empty string = ConvertAuto (recorded batch limitation).
	result := t.engine.Process(ctx, series.ID, "series", filePath,
		providers.SubtitleQuery{Title: series.Title}, "",
		ProcessOptions{})
	t.logResult(req.ID, series.ID, series.Title, result)
}

func (t *RequestCompletionTrigger) logResult(requestID, mediaID, title string, result EngineResult) {
	if result.Success {
		slog.Info("Automatic subtitle search succeeded",
			"request_id", requestID, "media_id", mediaID, "title", title,
			"language", result.Language, "provider", result.ProviderUsed)
		return
	}
	errMsg := "no subtitle found"
	if result.Error != nil {
		errMsg = result.Error.Error()
	}
	// Best-effort by design: the failure is logged, the request stays completed,
	// and the media keeps whatever subtitle_status the engine stamped
	// (not_found → picked up by the next batch run).
	slog.Warn("Automatic subtitle search did not land a subtitle",
		"request_id", requestID, "media_id", mediaID, "title", title, "error", errMsg)
}
