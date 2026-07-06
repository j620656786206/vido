package handlers

// Story 9R-18 AC 6 (Murat, blocking): the permanent tripwire for media-id
// format assumptions. A movie created through the REAL creation path
// (LibraryService.SaveMovieFromTMDb → uuid.New().String() PK, real sqlite +
// production migrations) must flow through the whole Route C chain over HTTP:
//
//	POST /movies/{uuid}/transcribe   → NOT 400 (503-or-202 per availability)
//	GET  /subtitles/generation-batch/preview?scope=missing → counts it
//	POST /subtitles/generation-batch {scope:"missing"}     → enumerates it
//
// Composed from library_service_test.go setupTestDB (real sqlite + migrations
// + real repos) and the glossary_handler_test.go httptest pattern.
//
// Media-id fixture convention (9R-18 AC 7): media ids are UUID STRINGS — this
// test does not even invent one: it uses whatever the real creation path mints.

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/tmdb"

	_ "modernc.org/sqlite"
)

// setupRouteCTestDB mirrors services/library_service_test.go setupTestDB:
// a temp sqlite file with the FULL production migration set applied.
func setupRouteCTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test_route_c_*.db")
	require.NoError(t, err)
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	db, err := sql.Open("sqlite", tmpFile.Name()+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

// integrationGenerationRunner is available-and-instantly-successful so the
// batch legs stay deterministic (real transcription needs ffmpeg + an API key;
// the id contract — not the pipeline — is under test here).
type integrationGenerationRunner struct {
	mu    chan struct{} // buffered(1) as a tiny mutex
	calls []string
}

func newIntegrationGenerationRunner() *integrationGenerationRunner {
	return &integrationGenerationRunner{mu: make(chan struct{}, 1)}
}

func (r *integrationGenerationRunner) IsAvailable() bool { return true }
func (r *integrationGenerationRunner) RunTranscription(_ context.Context, mediaID string, _ string, _ string, _ ...services.TranscriptionOption) error {
	r.mu <- struct{}{}
	r.calls = append(r.calls, mediaID)
	<-r.mu
	return nil
}
func (r *integrationGenerationRunner) callIDs() []string {
	r.mu <- struct{}{}
	defer func() { <-r.mu }()
	out := make([]string, len(r.calls))
	copy(out, r.calls)
	return out
}

func TestRouteC_UUIDMovie_FlowsThroughWholeChain(t *testing.T) {
	db := setupRouteCTestDB(t)
	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)
	libSvc := services.NewLibraryService(movieRepo, seriesRepo, episodeRepo)
	ctx := context.Background()

	// A real media file on disk — the transcribe handler os.Stats it.
	mediaDir := t.TempDir()
	mediaPath := filepath.Join(mediaDir, "Fight.Club.1999.mkv")
	require.NoError(t, os.WriteFile(mediaPath, []byte("fake mkv"), 0o644))

	// REAL creation path: the PK is uuid.New().String() — exactly what every
	// scanned library row looks like (retro-8-TD4 prod data).
	movie, err := libSvc.SaveMovieFromTMDb(ctx, &tmdb.MovieDetails{
		Movie: tmdb.Movie{ID: 550, Title: "Fight Club", ReleaseDate: "1999-10-15"},
	}, mediaPath)
	require.NoError(t, err)
	movieUUID := movie.ID
	_, err = uuid.Parse(movieUUID)
	require.NoError(t, err, "creation path must mint a real UUID id (got %q)", movieUUID)
	require.Equal(t, models.SubtitleStatus(""), movie.SubtitleStatus, "fresh row must count as missing zh-Hant")

	// Wire the REAL handlers over the real repo.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1")

	// Transcribe leg: real TranscriptionService — availability depends on the
	// environment's ffmpeg (503 TRANSCRIPTION_DISABLED) vs full pass (202).
	transcriptionSvc := services.NewTranscriptionService(
		services.NewAudioExtractorService(1, time.Minute, nil),
		ai.NewWhisperClient("test-key"), nil, nil)
	NewTranscriptionHandler(services.NewMovieService(movieRepo), transcriptionSvc).RegisterRoutes(api)

	// Batch legs: real processor + real repo finder; deterministic stub runner.
	runner := newIntegrationGenerationRunner()
	processor := services.NewGenerationBatchProcessor(runner, movieRepo, nil, 5, nil)
	NewGenerationBatchHandler(processor).RegisterRoutes(api)

	// ── Leg 1: POST /movies/{uuid}/transcribe must NOT 400 ────────────────────
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/movies/"+movieUUID+"/transcribe", nil)
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusBadRequest, w.Code,
		"a UUID id must never 400 — the int64 ParseInt regression (9R-18)")
	require.Contains(t, []int{http.StatusAccepted, http.StatusServiceUnavailable}, w.Code,
		"503-or-202 depending on the availability gate; body: %s", w.Body.String())
	if w.Code == http.StatusServiceUnavailable {
		assert.Contains(t, w.Body.String(), "TRANSCRIPTION_DISABLED",
			"a 503 here must be the availability gate, not an id problem")
	}

	// Hermeticity drain (CR 9R-18): a 202 spawns the DETACHED async pipeline
	// goroutine (context.Background + service timeout) which runs ffmpeg on the
	// fake mkv and fails within ms — but nothing else bounds it. Wait for the
	// single-flight slot to clear so the goroutine (and its ffmpeg child) cannot
	// outlive the test, race t.TempDir()/db cleanup, or pollute the package run.
	if w.Code == http.StatusAccepted {
		drainDeadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(drainDeadline) && transcriptionSvc.IsInProgress(movieUUID) {
			time.Sleep(10 * time.Millisecond)
		}
		require.False(t, transcriptionSvc.IsInProgress(movieUUID),
			"async transcription goroutine must finish (extract fails fast on the fake mkv) before the test proceeds")
	}

	// ── Leg 2: preview counts the UUID movie ──────────────────────────────────
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/subtitles/generation-batch/preview?scope=missing", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	var preview struct {
		Data struct {
			TotalItems int `json:"total_items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &preview))
	assert.Equal(t, 1, preview.Data.TotalItems, "the UUID movie must be counted")

	// ── Leg 3: batch start enumerates the UUID movie ──────────────────────────
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/generation-batch",
		strings.NewReader(`{"scope":"missing"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusAccepted, w.Code,
		"batch must enumerate 1 item, not 0 (the toItem ParseInt-skip regression); body: %s", w.Body.String())
	var start struct {
		Data struct {
			TotalItems int `json:"total_items"`
			Items      []struct {
				MediaID string `json:"media_id"`
				Title   string `json:"title"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &start))
	require.Equal(t, 1, start.Data.TotalItems)
	require.Len(t, start.Data.Items, 1)
	assert.Equal(t, movieUUID, start.Data.Items[0].MediaID,
		"items[0].media_id must be the UUID the creation path minted")

	// The queue actually runs against the UUID (RunTranscription receives it).
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && processor.IsRunning() {
		time.Sleep(5 * time.Millisecond)
	}
	assert.Equal(t, []string{movieUUID}, runner.callIDs())
}
