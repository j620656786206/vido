package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sub-7-2a AC #1 — the one cast renderer both translation legs share, so the
// two prompts (and their MetadataHash) cannot drift apart.

func TestCredits_CastLabels_NameWithCharacterInFullWidthParens(t *testing.T) {
	c := &Credits{Cast: []CastMember{
		{Name: "Winona Ryder", Character: "Joyce Byers", Order: 0},
		{Name: "David Harbour", Character: "Jim Hopper", Order: 1},
	}}

	assert.Equal(t, []string{"Winona Ryder（Joyce Byers）", "David Harbour（Jim Hopper）"}, c.CastLabels(10))
}

func TestCredits_CastLabels_NoCharacterMeansNoParens(t *testing.T) {
	c := &Credits{Cast: []CastMember{{Name: "Millie Bobby Brown"}, {Name: "Finn Wolfhard", Character: "   "}}}

	assert.Equal(t, []string{"Millie Bobby Brown", "Finn Wolfhard"}, c.CastLabels(10))
}

func TestCredits_CastLabels_KeepsBillingOrderAndCapsAtLimit(t *testing.T) {
	c := &Credits{}
	for i := 1; i <= 15; i++ {
		c.Cast = append(c.Cast, CastMember{Name: "Actor " + string(rune('A'+i-1)), Order: i})
	}

	got := c.CastLabels(10)
	require.Len(t, got, 10)
	assert.Equal(t, "Actor A", got[0], "billing order is the prompt order — never sorted")
	assert.Equal(t, "Actor J", got[9])
	assert.Len(t, c.CastLabels(0), 15, "limit <= 0 means no cap")
}

func TestCredits_CastLabels_BlankNamesAreSkippedNotCounted(t *testing.T) {
	c := &Credits{Cast: []CastMember{{Name: "  "}, {Name: "", Character: "Ghost"}, {Name: "Real Actor"}}}

	assert.Equal(t, []string{"Real Actor"}, c.CastLabels(2))
}

func TestCredits_CastLabels_EmptyIsNil(t *testing.T) {
	var none *Credits
	assert.Nil(t, none.CastLabels(10))
	assert.Nil(t, (&Credits{}).CastLabels(10))
	assert.Nil(t, (&Credits{Cast: []CastMember{{Name: " "}}}).CastLabels(10))
}

// The repository scan fills the parsed Credits pointer only when the JSON is
// non-empty; a row read another way still carries the JSON blob, and the prompt
// must not go cast-blind because of which path loaded it.
func TestMovie_CastLabels_FallsBackToTheStoredJSON(t *testing.T) {
	m := &Movie{}
	require.NoError(t, m.SetCredits(&Credits{Cast: []CastMember{{Name: "Keanu Reeves", Character: "Neo"}}}))
	require.Nil(t, m.Credits, "precondition: only the JSON is set")

	assert.Equal(t, []string{"Keanu Reeves（Neo）"}, m.CastLabels(10))

	m.Credits = &Credits{Cast: []CastMember{{Name: "Carrie-Anne Moss", Character: "Trinity"}}}
	assert.Equal(t, []string{"Carrie-Anne Moss（Trinity）"}, m.CastLabels(10), "the parsed pointer wins when present")
	assert.Nil(t, (&Movie{}).CastLabels(10))
}

func TestSeries_CastLabels_FallsBackToTheStoredJSON(t *testing.T) {
	s := &Series{}
	require.NoError(t, s.SetCredits(&Credits{Cast: []CastMember{{Name: "Bryan Cranston", Character: "Walter White"}}}))

	assert.Equal(t, []string{"Bryan Cranston（Walter White）"}, s.CastLabels(10))
	assert.Nil(t, (&Series{}).CastLabels(10))
}

// sub-7-2b: the series countries column (migration 040), mirroring Movie.
func TestSeries_ProductionCountries_RoundTrip(t *testing.T) {
	s := &Series{}
	got, err := s.GetProductionCountries()
	require.NoError(t, err)
	assert.Empty(t, got)

	require.NoError(t, s.SetProductionCountries([]ProductionCountry{{ISO3166_1: "CN", Name: "China"}}))
	assert.True(t, s.ProductionCountriesJSON.Valid)
	assert.Equal(t, []ProductionCountry{{ISO3166_1: "CN", Name: "China"}}, s.ProductionCountries, "the parsed field is kept in step")
	got, err = s.GetProductionCountries()
	require.NoError(t, err)
	assert.Equal(t, []ProductionCountry{{ISO3166_1: "CN", Name: "China"}}, got)

	require.NoError(t, s.SetProductionCountries(nil))
	assert.False(t, s.ProductionCountriesJSON.Valid)
	assert.Nil(t, s.ProductionCountries)
}
