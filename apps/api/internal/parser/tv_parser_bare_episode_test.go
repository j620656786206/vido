package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// disc-2026-10-bare-episode-filename-uses-folder-title: "Season 1/S01E03.mkv"
// carries only the season and episode — the show's name is the folder's. The
// name used to go to AI, which cannot see a title either and answered with an
// empty one about half the time (the NAS's 力量之戒 S1: 4 of 8 imported).
func TestTVParser_BareEpisodeFilename(t *testing.T) {
	cases := []struct {
		filename        string
		season, episode int
		episodeEnd      int
		quality         string
	}{
		{"S01E03.mkv", 1, 3, 0, ""},
		{"s02e10.mp4", 2, 10, 0, ""},
		{"S01E03.Pilot.1080p.mkv", 1, 3, 0, "1080p"},
		{"S01E01-E02.mkv", 1, 1, 2, ""},
		{"[Grp] S01E05.mkv", 1, 5, 0, ""},
		{"S1E7 - The Return.mkv", 1, 7, 0, ""},
	}
	p := NewTVParser()
	for _, tc := range cases {
		t.Run(tc.filename, func(t *testing.T) {
			require.True(t, p.CanParse(tc.filename), "CanParse must agree with Parse")
			got := p.Parse(tc.filename)
			require.Equal(t, ParseStatusSuccess, got.Status)
			assert.Equal(t, MediaTypeTVShow, got.MediaType)
			assert.Empty(t, got.Title, "no title in the name — the scanner takes the folder's")
			assert.Empty(t, got.CleanedTitle)
			assert.Equal(t, tc.season, got.Season)
			assert.Equal(t, tc.episode, got.Episode)
			assert.Equal(t, tc.episodeEnd, got.EpisodeEnd)
			assert.Equal(t, tc.quality, got.Quality)
		})
	}
}

func TestTVParser_TitledNamesStillTakeTheStandardFormat(t *testing.T) {
	got := NewTVParser().Parse("The.Rings.of.Power.S02E04.Eldest.2160p.mkv")
	require.Equal(t, ParseStatusSuccess, got.Status)
	assert.Equal(t, "The Rings of Power", got.Title)
	assert.Equal(t, 2, got.Season)
	assert.Equal(t, 4, got.Episode)
}

func TestTVParser_NotABareEpisode(t *testing.T) {
	p := NewTVParser()
	for _, name := range []string{"S01.mkv", "E03.mkv", "SE01E03.mkv", "Season 1.mkv"} {
		assert.NotEqual(t, ParseStatusSuccess, p.Parse(name).Status, name)
	}
}
