package plugins

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

// --- test doubles -----------------------------------------------------------

type mockSettingsRepo struct {
	mu      sync.Mutex
	strings map[string]string
	bools   map[string]bool
	ints    map[string]int
}

func newMockSettingsRepo() *mockSettingsRepo {
	return &mockSettingsRepo{strings: map[string]string{}, bools: map[string]bool{}, ints: map[string]int{}}
}

func (m *mockSettingsRepo) Set(ctx context.Context, setting *models.Setting) error { return nil }
func (m *mockSettingsRepo) Get(ctx context.Context, key string) (*models.Setting, error) {
	return nil, errors.New("not implemented")
}
func (m *mockSettingsRepo) GetAll(ctx context.Context) ([]models.Setting, error) { return nil, nil }
func (m *mockSettingsRepo) Delete(ctx context.Context, key string) error         { return nil }
func (m *mockSettingsRepo) GetString(ctx context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.strings[key]
	if !ok {
		return "", fmt.Errorf("setting %s not found", key)
	}
	return v, nil
}
func (m *mockSettingsRepo) GetInt(ctx context.Context, key string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.ints[key]
	if !ok {
		return 0, fmt.Errorf("setting %s not found", key)
	}
	return v, nil
}
func (m *mockSettingsRepo) GetBool(ctx context.Context, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.bools[key]
	if !ok {
		return false, fmt.Errorf("setting %s not found", key)
	}
	return v, nil
}
func (m *mockSettingsRepo) SetString(ctx context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strings[key] = value
	return nil
}
func (m *mockSettingsRepo) SetInt(ctx context.Context, key string, value int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ints[key] = value
	return nil
}
func (m *mockSettingsRepo) SetBool(ctx context.Context, key string, value bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bools[key] = value
	return nil
}

type mockSecretsService struct {
	mu      sync.Mutex
	secrets map[string]string
}

func newMockSecretsService() *mockSecretsService {
	return &mockSecretsService{secrets: map[string]string{}}
}

func (m *mockSecretsService) Store(ctx context.Context, name string, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.secrets[name] = value
	return nil
}
func (m *mockSecretsService) Retrieve(ctx context.Context, name string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.secrets[name]
	if !ok {
		return "", errors.New("secret not found")
	}
	return v, nil
}
func (m *mockSecretsService) Delete(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.secrets, name)
	return nil
}
func (m *mockSecretsService) Exists(ctx context.Context, name string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.secrets[name]
	return ok, nil
}
func (m *mockSecretsService) List(ctx context.Context) ([]string, error) { return nil, nil }

type mockHistoryRepo struct {
	mu     sync.Mutex
	events []models.ConnectionEvent
}

func (m *mockHistoryRepo) Create(ctx context.Context, event *models.ConnectionEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, *event)
	return nil
}
func (m *mockHistoryRepo) GetHistory(ctx context.Context, service string, limit int) ([]models.ConnectionEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.events, nil
}

func (m *mockHistoryRepo) recorded() []models.ConnectionEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.ConnectionEvent, len(m.events))
	copy(out, m.events)
	return out
}

// stubPlugin is a controllable DVRPlugin double.
type stubPlugin struct {
	mu       sync.Mutex
	testErr  error
	tested   int
	instance int // distinguishes rebuilt instances
}

func (s *stubPlugin) Name() string { return "radarr" }
func (s *stubPlugin) TestConnection(ctx context.Context, config PluginConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tested++
	return s.testErr
}
func (s *stubPlugin) AddMovie(ctx context.Context, tmdbID int64, opts AddOptions) (int64, error) {
	return 1, nil
}
func (s *stubPlugin) AddSeries(ctx context.Context, tmdbID int64, opts AddOptions) (int64, error) {
	return 0, &PluginError{Code: ErrCodeNotSupported, Message: "movie-only"}
}
func (s *stubPlugin) GetQueue(ctx context.Context) ([]QueueItem, error) { return nil, nil }

func (s *stubPlugin) setTestErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.testErr = err
}

func (s *stubPlugin) testedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tested
}

// --- helpers ----------------------------------------------------------------

