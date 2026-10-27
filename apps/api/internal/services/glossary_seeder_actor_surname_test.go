package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/tmdb"
)

// ─── disc-2026-10-glossary-seeds-actor-names-and-fan-surname-a ──────────────

// genreCredits is fakeCreditsClient plus the cached details the real client
// (tmdb.CacheService) has.
type genreCredits struct {
	*fakeCreditsClient
	genres []tmdb.Genre
	err    error
}

func (g *genreCredits) GetTVShowDetails(context.Context, int) (*tmdb.TVShowDetails, error) {
	if g.err != nil {
		return nil, g.err
	}
	return &tmdb.TVShowDetails{Genres: g.genres}, nil
}

func (g *genreCredits) GetMovieDetails(context.Context, int) (*tmdb.MovieDetails, error) {
	if g.err != nil {
		return nil, g.err
	}
	return &tmdb.MovieDetails{Genres: g.genres}, nil
}

// 《百花殺》's shape: an actor and the character she plays.
func blossomsCredits() *fakeCreditsClient {
	return &fakeCreditsClient{
		tv: map[string]*tmdb.TVAggregateCredits{
			"en-US": {Cast: []tmdb.AggregateCast{{ID: 1, Name: "Meng Ziyi", Roles: []tmdb.AggregateRole{{CreditID: "r1", Character: "Shen Xihe"}}}}},
			"zh-TW": {Cast: []tmdb.AggregateCast{{ID: 1, Name: "孟子義", Roles: []tmdb.AggregateRole{{CreditID: "r1", Character: "沈汐和"}}}}},
		},
		movie: map[string]*tmdb.MovieCredits{
			"en-US": {Cast: []tmdb.CreditCast{{CreditID: "c1", Name: "He Yu", Character: "Xiao Huayong"}}},
			"zh-TW": {Cast: []tmdb.CreditCast{{CreditID: "c1", Name: "何與", Character: "蕭華雍"}}},
		},
	}
}

func pairKinds(pairs []CastPair) []string {
	var out []string
	for _, p := range pairs {
		out = append(out, p.Kind+":"+p.Src)
	}
	return out
}

func TestFetchCredits_ScriptedTitlesSeedCharactersOnly(t *testing.T) {
	for _, mediaType := range []string{"tv", "movie"} {
		t.Run(mediaType, func(t *testing.T) {
			client := &genreCredits{fakeCreditsClient: blossomsCredits(), genres: []tmdb.Genre{{ID: 18, Name: "Drama"}}}
			_, pairs, err := NewGlossarySeeder(client, nil, nil, nil).FetchCredits(context.Background(), mediaType, 286506)
			require.NoError(t, err)
			for _, p := range pairs {
				assert.Equal(t, "character", p.Kind, "a drama never says its actors' names: %v", pairKinds(pairs))
			}
			assert.Len(t, pairs, 1)
		})
	}
}

func TestFetchCredits_UnscriptedTitlesKeepActorNames(t *testing.T) {
	for _, g := range []tmdb.Genre{{ID: 10764, Name: "Reality"}, {ID: 10767, Name: "Talk"}, {ID: 10763, Name: "News"}, {ID: 99, Name: "Documentary"}} {
		t.Run(g.Name, func(t *testing.T) {
			client := &genreCredits{fakeCreditsClient: blossomsCredits(), genres: []tmdb.Genre{{ID: 35, Name: "Comedy"}, g}}
			_, pairs, err := NewGlossarySeeder(client, nil, nil, nil).FetchCredits(context.Background(), "tv", 1)
			require.NoError(t, err)
			assert.Contains(t, pairKinds(pairs), "actor:Meng Ziyi", "《種地吧》 calls its cast by name")
		})
	}
}

func TestFetchCredits_UnknownGenreKeepsActorNames(t *testing.T) {
	t.Run("details fetch failed", func(t *testing.T) {
		client := &genreCredits{fakeCreditsClient: blossomsCredits(), err: errors.New("TMDb down")}
		_, pairs, err := NewGlossarySeeder(client, nil, nil, nil).FetchCredits(context.Background(), "movie", 1)
		require.NoError(t, err)
		assert.Contains(t, pairKinds(pairs), "actor:He Yu")
	})
	t.Run("client has no details", func(t *testing.T) {
		_, pairs, err := NewGlossarySeeder(blossomsCredits(), nil, nil, nil).FetchCredits(context.Background(), "movie", 1)
		require.NoError(t, err)
		assert.Contains(t, pairKinds(pairs), "actor:He Yu")
	})
}

// OpenCC s2twp, checked against the CLI 2026-10-27.
func TestRestoreSurname(t *testing.T) {
	cases := []struct{ before, after, want string }{
		{"范帅琦", "範帥琦", "范帥琦"},
		{"范闲", "範閒", "范閒"},
		{"于正", "於正", "于正"},
		{"党国文", "黨國文", "党國文"},
		{"种丹妮", "種丹妮", "种丹妮"},
		{"涂松岩", "塗松巖", "涂松巖"},
		{"叶祖新", "葉祖新", "葉祖新"}, // 葉 is the right surname
		{"钟汉良", "鍾漢良", "鍾漢良"}, // so is 鍾
		{"李范文", "李範文", "李範文"}, // only the first character is a surname
		{"範例", "範例", "範例"},    // already 範: nothing to put back
		{"", "", ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, restoreSurname(c.before, c.after), c.before)
	}
}

func TestSeedFromCredits_PutsTheSurnameBackAfterOpenCC(t *testing.T) {
	repo := &fakeSeedInserter{}
	cc := &surnameOpenCC{}
	seeder := NewGlossarySeeder(nil, repo, cc, nil)

	res := seeder.SeedFromCredits(context.Background(), "tmdb:tv:286506", "m1", []CastPair{
		{Kind: "character", Src: "Fan Xian", Zh: "范闲"},
		{Kind: "character", Src: "Ye Zuxin", Zh: "叶祖新"},
	})

	assert.Equal(t, 2, res.Seeded)
	got := map[string]string{}
	for _, term := range repo.terms {
		got[term.TermSrc] = term.TermZh
	}
	assert.Equal(t, "范閒", got["Fan Xian"])
	assert.Equal(t, "葉祖新", got["Ye Zuxin"])
}

// surnameOpenCC converts the way the real s2twp does for these names.
type surnameOpenCC struct{}

func (surnameOpenCC) IsAvailable() bool { return true }
func (surnameOpenCC) ConvertS2TWP(b []byte) ([]byte, error) {
	return []byte(map[string]string{"范闲\n叶祖新": "範閒\n葉祖新"}[string(b)]), nil
}
func (s surnameOpenCC) ConvertS2TW(b []byte) ([]byte, error) { return s.ConvertS2TWP(b) }
