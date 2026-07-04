package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/plugins"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/secrets"
)

// --- map-backed fakes (stateful save/load round-trips need real storage) ----

type fakeDVRSettingsRepo struct {
	mu      sync.Mutex
	strings map[string]string
	bools   map[string]bool
	ints    map[string]int
}

func newFakeDVRSettingsRepo() *fakeDVRSettingsRepo {
	return &fakeDVRSettingsRepo{strings: map[string]string{}, bools: map[string]bool{}, ints: map[string]int{}}
}

func (f *fakeDVRSettingsRepo) Set(ctx context.Context, setting *models.Setting) error { return nil }
func (f *fakeDVRSettingsRepo) Get(ctx context.Context, key string) (*models.Setting, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeDVRSettingsRepo) GetAll(ctx context.Context) ([]models.Setting, error) {
	return nil, nil
}
func (f *fakeDVRSettingsRepo) Delete(ctx context.Context, key string) error { return nil }
func (f *fakeDVRSettingsRepo) GetString(ctx context.Context, key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.strings[key]
	if !ok {
		return "", fmt.Errorf("setting %s not found", key)
	}
	return v, nil
}
func (f *fakeDVRSettingsRepo) GetInt(ctx context.Context, key string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.ints[key]
	if !ok {
		return 0, fmt.Errorf("setting %s not found", key)
	}
	return v, nil
}
func (f *fakeDVRSettingsRepo) GetBool(ctx context.Context, key string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.bools[key]
	if !ok {
		return false, fmt.Errorf("setting %s not found", key)
	}
	return v, nil
}
func (f *fakeDVRSettingsRepo) SetString(ctx context.Context, key, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.strings[key] = value
	return nil
}
func (f *fakeDVRSettingsRepo) SetInt(ctx context.Context, key string, value int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ints[key] = value
	return nil
}
func (f *fakeDVRSettingsRepo) SetBool(ctx context.Context, key string, value bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bools[key] = value
	return nil
}

var _ repository.SettingsRepositoryInterface = (*fakeDVRSettingsRepo)(nil)

type fakeDVRSecrets struct {
	mu    sync.Mutex
	store map[string]string
}

func newFakeDVRSecrets() *fakeDVRSecrets { return &fakeDVRSecrets{store: map[string]string{}} }

func (f *fakeDVRSecrets) Store(ctx context.Context, name string, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.store[name] = value
	return nil
}
func (f *fakeDVRSecrets) Retrieve(ctx context.Context, name string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.store[name]
	if !ok {
		return "", errors.New("secret not found")
	}
	return v, nil
}
func (f *fakeDVRSecrets) Delete(ctx context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.store, name)
	return nil
}
func (f *fakeDVRSecrets) Exists(ctx context.Context, name string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.store[name]
	return ok, nil
}
func (f *fakeDVRSecrets) List(ctx context.Context) ([]string, error) { return nil, nil }

var _ secrets.SecretsServiceInterface = (*fakeDVRSecrets)(nil)

type fakeDVRHistoryRepo struct{}

func (f *fakeDVRHistoryRepo) Create(ctx context.Context, event *models.ConnectionEvent) error {
	return nil
}
func (f *fakeDVRHistoryRepo) GetHistory(ctx context.Context, service string, limit int) ([]models.ConnectionEvent, error) {
	return nil, nil
}

// fakeDVRPlugin implements plugins.DVRPlugin + plugins.ProfileLister.
type fakeDVRPlugin struct {
	mu          sync.Mutex
	testErr     error
	lastTested  plugins.PluginConfig
	testedCount int
	profiles    []plugins.QualityProfile
	folders     []plugins.RootFolder

	addMovieID   int64
	addMovieErr  error
	lastAddTMDb  int64
	lastAddOpts  plugins.AddOptions
	addMovieHits int

	addSeriesID   int64
	addSeriesErr  error
	addSeriesHits int
}

