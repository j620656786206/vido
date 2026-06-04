package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_SearchPeople(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/search/person", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("query"))
		assert.NotEmpty(t, r.URL.Query().Get("language"))
		assert.NotEmpty(t, r.URL.Query().Get("api_key"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SearchResultPeople{
			Page: 1,
			Results: []Person{
				{ID: 5655, Name: "Makoto Shinkai", OriginalName: "新海誠", KnownForDepartment: "Directing"},
			},
			TotalPages:   1,
			TotalResults: 1,
		})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{APIKey: "test-key", BaseURL: server.URL})

	result, err := client.SearchPeople(context.Background(), "shinkai", 1)
	require.NoError(t, err)
	require.Len(t, result.Results, 1)
	assert.Equal(t, 5655, result.Results[0].ID)
	assert.Equal(t, "新海誠", result.Results[0].OriginalName)
	assert.Equal(t, "Directing", result.Results[0].KnownForDepartment)
}

func TestClient_SearchPeople_EmptyQuery(t *testing.T) {
	client := NewClient(ClientConfig{APIKey: "test-key"})

	result, err := client.SearchPeople(context.Background(), "", 1)
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestClient_SearchPeopleWithLanguage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "en", r.URL.Query().Get("language"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SearchResultPeople{Page: 1, Results: []Person{}})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{APIKey: "test-key", BaseURL: server.URL})

	result, err := client.SearchPeopleWithLanguage(context.Background(), "test", "en", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
