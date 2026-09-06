package tmdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetMovieCreditsWithLanguage(t *testing.T) {
	var gotLanguage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/movie/550/credits", r.URL.Path)
		gotLanguage = r.URL.Query().Get("language")
		assert.NotEmpty(t, r.URL.Query().Get("api_key"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":550,"cast":[{"id":819,"name":"愛德華·諾頓","original_name":"Edward Norton","character":"The Narrator","credit_id":"52fe4250c3a36847f80149f3","order":0}],"crew":[{"id":7467,"name":"David Fincher","original_name":"David Fincher","department":"Directing","job":"Director","credit_id":"x"}]}`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{APIKey: "test-key", BaseURL: server.URL, Language: "zh-TW"})
	result, err := client.GetMovieCreditsWithLanguage(context.Background(), 550, "zh-TW")
	require.NoError(t, err)
	assert.Equal(t, "zh-TW", gotLanguage)
	require.Len(t, result.Cast, 1)
	assert.Equal(t, "愛德華·諾頓", result.Cast[0].Name)
	assert.Equal(t, "Edward Norton", result.Cast[0].OriginalName)
	assert.Equal(t, "The Narrator", result.Cast[0].Character)
	assert.Equal(t, "52fe4250c3a36847f80149f3", result.Cast[0].CreditID)
	require.Len(t, result.Crew, 1)
	assert.Equal(t, "Director", result.Crew[0].Job)

	// The default-language wrapper forwards the client language.
	_, err = client.GetMovieCredits(context.Background(), 550)
	require.NoError(t, err)
	assert.Equal(t, "zh-TW", gotLanguage)

	// An explicit language overrides the client default.
	_, err = client.GetMovieCreditsWithLanguage(context.Background(), 550, "en-US")
	require.NoError(t, err)
	assert.Equal(t, "en-US", gotLanguage)
}

func TestClient_GetMovieCredits_RejectsBadID(t *testing.T) {
	client := NewClient(ClientConfig{APIKey: "test-key"})
	_, err := client.GetMovieCreditsWithLanguage(context.Background(), 0, "en-US")
	require.Error(t, err)
}

func TestClient_GetTVAggregateCreditsWithLanguage(t *testing.T) {
	var gotLanguage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/tv/1396/aggregate_credits", r.URL.Path)
		gotLanguage = r.URL.Query().Get("language")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1396,"cast":[{"id":17419,"name":"Bryan Cranston","original_name":"Bryan Cranston","roles":[{"credit_id":"52542282760ee313280017f9","character":"Walter White","episode_count":62}],"total_episode_count":62,"order":0}],"crew":[{"id":1,"name":"Vince Gilligan","original_name":"Vince Gilligan","department":"Writing","jobs":[{"credit_id":"j","job":"Writer","episode_count":13}],"total_episode_count":13}]}`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{APIKey: "test-key", BaseURL: server.URL})
	result, err := client.GetTVAggregateCreditsWithLanguage(context.Background(), 1396, "en-US")
	require.NoError(t, err)
	assert.Equal(t, "en-US", gotLanguage)
	require.Len(t, result.Cast, 1)
	require.Len(t, result.Cast[0].Roles, 1)
	assert.Equal(t, "Walter White", result.Cast[0].Roles[0].Character)
	assert.Equal(t, 62, result.Cast[0].TotalEpisodeCount)
	require.Len(t, result.Crew, 1)
	assert.Equal(t, "Writer", result.Crew[0].Jobs[0].Job)
}

func TestClient_GetTVAggregateCredits_RejectsBadID(t *testing.T) {
	client := NewClient(ClientConfig{APIKey: "test-key"})
	_, err := client.GetTVAggregateCreditsWithLanguage(context.Background(), -1, "en-US")
	require.Error(t, err)
}
