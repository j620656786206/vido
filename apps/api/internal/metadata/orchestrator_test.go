package metadata

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

func TestNewOrchestrator(t *testing.T) {
	cfg := OrchestratorConfig{
		FallbackDelay: 100 * time.Millisecond,
	}
	orch := NewOrchestrator(cfg)

	assert.NotNil(t, orch)
	assert.Empty(t, orch.Providers())
}

func TestOrchestrator_RegisterProvider(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{})

	provider := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
	}

	orch.RegisterProvider(provider)

	providers := orch.Providers()
	assert.Len(t, providers, 1)
	assert.Equal(t, "tmdb", providers[0].Name())
}

func TestOrchestrator_Search_FirstProviderSuccess(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		FallbackDelay: 10 * time.Millisecond,
	})

	expectedResult := &SearchResult{
		Items: []MetadataItem{
			{ID: "123", Title: "Test Movie"},
		},
		Source:     models.MetadataSourceTMDb,
		TotalCount: 1,
	}

	provider1 := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			return expectedResult, nil
		},
	}

	provider2 := &MockProvider{
		name:      "douban",
		source:    models.MetadataSourceDouban,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			t.Error("second provider should not be called")
			return nil, errors.New("should not reach here")
		},
	}

	orch.RegisterProvider(provider1)
	orch.RegisterProvider(provider2)

	result, status := orch.Search(context.Background(), &SearchRequest{
		Query:     "Test",
		MediaType: MediaTypeMovie,
	})

	require.NotNil(t, result)
	assert.Equal(t, models.MetadataSourceTMDb, result.Source)
	assert.Len(t, result.Items, 1)

	require.NotNil(t, status)
	assert.Len(t, status.Attempts, 1)
	assert.True(t, status.Attempts[0].Success)
	assert.Equal(t, models.MetadataSourceTMDb, status.Attempts[0].Source)
}

func TestOrchestrator_Search_FallbackOnNoResults(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		FallbackDelay: 10 * time.Millisecond,
	})

	provider1 := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			return &SearchResult{Items: []MetadataItem{}, TotalCount: 0, Source: models.MetadataSourceTMDb}, nil
		},
	}

	expectedResult := &SearchResult{
		Items: []MetadataItem{
			{ID: "456", Title: "Test from Douban"},
		},
		Source:     models.MetadataSourceDouban,
		TotalCount: 1,
	}

	provider2 := &MockProvider{
		name:      "douban",
		source:    models.MetadataSourceDouban,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			return expectedResult, nil
		},
	}

	orch.RegisterProvider(provider1)
	orch.RegisterProvider(provider2)

	result, status := orch.Search(context.Background(), &SearchRequest{
		Query:     "Test",
		MediaType: MediaTypeMovie,
	})

	require.NotNil(t, result)
	assert.Equal(t, models.MetadataSourceDouban, result.Source)

	require.NotNil(t, status)
	assert.Len(t, status.Attempts, 2)
	assert.False(t, status.Attempts[0].Success)
	assert.True(t, status.Attempts[1].Success)
}

func TestOrchestrator_Search_FallbackOnError(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		FallbackDelay: 10 * time.Millisecond,
	})

	provider1 := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			return nil, errors.New("TMDb API error")
		},
	}

	expectedResult := &SearchResult{
		Items: []MetadataItem{
			{ID: "789", Title: "Test from Wikipedia"},
		},
		Source:     models.MetadataSourceWikipedia,
		TotalCount: 1,
	}

	provider2 := &MockProvider{
		name:      "wikipedia",
		source:    models.MetadataSourceWikipedia,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			return expectedResult, nil
		},
	}

	orch.RegisterProvider(provider1)
	orch.RegisterProvider(provider2)

	result, status := orch.Search(context.Background(), &SearchRequest{
		Query:     "Test",
		MediaType: MediaTypeMovie,
	})

	require.NotNil(t, result)
	assert.Equal(t, models.MetadataSourceWikipedia, result.Source)

	require.NotNil(t, status)
	assert.Len(t, status.Attempts, 2)
	assert.False(t, status.Attempts[0].Success)
	assert.NotNil(t, status.Attempts[0].Error)
}