func (f *fakeDVRPlugin) Name() string { return "radarr" }
func (f *fakeDVRPlugin) TestConnection(ctx context.Context, config plugins.PluginConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastTested = config
	f.testedCount++
	return f.testErr
}
func (f *fakeDVRPlugin) AddMovie(ctx context.Context, tmdbID int64, opts plugins.AddOptions) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastAddTMDb = tmdbID
	f.lastAddOpts = opts
	f.addMovieHits++
	if f.addMovieErr != nil {
		return 0, f.addMovieErr
	}
	if f.addMovieID != 0 {
		return f.addMovieID, nil
	}
	return 1, nil
}
func (f *fakeDVRPlugin) AddSeries(ctx context.Context, tmdbID int64, opts plugins.AddOptions) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastAddTMDb = tmdbID
	f.lastAddOpts = opts
	f.addSeriesHits++
	if f.addSeriesErr != nil {
		return 0, f.addSeriesErr
	}
	if f.addSeriesID != 0 {
		return f.addSeriesID, nil
	}
	return 0, &plugins.PluginError{Code: plugins.ErrCodeNotSupported, Message: "movie-only"}
}
func (f *fakeDVRPlugin) GetQueue(ctx context.Context) ([]plugins.QueueItem, error) { return nil, nil }
func (f *fakeDVRPlugin) GetQualityProfiles(ctx context.Context) ([]plugins.QualityProfile, error) {
	return f.profiles, nil
}
func (f *fakeDVRPlugin) GetRootFolders(ctx context.Context) ([]plugins.RootFolder, error) {
	return f.folders, nil
}

// --- harness -----------------------------------------------------------------

type dvrTestEnv struct {
	service  *DVRSettingsService
	manager  *plugins.Manager
	settings *fakeDVRSettingsRepo
	secrets  *fakeDVRSecrets
	plugin   *fakeDVRPlugin
}

func newDVRTestEnv(t *testing.T) *dvrTestEnv {
	t.Helper()
	settings := newFakeDVRSettingsRepo()
	secretsSvc := newFakeDVRSecrets()
	plugin := &fakeDVRPlugin{}

	manager := plugins.NewManager(settings, secretsSvc, &fakeDVRHistoryRepo{}, nil, 0)
	manager.Register("radarr", func(config plugins.PluginConfig) plugins.DVRPlugin { return plugin })

	service := NewDVRSettingsService(manager, settings, secretsSvc)
	return &dvrTestEnv{service: service, manager: manager, settings: settings, secrets: secretsSvc, plugin: plugin}
}

func (e *dvrTestEnv) seedSaved(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, e.settings.SetString(ctx, plugins.SettingKeyURL("radarr"), "http://radarr:7878"))
	require.NoError(t, e.settings.SetBool(ctx, plugins.SettingKeyEnabled("radarr"), true))
	require.NoError(t, e.settings.SetInt(ctx, plugins.SettingKeyQualityProfileID("radarr"), 4))
	require.NoError(t, e.settings.SetString(ctx, plugins.SettingKeyRootFolderPath("radarr"), "/movies"))
	require.NoError(t, e.secrets.Store(ctx, plugins.SettingKeyAPIKey("radarr"), "saved-key"))
}

// --- tests --------------------------------------------------------------------

func TestDVRSettingsService_SaveConfig_TestBeforeSaveRefusal(t *testing.T) {
	env := newDVRTestEnv(t)
	env.plugin.testErr = errors.New("dial tcp: connection refused")

	err := env.service.SaveConfig(context.Background(), "radarr", DVRConfigInput{
		URL:     "http://radarr:7878",
		APIKey:  "new-key",
		Enabled: true,
	})

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeTestFailed, pluginErr.Code)

	// Refused persistence — nothing may be saved (AC #4 server-side guard).
	_, urlErr := env.settings.GetString(context.Background(), plugins.SettingKeyURL("radarr"))
	assert.Error(t, urlErr, "url must NOT be persisted after a failed test")
	exists, _ := env.secrets.Exists(context.Background(), plugins.SettingKeyAPIKey("radarr"))
	assert.False(t, exists, "api key must NOT be persisted after a failed test")
}

