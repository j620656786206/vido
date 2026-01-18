package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGeminiProvider(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		p := NewGeminiProvider("test-api-key")

		assert.Equal(t, "test-api-key", p.apiKey)
		assert.Equal(t, DefaultGeminiBaseURL, p.baseURL)
		assert.Equal(t, DefaultGeminiModel, p.model)
		assert.Equal(t, time.Duration(DefaultTimeoutSeconds)*time.Second, p.timeout)
		assert.NotNil(t, p.httpClient)
	})

	t.Run("custom configuration", func(t *testing.T) {
		customClient := &http.Client{Timeout: 5 * time.Second}
		p := NewGeminiProvider("test-api-key",
			WithGeminiBaseURL("https://custom.url"),
			WithGeminiModel("custom-model"),
			WithGeminiHTTPClient(customClient),
			WithGeminiTimeout(30*time.Second),
		)

		assert.Equal(t, "https://custom.url", p.baseURL)
		assert.Equal(t, "custom-model", p.model)
		assert.Equal(t, customClient, p.httpClient)
		assert.Equal(t, 30*time.Second, p.timeout)
	})
}

func TestGeminiProvider_Name(t *testing.T) {
	p := NewGeminiProvider("test-key")
	assert.Equal(t, ProviderGemini, p.Name())
}

func TestGeminiProvider_Parse_Success(t *testing.T) {
	// Create mock response
	mockResponse := geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Parts: []geminiPart{
						{Text: `{"title": "Attack on Titan", "year": 2013, "season": 1, "episode": 1, "media_type": "tv", "quality": "1080p", "fansub_group": "SubsPlease", "confidence": 0.95}`},
					},
				},
			},
		},
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "generateContent")
		assert.Contains(t, r.URL.RawQuery, "key=test-api-key")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key",
		WithGeminiBaseURL(server.URL),
	)

	req := &ParseRequest{
		Filename: "[SubsPlease] Shingeki no Kyojin - 01 [1080p].mkv",
	}

	result, err := p.Parse(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "Attack on Titan", result.Title)
	assert.Equal(t, 2013, result.Year)
	assert.Equal(t, 1, result.Season)
	assert.Equal(t, 1, result.Episode)
	assert.Equal(t, "tv", result.MediaType)
	assert.Equal(t, "1080p", result.Quality)
	assert.Equal(t, "SubsPlease", result.FansubGroup)
	assert.Equal(t, 0.95, result.Confidence)
}

func TestGeminiProvider_Parse_MovieSuccess(t *testing.T) {
	mockResponse := geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Parts: []geminiPart{
						{Text: `{"title": "Your Name", "year": 2016, "media_type": "movie", "quality": "1080p", "confidence": 0.9}`},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	req := &ParseRequest{Filename: "Kimi.no.Na.wa.2016.1080p.BluRay.mkv"}

	result, err := p.Parse(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "Your Name", result.Title)
	assert.Equal(t, 2016, result.Year)
	assert.Equal(t, "movie", result.MediaType)
	assert.True(t, result.IsMovie())
}

func TestGeminiProvider_Parse_ValidationError(t *testing.T) {
	p := NewGeminiProvider("test-api-key")
	req := &ParseRequest{Filename: ""}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "filename is required")
}

func TestGeminiProvider_Parse_Timeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key",
		WithGeminiBaseURL(server.URL),
		WithGeminiTimeout(50*time.Millisecond),
	)

	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAITimeout)
}

func TestGeminiProvider_Parse_QuotaExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIQuotaExceeded)
}

func TestGeminiProvider_Parse_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIProviderError)
}

func TestGeminiProvider_Parse_InvalidJSONResponse(t *testing.T) {
	mockResponse := geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Parts: []geminiPart{
						{Text: "not valid json"},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIInvalidResponse)
}

func TestGeminiProvider_Parse_MissingTitle(t *testing.T) {
	mockResponse := geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Parts: []geminiPart{
						{Text: `{"media_type": "movie", "confidence": 0.8}`},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIInvalidResponse)
}

func TestGeminiProvider_Parse_EmptyResponse(t *testing.T) {
	mockResponse := geminiResponse{
		Candidates: []geminiCandidate{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-api-key", WithGeminiBaseURL(server.URL))
	req := &ParseRequest{Filename: "test.mkv"}

	_, err := p.Parse(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIInvalidResponse)
}

func TestGeminiResponse_GetText(t *testing.T) {
	tests := []struct {
		name     string
		response geminiResponse
		want     string
	}{
		{
			name: "normal response",
			response: geminiResponse{
				Candidates: []geminiCandidate{
					{
						Content: geminiContent{
							Parts: []geminiPart{{Text: "hello"}},
						},
					},
				},
			},
			want: "hello",
		},
		{
			name:     "empty candidates",
			response: geminiResponse{Candidates: []geminiCandidate{}},
			want:     "",
		},
		{
			name: "empty parts",
			response: geminiResponse{
				Candidates: []geminiCandidate{
					{Content: geminiContent{Parts: []geminiPart{}}},
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.response.GetText())
		})
	}
}

func TestParseJSONResponse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *ParseResponse
		wantErr bool
	}{
		{
			name:  "valid movie response",
			input: `{"title": "Inception", "year": 2010, "media_type": "movie", "confidence": 0.95}`,
			want: &ParseResponse{
				Title:      "Inception",
				Year:       2010,
				MediaType:  "movie",
				Confidence: 0.95,
			},
			wantErr: false,
		},
		{
			name:  "valid tv response",
			input: `{"title": "Breaking Bad", "season": 1, "episode": 1, "media_type": "tv", "confidence": 0.9}`,
			want: &ParseResponse{
				Title:      "Breaking Bad",
				Season:     1,
				Episode:    1,
				MediaType:  "tv",
				Confidence: 0.9,
			},
			wantErr: false,
		},
		{
			name:    "missing title",
			input:   `{"media_type": "movie", "confidence": 0.8}`,
			wantErr: true,
		},
		{
			name:    "missing media_type",
			input:   `{"title": "Test", "confidence": 0.8}`,
			wantErr: true,
		},
		{
			name:    "invalid media_type",
			input:   `{"title": "Test", "media_type": "anime", "confidence": 0.8}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			input:   `not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseJSONResponse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.Title, result.Title)
				assert.Equal(t, tt.want.MediaType, result.MediaType)
			}
		})
	}
}
