package services

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/qbittorrent"
)

// MockQBServiceForDownload mocks the QBittorrentServiceInterface for download tests.
type MockQBServiceForDownload struct {
	mock.Mock
}

func (m *MockQBServiceForDownload) GetConfig(ctx context.Context) (*qbittorrent.Config, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*qbittorrent.Config), args.Error(1)
}

func (m *MockQBServiceForDownload) SaveConfig(ctx context.Context, config *qbittorrent.Config) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockQBServiceForDownload) TestConnection(ctx context.Context) (*qbittorrent.VersionInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*qbittorrent.VersionInfo), args.Error(1)
}

func (m *MockQBServiceForDownload) TestConnectionWithConfig(ctx context.Context, config *qbittorrent.Config) (*qbittorrent.VersionInfo, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*qbittorrent.VersionInfo), args.Error(1)
}

func (m *MockQBServiceForDownload) IsConfigured(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func newTestDownloadService(mockQB *MockQBServiceForDownload) *DownloadService {
	return NewDownloadService(mockQB, slog.Default())
}

func TestNewDownloadService(t *testing.T) {
	mockQB := new(MockQBServiceForDownload)
	service := NewDownloadService(mockQB, slog.Default())
	assert.NotNil(t, service)
}

func TestDownloadService_GetAllDownloads_NotConfigured(t *testing.T) {
	mockQB := new(MockQBServiceForDownload)
	mockQB.On("GetConfig", mock.Anything).Return(&qbittorrent.Config{Host: ""}, nil)

	service := newTestDownloadService(mockQB)
	torrents, err := service.GetAllDownloads(context.Background(), "", "")

	assert.Nil(t, torrents)
	assert.Error(t, err)

	var connErr *qbittorrent.ConnectionError
	require.ErrorAs(t, err, &connErr)
	assert.Equal(t, qbittorrent.ErrCodeNotConfigured, connErr.Code)
	mockQB.AssertExpectations(t)
}

func TestDownloadService_GetAllDownloads_ConfigError(t *testing.T) {
	mockQB := new(MockQBServiceForDownload)
	mockQB.On("GetConfig", mock.Anything).Return(nil, errors.New("database error"))

	service := newTestDownloadService(mockQB)
	torrents, err := service.GetAllDownloads(context.Background(), "", "")

	assert.Nil(t, torrents)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get qBittorrent config")
	mockQB.AssertExpectations(t)
}

func TestDownloadService_GetDownloadDetails_NotConfigured(t *testing.T) {
	mockQB := new(MockQBServiceForDownload)
	mockQB.On("GetConfig", mock.Anything).Return(&qbittorrent.Config{Host: ""}, nil)

	service := newTestDownloadService(mockQB)
	details, err := service.GetDownloadDetails(context.Background(), "abc123")

	assert.Nil(t, details)
	assert.Error(t, err)

	var connErr *qbittorrent.ConnectionError
	require.ErrorAs(t, err, &connErr)
	assert.Equal(t, qbittorrent.ErrCodeNotConfigured, connErr.Code)
	mockQB.AssertExpectations(t)
}

func TestDownloadService_GetDownloadDetails_ConfigError(t *testing.T) {
	mockQB := new(MockQBServiceForDownload)
	mockQB.On("GetConfig", mock.Anything).Return(nil, errors.New("database error"))

	service := newTestDownloadService(mockQB)
	details, err := service.GetDownloadDetails(context.Background(), "abc123")

	assert.Nil(t, details)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get qBittorrent config")
	mockQB.AssertExpectations(t)
}

func TestDownloadService_ImplementsInterface(t *testing.T) {
	var _ DownloadServiceInterface = (*DownloadService)(nil)
}
