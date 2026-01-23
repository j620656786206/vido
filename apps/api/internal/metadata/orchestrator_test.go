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

// [P1] Tests TV media type flows through orchestrator fallback chain correctly
func TestOrchestrator_Search_TVMediaType_Fallback(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		FallbackDelay: 10 * time.Millisecond,
	})

	// TMDb returns no results for TV search
	provider1 := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			assert.Equal(t, MediaTypeTV, req.MediaType)
			return &SearchResult{Items: []MetadataItem{}, TotalCount: 0, Source: models.MetadataSourceTMDb}, nil
		},
	}

	expectedResult := &SearchResult{
		Items: []MetadataItem{
			{ID: "1", Title: "Test TV Show", MediaType: MediaTypeTV},
		},
		Source:     models.MetadataSourceDouban,
		TotalCount: 1,
	}

	// Douban succeeds for TV search
	provider2 := &MockProvider{
		name:      "douban",
		source:    models.MetadataSourceDouban,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			assert.Equal(t, MediaTypeTV, req.MediaType)
			return expectedResult, nil
		},
	}

	orch.RegisterProvider(provider1)
	orch.RegisterProvider(provider2)

	result, status := orch.Search(context.Background(), &SearchRequest{
		Query:     "Test TV Show",
		MediaType: MediaTypeTV,
	})

	require.NotNil(t, result)
	assert.Equal(t, models.MetadataSourceDouban, result.Source)
	assert.Equal(t, MediaTypeTV, result.Items[0].MediaType)

	require.NotNil(t, status)
	assert.Len(t, status.Attempts, 2)
	assert.False(t, status.Attempts[0].Success) // TMDb no results
	assert.True(t, status.Attempts[1].Success)  // Douban succeeds
}

// [P1] Tests year filter is passed through orchestrator correctly
func TestOrchestrator_Search_WithYearFilter(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		FallbackDelay: 10 * time.Millisecond,
	})

	capturedYear := 0
	provider := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			capturedYear = req.Year
			return &SearchResult{
				Items:      []MetadataItem{{ID: "1", Title: "Test Movie", Year: 2024}},
				Source:     models.MetadataSourceTMDb,
				TotalCount: 1,
			}, nil
		},
	}

	orch.RegisterProvider(provider)

	orch.Search(context.Background(), &SearchRequest{
		Query:     "Test",
		MediaType: MediaTypeMovie,
		Year:      2024,
	})

	assert.Equal(t, 2024, capturedYear)
}

// [P2] Tests pagination parameter is passed through orchestrator
func TestOrchestrator_Search_WithPagination(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		FallbackDelay: 10 * time.Millisecond,
	})

	capturedPage := 0
	provider := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			capturedPage = req.Page
			return &SearchResult{
				Items:      []MetadataItem{{ID: "1", Title: "Test Movie"}},
				Source:     models.MetadataSourceTMDb,
				TotalCount: 100,
				Page:       req.Page,
			}, nil
		},
	}

	orch.RegisterProvider(provider)

	result, _ := orch.Search(context.Background(), &SearchRequest{
		Query:     "Test",
		MediaType: MediaTypeMovie,
		Page:      3,
	})

	assert.Equal(t, 3, capturedPage)
	assert.Equal(t, 3, result.Page)
}

// [P2] Tests language parameter is passed through orchestrator
func TestOrchestrator_Search_WithLanguage(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		FallbackDelay: 10 * time.Millisecond,
	})

	capturedLang := ""
	provider := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			capturedLang = req.Language
			return &SearchResult{
				Items:      []MetadataItem{{ID: "1", Title: "測試電影"}},
				Source:     models.MetadataSourceTMDb,
				TotalCount: 1,
			}, nil
		},
	}

	orch.RegisterProvider(provider)

	orch.Search(context.Background(), &SearchRequest{
		Query:     "測試",
		MediaType: MediaTypeMovie,
		Language:  "zh-TW",
	})

	assert.Equal(t, "zh-TW", capturedLang)
}

// [P2] Tests concurrent searches don't interfere with each other
func TestOrchestrator_Search_Concurrent(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		FallbackDelay: 10 * time.Millisecond,
	})

	provider := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			// Simulate some processing time
			time.Sleep(5 * time.Millisecond)
			return &SearchResult{
				Items:      []MetadataItem{{ID: "1", Title: req.Query}},
				Source:     models.MetadataSourceTMDb,
				TotalCount: 1,
			}, nil
		},
	}

	orch.RegisterProvider(provider)

	// Launch 10 concurrent searches
	var wg sync.WaitGroup
	results := make(chan *SearchResult, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(query string) {
			defer wg.Done()
			result, _ := orch.Search(context.Background(), &SearchRequest{
				Query:     query,
				MediaType: MediaTypeMovie,
			})
			if result != nil {
				results <- result
			} else {
				errors <- nil
			}
		}(t.Name() + string(rune('A'+i)))
	}

	wg.Wait()
	close(results)
	close(errors)

	// Verify all searches completed
	count := 0
	for range results {
		count++
	}
	assert.Equal(t, 10, count, "All concurrent searches should complete successfully")
}

