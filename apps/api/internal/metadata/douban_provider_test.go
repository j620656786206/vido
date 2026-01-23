package metadata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

func TestNewDoubanProvider(t *testing.T) {
	provider := NewDoubanProvider(DoubanProviderConfig{
		Enabled: true,
	})

	assert.NotNil(t, provider)
	assert.Equal(t, "Douban", provider.Name())
	assert.Equal(t, models.MetadataSourceDouban, provider.Source())
	assert.True(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusAvailable, provider.Status())
}

func TestDoubanProvider_Disabled(t *testing.T) {
	provider := NewDoubanProvider(DoubanProviderConfig{
		Enabled: false,
	})

	assert.False(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusUnavailable, provider.Status())
}

func TestDoubanProvider_Search_NotImplemented(t *testing.T) {
	provider := NewDoubanProvider(DoubanProviderConfig{
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

func TestDoubanProvider_Search_Disabled(t *testing.T) {
	provider := NewDoubanProvider(DoubanProviderConfig{
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

func TestDoubanProvider_SetEnabled(t *testing.T) {
	provider := NewDoubanProvider(DoubanProviderConfig{
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

func TestDoubanProvider_ImplementsInterface(t *testing.T) {
	provider := NewDoubanProvider(DoubanProviderConfig{})

	// Compile-time interface verification
	var _ MetadataProvider = provider
}
