package tmdb

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCreditsClient counts upstream calls per (endpoint, language).
type fakeCreditsClient struct {
	calls map[string]int
	err   error
}

func (f *fakeCreditsClient) key(kind string, id int, lang string) string {
	if f.calls == nil {
		f.calls = map[string]int{}
	}
	k := kind + "/" + itoa(id) + "?" + lang
	f.calls[k]++
	return k
}

func (f *fakeCreditsClient) GetMovieCreditsWithLanguage(_ context.Context, id int, lang string) (*MovieCredits, error) {
	f.key("movie", id, lang)
	if f.err != nil {
		return nil, f.err
	}
	return &MovieCredits{ID: id, Cast: []CreditCast{{ID: 1, Name: "name-" + lang, CreditID: "c1"}}}, nil
}

func (f *fakeCreditsClient) GetTVAggregateCreditsWithLanguage(_ context.Context, id int, lang string) (*TVAggregateCredits, error) {
	f.key("tv", id, lang)
	if f.err != nil {
		return nil, f.err
	}
	return &TVAggregateCredits{ID: id, Cast: []AggregateCast{{ID: 1, Name: "name-" + lang}}}, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// sub-7-3: credits are cached WITH the language in the key — the seeder asks
// for en-US and zh-TW of the same title, and both must be their own entry.
func TestCacheService_Credits_LanguageKeyedCacheMissThenHit(t *testing.T) {
	repo := NewMockCacheRepository()
	raw := &fakeCreditsClient{}
	svc := NewCacheService(&MockFallbackClient{}, repo, CacheServiceConfig{TTL: DefaultCacheTTL})
	svc.SetCreditsClient(raw)
	ctx := context.Background()

	zh, err := svc.GetMovieCreditsWithLanguage(ctx, 550, "zh-TW")
	require.NoError(t, err)
	assert.Equal(t, "name-zh-TW", zh.Cast[0].Name)
	en, err := svc.GetMovieCreditsWithLanguage(ctx, 550, "en-US")
	require.NoError(t, err)
	assert.Equal(t, "name-en-US", en.Cast[0].Name, "a different language is a different entry, not a hit")
	assert.Equal(t, DefaultCacheTTL, repo.lastSetTTL)

	a, _ := repo.Get(ctx, "tmdb:movie/550/credits:zh-TW")
	b, _ := repo.Get(ctx, "tmdb:movie/550/credits:en-US")
	require.NotNil(t, a)
	require.NotNil(t, b)

	// Second round: both hit, no upstream calls.
	_, err = svc.GetMovieCreditsWithLanguage(ctx, 550, "zh-TW")
	require.NoError(t, err)
	_, err = svc.GetMovieCreditsWithLanguage(ctx, 550, "en-US")
	require.NoError(t, err)
	assert.Equal(t, 1, raw.calls["movie/550?zh-TW"])
	assert.Equal(t, 1, raw.calls["movie/550?en-US"])

	// TV twin.
	tv, err := svc.GetTVAggregateCreditsWithLanguage(ctx, 1396, "zh-TW")
	require.NoError(t, err)
	assert.Equal(t, "name-zh-TW", tv.Cast[0].Name)
	_, err = svc.GetTVAggregateCreditsWithLanguage(ctx, 1396, "zh-TW")
	require.NoError(t, err)
	assert.Equal(t, 1, raw.calls["tv/1396?zh-TW"])
	c, _ := repo.Get(ctx, "tmdb:tv/1396/aggregate_credits:zh-TW")
	require.NotNil(t, c)
}

func TestCacheService_Credits_UpstreamErrorIsNotCached(t *testing.T) {
	repo := NewMockCacheRepository()
	raw := &fakeCreditsClient{err: errors.New("429")}
	svc := NewCacheService(&MockFallbackClient{}, repo, CacheServiceConfig{})
	svc.SetCreditsClient(raw)

	_, err := svc.GetMovieCreditsWithLanguage(context.Background(), 550, "zh-TW")
	require.Error(t, err)
	got, _ := repo.Get(context.Background(), "tmdb:movie/550/credits:zh-TW")
	assert.Nil(t, got)
}

func TestCacheService_Credits_NilClient(t *testing.T) {
	repo := NewMockCacheRepository()
	svc := NewCacheService(&MockFallbackClient{}, repo, CacheServiceConfig{})
	_, err := svc.GetMovieCreditsWithLanguage(context.Background(), 550, "zh-TW")
	require.Error(t, err, "must error (not panic) when the credits client is unset")
	_, err = svc.GetTVAggregateCreditsWithLanguage(context.Background(), 1396, "zh-TW")
	require.Error(t, err)
}