func TestDVRSettingsService_SaveConfig_SuccessRoundTrip(t *testing.T) {
	env := newDVRTestEnv(t)
	ctx := context.Background()

	err := env.service.SaveConfig(ctx, "radarr", DVRConfigInput{
		URL:              "http://radarr:7878",
		APIKey:           "brand-new-key",
		Enabled:          true,
		QualityProfileID: 4,
		RootFolderPath:   "/movies",
	})
	require.NoError(t, err)

	// Candidate config (not the saved one) was probed.
	assert.Equal(t, "http://radarr:7878", env.plugin.lastTested.URL)
	assert.Equal(t, "brand-new-key", env.plugin.lastTested.APIKey)

	// Secrets round-trip through the manager's loader.
	config, enabled, loadErr := env.manager.LoadConfig(ctx, "radarr")
	require.NoError(t, loadErr)
	assert.True(t, enabled)
	assert.Equal(t, "http://radarr:7878", config.URL)
	assert.Equal(t, "brand-new-key", config.APIKey)

	status, err := env.service.GetConfig(ctx, "radarr")
	require.NoError(t, err)
	assert.Equal(t, "http://radarr:7878", status.URL)
	assert.True(t, status.Enabled)
	assert.Equal(t, int64(4), status.QualityProfileID)
	assert.Equal(t, "/movies", status.RootFolderPath)
	assert.True(t, status.HasAPIKey)
}

func TestDVRSettingsService_SaveConfig_EmptyKeyKeepsStored(t *testing.T) {
	env := newDVRTestEnv(t)
	env.seedSaved(t)
	ctx := context.Background()

	err := env.service.SaveConfig(ctx, "radarr", DVRConfigInput{
		URL:              "http://radarr:9999", // URL change without re-entering the key
		Enabled:          true,
		QualityProfileID: 4,
		RootFolderPath:   "/movies",
	})
	require.NoError(t, err)

	assert.Equal(t, "saved-key", env.plugin.lastTested.APIKey, "test must use the stored key when body omits it")
	config, _, loadErr := env.manager.LoadConfig(ctx, "radarr")
	require.NoError(t, loadErr)
	assert.Equal(t, "saved-key", config.APIKey)
	assert.Equal(t, "http://radarr:9999", config.URL)
}

func TestDVRSettingsService_SaveConfig_DisabledSkipsTest(t *testing.T) {
	env := newDVRTestEnv(t)
	env.plugin.testErr = errors.New("server unreachable")

	// Disabling must not require a reachable server.
	err := env.service.SaveConfig(context.Background(), "radarr", DVRConfigInput{
		URL:     "http://radarr:7878",
		APIKey:  "k",
		Enabled: false,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, env.plugin.testedCount)
}

func TestDVRSettingsService_SaveConfig_RequiresURL(t *testing.T) {
	env := newDVRTestEnv(t)
	err := env.service.SaveConfig(context.Background(), "radarr", DVRConfigInput{Enabled: true, APIKey: "k"})
	assert.Error(t, err)
}

func TestDVRSettingsService_GetConfig_IncludesHealthBlock(t *testing.T) {
	env := newDVRTestEnv(t)
	env.seedSaved(t)
	ctx := context.Background()

	// Before any check the health block reports unconfigured (not-yet-checked).
	status, err := env.service.GetConfig(ctx, "radarr")
	require.NoError(t, err)
	assert.Equal(t, plugins.HealthStatusUnconfigured, status.Health.Status)

	env.manager.CheckHealth(ctx, "radarr")
	status, err = env.service.GetConfig(ctx, "radarr")
	require.NoError(t, err)
	assert.Equal(t, plugins.HealthStatusHealthy, status.Health.Status)
	assert.NotNil(t, status.Health.LastCheckedAt)
}

func TestDVRSettingsService_TestConnection_SavedVsBody(t *testing.T) {
	env := newDVRTestEnv(t)
	env.seedSaved(t)
	ctx := context.Background()

	// nil input → saved config.
	require.NoError(t, env.service.TestConnection(ctx, "radarr", nil))
	assert.Equal(t, "http://radarr:7878", env.plugin.lastTested.URL)
	assert.Equal(t, "saved-key", env.plugin.lastTested.APIKey)

	// body input → candidate config; empty body key falls back to stored.
	require.NoError(t, env.service.TestConnection(ctx, "radarr", &DVRConfigInput{URL: "http://candidate:7878"}))
	assert.Equal(t, "http://candidate:7878", env.plugin.lastTested.URL)
	assert.Equal(t, "saved-key", env.plugin.lastTested.APIKey)
}

func TestDVRSettingsService_TestConnection_NotConfigured(t *testing.T) {
	env := newDVRTestEnv(t)

	err := env.service.TestConnection(context.Background(), "radarr", nil)

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeNotConfigured, pluginErr.Code)
}

