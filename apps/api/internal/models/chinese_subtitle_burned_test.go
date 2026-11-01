package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// disc-2026-11-burned-in-subs-b1 AC #6: the burned-in value is the verdict's
// fourth input. Burned Chinese is Chinese evidence ranked with the rest
// (Traditional > Simplified > untold); burned "none" is one more known
// non-Chinese source — it never turns "no track info" into "none" by itself.
func TestChineseSubtitleVerdict_BurnedIn(t *testing.T) {
	const (
		noTracks  = ""
		empty     = `[]`
		engOnly   = `[{"language":"eng","format":"subrip","external":false,"stream_index":2}]`
		chiTrack  = `[{"language":"chi","format":"subrip","external":false,"stream_index":2}]`
		hansTrack = `[{"language":"zh-CN","format":"srt","external":true,"stream_index":0}]`
		hantTrack = `[{"language":"zh-TW","format":"srt","external":true,"stream_index":0}]`
		undTrack  = `[{"language":"und","format":"srt","external":true,"stream_index":0}]`
	)
	cases := []struct {
		name   string
		status string
		tracks string
		burned string
		want   ChineseSubtitle
	}{
		// The four burned values on a file with no tracks ('[]' — the only
		// episodes the check writes to).
		{"[] + burned zh-Hant → zh_hant", "not_searched", empty, BurnedInZhHant, ChineseSubtitleZhHant},
		{"[] + burned zh-Hans → zh_hans", "not_searched", empty, BurnedInZhHans, ChineseSubtitleZhHans},
		{"[] + burned zh → zh", "not_searched", empty, BurnedInZh, ChineseSubtitleZh},
		{"[] + burned none → none", "not_searched", empty, BurnedInNone, ChineseSubtitleNone},
		{"[] + never checked → none (unchanged)", "not_searched", empty, "", ChineseSubtitleNone},

		// The four burned values with no track information at all.
		{"NULL tracks + burned zh-Hant → zh_hant", "not_searched", noTracks, BurnedInZhHant, ChineseSubtitleZhHant},
		{"NULL tracks + burned zh-Hans → zh_hans", "not_searched", noTracks, BurnedInZhHans, ChineseSubtitleZhHans},
		{"NULL tracks + burned zh → zh", "not_searched", noTracks, BurnedInZh, ChineseSubtitleZh},
		{"NULL tracks + burned none stays unknown", "not_searched", noTracks, BurnedInNone, ChineseSubtitleUnknown},
		{"NULL tracks + not_found + burned none → none (status decides)", "not_found", noTracks, BurnedInNone, ChineseSubtitleNone},

		// Tracks AND a burned value: ranked together.
		{"English track + burned zh-Hans → zh_hans", "not_searched", engOnly, BurnedInZhHans, ChineseSubtitleZhHans},
		{"English track + burned none → none", "not_searched", engOnly, BurnedInNone, ChineseSubtitleNone},
		{"chi track + burned zh-Hans → zh_hans (Simplified beats untold)", "not_searched", chiTrack, BurnedInZhHans, ChineseSubtitleZhHans},
		{"zh-TW track + burned zh-Hans → zh_hant (Traditional beats Simplified)", "not_searched", hantTrack, BurnedInZhHans, ChineseSubtitleZhHant},
		{"zh-CN track + burned zh-Hant → zh_hant", "not_searched", hansTrack, BurnedInZhHant, ChineseSubtitleZhHant},
		{"zh-CN track + burned zh → zh_hans (untold never downgrades)", "not_searched", hansTrack, BurnedInZh, ChineseSubtitleZhHans},
		{"zh-CN track + burned none → zh_hans (a track's Chinese stands)", "not_searched", hansTrack, BurnedInNone, ChineseSubtitleZhHans},
		{"und track + burned none → unknown (the und track may be Chinese)", "not_searched", undTrack, BurnedInNone, ChineseSubtitleUnknown},
		{"found zh-Hant + burned zh-Hans → zh_hant", "found", noTracks, BurnedInZhHans, ChineseSubtitleZhHant},

		{"an unrecognised burned value says nothing", "not_searched", empty, "maybe", ChineseSubtitleNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			language := ""
			if tc.status == "found" {
				language = "zh-Hant"
			}
			assert.Equal(t, tc.want, ChineseSubtitleVerdict(tc.status, language, tc.tracks, tc.burned))
		})
	}
}

func TestHasTraditionalChineseSubtitle_CountsBurnedIn(t *testing.T) {
	assert.True(t, HasTraditionalChineseSubtitle("not_searched", "", `[]`, BurnedInZhHant))
	assert.False(t, HasTraditionalChineseSubtitle("not_searched", "", `[]`, BurnedInZhHans))
}

func TestIsBurnedInValue(t *testing.T) {
	for _, v := range []string{BurnedInZhHant, BurnedInZhHans, BurnedInZh, BurnedInNone} {
		assert.True(t, IsBurnedInValue(v), v)
	}
	for _, v := range []string{"", "zh_hans", "unknown", "ZH-HANS"} {
		assert.False(t, IsBurnedInValue(v), v)
	}
}
