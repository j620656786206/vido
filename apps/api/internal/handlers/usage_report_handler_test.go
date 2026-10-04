package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/services"
)

type fakeUsageReport struct {
	status  services.UsageReportStatus
	err     error
	setWith *bool
}

func (f *fakeUsageReport) Status(context.Context) (services.UsageReportStatus, error) {
	return f.status, f.err
}

func (f *fakeUsageReport) SetEnabled(_ context.Context, enabled bool) (services.UsageReportStatus, error) {
	f.setWith = &enabled
	f.status.Enabled = enabled
	return f.status, f.err
}

func usageReportRouter(svc services.UsageReportServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	NewUsageReportHandler(svc).RegisterRoutes(v1)
	return r
}

func doUsageReport(t *testing.T, r *gin.Engine, method, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, "/api/v1/settings/usage-report", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return w.Code, out
}

func TestUsageReportHandler_GetNeverSentIsNullNotEmpty(t *testing.T) {
	r := usageReportRouter(&fakeUsageReport{status: services.UsageReportStatus{Available: true}})

	code, out := doUsageReport(t, r, http.MethodGet, "")

	require.Equal(t, http.StatusOK, code)
	data := out["data"].(map[string]any)
	assert.Equal(t, true, data["available"])
	assert.Equal(t, false, data["enabled"])
	assert.Contains(t, data, "last_sent_at")
	assert.Nil(t, data["last_sent_at"], "never sent is null, not a zero time")
	assert.Contains(t, data, "last_payload")
	assert.Nil(t, data["last_payload"])
}

func TestUsageReportHandler_GetReturnsTheVerbatimPayload(t *testing.T) {
	sent := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	payload := `{"type":"event","payload":{"website":"w"}}`
	r := usageReportRouter(&fakeUsageReport{status: services.UsageReportStatus{
		Available: true, Enabled: true, LastSentAt: &sent, LastPayload: &payload,
	}})

	code, out := doUsageReport(t, r, http.MethodGet, "")

	require.Equal(t, http.StatusOK, code)
	data := out["data"].(map[string]any)
	assert.Equal(t, payload, data["last_payload"], "a string, byte for byte — not a re-encoded object")
	assert.Equal(t, "2026-10-04T12:00:00Z", data["last_sent_at"])
}

func TestUsageReportHandler_PutTurnsOnAndOff(t *testing.T) {
	svc := &fakeUsageReport{status: services.UsageReportStatus{Available: true}}
	r := usageReportRouter(svc)

	code, out := doUsageReport(t, r, http.MethodPut, `{"enabled":true}`)
	require.Equal(t, http.StatusOK, code)
	require.NotNil(t, svc.setWith)
	assert.True(t, *svc.setWith)
	assert.Equal(t, true, out["data"].(map[string]any)["enabled"])

	code, _ = doUsageReport(t, r, http.MethodPut, `{"enabled":false}`)
	require.Equal(t, http.StatusOK, code)
	assert.False(t, *svc.setWith)
}

func TestUsageReportHandler_PutWithoutEnabledIsAValidationError(t *testing.T) {
	svc := &fakeUsageReport{}
	r := usageReportRouter(svc)

	code, out := doUsageReport(t, r, http.MethodPut, `{}`)

	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, false, out["success"])
	assert.Nil(t, svc.setWith, "a missing field must not silently turn the report off")
}

func TestUsageReportHandler_ServiceErrorIs500(t *testing.T) {
	r := usageReportRouter(&fakeUsageReport{err: errors.New("db gone")})

	code, out := doUsageReport(t, r, http.MethodGet, "")

	assert.Equal(t, http.StatusInternalServerError, code)
	assert.Equal(t, false, out["success"])
}
