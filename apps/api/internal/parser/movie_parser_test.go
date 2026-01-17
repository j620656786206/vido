package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMovieParser_StandardFormat(t *testing.T) {
	parser := NewMovieParser()

	tests := []struct {
		name     string
		filename string
		want     *ParseResult
	}{
		{
			name:     "standard dot-separated with quality and source",
			filename: "The.Matrix.1999.1080p.BluRay.x264-GROUP.mkv",
			want: &ParseResult{
				Status:       ParseStatusSuccess,
				MediaType:    MediaTypeMovie,
				Title:        "The Matrix",
				Year:         1999,
				Quality:      "1080p",
				Source:       "BluRay",
				VideoCodec:   "x264",
				ReleaseGroup: "GROUP",
			},
		},
		{
			name:     "4K UHD movie",
			filename: "Inception.2010.2160p.UHD.BluRay.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeMovie,
				Title:     "Inception",
				Year:      2010,
				Quality:   "2160p",
				Source:    "BluRay",
			},
		},
		{
			name:     "WEB-DL source",
			filename: "Parasite.2019.720p.WEB-DL.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeMovie,
				Title:     "Parasite",
				Year:      2019,
				Quality:   "720p",
				Source:    "WEB-DL",
			},
		},
		{
			name:     "year in parentheses",
			filename: "The Dark Knight (2008) 1080p.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeMovie,
				Title:     "The Dark Knight",
				Year:      2008,
				Quality:   "1080p",
			},
		},
		{
			name:     "title with year in name - 2001 A Space Odyssey",
			filename: "2001.A.Space.Odyssey.1968.1080p.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeMovie,
				Title:     "2001 A Space Odyssey",
				Year:      1968,
				Quality:   "1080p",
			},
		},
		{
			name:     "title with year in name - Blade Runner 2049",
			filename: "Blade.Runner.2049.2017.1080p.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeMovie,
				Title:     "Blade Runner 2049",
				Year:      2017,
				Quality:   "1080p",
			},
		},
		{
			name:     "underscore separated",
			filename: "Avengers_Endgame_2019_1080p.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeMovie,
				Title:     "Avengers Endgame",
				Year:      2019,
				Quality:   "1080p",
			},
		},
		{
			name:     "space separated",
			filename: "Avengers Endgame 2019 1080p.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeMovie,
				Title:     "Avengers Endgame",
				Year:      2019,
				Quality:   "1080p",
			},
		},
		{
			name:     "with HEVC codec",
			filename: "Dune.2021.2160p.WEB-DL.HEVC-SPARKS.mkv",
			want: &ParseResult{
				Status:       ParseStatusSuccess,
				MediaType:    MediaTypeMovie,
				Title:        "Dune",
				Year:         2021,
				Quality:      "2160p",
				Source:       "WEB-DL",
				VideoCodec:   "x265",
				ReleaseGroup: "SPARKS",
			},
		},
		{
			name:     "with x265 codec",
			filename: "Oppenheimer.2023.1080p.BluRay.x265-YTS.mkv",
			want: &ParseResult{
				Status:       ParseStatusSuccess,
				MediaType:    MediaTypeMovie,
				Title:        "Oppenheimer",
				Year:         2023,
				Quality:      "1080p",
				Source:       "BluRay",
				VideoCodec:   "x265",
				ReleaseGroup: "YTS",
			},
		},
		{
			name:     "simple year only",
			filename: "Pulp.Fiction.1994.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeMovie,
				Title:     "Pulp Fiction",
				Year:      1994,
			},
		},
		{
			name:     "HDTV source",
			filename: "Some.Movie.2020.720p.HDTV.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeMovie,
				Title:     "Some Movie",
				Year:      2020,
				Quality:   "720p",
				Source:    "HDTV",
			},
		},
		{
			name:     "DVDRip source",
			filename: "Classic.Film.1990.480p.DVDRip.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeMovie,
				Title:     "Classic Film",
				Year:      1990,
				Quality:   "480p",
				Source:    "DVDRip",
			},
		},
		{
			name:     "WEBRip source",
			filename: "New.Release.2024.1080p.WEBRip.x264.mkv",
			want: &ParseResult{
				Status:     ParseStatusSuccess,
				MediaType:  MediaTypeMovie,
				Title:      "New Release",
				Year:       2024,
				Quality:    "1080p",
				Source:     "WEBRip",
				VideoCodec: "x264",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.filename)

			require.NotNil(t, result)
			assert.Equal(t, tt.want.Status, result.Status, "status mismatch")
			assert.Equal(t, tt.want.MediaType, result.MediaType, "media type mismatch")
			assert.Equal(t, tt.want.Title, result.Title, "title mismatch")
			assert.Equal(t, tt.want.Year, result.Year, "year mismatch")

			if tt.want.Quality != "" {
				assert.Equal(t, tt.want.Quality, result.Quality, "quality mismatch")
			}
			if tt.want.Source != "" {
				assert.Equal(t, tt.want.Source, result.Source, "source mismatch")
			}
			if tt.want.VideoCodec != "" {
				assert.Equal(t, tt.want.VideoCodec, result.VideoCodec, "codec mismatch")
			}
			if tt.want.ReleaseGroup != "" {
				assert.Equal(t, tt.want.ReleaseGroup, result.ReleaseGroup, "release group mismatch")
			}
		})
	}
}

func TestMovieParser_CannotParse(t *testing.T) {
	parser := NewMovieParser()

	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "TV show format",
			filename: "Breaking.Bad.S01E05.720p.BluRay.mkv",
		},
		{
			name:     "anime fansub format",
			filename: "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080).mkv",
		},
		{
			name:     "Chinese fansub format",
			filename: "【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P.mp4",
		},
		{
			name:     "no year",
			filename: "Some.Random.File.1080p.mkv",
		},
		{
			name:     "just a filename",
			filename: "movie.mkv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.filename)

			require.NotNil(t, result)
			assert.NotEqual(t, ParseStatusSuccess, result.Status, "should not successfully parse: %s", tt.filename)
		})
	}
}

func TestMovieParser_CanParse(t *testing.T) {
	parser := NewMovieParser()

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"standard movie format", "The.Matrix.1999.1080p.BluRay.mkv", true},
		{"movie with parentheses year", "The Dark Knight (2008) 1080p.mkv", true},
		{"TV show format", "Breaking.Bad.S01E05.720p.BluRay.mkv", false},
		{"anime fansub", "[Group] Anime - 01 [1080p].mkv", false},
		{"no year", "Some.File.1080p.mkv", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parser.CanParse(tt.filename))
		})
	}
}

func TestMovieParser_ImplementsInterface(t *testing.T) {
	var _ Parser = (*MovieParser)(nil)
}
