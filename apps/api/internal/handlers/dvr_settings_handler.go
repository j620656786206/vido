// Package handlers — DVRSettingsHandler (Story 13-4a, AC #4).
//
// Settings triad + passthrough for *arr DVR plugins, mirroring the
// qbittorrent_handler shapes. Routes are registered statically per plugin
// name (parameterized internally) so 13-4b's sonarr adds one string, zero
// handler duplication. The endpoint shapes carry [@contract-v1] (consumer 13-6).
package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vido/api/internal/plugins"
	"github.com/vido/api/internal/services"
)

// DVRSettingsHandler handles HTTP requests for DVR plugin settings.
type DVRSettingsHandler struct {
	service     services.DVRSettingsServiceInterface
	pluginNames []string
}

// NewDVRSettingsHandler creates a handler serving the given plugin names
// (13-4a: "radarr"; 13-4b appends "sonarr").
func NewDVRSettingsHandler(service services.DVRSettingsServiceInterface, pluginNames ...string) *DVRSettingsHandler {
	return &DVRSettingsHandler{service: service, pluginNames: pluginNames}
}

// RegisterRoutes registers the settings triad + passthrough routes for every
// served plugin. Static paths coexist with settingsHandler's /settings/:key
// param route (the qbittorrent precedent).
func (h *DVRSettingsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	for _, name := range h.pluginNames {
		plugin := name // capture per iteration
		group := rg.Group("/settings/" + plugin)
		{
			group.GET("", func(c *gin.Context) { h.getConfig(c, plugin) })
			group.PUT("", func(c *gin.Context) { h.saveConfig(c, plugin) })
			group.POST("/test", func(c *gin.Context) { h.testConnection(c, plugin) })
			group.GET("/quality-profiles", func(c *gin.Context) { h.getQualityProfiles(c, plugin) })
			group.GET("/root-folders", func(c *gin.Context) { h.getRootFolders(c, plugin) })
		}
	}
}

// getConfig handles GET /api/v1/settings/{plugin}
// @Summary Get DVR plugin configuration (sans API key) + live health block
// @Tags dvr-settings
// @Produce json
// @Success 200 {object} APIResponse{data=services.DVRConfigStatus}
// @Failure 500 {object} APIResponse "INTERNAL_ERROR"
// @Router /api/v1/settings/radarr [get]
func (h *DVRSettingsHandler) getConfig(c *gin.Context, plugin string) {
	status, err := h.service.GetConfig(c.Request.Context(), plugin)
	if err != nil {
		slog.Error("Failed to get DVR plugin config", "plugin", plugin, "error", err)
		InternalServerError(c, "無法載入 "+pluginDisplayName(plugin)+" 設定")
		return
	}
	SuccessResponse(c, status)
}

// saveConfig handles PUT /api/v1/settings/{plugin}
// @Summary Save DVR plugin configuration (server-side test-before-save guard)
// @Tags dvr-settings
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse "VALIDATION_INVALID_FORMAT | DVR_NOT_CONFIGURED"
// @Failure 409 {object} APIResponse "DVR_TEST_FAILED — connection test failed, config not saved"
// @Router /api/v1/settings/radarr [put]
func (h *DVRSettingsHandler) saveConfig(c *gin.Context, plugin string) {
	var input services.DVRConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		ValidationError(c, "Invalid request body: "+err.Error())
		return
	}
	if input.URL == "" {
		ValidationError(c, "url is required")
		return
	}

	if err := h.service.SaveConfig(c.Request.Context(), plugin, input); err != nil {
		h.respondError(c, plugin, err, "儲存 "+pluginDisplayName(plugin)+" 設定失敗")
		return
	}
	SuccessResponse(c, gin.H{"message": "Configuration saved"})
}

// dvrTestRequest is the optional POST /test body — when present, tests the
// given config without saving (qBT TestConnectionWithConfig pattern).
type dvrTestRequest struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

