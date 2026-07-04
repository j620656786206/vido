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

// Plugin routing by media type: movies → Radarr (13-4a), series → Sonarr
// (13-4b). Both ride the same manager/scheduler/settings.
const (
	dvrMoviePlugin  = "radarr"
	dvrSeriesPlugin = "sonarr"
)

// tvdbNotFoundReason is the zh-TW terminal annotation for the ONE fulfilment
// error retrying cannot fix (13-4b AC #1.2 — title absent from TVDB).
const tvdbNotFoundReason = "此影集不在 TVDB 上，Sonarr 無法搜尋"

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

	if request.MediaType == models.RequestMediaTypeTV {
		// 13-4b AC #4 — whole-series adds via Sonarr.
		s.fulfil(ctx, request, dvrSeriesPlugin, "Sonarr", plugins.DVRPlugin.AddSeries)
		return
	}
	s.fulfil(ctx, request, dvrMoviePlugin, "Radarr", plugins.DVRPlugin.AddMovie)
}

// fulfil runs the shared gate → add → transition flow (13-4a AC #6 /
// 13-4b AC #4): plugin enabled+healthy, config-derived options, synchronous
// add, then the [@contract-v1] searching transition. add is the media-typed
// plugin method (AddMovie / AddSeries).
func (s *FulfilmentService) fulfil(
	ctx context.Context,
	request *models.Request,
	pluginName string,
	displayName string,
	add func(plugins.DVRPlugin, context.Context, int64, plugins.AddOptions) (int64, error),
) {
	if !s.manager.IsConfigured(ctx, pluginName) {
		s.stayPending(ctx, request, displayName+" 未設定", nil)
		return
	}

	health := s.manager.Health(pluginName)
	if health.LastCheckedAt == nil {
		// Boot-edge race: a request can arrive before the scheduler's first
		// sweep — run one lazy check instead of spuriously annotating.
		health = s.manager.CheckHealth(ctx, pluginName)
	}
	if health.Status != plugins.HealthStatusHealthy {
		s.stayPending(ctx, request, displayName+" 連線失敗",
			fmt.Errorf("%s health: %s (%s)", pluginName, health.Status, health.Message))
		return
	}

	profileID, _ := s.settingsRepo.GetInt(ctx, plugins.SettingKeyQualityProfileID(pluginName))
	rootFolder, _ := s.settingsRepo.GetString(ctx, plugins.SettingKeyRootFolderPath(pluginName))
	if profileID == 0 || rootFolder == "" {
		s.stayPending(ctx, request, displayName+" 設定不完整（缺少品質設定檔或根資料夾）", nil)
		return
	}

	client, err := s.manager.GetClient(ctx, pluginName)
	if err != nil {
		s.stayPending(ctx, request, displayName+" 連線失敗", err)
		return
	}

	externalID, err := add(client, ctx, request.TMDbID, plugins.AddOptions{
		QualityProfileID: int64(profileID),
		RootFolderPath:   rootFolder,
		SearchNow:        true,
	})
	if err != nil {
		var pluginErr *plugins.PluginError
		if errors.As(err, &pluginErr) && pluginErr.Code == plugins.ErrCodeTVDBNotFound {
			// The ONE terminal fulfilment error (13-4b AC #1.2) — an honest
			// failed row, never a stranded pending.
			s.failTerminally(ctx, request, tvdbNotFoundReason, err)
			return
		}
		s.stayPending(ctx, request, addFailureReason(displayName, err), err)
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

	slog.Info("Request routed to DVR plugin",
		"request_id", request.ID, "tmdb_id", request.TMDbID,
		"media_type", request.MediaType, "plugin", pluginName, "external_id", externalID)
}

// failTerminally writes the terminal failed transition (13-4b AC #1.2 —
// currently only DVR_TVDB_NOT_FOUND). fulfilment_source and external_id stay
// NULL: nothing claimed the row.
func (s *FulfilmentService) failTerminally(ctx context.Context, request *models.Request, reason string, cause error) {
	slog.Error("Request fulfilment failed terminally",
		"request_id", request.ID, "tmdb_id", request.TMDbID,
		"media_type", request.MediaType, "reason", reason, "error", cause)

	status := models.RequestStatusFailed
	errMsg := models.NewNullString(reason)
	updatedAt, err := s.requestRepo.UpdateFulfilment(ctx, request.ID, status,
		models.NullString{}, models.NullString{}, errMsg)
	if err != nil {
		slog.Error("Failed to persist terminal fulfilment failure",
			"request_id", request.ID, "error", err)
		// Keep the response consistent with the row (still pending in DB).
		request.ErrorMessage = errMsg
		return
	}

	request.Status = status
	request.FulfilmentSource = models.NullString{}
	request.ExternalID = models.NullString{}
	request.ErrorMessage = errMsg
	request.UpdatedAt = updatedAt
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

// addFailureReason maps an AddMovie/AddSeries error to its zh-TW row annotation.
func addFailureReason(displayName string, err error) string {
	var pluginErr *plugins.PluginError
	if errors.As(err, &pluginErr) && pluginErr.Code == plugins.ErrCodeAddFailed {
		return displayName + " 新增失敗"
	}
	return displayName + " 連線失敗"
}