func TestDVRSettingsService_GetQualityProfilesAndRootFolders(t *testing.T) {
	env := newDVRTestEnv(t)
	env.seedSaved(t)
	env.plugin.profiles = []plugins.QualityProfile{{ID: 1, Name: "HD-1080p"}}
	env.plugin.folders = []plugins.RootFolder{{ID: 1, Path: "/movies"}}
	ctx := context.Background()

	profiles, err := env.service.GetQualityProfiles(ctx, "radarr")
	require.NoError(t, err)
	assert.Equal(t, []plugins.QualityProfile{{ID: 1, Name: "HD-1080p"}}, profiles)

	folders, err := env.service.GetRootFolders(ctx, "radarr")
	require.NoError(t, err)
	assert.Equal(t, []plugins.RootFolder{{ID: 1, Path: "/movies"}}, folders)
}

func TestDVRSettingsService_SonarrParameterization(t *testing.T) {
	// 13-4b AC #3 — sonarr lights up purely by registration/config: the SAME
	// service instance handles sonarr.* keys + secrets round-trip with zero
	// plugin-specific code.
	settings := newFakeDVRSettingsRepo()
	secretsSvc := newFakeDVRSecrets()
	plugin := &fakeDVRPlugin{}
	manager := plugins.NewManager(settings, secretsSvc, &fakeDVRHistoryRepo{}, nil, 0)
	manager.Register("sonarr", func(config plugins.PluginConfig) plugins.DVRPlugin { return plugin })
	service := NewDVRSettingsService(manager, settings, secretsSvc)
	ctx := context.Background()

	err := service.SaveConfig(ctx, "sonarr", DVRConfigInput{
		URL:              "http://sonarr:8989",
		APIKey:           "sonarr-key",
		Enabled:          true,
		QualityProfileID: 6,
		RootFolderPath:   "/tv",
	})
	require.NoError(t, err)

	// sonarr.* keys — not radarr's.
	url, err := settings.GetString(ctx, plugins.SettingKeyURL("sonarr"))
	require.NoError(t, err)
	assert.Equal(t, "http://sonarr:8989", url)
	storedKey, err := secretsSvc.Retrieve(ctx, plugins.SettingKeyAPIKey("sonarr"))
	require.NoError(t, err)
	assert.Equal(t, "sonarr-key", storedKey)
	_, radarrErr := settings.GetString(ctx, plugins.SettingKeyURL("radarr"))
	assert.Error(t, radarrErr, "radarr keys must be untouched")

	status, err := service.GetConfig(ctx, "sonarr")
	require.NoError(t, err)
	assert.Equal(t, "http://sonarr:8989", status.URL)
	assert.Equal(t, int64(6), status.QualityProfileID)
	assert.True(t, status.HasAPIKey)
}

func TestDVRSettingsService_GetQualityProfiles_NotConfigured(t *testing.T) {
	env := newDVRTestEnv(t)

	_, err := env.service.GetQualityProfiles(context.Background(), "radarr")

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeNotConfigured, pluginErr.Code)
}
