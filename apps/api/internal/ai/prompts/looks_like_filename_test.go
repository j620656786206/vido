package prompts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLooksLikeFilename(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		// Filenames — the shapes an unmatched row actually carries (eval-1).
		{"tracker prefix + dotted release", "[bitsearch.to] Wake.Up.Dead.Man.2025.2160p.WEB-DL.DDP5.1.Atmos.DV.HDR.H.265-NAHOM.mkv", true},
		{"chinese bracket tag", "[大奉打更人之世间无我这般人].No.One.Like.Me.2025.2160p.60fps.WEB-DL", true},
		{"dotted with resolution, no extension", "Predator.Badlands.2025.2160p.MA.WEB-DL.DV.HDR.TYMBLE", true},
		{"dotted with bare year only", "Youth.2017.HQ", true},
		{"extension only", "movie.mkv", true},
		{"extension case-insensitive", "Some Film.MP4", true},
		{"dotted codec tag", "Show.Name.HEVC.x265", true},

		// Titles — must stay titles.
		{"plain title", "Dune: Part Two", false},
		{"abbreviation with dots", "S.W.A.T.", false},
		{"honorific dot", "Mr. Robot", false},
		{"title with year in words", "Blade Runner 2049", false},
		{"chinese title", "駭客任務", false},
		{"title with parenthesised year", "Youth (2017)", false},
		{"spaced release-looking words are still a title", "Now You See Me", false},
		{"empty", "", false},
		{"two dots but no release token", "Mr.Robot.Season", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, LooksLikeFilename(tt.in), tt.in)
		})
	}
}
