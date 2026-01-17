package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// movieTestCase holds expected values for movie parsing tests.
type movieTestCase struct {
	name     string
	filename string
	title    string
	year     int
	quality  string
	source   string
	codec    string
	group    string
}

// tvTestCase holds expected values for TV parsing tests.
type tvTestCase struct {
	name       string
	filename   string
	title      string
	season     int
	episode    int
	episodeEnd int
	quality    string
	source     string
}

// TestComprehensiveMovieParsing tests movie parsing with 25+ real-world examples.
func TestComprehensiveMovieParsing(t *testing.T) {
	parser := NewMovieParser()

	tests := []movieTestCase{
		// Standard format with various quality levels
		{"Matrix 1080p BluRay", "The.Matrix.1999.1080p.BluRay.x264-SPARKS.mkv", "The Matrix", 1999, "1080p", "BluRay", "x264", "SPARKS"},
		{"Inception 2160p UHD", "Inception.2010.2160p.UHD.BluRay.HEVC-YTS.mkv", "Inception", 2010, "2160p", "BluRay", "x265", "YTS"},
		{"Parasite 720p WEB-DL", "Parasite.2019.720p.WEB-DL.AAC.mkv", "Parasite", 2019, "720p", "WEB-DL", "", ""},

		// Year in parentheses
		{"Dark Knight parentheses", "The Dark Knight (2008) 1080p BluRay.mkv", "The Dark Knight", 2008, "1080p", "BluRay", "", ""},
		{"Joker parentheses", "Joker (2019) 2160p WEB-DL.mkv", "Joker", 2019, "2160p", "WEB-DL", "", ""},

		// Titles with years
		{"2001 Space Odyssey", "2001.A.Space.Odyssey.1968.1080p.BluRay.mkv", "2001 A Space Odyssey", 1968, "1080p", "BluRay", "", ""},
		{"Blade Runner 2049", "Blade.Runner.2049.2017.2160p.UHD.BluRay.x265.mkv", "Blade Runner 2049", 2017, "2160p", "BluRay", "x265", ""},
		{"1917 movie", "1917.2019.1080p.BluRay.x264-SPARKS.mkv", "1917", 2019, "1080p", "BluRay", "x264", "SPARKS"},

		// Various separators
		{"Underscore separated", "Avengers_Endgame_2019_1080p_BluRay.mkv", "Avengers Endgame", 2019, "1080p", "BluRay", "", ""},
		{"Space separated", "Avengers Endgame 2019 1080p BluRay.mkv", "Avengers Endgame", 2019, "1080p", "BluRay", "", ""},
		{"Mixed separators", "Spider-Man.Far.From.Home.2019.1080p.BluRay.mkv", "Spider Man Far From Home", 2019, "1080p", "BluRay", "", ""},

		// Different sources
		{"HDTV source", "Movie.2020.720p.HDTV.x264.mkv", "Movie", 2020, "720p", "HDTV", "x264", ""},
		{"WEBRip source", "Movie.2020.1080p.WEBRip.x264.mkv", "Movie", 2020, "1080p", "WEBRip", "x264", ""},
		{"DVDRip source", "Classic.Movie.1990.480p.DVDRip.mkv", "Classic Movie", 1990, "480p", "DVDRip", "", ""},
		{"BDRip source", "Movie.2020.1080p.BDRip.x264.mkv", "Movie", 2020, "1080p", "BluRay", "x264", ""},

		// Different codecs
		{"HEVC codec", "Movie.2020.2160p.BluRay.HEVC.mkv", "Movie", 2020, "2160p", "BluRay", "x265", ""},
		{"AV1 codec", "Movie.2020.1080p.WEB-DL.AV1.mkv", "Movie", 2020, "1080p", "WEB-DL", "AV1", ""},
		{"H264 codec", "Movie.2020.1080p.BluRay.H264.mkv", "Movie", 2020, "1080p", "BluRay", "x264", ""},

		// Quality variations
		{"4K quality", "Movie.2020.4K.UHD.BluRay.mkv", "Movie", 2020, "2160p", "BluRay", "", ""},
		{"UHD quality", "Movie.2020.UHD.BluRay.HDR.mkv", "Movie", 2020, "2160p", "BluRay", "", ""},
		{"SD quality", "Movie.2020.SD.DVDRip.mkv", "Movie", 2020, "480p", "DVDRip", "", ""},

		// Simple format
		{"Year only", "Pulp.Fiction.1994.mkv", "Pulp Fiction", 1994, "", "", "", ""},
		{"Year and quality only", "Goodfellas.1990.1080p.mkv", "Goodfellas", 1990, "1080p", "", "", ""},

		// Long titles
		{"Long title", "The.Lord.of.the.Rings.The.Fellowship.of.the.Ring.2001.Extended.1080p.BluRay.mkv", "The Lord of the Rings The Fellowship of the Ring", 2001, "1080p", "BluRay", "", ""},
		{"Long title with year", "Pirates.of.the.Caribbean.The.Curse.of.the.Black.Pearl.2003.1080p.BluRay.mkv", "Pirates of the Caribbean The Curse of the Black Pearl", 2003, "1080p", "BluRay", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.filename)

			require.NotNil(t, result)
			assert.Equal(t, ParseStatusSuccess, result.Status, "should parse successfully")
			assert.Equal(t, MediaTypeMovie, result.MediaType, "should be movie type")
			assert.Equal(t, tt.title, result.Title, "title mismatch")
			assert.Equal(t, tt.year, result.Year, "year mismatch")

			if tt.quality != "" {
				assert.Equal(t, tt.quality, result.Quality, "quality mismatch")
			}
			if tt.source != "" {
				assert.Equal(t, tt.source, result.Source, "source mismatch")
			}
			if tt.codec != "" {
				assert.Equal(t, tt.codec, result.VideoCodec, "codec mismatch")
			}
			if tt.group != "" {
				assert.Equal(t, tt.group, result.ReleaseGroup, "group mismatch")
			}
		})
	}
}

