package handlers

// 補審 M4 — the free-auto-generation opt-in must only be OFFERED where the
// trigger that honours it exists.
//
// The `auto_subtitle` column is writable in every mode, but the AutoGenerator
// is constructed only inside main.go's `if cfg.SubtitlePipelineEnabled()`
// block, and the shipped default is `legacy`. The list response therefore
// carries the capability, so the settings page can stop promising a lane this
// deployment does not run.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// capabilityFakeLibraryService implements only what ListLibraries drives; every
// other method is an unused stub so the fake stays honest about its scope.
type capabilityFakeLibraryService struct{}

func (capabilityFakeLibraryService) GetAllLibraries(context.Context) ([]models.MediaLibraryWithPaths, error) {
	return []models.MediaLibraryWithPaths{}, nil
}

func (capabilityFakeLibraryService) GetLibrary(context.Context, string) (*models.MediaLibraryWithPaths, error) {
	return nil, nil
}

func (capabilityFakeLibraryService) CreateLibrary(context.Context, services.CreateLibraryRequest) (*models.MediaLibrary, error) {
	return nil, nil
}

func (capabilityFakeLibraryService) UpdateLibrary(context.Context, string, services.UpdateLibraryRequest) (*models.MediaLibrary, error) {
	return nil, nil
}

func (capabilityFakeLibraryService) DeleteLibrary(context.Context, string, bool) error { return nil }

func (capabilityFakeLibraryService) AddPath(context.Context, string, string) (*models.MediaLibraryPath, error) {
	return nil, nil
}

func (capabilityFakeLibraryService) RemovePath(context.Context, string, string) error { return nil }

func (capabilityFakeLibraryService) RefreshPathStatuses(context.Context, string) ([]models.MediaLibraryPath, error) {
	return nil, nil
}

func listLibrariesCapability(t *testing.T, opts ...MediaLibrariesHandlerOption) map[string]any {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	NewMediaLibrariesHandler(capabilityFakeLibraryService{}, opts...).RegisterRoutes(router.Group("/api/v1"))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/libraries", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.True(t, body.Success)
	return body.Data
}

func TestListLibraries_ReportsAutoSubtitleSupportedInPipelineMode(t *testing.T) {
	data := listLibrariesCapability(t, WithAutoSubtitleSupport(func() bool { return true }))

	assert.Equal(t, true, data["auto_subtitle_supported"],
		"pipeline mode builds the auto-generator, so the opt-in is a real offer")
}

func TestListLibraries_ReportsAutoSubtitleUnsupportedInLegacyMode(t *testing.T) {
	data := listLibrariesCapability(t, WithAutoSubtitleSupport(func() bool { return false }))

	assert.Equal(t, false, data["auto_subtitle_supported"],
		"legacy mode never builds the trigger — offering the checkbox there is a promise nothing keeps")
	assert.Contains(t, data, "libraries", "the capability must ride the existing payload, not replace it")
}

func TestListLibraries_UnwiredCapabilityReportsFalse(t *testing.T) {
	data := listLibrariesCapability(t)

	assert.Equal(t, false, data["auto_subtitle_supported"],
		"a caller that never declared the pipeline must not advertise a lane it may not have")
}
