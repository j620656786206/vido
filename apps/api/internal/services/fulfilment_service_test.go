package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/plugins"
)

// fakeFulfilmentRequestRepo captures UpdateFulfilment writes.
type fakeFulfilmentRequestRepo struct {
	mu      sync.Mutex
	updates []fulfilmentUpdate
	err     error
}

type fulfilmentUpdate struct {
	id               string
	status           string
	fulfilmentSource models.NullString
	externalID       models.NullString
	errorMessage     models.NullString
}

func (f *fakeFulfilmentRequestRepo) Create(ctx context.Context, request *models.Request) error {
	return nil
}
func (f *fakeFulfilmentRequestRepo) List(ctx context.Context) ([]models.Request, error) {
	return nil, nil
}
func (f *fakeFulfilmentRequestRepo) FindActiveByTMDbID(ctx context.Context, tmdbID int64, mediaType string) (*models.Request, error) {
	return nil, nil
}
func (f *fakeFulfilmentRequestRepo) UpdateFulfilment(ctx context.Context, id string, status string, fulfilmentSource, externalID, errorMessage models.NullString) (time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return time.Time{}, f.err
	}
	f.updates = append(f.updates, fulfilmentUpdate{id, status, fulfilmentSource, externalID, errorMessage})
	return fixedFulfilmentTime, nil
}

// fixedFulfilmentTime lets tests assert the in-memory row was synced with
// the repo-written updated_at (13-4a CR M1).
var fixedFulfilmentTime = time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)

func (f *fakeFulfilmentRequestRepo) lastUpdate(t *testing.T) fulfilmentUpdate {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	require.NotEmpty(t, f.updates, "expected at least one UpdateFulfilment write")
	return f.updates[len(f.updates)-1]
}

type fulfilmentTestEnv struct {
	service      *FulfilmentService
	manager      *plugins.Manager
	settings     *fakeDVRSettingsRepo
	secrets      *fakeDVRSecrets
	plugin       *fakeDVRPlugin // radarr
	sonarrPlugin *fakeDVRPlugin
	repo         *fakeFulfilmentRequestRepo
}

// newFulfilmentTestEnv builds a FULLY-configured, healthy radarr + sonarr
// environment; individual tests degrade it as needed.
func newFulfilmentTestEnv(t *testing.T) *fulfilmentTestEnv {
	t.Helper()
	settings := newFakeDVRSettingsRepo()
	secretsSvc := newFakeDVRSecrets()
	plugin := &fakeDVRPlugin{addMovieID: 42}
	sonarrPlugin := &fakeDVRPlugin{addSeriesID: 7}
	repo := &fakeFulfilmentRequestRepo{}

	ctx := context.Background()
	require.NoError(t, settings.SetString(ctx, plugins.SettingKeyURL("radarr"), "http://radarr:7878"))
	require.NoError(t, settings.SetBool(ctx, plugins.SettingKeyEnabled("radarr"), true))
	require.NoError(t, settings.SetInt(ctx, plugins.SettingKeyQualityProfileID("radarr"), 4))
	require.NoError(t, settings.SetString(ctx, plugins.SettingKeyRootFolderPath("radarr"), "/movies"))
	require.NoError(t, secretsSvc.Store(ctx, plugins.SettingKeyAPIKey("radarr"), "key"))

	require.NoError(t, settings.SetString(ctx, plugins.SettingKeyURL("sonarr"), "http://sonarr:8989"))
	require.NoError(t, settings.SetBool(ctx, plugins.SettingKeyEnabled("sonarr"), true))
	require.NoError(t, settings.SetInt(ctx, plugins.SettingKeyQualityProfileID("sonarr"), 6))
	require.NoError(t, settings.SetString(ctx, plugins.SettingKeyRootFolderPath("sonarr"), "/tv"))
	require.NoError(t, secretsSvc.Store(ctx, plugins.SettingKeyAPIKey("sonarr"), "key"))

	manager := plugins.NewManager(settings, secretsSvc, &fakeDVRHistoryRepo{}, nil, 0)
	manager.Register("radarr", func(config plugins.PluginConfig) plugins.DVRPlugin { return plugin })
	manager.Register("sonarr", func(config plugins.PluginConfig) plugins.DVRPlugin { return sonarrPlugin })

	service := NewFulfilmentService(manager, settings, repo)
	return &fulfilmentTestEnv{service: service, manager: manager, settings: settings, secrets: secretsSvc, plugin: plugin, sonarrPlugin: sonarrPlugin, repo: repo}
}

func pendingMovieRequest() *models.Request {
	return &models.Request{
		ID:        "req-1",
		TMDbID:    550,
		MediaType: models.RequestMediaTypeMovie,
		Title:     "鬥陣俱樂部",
		Status:    models.RequestStatusPending,
	}
}

