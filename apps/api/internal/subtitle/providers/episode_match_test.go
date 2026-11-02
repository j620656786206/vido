package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchesEpisode(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"天蝎 第一季 WEB精校版（全）/Scorpion S01E01-E22", true},
		{"Scorpion.S01E01-22.720p", true},
		{"天蝎 第一季 第十九集/Scorpion S01 E19", false},
		{"Scorpion.S01E02.Single.Point.of.Failure.720p", true},
		{"Scorpion.S02E02.720p", false},
		{"凶鬼恶灵 第1季【蓝光精校】", true},
		{"Scorpion.S01.720p.WEB-DL", true},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, MatchesEpisode(tc.name, 1, 2), tc.name)
	}
}
