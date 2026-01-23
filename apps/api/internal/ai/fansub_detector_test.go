package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsFansubFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		// Should detect as fansub
		{
			name:     "Japanese fansub with square brackets",
			filename: "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv",
			want:     true,
		},
		{
			name:     "Chinese fansub with fullwidth brackets",
			filename: "【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P【繁體】.mp4",
			want:     true,
		},
		{
			name:     "SubsPlease release",
			filename: "[SubsPlease] Demon Slayer - 01 (1080p) [ABCD1234].mkv",
			want:     true,
		},
		{
			name:     "Commie release",
			filename: "[Commie] Steins;Gate 0 - 01 [BD 1080p AAC] [12345678].mkv",
			want:     true,
		},
		{
			name:     "Chinese simplified fansub",
			filename: "【极影字幕社】★ 进击的巨人 第01话 HDTV 720P 【简体】.mp4",
			want:     true,
		},
		{
			name:     "Erai-raws release",
			filename: "[Erai-raws] One Piece - 1000 [1080p][Multiple Subtitle].mkv",
			want:     true,
		},
		{
			name:     "Chinese episode notation only - needs brackets for high confidence",
			filename: "進撃的巨人 第01話 1080P.mp4",
			want:     false, // Without brackets, confidence is below 0.5 threshold
		},
		// Should NOT detect as fansub
		{
			name:     "standard movie",
			filename: "The.Matrix.1999.1080p.BluRay.mkv",
			want:     false,
		},
		{
			name:     "standard TV show",
			filename: "Breaking.Bad.S01E05.720p.BluRay.mkv",
			want:     false,
		},
		{
			name:     "scene release",
			filename: "Movie.Name.2023.1080p.WEB-DL.DD5.1.H.264-GROUP.mkv",
			want:     false,
		},
		{
			name:     "simple filename",
			filename: "random_video_file.mkv",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsFansubFilename(tt.filename)
			assert.Equal(t, tt.want, got, "IsFansubFilename(%q)", tt.filename)
		})
	}
}

func TestDetectFansub_BracketTypes(t *testing.T) {
	tests := []struct {
		name           string
		filename       string
		wantBracket    FansubBracketType
		wantGroupName  string
		wantIsFansub   bool
	}{
		{
			name:           "square brackets",
			filename:       "[SubsPlease] Anime - 01.mkv",
			wantBracket:    BracketSquare,
			wantGroupName:  "SubsPlease",
			wantIsFansub:   true,
		},
		{
			name:           "fullwidth brackets",
			filename:       "【幻櫻字幕組】動漫 第01話.mp4",
			wantBracket:    BracketFullwidth,
			wantGroupName:  "幻櫻字幕組",
			wantIsFansub:   true,
		},
		{
			name:           "corner brackets",
			filename:       "「字幕組」動漫 第01話.mp4",
			wantBracket:    BracketCorner,
			wantGroupName:  "字幕組",
			wantIsFansub:   true,
		},
		{
			name:           "no brackets",
			filename:       "Movie.2023.1080p.mkv",
			wantBracket:    BracketNone,
			wantGroupName:  "",
			wantIsFansub:   false,
		},
		{
			name:           "brackets with spaces",
			filename:       "[ Leopard-Raws ] Show - 01.mkv",
			wantBracket:    BracketSquare,
			wantGroupName:  "Leopard-Raws",
			wantIsFansub:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectFansub(tt.filename)

			assert.Equal(t, tt.wantBracket, result.BracketType, "BracketType")
			assert.Equal(t, tt.wantGroupName, result.GroupName, "GroupName")
			assert.Equal(t, tt.wantIsFansub, result.IsFansub, "IsFansub")
		})
	}
}

func TestDetectFansub_ChineseEpisodeNotation(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantFound bool
	}{
		{
			name:      "traditional Chinese 話",
			filename:  "【字幕組】動漫 第01話 1080P.mp4",
			wantFound: true,
		},
		{
			name:      "traditional Chinese 集",
			filename:  "【字幕組】動漫 第12集 720P.mp4",
			wantFound: true,
		},
		{
			name:      "simplified Chinese 话",
			filename:  "【字幕组】动漫 第01话 1080P.mp4",
			wantFound: true,
		},
		{
			name:      "with spaces",
			filename:  "動漫 第 01 話.mp4",
			wantFound: true,
		},
		{
			name:      "no Chinese notation",
			filename:  "[SubsPlease] Anime - 01.mkv",
			wantFound: false,
		},
		{
			name:      "standard S01E01 format",
			filename:  "Show.S01E01.mkv",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectFansub(tt.filename)
			assert.Equal(t, tt.wantFound, result.HasChineseEpisode, "HasChineseEpisode")
		})
	}
}