func TestFulfilmentService_MovieSuccessTransition(t *testing.T) {
	env := newFulfilmentTestEnv(t)
	req := pendingMovieRequest()

	env.service.FulfilRequest(context.Background(), req)

	// AddMovie called with config-derived opts + SearchNow (AC #6).
	assert.Equal(t, int64(550), env.plugin.lastAddTMDb)
	assert.Equal(t, plugins.AddOptions{QualityProfileID: 4, RootFolderPath: "/movies", SearchNow: true}, env.plugin.lastAddOpts)

	// In-memory request mutated so the 201 response carries the transition.
	assert.Equal(t, models.RequestStatusSearching, req.Status)
	assert.Equal(t, "42", req.ExternalID.String)
	assert.Equal(t, models.RequestFulfilmentSourceArr, req.FulfilmentSource.String)
	assert.False(t, req.ErrorMessage.Valid)

	// Row persisted through the repo.
	update := env.repo.lastUpdate(t)
	assert.Equal(t, "req-1", update.id)
	assert.Equal(t, models.RequestStatusSearching, update.status)
	assert.Equal(t, "arr", update.fulfilmentSource.String)
	assert.Equal(t, "42", update.externalID.String)
	assert.False(t, update.errorMessage.Valid)

	// CR M1 — the in-memory row carries the repo-written updated_at, so the
	// 201 response matches what a subsequent GET returns.
	assert.Equal(t, fixedFulfilmentTime, req.UpdatedAt)
}

func TestFulfilmentService_RadarrUnconfigured_StaysPending(t *testing.T) {
	env := newFulfilmentTestEnv(t)
	require.NoError(t, env.settings.SetBool(context.Background(), plugins.SettingKeyEnabled("radarr"), false))
	req := pendingMovieRequest()

	env.service.FulfilRequest(context.Background(), req)

	assert.Equal(t, models.RequestStatusPending, req.Status, "graceful degradation: row stays pending")
	assert.Equal(t, "Radarr 未設定", req.ErrorMessage.String)
	assert.False(t, req.ExternalID.Valid)
	assert.Equal(t, 0, env.plugin.addMovieHits)

	update := env.repo.lastUpdate(t)
	assert.Equal(t, models.RequestStatusPending, update.status)
	assert.Equal(t, "Radarr 未設定", update.errorMessage.String)
}

func TestFulfilmentService_RadarrUnhealthy_StaysPending(t *testing.T) {
	env := newFulfilmentTestEnv(t)
	ctx := context.Background()
	env.plugin.testErr = assert.AnError
	env.manager.CheckHealth(ctx, "radarr") // record unhealthy state
	req := pendingMovieRequest()

	env.service.FulfilRequest(ctx, req)

	assert.Equal(t, models.RequestStatusPending, req.Status)
	assert.Equal(t, "Radarr 連線失敗", req.ErrorMessage.String)
	assert.Equal(t, 0, env.plugin.addMovieHits, "unhealthy gate must skip AddMovie")
}

func TestFulfilmentService_AddMovieError_StaysPending(t *testing.T) {
	env := newFulfilmentTestEnv(t)
	ctx := context.Background()
	env.manager.CheckHealth(ctx, "radarr") // healthy
	env.plugin.addMovieErr = &plugins.PluginError{Code: plugins.ErrCodeAddFailed, Message: "already exists"}
	req := pendingMovieRequest()

	env.service.FulfilRequest(ctx, req)

	assert.Equal(t, models.RequestStatusPending, req.Status)
	assert.Equal(t, "Radarr 新增失敗", req.ErrorMessage.String)
	assert.False(t, req.ExternalID.Valid)
}

func tvRequest() *models.Request {
	return &models.Request{
		ID:        "req-tv",
		TMDbID:    1399,
		MediaType: models.RequestMediaTypeTV,
		Title:     "冰與火之歌",
		Status:    models.RequestStatusPending,
	}
}

func TestFulfilmentService_TVSuccessTransition(t *testing.T) {
	// 13-4b AC #4 — tv requests now route to Sonarr as whole-series adds.
	env := newFulfilmentTestEnv(t)
	req := tvRequest()

	env.service.FulfilRequest(context.Background(), req)

	assert.Equal(t, int64(1399), env.sonarrPlugin.lastAddTMDb)
	assert.Equal(t, plugins.AddOptions{QualityProfileID: 6, RootFolderPath: "/tv", SearchNow: true}, env.sonarrPlugin.lastAddOpts)
	assert.Equal(t, 0, env.plugin.addMovieHits, "tv must never hit the radarr plugin")

	assert.Equal(t, models.RequestStatusSearching, req.Status)
	assert.Equal(t, "7", req.ExternalID.String)
	assert.Equal(t, models.RequestFulfilmentSourceArr, req.FulfilmentSource.String)
	assert.False(t, req.ErrorMessage.Valid)
	assert.Equal(t, fixedFulfilmentTime, req.UpdatedAt)

	update := env.repo.lastUpdate(t)
	assert.Equal(t, models.RequestStatusSearching, update.status)
	assert.Equal(t, "7", update.externalID.String)
}

