package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"
)

// ---- fakes ---------------------------------------------------------------

type fakeCreditsClient struct {
	movie map[string]*tmdb.MovieCredits       // keyed by language
	tv    map[string]*tmdb.TVAggregateCredits // keyed by language
	err   map[string]error                    // keyed by language
	calls []string
}

func (f *fakeCreditsClient) GetMovieCreditsWithLanguage(_ context.Context, id int, lang string) (*tmdb.MovieCredits, error) {
	f.calls = append(f.calls, fmt.Sprintf("movie/%d?%s", id, lang))
	if err := f.err[lang]; err != nil {
		return nil, err
	}
	return f.movie[lang], nil
}

func (f *fakeCreditsClient) GetTVAggregateCreditsWithLanguage(_ context.Context, id int, lang string) (*tmdb.TVAggregateCredits, error) {
	f.calls = append(f.calls, fmt.Sprintf("tv/%d?%s", id, lang))
	if err := f.err[lang]; err != nil {
		return nil, err
	}
	return f.tv[lang], nil
}

// fakeSeedOpenCC stands in for the official CLI: a fixed simplified→Taiwan map,
// line-preserving like the real thing.
type fakeSeedOpenCC struct {
	available bool
	err       error
	calls     int
}

var fakeSeedOpenCCReplacer = strings.NewReplacer("华", "華", "怀", "懷", "软件", "軟體", "别", "別")

func (f *fakeSeedOpenCC) IsAvailable() bool { return f.available }
func (f *fakeSeedOpenCC) ConvertS2TWP(content []byte) ([]byte, error) {
	f.calls++
	if f.err != nil {
		return content, f.err
	}
	return []byte(fakeSeedOpenCCReplacer.Replace(string(content))), nil
}

type fakeSeedInserter struct {
	terms []*models.GlossaryTerm
	err   error
}

func (f *fakeSeedInserter) InsertIfAbsent(_ context.Context, term *models.GlossaryTerm) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	for _, t := range f.terms {
		if strings.EqualFold(t.TermSrc, term.TermSrc) && t.Scope == term.Scope {
			return false, nil
		}
	}
	f.terms = append(f.terms, term)
	return true, nil
}

// ---- AC #3 / #2: the noise filter table ---------------------------------

