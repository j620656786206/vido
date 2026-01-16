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

func TestMovieService_GetByTMDbID(t *testing.T) {
	tests := []struct {
		name      string
		tmdbID    int64
		setupMock func(*MockMovieRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "success",
			tmdbID: 12345,
			setupMock: func(m *MockMovieRepository) {
				m.On("FindByTMDbID", mock.Anything, int64(12345)).Return(&models.Movie{
					ID:     "movie-123",
					Title:  "Test Movie",
					TMDbID: sql.NullInt64{Int64: 12345, Valid: true},
				}, nil)
			},
			wantErr: false,
		},
		{
			name:   "zero tmdb id returns error",
			tmdbID: 0,
			setupMock: func(m *MockMovieRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "tmdb id must be positive",
		},
		{
			name:   "negative tmdb id returns error",
			tmdbID: -1,
			setupMock: func(m *MockMovieRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "tmdb id must be positive",
		},
		{
			name:   "repository error",
			tmdbID: 99999,
			setupMock: func(m *MockMovieRepository) {
				m.On("FindByTMDbID", mock.Anything, int64(99999)).Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockMovieRepository)
			tt.setupMock(mockRepo)

			service := NewMovieService(mockRepo)
			movie, err := service.GetByTMDbID(context.Background(), tt.tmdbID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, movie)
				assert.Equal(t, tt.tmdbID, movie.TMDbID.Int64)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestMovieService_GetByIMDbID(t *testing.T) {
	tests := []struct {
		name      string
		imdbID    string
		setupMock func(*MockMovieRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "success",
			imdbID: "tt1234567",
			setupMock: func(m *MockMovieRepository) {
				m.On("FindByIMDbID", mock.Anything, "tt1234567").Return(&models.Movie{
					ID:     "movie-123",
					Title:  "Test Movie",
					IMDbID: sql.NullString{String: "tt1234567", Valid: true},
				}, nil)
			},
			wantErr: false,
		},
		{
			name:   "empty imdb id returns error",
			imdbID: "",
			setupMock: func(m *MockMovieRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "imdb id cannot be empty",
		},
		{
			name:   "repository error",
			imdbID: "tt9999999",
			setupMock: func(m *MockMovieRepository) {
				m.On("FindByIMDbID", mock.Anything, "tt9999999").Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockMovieRepository)
			tt.setupMock(mockRepo)

			service := NewMovieService(mockRepo)
			movie, err := service.GetByIMDbID(context.Background(), tt.imdbID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, movie)
				assert.Equal(t, tt.imdbID, movie.IMDbID.String)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestMovieService_Update(t *testing.T) {
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
				ID:    "movie-123",
				Title: "Updated Movie",
			},
			setupMock: func(m *MockMovieRepository) {
				m.On("Update", mock.Anything, mock.AnythingOfType("*models.Movie")).Return(nil)
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
			name: "empty id returns error",
			movie: &models.Movie{
				ID:    "",
				Title: "Test Movie",
			},
			setupMock: func(m *MockMovieRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "movie id cannot be empty",
		},
		{
			name: "empty title returns error",
			movie: &models.Movie{
				ID:    "movie-123",
				Title: "",
			},
			setupMock: func(m *MockMovieRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "movie title cannot be empty",
		},
		{
			name: "repository error",
			movie: &models.Movie{
				ID:    "movie-123",
				Title: "Test Movie",
			},
			setupMock: func(m *MockMovieRepository) {
				m.On("Update", mock.Anything, mock.AnythingOfType("*models.Movie")).Return(errors.New("update failed"))
			},
			wantErr: true,
			errMsg:  "failed to update movie",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockMovieRepository)
			tt.setupMock(mockRepo)

			service := NewMovieService(mockRepo)
			err := service.Update(context.Background(), tt.movie)

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

func TestMovieService_Delete(t *testing.T) {
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
				m.On("Delete", mock.Anything, "movie-123").Return(nil)
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
				m.On("Delete", mock.Anything, "movie-456").Return(errors.New("delete failed"))
			},
			wantErr: true,
			errMsg:  "failed to delete movie",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockMovieRepository)
			tt.setupMock(mockRepo)

			service := NewMovieService(mockRepo)
			err := service.Delete(context.Background(), tt.movieID)

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

func TestMovieService_SearchByTitle(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		setupMock func(*MockMovieRepository)
		wantErr   bool
		errMsg    string
		wantLen   int
	}{
		{
			name:  "success",
			title: "Test",
			setupMock: func(m *MockMovieRepository) {
				m.On("SearchByTitle", mock.Anything, "Test", mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Movie{{ID: "1", Title: "Test Movie"}},
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
			setupMock: func(m *MockMovieRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "search title cannot be empty",
		},
		{
			name:  "repository error",
			title: "Error",
			setupMock: func(m *MockMovieRepository) {
				m.On("SearchByTitle", mock.Anything, "Error", mock.AnythingOfType("repository.ListParams")).Return(
					nil, nil, errors.New("search failed"),
				)
			},
			wantErr: true,
			errMsg:  "failed to search movies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockMovieRepository)
			tt.setupMock(mockRepo)

			service := NewMovieService(mockRepo)
			movies, pagination, err := service.SearchByTitle(context.Background(), tt.title, repository.NewListParams())

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, movies, tt.wantLen)
				assert.NotNil(t, pagination)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestMovieService_List_Error(t *testing.T) {
	mockRepo := new(MockMovieRepository)
	mockRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
		nil, nil, errors.New("list failed"),
	)

	service := NewMovieService(mockRepo)
	movies, pagination, err := service.List(context.Background(), repository.NewListParams())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list movies")
	assert.Nil(t, movies)
	assert.Nil(t, pagination)
	mockRepo.AssertExpectations(t)
}
