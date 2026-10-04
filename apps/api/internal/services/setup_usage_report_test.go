package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// infra-optin-usage-report-a2 AC #8 — the wizard's opt-in answer.

type spyUsageReport struct {
	calls []bool
	err   error
}

func (s *spyUsageReport) Status(context.Context) (UsageReportStatus, error) {
	return UsageReportStatus{}, nil
}

func (s *spyUsageReport) SetEnabled(_ context.Context, enabled bool) (UsageReportStatus, error) {
	s.calls = append(s.calls, enabled)
	return UsageReportStatus{Enabled: enabled}, s.err
}

func minimalSetupRepo() *MockSettingsRepo {
	repo := new(MockSettingsRepo)
	repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
	repo.On("SetString", mock.Anything, "language", "zh-TW").Return(nil)
	repo.On("SetString", mock.Anything, "media_folder_path", "/media").Return(nil)
	repo.On("SetBool", mock.Anything, "setup_completed", true).Return(nil)
	return repo
}

func TestCompleteSetup_UsageReportYesTurnsItOn(t *testing.T) {
	spy := &spyUsageReport{}
	svc := NewSetupService(minimalSetupRepo(), nil)
	svc.SetUsageReport(spy)

	err := svc.CompleteSetup(context.Background(), SetupConfig{Language: "zh-TW", MediaFolderPath: "/media", UsageReportEnabled: true})

	require.NoError(t, err)
	assert.Equal(t, []bool{true}, spy.calls)
}

func TestCompleteSetup_UsageReportNoWritesNothing(t *testing.T) {
	spy := &spyUsageReport{}
	svc := NewSetupService(minimalSetupRepo(), nil)
	svc.SetUsageReport(spy)

	err := svc.CompleteSetup(context.Background(), SetupConfig{Language: "zh-TW", MediaFolderPath: "/media"})

	require.NoError(t, err)
	assert.Empty(t, spy.calls, "no is the default — no install id is created, no row is written")
}

func TestCompleteSetup_UsageReportFailureLeavesItOffAndSetupSucceeds(t *testing.T) {
	spy := &spyUsageReport{err: errors.New("db locked")}
	repo := minimalSetupRepo()
	svc := NewSetupService(repo, nil)
	svc.SetUsageReport(spy)

	err := svc.CompleteSetup(context.Background(), SetupConfig{Language: "zh-TW", MediaFolderPath: "/media", UsageReportEnabled: true})

	require.NoError(t, err, "a lost opt-in must not fail a setup whose libraries already exist")
	repo.AssertCalled(t, "SetBool", mock.Anything, "setup_completed", true)
}

func TestValidateStep_UsageReportIsAKnownStep(t *testing.T) {
	svc := NewSetupService(new(MockSettingsRepo), nil)

	assert.NoError(t, svc.ValidateStep(context.Background(), "usage-report", map[string]interface{}{"usage_report_enabled": true}))
	assert.NoError(t, svc.ValidateStep(context.Background(), "usage-report", map[string]interface{}{}))
}
