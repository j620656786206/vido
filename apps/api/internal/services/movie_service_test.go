package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// MockMovieRepository is a mock implementation of MovieRepositoryInterface
type MockMovieRepository struct {
	mock.Mock
}

func (m *MockMovieRepository) Create(ctx context.Context, movie *models.Movie) error {
	args := m.Called(ctx, movie)
	return args.Error(0)
}

func (m *MockMovieRepository) FindByID(ctx context.Context, id string) (*models.Movie, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Movie), args.Error(1)
}

func (m *MockMovieRepository) FindByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error) {
	args := m.Called(ctx, tmdbID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Movie), args.Error(1)
}

func (m *MockMovieRepository) FindByIMDbID(ctx context.Context, imdbID string) (*models.Movie, error) {
	args := m.Called(ctx, imdbID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Movie), args.Error(1)
}

func (m *MockMovieRepository) Update(ctx context.Context, movie *models.Movie) error {
	args := m.Called(ctx, movie)
	return args.Error(0)
}

func (m *MockMovieRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMovieRepository) List(ctx context.Context, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]models.Movie), args.Get(1).(*repository.PaginationResult), args.Error(2)
}

func (m *MockMovieRepository) SearchByTitle(ctx context.Context, title string, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	args := m.Called(ctx, title, params)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]models.Movie), args.Get(1).(*repository.PaginationResult), args.Error(2)
}

// Verify mock implements interface
var _ repository.MovieRepositoryInterface = (*MockMovieRepository)(nil)

func TestMovieService_GetByID(t *testing.T) {
	tests := []struct {
		name      string
		movieID   string
		setupMock func(*MockMovieRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:    "success",
			movieID: "movie-123",
			setupMock: func(m *MockMovieRepository) {
				m.On("FindByID", mock.Anything, "movie-123").Return(&models.Movie{
					ID:    "movie-123",
					Title: "Test Movie",
				}, nil)
			},
			wantErr: false,
		},
		{
			name:    "empty id returns error",
			movieID: "",
			setupMock: func(m *MockMovieRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "movie id cannot be empty",
		},
		{
			name:    "repository error",
			movieID: "movie-456",
			setupMock: func(m *MockMovieRepository) {
				m.On("FindByID", mock.Anything, "movie-456").Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockMovieRepository)
			tt.setupMock(mockRepo)

			service := NewMovieService(mockRepo)
			movie, err := service.GetByID(context.Background(), tt.movieID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, movie)
				assert.Equal(t, tt.movieID, movie.ID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestMovieService_Create(t *testing.T) {
	tests := []struct {
		name      string
		movie     *models.Movie
		setupMock func(*MockMovieRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "success",
			movie: &models.Movie{
				Title:       "New Movie",
				ReleaseDate: "2024-01-15",
			},
			setupMock: func(m *MockMovieRepository) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*models.Movie")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "nil movie returns error",
			movie: nil,
			setupMock: func(m *MockMovieRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "movie cannot be nil",
		},
		{
			name: "empty title returns error",
			movie: &models.Movie{
				Title:       "",
				ReleaseDate: "2024-01-15",
			},
			setupMock: func(m *MockMovieRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "movie title cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockMovieRepository)
			tt.setupMock(mockRepo)

			service := NewMovieService(mockRepo)
			err := service.Create(context.Background(), tt.movie)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, tt.movie.ID) // ID should be generated
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestMovieService_List(t *testing.T) {
	mockRepo := new(MockMovieRepository)
	mockRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
		[]models.Movie{{ID: "1", Title: "Movie 1"}},
		&repository.PaginationResult{Page: 1, PageSize: 20, TotalResults: 1, TotalPages: 1},
		nil,
	)

	service := NewMovieService(mockRepo)
	movies, pagination, err := service.List(context.Background(), repository.NewListParams())

	assert.NoError(t, err)
	assert.Len(t, movies, 1)
	assert.Equal(t, 1, pagination.TotalResults)
	mockRepo.AssertExpectations(t)
}
