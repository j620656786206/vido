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
		{"dotted year at the end", "Ip.Man.2008", true},
		{"extension only", "movie.mkv", true},
		{"extension case-insensitive", "Some Film.MP4", true},
		{"dotted codec tag", "Show.Name.HEVC.x265", true},
		{"spaced release name (CR M5)", "Predator Badlands 2025 2160p WEB-DL", true},
		{"spaced with BluRay", "Some Film 2019 1080p BluRay", true},
		{"bracket tag + dotted year without resolution", "[grp] Some.Show.2024.Complete", true},

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
		{"bracket-titled film (CR H2)", "[REC]", false},
		{"bracket-titled sequel", "[REC] 2", false},
		{"bracket-titled with accents", "[Rec]³ Génesis", false},
		{"two-letter tag is not evidence (CR M6)", "Dun.DV.Two", false},
		{"dotted accented year-only pair", "Amélie.2001", false},
		{"numeric title", "2001: A Space Odyssey", false},
		{"asterisk title", "M*A*S*H", false},
		{"se7en", "Se7en", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, LooksLikeFilename(tt.in), tt.in)
		})
	}
}

func TestUntrustedTitleReason(t *testing.T) {
	assert.Equal(t, "", UntrustedTitleReason("Dune: Part Two", true))
	assert.Equal(t, "", UntrustedTitleReason("Wake Up Dead Man", false), "a clean parsed or hand-typed title on an unmatched row is trusted")
	assert.Equal(t, "filename-shaped", UntrustedTitleReason("Predator.Badlands.2025.2160p.WEB-DL", true))
	assert.Equal(t, "unmatched-filename-shaped", UntrustedTitleReason("[bitsearch.to] Wake.Up.Dead.Man.2025.2160p.mkv", false))
}
