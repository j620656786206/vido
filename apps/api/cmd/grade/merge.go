package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/eval"
)

// defaultRatingsPath is where the embedded table lives, relative to apps/api
// (the module root CI runs from).
const defaultRatingsPath = "internal/ai/model_ratings.json"

// ratingFromReport turns a cmd/grade report into the row model_ratings.json
// stores. An incomplete run has no grade to store, so it is refused here —
// the workflow must fail loudly rather than publish a letter nobody measured.
func ratingFromReport(rep eval.Report) (ai.ModelRating, error) {
	if rep.Grade == eval.GradeIncomplete || rep.Grade == "" {
		return ai.ModelRating{}, fmt.Errorf("report for %s is %q (%s) — not mergeable", rep.ModelID, rep.Grade, rep.Incomplete)
	}
	if rep.SampleVersion != eval.SampleVersion {
		return ai.ModelRating{}, fmt.Errorf("report measured on %q but this build embeds %q — re-run cmd/grade", rep.SampleVersion, eval.SampleVersion)
	}
	gradedAt := rep.GradedAt
	if gradedAt.IsZero() {
		gradedAt = time.Now()
	}
	return ai.ModelRating{
		ModelID: rep.ModelID, Grade: rep.Grade,
		ZeroRate: rep.ZeroRate, NaturalRate: rep.NaturalRate, CostUSD: rep.CostUSD, Cues: rep.Cues,
		PromptVersion: rep.PromptVersion, LexiconVersion: rep.LexiconVersion,
		JudgeModel: rep.JudgeModel, JudgeVersion: rep.JudgeVersion,
		GradedAt: gradedAt.UTC().Format(time.RFC3339),
		Source:   ai.RatingSourceGolden, SampleVersion: rep.SampleVersion,
	}, nil
}

// mergeReport reads a report file and writes the updated ratings table.
func mergeReport(reportPath, ratingsPath string) (ai.ModelRating, error) {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return ai.ModelRating{}, err
	}
	var rep eval.Report
	if err := json.Unmarshal(data, &rep); err != nil {
		return ai.ModelRating{}, fmt.Errorf("%s: %w", reportPath, err)
	}
	row, err := ratingFromReport(rep)
	if err != nil {
		return row, err
	}
	existing, err := os.ReadFile(ratingsPath)
	if err != nil {
		return row, err
	}
	merged, err := ai.MergeRating(existing, row)
	if err != nil {
		return row, err
	}
	if err := os.WriteFile(ratingsPath, merged, 0o644); err != nil {
		return row, err
	}
	return row, nil
}