func configuredManager(t *testing.T) (*Manager, *mockSettingsRepo, *mockSecretsService, *mockHistoryRepo, *stubPlugin) {
	t.Helper()
	settings := newMockSettingsRepo()
	secretsSvc := newMockSecretsService()
	history := &mockHistoryRepo{}
	stub := &stubPlugin{}

	require.NoError(t, settings.SetString(context.Background(), SettingKeyURL("radarr"), "http://radarr:7878"))
	require.NoError(t, settings.SetBool(context.Background(), SettingKeyEnabled("radarr"), true))
	require.NoError(t, secretsSvc.Store(context.Background(), SettingKeyAPIKey("radarr"), "key-1"))

	mgr := NewManager(settings, secretsSvc, history, nil, 0)
	instances := 0
	mgr.Register("radarr", func(config PluginConfig) DVRPlugin {
		instances++
		stub.mu.Lock()
		stub.instance = instances
		stub.mu.Unlock()
		return stub
	})
	return mgr, settings, secretsSvc, history, stub
}

// --- tests ------------------------------------------------------------------

func TestManager_DefaultInterval(t *testing.T) {
	mgr := NewManager(newMockSettingsRepo(), newMockSecretsService(), &mockHistoryRepo{}, nil, 0)
	assert.Equal(t, 60*time.Second, mgr.interval)

	custom := NewManager(newMockSettingsRepo(), newMockSecretsService(), &mockHistoryRepo{}, nil, 5*time.Second)
	assert.Equal(t, 5*time.Second, custom.interval)
}

func TestManager_GetClient_UnknownPlugin(t *testing.T) {
	mgr := NewManager(newMockSettingsRepo(), newMockSecretsService(), &mockHistoryRepo{}, nil, 0)
	_, err := mgr.GetClient(context.Background(), "sonarr")

	var pluginErr *PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, ErrCodePluginInitFailed, pluginErr.Code)
}

func TestManager_GetClient_NotConfigured(t *testing.T) {
	mgr := NewManager(newMockSettingsRepo(), newMockSecretsService(), &mockHistoryRepo{}, nil, 0)
	mgr.Register("radarr", func(config PluginConfig) DVRPlugin { return &stubPlugin{} })

	_, err := mgr.GetClient(context.Background(), "radarr")

	var pluginErr *PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, ErrCodeNotConfigured, pluginErr.Code)
}

func TestManager_GetClient_DisabledIsNotConfigured(t *testing.T) {
	mgr, settings, _, _, _ := configuredManager(t)
	require.NoError(t, settings.SetBool(context.Background(), SettingKeyEnabled("radarr"), false))

	_, err := mgr.GetClient(context.Background(), "radarr")

	var pluginErr *PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, ErrCodeNotConfigured, pluginErr.Code)
}

func TestManager_GetClient_CachesByFingerprint(t *testing.T) {
	mgr, settings, secretsSvc, _, stub := configuredManager(t)
	ctx := context.Background()

	c1, err := mgr.GetClient(ctx, "radarr")
	require.NoError(t, err)
	c2, err := mgr.GetClient(ctx, "radarr")
	require.NoError(t, err)
	assert.Same(t, c1, c2, "same fingerprint must reuse the cached client")
	assert.Equal(t, 1, stub.instance, "factory must run once for an unchanged config")

	// Config change → fingerprint change → client rebuild.
	require.NoError(t, secretsSvc.Store(ctx, SettingKeyAPIKey("radarr"), "key-2"))
	_, err = mgr.GetClient(ctx, "radarr")
	require.NoError(t, err)
	assert.Equal(t, 2, stub.instance, "changed fingerprint must rebuild the client")

	// URL change too.
	require.NoError(t, settings.SetString(ctx, SettingKeyURL("radarr"), "http://radarr:8989"))
	_, err = mgr.GetClient(ctx, "radarr")
	require.NoError(t, err)
	assert.Equal(t, 3, stub.instance)
}

func TestManager_Health_BeforeFirstCheck(t *testing.T) {
	mgr, _, _, _, _ := configuredManager(t)
	h := mgr.Health("radarr")
	assert.Equal(t, HealthStatusUnconfigured, h.Status)
	assert.Nil(t, h.LastCheckedAt)
}

func TestManager_CheckHealth_Unconfigured(t *testing.T) {
	mgr := NewManager(newMockSettingsRepo(), newMockSecretsService(), &mockHistoryRepo{}, nil, 0)
	mgr.Register("radarr", func(config PluginConfig) DVRPlugin { return &stubPlugin{} })

	h := mgr.CheckHealth(context.Background(), "radarr")

	assert.Equal(t, HealthStatusUnconfigured, h.Status)
	require.NotNil(t, h.LastCheckedAt)
}

