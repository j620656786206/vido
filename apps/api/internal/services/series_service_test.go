package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// MockSeriesRepository is a mock implementation of SeriesRepositoryInterface
type MockSeriesRepository struct {
	mock.Mock
}

func (m *MockSeriesRepository) Create(ctx context.Context, series *models.Series) error {
	args := m.Called(ctx, series)
	return args.Error(0)
}

func (m *MockSeriesRepository) FindByID(ctx context.Context, id string) (*models.Series, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Series), args.Error(1)
}

func (m *MockSeriesRepository) FindByTMDbID(ctx context.Context, tmdbID int64) (*models.Series, error) {
	args := m.Called(ctx, tmdbID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Series), args.Error(1)
}

func (m *MockSeriesRepository) FindByIMDbID(ctx context.Context, imdbID string) (*models.Series, error) {
	args := m.Called(ctx, imdbID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Series), args.Error(1)
}

func (m *MockSeriesRepository) Update(ctx context.Context, series *models.Series) error {
	args := m.Called(ctx, series)
	return args.Error(0)
}

func (m *MockSeriesRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSeriesRepository) List(ctx context.Context, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]models.Series), args.Get(1).(*repository.PaginationResult), args.Error(2)
}

func (m *MockSeriesRepository) SearchByTitle(ctx context.Context, title string, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	args := m.Called(ctx, title, params)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]models.Series), args.Get(1).(*repository.PaginationResult), args.Error(2)
}

func (m *MockSeriesRepository) FullTextSearch(ctx context.Context, query string, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	args := m.Called(ctx, query, params)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]models.Series), args.Get(1).(*repository.PaginationResult), args.Error(2)
}

func (m *MockSeriesRepository) Upsert(ctx context.Context, series *models.Series) error {
	args := m.Called(ctx, series)
	return args.Error(0)
}

// Verify mock implements interface
var _ repository.SeriesRepositoryInterface = (*MockSeriesRepository)(nil)

