// Package testutil provides shared mock implementations and test fixtures
// for use across multiple test files in the Vido API.
//
// WARNING: This package must only be imported from _test.go files.
// It pulls in testify/mock which should not be in the production binary.
//
// All mocks use testify/mock.Called() delegation. If a method is called
// without a matching .On() expectation, testify will panic. Use
// SetupDefault*Expectations() to register catch-all expectations, then
// override specific methods with concrete matchers as needed.
package testutil

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// MockMovieRepository is a shared mock implementation of MovieRepositoryInterface.
// All methods delegate to testify/mock. Call SetupDefaultMovieExpectations to set up
// zero-value returns for all methods, then override the ones your test cares about.
type MockMovieRepository struct {
	mock.Mock
}

func (m *MockMovieRepository) Create(ctx context.Context, movie *models.Movie) error {
	return m.Called(ctx, movie).Error(0)
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
	return m.Called(ctx, movie).Error(0)
}

func (m *MockMovieRepository) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
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

func (m *MockMovieRepository) FullTextSearch(ctx context.Context, query string, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	args := m.Called(ctx, query, params)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]models.Movie), args.Get(1).(*repository.PaginationResult), args.Error(2)
}

func (m *MockMovieRepository) Upsert(ctx context.Context, movie *models.Movie) error {
	return m.Called(ctx, movie).Error(0)
}

func (m *MockMovieRepository) FindByFilePath(ctx context.Context, filePath string) (*models.Movie, error) {
	args := m.Called(ctx, filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Movie), args.Error(1)
}

func (m *MockMovieRepository) GetDistinctGenres(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockMovieRepository) GetYearRange(ctx context.Context) (int, int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Int(1), args.Error(2)
}

func (m *MockMovieRepository) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockMovieRepository) BulkCreate(ctx context.Context, movies []*models.Movie) error {
	return m.Called(ctx, movies).Error(0)
}

func (m *MockMovieRepository) FindByParseStatus(ctx context.Context, status models.ParseStatus) ([]models.Movie, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Movie), args.Error(1)
}

func (m *MockMovieRepository) UpdateSubtitleStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error {
	return m.Called(ctx, id, status, path, language, score).Error(0)
}

func (m *MockMovieRepository) FindBySubtitleStatus(ctx context.Context, status models.SubtitleStatus) ([]models.Movie, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Movie), args.Error(1)
}

func (m *MockMovieRepository) FindNeedingSubtitleSearch(ctx context.Context, olderThan time.Time) ([]models.Movie, error) {
	args := m.Called(ctx, olderThan)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Movie), args.Error(1)
}

func (m *MockMovieRepository) FindAllWithFilePath(ctx context.Context) ([]models.Movie, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Movie), args.Error(1)
}

func (m *MockMovieRepository) GetStats(ctx context.Context) (*repository.MediaStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.MediaStats), args.Error(1)
}

// Compile-time interface check
var _ repository.MovieRepositoryInterface = (*MockMovieRepository)(nil)

// MockSeriesRepository is a shared mock implementation of SeriesRepositoryInterface.
// All methods delegate to testify/mock. Call SetupDefaultSeriesExpectations to set up
// zero-value returns for all methods, then override the ones your test cares about.
type MockSeriesRepository struct {
	mock.Mock
}

func (m *MockSeriesRepository) Create(ctx context.Context, series *models.Series) error {
	return m.Called(ctx, series).Error(0)
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
	return m.Called(ctx, series).Error(0)
}

func (m *MockSeriesRepository) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
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
	return m.Called(ctx, series).Error(0)
}

func (m *MockSeriesRepository) GetDistinctGenres(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockSeriesRepository) GetYearRange(ctx context.Context) (int, int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Int(1), args.Error(2)
}

func (m *MockSeriesRepository) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockSeriesRepository) BulkCreate(ctx context.Context, seriesList []*models.Series) error {
	return m.Called(ctx, seriesList).Error(0)
}

func (m *MockSeriesRepository) FindByParseStatus(ctx context.Context, status models.ParseStatus) ([]models.Series, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Series), args.Error(1)
}

func (m *MockSeriesRepository) UpdateSubtitleStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error {
	return m.Called(ctx, id, status, path, language, score).Error(0)
}

