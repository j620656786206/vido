package metadata

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

func TestNewWikipediaProvider(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{
		Enabled: true,
	})

	assert.NotNil(t, provider)
	assert.Equal(t, "Wikipedia", provider.Name())
	assert.Equal(t, models.MetadataSourceWikipedia, provider.Source())
	assert.True(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusAvailable, provider.Status())
}

func TestWikipediaProvider_Disabled(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{
		Enabled: false,
	})

	assert.False(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusUnavailable, provider.Status())
}

func TestWikipediaProvider_Search_NotImplemented(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{
		Enabled: true,
	})

	result, err := provider.Search(context.Background(), &SearchRequest{
		Query:     "Test Movie",
		MediaType: MediaTypeMovie,
	})

	// Should return error indicating not implemented
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestWikipediaProvider_Search_Disabled(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{
		Enabled: false,
	})

	result, err := provider.Search(context.Background(), &SearchRequest{
		Query:     "Test Movie",
		MediaType: MediaTypeMovie,
	})

	// Should return error indicating provider is disabled
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "disabled")
}

func TestWikipediaProvider_RateLimiter(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{
		Enabled:       true,
		RateLimitPerSecond: 2, // 2 requests per second for faster testing
	})

	// Make 3 rapid requests
	start := time.Now()
	for i := 0; i < 3; i++ {
		provider.Search(context.Background(), &SearchRequest{
			Query:     "Test",
			MediaType: MediaTypeMovie,
		})
	}
	elapsed := time.Since(start)

	// Should have been rate limited - at least 1 second for 3 requests at 2/s
	// But since this is a stub that returns immediately, we're just testing
	// that the rate limiter doesn't crash. Full rate limiting tests will be
	// in Story 3.5 when actual API calls are made.
	assert.NotNil(t, provider)
	// The stub returns immediately, so we just verify it doesn't block forever
	assert.Less(t, elapsed, 5*time.Second)
}

func TestWikipediaProvider_SetEnabled(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{
		Enabled: true,
	})

	assert.True(t, provider.IsAvailable())

	provider.SetEnabled(false)
	assert.False(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusUnavailable, provider.Status())

	provider.SetEnabled(true)
	assert.True(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusAvailable, provider.Status())
}

func TestWikipediaProvider_DefaultRateLimit(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{
		Enabled: true,
		// RateLimitPerSecond not set, should default to 1
	})

	// Just verify provider is created successfully
	assert.NotNil(t, provider)
	assert.Equal(t, "Wikipedia", provider.Name())
}

func TestWikipediaProvider_ImplementsInterface(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{})

	// Compile-time interface verification
	var _ MetadataProvider = provider
}