// TestComprehensiveTVParsing tests TV show parsing with 25+ real-world examples.
func TestComprehensiveTVParsing(t *testing.T) {
	parser := NewTVParser()

	tests := []tvTestCase{
		// Standard S01E01 format
		{"Breaking Bad standard", "Breaking.Bad.S01E05.720p.BluRay.x264-DEMAND.mkv", "Breaking Bad", 1, 5, 0, "720p", "BluRay"},
		{"Game of Thrones", "Game.of.Thrones.S08E06.1080p.WEB-DL.x265.mkv", "Game of Thrones", 8, 6, 0, "1080p", "WEB-DL"},
		{"Stranger Things", "Stranger.Things.S04E09.1080p.WEB-DL.HEVC-GROUP.mkv", "Stranger Things", 4, 9, 0, "1080p", "WEB-DL"},

		// Lowercase
		{"Lowercase sXeX", "the.office.s02e10.720p.webrip.mkv", "the office", 2, 10, 0, "720p", "WEBRip"},

		// Episode ranges
		{"Episode range E01-E02", "Friends.S01E01-E02.DVDRip.mkv", "Friends", 1, 1, 2, "", "DVDRip"},
		{"Episode range E15-16", "House.S03E15-16.720p.BluRay.mkv", "House", 3, 15, 16, "720p", "BluRay"},

		// Double digits
		{"Double digit season", "Grey.Anatomy.S18E15.1080p.WEB-DL.mkv", "Grey Anatomy", 18, 15, 0, "1080p", "WEB-DL"},

		// 1x05 format
		{"1x05 format", "House.1x13.720p.mkv", "House", 1, 13, 0, "720p", ""},
		{"2x10 format", "Scrubs.2x10.DVDRip.mkv", "Scrubs", 2, 10, 0, "", "DVDRip"},

		// Space separated
		{"Space separated", "The Mandalorian S02E01 720p WEBRip.mkv", "The Mandalorian", 2, 1, 0, "720p", "WEBRip"},

		// Underscore separated
		{"Underscore separated", "Dark_S03E08_1080p_WEB-DL.mkv", "Dark", 3, 8, 0, "1080p", "WEB-DL"},

		// Daily shows
		{"Daily show", "The.Daily.Show.2024.01.15.720p.WEB-DL.mkv", "The Daily Show", 0, 0, 0, "720p", "WEB-DL"},
		{"Late night", "The.Tonight.Show.2023.12.25.1080p.WEB-DL.mkv", "The Tonight Show", 0, 0, 0, "1080p", "WEB-DL"},

		// Anime with Episode prefix
		{"Anime Episode prefix", "Naruto.Episode.01.720p.mkv", "Naruto", 0, 1, 0, "720p", ""},
		{"Anime Ep prefix", "One.Piece.Ep.100.1080p.WEB-DL.mkv", "One Piece", 0, 100, 0, "1080p", "WEB-DL"},

		// Anime with dash
		{"Anime dash format", "Attack.on.Titan.-.01.720p.mkv", "Attack on Titan", 0, 1, 0, "720p", ""},

		// Various sources and codecs
		{"HDTV source", "Show.S01E01.720p.HDTV.x264.mkv", "Show", 1, 1, 0, "720p", "HDTV"},
		{"WEBRip source", "Show.S01E01.1080p.WEBRip.HEVC.mkv", "Show", 1, 1, 0, "1080p", "WEBRip"},

		// Long titles
		{"Long TV title", "How.I.Met.Your.Mother.S09E24.1080p.BluRay.mkv", "How I Met Your Mother", 9, 24, 0, "1080p", "BluRay"},
		{"The title", "The.Walking.Dead.S11E24.1080p.WEB-DL.mkv", "The Walking Dead", 11, 24, 0, "1080p", "WEB-DL"},

		// Numbered show titles
		{"24 TV show", "24.S08E24.1080p.BluRay.mkv", "24", 8, 24, 0, "1080p", "BluRay"},
		{"9-1-1 TV show", "9-1-1.S05E18.1080p.WEB-DL.mkv", "9 1 1", 5, 18, 0, "1080p", "WEB-DL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.filename)

			require.NotNil(t, result)
			assert.Equal(t, ParseStatusSuccess, result.Status, "should parse successfully")
			assert.Equal(t, MediaTypeTVShow, result.MediaType, "should be TV type")
			assert.Equal(t, tt.title, result.Title, "title mismatch")
			assert.Equal(t, tt.season, result.Season, "season mismatch")
			assert.Equal(t, tt.episode, result.Episode, "episode mismatch")

			if tt.episodeEnd > 0 {
				assert.Equal(t, tt.episodeEnd, result.EpisodeEnd, "episode end mismatch")
			}
			if tt.quality != "" {
				assert.Equal(t, tt.quality, result.Quality, "quality mismatch")
			}
			if tt.source != "" {
				assert.Equal(t, tt.source, result.Source, "source mismatch")
			}
		})
	}
}