// testConnection handles POST /api/v1/settings/{plugin}/test
// @Summary Test DVR plugin connection (body config if provided, else saved)
// @Tags dvr-settings
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse "DVR_CONNECTION_FAILED | DVR_AUTH_FAILED | DVR_TIMEOUT | DVR_NOT_CONFIGURED"
// @Router /api/v1/settings/radarr/test [post]
func (h *DVRSettingsHandler) testConnection(c *gin.Context, plugin string) {
	var input *services.DVRConfigInput
	var req dvrTestRequest
	if c.ShouldBindJSON(&req) == nil && req.URL != "" {
		input = &services.DVRConfigInput{URL: req.URL, APIKey: req.APIKey}
	}

	if err := h.service.TestConnection(c.Request.Context(), plugin, input); err != nil {
		h.respondError(c, plugin, err, "無法連線到 "+pluginDisplayName(plugin))
		return
	}
	SuccessResponse(c, gin.H{"message": "Connection successful"})
}

// getQualityProfiles handles GET /api/v1/settings/{plugin}/quality-profiles
// @Summary List the DVR plugin's quality profiles (passthrough)
// @Tags dvr-settings
// @Produce json
// @Success 200 {object} APIResponse{data=object}
// @Failure 400 {object} APIResponse "DVR_NOT_CONFIGURED | DVR_CONNECTION_FAILED"
// @Router /api/v1/settings/radarr/quality-profiles [get]
func (h *DVRSettingsHandler) getQualityProfiles(c *gin.Context, plugin string) {
	profiles, err := h.service.GetQualityProfiles(c.Request.Context(), plugin)
	if err != nil {
		h.respondError(c, plugin, err, "無法載入 "+pluginDisplayName(plugin)+" 品質設定檔")
		return
	}
	if profiles == nil {
		profiles = []plugins.QualityProfile{}
	}
	SuccessResponse(c, gin.H{"quality_profiles": profiles})
}

// getRootFolders handles GET /api/v1/settings/{plugin}/root-folders
// @Summary List the DVR plugin's root folders (passthrough)
// @Tags dvr-settings
// @Produce json
// @Success 200 {object} APIResponse{data=object}
// @Failure 400 {object} APIResponse "DVR_NOT_CONFIGURED | DVR_CONNECTION_FAILED"
// @Router /api/v1/settings/radarr/root-folders [get]
func (h *DVRSettingsHandler) getRootFolders(c *gin.Context, plugin string) {
	folders, err := h.service.GetRootFolders(c.Request.Context(), plugin)
	if err != nil {
		h.respondError(c, plugin, err, "無法載入 "+pluginDisplayName(plugin)+" 根資料夾")
		return
	}
	if folders == nil {
		folders = []plugins.RootFolder{}
	}
	SuccessResponse(c, gin.H{"root_folders": folders})
}

// respondError lifts typed PluginError codes into the Rule 3 envelope
// (errors.As — the qbittorrent_handler:120-138 pattern). DVR_TEST_FAILED maps
// to 409 per AC #4 (save refused); other DVR_* failures are 400s. Expected
// 4xx flows log at Debug (CR M2 precedent), unexpected failures at Error.
func (h *DVRSettingsHandler) respondError(c *gin.Context, plugin string, err error, message string) {
	var pluginErr *plugins.PluginError
	if errors.As(err, &pluginErr) {
		status := http.StatusBadRequest
		if pluginErr.Code == plugins.ErrCodeTestFailed {
			status = http.StatusConflict
		}
		slog.Debug("DVR settings operation rejected",
			"plugin", plugin, "code", pluginErr.Code, "error", err)
		ErrorResponse(c, status, pluginErr.Code, message, err.Error())
		return
	}

	slog.Error("DVR settings operation failed", "plugin", plugin, "error", err)
	InternalServerError(c, message)
}

// pluginDisplayName renders the zh-TW-facing plugin name (Radarr/Sonarr are
// proper nouns — capitalize).
func pluginDisplayName(plugin string) string {
	if plugin == "" {
		return plugin
	}
	return fmt.Sprintf("%s%s", string(plugin[0]-'a'+'A'), plugin[1:])
}
