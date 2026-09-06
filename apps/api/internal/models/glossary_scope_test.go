package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSharedGlossaryScope(t *testing.T) {
	cases := []struct {
		scope  string
		kind   string
		tmdbID int64
		ok     bool
	}{
		{"tmdb:tv:1396", "tv", 1396, true},
		{"tmdb:movie:550", "movie", 550, true},
		{GlossaryScopeTV(66732), "tv", 66732, true},
		{"local:abc", "", 0, false},
		{"tmdb:tv:", "", 0, false},
		{"tmdb:tv:x", "", 0, false},
		{"tmdb:tv:0", "", 0, false},
		{"tmdb:tv:-5", "", 0, false},
		{"", "", 0, false},
	}
	for _, c := range cases {
		kind, id, ok := ParseSharedGlossaryScope(c.scope)
		assert.Equal(t, c.ok, ok, c.scope)
		assert.Equal(t, c.kind, kind, c.scope)
		assert.Equal(t, c.tmdbID, id, c.scope)
	}
}