func TestDetectFansub_KnownGroups(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantFound bool
	}{
		{
			name:      "Leopard-Raws",
			filename:  "[Leopard-Raws] Show - 01.mkv",
			wantFound: true,
		},
		{
			name:      "SubsPlease",
			filename:  "[SubsPlease] Anime - 01.mkv",
			wantFound: true,
		},
		{
			name:      "幻櫻字幕組",
			filename:  "【幻櫻字幕組】動漫 第01話.mp4",
			wantFound: true,
		},
		{
			name:      "case insensitive",
			filename:  "[subsplease] anime - 01.mkv",
			wantFound: true,
		},
		{
			name:      "unknown group",
			filename:  "[UnknownGroup] Show - 01.mkv",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectFansub(tt.filename)
			assert.Equal(t, tt.wantFound, result.HasKnownGroup, "HasKnownGroup")
		})
	}
}

func TestDetectFansub_Confidence(t *testing.T) {
	tests := []struct {
		name           string
		filename       string
		minConfidence  float64
		maxConfidence  float64
	}{
		{
			name:           "high confidence - fullwidth + Chinese episode + known group",
			filename:       "【幻櫻字幕組】我的英雄學院 第01話 1080P.mp4",
			minConfidence:  0.9,
			maxConfidence:  1.0,
		},
		{
			name:           "medium-high confidence - square + known group",
			filename:       "[SubsPlease] Anime - 01 (1080p).mkv",
			minConfidence:  0.6,
			maxConfidence:  0.9,
		},
		{
			name:           "medium confidence - square brackets only",
			filename:       "[UnknownGroup] Show - 01.mkv",
			minConfidence:  0.4,
			maxConfidence:  0.6,
		},
		{
			name:           "low confidence - no patterns",
			filename:       "regular.movie.2023.mkv",
			minConfidence:  0.0,
			maxConfidence:  0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectFansub(tt.filename)
			assert.GreaterOrEqual(t, result.Confidence, tt.minConfidence,
				"Confidence should be >= %v", tt.minConfidence)
			assert.LessOrEqual(t, result.Confidence, tt.maxConfidence,
				"Confidence should be <= %v", tt.maxConfidence)
		})
	}
}

func TestContainsCJKCharacters(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"Chinese simplified", "进击的巨人", true},
		{"Chinese traditional", "進擊的巨人", true},
		{"Japanese hiragana", "きめつのやいば", true},
		{"Japanese katakana", "キメツノヤイバ", true},
		{"Japanese kanji", "鬼滅の刃", true},
		{"Korean", "귀멸의 칼날", true},
		{"English only", "Attack on Titan", false},
		{"Mixed English with brackets", "[SubsPlease] Attack on Titan - 01.mkv", false},
		{"Mixed with CJK", "[SubsPlease] 進撃の巨人 - 01.mkv", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsCJKCharacters(tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestContainsEpisodeDashPattern(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"standard dash", "Show - 01", true},
		{"dash with extension", "Show - 01.mkv", true},
		{"dash with brackets", "Show - 01 [1080p]", true},
		{"dash with parentheses", "Show - 01 (1080p)", true},
		{"two digit episode", "Show - 26", true},
		{"three digit episode", "Show - 100", true},
		{"no dash", "Show.S01E01", false},
		{"dash in title", "Spider-Man.2023.mkv", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsEpisodeDashPattern(tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetKnownFansubGroups(t *testing.T) {
	groups := GetKnownFansubGroups()

	assert.NotEmpty(t, groups)
	assert.Contains(t, groups, "Leopard-Raws")
	assert.Contains(t, groups, "SubsPlease")
	assert.Contains(t, groups, "幻櫻字幕組")
}

func TestBracketTypeConstants(t *testing.T) {
	// Verify constants are properly defined
	assert.Equal(t, FansubBracketType("square"), BracketSquare)
	assert.Equal(t, FansubBracketType("fullwidth"), BracketFullwidth)
	assert.Equal(t, FansubBracketType("corner"), BracketCorner)
	assert.Equal(t, FansubBracketType("none"), BracketNone)
}
