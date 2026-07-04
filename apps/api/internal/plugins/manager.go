package plugins

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/secrets"
)

// Plugin health statuses surfaced by the settings GET health block (AC #4).
// The hardcoded 5-service ServicesHealth model is deliberately NOT extended —
// plugin health is self-contained here (13-4a ruling).
const (
	HealthStatusHealthy      = "healthy"
	HealthStatusUnhealthy    = "unhealthy"
	HealthStatusUnconfigured = "unconfigured"
)

// defaultHealthCheckInterval is the §7 default plugin health-check cadence.
const defaultHealthCheckInterval = 60 * time.Second

// Per-plugin settings keys follow the qBittorrent precedent
// (services.SettingQB* — "{plugin}.{field}"). Defined here so the plugins
// manager and the services-layer DVR settings service share one source of
// truth without a services import (Rule 19: services → plugins, never back).
func SettingKeyURL(plugin string) string     { return plugin + ".url" }
func SettingKeyEnabled(plugin string) string { return plugin + ".enabled" }
func SettingKeyAPIKey(plugin string) string  { return plugin + ".api_key" }
func SettingKeyQualityProfileID(plugin string) string {
	return plugin + ".quality_profile_id"
}
func SettingKeyRootFolderPath(plugin string) string { return plugin + ".root_folder_path" }

// PluginHealth is the live health block shape consumed by the settings GET
// endpoint (AC #4) and the 13-6 settings UI.
type PluginHealth struct {
	Status        string     `json:"status"`
	LastCheckedAt *time.Time `json:"last_checked_at"`
	Message       string     `json:"message"`
}

// ClientFactory builds a DVRPlugin client for a loaded config.
type ClientFactory func(config PluginConfig) DVRPlugin

// Manager owns plugin registration, per-plugin config loading (settings +
// secrets), fingerprint-cached clients (Rule 14), health state, and the 60s
// health-check scheduler (retry/scheduler.go lifecycle).
type Manager struct {
	settingsRepo repository.SettingsRepositoryInterface
	secrets      secrets.SecretsServiceInterface
	historyRepo  repository.ConnectionHistoryRepositoryInterface
	logger       *slog.Logger
	interval     time.Duration

	mu           sync.Mutex
	factories    map[string]ClientFactory
	clients      map[string]DVRPlugin
	fingerprints map[string]string
	health       map[string]PluginHealth

	runMu   sync.Mutex
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewManager creates a plugin manager. interval <= 0 selects the §7 default (60s).
func NewManager(
	settingsRepo repository.SettingsRepositoryInterface,
	secretsService secrets.SecretsServiceInterface,
	historyRepo repository.ConnectionHistoryRepositoryInterface,
	logger *slog.Logger,
	interval time.Duration,
) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	if interval <= 0 {
		interval = defaultHealthCheckInterval
	}
	return &Manager{
		settingsRepo: settingsRepo,
		secrets:      secretsService,
		historyRepo:  historyRepo,
		logger:       logger,
		interval:     interval,
		factories:    map[string]ClientFactory{},
		clients:      map[string]DVRPlugin{},
		fingerprints: map[string]string{},
		health:       map[string]PluginHealth{},
	}
}

// Register adds a plugin factory under its name (startup registration, §7).
func (m *Manager) Register(name string, factory ClientFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.factories[name] = factory
}

// RegisteredPlugins returns the sorted names of all registered plugins.
func (m *Manager) RegisteredPlugins() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.factories))
	for name := range m.factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// LoadConfig reads a plugin's config from settings (url, enabled) + secrets
// (api_key). Missing keys resolve to zero values, mirroring the qBittorrent
// GetConfig tolerance for a partially-configured plugin.
func (m *Manager) LoadConfig(ctx context.Context, name string) (PluginConfig, bool, error) {
	url, _ := m.settingsRepo.GetString(ctx, SettingKeyURL(name))
	enabled, _ := m.settingsRepo.GetBool(ctx, SettingKeyEnabled(name))

	var apiKey string
	exists, _ := m.secrets.Exists(ctx, SettingKeyAPIKey(name))
	if exists {
		var err error
		apiKey, err = m.secrets.Retrieve(ctx, SettingKeyAPIKey(name))
		if err != nil {
			return PluginConfig{}, false, fmt.Errorf("decrypt %s api key: %w", name, err)
		}
	}

	return PluginConfig{URL: url, APIKey: apiKey}, enabled, nil
}

// IsConfigured reports whether the plugin is enabled with a complete config.
func (m *Manager) IsConfigured(ctx context.Context, name string) bool {
	config, enabled, err := m.LoadConfig(ctx, name)
	return err == nil && enabled && config.URL != "" && config.APIKey != ""
}

// GetClient returns the plugin client for the current config, rebuilding it
// only when the config fingerprint (URL|APIKey) changes (download_service
// getClient pattern, Rule 14).
func (m *Manager) GetClient(ctx context.Context, name string) (DVRPlugin, error) {
	m.mu.Lock()
	factory, registered := m.factories[name]
	m.mu.Unlock()
	if !registered {
		return nil, &PluginError{
			Code:    ErrCodePluginInitFailed,
			Message: fmt.Sprintf("plugin %q is not registered", name),
		}
	}

	config, enabled, err := m.LoadConfig(ctx, name)
	if err != nil {
		return nil, &PluginError{Code: ErrCodePluginInitFailed, Message: "load plugin config", Cause: err}
	}
	if !enabled || config.URL == "" || config.APIKey == "" {
		return nil, &PluginError{
			Code:    ErrCodeNotConfigured,
			Message: fmt.Sprintf("plugin %q is not configured or disabled", name),
		}
	}

	fingerprint := config.URL + "|" + config.APIKey

	m.mu.Lock()
	defer m.mu.Unlock()
	if client, ok := m.clients[name]; ok && m.fingerprints[name] == fingerprint {
		return client, nil
	}
	client := factory(config)
	m.clients[name] = client
	m.fingerprints[name] = fingerprint
	return client, nil
}

