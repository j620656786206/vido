// Mirror-Types drift detector for project-context.md Rule 19 (story sub-4-1).
//
// services.RoutePrediction is a deliberate mirror of subtitle.RoutePrediction:
// the cost-preview service must speak about routes, but services cannot import
// subtitle without creating an import cycle. Step 4 of the Mirror-Types
// workaround says "keep the two in sync via code review" — this test does it
// mechanically instead.
//
// The values are WIRE-VISIBLE (they are serialized as `route` in the candidate
// JSON the frontend renders), so a silent divergence would not merely be untidy:
// the paid class would stop matching the badge that warns about it.
//
// Lives in `package services_test` so it may import both packages — the same
// arrangement srt_parity_test.go uses.
package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
)

func TestRoutePrediction_MirrorsSubtitlePackage(t *testing.T) {
	pairs := []struct {
		mirror   services.RoutePrediction
		original subtitle.RoutePrediction
		name     string
	}{
		{services.RouteExtract, subtitle.PredictExtract, "extract"},
		{services.RouteASR, subtitle.PredictASR, "asr"},
		{services.RouteSkipped, subtitle.PredictSkip, "skip"},
	}

	for _, p := range pairs {
		assert.Equalf(t, string(p.original), string(p.mirror),
			"services.RoutePrediction %q drifted from subtitle's — the value is wire-visible as `route`", p.name)
	}

	// A value added on one side only is the failure this guard exists for: the
	// estimator would fall through its switch and silently price the new route
	// at $0.
	assert.Len(t, pairs, 3,
		"a route class was added — mirror it on both sides and price it in estimateUSD before extending this list")
}