// TestEdgeCases tests edge cases and unusual filenames.
func TestEdgeCases(t *testing.T) {
	movieParser := NewMovieParser()
	tvParser := NewTVParser()

	t.Run("unusual characters in title", func(t *testing.T) {
		result := movieParser.Parse("M.A.S.H.1970.1080p.BluRay.mkv")
		assert.Equal(t, ParseStatusSuccess, result.Status)
		assert.Equal(t, "M A S H", result.Title)
	})

	t.Run("very long filename", func(t *testing.T) {
		filename := "A.Very.Long.Movie.Title.That.Goes.On.And.On.And.On.2020.1080p.BluRay.x264-VERYLONGGROUPNAME.mkv"
		result := movieParser.Parse(filename)
		assert.Equal(t, ParseStatusSuccess, result.Status)
		assert.Contains(t, result.Title, "A Very Long Movie Title")
	})

	t.Run("multiple file extensions", func(t *testing.T) {
		result := movieParser.Parse("Movie.2020.1080p.BluRay.x264.sample.mkv")
		assert.Equal(t, ParseStatusSuccess, result.Status)
	})

	t.Run("TV show with special character in title", func(t *testing.T) {
		result := tvParser.Parse("Marvel's.Agents.of.S.H.I.E.L.D.S07E13.1080p.mkv")
		assert.Equal(t, ParseStatusSuccess, result.Status)
		assert.Contains(t, result.Title, "Marvel")
	})

	t.Run("year at start of title", func(t *testing.T) {
		result := movieParser.Parse("2012.2009.1080p.BluRay.mkv")
		assert.Equal(t, ParseStatusSuccess, result.Status)
		assert.Equal(t, "2012", result.Title)
		assert.Equal(t, 2009, result.Year)
	})

	t.Run("TV show with year in title", func(t *testing.T) {
		result := tvParser.Parse("The.100.S07E16.1080p.WEB-DL.mkv")
		assert.Equal(t, ParseStatusSuccess, result.Status)
		assert.Equal(t, "The 100", result.Title)
		assert.Equal(t, 7, result.Season)
	})
}

// TestNeedsAIParsing tests filenames that should return NeedsAI status.
func TestNeedsAIParsing(t *testing.T) {
	movieParser := NewMovieParser()
	tvParser := NewTVParser()

	tests := []struct {
		name     string
		filename string
	}{
		{"anime fansub bracket", "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080).mkv"},
		{"Chinese fansub", "【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P.mp4"},
		{"Japanese fansub", "[字幕组] 进击的巨人 第四季 [01-16] [简体中文字幕].mkv"},
		{"no year movie", "Some.Random.File.1080p.mkv"},
		{"no pattern", "random_video.mkv"},
		{"music video", "Artist - Song Name (Official Video).mp4"},
		{"complex bracket", "[Group1][Group2] Anime Name 01 [720p][AAC].mkv"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			movieResult := movieParser.Parse(tt.filename)
			tvResult := tvParser.Parse(tt.filename)

			// At least one parser should return NeedsAI
			needsAI := movieResult.Status == ParseStatusNeedsAI || tvResult.Status == ParseStatusNeedsAI
			assert.True(t, needsAI, "Expected NeedsAI status for: %s", tt.filename)
		})
	}
}