// TestConfig probes an ARBITRARY (possibly unsaved) config against a fresh
// throwaway client — the settings PUT test-before-save guard (AC #4, §7
// "must pass before save").
func (m *Manager) TestConfig(ctx context.Context, name string, config PluginConfig) error {
	m.mu.Lock()
	factory, registered := m.factories[name]
	m.mu.Unlock()
	if !registered {
		return &PluginError{
			Code:    ErrCodePluginInitFailed,
			Message: fmt.Sprintf("plugin %q is not registered", name),
		}
	}
	if config.URL == "" || config.APIKey == "" {
		return &PluginError{
			Code:    ErrCodeNotConfigured,
			Message: "url and api key are required",
		}
	}
	return factory(config).TestConnection(ctx, config)
}

// Health returns the last known health for a plugin. Before the first check
// it reports unconfigured with a nil timestamp.
func (m *Manager) Health(name string) PluginHealth {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h, ok := m.health[name]; ok {
		return h
	}
	return PluginHealth{Status: HealthStatusUnconfigured, Message: "health not checked yet"}
}

// CheckHealth runs an immediate health check for one plugin, updates its
// health state, and records healthy↔unhealthy transitions to connection
// history (monitor.go transition pattern).
func (m *Manager) CheckHealth(ctx context.Context, name string) PluginHealth {
	now := time.Now()
	next := PluginHealth{LastCheckedAt: &now}

	if !m.IsConfigured(ctx, name) {
		next.Status = HealthStatusUnconfigured
		next.Message = "plugin not configured"
		m.storeHealth(ctx, name, next)
		return next
	}

	client, err := m.GetClient(ctx, name)
	if err != nil {
		next.Status = HealthStatusUnhealthy
		next.Message = (&PluginError{Code: ErrCodePluginInitFailed, Message: "client init failed", Cause: err}).Error()
		m.storeHealth(ctx, name, next)
		return next
	}

	config, _, _ := m.LoadConfig(ctx, name)
	if err := client.TestConnection(ctx, config); err != nil {
		next.Status = HealthStatusUnhealthy
		next.Message = (&PluginError{
			Code:    ErrCodePluginHealthCheckFailed,
			Message: fmt.Sprintf("%s health check failed", name),
			Cause:   err,
		}).Error()
	} else {
		next.Status = HealthStatusHealthy
	}

	m.storeHealth(ctx, name, next)
	return next
}

// storeHealth swaps in the new health state and records the transition.
func (m *Manager) storeHealth(ctx context.Context, name string, next PluginHealth) {
	m.mu.Lock()
	prev := m.health[name].Status
	m.health[name] = next
	m.mu.Unlock()

	m.recordTransition(ctx, name, prev, next)
}

// recordTransition writes healthy↔unhealthy transition events to the
// connection_history table so GET /health/services/:service/history works
// for radarr/sonarr (ValidServiceNames extended in this story).
func (m *Manager) recordTransition(ctx context.Context, name, prev string, next PluginHealth) {
	if m.historyRepo == nil || prev == next.Status {
		return
	}

	var eventType models.ConnectionEventType
	switch {
	case next.Status == HealthStatusHealthy && prev == HealthStatusUnhealthy:
		eventType = models.EventRecovered
	case next.Status == HealthStatusHealthy:
		eventType = models.EventConnected
	case next.Status == HealthStatusUnhealthy:
		eventType = models.EventDisconnected
	default:
		// * → unconfigured is a config removal, not a connection event.
		return
	}

	event := &models.ConnectionEvent{
		ID:        uuid.New().String(),
		Service:   name,
		EventType: eventType,
		Status:    next.Status,
		Message:   next.Message,
		CreatedAt: time.Now(),
	}

	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := m.historyRepo.Create(writeCtx, event); err != nil {
		m.logger.Error("Failed to record plugin connection event",
			"plugin", name, "event_type", eventType, "error", err)
	}
}

// checkAll checks every registered plugin once.
func (m *Manager) checkAll(ctx context.Context) {
	for _, name := range m.RegisteredPlugins() {
		m.CheckHealth(ctx, name)
	}
}

// Start begins the health-check scheduler (retry/scheduler.go lifecycle:
// stopCh + WaitGroup + idempotent Start/Stop). An immediate sweep runs first
// so health state is available before the first interval elapses.
func (m *Manager) Start(ctx context.Context) error {
	m.runMu.Lock()
	if m.running {
		m.runMu.Unlock()
		return nil
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.runMu.Unlock()

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.run(ctx)
	}()

	m.logger.Info("Plugin health scheduler started",
		"interval", m.interval, "plugins", m.RegisteredPlugins())
	return nil
}

// Stop gracefully stops the scheduler.
func (m *Manager) Stop() {
	m.runMu.Lock()
	if !m.running {
		m.runMu.Unlock()
		return
	}
	m.running = false
	close(m.stopCh)
	m.runMu.Unlock()

	m.wg.Wait()
	m.logger.Info("Plugin health scheduler stopped")
}

// IsRunning reports whether the scheduler is active.
func (m *Manager) IsRunning() bool {
	m.runMu.Lock()
	defer m.runMu.Unlock()
	return m.running
}

// run is the scheduler loop.
func (m *Manager) run(ctx context.Context) {
	m.checkAll(ctx)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkAll(ctx)
		}
	}
}