func TestNormalizeSeedPair_Table(t *testing.T) {
	tests := []struct {
		name    string
		pair    CastPair
		wantOK  bool
		wantSrc string
		wantZh  string
	}{
		// (a) zh-TW character name used as-is
		{"zh-TW name direct", CastPair{"character", "Walter White", "華特·懷特"}, true, "Walter White", "華特·懷特"},
		// (b) simplified survives the filter untouched — conversion is a
		// separate batched step (see TestSeedFromCredits_ConvertsSimplifiedInOneOpenCCCall)
		{"simplified passes filter unconverted", CastPair{"character", "Walter White", "华特·怀特"}, true, "Walter White", "华特·怀特"},
		// (c) English on the zh side → do not seed, never fabricate
		{"untranslated english", CastPair{"character", "Walter White", "Walter White"}, false, "", ""},
		{"empty zh", CastPair{"character", "Walter White", ""}, false, "", ""},
		{"empty src", CastPair{"character", "", "華特"}, false, "", ""},
		{"whitespace only", CastPair{"character", "   ", "   "}, false, "", ""},
		// (d) generic roles
		{"self", CastPair{"character", "Self", "本人"}, false, "", ""},
		{"himself", CastPair{"character", "Himself", "他自己"}, false, "", ""},
		{"self - host", CastPair{"character", "Self - Host", "主持人"}, false, "", ""},
		{"narrator", CastPair{"character", "Narrator", "旁白"}, false, "", ""},
		{"voice only", CastPair{"character", "(voice)", "(配音)"}, false, "", ""},
		{"additional voices", CastPair{"character", "Additional Voices", "其他配音"}, false, "", ""},
		{"man at bar is generic man? no — keeps named-looking roles", CastPair{"character", "Man at Bar", "酒吧男子"}, true, "Man at Bar", "酒吧男子"},
		{"plain man", CastPair{"character", "Man", "男人"}, false, "", ""},
		{"police officer", CastPair{"character", "Police Officer", "警察"}, false, "", ""},
		{"numbered extra", CastPair{"character", "Guard #2", "警衛2"}, false, "", ""},
		{"numbered extra no hash", CastPair{"character", "Thug 3", "惡棍3"}, false, "", ""},
		{"numbered extra no space", CastPair{"character", "Thug3", "惡棍3"}, false, "", ""},
		{"real name with digits kept", CastPair{"character", "Agent 47", "47號特工"}, true, "Agent 47", "47號特工"},
		{"real name with digits kept 2", CastPair{"character", "Android 18", "人造人18號"}, true, "Android 18", "人造人18號"},
		// parenthetical qualifiers are stripped, not fatal
		{"voice qualifier stripped, full-width too", CastPair{"character", "Bob (voice)", "鮑伯（配音）"}, true, "Bob", "鮑伯"},
		{"uncredited stripped both sides", CastPair{"character", "Bob (uncredited)", "鮑伯 (uncredited)"}, true, "Bob", "鮑伯"},
		{"nested parentheses fully stripped", CastPair{"character", "Bob (voice (uncredited))", "鮑伯"}, true, "Bob", "鮑伯"},
		{"unbalanced parenthesis rejected", CastPair{"character", "Bob (as Robert", "鮑伯"}, false, "", ""},
		{"stray close paren rejected", CastPair{"character", "Bob)", "鮑伯"}, false, "", ""},
		// length
		{"too long", CastPair{"character", strings.Repeat("A", 41), "長"}, false, "", ""},
		{"exactly 40 ok", CastPair{"character", strings.Repeat("A", 40), "長"}, true, strings.Repeat("A", 40), "長"},
		// file shape
		{"file extension", CastPair{"character", "Episode.mkv", "集"}, false, "", ""},
		{"path separator", CastPair{"character", "Movies/Bob", "鮑伯"}, false, "", ""},
		{"season episode tag", CastPair{"character", "Bob S01E02", "鮑伯"}, false, "", ""},
		{"release token", CastPair{"character", "Bob 1080p", "鮑伯"}, false, "", ""},
		{"zh side file shape", CastPair{"character", "Bob", "鮑伯.srt"}, false, "", ""},
		// script rules
		{"src already chinese (chinese-origin show)", CastPair{"character", "李四", "李四"}, false, "", ""},
		{"src has no latin letter", CastPair{"character", "123", "一二三"}, false, "", ""},
		{"kana zh side is not han", CastPair{"actor", "Ken Watanabe", "ワタナベ"}, false, "", ""},
		// multi-role
		{"multi role paired first", CastPair{"character", "Walter White / Heisenberg", "華特·懷特 / 海森堡"}, true, "Walter White", "華特·懷特"},
		{"multi role count mismatch", CastPair{"character", "Walter White / Heisenberg", "華特·懷特"}, false, "", ""},
		{"multi role unspaced slash", CastPair{"character", "Bruce Wayne/Batman", "布魯斯·韋恩/蝙蝠俠"}, true, "Bruce Wayne", "布魯斯·韋恩"},
		{"multi role mixed spacing", CastPair{"character", "Bruce Wayne / Batman", "布魯斯·韋恩/蝙蝠俠"}, true, "Bruce Wayne", "布魯斯·韋恩"},
		// actor names
		{"actor translated", CastPair{"actor", "Bryan Cranston", "布萊恩·克蘭斯頓"}, true, "Bryan Cranston", "布萊恩·克蘭斯頓"},
		{"actor untranslated", CastPair{"actor", "Bryan Cranston", "Bryan Cranston"}, false, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, zh, ok := normalizeSeedPair(tt.pair)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantSrc, src)
				assert.Equal(t, tt.wantZh, zh)
			}
		})
	}
}

// Every entry of the generic-role table must actually be rejected — the test
// enumerates the table so adding an entry that the regex/strip step mangles
// (e.g. one with parentheses) is caught.
func TestNormalizeSeedPair_GenericTableIsExhaustive(t *testing.T) {
	for role := range glossarySeedGenericRoles {
		_, _, ok := normalizeSeedPair(CastPair{Kind: "character", Src: strings.ToUpper(role[:1]) + role[1:], Zh: "某某"})
		assert.Falsef(t, ok, "generic role %q must be filtered", role)
	}
}

// ---- pairing --------------------------------------------------------------

func strp(s string) *string { return &s }