func TestManager_CheckHealth_TransitionsWriteHistory(t *testing.T) {
	mgr, _, _, history, stub := configuredManager(t)
	ctx := context.Background()

	// unconfigured (initial) → healthy = connected
	h := mgr.CheckHealth(ctx, "radarr")
	assert.Equal(t, HealthStatusHealthy, h.Status)
	events := history.recorded()
	require.Len(t, events, 1)
	assert.Equal(t, models.EventConnected, events[0].EventType)
	assert.Equal(t, "radarr", events[0].Service)
	assert.Equal(t, HealthStatusHealthy, events[0].Status)

	// healthy → healthy = steady state, no event
	mgr.CheckHealth(ctx, "radarr")
	require.Len(t, history.recorded(), 1)

	// healthy → unhealthy = disconnected, message carries PLUGIN_HEALTH_CHECK_FAILED
	stub.setTestErr(errors.New("dial tcp: connection refused"))
	h = mgr.CheckHealth(ctx, "radarr")
	assert.Equal(t, HealthStatusUnhealthy, h.Status)
	assert.Contains(t, h.Message, ErrCodePluginHealthCheckFailed)
	events = history.recorded()
	require.Len(t, events, 2)
	assert.Equal(t, models.EventDisconnected, events[1].EventType)

	// unhealthy → unhealthy = steady state, no event
	mgr.CheckHealth(ctx, "radarr")
	require.Len(t, history.recorded(), 2)

	// unhealthy → healthy = recovered
	stub.setTestErr(nil)
	h = mgr.CheckHealth(ctx, "radarr")
	assert.Equal(t, HealthStatusHealthy, h.Status)
	events = history.recorded()
	require.Len(t, events, 3)
	assert.Equal(t, models.EventRecovered, events[2].EventType)
}

func TestManager_SchedulerLifecycle(t *testing.T) {
	settings := newMockSettingsRepo()
	secretsSvc := newMockSecretsService()
	history := &mockHistoryRepo{}
	stub := &stubPlugin{}

	ctx := context.Background()
	require.NoError(t, settings.SetString(ctx, SettingKeyURL("radarr"), "http://radarr:7878"))
	require.NoError(t, settings.SetBool(ctx, SettingKeyEnabled("radarr"), true))
	require.NoError(t, secretsSvc.Store(ctx, SettingKeyAPIKey("radarr"), "key-1"))

	mgr := NewManager(settings, secretsSvc, history, nil, 20*time.Millisecond)
	mgr.Register("radarr", func(config PluginConfig) DVRPlugin { return stub })

	require.NoError(t, mgr.Start(ctx))
	assert.True(t, mgr.IsRunning())
	require.NoError(t, mgr.Start(ctx), "double Start must be a no-op")

	// Immediate startup check + at least one tick.
	assert.Eventually(t, func() bool { return stub.testedCount() >= 2 }, 2*time.Second, 10*time.Millisecond)

	mgr.Stop()
	assert.False(t, mgr.IsRunning())
	mgr.Stop() // double Stop must not panic

	after := stub.testedCount()
	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, after, stub.testedCount(), "no checks may run after Stop")
}

func TestManager_TestConfig(t *testing.T) {
	mgr, _, _, _, stub := configuredManager(t)
	ctx := context.Background()

	// Probes the GIVEN config on a throwaway client, without touching saved state.
	err := mgr.TestConfig(ctx, "radarr", PluginConfig{URL: "http://candidate:7878", APIKey: "candidate-key"})
	assert.NoError(t, err)
	assert.Equal(t, 1, stub.testedCount())

	stub.setTestErr(errors.New("boom"))
	err = mgr.TestConfig(ctx, "radarr", PluginConfig{URL: "http://candidate:7878", APIKey: "candidate-key"})
	assert.Error(t, err)

	// Incomplete candidate config → DVR_NOT_CONFIGURED.
	var pluginErr *PluginError
	err = mgr.TestConfig(ctx, "radarr", PluginConfig{URL: "", APIKey: "k"})
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, ErrCodeNotConfigured, pluginErr.Code)

	// Unknown plugin → PLUGIN_INIT_FAILED.
	err = mgr.TestConfig(ctx, "sonarr", PluginConfig{URL: "http://x", APIKey: "k"})
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, ErrCodePluginInitFailed, pluginErr.Code)
}

func TestSettingKeys(t *testing.T) {
	assert.Equal(t, "radarr.url", SettingKeyURL("radarr"))
	assert.Equal(t, "radarr.enabled", SettingKeyEnabled("radarr"))
	assert.Equal(t, "radarr.api_key", SettingKeyAPIKey("radarr"))
	assert.Equal(t, "radarr.quality_profile_id", SettingKeyQualityProfileID("radarr"))
	assert.Equal(t, "radarr.root_folder_path", SettingKeyRootFolderPath("radarr"))
}
