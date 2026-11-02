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

// Both optional halves are type-asserted at the call sites; a missing method
// would silently drop the duration or the cue count, so prove them here.
var (
	_ services.RouteDurationPredictor = routePredictorAdapter{}
	_ services.RouteCuePredictor      = routePredictorAdapter{}
)

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

// CuesFromTracks satisfies services.RouteCuePredictor: the cue count of the
// track the extract lane would translate, from an already-known track list.
func (a routePredictorAdapter) CuesFromTracks(tracks []services.SubtitleTrack) int {
	return subtitle.PredictCuesFromTracks(tracks)
}

// ProbeWithCues satisfies services.RouteCuePredictor — route, duration and
// cue count from ONE probe (disc-2026-11-translate-estimate-ignores-cue-density).
func (a routePredictorAdapter) ProbeWithCues(ctx context.Context, mediaPath string) (services.RoutePrediction, float64, int, error) {
	p, seconds, cues, err := a.router.PredictRouteWithCues(ctx, mediaPath)
	if err != nil {
		return "", 0, 0, err
	}
	return services.RoutePrediction(p), seconds, cues, nil
}

// ProbeWithDuration satisfies services.RouteDurationPredictor — the optional
// half of the port (sub-6-10a AC #1). Same single probe, one more number out
// of it, so an episode's real length reaches the estimate instead of the
// 45-minute assumption.
func (a routePredictorAdapter) ProbeWithDuration(ctx context.Context, mediaPath string) (services.RoutePrediction, float64, error) {
	p, seconds, err := a.router.PredictRouteWithDuration(ctx, mediaPath)
	if err != nil {
		return "", 0, err
	}
	return services.RoutePrediction(p), seconds, nil
}