func TestFetchCredits_MoviePairsByCreditIDAndCapsAtCastLimit(t *testing.T) {
	en := &tmdb.MovieCredits{ID: 550}
	zh := &tmdb.MovieCredits{ID: 550}
	// 12 cast rows; only the first MetadataCastLimit (10) by order are paired.
	for i := 0; i < 12; i++ {
		cid := fmt.Sprintf("c%d", i)
		en.Cast = append(en.Cast, tmdb.CreditCast{ID: i, CreditID: cid, Name: fmt.Sprintf("Actor %d", i), Character: fmt.Sprintf("Role %d", i), Order: 11 - i})
		zh.Cast = append(zh.Cast, tmdb.CreditCast{ID: i, CreditID: cid, Name: fmt.Sprintf("演員%d", i), Character: fmt.Sprintf("角色%d", i), Order: 11 - i, ProfilePath: strp("/p.jpg")})
	}
	en.Crew = []tmdb.CreditCrew{{ID: 1, Name: "David Fincher", Job: "Director", Department: "Directing"}, {ID: 2, Name: "Someone", Job: "Gaffer"}}
	zh.Crew = en.Crew

	client := &fakeCreditsClient{movie: map[string]*tmdb.MovieCredits{"en-US": en, "zh-TW": zh}}
	seeder := NewGlossarySeeder(client, &fakeSeedInserter{}, nil, nil)

	credits, pairs, err := seeder.FetchCredits(context.Background(), "movie", 550)
	require.NoError(t, err)
	assert.Equal(t, []string{"movie/550?zh-TW", "movie/550?en-US"}, client.calls)

	// Stored credits are the zh-TW response, cast sorted by TMDb order, key crew only.
	require.NotNil(t, credits)
	require.Len(t, credits.Cast, 12)
	assert.Equal(t, "演員11", credits.Cast[0].Name, "sorted by order ascending")
	assert.Equal(t, "/p.jpg", credits.Cast[0].ProfilePath)
	require.Len(t, credits.Crew, 1)
	assert.Equal(t, "Director", credits.Crew[0].Job)

	// Pairs: 10 rows × (actor + character), lowest order first.
	require.Len(t, pairs, 20)
	assert.Equal(t, CastPair{Kind: "actor", Src: "Actor 11", Zh: "演員11"}, pairs[0])
	assert.Equal(t, CastPair{Kind: "character", Src: "Role 11", Zh: "角色11"}, pairs[1])
	for _, p := range pairs {
		assert.NotContains(t, []string{"Actor 0", "Actor 1", "Role 0", "Role 1"}, p.Src, "rows past the cast limit are not seeded")
	}
}

func TestFetchCredits_TVRanksByEpisodeCountAndPairsRoles(t *testing.T) {
	en := &tmdb.TVAggregateCredits{ID: 1396, Cast: []tmdb.AggregateCast{
		{ID: 1, Name: "Guest Star", Order: 0, TotalEpisodeCount: 2, Roles: []tmdb.AggregateRole{{CreditID: "g", Character: "Cameo Guy", EpisodeCount: 2}}},
		{ID: 2, Name: "Bryan Cranston", Order: 5, TotalEpisodeCount: 62, Roles: []tmdb.AggregateRole{{CreditID: "w", Character: "Walter White", EpisodeCount: 62}, {CreditID: "h", Character: "Heisenberg", EpisodeCount: 3}}},
	}}
	zh := &tmdb.TVAggregateCredits{ID: 1396, Cast: []tmdb.AggregateCast{
		{ID: 2, Name: "布萊恩·克蘭斯頓", Roles: []tmdb.AggregateRole{{CreditID: "w", Character: "華特·懷特"}, {CreditID: "h", Character: "海森堡"}}},
		{ID: 1, Name: "Guest Star", Roles: []tmdb.AggregateRole{{CreditID: "g", Character: "客串"}}},
	}}
	client := &fakeCreditsClient{tv: map[string]*tmdb.TVAggregateCredits{"en-US": en, "zh-TW": zh}}
	seeder := NewGlossarySeeder(client, &fakeSeedInserter{}, nil, nil)

	credits, pairs, err := seeder.FetchCredits(context.Background(), "tv", 1396)
	require.NoError(t, err)
	assert.Equal(t, []string{"tv/1396?zh-TW", "tv/1396?en-US"}, client.calls)

	require.Len(t, credits.Cast, 2)
	assert.Equal(t, "布萊恩·克蘭斯頓", credits.Cast[0].Name, "most episodes first regardless of TMDb order")
	assert.Equal(t, "華特·懷特 / 海森堡", credits.Cast[0].Character)

	require.Len(t, pairs, 5)
	assert.Equal(t, CastPair{Kind: "actor", Src: "Bryan Cranston", Zh: "布萊恩·克蘭斯頓"}, pairs[0])
	assert.Equal(t, CastPair{Kind: "character", Src: "Walter White", Zh: "華特·懷特"}, pairs[1])
	assert.Equal(t, CastPair{Kind: "character", Src: "Heisenberg", Zh: "海森堡"}, pairs[2])
}

