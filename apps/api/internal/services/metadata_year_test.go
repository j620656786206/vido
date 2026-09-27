package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// bugfix-editor-year-keeps-unknown-and-full-date: a save must not rewrite the
// release date unless the year really changed, and then keeps month and day.

func TestWithYear(t *testing.T) {
	for _, tc := range []struct {
		date string
		year int
		want string
	}{
		{"2016-08-26", 0, "2016-08-26"},    // no year sent → untouched
		{"2016-08-26", 2016, "2016-08-26"}, // same year → untouched
		{"2016-08-26", 2017, "2017-08-26"}, // new year keeps month/day
		{"", 2017, "2017-01-01"},           // unknown date → Jan 1
		{"", 0, ""},                        // unknown stays unknown
		{"2016", 2018, "2018-01-01"},       // not a full date → Jan 1
		{"2020-02-29", 2021, "2021-02-28"}, // leap day into a common year
		{"2020-02-29", 2024, "2024-02-29"}, // …and into another leap year
	} {
		assert.Equal(t, tc.want, withYear(tc.date, tc.year), "%q + %d", tc.date, tc.year)
	}
}

func TestUpdateMetadataRequest_YearIsOptionalButBounded(t *testing.T) {
	ok := &UpdateMetadataRequest{ID: "m1", Title: "片名"}
	assert.NoError(t, ok.Validate(), "no year = leave the date alone")

	for _, y := range []int{1899, 2101, -5} {
		r := &UpdateMetadataRequest{ID: "m1", Title: "片名", Year: y}
		assert.ErrorIs(t, r.Validate(), ErrUpdateMetadataYearOutOfRange, "%d", y)
	}
}

func TestMetadataEditService_UpdateMetadata_KeepsTheReleaseDate(t *testing.T) {
	movieRepo := newMockMovieRepo()
	movieRepo.movies["m1"] = &models.Movie{ID: "m1", Title: "你的名字", ReleaseDate: "2016-08-26"}
	seriesRepo := newMockSeriesRepo()
	seriesRepo.series["s1"] = &models.Series{ID: "s1", Title: "S", FirstAirDate: "2019-04-06"}
	svc := NewMetadataEditService(movieRepo, seriesRepo, nil)
	ctx := context.Background()

	// Only the title changed (the editor no longer sends an untouched year).
	_, err := svc.UpdateMetadata(ctx, &UpdateMetadataRequest{ID: "m1", MediaType: "movie", Title: "你的名字。"})
	require.NoError(t, err)
	assert.Equal(t, "2016-08-26", movieRepo.movies["m1"].ReleaseDate)

	// Same year sent by another caller → still untouched.
	_, err = svc.UpdateMetadata(ctx, &UpdateMetadataRequest{ID: "m1", MediaType: "movie", Title: "你的名字。", Year: 2016})
	require.NoError(t, err)
	assert.Equal(t, "2016-08-26", movieRepo.movies["m1"].ReleaseDate)

	// Series, year changed → month/day kept.
	_, err = svc.UpdateMetadata(ctx, &UpdateMetadataRequest{ID: "s1", MediaType: "series", Title: "S", Year: 2020})
	require.NoError(t, err)
	assert.Equal(t, "2020-04-06", seriesRepo.series["s1"].FirstAirDate)
}