// [P1] Tests circuit breaker metrics are tracked correctly
func TestOrchestrator_CircuitBreaker_MetricsTracking(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		FallbackDelay:        10 * time.Millisecond,
		EnableCircuitBreaker: true,
		CircuitBreakerConfig: CircuitBreakerConfig{
			FailureThreshold: 3,
			Timeout:          time.Hour,
		},
	})

	failureCount := 0
	provider := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			failureCount++
			return nil, errors.New("simulated failure")
		},
	}

	orch.RegisterProvider(provider)

	// Trigger failures up to threshold
	for i := 0; i < 3; i++ {
		orch.Search(context.Background(), &SearchRequest{
			Query:     "Test",
			MediaType: MediaTypeMovie,
		})
	}

	assert.Equal(t, 3, failureCount)

	// After circuit opens, provider should not be called
	_, status := orch.Search(context.Background(), &SearchRequest{
		Query:     "Test",
		MediaType: MediaTypeMovie,
	})

	// Still 3 - circuit breaker prevented the call
	assert.Equal(t, 3, failureCount)
	require.NotNil(t, status)
	assert.True(t, status.Attempts[0].Skipped)
	assert.Equal(t, "circuit breaker open", status.Attempts[0].SkipReason)
}

// [P2] Tests GetCircuitBreakerState for existing provider
func TestOrchestrator_GetCircuitBreakerState_Exists(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		EnableCircuitBreaker: true,
		CircuitBreakerConfig: CircuitBreakerConfig{
			FailureThreshold: 5,
		},
	})

	provider := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
	}

	orch.RegisterProvider(provider)

	// GIVEN: A registered provider with circuit breaker enabled
	// WHEN: Getting the circuit breaker state
	state, exists := orch.GetCircuitBreakerState("tmdb")

	// THEN: State should be closed and exists should be true
	assert.True(t, exists)
	assert.Equal(t, CircuitStateClosed, state)
}

// [P2] Tests GetCircuitBreakerState for non-existing provider
func TestOrchestrator_GetCircuitBreakerState_NotExists(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		EnableCircuitBreaker: true,
	})

	// GIVEN: No providers registered
	// WHEN: Getting the circuit breaker state for a non-existing provider
	state, exists := orch.GetCircuitBreakerState("nonexistent")

	// THEN: Should return closed state and exists should be false
	assert.False(t, exists)
	assert.Equal(t, CircuitStateClosed, state)
}

// [P2] Tests GetCircuitBreakerState without circuit breaker enabled
func TestOrchestrator_GetCircuitBreakerState_Disabled(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		EnableCircuitBreaker: false,
	})

	provider := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
	}

	orch.RegisterProvider(provider)

	// GIVEN: Circuit breaker is disabled
	// WHEN: Getting the circuit breaker state
	state, exists := orch.GetCircuitBreakerState("tmdb")

	// THEN: Should return false for exists (no circuit breaker created)
	assert.False(t, exists)
	assert.Equal(t, CircuitStateClosed, state)
}

// [P2] Tests ResetCircuitBreaker for existing provider
func TestOrchestrator_ResetCircuitBreaker_Exists(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		EnableCircuitBreaker: true,
		CircuitBreakerConfig: CircuitBreakerConfig{
			FailureThreshold: 1,
			Timeout:          time.Hour,
		},
	})

	provider := &MockProvider{
		name:      "tmdb",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
		searchFunc: func(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
			return nil, errors.New("simulated failure")
		},
	}

	orch.RegisterProvider(provider)

	// GIVEN: Provider circuit breaker is open
	orch.Search(context.Background(), &SearchRequest{
		Query:     "Test",
		MediaType: MediaTypeMovie,
	})
	state, _ := orch.GetCircuitBreakerState("tmdb")
	assert.Equal(t, CircuitStateOpen, state)

	// WHEN: Resetting the circuit breaker
	orch.ResetCircuitBreaker("tmdb")

	// THEN: Circuit breaker should be closed
	state, _ = orch.GetCircuitBreakerState("tmdb")
	assert.Equal(t, CircuitStateClosed, state)
}

// [P2] Tests ResetCircuitBreaker for non-existing provider (no-op)
func TestOrchestrator_ResetCircuitBreaker_NotExists(t *testing.T) {
	orch := NewOrchestrator(OrchestratorConfig{
		EnableCircuitBreaker: true,
	})

	// GIVEN: No providers registered
	// WHEN: Resetting a non-existing circuit breaker
	// THEN: Should not panic, just no-op
	assert.NotPanics(t, func() {
		orch.ResetCircuitBreaker("nonexistent")
	})
}

// [P2] Tests sourceDisplayName for unknown source
func TestSourceDisplayName_Unknown(t *testing.T) {
	// GIVEN: An unknown metadata source
	source := models.MetadataSource("custom_source")

	// WHEN: Getting display name
	name := sourceDisplayName(source)

	// THEN: Should return the source string as-is
	assert.Equal(t, "custom_source", name)
}

// [P2] Tests AllFailed with cancelled status
func TestFallbackStatus_AllFailed_Cancelled(t *testing.T) {
	// GIVEN: A cancelled status with no attempts
	status := &FallbackStatus{
		Cancelled: true,
		Attempts:  []SourceAttempt{},
	}

	// WHEN: Checking if all failed
	result := status.AllFailed()

	// THEN: Should return true (no successes)
	assert.True(t, result)
}

// [P2] Tests AllFailed with empty attempts
func TestFallbackStatus_AllFailed_Empty(t *testing.T) {
	// GIVEN: Empty attempts
	status := &FallbackStatus{
		Attempts: []SourceAttempt{},
	}

	// WHEN: Checking if all failed
	result := status.AllFailed()

	// THEN: Should return true (no successes possible)
	assert.True(t, result)
}