func TestFetchCredits_ZhFailureReturnsError_EnFailureStillReturnsCredits(t *testing.T) {
	zhErr := errors.New("boom zh")
	client := &fakeCreditsClient{err: map[string]error{"zh-TW": zhErr}}
	seeder := NewGlossarySeeder(client, &fakeSeedInserter{}, nil, nil)
	credits, pairs, err := seeder.FetchCredits(context.Background(), "movie", 1)
	assert.ErrorIs(t, err, zhErr)
	assert.Nil(t, credits)
	assert.Nil(t, pairs)

	enErr := errors.New("boom en")
	client = &fakeCreditsClient{
		movie: map[string]*tmdb.MovieCredits{"zh-TW": {Cast: []tmdb.CreditCast{{ID: 1, Name: "某某"}}}},
		err:   map[string]error{"en-US": enErr},
	}
	seeder = NewGlossarySeeder(client, &fakeSeedInserter{}, nil, nil)
	credits, pairs, err = seeder.FetchCredits(context.Background(), "movie", 1)
	assert.ErrorIs(t, err, enErr)
	require.NotNil(t, credits, "zh-TW credits are still worth storing when only the en pass failed")
	assert.Len(t, credits.Cast, 1)
	assert.Nil(t, pairs)
}

func TestFetchCredits_NoClientOrBadIDIsANoop(t *testing.T) {
	seeder := NewGlossarySeeder(nil, &fakeSeedInserter{}, nil, nil)
	credits, pairs, err := seeder.FetchCredits(context.Background(), "movie", 1)
	assert.NoError(t, err)
	assert.Nil(t, credits)
	assert.Nil(t, pairs)

	client := &fakeCreditsClient{}
	seeder = NewGlossarySeeder(client, &fakeSeedInserter{}, nil, nil)
	_, _, err = seeder.FetchCredits(context.Background(), "movie", 0)
	assert.NoError(t, err)
	assert.Empty(t, client.calls)
}

// ---- seeding --------------------------------------------------------------

func TestSeedFromCredits_CountsAndDedupes(t *testing.T) {
	repo := &fakeSeedInserter{}
	cc := &fakeSeedOpenCC{available: true}
	seeder := NewGlossarySeeder(nil, repo, cc, nil)
	res := seeder.SeedFromCredits(context.Background(), "tmdb:tv:1396", "series-1", []CastPair{
		{"actor", "Bryan Cranston", "布萊恩·克蘭斯頓"},
		{"character", "Walter White", "华特·怀特"},        // simplified → converted
		{"character", "walter white", "別的"},           // same term, different case → dedupe in-pass
		{"character", "Self", "本人"},                   // generic
		{"character", "Skyler White", "Skyler White"}, // untranslated
	})
	assert.Equal(t, SeedResult{Seeded: 2, Skipped: 3}, res)
	require.Len(t, repo.terms, 2)
	assert.Equal(t, "series-1", repo.terms[0].MediaID)
	assert.Equal(t, "tmdb:tv:1396", repo.terms[0].Scope)
	assert.Equal(t, models.GlossarySourceMetadata, repo.terms[0].Source)
	assert.False(t, repo.terms[0].Confirmed)
	assert.Equal(t, "華特·懷特", repo.terms[1].TermZh)
	assert.Equal(t, 1, cc.calls, "one OpenCC subprocess per title, not per name")
}

func TestSeedFromCredits_ConvertsSimplifiedInOneOpenCCCall(t *testing.T) {
	repo := &fakeSeedInserter{}
	cc := &fakeSeedOpenCC{available: true}
	seeder := NewGlossarySeeder(nil, repo, cc, nil)
	res := seeder.SeedFromCredits(context.Background(), "tmdb:movie:1", "m1", []CastPair{
		{"character", "Walter White", "华特·怀特"},
		{"character", "Software", "软件"},
		{"actor", "Bob", "鮑伯"},
	})
	assert.Equal(t, SeedResult{Seeded: 3}, res)
	assert.Equal(t, 1, cc.calls)
	assert.Equal(t, "華特·懷特", repo.terms[0].TermZh)
	assert.Equal(t, "軟體", repo.terms[1].TermZh, "s2twp phrase table, not just character map")
	assert.Equal(t, "鮑伯", repo.terms[2].TermZh, "traditional passes through")
}

