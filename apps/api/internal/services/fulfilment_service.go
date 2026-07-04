// Package services — FulfilmentService (Story 13-4a, AC #6).
//
// Best-effort synchronous movie fulfilment over the DVR plugin manager. The
// transition semantics (pending→searching + external_id + fulfilment_source
// writes; failures stay pending with a zh-TW error_message) carry
// [@contract-v1] — consumers 13-3a (reconcile/retry) and 13-5.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/plugins"
	"github.com/vido/api/internal/repository"
)

// dvrMoviePlugin is the plugin that fulfils movie requests. Series routing
// (sonarr) lands in 13-4b via the same manager.
const dvrMoviePlugin = "radarr"

// FulfilmentServiceInterface is the optional RequestService dependency
// (nil-safe — 13-1a create behavior is preserved exactly when absent).
type FulfilmentServiceInterface interface {
	// FulfilRequest attempts best-effort fulfilment of a request row. It
	// never fails the caller: on any degradation the row STAYS pending with
	// a zh-TW error_message and the cause is slog-logged (Rule 13 — recorded,
	// never swallowed). The request is mutated in place and persisted, so a
	// create response carries the resulting state.
	FulfilRequest(ctx context.Context, request *models.Request)
}

// FulfilmentService implements FulfilmentServiceInterface over the plugin
// manager. Stranded-pending retry is EXPLICITLY 13-3a's reconcile loop —
// no retry loop lives here (story scope wall).
type FulfilmentService struct {
	manager      *plugins.Manager
	settingsRepo repository.SettingsRepositoryInterface
	requestRepo  repository.RequestRepositoryInterface
}

// Compile-time interface verification.
var _ FulfilmentServiceInterface = (*FulfilmentService)(nil)

// NewFulfilmentService creates a new FulfilmentService.
func NewFulfilmentService(
	manager *plugins.Manager,
	settingsRepo repository.SettingsRepositoryInterface,
	requestRepo repository.RequestRepositoryInterface,
) *FulfilmentService {
	return &FulfilmentService{manager: manager, settingsRepo: settingsRepo, requestRepo: requestRepo}
}

// FulfilRequest routes a request to its fulfilment path by media type.
func (s *FulfilmentService) FulfilRequest(ctx context.Context, request *models.Request) {
	if request == nil {
		return
	}

	if request.MediaType != models.RequestMediaTypeMovie {
		// TV fulfilment needs the Sonarr plugin (13-4b); the reason string
		// keeps the row honest until 13-3a's reconcile retries it.
		s.stayPending(ctx, request, "Sonarr 尚未支援（13-4b）", nil)
		return
	}
	s.fulfilMovie(ctx, request)
}

// fulfilMovie runs the AC #6 movie branch: gate on Radarr enabled+healthy,
// then a synchronous AddMovie with config-derived options.
func (s *FulfilmentService) fulfilMovie(ctx context.Context, request *models.Request) {
	if !s.manager.IsConfigured(ctx, dvrMoviePlugin) {
		s.stayPending(ctx, request, "Radarr 未設定", nil)
		return
	}

	health := s.manager.Health(dvrMoviePlugin)
	if health.LastCheckedAt == nil {
		// Boot-edge race: a request can arrive before the scheduler's first
		// sweep — run one lazy check instead of spuriously annotating.
		health = s.manager.CheckHealth(ctx, dvrMoviePlugin)
	}
	if health.Status != plugins.HealthStatusHealthy {
		s.stayPending(ctx, request, "Radarr 連線失敗", fmt.Errorf("radarr health: %s (%s)", health.Status, health.Message))
		return
	}

	profileID, _ := s.settingsRepo.GetInt(ctx, plugins.SettingKeyQualityProfileID(dvrMoviePlugin))
	rootFolder, _ := s.settingsRepo.GetString(ctx, plugins.SettingKeyRootFolderPath(dvrMoviePlugin))
	if profileID == 0 || rootFolder == "" {
		s.stayPending(ctx, request, "Radarr 設定不完整（缺少品質設定檔或根資料夾）", nil)
		return
	}

	client, err := s.manager.GetClient(ctx, dvrMoviePlugin)
	if err != nil {
		s.stayPending(ctx, request, "Radarr 連線失敗", err)
		return
	}

	externalID, err := client.AddMovie(ctx, request.TMDbID, plugins.AddOptions{
		QualityProfileID: int64(profileID),
		RootFolderPath:   rootFolder,
		SearchNow:        true,
	})
	if err != nil {
		s.stayPending(ctx, request, addMovieFailureReason(err), err)
		return
	}

	// [@contract-v1] success transition — the create response carries
	// status "searching" (a value inside the 13-1a 5-value enum, no bump).
	status := models.RequestStatusSearching
	source := models.NewNullString(models.RequestFulfilmentSourceArr)
	external := models.NewNullString(strconv.FormatInt(externalID, 10))
	updatedAt, err := s.requestRepo.UpdateFulfilment(ctx, request.ID, status, source, external, models.NullString{})
	if err != nil {
		// The DB row is still pending — keep the response consistent with
		// the row rather than claiming a transition that was never stored.
		slog.Error("Fulfilment transition write failed; row stays pending",
			"request_id", request.ID, "tmdb_id", request.TMDbID, "radarr_id", externalID, "error", err)
		request.ErrorMessage = models.NewNullString("Radarr 已接受請求，狀態寫入失敗")
		return
	}

	request.Status = status
	request.FulfilmentSource = source
	request.ExternalID = external
	request.ErrorMessage = models.NullString{}
	request.UpdatedAt = updatedAt

	slog.Info("Movie request routed to Radarr",
		"request_id", request.ID, "tmdb_id", request.TMDbID, "radarr_id", externalID)
}

// stayPending annotates a request with a zh-TW degradation reason, keeping
// its status untouched (graceful degradation — fulfilment never fails the
// request). The cause, when present, is logged per Rule 13.
func (s *FulfilmentService) stayPending(ctx context.Context, request *models.Request, reason string, cause error) {
	request.ErrorMessage = models.NewNullString(reason)

	if cause != nil {
		slog.Error("Request fulfilment degraded; row stays pending",
			"request_id", request.ID, "tmdb_id", request.TMDbID,
			"media_type", request.MediaType, "reason", reason, "error", cause)
	} else {
		slog.Info("Request fulfilment skipped; row stays pending",
			"request_id", request.ID, "tmdb_id", request.TMDbID,
			"media_type", request.MediaType, "reason", reason)
	}

	updatedAt, err := s.requestRepo.UpdateFulfilment(ctx, request.ID, request.Status,
		request.FulfilmentSource, request.ExternalID, request.ErrorMessage)
	if err != nil {
		slog.Error("Failed to persist fulfilment annotation",
			"request_id", request.ID, "error", err)
		return
	}
	request.UpdatedAt = updatedAt
}

// addMovieFailureReason maps an AddMovie error to its zh-TW row annotation.
func addMovieFailureReason(err error) string {
	var pluginErr *plugins.PluginError
	if errors.As(err, &pluginErr) && pluginErr.Code == plugins.ErrCodeAddFailed {
		return "Radarr 新增失敗"
	}
	return "Radarr 連線失敗"
}
