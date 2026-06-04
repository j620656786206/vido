package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// mockFilterPresetRepo is an in-memory FilterPresetRepositoryInterface for tests.
type mockFilterPresetRepo struct {
	presets   []models.FilterPreset
	createErr error
}

func (m *mockFilterPresetRepo) Create(_ context.Context, p *models.FilterPreset) error {
	if m.createErr != nil {
		return m.createErr
	}
	if p.ID == "" {
		p.ID = "mock-id"
	}
	m.presets = append(m.presets, *p)
	return nil
}

func (m *mockFilterPresetRepo) GetAll(_ context.Context) ([]models.FilterPreset, error) {
	return m.presets, nil
}

func (m *mockFilterPresetRepo) Delete(_ context.Context, id string) error {
	for i, p := range m.presets {
		if p.ID == id {
			m.presets = append(m.presets[:i], m.presets[i+1:]...)
			return nil
		}
	}
	return repository.ErrFilterPresetNotFound
}

func (m *mockFilterPresetRepo) Count(_ context.Context) (int, error) {
	return len(m.presets), nil
}

var _ repository.FilterPresetRepositoryInterface = (*mockFilterPresetRepo)(nil)

func TestFilterPresetService_CreatePreset(t *testing.T) {
	repo := &mockFilterPresetRepo{}
	svc := NewFilterPresetService(repo)

	preset, err := svc.CreatePreset(context.Background(), CreateFilterPresetRequest{
		Name:    "高評分動畫",
		Filters: `{"genre":"16","rating_gte":"8"}`,
	})
	require.NoError(t, err)
	assert.Equal(t, "高評分動畫", preset.Name)
	assert.Equal(t, 0, preset.SortOrder, "first preset appends at sort_order 0")
	assert.Len(t, repo.presets, 1)
}

func TestFilterPresetService_CreatePreset_AppendsSortOrder(t *testing.T) {
	repo := &mockFilterPresetRepo{}
	svc := NewFilterPresetService(repo)
	ctx := context.Background()

	_, err := svc.CreatePreset(ctx, CreateFilterPresetRequest{Name: "一", Filters: `{}`})
	require.NoError(t, err)
	second, err := svc.CreatePreset(ctx, CreateFilterPresetRequest{Name: "二", Filters: `{}`})
	require.NoError(t, err)
	assert.Equal(t, 1, second.SortOrder)
}

func TestFilterPresetService_CreatePreset_RejectsEmptyName(t *testing.T) {
	svc := NewFilterPresetService(&mockFilterPresetRepo{})
	_, err := svc.CreatePreset(context.Background(), CreateFilterPresetRequest{Name: "   ", Filters: `{}`})
	require.Error(t, err)
	var ve *models.ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestFilterPresetService_CreatePreset_RejectsLongName(t *testing.T) {
	svc := NewFilterPresetService(&mockFilterPresetRepo{})
	longName := ""
	for i := 0; i < 31; i++ {
		longName += "字"
	}
	_, err := svc.CreatePreset(context.Background(), CreateFilterPresetRequest{Name: longName, Filters: `{}`})
	var ve *models.ValidationError
	require.True(t, errors.As(err, &ve))
	assert.Equal(t, "name", ve.Field)
}

func TestFilterPresetService_CreatePreset_RejectsInvalidJSON(t *testing.T) {
	svc := NewFilterPresetService(&mockFilterPresetRepo{})
	_, err := svc.CreatePreset(context.Background(), CreateFilterPresetRequest{Name: "壞", Filters: `{not json`})
	var ve *models.ValidationError
	require.True(t, errors.As(err, &ve))
	assert.Equal(t, "filters", ve.Field)
}

func TestFilterPresetService_CreatePreset_EnforcesMaxLimit(t *testing.T) {
	repo := &mockFilterPresetRepo{}
	for i := 0; i < models.FilterPresetMaxCount; i++ {
		repo.presets = append(repo.presets, models.FilterPreset{ID: string(rune('a' + i)), Name: "x", Filters: "{}"})
	}
	svc := NewFilterPresetService(repo)

	_, err := svc.CreatePreset(context.Background(), CreateFilterPresetRequest{Name: "超出", Filters: `{}`})
	assert.True(t, errors.Is(err, ErrFilterPresetLimitReached))
	assert.Len(t, repo.presets, models.FilterPresetMaxCount, "no preset added past the cap")
}

func TestFilterPresetService_DeletePreset(t *testing.T) {
	repo := &mockFilterPresetRepo{presets: []models.FilterPreset{{ID: "p1", Name: "x", Filters: "{}"}}}
	svc := NewFilterPresetService(repo)

	require.NoError(t, svc.DeletePreset(context.Background(), "p1"))
	assert.Empty(t, repo.presets)
}

func TestFilterPresetService_GetAllPresets(t *testing.T) {
	repo := &mockFilterPresetRepo{presets: []models.FilterPreset{
		{ID: "p1", Name: "一", Filters: "{}"},
		{ID: "p2", Name: "二", Filters: "{}"},
	}}
	svc := NewFilterPresetService(repo)

	presets, err := svc.GetAllPresets(context.Background())
	require.NoError(t, err)
	assert.Len(t, presets, 2)
}