func TestSeedFromCredits_OpenCCUnavailableOrFailingStoresAsReturned(t *testing.T) {
	for _, cc := range []OpenCCConverter{nil, &fakeSeedOpenCC{available: false}, &fakeSeedOpenCC{available: true, err: errors.New("cli gone")}} {
		repo := &fakeSeedInserter{}
		seeder := NewGlossarySeeder(nil, repo, cc, nil)
		res := seeder.SeedFromCredits(context.Background(), "tmdb:movie:1", "m1", []CastPair{{"character", "Walter White", "华特·怀特"}})
		assert.Equal(t, SeedResult{Seeded: 1}, res)
		require.Len(t, repo.terms, 1)
		assert.Equal(t, "华特·怀特", repo.terms[0].TermZh, "a Simplified seed in the review list beats a dropped one")
	}
}

func TestSeedFromCredits_RepoErrorIsCountedNotFatal(t *testing.T) {
	repo := &fakeSeedInserter{err: errors.New("disk on fire")}
	seeder := NewGlossarySeeder(nil, repo, nil, nil)
	res := seeder.SeedFromCredits(context.Background(), "tmdb:movie:1", "m1", []CastPair{{"actor", "Bob", "鮑伯"}})
	assert.Equal(t, SeedResult{Failed: 1}, res)
}

func TestSeedFromCredits_EmptyInputsAreNoops(t *testing.T) {
	repo := &fakeSeedInserter{}
	seeder := NewGlossarySeeder(nil, repo, nil, nil)
	assert.Equal(t, SeedResult{}, seeder.SeedFromCredits(context.Background(), "", "m1", []CastPair{{"actor", "Bob", "鮑伯"}}))
	assert.Equal(t, SeedResult{}, seeder.SeedFromCredits(context.Background(), "tmdb:movie:1", "m1", nil))
	assert.Empty(t, repo.terms)
}

// (e) Idempotent re-seed and never-overwrite, against the REAL repository so
// the NOCASE unique index and INSERT … DO NOTHING are what is under test,
// not the fake.
func TestSeedFromCredits_RescanIsIdempotentAndNeverOverwritesUserTerms(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewGlossaryRepository(db)
	ctx := context.Background()
	scope := models.GlossaryScopeTV(1396)

	// The user already fixed one term by hand and confirmed another.
	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "s1", Scope: scope, TermSrc: "Walter White", TermZh: "老白", Source: models.GlossarySourceManual}))
	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "s1", Scope: scope, TermSrc: "Jesse Pinkman", TermZh: "傑西", Source: models.GlossarySourceSubtitle, Confirmed: true}))

	seeder := NewGlossarySeeder(nil, repo, &fakeSeedOpenCC{available: true}, nil)
	pairs := []CastPair{
		{"character", "walter white", "華特·懷特"}, // case differs — still the same term
		{"character", "Jesse Pinkman", "傑西·平克曼"},
		{"character", "Skyler White", "史凱勒·懷特"},
	}

	first := seeder.SeedFromCredits(ctx, scope, "s1", pairs)
	assert.Equal(t, SeedResult{Seeded: 1, Skipped: 2}, first)

	second := seeder.SeedFromCredits(ctx, scope, "s1", pairs)
	assert.Equal(t, SeedResult{Seeded: 0, Skipped: 3}, second, "re-scan seeds nothing new")

	terms, err := repo.ListByScope(ctx, scope)
	require.NoError(t, err)
	byTerm := map[string]models.GlossaryTerm{}
	for _, tm := range terms {
		byTerm[strings.ToLower(tm.TermSrc)] = tm
	}
	require.Len(t, byTerm, 3)
	assert.Equal(t, "老白", byTerm["walter white"].TermZh, "manual term untouched")
	assert.Equal(t, models.GlossarySourceManual, byTerm["walter white"].Source)
	assert.Equal(t, "傑西", byTerm["jesse pinkman"].TermZh, "confirmed term untouched")
	assert.True(t, byTerm["jesse pinkman"].Confirmed)
	assert.Equal(t, "史凱勒·懷特", byTerm["skyler white"].TermZh)
	assert.Equal(t, models.GlossarySourceMetadata, byTerm["skyler white"].Source)
	assert.False(t, byTerm["skyler white"].Confirmed)
}
