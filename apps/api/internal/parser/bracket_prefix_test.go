package parser

import (
	"bufio"
	"os"
	"testing"
)

// The corpus mirrors the exact set of episode files that the 2026-08-24 NAS rescan failed
// to turn into episode rows — each collapsed onto a single `episode_number = 0` placeholder
// per series, so the library silently lost ~17% of its episodes.
//
// Show titles and release-group tags are replaced with placeholders; every structural
// feature the parser keys on (bracket prefixes, separators, episode numbering style,
// resolution and codec tokens, CJK runs) is preserved verbatim, and the coverage figure is
// identical to the one measured on the raw names.
const nasCorpusPath = "testdata/nas-tv-filenames-2026-08.txt"

// rescuedFloor is the number of corpus filenames TVParser must resolve to a concrete
// season/episode. Pinned so a future pattern change that silently narrows coverage fails
// here instead of on someone's disk.
const rescuedFloor = 404

func loadNASCorpus(t *testing.T) []string {
	t.Helper()

	f, err := os.Open(nasCorpusPath)
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	defer f.Close()

	var names []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		if name := sc.Text(); name != "" {
			names = append(names, name)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("corpus is empty")
	}
	return names
}

func TestTVParserResolvesNASCorpus(t *testing.T) {
	names := loadNASCorpus(t)
	p := NewTVParser()

	var resolved int
	for _, name := range names {
		r := p.Parse(name)
		if r.Status == ParseStatusSuccess && r.MediaType == MediaTypeTVShow && r.Episode > 0 {
			resolved++
		}
	}

	if resolved < rescuedFloor {
		t.Errorf("resolved %d/%d corpus filenames, want at least %d", resolved, len(names), rescuedFloor)
	}
	t.Logf("corpus coverage: %d/%d (%.1f%%)", resolved, len(names),
		float64(resolved)/float64(len(names))*100)
}

// Each case mirrors a real filename shape from the corpus (titles and group tags replaced
// with placeholders), kept separate from the bulk count so a failure names the shape that
// broke rather than just moving a number.
func TestTVParserLeadingReleaseGroupTag(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		season   int
		episode  int
		title    string
	}{
		{
			name:     "chinese bracket title with standard SxxExx",
			filename: "[劇名01].Some.Drama.S01E48.2026.2160p.WEB-DL.H265.DV.DDP5.1-GRP.mkv",
			season:   1,
			episode:  48,
			title:    "Some Drama",
		},
		{
			name:     "fansub group with spaced SxxExx",
			filename: "[Grp01] Some Long Show Title S01E05 1080p AMZN WEB-DL DUAL DDP2.0 H.264.mkv",
			season:   1,
			episode:  5,
			title:    "Some Long Show Title",
		},
		{
			name:     "tracker tag with season two",
			filename: "[grp02.example] Some Show 2021 S03E03 Subtitle 1080p ATVP WEB-DL DDP5 1 Atmos H 264-GRP.mkv",
			season:   3,
			episode:  3,
			title:    "Some Show 2021",
		},
		{
			name:     "double closing bracket does not break the strip",
			filename: "[Grp03][劇名02 第一季]]Some.Doc.2019.S01E02.2160p.BluRay.HEVC.AAC.mp4",
			season:   1,
			episode:  2,
			title:    "Some Doc 2019",
		},
	}

	p := NewTVParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := p.Parse(tt.filename)

			if r.Status != ParseStatusSuccess {
				t.Fatalf("status = %v, want %v", r.Status, ParseStatusSuccess)
			}
			if r.MediaType != MediaTypeTVShow {
				t.Errorf("mediaType = %v, want %v", r.MediaType, MediaTypeTVShow)
			}
			if r.Season != tt.season {
				t.Errorf("season = %d, want %d", r.Season, tt.season)
			}
			if r.Episode != tt.episode {
				t.Errorf("episode = %d, want %d", r.Episode, tt.episode)
			}
			if r.Title != tt.title {
				t.Errorf("title = %q, want %q", r.Title, tt.title)
			}
			// The caller stores this verbatim; stripping must not leak into it.
			if r.OriginalFilename != tt.filename {
				t.Errorf("originalFilename = %q, want the untouched input", r.OriginalFilename)
			}
			if !p.CanParse(tt.filename) {
				t.Error("CanParse = false for a filename Parse resolves")
			}
		})
	}
}

func TestTVParserStillGivesUpOnUnparseableNames(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"nothing but a tag", "[Grp04].mkv"},
		{"tag with no episode marker at all", "[Group] Some Movie Title 1080p BluRay.mkv"},
		{"empty after strip", "[a][b][c]"},
	}

	p := NewTVParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := p.Parse(tt.filename)

			if r.Status != ParseStatusNeedsAI {
				t.Errorf("status = %v, want %v", r.Status, ParseStatusNeedsAI)
			}
			// A fabricated season/episode is worse than admitting defeat: it silently
			// collides with every other unparsed file in the same series.
			if r.Season != 0 || r.Episode != 0 {
				t.Errorf("season/episode = %d/%d, want 0/0 for an unresolved name", r.Season, r.Episode)
			}
		})
	}
}

func TestTVShowPatternIgnoresVideoResolution(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"1920x1080 is a resolution, not 20x108", "Some Anime Title - 82 (B-Global Donghua 1920x1080 HEVC AAC MKV).mkv", false},
		{"1280x720 is a resolution, not 80x720", "Show - 05 (1280x720).mkv", false},
		{"genuine 1x05 still counts as a TV marker", "Show.1x05.mkv", true},
		{"genuine 12x05 still counts as a TV marker", "Show.12x05.mkv", true},
		{"SxxExx still counts as a TV marker", "Show.S01E05.mkv", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tvShowPattern.MatchString(tt.filename); got != tt.want {
				t.Errorf("tvShowPattern.MatchString(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestTVParserResolvesAnimeDashDespiteResolution(t *testing.T) {
	p := NewTVParser()
	// Previously unreachable: the resolution token tripped the SxxExx guard in
	// parseAnimeDashFormat, so the dash pattern was never tried.
	r := p.Parse("[Grp05] Some Anime Title - 82 (B-Global Donghua 1920x1080 HEVC AAC MKV) [6AE83A26].mkv")

	if r.Status != ParseStatusSuccess {
		t.Fatalf("status = %v, want %v", r.Status, ParseStatusSuccess)
	}
	if r.Episode != 82 {
		t.Errorf("episode = %d, want 82", r.Episode)
	}
}

func TestStripLeadingTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no tag is a no-op", "Show.S01E05.mkv", "Show.S01E05.mkv"},
		{"single tag", "[Group] Show S01E05.mkv", "Show S01E05.mkv"},
		{"consecutive tags", "[Grp03][劇名02]Some.Doc.S01E02.mp4", "Some.Doc.S01E02.mp4"},
		{"dot separator after tag", "[劇名01].Some.Drama.S01E48.mkv", "Some.Drama.S01E48.mkv"},
		{"stray closing bracket is left alone", "[Grp03][x]]Some.Doc.mp4", "Some.Doc.mp4"},
		{"only tags yields empty", "[a][b]", ""},
		{"trailing bracket is not a leading tag", "Show.S01E05 [ABC123].mkv", "Show.S01E05 [ABC123].mkv"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripLeadingTags(tt.input); got != tt.want {
				t.Errorf("StripLeadingTags(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