func TestOrchestrator_Search_SkipsUnavailableProvider(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		FallbackDelay: 10 * time.Millisecond,
	})

	provider1 := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: false, // Unavailable
		status:    ProviderStatusUnavailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			t.Error("unavailable provider should not be called")
			return nil, errors.New("should not reach here")
		},
	}

	expectedResult := &SearchResult{
		Items: []MetadataItem{
			{ID: "123", Title: "Test"},
		},
		Source:     models.MetadataSourceDouban,
		TotalCount: 1,
	}

	provider2 := &MockProvider{
		name:      "douban",
		source:    models.MetadataSourceDouban,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			return expectedResult, nil
		},
	}

	orch.RegisterProvider(provider1)
	orch.RegisterProvider(provider2)

	result, status := orch.Search(context.Background(), &SearchRequest{
		Query:     "Test",
		MediaType: MediaTypeMovie,
	})

	require.NotNil(t, result)
	assert.Equal(t, models.MetadataSourceDouban, result.Source)

	require.NotNil(t, status)
	// First provider should be marked as skipped
	assert.Len(t, status.Attempts, 2)
	assert.True(t, status.Attempts[0].Skipped)
	assert.True(t, status.Attempts[1].Success)
}

func TestOrchestrator_Search_AllProvidersFail(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		FallbackDelay: 10 * time.Millisecond,
	})

	provider1 := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			return nil, errors.New("TMDb error")
		},
	}

	provider2 := &MockProvider{
		name:      "douban",
		source:    models.MetadataSourceDouban,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			return nil, errors.New("Douban error")
		},
	}

	orch.RegisterProvider(provider1)
	orch.RegisterProvider(provider2)

	result, status := orch.Search(context.Background(), &SearchRequest{
		Query:     "Test",
		MediaType: MediaTypeMovie,
	})

	assert.Nil(t, result)

	require.NotNil(t, status)
	assert.Len(t, status.Attempts, 2)
	assert.False(t, status.Attempts[0].Success)
	assert.False(t, status.Attempts[1].Success)
	assert.True(t, status.AllFailed())
}

func TestOrchestrator_Search_CircuitBreakerSkips(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		FallbackDelay:    10 * time.Millisecond,
		EnableCircuitBreaker: true,
		CircuitBreakerConfig: CircuitBreakerConfig{
			FailureThreshold: 1,
			Timeout:          time.Hour, // Long timeout to keep open
		},
	})

	callCount := 0
	provider1 := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			callCount++
			return nil, errors.New("TMDb error")
		},
	}

	provider2 := &MockProvider{
		name:      "douban",
		source:    models.MetadataSourceDouban,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			return &SearchResult{
				Items:      []MetadataItem{{ID: "1", Title: "Test"}},
				Source:     models.MetadataSourceDouban,
				TotalCount: 1,
			}, nil
		},
	}

	orch.RegisterProvider(provider1)
	orch.RegisterProvider(provider2)

	// First call - triggers circuit breaker
	orch.Search(context.Background(), &SearchRequest{
		Query:     "Test",
		MediaType: MediaTypeMovie,
	})
	assert.Equal(t, 1, callCount)

	// Second call - circuit breaker should skip provider1
	result, status := orch.Search(context.Background(), &SearchRequest{
		Query:     "Test2",
		MediaType: MediaTypeMovie,
	})

	assert.Equal(t, 1, callCount) // Should not have called provider1 again
	require.NotNil(t, result)
	assert.Equal(t, models.MetadataSourceDouban, result.Source)

	// Check that TMDb was skipped due to circuit breaker
	require.NotNil(t, status)
	assert.True(t, status.Attempts[0].Skipped)
	assert.Equal(t, "circuit breaker open", status.Attempts[0].SkipReason)
}

func TestOrchestrator_Search_FallbackDelayRespected(t *testing.T) {
	delay := 50 * time.Millisecond
	orch := NewOrchestrator(OrchestratorConfig{
		FallbackDelay: delay,
	})

	provider1 := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			return &SearchResult{Items: []MetadataItem{}, TotalCount: 0, Source: models.MetadataSourceTMDb}, nil
		},
	}

	provider2 := &MockProvider{
		name:      "douban",
		source:    models.MetadataSourceDouban,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			return &SearchResult{
				Items:      []MetadataItem{{ID: "1", Title: "Test"}},
				Source:     models.MetadataSourceDouban,
				TotalCount: 1,
			}, nil
		},
	}

	orch.RegisterProvider(provider1)
	orch.RegisterProvider(provider2)

	start := time.Now()
	orch.Search(context.Background(), &SearchRequest{
		Query:     "Test",
		MediaType: MediaTypeMovie,
	})
	elapsed := time.Since(start)

	// Should have waited at least the fallback delay
	assert.GreaterOrEqual(t, elapsed, delay)
	// But not too much more
	assert.Less(t, elapsed, delay+50*time.Millisecond)
}

