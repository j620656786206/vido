package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/tmdb"
	"github.com/vido/api/internal/wikipedia"
)

// ─── disc-2026-10-glossary-seeds-actor-names-and-fan-surname-b ──────────────

// titleCredits is fakeCreditsClient plus details with a name and a language.
type titleCredits struct {
	*fakeCreditsClient
	show tmdb.TVShowDetails
}

func (c *titleCredits) GetTVShowDetails(context.Context, int) (*tmdb.TVShowDetails, error) {
	return &c.show, nil
}

func (c *titleCredits) GetMovieDetails(context.Context, int) (*tmdb.MovieDetails, error) {
	return &tmdb.MovieDetails{}, nil
}

type fakeWiki struct {
	pages map[string]string
	err   error
	asked []string
}

func (w *fakeWiki) GetWikitext(_ context.Context, titles ...string) (string, string, error) {
	w.asked = append(w.asked, titles...)
	if w.err != nil {
		return "", "", w.err
	}
	for _, t := range titles {
		if text, ok := w.pages[t]; ok {
			return text, t, nil
		}
	}
	return "", "", &wikipedia.NotFoundError{PageTitle: strings.Join(titles, "/")}
}

// toTWOpenCC converts the one character these fixtures need.
type toTWOpenCC struct{}

func (toTWOpenCC) IsAvailable() bool { return true }
func (toTWOpenCC) ConvertS2TWP(b []byte) ([]byte, error) {
	return []byte(strings.ReplaceAll(string(b), "与", "與")), nil
}
func (c toTWOpenCC) ConvertS2TW(b []byte) ([]byte, error) { return c.ConvertS2TWP(b) }

// 《百花殺》 on TMDb: Chinese actor names, pinyin character names in every language.
func blossomsShow(language string) *titleCredits {
	cast := func(actorMeng, actorHe, roleMeng, roleHe string) *tmdb.TVAggregateCredits {
		return &tmdb.TVAggregateCredits{Cast: []tmdb.AggregateCast{
			{ID: 1, Name: actorMeng, TotalEpisodeCount: 40, Roles: []tmdb.AggregateRole{{CreditID: "r1", Character: roleMeng}}},
			{ID: 2, Name: actorHe, TotalEpisodeCount: 39, Roles: []tmdb.AggregateRole{{CreditID: "r2", Character: roleHe}}},
		}}
	}
	return &titleCredits{
		fakeCreditsClient: &fakeCreditsClient{tv: map[string]*tmdb.TVAggregateCredits{
			"en-US": cast("Meng Ziyi", "He Yu", "Shen Xihe", "Xiao Huayong"),
			"zh-TW": cast("孟子義", "何與", "Shen Xihe", "Xiao Huayong"),
		}},
		show: tmdb.TVShowDetails{
			TVShow: tmdb.TVShow{Name: "百花殺", OriginalName: "百花杀", OriginalLanguage: language, FirstAirDate: "2026-07-09"},
			Genres: []tmdb.Genre{{ID: 18, Name: "Drama"}},
		},
	}
}

const blossomsWiki = `==演員陣容==
{|class="wikitable"
|-
!演員
!角色
|-
|[[何与]]
|蕭華雍 ||七皇子
|-
|[[孟子義]]
|沈汐和 ||昭寧郡主
|}`

func characterPairs(pairs []CastPair) map[string]string {
	out := map[string]string{}
	for _, p := range pairs {
		if p.Kind == "character" {
			out[p.Src] = p.Zh
		}
	}
	return out
}

func TestFetchCredits_ChineseCharacterNamesComeFromZhWikipedia(t *testing.T) {
	wiki := &fakeWiki{pages: map[string]string{"百花殺": blossomsWiki}}
	seeder := NewGlossarySeeder(blossomsShow("zh"), nil, toTWOpenCC{}, nil)
	seeder.SetWikiCastSource(wiki)

	_, pairs, err := seeder.FetchCredits(context.Background(), "tv", 286506)
	require.NoError(t, err)

	assert.Equal(t, map[string]string{"Shen Xihe": "沈汐和", "Xiao Huayong": "蕭華雍"}, characterPairs(pairs),
		"何与 on the page is 何與 on TMDb once both are in Taiwan script")
	for _, p := range pairs {
		assert.NotEqual(t, "actor", p.Kind, "still a drama: no actor names")
	}
	assert.Equal(t, []string{"百花殺 (2026年電視劇)", "百花殺 (電視劇)", "百花殺"}, wiki.asked,
		"the remake-style title first, then the plain one; stop at the first page that matches")
}

func TestFetchCredits_APageThatIsNotThisShowIsNotUsed(t *testing.T) {
	// Only one of the two actors appears: a page about something else.
	wiki := &fakeWiki{pages: map[string]string{"百花殺": "{|class=\"wikitable\"\n!演員\n!角色\n|-\n|孟子義\n|別人\n|}"}}
	seeder := NewGlossarySeeder(blossomsShow("zh"), nil, toTWOpenCC{}, nil)
	seeder.SetWikiCastSource(wiki)

	_, pairs, err := seeder.FetchCredits(context.Background(), "tv", 286506)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"Shen Xihe": "Shen Xihe", "Xiao Huayong": "Xiao Huayong"}, characterPairs(pairs),
		"left as TMDb has them (the seed filter drops untranslated names later)")
}

func TestFetchCredits_WikiIsOnlyAskedWhenItCanHelp(t *testing.T) {
	t.Run("not a Chinese title", func(t *testing.T) {
		wiki := &fakeWiki{}
		seeder := NewGlossarySeeder(blossomsShow("en"), nil, toTWOpenCC{}, nil)
		seeder.SetWikiCastSource(wiki)
		_, _, err := seeder.FetchCredits(context.Background(), "tv", 1)
		require.NoError(t, err)
		assert.Empty(t, wiki.asked)
	})
	t.Run("TMDb already has the Chinese names", func(t *testing.T) {
		client := blossomsShow("zh")
		client.tv["zh-TW"].Cast[0].Roles[0].Character = "沈汐和"
		client.tv["zh-TW"].Cast[1].Roles[0].Character = "蕭華雍"
		wiki := &fakeWiki{}
		seeder := NewGlossarySeeder(client, nil, toTWOpenCC{}, nil)
		seeder.SetWikiCastSource(wiki)
		_, _, err := seeder.FetchCredits(context.Background(), "tv", 1)
		require.NoError(t, err)
		assert.Empty(t, wiki.asked)
	})
	t.Run("no wiki source wired", func(t *testing.T) {
		_, pairs, err := NewGlossarySeeder(blossomsShow("zh"), nil, toTWOpenCC{}, nil).FetchCredits(context.Background(), "tv", 1)
		require.NoError(t, err)
		assert.Equal(t, "Shen Xihe", characterPairs(pairs)["Shen Xihe"])
	})
}

func TestFetchCredits_WikiOutageLeavesTMDbNames(t *testing.T) {
	wiki := &fakeWiki{err: errors.New("dial tcp: i/o timeout")}
	seeder := NewGlossarySeeder(blossomsShow("zh"), nil, toTWOpenCC{}, nil)
	seeder.SetWikiCastSource(wiki)

	_, pairs, err := seeder.FetchCredits(context.Background(), "tv", 1)
	require.NoError(t, err, "the seed itself never fails on wikipedia")
	assert.Equal(t, "Shen Xihe", characterPairs(pairs)["Shen Xihe"])
	assert.Len(t, wiki.asked, 1, "an outage stops at the first try")
}
