package main

import (
	"context"

	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
)

// routePredictorAdapter bridges *subtitle.Router to services.RoutePredictor
// (story sub-4-1).
//
// It lives HERE for the same reason subtitlePlacerAdapter and
// pipelineASRAdapter do: `services` must not import `subtitle` (Rule 19 —
// subtitle already imports services, so the reverse edge is an import cycle).
// main.go is the one place that legitimately sees both sides.
//
// The two RoutePrediction types are the documented Mirror-Types workaround;
// route_prediction_parity_test.go fails if their values ever diverge, so the
// string conversions below cannot silently mistranslate a route.
type routePredictorAdapter struct {
	router *subtitle.Router
}

// FromTracks classifies an already-persisted track list — no disk access.
func (a routePredictorAdapter) FromTracks(tracks []services.SubtitleTrack) services.RoutePrediction {
	return services.RoutePrediction(subtitle.PredictFromTracks(tracks))
}

// Probe classifies a file by probing it. Never extracts — that is the whole
// point of the probe-only entry point (see subtitle.Router.PredictRoute).
func (a routePredictorAdapter) Probe(ctx context.Context, mediaPath string) (services.RoutePrediction, error) {
	p, err := a.router.PredictRoute(ctx, mediaPath)
	if err != nil {
		return "", err
	}
	return services.RoutePrediction(p), nil
}
