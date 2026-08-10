package subtitle

// Story sub-4-1 AC #3 — probe-only route prediction.
//
// The cost-preview screen has to tell the user which files would need PAID
// speech recognition BEFORE anything is spent. That answer is decidable from
// the probe alone, and this suite pins the two properties that make it safe to
// run over a whole library: the classification is correct, and it never
// extracts (extraction is the expensive half — an ffmpeg pass per file).

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/services"
)

func predictRouter(t *testing.T, tracks []services.SubtitleTrack, probeErr error) (*Router, *fakeProber, *fakeExtractor) {
	t.Helper()
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: tracks}, err: probeErr}
	extractor := &fakeExtractor{}
	return NewRouter(prober, extractor, nil), prober, extractor
}

func TestPredictRoute_Classification(t *testing.T) {
	tests := []struct {
		name   string
		tracks []services.SubtitleTrack
		want   RoutePrediction
		why    string
	}{
		{
			name:   "embedded English text track — extractable, no ASR spend",
			tracks: []services.SubtitleTrack{{Language: "eng", Format: "subrip", StreamIndex: 2}},
			want:   PredictExtract,
		},
		{
			name:   "embedded Chinese text track — extractable",
			tracks: []services.SubtitleTrack{{Language: "chi", Format: "ass", StreamIndex: 3}},
			want:   PredictExtract,
		},
		{
			name:   "image-only tracks — nothing to extract, speech recognition is the only route",
			tracks: []services.SubtitleTrack{{Language: "eng", Format: "hdmv_pgs_subtitle", StreamIndex: 2}},
			want:   PredictASR,
		},
		{
			name:   "no subtitle tracks at all — speech recognition",
			tracks: nil,
			want:   PredictASR,
		},
		{
			name:   "text tracks present but none tagged Chinese or eng/en — pipeline declines it",
			tracks: []services.SubtitleTrack{{Language: "und", Format: "subrip", StreamIndex: 2}},
			want:   PredictSkip,
			why:    "an und-tagged track is never treated as English (P0) — such an item is not actionable by generation at all",
		},
		{
			name: "external sidecar is ignored — M1 is embedded-only",
			tracks: []services.SubtitleTrack{
				{Language: "eng", Format: "subrip", External: true},
			},
			want: PredictASR,
			why:  "a sidecar is the search engine's domain; it must not make the item look extractable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router, prober, extractor := predictRouter(t, tc.tracks, nil)

			got, err := router.PredictRoute(context.Background(), "/media/Movie.mkv")
			require.NoError(t, err)
			assert.Equal(t, tc.want, got, tc.why)

			assert.Equal(t, 1, prober.callCount, "exactly one probe per prediction")
			assert.Zero(t, extractor.callCount,
				"prediction must NEVER extract — running ffmpeg per file over a whole library is the cost this screen exists to avoid")
		})
	}
}

func TestPredictRoute_ProbeFailurePropagates(t *testing.T) {
	router, _, extractor := predictRouter(t, nil, errors.New("ffprobe exploded"))

	_, err := router.PredictRoute(context.Background(), "/media/Movie.mkv")
	require.Error(t, err,
		"a failed probe must not silently classify as ASR — that would quote the user a paid route for a file we could not read")
	assert.Zero(t, extractor.callCount)
}

// PredictRoute is also the classifier for tracks ALREADY persisted at scan time
// (movies carry subtitle_tracks JSON), so the pure half must be reusable
// without a probe — otherwise every movie would be re-probed for nothing.
func TestPredictFromTracks_IsUsableWithoutProbing(t *testing.T) {
	assert.Equal(t, PredictExtract,
		PredictFromTracks([]services.SubtitleTrack{{Language: "eng", Format: "subrip"}}))
	assert.Equal(t, PredictASR,
		PredictFromTracks([]services.SubtitleTrack{{Language: "eng", Format: "dvd_subtitle"}}))
	assert.Equal(t, PredictASR, PredictFromTracks(nil))
	assert.Equal(t, PredictSkip,
		PredictFromTracks([]services.SubtitleTrack{{Language: "und", Format: "subrip"}}))
}
