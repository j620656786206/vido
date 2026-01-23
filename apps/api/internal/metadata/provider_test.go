package metadata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// MockProvider implements MetadataProvider for testing
type MockProvider struct {
	name        string
	source      models.MetadataSource
	available   bool
	status      ProviderStatus
	searchFunc  func(ctx context.Context, req *SearchRequest) (*SearchResult, error)
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Source() models.MetadataSource {
	return m.source
}

func (m *MockProvider) IsAvailable() bool {
	return m.available
}

func (m *MockProvider) Status() ProviderStatus {
	return m.status
}

func (m *MockProvider) Search(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, req)
	}
	return &SearchResult{
		Items:      []MetadataItem{},
		Source:     m.source,
		TotalCount: 0,
	}, nil
}

// Compile-time interface verification
var _ MetadataProvider = (*MockProvider)(nil)

func TestProviderStatus_String(t *testing.T) {
	tests := []struct {
		status   ProviderStatus
		expected string
	}{
		{ProviderStatusAvailable, "available"},
		{ProviderStatusUnavailable, "unavailable"},
		{ProviderStatusRateLimited, "rate_limited"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestProviderStatus_IsValid(t *testing.T) {
	tests := []struct {
		status   ProviderStatus
		expected bool
	}{
		{ProviderStatusAvailable, true},
		{ProviderStatusUnavailable, true},
		{ProviderStatusRateLimited, true},
		{ProviderStatus("invalid"), false},
		{ProviderStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsValid())
		})
	}
}

func TestSearchRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     *SearchRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid movie request",
			req: &SearchRequest{
				Query:     "Inception",
				MediaType: MediaTypeMovie,
				Language:  "zh-TW",
			},
			wantErr: false,
		},
		{
			name: "valid tv request with year",
			req: &SearchRequest{
				Query:     "Game of Thrones",
				MediaType: MediaTypeTV,
				Year:      2011,
				Language:  "en",
			},
			wantErr: false,
		},
		{
			name: "empty query",
			req: &SearchRequest{
				Query:     "",
				MediaType: MediaTypeMovie,
			},
			wantErr: true,
			errMsg:  "query is required",
		},
		{
			name: "invalid media type",
			req: &SearchRequest{
				Query:     "Test",
				MediaType: MediaType("invalid"),
			},
			wantErr: true,
			errMsg:  "invalid media type",
		},
		{
			name: "empty media type defaults to movie",
			req: &SearchRequest{
				Query:     "Test",
				MediaType: "",
				Language:  "zh-TW",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSearchResult_HasResults(t *testing.T) {
	tests := []struct {
		name     string
		result   *SearchResult
		expected bool
	}{
		{
			name: "with results",
			result: &SearchResult{
				Items: []MetadataItem{
					{ID: "1", Title: "Test"},
				},
				TotalCount: 1,
			},
			expected: true,
		},
		{
			name: "empty items",
			result: &SearchResult{
				Items:      []MetadataItem{},
				TotalCount: 0,
			},
			expected: false,
		},
		{
			name: "nil items",
			result: &SearchResult{
				Items:      nil,
				TotalCount: 0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.HasResults())
		})
	}
}

func TestMetadataItem_HasTitle(t *testing.T) {
	tests := []struct {
		name     string
		item     MetadataItem
		expected bool
	}{
		{
			name:     "has title",
			item:     MetadataItem{Title: "Test Movie"},
			expected: true,
		},
		{
			name:     "has zh-TW title only",
			item:     MetadataItem{TitleZhTW: "測試電影"},
			expected: true,
		},
		{
			name:     "has both titles",
			item:     MetadataItem{Title: "Test", TitleZhTW: "測試"},
			expected: true,
		},
		{
			name:     "no title",
			item:     MetadataItem{ID: "123"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.item.HasTitle())
		})
	}
}

func TestMetadataItem_GetDisplayTitle(t *testing.T) {
	tests := []struct {
		name     string
		item     MetadataItem
		lang     string
		expected string
	}{
		{
			name:     "zh-TW preference with zh-TW title",
			item:     MetadataItem{Title: "Inception", TitleZhTW: "全面啟動"},
			lang:     "zh-TW",
			expected: "全面啟動",
		},
		{
			name:     "zh-TW preference without zh-TW title",
			item:     MetadataItem{Title: "Inception"},
			lang:     "zh-TW",
			expected: "Inception",
		},
		{
			name:     "en preference",
			item:     MetadataItem{Title: "Inception", TitleZhTW: "全面啟動"},
			lang:     "en",
			expected: "Inception",
		},
		{
			name:     "fallback to any title",
			item:     MetadataItem{TitleZhTW: "全面啟動"},
			lang:     "en",
			expected: "全面啟動",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.item.GetDisplayTitle(tt.lang))
		})
	}
}

