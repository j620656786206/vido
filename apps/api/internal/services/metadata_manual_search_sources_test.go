package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
)

// disc-2026-09-manual-search-hides-source-errors: a search in which no source
// could search at all must be an error, not 200 with nothing — the 手動選片
// dialog would otherwise tell the user to try another keyword.

type stubSourceProvider struct {
	source    models.MetadataSource
	available bool
	items     []metadata.MetadataItem
	err       error
}

func (p *stubSourceProvider) Name() string                  { return string(p.source) }
func (p *stubSourceProvider) Source() models.MetadataSource { return p.source }
func (p *stubSourceProvider) IsAvailable() bool             { return p.available }
func (p *stubSourceProvider) Status() metadata.ProviderStatus {
	return metadata.ProviderStatusAvailable
}
func (p *stubSourceProvider) Search(_ context.Context, _ *metadata.SearchRequest) (*metadata.SearchResult, error) {
	if p.err != nil {
		return nil, p.err
	}
	return &metadata.SearchResult{Items: p.items, Source: p.source}, nil
}

func manualSearchWith(providers ...metadata.MetadataProvider) *MetadataService {
	o := metadata.NewOrchestrator(metadata.OrchestratorConfig{})
	for _, p := range providers {
		o.RegisterProvider(p)
	}
	return &MetadataService{orchestrator: o}
}

func TestManualSearch_NoSourceCouldSearch_IsAnError(t *testing.T) {
	down := errors.New("dial tcp: connection refused")
	cases := []struct {
		name      string
		source    string
		providers []metadata.MetadataProvider
	}{
		{"tmdb errors", "tmdb", []metadata.MetadataProvider{
			&stubSourceProvider{source: models.MetadataSourceTMDb, available: true, err: down},
		}},
		{"tmdb circuit open", "tmdb", []metadata.MetadataProvider{
			&stubSourceProvider{source: models.MetadataSourceTMDb, available: true, err: metadata.ErrCircuitOpen},
		}},
		{"tmdb unavailable (no API key)", "tmdb", []metadata.MetadataProvider{
			&stubSourceProvider{source: models.MetadataSourceTMDb, available: false},
		}},
		{"tmdb not registered", "tmdb", nil},
		{"all: tmdb errors, douban and wikipedia off", "all", []metadata.MetadataProvider{
			&stubSourceProvider{source: models.MetadataSourceTMDb, available: true, err: down},
			&stubSourceProvider{source: models.MetadataSourceDouban, available: false},
			&stubSourceProvider{source: models.MetadataSourceWikipedia, available: false},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := manualSearchWith(tc.providers...)
			resp, err := svc.ManualSearch(context.Background(), &ManualSearchRequest{
				Query: "Inception", MediaType: "movie", Source: tc.source,
			})
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrManualSearchSourcesUnavailable)
			assert.Nil(t, resp)
			// The cause travels with it, for the log.
			if len(tc.providers) > 0 && tc.providers[0].IsAvailable() {
				assert.True(t, errors.Is(err, down) || errors.Is(err, metadata.ErrCircuitOpen), "cause kept: %v", err)
			} else {
				assert.ErrorIs(t, err, metadata.ErrSourceUnavailable)
			}
		})
	}
}

// CR LOW-2: a whitespace-only query is a bad request, not a source outage —
// the provider rejects it after trimming, which would read as 503 and count
// against TMDb's circuit breaker.
func TestManualSearchRequest_Validate_WhitespaceQuery(t *testing.T) {
	req := &ManualSearchRequest{Query: "   ", MediaType: "movie", Source: "tmdb"}
	assert.Equal(t, ErrManualSearchQueryRequired, req.Validate())
}

func TestManualSearch_ASourceThatSearched_IsNotAnError(t *testing.T) {
	t.Run("tmdb searched and found nothing: a real empty result", func(t *testing.T) {
		svc := manualSearchWith(&stubSourceProvider{source: models.MetadataSourceTMDb, available: true})
		resp, err := svc.ManualSearch(context.Background(), &ManualSearchRequest{
			Query: "xyznonexistent", MediaType: "movie", Source: "tmdb",
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Results)
		assert.Equal(t, 0, resp.TotalCount)
	})

	t.Run("all: tmdb down but douban answered — the user still gets douban's results", func(t *testing.T) {
		svc := manualSearchWith(
			&stubSourceProvider{source: models.MetadataSourceTMDb, available: true, err: errors.New("timeout")},
			&stubSourceProvider{source: models.MetadataSourceDouban, available: true, items: []metadata.MetadataItem{
				{ID: "30277296", Title: "鬼灭之刃", TitleZhTW: "鬼滅之刃", Year: 2019, MediaType: metadata.MediaTypeTV},
			}},
			&stubSourceProvider{source: models.MetadataSourceWikipedia, available: false},
		)
		resp, err := svc.ManualSearch(context.Background(), &ManualSearchRequest{
			Query: "鬼滅之刃", MediaType: "tv", Source: "all",
		})
		require.NoError(t, err)
		require.Len(t, resp.Results, 1)
		assert.Equal(t, models.MetadataSourceDouban, resp.Results[0].Source)
		assert.Equal(t, []string{"tmdb", "douban", "wikipedia"}, resp.SearchedSources)
	})
}
