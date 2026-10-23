package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitTrailingYear(t *testing.T) {
	cases := []struct {
		in        string
		wantTitle string
		wantYear  int
	}{
		{"Ever Night 2026", "Ever Night", 2026},
		{"將夜 (2026)", "將夜", 2026},
		{"將夜.2026", "將夜", 2026},
		{"Doctor Who 2005", "Doctor Who", 2005},
		{"將夜", "將夜", 0},
		{"1923", "1923", 0},             // the whole name is the year: a title, not a year
		{"(2026)", "(2026)", 0},         // nothing before it
		{"Station 19", "Station 19", 0}, // not a year
		{"Show 2026 Special", "Show 2026 Special", 0},
		{"Show 1850", "Show 1850", 0},
	}
	for _, tc := range cases {
		title, year := SplitTrailingYear(tc.in)
		assert.Equal(t, tc.wantTitle, title, tc.in)
		assert.Equal(t, tc.wantYear, year, tc.in)
	}
}

// The real Ever Night file: the TV parser keeps the year in its title, and
// SplitTrailingYear is what finds it.
func TestSplitTrailingYear_EverNightFilename(t *testing.T) {
	r := NewTVParser().Parse("[将夜].Ever.Night.2026.S01E01.2160p.WEB-DL.HDR10.H265.10bit.AAC-UBWEB.mp4")
	if assert.NotNil(t, r) {
		title, year := SplitTrailingYear(r.Title)
		assert.Equal(t, "Ever Night", title)
		assert.Equal(t, 2026, year)
	}
}