func TestMockProvider_ImplementsInterface(t *testing.T) {
	provider := &MockProvider{
		name:      "test",
		source:    models.MetadataSourceTMDb,
		available: true,
		status:    ProviderStatusAvailable,
	}

	assert.Equal(t, "test", provider.Name())
	assert.Equal(t, models.MetadataSourceTMDb, provider.Source())
	assert.True(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusAvailable, provider.Status())

	result, err := provider.Search(context.Background(), &SearchRequest{
		Query:     "Test",
		MediaType: MediaTypeMovie,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.MetadataSourceTMDb, result.Source)
}

func TestMediaType_IsValid(t *testing.T) {
	tests := []struct {
		mediaType MediaType
		expected  bool
	}{
		{MediaTypeMovie, true},
		{MediaTypeTV, true},
		{MediaType("invalid"), false},
		{MediaType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mediaType), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.mediaType.IsValid())
		})
	}
}

// [P2] Tests ProviderError.Unwrap returns the underlying error
func TestProviderError_Unwrap(t *testing.T) {
	// GIVEN: A ProviderError wrapping another error
	underlyingErr := assert.AnError
	providerErr := NewProviderError(
		"TMDb",
		models.MetadataSourceTMDb,
		ErrCodeUnavailable,
		"API call failed",
		underlyingErr,
	)

	// WHEN: Unwrapping the error
	unwrapped := providerErr.Unwrap()

	// THEN: Should return the underlying error
	assert.Equal(t, underlyingErr, unwrapped)
}

// [P2] Tests ProviderError.Unwrap returns nil when no underlying error
func TestProviderError_Unwrap_Nil(t *testing.T) {
	// GIVEN: A ProviderError without underlying error
	providerErr := NewProviderError(
		"TMDb",
		models.MetadataSourceTMDb,
		ErrCodeUnavailable,
		"API unavailable",
		nil,
	)

	// WHEN: Unwrapping the error
	unwrapped := providerErr.Unwrap()

	// THEN: Should return nil
	assert.Nil(t, unwrapped)
}

// [P2] Tests ProviderError.Error with underlying error
func TestProviderError_Error_WithUnderlying(t *testing.T) {
	// GIVEN: A ProviderError with underlying error
	underlyingErr := assert.AnError
	providerErr := NewProviderError(
		"TMDb",
		models.MetadataSourceTMDb,
		ErrCodeUnavailable,
		"API call failed",
		underlyingErr,
	)

	// WHEN: Getting error string
	errStr := providerErr.Error()

	// THEN: Should include provider name, message, and underlying error
	assert.Contains(t, errStr, "TMDb")
	assert.Contains(t, errStr, "API call failed")
	assert.Contains(t, errStr, underlyingErr.Error())
}

// [P2] Tests ProviderError.Error without underlying error
func TestProviderError_Error_WithoutUnderlying(t *testing.T) {
	// GIVEN: A ProviderError without underlying error
	providerErr := NewProviderError(
		"Douban",
		models.MetadataSourceDouban,
		ErrCodeUnavailable,
		"Not implemented",
		nil,
	)

	// WHEN: Getting error string
	errStr := providerErr.Error()

	// THEN: Should include provider name and message only
	assert.Equal(t, "Douban: Not implemented", errStr)
}

// [P2] Tests NewProviderError creates correct error
func TestNewProviderError(t *testing.T) {
	// GIVEN: Error details
	provider := "Wikipedia"
	source := models.MetadataSourceWikipedia
	code := ErrCodeRateLimited
	message := "Rate limit exceeded"
	underlyingErr := assert.AnError

	// WHEN: Creating a ProviderError
	providerErr := NewProviderError(provider, source, code, message, underlyingErr)

	// THEN: All fields should be set correctly
	assert.Equal(t, provider, providerErr.Provider)
	assert.Equal(t, source, providerErr.Source)
	assert.Equal(t, code, providerErr.Code)
	assert.Equal(t, message, providerErr.Message)
	assert.Equal(t, underlyingErr, providerErr.Err)
}
