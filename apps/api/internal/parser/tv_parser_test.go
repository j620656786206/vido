package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTVParser_StandardFormat(t *testing.T) {
	parser := NewTVParser()

	tests := []struct {
		name     string
		filename string
		want     *ParseResult
	}{
		{
			name:     "standard S01E05 format",
			filename: "Breaking.Bad.S01E05.720p.BluRay.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeTVShow,
				Title:     "Breaking Bad",
				Season:    1,
				Episode:   5,
				Quality:   "720p",
				Source:    "BluRay",
			},
		},
		{
			name:     "double digit season and episode",
			filename: "Game.of.Thrones.S08E06.1080p.WEB-DL.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeTVShow,
				Title:     "Game of Thrones",
				Season:    8,
				Episode:   6,
				Quality:   "1080p",
				Source:    "WEB-DL",
			},
		},
		{
			name:     "lowercase s and e",
			filename: "The.Office.s02e10.720p.WEBRip.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeTVShow,
				Title:     "The Office",
				Season:    2,
				Episode:   10,
				Quality:   "720p",
				Source:    "WEBRip",
			},
		},
		{
			name:     "episode range E01-E02",
			filename: "Friends.S01E01-E02.DVDRip.mkv",
			want: &ParseResult{
				Status:     ParseStatusSuccess,
				MediaType:  MediaTypeTVShow,
				Title:      "Friends",
				Season:     1,
				Episode:    1,
				EpisodeEnd: 2,
				Source:     "DVDRip",
			},
		},
		{
			name:     "episode range E01-02",
			filename: "House.S03E15-16.720p.mkv",
			want: &ParseResult{
				Status:     ParseStatusSuccess,
				MediaType:  MediaTypeTVShow,
				Title:      "House",
				Season:     3,
				Episode:    15,
				EpisodeEnd: 16,
				Quality:    "720p",
			},
		},
		{
			name:     "with codec and release group",
			filename: "Stranger.Things.S04E09.1080p.WEB-DL.x265-GROUP.mkv",
			want: &ParseResult{
				Status:       ParseStatusSuccess,
				MediaType:    MediaTypeTVShow,
				Title:        "Stranger Things",
				Season:       4,
				Episode:      9,
				Quality:      "1080p",
				Source:       "WEB-DL",
				VideoCodec:   "x265",
				ReleaseGroup: "GROUP",
			},
		},
		{
			name:     "space separated",
			filename: "The Mandalorian S02E01 720p.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeTVShow,
				Title:     "The Mandalorian",
				Season:    2,
				Episode:   1,
				Quality:   "720p",
			},
		},
		{
			name:     "underscore separated",
			filename: "Dark_S03E08_1080p_WEB-DL.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeTVShow,
				Title:     "Dark",
				Season:    3,
				Episode:   8,
				Quality:   "1080p",
				Source:    "WEB-DL",
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
			assert.Equal(t, tt.want.Season, result.Season, "season mismatch")
			assert.Equal(t, tt.want.Episode, result.Episode, "episode mismatch")

			if tt.want.EpisodeEnd != 0 {
				assert.Equal(t, tt.want.EpisodeEnd, result.EpisodeEnd, "episode end mismatch")
			}
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

func TestTVParser_AlternativeFormat(t *testing.T) {
	parser := NewTVParser()

	tests := []struct {
		name     string
		filename string
		want     *ParseResult
	}{
		{
			name:     "1x05 format",
			filename: "House.1x13.720p.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeTVShow,
				Title:     "House",
				Season:    1,
				Episode:   13,
				Quality:   "720p",
			},
		},
		{
			name:     "2x10 format",
			filename: "Scrubs.2x10.DVDRip.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeTVShow,
				Title:     "Scrubs",
				Season:    2,
				Episode:   10,
				Source:    "DVDRip",
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
			assert.Equal(t, tt.want.Season, result.Season, "season mismatch")
			assert.Equal(t, tt.want.Episode, result.Episode, "episode mismatch")

			if tt.want.Quality != "" {
				assert.Equal(t, tt.want.Quality, result.Quality, "quality mismatch")
			}
			if tt.want.Source != "" {
				assert.Equal(t, tt.want.Source, result.Source, "source mismatch")
			}
		})
	}
}

func TestTVParser_DailyShowFormat(t *testing.T) {
	parser := NewTVParser()

	tests := []struct {
		name     string
		filename string
		want     *ParseResult
	}{
		{
			name:     "daily show format",
			filename: "The.Daily.Show.2024.01.15.720p.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeTVShow,
				Title:     "The Daily Show",
				Year:      2024,
				Quality:   "720p",
			},
		},
		{
			name:     "late night show",
			filename: "The.Tonight.Show.2023.12.25.1080p.WEB-DL.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeTVShow,
				Title:     "The Tonight Show",
				Year:      2023,
				Quality:   "1080p",
				Source:    "WEB-DL",
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
		})
	}
}

func TestTVParser_AnimeFormat(t *testing.T) {
	parser := NewTVParser()

	// Simple anime formats that use standard numbering
	tests := []struct {
		name     string
		filename string
		want     *ParseResult
	}{
		{
			name:     "anime with Episode prefix",
			filename: "Naruto.Episode.01.720p.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeTVShow,
				Title:     "Naruto",
				Episode:   1,
				Quality:   "720p",
			},
		},
		{
			name:     "anime with Ep prefix",
			filename: "One.Piece.Ep.100.1080p.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeTVShow,
				Title:     "One Piece",
				Episode:   100,
				Quality:   "1080p",
			},
		},
		{
			name:     "anime with just number after dash",
			filename: "Attack.on.Titan.-.01.720p.mkv",
			want: &ParseResult{
				Status:    ParseStatusSuccess,
				MediaType: MediaTypeTVShow,
				Title:     "Attack on Titan",
				Episode:   1,
				Quality:   "720p",
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
			assert.Equal(t, tt.want.Episode, result.Episode, "episode mismatch")

			if tt.want.Quality != "" {
				assert.Equal(t, tt.want.Quality, result.Quality, "quality mismatch")
			}
		})
	}
}

func TestTVParser_CannotParse(t *testing.T) {
	parser := NewTVParser()

	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "movie format",
			filename: "The.Matrix.1999.1080p.BluRay.mkv",
		},
		{
			name:     "fansub bracket format",
			filename: "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080).mkv",
		},
		{
			name:     "Chinese fansub",
			filename: "【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P.mp4",
		},
		{
			name:     "just a filename",
			filename: "video.mkv",
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

func TestTVParser_CanParse(t *testing.T) {
	parser := NewTVParser()

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"standard S01E05 format", "Breaking.Bad.S01E05.720p.BluRay.mkv", true},
		{"1x05 format", "House.1x13.720p.mkv", true},
		{"daily show format", "The.Daily.Show.2024.01.15.720p.mkv", true},
		{"movie format", "The.Matrix.1999.1080p.BluRay.mkv", false},
		{"fansub bracket format", "[Group] Anime - 01 [1080p].mkv", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parser.CanParse(tt.filename))
		})
	}
}

func TestTVParser_ImplementsInterface(t *testing.T) {
	var _ Parser = (*TVParser)(nil)
}