func (m *MockSeriesRepository) FindBySubtitleStatus(ctx context.Context, status models.SubtitleStatus) ([]models.Series, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Series), args.Error(1)
}

func (m *MockSeriesRepository) FindNeedingSubtitleSearch(ctx context.Context, olderThan time.Time) ([]models.Series, error) {
	args := m.Called(ctx, olderThan)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Series), args.Error(1)
}

func (m *MockSeriesRepository) GetStats(ctx context.Context) (*repository.MediaStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.MediaStats), args.Error(1)
}

// Compile-time interface check
var _ repository.SeriesRepositoryInterface = (*MockSeriesRepository)(nil)

// SetupDefaultMovieExpectations registers Maybe() expectations that return zero values
// for all MockMovieRepository methods. Call this first, then override specific methods.
func SetupDefaultMovieExpectations(m *MockMovieRepository) {
	m.On("Create", mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("FindByID", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	m.On("FindByTMDbID", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	m.On("FindByIMDbID", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	m.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("Delete", mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("List", mock.Anything, mock.Anything).Maybe().Return([]models.Movie(nil), (*repository.PaginationResult)(nil), nil)
	m.On("SearchByTitle", mock.Anything, mock.Anything, mock.Anything).Maybe().Return([]models.Movie(nil), (*repository.PaginationResult)(nil), nil)
	m.On("FullTextSearch", mock.Anything, mock.Anything, mock.Anything).Maybe().Return([]models.Movie(nil), (*repository.PaginationResult)(nil), nil)
	m.On("Upsert", mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("FindByFilePath", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	m.On("GetDistinctGenres", mock.Anything).Maybe().Return([]string(nil), nil)
	m.On("GetYearRange", mock.Anything).Maybe().Return(0, 0, nil)
	m.On("Count", mock.Anything).Maybe().Return(0, nil)
	m.On("BulkCreate", mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("FindByParseStatus", mock.Anything, mock.Anything).Maybe().Return([]models.Movie(nil), nil)
	m.On("UpdateSubtitleStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("FindBySubtitleStatus", mock.Anything, mock.Anything).Maybe().Return([]models.Movie(nil), nil)
	m.On("FindNeedingSubtitleSearch", mock.Anything, mock.Anything).Maybe().Return([]models.Movie(nil), nil)
	m.On("FindAllWithFilePath", mock.Anything).Maybe().Return([]models.Movie(nil), nil)
	m.On("GetStats", mock.Anything).Maybe().Return((*repository.MediaStats)(nil), nil)
}

// SetupDefaultSeriesExpectations registers Maybe() expectations that return zero values
// for all MockSeriesRepository methods. Call this first, then override specific methods.
func SetupDefaultSeriesExpectations(m *MockSeriesRepository) {
	m.On("Create", mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("FindByID", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	m.On("FindByTMDbID", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	m.On("FindByIMDbID", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	m.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("Delete", mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("List", mock.Anything, mock.Anything).Maybe().Return([]models.Series(nil), (*repository.PaginationResult)(nil), nil)
	m.On("SearchByTitle", mock.Anything, mock.Anything, mock.Anything).Maybe().Return([]models.Series(nil), (*repository.PaginationResult)(nil), nil)
	m.On("FullTextSearch", mock.Anything, mock.Anything, mock.Anything).Maybe().Return([]models.Series(nil), (*repository.PaginationResult)(nil), nil)
	m.On("Upsert", mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("GetDistinctGenres", mock.Anything).Maybe().Return([]string(nil), nil)
	m.On("GetYearRange", mock.Anything).Maybe().Return(0, 0, nil)
	m.On("Count", mock.Anything).Maybe().Return(0, nil)
	m.On("BulkCreate", mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("FindByParseStatus", mock.Anything, mock.Anything).Maybe().Return([]models.Series(nil), nil)
	m.On("UpdateSubtitleStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("FindBySubtitleStatus", mock.Anything, mock.Anything).Maybe().Return([]models.Series(nil), nil)
	m.On("FindNeedingSubtitleSearch", mock.Anything, mock.Anything).Maybe().Return([]models.Series(nil), nil)
	m.On("GetStats", mock.Anything).Maybe().Return((*repository.MediaStats)(nil), nil)
}
