package models

import "testing"

func TestParseGlossarySeasonScope(t *testing.T) {
	base, season, ok := ParseGlossarySeasonScope("tmdb:tv:80752:s1")
	if !ok || base != "tmdb:tv:80752" || season != 1 {
		t.Fatalf("got %q %d %v", base, season, ok)
	}
	if _, _, ok := ParseGlossarySeasonScope("tmdb:tv:80752"); ok {
		t.Fatal("show-wide scope is not a drawer")
	}
	if _, _, ok := ParseGlossarySeasonScope("local:abc:s1"); ok {
		t.Fatal("drawers exist only under tmdb:tv")
	}
	if GlossarySeasonScope("tmdb:tv:80752", 2) != "tmdb:tv:80752:s2" {
		t.Fatal("round trip")
	}
}