func TestFulfilmentService_TV_NoTVDBEntry_TerminalFailed(t *testing.T) {
	// 13-4b AC #1.2/#4 — no TVDB entry is the ONE terminal fulfilment error:
	// honest failed, never a stranded pending (retrying cannot fix TVDB absence).
	env := newFulfilmentTestEnv(t)
	env.sonarrPlugin.addSeriesErr = &plugins.PluginError{
		Code:    plugins.ErrCodeTVDBNotFound,
		Message: "tmdb series 1399 has no TVDB entry",
	}
	req := tvRequest()

	env.service.FulfilRequest(context.Background(), req)

	assert.Equal(t, models.RequestStatusFailed, req.Status)
	assert.Equal(t, "此影集不在 TVDB 上，Sonarr 無法搜尋", req.ErrorMessage.String)
	assert.False(t, req.FulfilmentSource.Valid, "nothing claimed the row — source stays NULL")
	assert.False(t, req.ExternalID.Valid)

	update := env.repo.lastUpdate(t)
	assert.Equal(t, models.RequestStatusFailed, update.status)
	assert.Equal(t, "此影集不在 TVDB 上，Sonarr 無法搜尋", update.errorMessage.String)
}

func TestFulfilmentService_TV_SonarrUnconfigured_StaysPending(t *testing.T) {
	env := newFulfilmentTestEnv(t)
	require.NoError(t, env.settings.SetBool(context.Background(), plugins.SettingKeyEnabled("sonarr"), false))
	req := tvRequest()

	env.service.FulfilRequest(context.Background(), req)

	assert.Equal(t, models.RequestStatusPending, req.Status)
	assert.Equal(t, "Sonarr 未設定", req.ErrorMessage.String)
	assert.Equal(t, 0, env.sonarrPlugin.addSeriesHits)
}

func TestFulfilmentService_TV_SonarrUnhealthy_StaysPending(t *testing.T) {
	env := newFulfilmentTestEnv(t)
	ctx := context.Background()
	env.sonarrPlugin.testErr = assert.AnError
	env.manager.CheckHealth(ctx, "sonarr")
	req := tvRequest()

	env.service.FulfilRequest(ctx, req)

	assert.Equal(t, models.RequestStatusPending, req.Status)
	assert.Equal(t, "Sonarr 連線失敗", req.ErrorMessage.String)
	assert.Equal(t, 0, env.sonarrPlugin.addSeriesHits)
}

func TestFulfilmentService_TV_AddSeriesError_StaysPending(t *testing.T) {
	env := newFulfilmentTestEnv(t)
	env.sonarrPlugin.addSeriesErr = &plugins.PluginError{Code: plugins.ErrCodeAddFailed, Message: "already exists"}
	req := tvRequest()

	env.service.FulfilRequest(context.Background(), req)

	assert.Equal(t, models.RequestStatusPending, req.Status)
	assert.Equal(t, "Sonarr 新增失敗", req.ErrorMessage.String)
}

func TestFulfilmentService_IncompleteAddOptions_StaysPending(t *testing.T) {
	env := newFulfilmentTestEnv(t)
	ctx := context.Background()
	env.manager.CheckHealth(ctx, "radarr")
	// Remove the root folder → Radarr POST /movie would 400; gate it up front.
	require.NoError(t, env.settings.SetString(ctx, plugins.SettingKeyRootFolderPath("radarr"), ""))
	req := pendingMovieRequest()

	env.service.FulfilRequest(ctx, req)

	assert.Equal(t, models.RequestStatusPending, req.Status)
	assert.Equal(t, "Radarr 設定不完整（缺少品質設定檔或根資料夾）", req.ErrorMessage.String)
	assert.Equal(t, 0, env.plugin.addMovieHits)
}

func TestFulfilmentService_LazyHealthInitOnFirstCreate(t *testing.T) {
	// Boot-edge race: a request arriving before the scheduler's first sweep
	// must not be spuriously annotated — the service runs one lazy check.
	env := newFulfilmentTestEnv(t)
	req := pendingMovieRequest()

	env.service.FulfilRequest(context.Background(), req)

	assert.Equal(t, models.RequestStatusSearching, req.Status)
	assert.Equal(t, 1, env.plugin.testedCount, "exactly one lazy health check")
}