func TestOrchestrator_Search_ContextCancellation(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		FallbackDelay: 100 * time.Millisecond,
	})

	provider := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			// Simulate slow provider
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Second):
				return &SearchResult{Items: []MetadataItem{{ID: "1"}}}, nil
			}
		},
	}

	orch.RegisterProvider(provider)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, status := orch.Search(ctx, &SearchRequest{
		Query:     "Test",
		MediaType: MediaTypeMovie,
	})

	assert.Nil(t, result)
	require.NotNil(t, status)
	assert.True(t, status.Cancelled)
}

func TestOrchestrator_Search_ProgressCallback(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		FallbackDelay: 10 * time.Millisecond,
	})

	provider1 := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			return &SearchResult{Items: []MetadataItem{}, TotalCount: 0, Source: models.MetadataSourceTMDb}, nil
		},
	}

	provider2 := &MockProvider{
		name:      "douban",
		source:    models.MetadataSourceDouban,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			return &SearchResult{
				Items:      []MetadataItem{{ID: "1", Title: "Test"}},
				Source:     models.MetadataSourceDouban,
				TotalCount: 1,
			}, nil
		},
	}

	orch.RegisterProvider(provider1)
	orch.RegisterProvider(provider2)

	var progressEvents []SourceAttempt
	var mu sync.Mutex

	orch.Search(context.Background(), &SearchRequest{
		Query:     "Test",
		MediaType: MediaTypeMovie,
	}, WithProgressCallback(func(attempt SourceAttempt) {
		mu.Lock()
		progressEvents = append(progressEvents, attempt)
		mu.Unlock()
	}))

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, progressEvents, 2)
	assert.Equal(t, models.MetadataSourceTMDb, progressEvents[0].Source)
	assert.Equal(t, models.MetadataSourceDouban, progressEvents[1].Source)
}

func TestFallbackStatus_StatusString(t *testing.T) {
	tests := []struct {
		name     string
		status   *FallbackStatus
		expected string
	}{
		{
			name: "single success",
			status: &FallbackStatus{
				Attempts: []SourceAttempt{
					{Source: models.MetadataSourceTMDb, Success: true},
				},
			},
			expected: "TMDb ✓",
		},
		{
			name: "fallback success",
			status: &FallbackStatus{
				Attempts: []SourceAttempt{
					{Source: models.MetadataSourceTMDb, Success: false},
					{Source: models.MetadataSourceDouban, Success: true},
				},
			},
			expected: "TMDb ❌ → Douban ✓",
		},
		{
			name: "all failed",
			status: &FallbackStatus{
				Attempts: []SourceAttempt{
					{Source: models.MetadataSourceTMDb, Success: false},
					{Source: models.MetadataSourceDouban, Success: false},
					{Source: models.MetadataSourceWikipedia, Success: false},
				},
			},
			expected: "TMDb ❌ → Douban ❌ → Wikipedia ❌ → Manual search",
		},
		{
			name: "skipped provider",
			status: &FallbackStatus{
				Attempts: []SourceAttempt{
					{Source: models.MetadataSourceTMDb, Skipped: true},
					{Source: models.MetadataSourceDouban, Success: true},
				},
			},
			expected: "TMDb ⏭ → Douban ✓",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.StatusString())
		})
	}
}

func TestFallbackStatus_AllFailed(t *testing.T) {
	tests := []struct {
		name     string
		status   *FallbackStatus
		expected bool
	}{
		{
			name: "some success",
			status: &FallbackStatus{
				Attempts: []SourceAttempt{
					{Success: false},
					{Success: true},
				},
			},
			expected: false,
		},
		{
			name: "all failed",
			status: &FallbackStatus{
				Attempts: []SourceAttempt{
					{Success: false},
					{Success: false},
				},
			},
			expected: true,
		},
		{
			name: "all skipped",
			status: &FallbackStatus{
				Attempts: []SourceAttempt{
					{Skipped: true},
					{Skipped: true},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.AllFailed())
		})
	}
}