func TestSeriesService_GetByID(t *testing.T) {
	tests := []struct {
		name      string
		seriesID  string
		setupMock func(*MockSeriesRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "success",
			seriesID: "series-123",
			setupMock: func(m *MockSeriesRepository) {
				m.On("FindByID", mock.Anything, "series-123").Return(&models.Series{
					ID:    "series-123",
					Title: "Test Series",
				}, nil)
			},
			wantErr: false,
		},
		{
			name:     "empty id returns error",
			seriesID: "",
			setupMock: func(m *MockSeriesRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "series id cannot be empty",
		},
		{
			name:     "repository error",
			seriesID: "series-456",
			setupMock: func(m *MockSeriesRepository) {
				m.On("FindByID", mock.Anything, "series-456").Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSeriesRepository)
			tt.setupMock(mockRepo)

			service := NewSeriesService(mockRepo)
			series, err := service.GetByID(context.Background(), tt.seriesID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, series)
				assert.Equal(t, tt.seriesID, series.ID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSeriesService_Create(t *testing.T) {
	tests := []struct {
		name      string
		series    *models.Series
		setupMock func(*MockSeriesRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "success",
			series: &models.Series{
				Title:        "New Series",
				FirstAirDate: "2024-01-15",
			},
			setupMock: func(m *MockSeriesRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*models.Series")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "nil series returns error",
			series: nil,
			setupMock: func(m *MockSeriesRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "series cannot be nil",
		},
		{
			name: "empty title returns error",
			series: &models.Series{
				Title:        "",
				FirstAirDate: "2024-01-15",
			},
			setupMock: func(m *MockSeriesRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "series title cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSeriesRepository)
			tt.setupMock(mockRepo)

			service := NewSeriesService(mockRepo)
			err := service.Create(context.Background(), tt.series)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, tt.series.ID) // ID should be generated
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSeriesService_List(t *testing.T) {
	mockRepo := new(MockSeriesRepository)
	mockRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
		[]models.Series{{ID: "1", Title: "Series 1"}},
		&repository.PaginationResult{Page: 1, PageSize: 20, TotalResults: 1, TotalPages: 1},
		nil,
	)

	service := NewSeriesService(mockRepo)
	series, pagination, err := service.List(context.Background(), repository.NewListParams())

	assert.NoError(t, err)
	assert.Len(t, series, 1)
	assert.Equal(t, 1, pagination.TotalResults)
	mockRepo.AssertExpectations(t)
}

func TestSeriesService_GetByTMDbID(t *testing.T) {
	tests := []struct {
		name      string
		tmdbID    int64
		setupMock func(*MockSeriesRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "success",
			tmdbID: 12345,
			setupMock: func(m *MockSeriesRepository) {
				m.On("FindByTMDbID", mock.Anything, int64(12345)).Return(&models.Series{
					ID:     "series-123",
					Title:  "Test Series",
					TMDbID: sql.NullInt64{Int64: 12345, Valid: true},
				}, nil)
			},
			wantErr: false,
		},
		{
			name:   "zero tmdb id returns error",
			tmdbID: 0,
			setupMock: func(m *MockSeriesRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "tmdb id must be positive",
		},
		{
			name:   "negative tmdb id returns error",
			tmdbID: -1,
			setupMock: func(m *MockSeriesRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "tmdb id must be positive",
		},
		{
			name:   "repository error",
			tmdbID: 99999,
			setupMock: func(m *MockSeriesRepository) {
				m.On("FindByTMDbID", mock.Anything, int64(99999)).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSeriesRepository)
			tt.setupMock(mockRepo)

			service := NewSeriesService(mockRepo)
			series, err := service.GetByTMDbID(context.Background(), tt.tmdbID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, series)
				assert.Equal(t, tt.tmdbID, series.TMDbID.Int64)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSeriesService_GetByIMDbID(t *testing.T) {
	tests := []struct {
		name      string
		imdbID    string
		setupMock func(*MockSeriesRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "success",
			imdbID: "tt1234567",
			setupMock: func(m *MockSeriesRepository) {
				m.On("FindByIMDbID", mock.Anything, "tt1234567").Return(&models.Series{
					ID:     "series-123",
					Title:  "Test Series",
					IMDbID: sql.NullString{String: "tt1234567", Valid: true},
				}, nil)
			},
			wantErr: false,
		},
		{
			name:   "empty imdb id returns error",
			imdbID: "",
			setupMock: func(m *MockSeriesRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "imdb id cannot be empty",
		},
		{
			name:   "repository error",
			imdbID: "tt9999999",
			setupMock: func(m *MockSeriesRepository) {
				m.On("FindByIMDbID", mock.Anything, "tt9999999").Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSeriesRepository)
			tt.setupMock(mockRepo)

			service := NewSeriesService(mockRepo)
			series, err := service.GetByIMDbID(context.Background(), tt.imdbID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, series)
				assert.Equal(t, tt.imdbID, series.IMDbID.String)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSeriesService_Update(t *testing.T) {
	tests := []struct {
		name      string
		series    *models.Series
		setupMock func(*MockSeriesRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "success",
			series: &models.Series{
				ID:    "series-123",
				Title: "Updated Series",
			},
			setupMock: func(m *MockSeriesRepository) {
				m.On("Update", mock.Anything, mock.AnythingOfType("*models.Series")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "nil series returns error",
			series: nil,
			setupMock: func(m *MockSeriesRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "series cannot be nil",
		},
		{
			name: "empty id returns error",
			series: &models.Series{
				ID:    "",
				Title: "Test Series",
			},
			setupMock: func(m *MockSeriesRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "series id cannot be empty",
		},
		{
			name: "empty title returns error",
			series: &models.Series{
				ID:    "series-123",
				Title: "",
			},
			setupMock: func(m *MockSeriesRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "series title cannot be empty",
		},
		{
			name: "repository error",
			series: &models.Series{
				ID:    "series-123",
				Title: "Test Series",
			},
			setupMock: func(m *MockSeriesRepository) {
				m.On("Update", mock.Anything, mock.AnythingOfType("*models.Series")).Return(errors.New("update failed"))
			},
			wantErr: true,
			errMsg:  "failed to update series",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSeriesRepository)
			tt.setupMock(mockRepo)

			service := NewSeriesService(mockRepo)
			err := service.Update(context.Background(), tt.series)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSeriesService_Delete(t *testing.T) {
	tests := []struct {
		name      string
		seriesID  string
		setupMock func(*MockSeriesRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "success",
			seriesID: "series-123",
			setupMock: func(m *MockSeriesRepository) {
				m.On("Delete", mock.Anything, "series-123").Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "empty id returns error",
			seriesID: "",
			setupMock: func(m *MockSeriesRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "series id cannot be empty",
		},
		{
			name:     "repository error",
			seriesID: "series-456",
			setupMock: func(m *MockSeriesRepository) {
				m.On("Delete", mock.Anything, "series-456").Return(errors.New("delete failed"))
			},
			wantErr: true,
			errMsg:  "failed to delete series",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSeriesRepository)
			tt.setupMock(mockRepo)

			service := NewSeriesService(mockRepo)
			err := service.Delete(context.Background(), tt.seriesID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSeriesService_SearchByTitle(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		setupMock func(*MockSeriesRepository)
		wantErr   bool
		errMsg    string
		wantLen   int
	}{
		{
			name:  "success",
			title: "Test",
			setupMock: func(m *MockSeriesRepository) {
				m.On("SearchByTitle", mock.Anything, "Test", mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Series{{ID: "1", Title: "Test Series"}},
					&repository.PaginationResult{Page: 1, PageSize: 20, TotalResults: 1, TotalPages: 1},
					nil,
				)
			},
			wantErr: false,
			wantLen: 1,
		},
		{
			name:  "empty title returns error",
			title: "",
			setupMock: func(m *MockSeriesRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "search title cannot be empty",
		},
		{
			name:  "repository error",
			title: "Error",
			setupMock: func(m *MockSeriesRepository) {
				m.On("SearchByTitle", mock.Anything, "Error", mock.AnythingOfType("repository.ListParams")).Return(
					nil, nil, errors.New("search failed"),
				)
			},
			wantErr: true,
			errMsg:  "failed to search series",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSeriesRepository)
			tt.setupMock(mockRepo)

			service := NewSeriesService(mockRepo)
			series, pagination, err := service.SearchByTitle(context.Background(), tt.title, repository.NewListParams())

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, series, tt.wantLen)
				assert.NotNil(t, pagination)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSeriesService_List_Error(t *testing.T) {
	mockRepo := new(MockSeriesRepository)
	mockRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
		nil, nil, errors.New("list failed"),
	)

	service := NewSeriesService(mockRepo)
	series, pagination, err := service.List(context.Background(), repository.NewListParams())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list series")
	assert.Nil(t, series)
	assert.Nil(t, pagination)
	mockRepo.AssertExpectations(t)
}
